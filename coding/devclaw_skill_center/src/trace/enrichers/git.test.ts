import { describe, it, expect, vi, beforeEach } from 'vitest';

// Mock child_process to avoid needing a real git repo
vi.mock('node:child_process', () => {
  return {
    execFile: vi.fn(),
  };
});

// Mock session state
vi.mock('../session/state.js', () => ({
  loadSessionState: vi.fn().mockResolvedValue(null),
}));

import { execFile } from 'node:child_process';
import { loadSessionState } from '../session/state.js';
import { collectGitContext, collectRepoSnapshots, collectMultiRepoDiffs } from './git.js';

const mockExecFile = vi.mocked(execFile);

function setupGitMock(responses: Record<string, string>): void {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  mockExecFile.mockImplementation((_cmd: string, args: any, _opts: any, _cb?: any) => {
    const callback = typeof _opts === 'function' ? _opts : _cb;
    const argsArray = args as string[];
    const key = argsArray.join(' ');

    for (const [pattern, value] of Object.entries(responses)) {
      if (key.includes(pattern)) {
        callback(null, { stdout: value + '\n', stderr: '' });
        return undefined as never;
      }
    }

    callback(new Error(`git command not mocked: ${key}`), { stdout: '', stderr: '' });
    return undefined as never;
  });
}

describe('collectGitContext', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('collects full git context from a git repo', async () => {
    setupGitMock({
      '--show-toplevel': '/Users/test/project',
      'remote.origin.url': 'git@code.byted.org:team/repo.git',
      '--abbrev-ref HEAD': 'feature/test',
      'rev-parse HEAD': 'abc123def456789',
      '--porcelain': 'M src/file.ts',
      'diff HEAD': 'diff --git a/file.ts b/file.ts\n+new line',
    });

    const ctx = await collectGitContext('/Users/test/project');

    expect(ctx.workspaceRoot).toBe('/Users/test/project');
    expect(ctx.repoUrl).toBe('git@code.byted.org:team/repo.git');
    expect(ctx.branch).toBe('feature/test');
    expect(ctx.commitHead).toBe('abc123def456789');
    expect(ctx.repoDirty).toBe(true);
    expect(ctx.diffFromStart).toContain('diff --git');
  });

  it('returns empty context in non-git directory', async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockExecFile.mockImplementation((_cmd: string, _args: any, _opts: any, _cb?: any) => {
      const callback = typeof _opts === 'function' ? _opts : _cb;
      callback(new Error('not a git repo'), { stdout: '', stderr: '' });
      return undefined as never;
    });

    const ctx = await collectGitContext('/tmp/not-a-repo');
    expect(ctx).toEqual({});
  });

  it('uses session state start commit for diff', async () => {
    vi.mocked(loadSessionState).mockResolvedValue({
      sessionId: 'sess-1',
      startCommit: 'start123',
      cwd: '/test',
      startedAt: '2026-01-01T00:00:00Z',
    });

    setupGitMock({
      '--show-toplevel': '/Users/test/project',
      'remote.origin.url': 'https://github.com/test/repo.git',
      '--abbrev-ref HEAD': 'main',
      'rev-parse HEAD': 'current456',
      '--porcelain': '',
      'diff start123': 'diff --git a/file.ts\n+added line',
    });

    const ctx = await collectGitContext('/Users/test/project', 'sess-1');

    expect(ctx.repoDirty).toBe(false);
    expect(ctx.diffFromStart).toContain('diff --git');
  });

  it('falls back to diff HEAD when no session state', async () => {
    vi.mocked(loadSessionState).mockResolvedValue(null);

    setupGitMock({
      '--show-toplevel': '/Users/test/project',
      'remote.origin.url': 'https://github.com/test/repo.git',
      '--abbrev-ref HEAD': 'main',
      'rev-parse HEAD': 'abc123',
      '--porcelain': '',
      'diff HEAD': '',
    });

    const ctx = await collectGitContext('/Users/test/project', 'no-session');
    expect(ctx.diffFromStart).toBe('');
  });
});

describe('collectRepoSnapshots', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('collects snapshots for multiple repos in parallel', async () => {
    setupGitMock({
      'rev-parse HEAD': 'commit-abc',
      '--abbrev-ref HEAD': 'main',
      'remote.origin.url': 'git@code.byted.org:team/repo.git',
      '--porcelain': '',
    });

    const repos = [
      { relativePath: '.', absolutePath: '/workspace' },
      { relativePath: 'repos/sub-a', absolutePath: '/workspace/repos/sub-a' },
    ];

    const snapshots = await collectRepoSnapshots(repos);

    expect(snapshots).toHaveLength(2);
    expect(snapshots[0]!.relativePath).toBe('.');
    expect(snapshots[0]!.startCommit).toBe('commit-abc');
    expect(snapshots[0]!.branch).toBe('main');
    expect(snapshots[0]!.dirty).toBe(false);
    expect(snapshots[1]!.relativePath).toBe('repos/sub-a');
  });

  it('handles individual repo failures gracefully', async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockExecFile.mockImplementation((_cmd: string, _args: any, opts: any, _cb?: any) => {
      const callback = typeof opts === 'function' ? opts : _cb;
      const cwd = (typeof opts === 'object' ? opts.cwd : undefined) as string | undefined;

      if (cwd === '/workspace/repos/broken') {
        callback(new Error('git failed'), { stdout: '', stderr: '' });
      } else {
        const argsArray = _args as string[];
        const key = argsArray.join(' ');
        if (key.includes('rev-parse HEAD') && !key.includes('--abbrev-ref')) {
          callback(null, { stdout: 'ok-commit\n', stderr: '' });
        } else if (key.includes('--abbrev-ref')) {
          callback(null, { stdout: 'main\n', stderr: '' });
        } else if (key.includes('--porcelain')) {
          callback(null, { stdout: '\n', stderr: '' });
        } else {
          callback(new Error('not mocked'), { stdout: '', stderr: '' });
        }
      }
      return undefined as never;
    });

    const repos = [
      { relativePath: '.', absolutePath: '/workspace' },
      { relativePath: 'repos/broken', absolutePath: '/workspace/repos/broken' },
    ];

    const snapshots = await collectRepoSnapshots(repos);

    expect(snapshots).toHaveLength(2);
    expect(snapshots[0]!.startCommit).toBe('ok-commit');
    // broken repo should have empty defaults
    expect(snapshots[1]!.startCommit).toBe('');
    expect(snapshots[1]!.branch).toBe('');
  });
});

describe('collectMultiRepoDiffs', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('collects diffs for repos with start commits', async () => {
    setupGitMock({
      'diff start-abc': 'diff --git a/file.ts\n+changed',
      'diff start-def': '',
    });

    const snapshots = [
      { relativePath: '.', branch: 'main', startCommit: 'start-abc', dirty: true },
      { relativePath: 'repos/sub', branch: 'dev', startCommit: 'start-def', dirty: false },
    ];

    const diffs = await collectMultiRepoDiffs('/workspace', snapshots);

    expect(diffs.size).toBe(1);
    expect(diffs.get('.')).toContain('diff --git');
    expect(diffs.has('repos/sub')).toBe(false); // empty diff not included
  });

  it('skips repos without start commit', async () => {
    const snapshots = [
      { relativePath: '.', branch: 'main', startCommit: '', dirty: false },
    ];

    const diffs = await collectMultiRepoDiffs('/workspace', snapshots);
    expect(diffs.size).toBe(0);
    expect(mockExecFile).not.toHaveBeenCalled();
  });
});
