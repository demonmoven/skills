import { readFile, writeFile, rename, mkdir, unlink } from 'node:fs/promises';
import { join } from 'node:path';
import { homedir } from 'node:os';
import { randomBytes } from 'node:crypto';
import { logger } from '../utils/logger.js';
import type { SessionTree } from '../analyzer/types.js';

// ─── Constants ─────────────────────────────────────────────────────

const DEFAULT_CACHE_DIR = join(homedir(), '.xtrace', 'cache', 'jsonl');

// ─── Public API ────────────────────────────────────────────────────

/**
 * Build the local cache file path for a given TOS object key.
 * Replaces `/` with `_` and strips the `.gz` suffix so the cached file
 * represents the decompressed JSONL content.
 *
 * @param objectKey  TOS object key, e.g. `xtrace/user/2026-04-09/claude-code/sess.jsonl.gz`
 * @param cacheDir   Override cache directory (useful for testing)
 */
export function buildCachePath(objectKey: string, cacheDir?: string): string {
  const dir = cacheDir ?? DEFAULT_CACHE_DIR;
  const sanitized = objectKey.replace(/\//g, '_').replace(/\.gz$/, '');
  return join(dir, sanitized);
}

/**
 * Read cached SessionTree and its associated ETag from the local filesystem.
 * Returns `null` on cache miss.
 */
export async function getCachedSessionTree(
  objectKey: string,
  cacheDir?: string,
): Promise<{ tree: SessionTree; etag: string } | null> {
  const filePath = buildCachePath(objectKey, cacheDir);
  const metaPath = filePath + '.meta';

  try {
    const [content, metaRaw] = await Promise.all([
      readFile(filePath, 'utf-8'),
      readFile(metaPath, 'utf-8'),
    ]);
    const meta = JSON.parse(metaRaw) as { etag?: string };
    if (!meta.etag) return null;
    const tree = JSON.parse(content) as SessionTree;
    return { tree, etag: meta.etag };
  } catch {
    return null;
  }
}

/**
 * Write a SessionTree and its ETag to the local cache.
 * Uses atomic write (temp file + rename) to avoid partial reads.
 */
export async function putCachedSessionTree(
  objectKey: string,
  tree: SessionTree,
  etag: string,
  cacheDir?: string,
): Promise<void> {
  const filePath = buildCachePath(objectKey, cacheDir);
  const metaPath = filePath + '.meta';
  const dir = cacheDir ?? DEFAULT_CACHE_DIR;

  await mkdir(dir, { recursive: true });

  const suffix = randomBytes(6).toString('hex');
  const tmpContent = filePath + `.tmp.${suffix}`;
  const tmpMeta = metaPath + `.tmp.${suffix}`;

  try {
    await Promise.all([
      writeFile(tmpContent, JSON.stringify(tree), 'utf-8'),
      writeFile(tmpMeta, JSON.stringify({ etag }), 'utf-8'),
    ]);
    await Promise.all([
      rename(tmpContent, filePath),
      rename(tmpMeta, metaPath),
    ]);
  } catch (err) {
    await unlink(tmpContent).catch(() => {});
    await unlink(tmpMeta).catch(() => {});
    logger.debug(`Failed to write SessionTree cache for ${objectKey}: ${err instanceof Error ? err.message : String(err)}`);
  }
}

