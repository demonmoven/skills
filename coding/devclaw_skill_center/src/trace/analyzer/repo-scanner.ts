import { execFile as execFileCb } from 'node:child_process';
import { promisify } from 'node:util';
import { join } from 'node:path';
import { homedir } from 'node:os';
import { createHash } from 'node:crypto';
import { access, rm } from 'node:fs/promises';
import { classifyPath, normalizePath } from './harness-metrics.js';
import { logger } from '../utils/logger.js';

const execFile = promisify(execFileCb);

// ─── Constants ─────────────────────────────────────────────────────

const CACHE_DIR = join(homedir(), '.xtrace', 'cache', 'repos');
const GIT_TIMEOUT_MS = 120_000; // 120s for clone/fetch

// ─── Public API ────────────────────────────────────────────────────

/**
 * Scan a git repository at a specific commit and return all harness file paths.
 *
 * Flow:
 *   1. Ensure a bare clone of the repo exists in the local cache
 *   2. Fetch latest objects (so the target commit is reachable)
 *   3. Run `git ls-tree -r --name-only <commit>` to list all files
 *   4. Filter through classifyPath() to keep only harness artifacts
 *
 * Returns normalized repo-relative paths (e.g. "docs/rules/invariants.md").
 * Returns empty array on any failure (graceful degradation).
 */
export async function scanRepoHarnessFiles(
  gitUrl: string,
  gitCommit: string,
): Promise<string[]> {
  try {
    const repoDir = getRepoCacheDir(gitUrl);
    await ensureBareClone(gitUrl, repoDir);
    await fetchOrigin(repoDir);
    const allFiles = await listFilesAtCommit(repoDir, gitCommit);
    return allFiles
      .map(f => {
        const category = classifyPath(f);
        return category ? normalizePath(f) : null;
      })
      .filter((f): f is string => f !== null);
  } catch (err) {
    logger.error(`[repo-scanner] Failed to scan ${gitUrl}@${gitCommit.slice(0, 8)}: ${err instanceof Error ? err.message : String(err)}`);
    return [];
  }
}

/**
 * Remove the cached bare clone for a given repo URL.
 */
export async function clearRepoCache(gitUrl: string): Promise<void> {
  const repoDir = getRepoCacheDir(gitUrl);
  await rm(repoDir, { recursive: true, force: true });
  logger.info(`[repo-scanner] Cleared cache for ${gitUrl}`);
}

// ─── Internals ─────────────────────────────────────────────────────

/**
 * Compute the local cache directory for a repo URL.
 * Uses SHA-256 hash of the URL to avoid path conflicts.
 */
function getRepoCacheDir(gitUrl: string): string {
  const hash = createHash('sha256').update(gitUrl).digest('hex').slice(0, 16);
  return join(CACHE_DIR, `${hash}.git`);
}

/**
 * Ensure a bare clone exists. If the directory already exists, skip clone.
 */
async function ensureBareClone(gitUrl: string, repoDir: string): Promise<void> {
  try {
    await access(repoDir);
    // Directory exists — bare clone already cached
    return;
  } catch {
    // Directory doesn't exist — need to clone
  }

  logger.info(`[repo-scanner] Bare cloning ${gitUrl} → ${repoDir}`);
  await execFile('git', ['clone', '--bare', '--filter=blob:none', gitUrl, repoDir], {
    timeout: GIT_TIMEOUT_MS,
    env: { ...process.env },
  });
}

/**
 * Fetch latest objects from origin so the target commit is reachable.
 */
async function fetchOrigin(repoDir: string): Promise<void> {
  try {
    await execFile('git', ['fetch', 'origin'], {
      timeout: GIT_TIMEOUT_MS,
      cwd: repoDir,
      env: { ...process.env },
    });
  } catch (err) {
    // Fetch failure is non-fatal — the commit might already be local
    logger.debug(`[repo-scanner] git fetch failed (non-fatal): ${err instanceof Error ? err.message : String(err)}`);
  }
}

/**
 * List all files at a specific commit using `git ls-tree`.
 * Returns repo-relative paths.
 */
async function listFilesAtCommit(repoDir: string, commit: string): Promise<string[]> {
  const { stdout } = await execFile('git', ['ls-tree', '-r', '--name-only', commit], {
    timeout: 30_000,
    cwd: repoDir,
    maxBuffer: 10 * 1024 * 1024,
    env: { ...process.env },
  });
  return stdout.trim().split('\n').filter(Boolean);
}
