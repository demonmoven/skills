import { describe, it, expect, vi, beforeEach } from 'vitest';
import { join } from 'node:path';

const fixturesDir = join(import.meta.dirname, '../test/fixtures');

// Mock all external dependencies
vi.mock('./auth.js', () => ({
  ensureValidToken: vi.fn().mockResolvedValue({
    accessToken: 'mock-access-token',
    refreshToken: 'mock-refresh-token',
    expiresAt: Math.floor(Date.now() / 1000) + 3600,
  }),
}));

vi.mock('../enrichers/git.js', () => ({
  collectGitContext: vi.fn().mockResolvedValue({
    repoUrl: 'git@code.byted.org:team/repo.git',
    branch: 'main',
    commitHead: 'abc123def456',
    workspaceRoot: '/Users/test/project',
    repoDirty: false,
    diffFromStart: 'diff --git a/file.ts\n+new line',
  }),
  collectMultiRepoDiffs: vi.fn().mockResolvedValue(new Map()),
}));

vi.mock('../enrichers/user.js', () => ({
  getUserInfo: vi.fn().mockResolvedValue({
    userId: 'testuser',
    email: 'testuser@bytedance.com',
  }),
}));

vi.mock('../enrichers/device.js', () => ({
  getClientId: vi.fn().mockResolvedValue('device-mock-uuid'),
}));

vi.mock('../config/loader.js', () => ({
  loadConfig: vi.fn().mockResolvedValue({
    tos: { bucket: 'test-bucket', region: 'cn-beijing' },
    privacy: { collect_content: false, allowed_dirs: [] },
  }),
}));

vi.mock('../session/state.js', () => ({
  cleanupSessionState: vi.fn().mockResolvedValue(undefined),
  loadSessionState: vi.fn().mockResolvedValue(null),
}));

// Mock TOS upload to capture calls
const mockUploadToTOS = vi.fn().mockResolvedValue({
  objectKey: 'mock-key',
  bucket: 'test-bucket',
  etag: '"mock-etag"',
});

vi.mock('../upload/tos.js', async () => {
  const actual = await vi.importActual('../upload/tos.js');
  return {
    ...actual,
    uploadToTOS: mockUploadToTOS,
  };
});

const { runForward } = await import('./forward.js');
const { cleanupSessionState, loadSessionState } = await import('../session/state.js');
const { collectMultiRepoDiffs } = await import('../enrichers/git.js');

const mockCleanupSessionState = vi.mocked(cleanupSessionState);
const mockLoadSessionState = vi.mocked(loadSessionState);
const mockCollectMultiRepoDiffs = vi.mocked(collectMultiRepoDiffs);

describe('forward command integration', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Re-setup default mock returns after clearAllMocks
    mockUploadToTOS.mockResolvedValue({
      objectKey: 'mock-key',
      bucket: 'test-bucket',
      etag: '"mock-etag"',
    });

  });

  it('processes Claude Code fixture end-to-end', async () => {
    const filePath = join(fixturesDir, 'claude-sample.jsonl');
    const result = await runForward(filePath, { source: 'claude-code', cwd: '/tmp' });

    // Should upload JSONL only; per-forward diffs are stored in multi-repo diffs.
    expect(mockUploadToTOS).toHaveBeenCalledTimes(1);
    expect(result.jsonl).toBeDefined();
    expect(result.diff).toBeUndefined();

    // Check first call (JSONL upload) has correct object key pattern: user/date/tool/session
    const firstCall = mockUploadToTOS.mock.calls[0]!;
    const objectKey = firstCall[1] as string;
    expect(objectKey).toMatch(/^xtrace\/testuser\/\d{4}-\d{2}-\d{2}\/claude-code\/sess-claude-abc123\.jsonl\.gz$/);
  });

  it('processes OpenCode fixture end-to-end', async () => {
    const filePath = join(fixturesDir, 'opencode-sample.jsonl');
    const result = await runForward(filePath, { source: 'opencode', cwd: '/tmp' });

    expect(mockUploadToTOS).toHaveBeenCalledTimes(1);
    expect(result.jsonl).toBeDefined();

    const firstCall = mockUploadToTOS.mock.calls[0]!;
    const objectKey = firstCall[1] as string;
    expect(objectKey).toMatch(/^xtrace\/testuser\/\d{4}-\d{2}-\d{2}\/opencode\/sess-opencode-xyz789\.jsonl\.gz$/);
  });

  it('throws FileNotFoundError for missing file', async () => {
    await expect(
      runForward('/nonexistent/file.jsonl', { cwd: '/tmp' }),
    ).rejects.toThrow('File not found');
  });

  it('throws AdapterDetectionError for unrecognizable file', async () => {
    // Create a temp file that won't match any adapter
    const { writeFile } = await import('node:fs/promises');
    const { tmpdir } = await import('node:os');
    const tempFile = join(tmpdir(), `trace-test-unknown-${Date.now()}.jsonl`);
    await writeFile(tempFile, '{"random": "data"}\n');

    try {
      await expect(
        runForward(tempFile, { cwd: '/tmp' }),
      ).rejects.toThrow('Cannot detect tool type');
    } finally {
      const { unlink } = await import('node:fs/promises');
      await unlink(tempFile).catch(() => {});
    }
  });

  it('uploads metadata with correct fields', async () => {
    const filePath = join(fixturesDir, 'claude-sample.jsonl');
    await runForward(filePath, { source: 'claude-code', cwd: '/tmp' });

    const firstCall = mockUploadToTOS.mock.calls[0]!;
    const meta = firstCall[2] as Record<string, unknown>;

    expect(meta['toolName']).toBe('claude-code');
    expect(meta['sessionId']).toBe('sess-claude-abc123');
    expect(meta['userId']).toBe('testuser');
    expect(meta['deviceId']).toBe('device-mock-uuid');
    expect(meta['gitUrl']).toBe('git@code.byted.org:team/repo.git');
    expect(meta['gitBranch']).toBe('main');
    expect(meta['gitCommit']).toBe('abc123def456');
    expect(meta['cliVersion']).toBe('0.1.0');
  });

  it('uploads fixed manifest and timestamped multi-repo diffs when session has repoSnapshots', async () => {
    mockLoadSessionState.mockResolvedValue({
      sessionId: 'sess-claude-abc123',
      startCommit: 'start-commit-abc',
      repoSnapshots: [
        { relativePath: '.', repoUrl: 'git@code.byted.org:team/main.git', branch: 'main', startCommit: 'start-commit-abc', dirty: false },
        { relativePath: 'repos/sub-a', repoUrl: 'git@code.byted.org:team/sub-a.git', branch: 'dev', startCommit: 'sub-a-commit', dirty: true },
      ],
      cwd: '/tmp',
      startedAt: '2026-04-28T10:00:00Z',
    });
    mockCollectMultiRepoDiffs.mockResolvedValue(new Map([
      ['repos/sub-a', 'diff --git a/src/foo.ts\n+added'],
    ]));

    const filePath = join(fixturesDir, 'claude-sample.jsonl');
    await runForward(filePath, { source: 'claude-code', cwd: '/tmp' });

    // Should upload: JSONL + fixed manifest + timestamped multi-repo diffs = 3 uploads.
    expect(mockUploadToTOS).toHaveBeenCalledTimes(3);

    const uploadKeys = mockUploadToTOS.mock.calls.map(c => c[1] as string);
    expect(uploadKeys).toContain('xtrace/testuser/2026-04-02/claude-code/sess-claude-abc123.manifest.json.gz');
    expect(uploadKeys.some(k => /sess-claude-abc123\.\d{4}-\d{2}-\d{2}T.*\.diffs\.json\.gz$/.test(k))).toBe(true);
    expect(uploadKeys.every(k => !k.endsWith('.diff.gz'))).toBe(true);

    // Check gitStartCommit is in metadata
    const firstCall = mockUploadToTOS.mock.calls[0]!;
    const meta = firstCall[2] as Record<string, unknown>;
    expect(meta['gitStartCommit']).toBe('start-commit-abc');
    expect(mockCleanupSessionState).not.toHaveBeenCalled();
  });

  it('does not upload snapshot files when session has no repoSnapshots', async () => {
    mockLoadSessionState.mockResolvedValue(null);

    const filePath = join(fixturesDir, 'claude-sample.jsonl');
    await runForward(filePath, { source: 'claude-code', cwd: '/tmp' });

    // Should upload only JSONL (no manifest/diffs)
    expect(mockUploadToTOS).toHaveBeenCalledTimes(1);
    const uploadKeys = mockUploadToTOS.mock.calls.map(c => c[1] as string);
    expect(uploadKeys.every(k => !k.includes('.manifest.json.gz'))).toBe(true);
    expect(uploadKeys.every(k => !k.includes('.diffs.json.gz'))).toBe(true);
    expect(uploadKeys.every(k => !k.endsWith('.diff.gz'))).toBe(true);
  });

  it('keeps session state after forward so later forwards use the same baseline', async () => {
    const filePath = join(fixturesDir, 'claude-sample.jsonl');
    await runForward(filePath, { source: 'claude-code', cwd: '/tmp' });

    expect(mockCleanupSessionState).not.toHaveBeenCalled();
  });

  it('uses the session-start cwd for multi-repo diffs when Stop runs inside a child repo', async () => {
    mockLoadSessionState.mockResolvedValue({
      sessionId: 'sess-claude-abc123',
      startCommit: 'workspace-start',
      repoSnapshots: [
        { relativePath: '.', repoUrl: 'git@code.byted.org:team/main.git', branch: 'main', startCommit: 'workspace-start', dirty: false },
        { relativePath: 'repos/sub-a', repoUrl: 'git@code.byted.org:team/sub-a.git', branch: 'dev', startCommit: 'sub-a-start', dirty: false },
      ],
      cwd: '/workspace',
      startedAt: '2026-04-29T07:19:53Z',
    });
    mockCollectMultiRepoDiffs.mockResolvedValue(new Map([
      ['repos/sub-a', 'diff --git a/file.ts\n+changed'],
    ]));

    const filePath = join(fixturesDir, 'claude-sample.jsonl');
    await runForward(filePath, { source: 'claude-code', cwd: '/workspace/repos/sub-a' });

    expect(mockCollectMultiRepoDiffs).toHaveBeenCalledWith('/workspace', expect.any(Array));
    const uploadKeys = mockUploadToTOS.mock.calls.map(c => c[1] as string);
    expect(uploadKeys.some(k => k.endsWith('.diffs.json.gz'))).toBe(true);
  });
});
