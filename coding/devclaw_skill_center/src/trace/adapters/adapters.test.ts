import { describe, it, expect } from 'vitest';
import { join } from 'node:path';
import { claudeCodeAdapter } from './claude-code.js';
import { opencodeAdapter } from './opencode.js';
import { detectAdapter } from './detect.js';

const fixturesDir = join(import.meta.dirname, '../test/fixtures');

describe('claudeCodeAdapter', () => {
  it('detects Claude Code paths', () => {
    expect(claudeCodeAdapter.detect('/Users/test/.claude/projects/abc123/session.jsonl')).toBe(true);
    expect(claudeCodeAdapter.detect('/home/user/.claude/projects/hash/sess.jsonl')).toBe(true);
  });

  it('rejects non-Claude paths', () => {
    expect(claudeCodeAdapter.detect('/tmp/random.jsonl')).toBe(false);
    expect(claudeCodeAdapter.detect('/Users/test/.opencode/session.jsonl')).toBe(false);
  });

  it('extracts metadata from Claude JSONL fixture', async () => {
    const filePath = join(fixturesDir, 'claude-sample.jsonl');
    const meta = await claudeCodeAdapter.extractMeta(filePath);

    expect(meta.toolName).toBe('claude-code');
    expect(meta.sessionId).toBe('sess-claude-abc123');
    expect(meta.startedAt).toBe('2026-04-02T10:00:00Z');
    expect(meta.toolVersion).toBe('1.0.21');
  });
});

describe('opencodeAdapter', () => {
  it('detects OpenCode paths', () => {
    expect(opencodeAdapter.detect('/Users/test/.opencode/sessions/abc.jsonl')).toBe(true);
    expect(opencodeAdapter.detect('/tmp/opencode/worktree/session.jsonl')).toBe(true);
    expect(opencodeAdapter.detect('/tmp/random/file.jsonl')).toBe(false);
  });

  it('extracts metadata from OpenCode JSONL fixture', async () => {
    const filePath = join(fixturesDir, 'opencode-sample.jsonl');
    const meta = await opencodeAdapter.extractMeta(filePath);

    expect(meta.toolName).toBe('opencode');
    expect(meta.sessionId).toBe('sess-opencode-xyz789');
    expect(meta.startedAt).toBe('2026-04-02T11:00:00Z');
    expect(meta.toolVersion).toBe('0.5.0');
  });
});

describe('detectAdapter', () => {
  it('auto-detects Claude Code from path', () => {
    const adapter = detectAdapter('/Users/test/.claude/projects/abc/session.jsonl');
    expect(adapter?.name).toBe('claude-code');
  });

  it('auto-detects OpenCode from path', () => {
    const adapter = detectAdapter('/tmp/opencode/worktree/global/session.jsonl');
    expect(adapter?.name).toBe('opencode');
  });

  it('returns null for unknown path', () => {
    const adapter = detectAdapter('/tmp/random-file.jsonl');
    expect(adapter).toBeNull();
  });

  it('respects explicit --source override', () => {
    const adapter = detectAdapter('/tmp/random-file.jsonl', 'opencode');
    expect(adapter?.name).toBe('opencode');
  });

  it('returns null for unknown source name', () => {
    const adapter = detectAdapter('/tmp/file.jsonl', 'unknown-tool');
    expect(adapter).toBeNull();
  });
});
