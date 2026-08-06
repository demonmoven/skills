import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdirSync, writeFileSync, rmSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { randomUUID } from 'node:crypto';
import { discoverSessionDir, discoverSubagents, buildSessionTree } from './subagent-discovery.js';

let testDir: string;

beforeEach(() => {
  testDir = join(tmpdir(), `xtrace-test-${randomUUID()}`);
  mkdirSync(testDir, { recursive: true });
});

afterEach(() => {
  rmSync(testDir, { recursive: true, force: true });
});

describe('discoverSessionDir', () => {
  it('returns session directory path when it exists', async () => {
    const sessionId = 'abc-123';
    const mainJsonl = join(testDir, `${sessionId}.jsonl`);
    writeFileSync(mainJsonl, '{}');
    mkdirSync(join(testDir, sessionId));

    const result = await discoverSessionDir(mainJsonl);
    expect(result).toBe(join(testDir, sessionId));
  });

  it('returns null when session directory does not exist', async () => {
    const mainJsonl = join(testDir, 'no-dir.jsonl');
    writeFileSync(mainJsonl, '{}');

    const result = await discoverSessionDir(mainJsonl);
    expect(result).toBeNull();
  });

  it('returns null when path is a file not a directory', async () => {
    const sessionId = 'file-not-dir';
    const mainJsonl = join(testDir, `${sessionId}.jsonl`);
    writeFileSync(mainJsonl, '{}');
    // Create a file with the session ID name (not a directory)
    writeFileSync(join(testDir, sessionId), 'not a dir');

    const result = await discoverSessionDir(mainJsonl);
    expect(result).toBeNull();
  });
});

describe('discoverSubagents', () => {
  it('discovers subagent JSONL files with meta', async () => {
    const sessionDir = join(testDir, 'session-1');
    const subagentsDir = join(sessionDir, 'subagents');
    mkdirSync(subagentsDir, { recursive: true });

    writeFileSync(join(subagentsDir, 'agent-aaa111.jsonl'), '{"type":"user"}');
    writeFileSync(join(subagentsDir, 'agent-aaa111.meta.json'), JSON.stringify({
      agentType: 'Explore',
      description: 'Search for files',
    }));
    writeFileSync(join(subagentsDir, 'agent-bbb222.jsonl'), '{"type":"user"}');
    // No meta.json for bbb222

    const results = await discoverSubagents(sessionDir);
    expect(results).toHaveLength(2);

    const aaa = results.find(r => r.agentId === 'aaa111');
    expect(aaa).toBeDefined();
    expect(aaa!.meta.agentType).toBe('Explore');
    expect(aaa!.meta.description).toBe('Search for files');
    expect(aaa!.metaPath).toBeTruthy();

    const bbb = results.find(r => r.agentId === 'bbb222');
    expect(bbb).toBeDefined();
    expect(bbb!.meta.agentType).toBe('unknown');
    expect(bbb!.metaPath).toBeNull();
  });

  it('returns empty array when subagents directory does not exist', async () => {
    const sessionDir = join(testDir, 'no-subagents');
    mkdirSync(sessionDir, { recursive: true });

    const results = await discoverSubagents(sessionDir);
    expect(results).toHaveLength(0);
  });

  it('ignores non-agent files in subagents directory', async () => {
    const subagentsDir = join(testDir, 'session-2', 'subagents');
    mkdirSync(subagentsDir, { recursive: true });

    writeFileSync(join(subagentsDir, 'agent-aaa.jsonl'), '{}');
    writeFileSync(join(subagentsDir, 'other-file.txt'), 'not an agent');
    writeFileSync(join(subagentsDir, 'random.jsonl'), '{}'); // doesn't start with agent-

    const results = await discoverSubagents(join(testDir, 'session-2'));
    expect(results).toHaveLength(1);
    expect(results[0]!.agentId).toBe('aaa');
  });

  it('handles malformed meta.json gracefully', async () => {
    const subagentsDir = join(testDir, 'session-3', 'subagents');
    mkdirSync(subagentsDir, { recursive: true });

    writeFileSync(join(subagentsDir, 'agent-ccc.jsonl'), '{}');
    writeFileSync(join(subagentsDir, 'agent-ccc.meta.json'), 'not json{{{');

    const results = await discoverSubagents(join(testDir, 'session-3'));
    expect(results).toHaveLength(1);
    expect(results[0]!.meta.agentType).toBe('unknown');
  });
});

describe('buildSessionTree', () => {
  it('builds a full session tree with subagents', async () => {
    const sessionId = 'full-session';
    const mainJsonl = join(testDir, `${sessionId}.jsonl`);
    const mainContent = '{"type":"user","sessionId":"full-session","message":{"role":"user","content":"hello"}}';
    writeFileSync(mainJsonl, mainContent);

    const subagentsDir = join(testDir, sessionId, 'subagents');
    mkdirSync(subagentsDir, { recursive: true });

    const subContent = '{"type":"user","agentId":"sub1","slug":"test-slug","message":{"role":"user","content":"sub task"}}';
    writeFileSync(join(subagentsDir, 'agent-sub1.jsonl'), subContent);
    writeFileSync(join(subagentsDir, 'agent-sub1.meta.json'), JSON.stringify({
      agentType: 'general-purpose',
      description: 'Test subagent',
    }));

    const tree = await buildSessionTree(mainJsonl);
    expect(tree.mainJsonlContent).toBe(mainContent);
    expect(tree.subagents).toHaveLength(1);
    expect(tree.subagents[0]!.agentId).toBe('sub1');
    expect(tree.subagents[0]!.jsonlContent).toBe(subContent);
    expect(tree.subagents[0]!.meta.agentType).toBe('general-purpose');
  });

  it('returns empty subagents when no session directory exists', async () => {
    const mainJsonl = join(testDir, 'standalone.jsonl');
    writeFileSync(mainJsonl, '{"type":"user"}');

    const tree = await buildSessionTree(mainJsonl);
    expect(tree.mainJsonlContent).toBe('{"type":"user"}');
    expect(tree.subagents).toHaveLength(0);
  });

  it('skips unreadable subagent files gracefully', async () => {
    const sessionId = 'partial';
    const mainJsonl = join(testDir, `${sessionId}.jsonl`);
    writeFileSync(mainJsonl, '{}');

    const subagentsDir = join(testDir, sessionId, 'subagents');
    mkdirSync(subagentsDir, { recursive: true });

    writeFileSync(join(subagentsDir, 'agent-good.jsonl'), '{"type":"user"}');
    writeFileSync(join(subagentsDir, 'agent-good.meta.json'), '{"agentType":"Explore","description":"ok"}');
    // Create a directory with agent- name to simulate unreadable file
    mkdirSync(join(subagentsDir, 'agent-bad.jsonl'));

    const tree = await buildSessionTree(mainJsonl);
    // Should skip the bad one and keep the good one
    expect(tree.subagents).toHaveLength(1);
    expect(tree.subagents[0]!.agentId).toBe('good');
  });
});
