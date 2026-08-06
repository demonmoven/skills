import { describe, it, expect } from 'vitest';
import { parseObjectKey, buildAnalysisCacheKey, buildEvalCacheKey } from './tos-reader.js';
import type { SessionListItem } from './tos-reader.js';

describe('parseObjectKey', () => {
  it('parses a valid Claude Code session key (.jsonl.gz)', () => {
    const key = 'xtrace/user123/2026-04-02/claude-code/sess-abc123.jsonl.gz';
    const result = parseObjectKey(key);
    expect(result).toEqual({
      sessionId: 'sess-abc123',
      toolName: 'claude-code',
      userId: 'user123',
      date: '2026-04-02',
      objectKey: key,
      format: 'jsonl',
    });
  });

  it('parses a valid OpenCode session key (.jsonl.gz)', () => {
    const key = 'xtrace/user456/2026-04-03/opencode/sess-xyz789.jsonl.gz';
    const result = parseObjectKey(key);
    expect(result).toEqual({
      sessionId: 'sess-xyz789',
      toolName: 'opencode',
      userId: 'user456',
      date: '2026-04-03',
      objectKey: key,
      format: 'jsonl',
    });
  });

  it('parses a session bundle key (.session.tar.gz)', () => {
    const key = 'xtrace/user123/2026-04-11/claude-code/sess-bundle1.session.tar.gz';
    const result = parseObjectKey(key);
    expect(result).toEqual({
      sessionId: 'sess-bundle1',
      toolName: 'claude-code',
      userId: 'user123',
      date: '2026-04-11',
      objectKey: key,
      format: 'bundle',
    });
  });

  it('returns null for non-xtrace prefix', () => {
    expect(parseObjectKey('other/user/2026-04-02/claude-code/sess.jsonl.gz')).toBeNull();
  });

  it('returns null for wrong extension', () => {
    expect(parseObjectKey('xtrace/user/2026-04-02/claude-code/sess.json')).toBeNull();
  });

  it('returns null for diff files', () => {
    expect(parseObjectKey('xtrace/user/2026-04-02/claude-code/sess.2026-04-02.diff.gz')).toBeNull();
  });

  it('returns null for missing segments', () => {
    expect(parseObjectKey('xtrace/user/claude-code/sess.jsonl.gz')).toBeNull();
  });

  it('returns null for invalid date format', () => {
    expect(parseObjectKey('xtrace/user/20260402/claude-code/sess.jsonl.gz')).toBeNull();
  });

  it('handles session IDs with hyphens and special chars', () => {
    const key = 'xtrace/john.doe/2026-01-15/claude-code/sess-a1b2-c3d4-e5f6.jsonl.gz';
    const result = parseObjectKey(key);
    expect(result).not.toBeNull();
    expect(result!.sessionId).toBe('sess-a1b2-c3d4-e5f6');
    expect(result!.userId).toBe('john.doe');
    expect(result!.format).toBe('jsonl');
  });
});

describe('session deduplication', () => {
  /** Replicate the dedup logic from listSessionObjects for unit testing. */
  function dedup(keys: string[]): SessionListItem[] {
    const parsed = keys
      .map(k => parseObjectKey(k))
      .filter((s): s is SessionListItem => s !== null);
    const deduped = new Map<string, SessionListItem>();
    for (const s of parsed) {
      const existing = deduped.get(s.sessionId);
      if (!existing || s.format === 'bundle') {
        deduped.set(s.sessionId, s);
      }
    }
    return Array.from(deduped.values());
  }

  it('prefers bundle over jsonl for the same sessionId', () => {
    const keys = [
      'xtrace/user/2026-04-11/claude-code/sess-1.jsonl.gz',
      'xtrace/user/2026-04-11/claude-code/sess-1.session.tar.gz',
    ];
    const result = dedup(keys);
    expect(result).toHaveLength(1);
    expect(result[0]!.format).toBe('bundle');
    expect(result[0]!.objectKey).toMatch(/\.session\.tar\.gz$/);
  });

  it('keeps both when sessionIds differ', () => {
    const keys = [
      'xtrace/user/2026-04-11/claude-code/sess-1.jsonl.gz',
      'xtrace/user/2026-04-11/claude-code/sess-2.session.tar.gz',
    ];
    const result = dedup(keys);
    expect(result).toHaveLength(2);
  });

  it('handles bundle-first ordering', () => {
    const keys = [
      'xtrace/user/2026-04-11/claude-code/sess-1.session.tar.gz',
      'xtrace/user/2026-04-11/claude-code/sess-1.jsonl.gz',
    ];
    const result = dedup(keys);
    expect(result).toHaveLength(1);
    expect(result[0]!.format).toBe('bundle');
  });

  it('keeps jsonl when no bundle exists', () => {
    const keys = [
      'xtrace/user/2026-04-11/claude-code/sess-1.jsonl.gz',
    ];
    const result = dedup(keys);
    expect(result).toHaveLength(1);
    expect(result[0]!.format).toBe('jsonl');
  });
});

describe('buildAnalysisCacheKey', () => {
  it('builds correct analysis cache key', () => {
    const item = {
      sessionId: 'sess-abc123',
      toolName: 'claude-code',
      userId: 'user123',
      date: '2026-04-02',
      objectKey: 'xtrace/user123/2026-04-02/claude-code/sess-abc123.jsonl.gz',
      format: 'jsonl' as const,
    };
    expect(buildAnalysisCacheKey(item)).toBe(
      'xtrace-analysis/user123/2026-04-02/claude-code/sess-abc123.analysis.json'
    );
  });
});

describe('buildEvalCacheKey', () => {
  it('builds correct eval cache key', () => {
    const item = {
      sessionId: 'sess-abc123',
      toolName: 'claude-code',
      userId: 'user123',
      date: '2026-04-02',
      objectKey: 'xtrace/user123/2026-04-02/claude-code/sess-abc123.jsonl.gz',
      format: 'jsonl' as const,
    };
    expect(buildEvalCacheKey(item)).toBe(
      'xtrace-eval/user123/2026-04-02/claude-code/sess-abc123.eval.json'
    );
  });

  it('uses xtrace-eval prefix, not xtrace-analysis', () => {
    const item = {
      sessionId: 'sess-xyz',
      toolName: 'opencode',
      userId: 'user456',
      date: '2026-04-03',
      objectKey: 'xtrace/user456/2026-04-03/opencode/sess-xyz.jsonl.gz',
      format: 'jsonl' as const,
    };
    const key = buildEvalCacheKey(item);
    expect(key).toMatch(/^xtrace-eval\//);
    expect(key).toMatch(/\.eval\.json$/);
  });
});
