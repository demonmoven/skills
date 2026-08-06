import { describe, it, expect } from 'vitest';
import { buildObjectKey, buildObjectMetadata } from './tos.js';
import type { UploadMeta } from './types.js';

const sampleMeta: UploadMeta = {
  toolName: 'claude-code',
  sessionId: 'sess-abc123',
  traceId: 'trace-abc123',
  sessionStartedAt: '2026-04-02T10:00:00Z',
  userId: 'yaoqiyu',
  gitUrl: 'git@code.byted.org:team/repo.git',
  gitBranch: 'feature/test',
  gitCommit: 'abc123def',
  workspaceRoot: '/Users/test/project',
  repoDirty: true,
  deviceId: 'device-uuid-123',
  cliVersion: '0.1.0',
};

describe('buildObjectKey', () => {
  it('builds correct path format', () => {
    const key = buildObjectKey(sampleMeta, '.jsonl.gz');
    // Format: xtrace/user_id/yyyy-mm-dd/tool_name/session_id.jsonl.gz (date from sessionStartedAt)
    expect(key).toBe('xtrace/yaoqiyu/2026-04-02/claude-code/sess-abc123.jsonl.gz');
  });

  it('uses anonymous for missing userId', () => {
    const meta = { ...sampleMeta, userId: undefined };
    const key = buildObjectKey(meta, '.jsonl.gz');
    expect(key.startsWith('xtrace/anonymous/')).toBe(true);
  });

  it('supports .diff.gz suffix', () => {
    const key = buildObjectKey(sampleMeta, '.diff.gz');
    expect(key.endsWith('sess-abc123.diff.gz')).toBe(true);
  });
});

describe('buildObjectMetadata', () => {
  it('includes all provided fields', () => {
    const metadata = buildObjectMetadata(sampleMeta);

    expect(metadata['x-tos-meta-trace-id']).toBe('trace-abc123');
    expect(metadata['x-tos-meta-tool-name']).toBe('claude-code');
    expect(metadata['x-tos-meta-session-id']).toBe('sess-abc123');
    expect(metadata['x-tos-meta-device-id']).toBe('device-uuid-123');
    expect(metadata['x-tos-meta-cli-version']).toBe('0.1.0');
    expect(metadata['x-tos-meta-user-id']).toBe('yaoqiyu');
    expect(metadata['x-tos-meta-git-url']).toBe('git@code.byted.org:team/repo.git');
    expect(metadata['x-tos-meta-git-branch']).toBe('feature/test');
    expect(metadata['x-tos-meta-git-commit']).toBe('abc123def');
    expect(metadata['x-tos-meta-workspace-root']).toBe('/Users/test/project');
    expect(metadata['x-tos-meta-repo-dirty']).toBe('true');
  });

  it('omits optional fields when not provided', () => {
    const meta: UploadMeta = {
      toolName: 'opencode',
      sessionId: 'sess-1',
      traceId: 'trace-1',
      deviceId: 'dev-1',
      cliVersion: '0.1.0',
    };

    const metadata = buildObjectMetadata(meta);

    expect(metadata['x-tos-meta-user-id']).toBeUndefined();
    expect(metadata['x-tos-meta-git-url']).toBeUndefined();
    expect(metadata['x-tos-meta-git-branch']).toBeUndefined();
    expect(metadata['x-tos-meta-git-commit']).toBeUndefined();
  });
});
