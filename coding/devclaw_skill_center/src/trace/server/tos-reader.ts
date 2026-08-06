import { gunzipSync } from 'node:zlib';
import { createRequire } from 'node:module';
import type { TosConfig } from '../config/schema.js';
import type { UserMetaRecord } from '../upload/user-meta.js';
import { logger } from '../utils/logger.js';

const require = createRequire(import.meta.url);

// ─── Types ─────────────────────────────────────────────────────────

export interface SessionListItem {
  sessionId: string;
  toolName: string;
  userId: string;
  date: string;
  objectKey: string;
  /** Whether this session was stored as a raw JSONL or a session bundle (tar.gz). */
  format: 'jsonl' | 'bundle';
  /** User's department (populated from user meta index, may be undefined for historical data). */
  department?: string;
  /** User's real name / nickname (populated from user meta index). */
  nickname?: string;
}

export interface ListSessionsOptions {
  userId?: string;
  toolName?: string;
  sessionId?: string;
  dateFrom?: string;
  dateTo?: string;
  department?: string;
  page?: number;
  pageSize?: number;
}

// ─── Object Key Parsing ────────────────────────────────────────────

/**
 * Parse a TOS object key like `xtrace/{userId}/{date}/{toolName}/{sessionId}.jsonl.gz`
 * or `xtrace/{userId}/{date}/{toolName}/{sessionId}.session.tar.gz`
 * into structured session metadata. Returns null if the key doesn't match the expected pattern.
 */
export function parseObjectKey(key: string): SessionListItem | null {
  // Standard format: xtrace/{userId}/{date}/{toolName}/{sessionId}.(jsonl.gz|session.tar.gz)
  // For Trae threads, sessionId = threadId, toolName = "trae"
  const match = key.match(
    /^xtrace\/([^/]+)\/(\d{4}-\d{2}-\d{2})\/([^/]+)\/([^/]+)\.(jsonl\.gz|session\.tar\.gz)$/
  );
  if (!match) return null;

  const [, userId, date, toolName, sessionId, suffix] = match;
  return {
    sessionId: sessionId!,
    toolName: toolName!,
    userId: userId!,
    date: date!,
    objectKey: key,
    format: suffix === 'session.tar.gz' ? 'bundle' : 'jsonl',
  };
}

/**
 * Build the analysis result cache key from a session's metadata.
 * Format: xtrace-analysis/{userId}/{date}/{toolName}/{sessionId}.analysis.json
 *
 * @deprecated Use {@link buildEvalCacheKey} instead. Full analysis results are
 * no longer cached to TOS — only LLM evaluation results are persisted.
 */
export function buildAnalysisCacheKey(item: SessionListItem): string {
  return `xtrace-analysis/${item.userId}/${item.date}/${item.toolName}/${item.sessionId}.analysis.json`;
}

/**
 * Build the TOS key for caching an LLM evaluation result.
 * Format: xtrace-eval/{userId}/{date}/{toolName}/{sessionId}.eval.json
 */
export function buildEvalCacheKey(item: SessionListItem): string {
  return `xtrace-eval/${item.userId}/${item.date}/${item.toolName}/${item.sessionId}.eval.json`;
}

/**
 * Build the TOS key for a session bundle (tar.gz).
 * Format: xtrace/{userId}/{date}/{toolName}/{sessionId}.session.tar.gz
 *
 * @deprecated New uploads always use `.session.tar.gz` as the primary key.
 * This helper is only needed for legacy `.jsonl.gz` entries that may have
 * a companion bundle uploaded alongside them.
 */
export function buildBundleKey(item: SessionListItem): string {
  return `xtrace/${item.userId}/${item.date}/${item.toolName}/${item.sessionId}.session.tar.gz`;
}

// ─── TOS Operations ────────────────────────────────────────────────

function createTosClient(config: TosConfig) {
  const Tos = require('@byted-service/tos');
  return new Tos({
    bucket: config.bucket,
    accessKey: config.accessKey,
    secretKey: config.secretKey,
    signatureVersion: 'sign_v1',
    disableWatcher: true,
    reqTimeout: 30000,
    ...(config.endpoint ? { endpoints: config.endpoint } : {}),
  });
}

// ─── User Meta Index ───────────────────────────────────────────────

/** In-memory cache of userId → UserMetaRecord, populated on first use. */
let userMetaCache: Map<string, UserMetaRecord> | null = null;

/**
 * Load all user meta files from TOS (xtrace/_meta/users/*.json).
 * Returns a userId → UserMetaRecord map. Results are cached in-memory
 * for the lifetime of the serve process.
 */
export async function loadUserMetaIndex(config: TosConfig, force = false): Promise<Map<string, UserMetaRecord>> {
  if (userMetaCache && !force) return userMetaCache;

  const tos = createTosClient(config);
  const map = new Map<string, UserMetaRecord>();

  try {
    const prefix = 'xtrace/_meta/users/';
    const objects = await listAllObjects(tos, prefix);
    const jsonFiles = objects.filter(o => o.key.endsWith('.json'));

    logger.info(`[user-meta] Loading ${jsonFiles.length} user meta files from TOS`);

    // Download all user meta files in parallel (they're tiny)
    const downloads = jsonFiles.map(async (obj) => {
      try {
        const result = await tos.getObject(obj.key);
        const buffer = result?.objectBuffer ?? result?.data ?? result;
        if (!Buffer.isBuffer(buffer)) return;
        const record = JSON.parse(buffer.toString('utf-8')) as UserMetaRecord;
        if (record.userId) {
          map.set(record.userId, record);
        }
      } catch (err) {
        logger.debug(`[user-meta] Failed to download ${obj.key}: ${err instanceof Error ? err.message : String(err)}`);
      }
    });

    await Promise.all(downloads);
    logger.info(`[user-meta] Loaded ${map.size} user records`);
  } catch (err) {
    logger.warn(`[user-meta] Failed to load user meta index: ${err instanceof Error ? err.message : String(err)}`);
  } finally {
    tos.destroy();
  }

  userMetaCache = map;
  return map;
}

/**
 * Recursively list all objects under a prefix.
 * Subdirectories at the same level are listed in parallel to minimize latency.
 */
async function listAllObjects(
  tos: ReturnType<typeof createTosClient>,
  prefix: string,
): Promise<Array<{ key: string }>> {
  const rawResult = await tos.listPrefix(prefix);
  const result = rawResult?.result ?? rawResult;
  const objects: Array<{ key: string }> = result?.objects ?? [];
  const subdirs: string[] = result?.commonPrefix ?? [];

  logger.debug(`listAllObjects(${prefix}): objects=${objects.length}, subdirs=${subdirs.length}`);

  // Recurse into subdirectories IN PARALLEL
  if (subdirs.length > 0) {
    const subResults = await Promise.all(
      subdirs.map(subdir => listAllObjects(tos, subdir))
    );
    for (const subObjects of subResults) {
      objects.push(...subObjects);
    }
  }

  return objects;
}

/**
 * List all session JSONL objects under the `xtrace/` prefix in TOS.
 * Subdirectories are listed in parallel for speed.
 * Supports filtering by userId, toolName, date range, and department.
 */
export async function listSessionObjects(
  config: TosConfig,
  options: ListSessionsOptions = {},
): Promise<{ sessions: SessionListItem[]; total: number; departments: string[] }> {
  // Load user meta index in parallel with session listing
  const [userMeta] = await Promise.all([
    loadUserMetaIndex(config),
  ]);

  // If filtering by department, resolve matching userIds first
  let departmentUserIds: Set<string> | null = null;
  if (options.department) {
    const needle = options.department.toLowerCase();
    departmentUserIds = new Set<string>();
    for (const [uid, meta] of userMeta) {
      if (meta.department?.toLowerCase().includes(needle)) {
        departmentUserIds.add(uid);
      }
    }
    // If no users match the department, return empty
    if (departmentUserIds.size === 0) {
      return { sessions: [], total: 0, departments: getAllDepartments(userMeta) };
    }
  }

  const tos = createTosClient(config);

  try {
    // Narrow prefix if userId filter is specified
    let prefix = 'xtrace/';
    if (options.userId) {
      prefix = `xtrace/${options.userId}/`;
    }

    const objects = await listAllObjects(tos, prefix);
    let sessions = objects
      .filter((obj: { key: string }) => obj.key.endsWith('.jsonl.gz') || obj.key.endsWith('.session.tar.gz'))
      .map((obj: { key: string }) => parseObjectKey(obj.key))
      .filter((s): s is SessionListItem => s !== null);

    // Deduplicate: if same sessionId exists in both formats, prefer bundle
    const deduped = new Map<string, SessionListItem>();
    for (const s of sessions) {
      const existing = deduped.get(s.sessionId);
      if (!existing || s.format === 'bundle') {
        deduped.set(s.sessionId, s);
      }
    }
    sessions = Array.from(deduped.values());

    // Enrich sessions with department/nickname from user meta index
    for (const s of sessions) {
      const meta = userMeta.get(s.userId);
      if (meta) {
        s.department = meta.department;
        s.nickname = meta.nickname;
      }
    }

    // Sort by date descending (newest first)
    sessions.sort((a, b) => b.date.localeCompare(a.date));

    // Apply filters
    if (options.sessionId) {
      const needle = options.sessionId.toLowerCase();
      sessions = sessions.filter(s => s.sessionId.toLowerCase().includes(needle));
    }
    if (options.toolName) {
      sessions = sessions.filter(s => s.toolName === options.toolName);
    }
    if (options.dateFrom) {
      sessions = sessions.filter(s => s.date >= options.dateFrom!);
    }
    if (options.dateTo) {
      sessions = sessions.filter(s => s.date <= options.dateTo!);
    }
    if (departmentUserIds) {
      sessions = sessions.filter(s => departmentUserIds!.has(s.userId));
    }

    // Collect all unique departments for the filter dropdown
    const departments = getAllDepartments(userMeta);

    // Paginate
    const total = sessions.length;
    const page = options.page ?? 1;
    const pageSize = options.pageSize ?? 20;
    const start = (page - 1) * pageSize;
    sessions = sessions.slice(start, start + pageSize);

    return { sessions, total, departments };
  } finally {
    tos.destroy();
  }
}

/** Collect all unique department names from the user meta index. */
function getAllDepartments(userMeta: Map<string, UserMetaRecord>): string[] {
  const deptSet = new Set<string>();
  for (const meta of userMeta.values()) {
    if (meta.department) deptSet.add(meta.department);
  }
  return Array.from(deptSet).sort();
}

/**
 * Download a .jsonl.gz object from TOS and decompress to string.
 * Returns the decompressed content together with the TOS ETag for cache validation.
 */
export async function downloadAndDecompress(
  config: TosConfig,
  objectKey: string,
): Promise<{ content: string; etag: string }> {
  const tos = createTosClient(config);

  try {
    const result = await tos.getObject(objectKey);
    const buffer = result?.objectBuffer ?? result?.data ?? result;

    if (!Buffer.isBuffer(buffer)) {
      throw new Error(`TOS download returned unexpected type: ${typeof buffer}, keys: ${Object.keys(result ?? {})}`);
    }

    const etag: string = result?.headers?.etag ?? result?.headers?.['x-tos-md5'] ?? '';
    const decompressed = gunzipSync(buffer);
    return { content: decompressed.toString('utf-8'), etag };
  } finally {
    tos.destroy();
  }
}

/**
 * Download a raw buffer from TOS (no decompression).
 * Used for session bundles (tar.gz) that need to be extracted, not gunzipped.
 * Returns null if the object doesn't exist.
 */
export async function downloadRawBuffer(
  config: TosConfig,
  objectKey: string,
): Promise<{ buffer: Buffer; etag: string } | null> {
  const tos = createTosClient(config);

  try {
    const result = await tos.getObject(objectKey);
    const buffer = result?.objectBuffer ?? result?.data ?? result;
    if (!Buffer.isBuffer(buffer)) return null;
    const etag: string = result?.headers?.etag ?? result?.headers?.['x-tos-md5'] ?? '';
    return { buffer, etag };
  } catch (err) {
    logger.debug(`Raw buffer download failed for ${objectKey}: ${err instanceof Error ? err.message : String(err)}`);
    return null;
  } finally {
    tos.destroy();
  }
}

/**
 * HEAD request to TOS — retrieve object metadata (ETag, content-length,
 * and custom x-tos-meta-* headers) without downloading the body.
 * Returns null if the object doesn't exist.
 */
export async function headObject(
  config: TosConfig,
  objectKey: string,
): Promise<{
  etag: string;
  contentLength: number;
  gitUrl?: string;
  gitCommit?: string;
} | null> {
  const tos = createTosClient(config);

  try {
    const headers = await tos.headObject(objectKey);
    return {
      etag: headers?.etag ?? '',
      contentLength: parseInt(headers?.['content-length'] ?? headers?.size ?? '0', 10),
      gitUrl: headers?.['x-tos-meta-git-url'] ?? undefined,
      gitCommit: headers?.['x-tos-meta-git-commit'] ?? undefined,
    };
  } catch {
    return null;
  } finally {
    tos.destroy();
  }
}

/**
 * Convenience: get git repo info (URL + commit SHA) from a TOS object's metadata.
 * Returns null if the object doesn't exist or git info is missing.
 */
export async function getObjectGitInfo(
  config: TosConfig,
  objectKey: string,
): Promise<{ gitUrl: string; gitCommit: string } | null> {
  const meta = await headObject(config, objectKey);
  if (!meta?.gitUrl || !meta?.gitCommit) return null;
  return { gitUrl: meta.gitUrl, gitCommit: meta.gitCommit };
}

/**
 * Upload analysis result JSON to TOS for caching.
 *
 * @deprecated Full analysis results are no longer cached. Use {@link uploadEvalResult}.
 */
export async function uploadAnalysisResult(
  config: TosConfig,
  objectKey: string,
  data: unknown,
): Promise<void> {
  const tos = createTosClient(config);

  try {
    const content = Buffer.from(JSON.stringify(data), 'utf-8');
    await tos.upload(content, objectKey, {
      headers: {
        'Content-Type': 'application/json',
        'x-tos-meta-type': 'analysis-cache',
      },
    });
    logger.debug(`Uploaded analysis cache: ${objectKey}`);
  } finally {
    tos.destroy();
  }
}

/**
 * Download cached analysis result JSON from TOS.
 * Returns null if the object doesn't exist.
 *
 * @deprecated Full analysis results are no longer cached. Use {@link downloadEvalResult}.
 */
export async function downloadAnalysisResult(
  config: TosConfig,
  objectKey: string,
): Promise<unknown | null> {
  const tos = createTosClient(config);

  try {
    const result = await tos.getObject(objectKey);
    const buffer = result?.objectBuffer ?? result?.data ?? result;

    if (!Buffer.isBuffer(buffer)) {
      return null;
    }

    return JSON.parse(buffer.toString('utf-8'));
  } catch (err) {
    // Object not found or other error — return null for cache miss
    logger.debug(`Analysis cache miss for ${objectKey}: ${err instanceof Error ? err.message : String(err)}`);
    return null;
  } finally {
    tos.destroy();
  }
}

// ─── Eval Cache Operations ─────────────────────────────────────────

/**
 * Upload an LLM evaluation result to TOS for caching.
 */
export async function uploadEvalResult(
  config: TosConfig,
  objectKey: string,
  data: import('../analyzer/types.js').EvalResult,
): Promise<void> {
  const tos = createTosClient(config);

  try {
    const content = Buffer.from(JSON.stringify(data), 'utf-8');
    await tos.upload(content, objectKey, {
      headers: {
        'Content-Type': 'application/json',
        'x-tos-meta-type': 'eval-cache',
      },
    });
    logger.debug(`Uploaded eval cache: ${objectKey}`);
  } finally {
    tos.destroy();
  }
}

/**
 * Download a cached LLM evaluation result from TOS.
 * Returns null if the object doesn't exist or parsing fails.
 */
export async function downloadEvalResult(
  config: TosConfig,
  objectKey: string,
): Promise<import('../analyzer/types.js').EvalResult | null> {
  const tos = createTosClient(config);

  try {
    const result = await tos.getObject(objectKey);
    const buffer = result?.objectBuffer ?? result?.data ?? result;

    if (!Buffer.isBuffer(buffer)) {
      return null;
    }

    return JSON.parse(buffer.toString('utf-8'));
  } catch (err) {
    logger.debug(`Eval cache miss for ${objectKey}: ${err instanceof Error ? err.message : String(err)}`);
    return null;
  } finally {
    tos.destroy();
  }
}
