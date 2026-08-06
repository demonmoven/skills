import { readFile, writeFile, unlink, mkdir } from 'node:fs/promises';
import { join } from 'node:path';
import { homedir } from 'node:os';

const SESSIONS_DIR = join(homedir(), '.trace', 'sessions');

export interface RepoSnapshot {
  relativePath: string;   // relative to cwd, e.g. "." or "repos/repo-a"
  repoUrl?: string;       // remote.origin.url
  branch: string;
  startCommit: string;    // HEAD at session start
  dirty: boolean;         // whether working tree has uncommitted changes
}

export interface SessionState {
  sessionId: string;
  startCommit: string;
  repoSnapshots?: RepoSnapshot[];   // all discovered repos at session start
  cwd: string;
  startedAt: string; // ISO timestamp
}

export async function saveSessionState(state: SessionState): Promise<void> {
  await mkdir(SESSIONS_DIR, { recursive: true });
  const filePath = join(SESSIONS_DIR, `${state.sessionId}.json`);
  await writeFile(filePath, JSON.stringify(state, null, 2), 'utf-8');
}

export async function saveSessionStateIfAbsent(state: SessionState): Promise<boolean> {
  const existing = await loadSessionState(state.sessionId);
  if (existing) return false;

  await saveSessionState(state);
  return true;
}

export async function loadSessionState(sessionId: string): Promise<SessionState | null> {
  const filePath = join(SESSIONS_DIR, `${sessionId}.json`);
  try {
    const content = await readFile(filePath, 'utf-8');
    return JSON.parse(content) as SessionState;
  } catch (err: unknown) {
    if (err instanceof Error && 'code' in err && (err as NodeJS.ErrnoException).code === 'ENOENT') {
      return null;
    }
    throw err;
  }
}

export async function cleanupSessionState(sessionId: string): Promise<void> {
  const filePath = join(SESSIONS_DIR, `${sessionId}.json`);
  try {
    await unlink(filePath);
  } catch (err: unknown) {
    if (err instanceof Error && 'code' in err && (err as NodeJS.ErrnoException).code === 'ENOENT') {
      return;
    }
    throw err;
  }
}

export { SESSIONS_DIR };
