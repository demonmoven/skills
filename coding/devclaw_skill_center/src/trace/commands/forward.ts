import { access, constants, writeFile, mkdir, stat } from 'node:fs/promises';
import { join } from 'node:path';
import { detectAdapter } from '../adapters/detect.js';
import { collectGitContext, collectMultiRepoDiffs } from '../enrichers/git.js';
import { getUserInfo } from '../enrichers/user.js';
import { getClientId } from '../enrichers/device.js';
import { loadConfig } from '../config/loader.js';
import { ensureValidToken } from './auth.js';
import { gzipFile, gzipString, cleanupTempFile } from '../utils/compress.js';
import { buildObjectKey, uploadToTOS, buildObjectMetadata } from '../upload/tos.js';
import { ensureUserMeta } from '../upload/user-meta.js';
import { loadSessionState } from '../session/state.js';
import { logger } from '../utils/logger.js';
import { appendHookLog } from '../utils/hook-log.js';
import { FileNotFoundError, AdapterDetectionError } from '../utils/errors.js';
import type { UploadMeta } from '../upload/types.js';
import type { ForwardResult } from '../upload/types.js';

export interface ForwardOptions {
  source?: string;
  cwd?: string;
  dryRun?: boolean;
  dryRunOutput?: string;
}

/**
 * Forward command: collect metadata, compress, and upload a session file to TOS.
 * With --dry-run: runs full pipeline but writes to local directory instead of uploading.
 */
export async function runForward(
  filePath: string,
  options: ForwardOptions,
): Promise<ForwardResult> {
  let loggedSessionId: string | undefined;
  let loggedTool: string | undefined;

  try {
    return await runForwardInner(filePath, options, (id, tool) => {
      loggedSessionId = id;
      loggedTool = tool;
    });
  } catch (err) {
    appendHookLog({
      event: 'forward',
      result: 'error',
      sessionId: loggedSessionId,
      tool: loggedTool,
      errorCode: err instanceof Error ? err.constructor.name : 'Unknown',
      errorMessage: err instanceof Error ? err.message : String(err),
    });
    throw err;
  }
}

async function runForwardInner(
  filePath: string,
  options: ForwardOptions,
  onMetaResolved: (sessionId: string, toolName: string) => void,
): Promise<ForwardResult> {
  const workDir = options.cwd ?? process.cwd();

  // 1. Validate file exists
  try {
    await access(filePath, constants.R_OK);
  } catch {
    throw new FileNotFoundError(filePath);
  }

  const fileStat = await stat(filePath);
  logger.info(`Input file: ${filePath} (${fileStat.size} bytes)`);

  // 2. Check authentication (skip in dry-run, optional in real mode for userInfo)
  if (!options.dryRun) {
    // Auth is optional for upload (AK/SK in config), but needed for userInfo
    await ensureValidToken().catch(() => null);
  }

  // 3. Detect tool type
  const adapter = detectAdapter(filePath, options.source);
  if (!adapter) {
    throw new AdapterDetectionError(filePath);
  }
  logger.info(`Detected tool: ${adapter.name}`);

  // 4. Extract session metadata
  const adapterMeta = await adapter.extractMeta(filePath);
  onMetaResolved(adapterMeta.sessionId, adapterMeta.toolName);
  logger.info(`Session ID: ${adapterMeta.sessionId}`);
  if (adapterMeta.toolVersion) logger.info(`Tool version: ${adapterMeta.toolVersion}`);
  if (adapterMeta.startedAt) logger.info(`Started at: ${adapterMeta.startedAt}`);

  // 5. Collect enrichment data in parallel
  const [gitContext, userInfo, deviceId, config] = await Promise.all([
    collectGitContext(workDir, adapterMeta.sessionId),
    getUserInfo(),
    getClientId(),
    loadConfig(workDir),
  ]);

  logger.info(`Git: ${gitContext.repoUrl ?? '(none)'} @ ${gitContext.branch ?? '(none)'} [${gitContext.commitHead?.slice(0, 8) ?? '(none)'}]`);
  logger.info(`User: ${userInfo.userId ?? '(anonymous)'}`);
  logger.info(`Device: ${deviceId}`);
  if (gitContext.repoDirty !== undefined) logger.info(`Repo dirty: ${gitContext.repoDirty}`);
  if (gitContext.diffFromStart !== undefined) logger.info(`Diff size: ${gitContext.diffFromStart.length} chars`);

  // 6. Build upload metadata
  const uploadMeta: UploadMeta = {
    toolName: adapterMeta.toolName,
    sessionId: adapterMeta.sessionId,
    traceId: `${adapterMeta.toolName}-${adapterMeta.sessionId}`,
    sessionStartedAt: adapterMeta.startedAt,
    userId: userInfo.userId,
    nickname: userInfo.nickname,
    department: userInfo.department,
    gitUrl: gitContext.repoUrl,
    gitBranch: gitContext.branch,
    gitCommit: gitContext.commitHead,
    workspaceRoot: gitContext.workspaceRoot,
    repoDirty: gitContext.repoDirty,
    deviceId,
    cliVersion: '0.1.0',
  };

  // 6a. Load session state for multi-repo snapshot
  const sessionState = await loadSessionState(adapterMeta.sessionId);
  if (sessionState?.startCommit) {
    uploadMeta.gitStartCommit = sessionState.startCommit;
  }

  // 6b. Build multi-repo snapshot data (manifest + diffs)
  let snapshotManifest: string | undefined;
  let snapshotDiffs: string | undefined;

  if (sessionState?.repoSnapshots && sessionState.repoSnapshots.length > 0) {
    const snapshotWorkDir = sessionState.cwd || workDir;
    const manifest = {
      version: 1,
      workspace: {
        path: snapshotWorkDir,
        isRepo: Boolean(sessionState.startCommit),
      },
      repos: sessionState.repoSnapshots.map(r => ({
        relativePath: r.relativePath,
        repoUrl: r.repoUrl,
        branch: r.branch,
        startCommit: r.startCommit,
        dirty: r.dirty,
      })),
    };
    snapshotManifest = JSON.stringify(manifest, null, 2);

    try {
      const diffsMap = await collectMultiRepoDiffs(snapshotWorkDir, sessionState.repoSnapshots);
      if (diffsMap.size > 0) {
        const diffsObj: Record<string, string> = {};
        for (const [key, value] of diffsMap) {
          diffsObj[key] = value;
        }
        snapshotDiffs = JSON.stringify({ version: 1, diffs: diffsObj });
      }
      logger.info(`Multi-repo snapshot: ${sessionState.repoSnapshots.length} repo(s), ${diffsMap.size} diff(s)`);
    } catch (err) {
      logger.debug(`Multi-repo diff collection failed: ${err instanceof Error ? err.message : String(err)}`);
    }
  }

  const jsonlObjectKey = buildObjectKey(uploadMeta, '.jsonl.gz');
  // diff key includes timestamp so each upload is preserved (not overwritten)
  const diffTimestamp = new Date().toISOString().replace(/[:.]/g, '-');

  // === DRY RUN MODE ===
  if (options.dryRun) {
    const dryResult = await dryRunForward(filePath, snapshotManifest, snapshotDiffs, uploadMeta, jsonlObjectKey, config, options.dryRunOutput);
    appendHookLog({
      event: 'forward',
      result: 'dry-run',
      sessionId: adapterMeta.sessionId,
      tool: adapterMeta.toolName,
      jsonlObjectKey,
      bucket: config.tos.bucket,
      dryRunOutput: options.dryRunOutput ?? join(process.cwd(), '.trace-dry-run'),
    });
    return dryResult;
  }

  // === REAL UPLOAD MODE ===
  // 8. Upload JSONL file
  const jsonlGzPath = await gzipFile(filePath);
  let jsonlResult;
  try {
    jsonlResult = await uploadToTOS(jsonlGzPath, jsonlObjectKey, uploadMeta, config.tos);
    logger.info(`Uploaded JSONL: ${jsonlResult.bucket}/${jsonlResult.objectKey}`);
  } finally {
    await cleanupTempFile(jsonlGzPath);
  }

  // 9. Build result. Per-forward diffs are uploaded as multi-repo diffs.
  const result: ForwardResult = { jsonl: jsonlResult };

  // 9a. Ensure user meta index file exists on TOS (best-effort, don't block)
  ensureUserMeta(config.tos, userInfo).catch(err => {
    logger.debug(`[forward] ensureUserMeta failed: ${err instanceof Error ? err.message : String(err)}`);
  });

  // 9b. Upload multi-repo snapshot files (manifest + diffs)
  let snapshotManifestKey: string | undefined;
  let snapshotDiffsKey: string | undefined;

  if (snapshotManifest) {
    const manifestObjectKey = buildObjectKey(uploadMeta, '.manifest.json.gz');
    const manifestGzPath = await gzipString(snapshotManifest);
    try {
      await uploadToTOS(manifestGzPath, manifestObjectKey, uploadMeta, config.tos);
      snapshotManifestKey = manifestObjectKey;
      logger.info(`Uploaded manifest: ${config.tos.bucket}/${manifestObjectKey}`);
    } finally {
      await cleanupTempFile(manifestGzPath);
    }
  }

  if (snapshotDiffs) {
    const diffsObjectKey = buildObjectKey(uploadMeta, `.${diffTimestamp}.diffs.json.gz`);
    const diffsGzPath = await gzipString(snapshotDiffs);
    try {
      await uploadToTOS(diffsGzPath, diffsObjectKey, uploadMeta, config.tos);
      snapshotDiffsKey = diffsObjectKey;
      logger.info(`Uploaded diffs: ${config.tos.bucket}/${diffsObjectKey}`);
    } finally {
      await cleanupTempFile(diffsGzPath);
    }
  }

  appendHookLog({
    event: 'forward',
    result: 'ok',
    sessionId: adapterMeta.sessionId,
    tool: adapterMeta.toolName,
    jsonlObjectKey: result.jsonl.objectKey,
    diffObjectKey: result.diff?.objectKey,
    snapshotManifestKey,
    snapshotDiffsKey,
    bucket: result.jsonl.bucket,
    etag: result.jsonl.etag || undefined,
  });

  logger.info('Forward complete.');
  return result;
}

/**
 * Dry-run mode: compress files and write to local output directory with metadata.
 */
async function dryRunForward(
  filePath: string,
  snapshotManifest: string | undefined,
  snapshotDiffs: string | undefined,
  meta: UploadMeta,
  jsonlObjectKey: string,
  config: { tos: { bucket: string; region: string } },
  outputDir?: string,
): Promise<ForwardResult> {
  const outDir = outputDir ?? join(process.cwd(), '.trace-dry-run');
  await mkdir(outDir, { recursive: true });

  logger.info(`[DRY-RUN] Output directory: ${outDir}`);

  // Compress and save JSONL
  const jsonlGzPath = await gzipFile(filePath);
  const { copyFile } = await import('node:fs/promises');
  const localJsonlPath = join(outDir, `${meta.sessionId}.jsonl.gz`);
  await copyFile(jsonlGzPath, localJsonlPath);
  await cleanupTempFile(jsonlGzPath);

  const jsonlGzStat = await stat(localJsonlPath);
  logger.info(`[DRY-RUN] JSONL compressed: ${localJsonlPath} (${jsonlGzStat.size} bytes)`);
  logger.info(`[DRY-RUN] Would upload to: ${config.tos.bucket}/${jsonlObjectKey}`);

  const result: ForwardResult = {
    jsonl: { objectKey: jsonlObjectKey, bucket: config.tos.bucket, etag: '(dry-run)' },
  };

  // Compress and save multi-repo snapshot files
  if (snapshotManifest) {
    const manifestGzPath = await gzipString(snapshotManifest);
    const localManifestPath = join(outDir, `${meta.sessionId}.manifest.json.gz`);
    await copyFile(manifestGzPath, localManifestPath);
    await cleanupTempFile(manifestGzPath);
    logger.info(`[DRY-RUN] Manifest: ${localManifestPath}`);
  }

  if (snapshotDiffs) {
    const diffsGzPath = await gzipString(snapshotDiffs);
    const localDiffsPath = join(outDir, `${meta.sessionId}.diffs.json.gz`);
    await copyFile(diffsGzPath, localDiffsPath);
    await cleanupTempFile(diffsGzPath);
    logger.info(`[DRY-RUN] Diffs: ${localDiffsPath}`);
  }

  // Write metadata JSON
  const metadataPath = join(outDir, `${meta.sessionId}.metadata.json`);
  const objectMetadata = buildObjectMetadata(meta);
  await writeFile(metadataPath, JSON.stringify({ uploadMeta: meta, objectMetadata, jsonlObjectKey }, null, 2));
  logger.info(`[DRY-RUN] Metadata: ${metadataPath}`);

  logger.info('[DRY-RUN] Forward complete (no actual upload).');
  return result;
}
