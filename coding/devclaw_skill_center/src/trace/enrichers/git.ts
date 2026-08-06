import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { resolve } from 'node:path';
import { loadSessionState } from '../session/state.js';
import type { RepoSnapshot } from '../session/state.js';
import type { DiscoveredRepo } from './repo-discovery.js';
import { logger } from '../utils/logger.js';

const execFileAsync = promisify(execFile);

const GIT_TIMEOUT_MS = 3000;

export interface GitContext {
  repoUrl?: string;
  branch?: string;
  commitHead?: string;
  workspaceRoot?: string;
  repoDirty?: boolean;
  diffFromStart?: string;
}

export async function runGit(args: string[], cwd: string): Promise<string> {
  const { stdout } = await execFileAsync('git', args, {
    cwd,
    timeout: GIT_TIMEOUT_MS,
    encoding: 'utf-8',
  });
  return stdout.trim();
}

/**
 * Collect Git context from the given working directory.
 * Each field is collected independently - if one fails, others still populate.
 *
 * @param cwd - working directory to search for .git
 * @param sessionId - optional session ID to look up start commit for diff calculation
 */
export async function collectGitContext(cwd: string, sessionId?: string): Promise<GitContext> {
  const ctx: GitContext = {};

  // Get workspace root (also validates we're in a git repo)
  try {
    ctx.workspaceRoot = await runGit(['rev-parse', '--show-toplevel'], cwd);
  } catch {
    logger.debug('Not a git repository or git not available, skipping git context');
    return ctx;
  }

  const gitCwd = ctx.workspaceRoot;

  // Collect fields in parallel, each independently degrading
  const tasks = [
    // repoUrl
    (async () => {
      try {
        ctx.repoUrl = await runGit(['config', '--get', 'remote.origin.url'], gitCwd);
      } catch {
        logger.debug('Could not get remote.origin.url');
      }
    })(),
    // branch
    (async () => {
      try {
        ctx.branch = await runGit(['rev-parse', '--abbrev-ref', 'HEAD'], gitCwd);
      } catch {
        logger.debug('Could not get current branch');
      }
    })(),
    // commitHead
    (async () => {
      try {
        ctx.commitHead = await runGit(['rev-parse', 'HEAD'], gitCwd);
      } catch {
        logger.debug('Could not get HEAD commit');
      }
    })(),
    // repoDirty
    (async () => {
      try {
        const status = await runGit(['status', '--porcelain'], gitCwd);
        ctx.repoDirty = status.length > 0;
      } catch {
        logger.debug('Could not check repo dirty status');
      }
    })(),
  ];

  await Promise.all(tasks);

  // diffFromStart - depends on sessionId and session state
  if (sessionId) {
    try {
      const sessionState = await loadSessionState(sessionId);
      if (sessionState?.startCommit) {
        ctx.diffFromStart = await runGit(['diff', sessionState.startCommit], gitCwd);
      } else {
        // No session state, fallback to uncommitted changes
        ctx.diffFromStart = await runGit(['diff', 'HEAD'], gitCwd);
      }
    } catch {
      logger.debug('Could not compute diff from start');
    }
  } else {
    // No sessionId, just get uncommitted changes
    try {
      ctx.diffFromStart = await runGit(['diff', 'HEAD'], gitCwd);
    } catch {
      logger.debug('Could not compute diff');
    }
  }

  return ctx;
}

/**
 * Collect snapshot metadata for each discovered repo in parallel.
 * Used at session-start to record the starting state of all repos.
 */
export async function collectRepoSnapshots(
  repos: DiscoveredRepo[],
): Promise<RepoSnapshot[]> {
  const snapshots = await Promise.all(
    repos.map(async (repo): Promise<RepoSnapshot> => {
      const dir = repo.absolutePath;
      let startCommit = '';
      let branch = '';
      let repoUrl: string | undefined;
      let dirty = false;

      const tasks = [
        (async () => {
          try {
            startCommit = await runGit(['rev-parse', 'HEAD'], dir);
          } catch {
            logger.debug(`Could not get HEAD for ${repo.relativePath}`);
          }
        })(),
        (async () => {
          try {
            branch = await runGit(['rev-parse', '--abbrev-ref', 'HEAD'], dir);
          } catch {
            logger.debug(`Could not get branch for ${repo.relativePath}`);
          }
        })(),
        (async () => {
          try {
            repoUrl = await runGit(['config', '--get', 'remote.origin.url'], dir);
          } catch {
            logger.debug(`Could not get remote URL for ${repo.relativePath}`);
          }
        })(),
        (async () => {
          try {
            const status = await runGit(['status', '--porcelain'], dir);
            dirty = status.length > 0;
          } catch {
            logger.debug(`Could not check dirty status for ${repo.relativePath}`);
          }
        })(),
      ];

      await Promise.all(tasks);

      return {
        relativePath: repo.relativePath,
        repoUrl,
        branch,
        startCommit,
        dirty,
      };
    }),
  );

  return snapshots;
}

/**
 * Collect diffs for multiple repos by comparing current state to their start commits.
 * Used at forward time to compute what changed during the session.
 *
 * @returns Map from relativePath to diff content (only non-empty diffs included)
 */
export async function collectMultiRepoDiffs(
  cwd: string,
  repoSnapshots: RepoSnapshot[],
): Promise<Map<string, string>> {
  const diffs = new Map<string, string>();

  await Promise.all(
    repoSnapshots.map(async (snap) => {
      if (!snap.startCommit) return;

      const dir = resolve(cwd, snap.relativePath);
      try {
        const diff = await runGit(['diff', snap.startCommit], dir);
        if (diff.length > 0) {
          diffs.set(snap.relativePath, diff);
        }
      } catch {
        logger.debug(`Could not compute diff for ${snap.relativePath}`);
      }
    }),
  );

  return diffs;
}
