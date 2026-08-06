import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('node:child_process', () => ({
  execFile: vi.fn(),
}));

vi.mock('node:fs/promises', () => ({
  readdir: vi.fn(),
  access: vi.fn(),
}));

import { execFile } from 'node:child_process';
import { readdir, access } from 'node:fs/promises';
import { discoverRepos } from './repo-discovery.js';

const mockExecFile = vi.mocked(execFile);
const mockReaddir = vi.mocked(readdir);
const mockAccess = vi.mocked(access);

function setupGitCliMock(gitRepoDirs: Set<string>): void {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  mockExecFile.mockImplementation((_cmd: string, _args: any, opts: any, _cb?: any) => {
    const callback = typeof opts === 'function' ? opts : _cb;
    const cwd = (typeof opts === 'object' ? opts.cwd : undefined) as string | undefined;
    if (cwd && gitRepoDirs.has(cwd)) {
      callback(null, { stdout: cwd + '\n', stderr: '' });
    } else {
      callback(new Error('not a git repo'), { stdout: '', stderr: '' });
    }
    return undefined as never;
  });
}

function setupFsMock(
  dirContents: Record<string, Array<{ name: string; isDirectory: boolean }>>,
  gitDirs: Set<string>,
): void {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  mockReaddir.mockImplementation(async (dir: any, _opts?: any) => {
    const entries = dirContents[dir as string] ?? [];
    return entries.map(e => ({
      name: e.name,
      isDirectory: () => e.isDirectory,
      isFile: () => !e.isDirectory,
      isBlockDevice: () => false,
      isCharacterDevice: () => false,
      isFIFO: () => false,
      isSocket: () => false,
      isSymbolicLink: () => false,
      path: '',
      parentPath: dir as string,
    })) as never;
  });

  mockAccess.mockImplementation(async (path: unknown) => {
    if (gitDirs.has(path as string)) return;
    throw Object.assign(new Error('ENOENT'), { code: 'ENOENT' });
  });
}

describe('discoverRepos', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('discovers cwd as git repo + auto-scans sub-repos', async () => {
    setupGitCliMock(new Set(['/workspace']));
    setupFsMock(
      {
        '/workspace': [
          { name: 'repos', isDirectory: true },
          { name: 'file.txt', isDirectory: false },
        ],
        '/workspace/repos': [
          { name: 'repo-a', isDirectory: true },
          { name: 'repo-b', isDirectory: true },
        ],
      },
      new Set([
        '/workspace/.git',
        '/workspace/repos/repo-a/.git',
        '/workspace/repos/repo-b/.git',
      ]),
    );

    const repos = await discoverRepos('/workspace');

    expect(repos).toHaveLength(3);
    expect(repos[0]).toEqual({ relativePath: '.', absolutePath: '/workspace' });
    expect(repos).toContainEqual({ relativePath: 'repos/repo-a', absolutePath: '/workspace/repos/repo-a' });
    expect(repos).toContainEqual({ relativePath: 'repos/repo-b', absolutePath: '/workspace/repos/repo-b' });
  });

  it('returns only sub-repos when cwd is not a git repo', async () => {
    setupGitCliMock(new Set());
    setupFsMock(
      {
        '/plain-dir': [
          { name: 'repo-a', isDirectory: true },
        ],
      },
      new Set(['/plain-dir/repo-a/.git']),
    );

    const repos = await discoverRepos('/plain-dir');

    expect(repos).toHaveLength(1);
    expect(repos[0]).toEqual({ relativePath: 'repo-a', absolutePath: '/plain-dir/repo-a' });
  });

  it('uses configured repos instead of scanning', async () => {
    setupGitCliMock(new Set(['/workspace']));
    setupFsMock(
      {},
      new Set([
        '/workspace/.git',
        '/workspace/repos/repo-a/.git',
      ]),
    );

    const repos = await discoverRepos('/workspace', ['repos/repo-a', 'repos/missing']);

    expect(repos).toHaveLength(2);
    expect(repos[0]).toEqual({ relativePath: '.', absolutePath: '/workspace' });
    expect(repos[1]).toEqual({ relativePath: 'repos/repo-a', absolutePath: '/workspace/repos/repo-a' });
  });

  it('skips configured paths that are not git repos', async () => {
    setupGitCliMock(new Set(['/workspace']));
    setupFsMock({}, new Set(['/workspace/.git']));

    const repos = await discoverRepos('/workspace', ['not-a-repo']);

    expect(repos).toHaveLength(1);
    expect(repos[0].relativePath).toBe('.');
  });

  it('returns only cwd when no sub-repos found', async () => {
    setupGitCliMock(new Set(['/solo-repo']));
    setupFsMock(
      {
        '/solo-repo': [
          { name: 'src', isDirectory: true },
        ],
        '/solo-repo/src': [],
      },
      new Set(['/solo-repo/.git']),
    );

    const repos = await discoverRepos('/solo-repo');

    expect(repos).toHaveLength(1);
    expect(repos[0]).toEqual({ relativePath: '.', absolutePath: '/solo-repo' });
  });

  it('excludes node_modules and hidden directories during scan', async () => {
    setupGitCliMock(new Set());
    setupFsMock(
      {
        '/workspace': [
          { name: 'node_modules', isDirectory: true },
          { name: '.hidden', isDirectory: true },
          { name: 'actual-repo', isDirectory: true },
        ],
      },
      new Set([
        '/workspace/node_modules/.git',
        '/workspace/.hidden/.git',
        '/workspace/actual-repo/.git',
      ]),
    );

    const repos = await discoverRepos('/workspace');

    expect(repos).toHaveLength(1);
    expect(repos[0].relativePath).toBe('actual-repo');
  });
});
