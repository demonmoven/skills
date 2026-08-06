import { execFileSync } from 'node:child_process';
import { logger } from '../utils/logger.js';
import type { FornaxTraceData, FornaxSpan, FornaxFetchOptions } from './types.js';

/**
 * Locate the fornax-cli binary on PATH.
 * Returns the absolute path or null if not found.
 */
function findFornaxCli(): string | null {
  try {
    const result = execFileSync('which', ['fornax-cli'], {
      encoding: 'utf-8',
      timeout: 5000,
      stdio: ['pipe', 'pipe', 'pipe'],
    });
    return result.trim() || null;
  } catch {
    return null;
  }
}

function ensureFornaxCli(): string {
  const cliBin = findFornaxCli();
  if (!cliBin) {
    throw new Error(
      'fornax-cli is not installed. Install it with:\n' +
      '  curl -fsSL "https://lf3-static.bytednsdoc.com/obj/eden-cn/dvsM/ljhwZthlaukjlkulzlp/fornax_cli_install.sh" | bash',
    );
  }
  return cliBin;
}

function buildEnv(options?: FornaxFetchOptions): Record<string, string | undefined> {
  const env: Record<string, string | undefined> = { ...process.env };
  if (options?.ak) env['FORNAX_AK'] = options.ak;
  if (options?.sk) env['FORNAX_SK'] = options.sk;
  if (options?.region) env['FORNAX_CUSTOM_REGION'] = options.region;
  if (options?.endpoint) env['FORNAX_ENDPOINT'] = options.endpoint;
  return env;
}

function buildAuthArgs(options?: FornaxFetchOptions): string[] {
  const args: string[] = [];
  if (options?.ak) args.push('--ak', options.ak);
  if (options?.sk) args.push('--sk', options.sk);
  if (options?.region) args.push('--custom-region', options.region);
  if (options?.endpoint) args.push('--endpoint', options.endpoint);
  return args;
}

function buildTimeArgs(options?: FornaxFetchOptions): string[] {
  if (options?.since) {
    const args = ['--since', options.since];
    if (options?.until) args.push('--until', options.until);
    return args;
  }
  return ['--last-n-minutes', String(7 * 24 * 60)]; // default 7 days
}

/**
 * Parse fornax-cli output. Supports two formats:
 * - JSON array (`--format json`): `[{...}, {...}]`
 * - NDJSON (`--format raw`): one JSON object per line
 *
 * Also handles x-tt-logid lines and "null" empty results.
 */
function parseFornaxOutput(stdout: string, label: string): FornaxSpan[] {
  const trimmed = stdout.trim();
  if (!trimmed || trimmed.endsWith('null')) return [];

  // Try JSON array first
  const jsonStart = trimmed.indexOf('[');
  if (jsonStart !== -1) {
    try {
      return JSON.parse(trimmed.slice(jsonStart)) as FornaxSpan[];
    } catch {
      // Fall through to NDJSON parsing
    }
  }

  // NDJSON: each line starting with '{' is a span
  const spans: FornaxSpan[] = [];
  for (const line of trimmed.split('\n')) {
    const l = line.trim();
    if (!l.startsWith('{')) continue;
    try {
      spans.push(JSON.parse(l) as FornaxSpan);
    } catch {
      // skip malformed lines (e.g. x-tt-logid)
    }
  }

  if (spans.length === 0) {
    throw new Error(`fornax-cli ${label} returned no parseable spans. Output preview: ${trimmed.slice(0, 200)}`);
  }

  return spans;
}

/**
 * Fetch a single Fornax trace by trace_id.
 */
export async function fetchTraeTrace(
  traceId: string,
  options?: FornaxFetchOptions,
): Promise<FornaxTraceData> {
  const cliBin = ensureFornaxCli();
  const timeout = options?.timeout ?? '60s';

  const args = [
    'trace', 'get',
    '--trace-id', traceId,
    '--format', 'raw',
    '--timeout', timeout,
    ...buildAuthArgs(options),
    ...buildTimeArgs(options),
  ];

  logger.info(`[fornax] Fetching trace ${traceId} via ${cliBin}`);

  let stdout: string;
  try {
    stdout = execFileSync(cliBin, args, {
      encoding: 'utf-8',
      timeout: 180_000,
      maxBuffer: 200 * 1024 * 1024, // 200MB — single trace in raw format
      env: buildEnv(options),
      stdio: ['pipe', 'pipe', 'pipe'],
    });
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    throw new Error(`fornax-cli trace get failed: ${message}`);
  }

  const spans = parseFornaxOutput(stdout, 'trace get');

  if (spans.length === 0) {
    throw new Error(`Fornax returned empty trace for trace_id=${traceId}`);
  }

  logger.info(`[fornax] Fetched ${spans.length} spans for trace ${traceId}`);

  return { trace_id: traceId, spans };
}

/**
 * Fetch all traces for a Trae thread (conversation_id) from Fornax.
 *
 * Uses a sliding 1-hour window starting from `since`. Each window queries
 * `trace list` to find traces, then slides forward until no new traces are found
 * (or a maximum of 24 windows = 24 hours). This handles threads that span more
 * than 1 hour, which is Fornax's per-query time limit.
 *
 * @returns Array of FornaxTraceData, one per trace, sorted chronologically
 */
export async function fetchTraeThread(
  threadId: string,
  options?: FornaxFetchOptions,
): Promise<FornaxTraceData[]> {
  const cliBin = ensureFornaxCli();

  if (!options?.since) {
    throw new Error(
      'Start time (since) is required for thread import. ' +
      'Example: since="2026-04-09T11:00:00+08:00"',
    );
  }

  const WINDOW_MS = 60 * 60 * 1000; // 1 hour
  const DEFAULT_WINDOWS_WITHOUT_UNTIL = 24; // default span when caller omits `until`
  const MAX_CONSECUTIVE_EMPTY_WINDOWS = 6; // stop after this many consecutive idle hours
  const startMs = new Date(options.since).getTime();
  if (isNaN(startMs)) {
    throw new Error(`Invalid since time: ${options.since}`);
  }

  // Detect timezone offset from the user's since string to preserve in sliding windows
  // e.g. "+08:00" from "2026-04-09T11:00:00+08:00"
  const tzMatch = options.since.match(/([+-]\d{2}:\d{2})$/);
  const tz = tzMatch ? tzMatch[1] : '+08:00'; // default to CST

  // Never slide past "now" — Fornax rejects queries whose window lies entirely
  // in the future with "start&end time both exceed today" (code 600904002).
  const nowMs = Date.now();
  const hardStop = Math.min(
    options.until ? new Date(options.until).getTime() : startMs + DEFAULT_WINDOWS_WITHOUT_UNTIL * WINDOW_MS,
    nowMs,
  );

  const allSpans: FornaxSpan[] = [];
  const seenTraceIds = new Set<string>();
  let windowStart = startMs;
  let windowCount = 0;
  let rateLimitRetries = 0;
  let consecutiveEmpty = 0;
  const MAX_RATE_LIMIT_RETRIES = 3;

  /** Format epoch ms to ISO with original timezone */
  function toLocalISO(ms: number): string {
    const tzOffsetMinutes = parseInt(tz.slice(1, 3)) * 60 + parseInt(tz.slice(4, 6));
    const sign = tz.startsWith('+') ? 1 : -1;
    const local = new Date(ms + sign * tzOffsetMinutes * 60 * 1000);
    const pad = (n: number) => String(n).padStart(2, '0');
    return `${local.getUTCFullYear()}-${pad(local.getUTCMonth() + 1)}-${pad(local.getUTCDate())}T${pad(local.getUTCHours())}:${pad(local.getUTCMinutes())}:${pad(local.getUTCSeconds())}${tz}`;
  }

  logger.info(`[fornax] Discovering traces for thread ${threadId} starting from ${options.since}`);

  while (windowStart < hardStop) {
    const windowEnd = Math.min(windowStart + WINDOW_MS, hardStop);
    const sinceStr = toLocalISO(windowStart);
    const untilStr = toLocalISO(windowEnd);

    logger.info(`[fornax] Window ${windowCount + 1}: ${sinceStr} ~ ${untilStr}`);

    const listArgs = [
      'trace', 'list',
      '--trace-filter-expr', `conversation_id = '${threadId}'`,
      '--page-size', '50',
      '--format', 'raw',
      '--timeout', options?.timeout ?? '180s',
      '--since', sinceStr,
      '--until', untilStr,
      ...buildAuthArgs(options),
    ];

    let listStdout: string;
    try {
      listStdout = execFileSync(cliBin, listArgs, {
        encoding: 'utf-8',
        timeout: 600_000,
        maxBuffer: 500 * 1024 * 1024,
        env: buildEnv(options),
        stdio: ['pipe', 'pipe', 'pipe'],
      });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      // fornax-cli may exit non-zero but still have partial data in stdout
      // (e.g. rate limit after partial page fetch). Try to salvage stdout.
      const errStdout = (err as { stdout?: string })?.stdout ?? '';

      // Rate limit — try to parse whatever spans were returned before the limit
      if (message.includes('rate limit') || message.includes('Max retries exceeded')) {
        if (errStdout.trim()) {
          logger.warn(`[fornax] Window ${windowCount + 1}: rate limited, salvaging partial stdout`);
          listStdout = errStdout;
        } else {
          rateLimitRetries++;
          if (rateLimitRetries > MAX_RATE_LIMIT_RETRIES) {
            logger.warn(`[fornax] Window ${windowCount + 1}: rate limit retries exhausted, skipping window`);
            windowStart = windowEnd;
            windowCount++;
            continue;
          }
          logger.warn(`[fornax] Window ${windowCount + 1}: rate limited with no data, waiting 15s (retry ${rateLimitRetries}/${MAX_RATE_LIMIT_RETRIES})...`);
          await new Promise(resolve => setTimeout(resolve, 15_000));
          // Retry this same window
          continue;
        }
      }
      // Window lies in the future (clock skew or since very close to now) — stop sliding
      else if (message.includes('exceed today') || message.includes('600904002')) {
        logger.info(`[fornax] Window ${windowCount + 1}: reached future boundary, stopping`);
        break;
      }
      // "null" result or empty window — not a fatal error, just no data in this window
      else if (message.includes('No traces found') || message.includes('no parseable spans') || message.includes('null')) {
        logger.info(`[fornax] Window ${windowCount + 1}: empty (no spans)`);
        // Once we've seen traces, track consecutive empty windows to detect thread end
        // without stopping on normal idle gaps (meals, sleep, etc.)
        if (seenTraceIds.size > 0) {
          consecutiveEmpty++;
          if (consecutiveEmpty >= MAX_CONSECUTIVE_EMPTY_WINDOWS) {
            logger.info(`[fornax] ${MAX_CONSECUTIVE_EMPTY_WINDOWS}h of consecutive empty windows after ${seenTraceIds.size} traces, stopping`);
            break;
          }
        }
        windowStart = windowEnd;
        windowCount++;
        continue;
      } else {
        throw new Error(`fornax-cli trace list failed: ${message}`);
      }
    }

    const windowSpans = parseFornaxOutput(listStdout, 'trace list');
    rateLimitRetries = 0; // Reset on successful data

    let newTraces = 0;
    for (const span of windowSpans) {
      if (!seenTraceIds.has(span.trace_id)) {
        seenTraceIds.add(span.trace_id);
        newTraces++;
      }
      allSpans.push(span);
    }

    logger.info(`[fornax] Window ${windowCount + 1}: ${windowSpans.length} spans, ${newTraces} new traces`);

    // Truly empty window (no spans at all) — count toward idle threshold.
    // A window with spans-but-no-new-trace is still active (continuation of a long trace).
    if (windowSpans.length === 0) {
      if (seenTraceIds.size > 0) {
        consecutiveEmpty++;
        if (consecutiveEmpty >= MAX_CONSECUTIVE_EMPTY_WINDOWS) {
          logger.info(`[fornax] ${MAX_CONSECUTIVE_EMPTY_WINDOWS}h of consecutive empty windows after ${seenTraceIds.size} traces, stopping`);
          break;
        }
      }
    } else {
      consecutiveEmpty = 0;
    }

    windowStart = windowEnd;
    windowCount++;
  }

  if (allSpans.length === 0) {
    throw new Error(`No traces found for thread_id=${threadId}`);
  }

  // Group and sort by trace_id
  const traceTimeMap = new Map<string, number>();
  for (const span of allSpans) {
    const tid = span.trace_id;
    const t = parseInt(String(span.started_at ?? '0'), 10);
    const existing = traceTimeMap.get(tid);
    if (existing === undefined || t < existing) {
      traceTimeMap.set(tid, t);
    }
  }

  const traceIds = [...traceTimeMap.entries()]
    .sort(([, a], [, b]) => a - b)
    .map(([tid]) => tid);

  const traces: FornaxTraceData[] = traceIds.map(tid => ({
    trace_id: tid,
    spans: allSpans.filter(s => s.trace_id === tid),
  }));

  logger.info(`[fornax] Fetched ${traces.length} traces (${allSpans.length} total spans) for thread ${threadId} across ${windowCount} windows`);

  return traces;
}
