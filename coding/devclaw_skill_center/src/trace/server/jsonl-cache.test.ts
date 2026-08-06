import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtemp, rm } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { buildCachePath, getCachedSessionTree, putCachedSessionTree } from './jsonl-cache.js';
import type { SessionTree } from '../analyzer/types.js';

describe('buildCachePath', () => {
  it('replaces slashes with underscores', () => {
    const path = buildCachePath('xtrace/user/2026-04-09/claude-code/sess.jsonl.gz', '/tmp/cache');
    expect(path).toBe(join('/tmp/cache', 'xtrace_user_2026-04-09_claude-code_sess.jsonl'));
  });

  it('strips .gz suffix', () => {
    const path = buildCachePath('a/b/c.jsonl.gz', '/tmp/cache');
    expect(path).not.toMatch(/\.gz$/);
    expect(path).toMatch(/\.jsonl$/);
  });

  it('handles keys without .gz suffix gracefully', () => {
    const path = buildCachePath('a/b/c.jsonl', '/tmp/cache');
    expect(path).toBe(join('/tmp/cache', 'a_b_c.jsonl'));
  });
});

describe('getCachedSessionTree / putCachedSessionTree', () => {
  let tempDir: string;

  beforeEach(async () => {
    tempDir = await mkdtemp(join(tmpdir(), 'xtrace-tree-cache-test-'));
  });

  afterEach(async () => {
    await rm(tempDir, { recursive: true, force: true });
  });

  it('returns null when cache does not exist', async () => {
    const result = await getCachedSessionTree('xtrace/user/2026-04-11/tool/sess.session.tar.gz', tempDir);
    expect(result).toBeNull();
  });

  it('round-trips SessionTree without subagents', async () => {
    const objectKey = 'xtrace/user/2026-04-11/claude-code/sess-simple.session.tar.gz';
    const tree: SessionTree = { mainJsonlContent: '{"type":"message"}\n', subagents: [] };
    const etag = '"etag-tree-1"';

    await putCachedSessionTree(objectKey, tree, etag, tempDir);
    const cached = await getCachedSessionTree(objectKey, tempDir);

    expect(cached).not.toBeNull();
    expect(cached!.tree.mainJsonlContent).toBe(tree.mainJsonlContent);
    expect(cached!.tree.subagents).toEqual([]);
    expect(cached!.etag).toBe(etag);
  });

  it('round-trips SessionTree with subagents', async () => {
    const objectKey = 'xtrace/user/2026-04-11/claude-code/sess-sub.session.tar.gz';
    const tree: SessionTree = {
      mainJsonlContent: '{"type":"message","role":"user"}\n',
      subagents: [
        {
          agentId: 'agent-1',
          jsonlContent: '{"type":"message","role":"assistant"}\n',
          meta: { agentType: 'Explore', description: 'Search codebase' },
        },
      ],
    };

    await putCachedSessionTree(objectKey, tree, '"etag-sub"', tempDir);
    const cached = await getCachedSessionTree(objectKey, tempDir);

    expect(cached).not.toBeNull();
    expect(cached!.tree.subagents).toHaveLength(1);
    expect(cached!.tree.subagents[0]!.agentId).toBe('agent-1');
    expect(cached!.tree.subagents[0]!.meta.agentType).toBe('Explore');
  });
});
