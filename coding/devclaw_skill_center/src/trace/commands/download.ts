import { existsSync, mkdirSync, readdirSync, readFileSync, writeFileSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { loadConfig } from '../config/loader.js';
import { downloadRawBuffer } from '../server/tos-reader.js';
import { generatePresignedUrl } from '../upload/tos.js';
import { resolveLogDir } from '../utils/hook-log.js';
import { parseDuration } from '../utils/duration.js';
import { openInBrowser } from '../utils/open.js';

export interface DownloadOptions {
  lines: number;
  save?: string;
  open?: boolean;
  url?: boolean;
  withDiff?: boolean;
  tool?: string;
  session?: string;
  date?: string;
  expires?: string;
  baseDir?: string;
}

interface ForwardEntry {
  timestamp: string;
  sessionId: string;
  tool: string;
  jsonlObjectKey: string;
  diffObjectKey?: string;
  bucket: string;
}

export async function runDownload(options: DownloadOptions): Promise<void> {
  const action = resolveAction(options);
  const entries = collectForwardEntries(options);
  if (entries.length === 0) {
    const scope = options.date ? `for date ${options.date}` : 'in recent logs';
    process.stderr.write(`[trace] No forward:ok entries found ${scope}. Try --date YYYY-MM-DD or relax filters.\n`);
    return;
  }

  process.stderr.write(`[trace] Matched ${entries.length} session(s); action=${action}\n`);

  const config = await loadConfig(process.cwd());
  const expiresSec = parseDuration(options.expires ?? '24h');

  if (action === 'save') {
    const outDir = resolve(options.save ?? '.');
    mkdirSync(outDir, { recursive: true });
    for (const entry of entries) {
      await saveOne(entry, outDir, Boolean(options.withDiff), config.tos);
    }
  } else if (action === 'url') {
    for (const entry of entries) {
      const urlJsonl = generatePresignedUrl(config.tos, entry.jsonlObjectKey, expiresSec);
      process.stdout.write(urlJsonl + '\n');
      if (options.withDiff && entry.diffObjectKey) {
        const urlDiff = generatePresignedUrl(config.tos, entry.diffObjectKey, expiresSec);
        process.stdout.write(urlDiff + '\n');
      }
    }
  } else {
    if (entries.length > 10) {
      process.stderr.write(`[trace] Warning: about to open ${entries.length} browser tabs. Ctrl+C within 3s to abort.\n`);
      await sleep(3000);
    }
    for (const entry of entries) {
      const urlJsonl = generatePresignedUrl(config.tos, entry.jsonlObjectKey, expiresSec);
      openInBrowser(urlJsonl);
      await sleep(300);
      if (options.withDiff && entry.diffObjectKey) {
        const urlDiff = generatePresignedUrl(config.tos, entry.diffObjectKey, expiresSec);
        openInBrowser(urlDiff);
        await sleep(300);
      }
    }
  }
  process.stderr.write('[trace] done\n');
}

type Action = 'save' | 'open' | 'url';

function resolveAction(opts: DownloadOptions): Action {
  const flags = [
    opts.save !== undefined,
    Boolean(opts.open),
    Boolean(opts.url),
  ];
  const count = flags.filter(Boolean).length;
  if (count > 1) {
    throw new Error('--save, --open and --url are mutually exclusive');
  }
  if (opts.open) return 'open';
  if (opts.url) return 'url';
  return 'save';
}

export function collectForwardEntries(options: DownloadOptions): ForwardEntry[] {
  const logDir = resolveLogDir(options.baseDir);
  if (!existsSync(logDir)) return [];

  const files = options.date
    ? [`hook.${options.date}.log`]
    : readdirSync(logDir)
        .filter((f) => /^hook\.\d{4}-\d{2}-\d{2}\.log$/.test(f))
        .sort()
        .reverse();

  const seen = new Map<string, ForwardEntry>();
  const results: ForwardEntry[] = [];

  for (const file of files) {
    const fullPath = join(logDir, file);
    if (!existsSync(fullPath)) continue;
    const lines = readFileSync(fullPath, 'utf-8').split('\n');
    for (let i = lines.length - 1; i >= 0; i--) {
      const line = lines[i];
      if (!line) continue;
      const parsed = parseForwardLine(line);
      if (!parsed) continue;
      if (options.tool && parsed.tool !== options.tool) continue;
      if (options.session && !parsed.sessionId.startsWith(options.session)) continue;
      if (seen.has(parsed.sessionId)) continue;
      seen.set(parsed.sessionId, parsed);
      results.push(parsed);
      if (results.length >= options.lines) return results;
    }
  }
  return results;
}

function parseForwardLine(line: string): ForwardEntry | null {
  const spaceIdx1 = line.indexOf(' ');
  if (spaceIdx1 < 0) return null;
  const spaceIdx2 = line.indexOf(' ', spaceIdx1 + 1);
  if (spaceIdx2 < 0) return null;
  const timestamp = line.slice(0, spaceIdx1);
  const event = line.slice(spaceIdx1 + 1, spaceIdx2);
  const json = line.slice(spaceIdx2 + 1);
  if (event !== 'forward') return null;
  let payload: {
    result?: string;
    sessionId?: string;
    tool?: string;
    jsonlObjectKey?: string;
    diffObjectKey?: string;
    snapshotDiffsKey?: string;
    bucket?: string;
  };
  try {
    payload = JSON.parse(json);
  } catch {
    return null;
  }
  if (payload.result !== 'ok') return null;
  if (!payload.sessionId || !payload.jsonlObjectKey || !payload.bucket || !payload.tool) return null;
  return {
    timestamp,
    sessionId: payload.sessionId,
    tool: payload.tool,
    jsonlObjectKey: payload.jsonlObjectKey,
    diffObjectKey: payload.snapshotDiffsKey ?? payload.diffObjectKey,
    bucket: payload.bucket,
  };
}

async function saveOne(
  entry: ForwardEntry,
  outDir: string,
  withDiff: boolean,
  tosConfig: Parameters<typeof downloadRawBuffer>[0],
): Promise<void> {
  const safeSession = entry.sessionId.replace(/[^a-zA-Z0-9_-]/g, '_');
  try {
    const jsonlKey = entry.jsonlObjectKey.replace(/^\//, '');
    const dl = await downloadRawBuffer(tosConfig, jsonlKey);
    if (!dl) {
      process.stderr.write(`[trace] skip ${safeSession}: object not found (${jsonlKey})\n`);
      return;
    }
    const outJsonl = join(outDir, `${safeSession}.jsonl.gz`);
    writeFileSync(outJsonl, dl.buffer);
    process.stderr.write(`[trace] saved ${outJsonl} (${dl.buffer.length} bytes)\n`);

    if (withDiff && entry.diffObjectKey) {
      const diffKey = entry.diffObjectKey.replace(/^\//, '');
      const diffDl = await downloadRawBuffer(tosConfig, diffKey);
      if (diffDl) {
        const outDiff = join(outDir, diffKey.endsWith('.diffs.json.gz') ? `${safeSession}.diffs.json.gz` : `${safeSession}.diff.gz`);
        writeFileSync(outDiff, diffDl.buffer);
        process.stderr.write(`[trace] saved ${outDiff} (${diffDl.buffer.length} bytes)\n`);
      }
    }
  } catch (err) {
    process.stderr.write(`[trace] ERROR ${safeSession}: ${(err as Error).message}\n`);
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
