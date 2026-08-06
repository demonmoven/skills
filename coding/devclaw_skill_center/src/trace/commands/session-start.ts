import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { saveSessionStateIfAbsent } from '../session/state.js';
import { loadConfig } from '../config/loader.js';
import { discoverRepos } from '../enrichers/repo-discovery.js';
import { collectRepoSnapshots } from '../enrichers/git.js';
import { logger } from '../utils/logger.js';
import { appendHookLog } from '../utils/hook-log.js';

const execFileAsync = promisify(execFile);

/**
 * Record the Git baseline at the start of an AI coding session.
 * Called by AI tools via Hook at session start.
 *
 * In addition to the cwd's own commit, this now discovers and snapshots
 * all git repos under the workspace (via config or auto-scan).
 */
export async function runSessionStart(sessionId: string, cwd?: string): Promise<void> {
  const workDir = cwd ?? process.cwd();

  let startCommit = '';
  let startBranch = '';
  try {
    const { stdout } = await execFileAsync('git', ['rev-parse', 'HEAD'], {
      cwd: workDir,
      timeout: 3000,
    });
    startCommit = stdout.trim();
  } catch {
    logger.warn('Not in a git repository. Session state will have empty start commit.');
  }

  try {
    const { stdout } = await execFileAsync('git', ['rev-parse', '--abbrev-ref', 'HEAD'], {
      cwd: workDir,
      timeout: 3000,
    });
    startBranch = stdout.trim();
  } catch {
    // Ignore, branch is best-effort metadata for logging
  }

  // Discover and snapshot all repos under the workspace
  let repoSnapshots;
  try {
    const config = await loadConfig(workDir);
    const repos = await discoverRepos(workDir, config.workspace?.repos);
    if (repos.length > 0) {
      repoSnapshots = await collectRepoSnapshots(repos);
      logger.info(`Discovered ${repoSnapshots.length} repo(s): ${repoSnapshots.map(r => r.relativePath).join(', ')}`);
    }
  } catch (err) {
    logger.debug(`Multi-repo discovery failed, continuing with single-repo: ${err instanceof Error ? err.message : String(err)}`);
  }

  try {
    const saved = await saveSessionStateIfAbsent({
      sessionId,
      startCommit,
      repoSnapshots,
      cwd: workDir,
      startedAt: new Date().toISOString(),
    });
    if (!saved) {
      logger.info(`Session ${sessionId} already has a start state. Keeping the original start commit.`);
    }
    logger.info(`Session ${sessionId} started. Start commit: ${startCommit || '(none)'}`);
    appendHookLog({
      event: 'session_start',
      result: 'ok',
      sessionId,
      gitCommit: startCommit || undefined,
      gitBranch: startBranch || undefined,
      repoCount: repoSnapshots?.length,
    });
  } catch (err) {
    appendHookLog({
      event: 'session_start',
      result: 'error',
      sessionId,
      errorCode: err instanceof Error ? err.constructor.name : 'Unknown',
      errorMessage: err instanceof Error ? err.message : String(err),
    });
    throw err;
  }
}
