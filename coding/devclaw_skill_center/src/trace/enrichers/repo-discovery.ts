import { readdir, access } from 'node:fs/promises';
import { join, relative, resolve } from 'node:path';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { logger } from '../utils/logger.js';

const execFileAsync = promisify(execFile);

export interface DiscoveredRepo {
  relativePath: string;
  absolutePath: string;
}

const EXCLUDED_DIRS = new Set([
  'node_modules', '.git', 'vendor', 'dist', 'build', '__pycache__',
]);

function isHiddenDir(name: string): boolean {
  return name.startsWith('.') && name !== '.git';
}

function shouldSkip(name: string): boolean {
  return EXCLUDED_DIRS.has(name) || isHiddenDir(name);
}

async function isGitRepo(dirPath: string): Promise<boolean> {
  try {
    await access(join(dirPath, '.git'));
    return true;
  } catch {
    return false;
  }
}

async function isGitRepoViaCli(dirPath: string): Promise<boolean> {
  try {
    await execFileAsync('git', ['rev-parse', '--show-toplevel'], {
      cwd: dirPath,
      timeout: 3000,
    });
    return true;
  } catch {
    return false;
  }
}

async function scanForRepos(
  baseDir: string,
  maxDepth: number,
  signal: AbortSignal,
): Promise<string[]> {
  const found: string[] = [];

  async function scan(dir: string, depth: number): Promise<void> {
    if (signal.aborted || depth > maxDepth) return;

    let entries;
    try {
      entries = await readdir(dir, { withFileTypes: true });
    } catch {
      return;
    }

    for (const entry of entries) {
      if (signal.aborted) return;
      if (!entry.isDirectory() || shouldSkip(entry.name)) continue;

      const fullPath = join(dir, entry.name);

      if (await isGitRepo(fullPath)) {
        found.push(fullPath);
      } else if (depth < maxDepth) {
        await scan(fullPath, depth + 1);
      }
    }
  }

  await scan(baseDir, 1);
  return found;
}

/**
 * Discover git repositories under a workspace directory.
 *
 * Priority: configured repos > auto-scan.
 * - If configuredRepos is provided, only those paths are checked.
 * - Otherwise, auto-scan subdirectories up to 2 levels deep.
 * - The cwd itself is always checked and included if it's a git repo.
 *
 * @param cwd - workspace root directory
 * @param configuredRepos - optional list of relative paths from .trace/config.yaml
 */
export async function discoverRepos(
  cwd: string,
  configuredRepos?: string[],
): Promise<DiscoveredRepo[]> {
  const results: DiscoveredRepo[] = [];
  const resolvedCwd = resolve(cwd);

  const cwdIsRepo = await isGitRepoViaCli(resolvedCwd);
  if (cwdIsRepo) {
    results.push({ relativePath: '.', absolutePath: resolvedCwd });
  }

  if (configuredRepos && configuredRepos.length > 0) {
    for (const repoPath of configuredRepos) {
      const absPath = resolve(resolvedCwd, repoPath);
      if (absPath === resolvedCwd) continue;
      if (await isGitRepo(absPath)) {
        results.push({
          relativePath: relative(resolvedCwd, absPath),
          absolutePath: absPath,
        });
      } else {
        logger.debug(`Configured repo path is not a git repo: ${repoPath}`);
      }
    }
  } else {
    const ac = new AbortController();
    const timeout = setTimeout(() => ac.abort(), 5000);

    try {
      const found = await scanForRepos(resolvedCwd, 2, ac.signal);
      for (const absPath of found) {
        if (absPath === resolvedCwd) continue;
        results.push({
          relativePath: relative(resolvedCwd, absPath),
          absolutePath: absPath,
        });
      }
    } finally {
      clearTimeout(timeout);
    }
  }

  return results;
}
