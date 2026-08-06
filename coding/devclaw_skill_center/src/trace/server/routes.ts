import { Router } from 'express';
import type { TraceConfig } from '../config/schema.js';
import {
  listSessionObjects,
  downloadAndDecompress,
  downloadRawBuffer,
  headObject,
  getObjectGitInfo,
  parseObjectKey,
  buildEvalCacheKey,
  buildBundleKey,
  uploadEvalResult,
  downloadEvalResult,
  loadUserMetaIndex,
} from './tos-reader.js';
import type { SessionListItem } from './tos-reader.js';
import { getCachedSessionTree, putCachedSessionTree } from './jsonl-cache.js';
import { analyzeSessionTree } from '../analyzer/index.js';
import type { EvalResult, SessionTree } from '../analyzer/types.js';
import { detectCli, evaluate } from '../analyzer/evaluator.js';
import { extractHarnessMetrics } from '../analyzer/harness-metrics.js';
import { scanRepoHarnessFiles } from '../analyzer/repo-scanner.js';
import { parseJsonl } from '../analyzer/parser.js';
import { normalize, normalizeSessionTree } from '../analyzer/normalizer.js';
import { extractSubagentStats } from '../analyzer/metrics.js';
import { extractSessionTree } from '../upload/extract.js';
import { fetchTraeThread } from '../fornax/index.js';
import type { FornaxFetchOptions } from '../fornax/types.js';
import { logger } from '../utils/logger.js';

/**
 * Create the API router with all endpoints.
 */
export function createApiRouter(config: TraceConfig): Router {
  const router = Router();

  // Health check
  router.get('/health', (_req, res) => {
    res.json({ status: 'ok' });
  });

  // GET /api/sessions — List sessions from TOS
  router.get('/sessions', async (req, res) => {
    try {
      const options = {
        userId: req.query['userId'] as string | undefined,
        toolName: req.query['toolName'] as string | undefined,
        sessionId: req.query['sessionId'] as string | undefined,
        dateFrom: req.query['dateFrom'] as string | undefined,
        dateTo: req.query['dateTo'] as string | undefined,
        department: req.query['department'] as string | undefined,
        page: req.query['page'] ? parseInt(req.query['page'] as string, 10) : 1,
        pageSize: req.query['pageSize'] ? parseInt(req.query['pageSize'] as string, 10) : 20,
      };

      logger.info(`[sessions] Listing with options: ${JSON.stringify(options)}`);
      const result = await listSessionObjects(config.tos, options);
      logger.info(`[sessions] Found ${result.total} sessions`);
      res.json({
        sessions: result.sessions,
        total: result.total,
        page: options.page,
        pageSize: options.pageSize,
        departments: result.departments,
      });
    } catch (err) {
      logger.error(`Failed to list sessions: ${err instanceof Error ? err.message : String(err)}`);
      const statusCode = isAuthError(err) ? 401 : 502;
      res.status(statusCode).json({
        error: 'Failed to list sessions from TOS',
        message: err instanceof Error ? err.message : String(err),
      });
    }
  });

  // GET /api/sessions/:sessionId/analysis — Always compute analysis in real-time
  // JSONL content is cached locally with ETag validation; only eval results are cached on TOS.
  router.get('/sessions/:sessionId/analysis', async (req, res) => {
    try {
      const sessionId = req.params['sessionId']!;
      const objectKey = req.query['objectKey'] as string | undefined;
      const force = req.query['force'] === 'true';

      if (!objectKey) {
        res.status(400).json({ error: 'objectKey query parameter is required' });
        return;
      }

      // Parse the object key to get metadata
      const sessionMeta = parseObjectKey(objectKey);
      if (!sessionMeta) {
        res.status(400).json({ error: 'Invalid objectKey format' });
        return;
      }

      // 1. Obtain SessionTree (local cache with ETag validation)
      const tree = await obtainSessionTree(config, sessionMeta, force);

      // 2. Run analysis (unified path through SessionTree)
      logger.info(`Analyzing session ${sessionId} (objectKey: ${objectKey}, format: ${sessionMeta.format}, subagents: ${tree.subagents.length})`);
      const analysis = analyzeSessionTree(tree, sessionId, sessionMeta.toolName);

      // 3. Try to load cached eval result from TOS (unless force refresh)
      if (!force) {
        const evalCacheKey = buildEvalCacheKey(sessionMeta);
        const cachedEval = await downloadEvalResult(config.tos, evalCacheKey);
        if (cachedEval) {
          logger.info(`Eval cache hit for session ${sessionId}`);
          analysis.evaluation = cachedEval;
        }
      }

      // 4. Attach user info from objectKey + user meta index
      const responseBody: Record<string, unknown> = { ...analysis };
      responseBody['userId'] = sessionMeta.userId;
      const userMeta = await loadUserMetaIndex(config.tos);
      const userRecord = userMeta.get(sessionMeta.userId);
      if (userRecord) {
        responseBody['nickname'] = userRecord.nickname;
        responseBody['department'] = userRecord.department;
      }

      // 5. For Trae threads: split by trace boundary markers and return per-trace analyses
      if (sessionMeta.toolName === 'trae') {
        const traceSlices = splitTraeByBoundary(tree.mainJsonlContent, sessionId);
        if (traceSlices.length > 1) {
          responseBody['traceSlices'] = traceSlices;
        }
      }

      res.json(responseBody);
    } catch (err) {
      logger.error(`Failed to analyze session: ${err instanceof Error ? err.message : String(err)}`);

      if (isNotFoundError(err)) {
        res.status(404).json({ error: 'Session not found in TOS' });
        return;
      }

      const statusCode = isAuthError(err) ? 401 : 502;
      res.status(statusCode).json({
        error: 'Failed to analyze session',
        message: err instanceof Error ? err.message : String(err),
      });
    }
  });

  // POST /api/analyze — Analyze JSONL content directly (no TOS dependency)
  // Accepts JSON body: { jsonlContent: string, sessionId?: string, toolName?: string }
  router.post('/analyze', (req, res) => {
    try {
      const body = req.body as Record<string, unknown>;
      const jsonlContent = body['jsonlContent'] as string | undefined;
      if (!jsonlContent || typeof jsonlContent !== 'string') {
        res.status(400).json({ error: 'jsonlContent (string) is required in request body' });
        return;
      }

      const sessionId = (body['sessionId'] as string) ?? 'local-analysis';
      const toolName = body['toolName'] as string | undefined;

      const tree: SessionTree = { mainJsonlContent: jsonlContent, subagents: [] };
      const analysis = analyzeSessionTree(tree, sessionId, toolName);
      res.json(analysis);
    } catch (err) {
      logger.error(`Failed to analyze: ${err instanceof Error ? err.message : String(err)}`);
      res.status(500).json({
        error: 'Analysis failed',
        message: err instanceof Error ? err.message : String(err),
      });
    }
  });

  // POST /api/sessions/:sessionId/evaluate — Run LLM-as-Judge evaluation
  router.post('/sessions/:sessionId/evaluate', async (req, res) => {
    try {
      const sessionId = req.params['sessionId']!;
      const body = req.body as Record<string, unknown>;
      const objectKey = body['objectKey'] as string | undefined;
      const preferredCli = (body['cli'] as string | undefined) ?? 'claude-code';
      const traceContext = body['traceContext'] as string | undefined;

      if (!objectKey) {
        res.status(400).json({ error: 'objectKey is required in request body' });
        return;
      }

      // 1. Detect CLI
      const cli = await detectCli(preferredCli as 'claude-code' | 'opencode');
      if (!cli) {
        res.status(503).json({
          error: 'No Coding CLI available',
          message: 'Please install claude or opencode CLI on this machine. ' +
            'See https://docs.anthropic.com/claude-code or https://opencode.ai for installation instructions.',
        });
        return;
      }

      logger.info(`[evaluate] Using ${cli.type} CLI at ${cli.path} for session ${sessionId}`);

      // 2. Obtain SessionTree (from TOS with cache)
      const sessionMeta = parseObjectKey(objectKey);
      if (!sessionMeta) {
        res.status(400).json({ error: 'Invalid objectKey format' });
        return;
      }

      const tree = await obtainSessionTree(config, sessionMeta, false);
      let jsonlContent = tree.mainJsonlContent;

      // For multi-trace Trae evaluations, prepend context about the trace relationship
      if (traceContext) {
        const contextLine = JSON.stringify({ _trae_format: true, role: 'system', timestamp: '', content: `[Evaluation Context] ${traceContext}` });
        jsonlContent = contextLine + '\n' + jsonlContent;
      }

      // 3. Extract harness metrics for the evaluation prompt
      const toolName = sessionMeta.toolName;
      const rawMessages = parseJsonl(jsonlContent);
      const normalizedMessages = normalize(rawMessages, toolName);

      // 3a. Try to scan the full repo for harness files (graceful degradation if unavailable)
      let repoHarnessFiles: string[] | undefined;
      const gitInfo = await getObjectGitInfo(config.tos, objectKey);
      if (gitInfo) {
        logger.info(`[evaluate] Scanning repo ${gitInfo.gitUrl}@${gitInfo.gitCommit.slice(0, 8)} for harness files`);
        repoHarnessFiles = await scanRepoHarnessFiles(gitInfo.gitUrl, gitInfo.gitCommit);
        logger.info(`[evaluate] Repo scan found ${repoHarnessFiles.length} harness files`);
      } else {
        logger.info(`[evaluate] No git info in TOS metadata, skipping repo scan`);
      }

      const harnessMetrics = extractHarnessMetrics(normalizedMessages, repoHarnessFiles);
      logger.info(`[evaluate] Harness metrics: ${harnessMetrics.totalArtifacts} artifacts (${harnessMetrics.usedArtifacts} used, repoScanned=${harnessMetrics.repoScanned})`);

      // 4. Extract subagent data from tree
      let subagentStats;
      let subagentMessages;
      if (tree.subagents.length > 0) {
        const treeResult = normalizeSessionTree(tree, toolName);
        subagentMessages = treeResult.subagentMessages;
        subagentStats = extractSubagentStats(subagentMessages, tree.subagents);
        logger.info(`[evaluate] Loaded ${tree.subagents.length} subagents for evaluation`);
      }

      // 5. Run evaluation
      const evalResult = await evaluate(jsonlContent, cli, harnessMetrics, subagentStats, subagentMessages);
      logger.info(`[evaluate] Session ${sessionId} scored ${evalResult.score} by ${evalResult.evaluatedBy}`);

      // Attach programmatic harness stats (not from LLM output)
      evalResult.harnessStats = {
        totalArtifacts: harnessMetrics.totalArtifacts,
        usedArtifacts: harnessMetrics.usedArtifacts,
        unusedArtifacts: harnessMetrics.unusedArtifacts,
        repoScanned: harnessMetrics.repoScanned,
      };

      // 6. Cache eval result to TOS (best-effort)
      uploadEvalCache(config, sessionMeta, evalResult).catch(err => {
        logger.error(`[evaluate] Failed to cache evaluation: ${err instanceof Error ? err.message : String(err)}`);
      });

      // 7. Return result
      res.json(evalResult);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      logger.error(`[evaluate] Failed: ${message}`);

      if (message.includes('timed out')) {
        res.status(504).json({ error: 'Evaluation timed out', message });
        return;
      }

      const statusCode = isAuthError(err) ? 401 : 500;
      res.status(statusCode).json({
        error: 'Evaluation failed',
        message,
      });
    }
  });

  // POST /api/trae/import — Import all traces for a Trae thread from Fornax
  // Flattens all spans into a single unified JSONL (no boundary markers).
  // Rolling context means the last model span contains the full conversation.
  router.post('/trae/import', async (req, res) => {
    try {
      const body = req.body as Record<string, unknown>;
      const threadId = body['threadId'] as string | undefined;
      if (!threadId || typeof threadId !== 'string') {
        res.status(400).json({ error: 'threadId (string) is required in request body' });
        return;
      }

      const fornaxConfig = body['fornaxConfig'] as FornaxFetchOptions | undefined;
      const commitSha = (body['commitSha'] as string | undefined)?.trim() || undefined;
      const repoUrl = (body['repoUrl'] as string | undefined)?.trim() || undefined;

      const fetchOptions: FornaxFetchOptions = {
        ak: fornaxConfig?.ak ?? config.fornax?.ak,
        sk: fornaxConfig?.sk ?? config.fornax?.sk,
        region: fornaxConfig?.region ?? config.fornax?.region,
        endpoint: fornaxConfig?.endpoint ?? config.fornax?.endpoint,
        timeout: fornaxConfig?.timeout ?? '180s',
        since: (fornaxConfig?.since as string | undefined),
        until: (fornaxConfig?.until as string | undefined),
      };

      logger.info(`[trae/import] Fetching thread ${threadId} from Fornax`);

      // 1. Fetch all traces for this thread
      const traces = await fetchTraeThread(threadId, fetchOptions);

      // 2. Flatten all spans and normalize once — rolling context means the
      //    last model span already contains the full thread conversation.
      const { jsonl: combinedJsonl, analysis } = buildFlattenedTraeJsonl(traces, threadId);

      logger.info(`[trae/import] Flattened ${traces.length} traces → ${analysis.assistantMsgCount} assistant msgs, ${analysis.totalToolCalls} tool calls`);

      // 3. Upload single file to TOS — standard 4-segment path
      const date = new Date().toISOString().slice(0, 10);
      const userId = fetchOptions.ak?.slice(0, 8) ?? 'anonymous';
      const objectKey = `xtrace/${userId}/${date}/trae/${threadId}.jsonl.gz`;

      // Upload to TOS — wait for completion so it appears in session list
      let uploadError: string | undefined;
      try {
        await uploadTraeToTos(config, objectKey, combinedJsonl, { commitSha, repoUrl });
      } catch (err) {
        uploadError = err instanceof Error ? err.message : String(err);
        logger.error(`[trae/import] Failed to upload to TOS: ${uploadError}`);
      }

      res.json({
        threadId,
        objectKey,
        traceCount: traces.length,
        analysis,
        ...(uploadError ? { uploadError } : {}),
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      logger.error(`[trae/import] Failed: ${message}`);
      res.status(500).json({ error: 'Trae thread import failed', message });
    }
  });

  return router;
}

// ─── Internal helpers ──────────────────────────────────────────────

/**
 * Obtain a SessionTree for a session, using local file cache with TOS ETag validation.
 *
 * Handles both formats transparently:
 *   - `bundle` (.session.tar.gz): download raw buffer, extract SessionTree
 *   - `jsonl` (.jsonl.gz, legacy): try companion bundle first, then gunzip and wrap
 *
 * Flow:
 *   1. If !force, try local cache → HEAD request to compare ETag
 *   2. If ETag matches, return cached tree (skip download)
 *   3. If ETag differs, cache miss, or force: download from TOS
 *   4. After download, update local cache (best-effort)
 *   5. If HEAD request fails, fall back to cached tree (graceful degradation)
 */
async function obtainSessionTree(
  config: TraceConfig,
  sessionMeta: SessionListItem,
  force: boolean,
): Promise<SessionTree> {
  const objectKey = sessionMeta.objectKey;

  // Try local cache first (unless force refresh)
  if (!force) {
    const cached = await getCachedSessionTree(objectKey);
    if (cached) {
      const remote = await headObject(config.tos, objectKey).catch(() => null);
      if (!remote) {
        logger.debug(`HEAD failed for ${objectKey}, using local cache (graceful degradation)`);
        return cached.tree;
      }
      if (remote.etag && remote.etag === cached.etag) {
        logger.debug(`Local cache hit (ETag match) for ${objectKey}`);
        return cached.tree;
      }
      logger.debug(`Local cache stale (ETag mismatch) for ${objectKey}, re-downloading`);
    }
  }

  // Download and extract based on format
  let tree: SessionTree;
  let etag: string;

  if (sessionMeta.format === 'bundle') {
    // New format: download tar.gz, extract SessionTree
    const dl = await downloadRawBuffer(config.tos, objectKey);
    if (!dl) throw new Error(`Bundle not found: ${objectKey}`);
    etag = dl.etag;
    logger.info(`Downloaded session bundle: ${objectKey} (${dl.buffer.length} bytes)`);
    tree = await extractSessionTree(dl.buffer);
  } else {
    // Legacy format: .jsonl.gz
    // First, try to find a companion bundle (old uploads may have both)
    const bundleKey = buildBundleKey(sessionMeta);
    const bundleDl = await downloadRawBuffer(config.tos, bundleKey);
    if (bundleDl) {
      logger.info(`Downloaded companion bundle: ${bundleKey} (${bundleDl.buffer.length} bytes)`);
      etag = bundleDl.etag;
      tree = await extractSessionTree(bundleDl.buffer);
    } else {
      // No bundle: download raw JSONL, wrap as minimal SessionTree
      const result = await downloadAndDecompress(config.tos, objectKey);
      etag = result.etag;
      tree = { mainJsonlContent: result.content, subagents: [] };
    }
  }

  // Update local cache (best-effort, don't block response)
  putCachedSessionTree(objectKey, tree, etag).catch(err => {
    logger.debug(`Failed to update local SessionTree cache: ${err instanceof Error ? err.message : String(err)}`);
  });

  return tree;
}

/**
 * Upload eval result to TOS cache.
 */
async function uploadEvalCache(
  config: TraceConfig,
  sessionMeta: SessionListItem,
  evalResult: EvalResult,
): Promise<void> {
  const cacheKey = buildEvalCacheKey(sessionMeta);
  await uploadEvalResult(config.tos, cacheKey, evalResult);
  logger.info(`[evaluate] Cached eval result at ${cacheKey}`);
}

function isAuthError(err: unknown): boolean {
  if (err instanceof Error) {
    return err.message.includes('401') || err.message.includes('auth') || err.message.includes('token');
  }
  return false;
}

/**
 * Upload Trae trace JSONL to TOS (gzipped).
 * Best-effort: failure is logged but doesn't block the import response.
 */
async function uploadTraeToTos(
  config: TraceConfig,
  objectKey: string,
  jsonlContent: string,
  gitInfo?: { commitSha?: string; repoUrl?: string },
): Promise<void> {
  try {
    const { gzipSync } = await import('node:zlib');
    const compressed = gzipSync(Buffer.from(jsonlContent, 'utf-8'));

    const { createRequire } = await import('node:module');
    const require = createRequire(import.meta.url);
    const Tos = require('@byted-service/tos');
    const tos = new Tos({
      bucket: config.tos.bucket,
      accessKey: config.tos.accessKey,
      secretKey: config.tos.secretKey,
      signatureVersion: 'sign_v1',
      disableWatcher: true,
      reqTimeout: 30000,
    });

    try {
      await tos.upload(compressed, objectKey, {
        headers: {
          'Content-Type': 'application/gzip',
          'Content-Encoding': 'gzip',
          'x-tos-meta-tool-name': 'trae',
          'x-tos-meta-source': 'fornax-import',
          ...(gitInfo?.commitSha ? { 'x-tos-meta-commit-sha': gitInfo.commitSha } : {}),
          ...(gitInfo?.repoUrl ? { 'x-tos-meta-repo-url': gitInfo.repoUrl } : {}),
        },
      });
      logger.info(`[trae/import] Uploaded to TOS: ${objectKey} (${compressed.length} bytes)`);
    } finally {
      tos.destroy();
    }
  } catch (err) {
    throw err;
  }
}

/**
 * Split a Trae multi-trace JSONL into per-trace analyses using boundary markers.
 * Returns an array of { traceId, traceIndex, traceCount, analysis }.
 * If no boundary markers found, returns empty array.
 */
export function splitTraeByBoundary(
  jsonlContent: string,
  _sessionId: string,
): Array<{ traceId: string; traceIndex: number; traceCount: number; analysis: import('../analyzer/types.js').SessionAnalysis }> {
  const lines = jsonlContent.split('\n').filter(l => l.trim());
  const parsed: Array<Record<string, unknown>> = [];
  for (const line of lines) {
    try { parsed.push(JSON.parse(line)); } catch { /* skip */ }
  }

  // Find boundary markers
  const boundaries = parsed
    .map((obj, idx) => ({ obj, idx }))
    .filter(({ obj }) => obj['_trae_trace_boundary'] === true);

  if (boundaries.length <= 1) return [];

  const results: Array<{ traceId: string; traceIndex: number; traceCount: number; analysis: import('../analyzer/types.js').SessionAnalysis }> = [];

  for (let b = 0; b < boundaries.length; b++) {
    const boundary = boundaries[b]!;
    const traceId = (boundary.obj['traceId'] as string) ?? `trace-${b}`;
    const traceIndex = (boundary.obj['traceIndex'] as number) ?? b;
    const traceCount = (boundary.obj['traceCount'] as number) ?? boundaries.length;

    // Collect lines between this boundary and the next (or end)
    const startIdx = boundary.idx + 1;
    const endIdx = b + 1 < boundaries.length ? boundaries[b + 1]!.idx : parsed.length;
    const traceLines = parsed.slice(startIdx, endIdx);

    // Build a mini JSONL and analyze
    const traceJsonl = traceLines.map(obj => JSON.stringify(obj)).join('\n') + '\n';
    const traceTree: import('../analyzer/types.js').SessionTree = {
      mainJsonlContent: traceJsonl,
      subagents: [],
    };
    const traceAnalysis = analyzeSessionTree(traceTree, traceId, 'trae');

    results.push({ traceId, traceIndex, traceCount, analysis: traceAnalysis });
  }

  return results;
}

/**
 * Flatten all spans from every trace in a thread, normalize once, and emit
 * a single unified JSONL payload (no `_trae_trace_boundary` markers).
 *
 * Relies on Trae's rolling-context property: the globally last `type:"model"`
 * span's `input.messages + output` contains the full thread conversation,
 * which `normalizeTraeFromSpans` already reconstructs. Trailing tool spans
 * after the last model span are attached to the final assistant turn.
 */
export function buildFlattenedTraeJsonl(
  traces: import('../fornax/types.js').FornaxTraceData[],
  threadId: string,
): { jsonl: string; analysis: import('../analyzer/types.js').SessionAnalysis } {
  const allSpans = traces.flatMap(t => t.spans);
  const rawSpanJsonl = allSpans.map(span => JSON.stringify(span)).join('\n') + '\n';
  const tree: import('../analyzer/types.js').SessionTree = {
    mainJsonlContent: rawSpanJsonl,
    subagents: [],
  };
  const analysis = analyzeSessionTree(tree, threadId, 'trae');
  const lines = analysis.conversation.map(msg =>
    JSON.stringify({ _trae_format: true, ...msg })
  );
  return { jsonl: lines.join('\n') + '\n', analysis };
}

function isNotFoundError(err: unknown): boolean {
  if (err instanceof Error) {
    return err.message.includes('404') || err.message.includes('NoSuchKey') || err.message.includes('not found');
  }
  return false;
}
