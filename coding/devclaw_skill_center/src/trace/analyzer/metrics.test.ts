import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { analyzeSession, analyzeSessionTree } from './index.js';
import { extractSubagentStats } from './metrics.js';
import { normalizeSessionTree } from './normalizer.js';
import type { SessionTree } from './types.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const fixturesDir = join(__dirname, '..', 'test', 'fixtures');

describe('analyzeSession - Claude Code', () => {
  const content = readFileSync(join(fixturesDir, 'claude-rich-sample.jsonl'), 'utf-8');

  it('produces valid analysis with correct counts', () => {
    const result = analyzeSession(content, 'sess-rich-001', 'claude-code');

    expect(result.sessionId).toBe('sess-rich-001');
    expect(result.toolName).toBe('claude-code');
    // 3 of the 4 original user messages are pure tool_result carriers
    // that get merged back onto assistant messages by mergeToolResults.
    // Only 1 real user message with text content remains.
    expect(result.userMsgCount).toBe(1);
    expect(result.assistantMsgCount).toBe(4);
  });

  it('calculates token totals', () => {
    const result = analyzeSession(content, 'sess-rich-001', 'claude-code');

    expect(result.hasUsageData).toBe(true);
    expect(result.totalInput).toBe(1000 + 1500 + 2000 + 2500);
    expect(result.totalOutput).toBe(500 + 800 + 300 + 200);
    expect(result.totalCacheWrite).toBe(200);
    expect(result.totalCacheRead).toBe(800 + 1200 + 1500 + 2000);
  });

  it('calculates cost correctly', () => {
    const result = analyzeSession(content, 'sess-rich-001', 'claude-code');

    // Sonnet pricing: input=$3/MTok, output=$15/MTok, cache_write=$3.75/MTok, cache_read=$0.30/MTok
    // Turn 1: (1000*3 + 500*15 + 200*3.75 + 800*0.30) / 1_000_000 = (3000+7500+750+240)/1M = 0.01149
    expect(result.totalCost).toBeGreaterThan(0);
    expect(result.perTurnData).toHaveLength(4);
    expect(result.cumulativeCosts).toHaveLength(4);
  });

  it('counts tool calls', () => {
    const result = analyzeSession(content, 'sess-rich-001', 'claude-code');

    expect(result.totalToolCalls).toBe(3); // Read, Edit, Bash
    expect(result.toolCounter['Read']).toBe(1);
    expect(result.toolCounter['Edit']).toBe(1);
    expect(result.toolCounter['Bash']).toBe(1);
  });

  it('detects thinking blocks', () => {
    const result = analyzeSession(content, 'sess-rich-001', 'claude-code');
    expect(result.thinkingBlocks).toBe(1);
  });

  it('calculates duration', () => {
    const result = analyzeSession(content, 'sess-rich-001', 'claude-code');
    expect(result.duration).toBe(40); // 10:00:00 to 10:00:40
    expect(result.durationStr).toBe('40s');
  });

  it('computes efficiency score', () => {
    const result = analyzeSession(content, 'sess-rich-001', 'claude-code');
    expect(result.efficiencyScore).toBeGreaterThan(0);
    expect(result.efficiencyScore).toBeLessThanOrEqual(100);
  });

  it('computes latencies', () => {
    const result = analyzeSession(content, 'sess-rich-001', 'claude-code');
    expect(result.latencies.length).toBeGreaterThan(0);
    // First latency: user at :00, assistant at :05 → 5s
    expect(result.latencies[0]).toBe(5);
  });

  it('returns conversation with all messages', () => {
    const result = analyzeSession(content, 'sess-rich-001', 'claude-code');
    expect(result.conversation.length).toBe(result.userMsgCount + result.assistantMsgCount);
  });
});

describe('analyzeSession - OpenCode', () => {
  const content = readFileSync(join(fixturesDir, 'opencode-rich-sample.jsonl'), 'utf-8');

  it('produces valid analysis', () => {
    const result = analyzeSession(content, 'sess-oc-001', 'opencode');

    expect(result.sessionId).toBe('sess-oc-001');
    expect(result.toolName).toBe('opencode');
    expect(result.userMsgCount).toBeGreaterThan(0);
    expect(result.assistantMsgCount).toBeGreaterThan(0);
  });

  it('marks hasUsageData as false when no usage info', () => {
    const result = analyzeSession(content, 'sess-oc-001', 'opencode');
    expect(result.hasUsageData).toBe(false);
    expect(result.totalCost).toBe(0);
  });

  it('counts tool calls from tool_use lines', () => {
    const result = analyzeSession(content, 'sess-oc-001', 'opencode');
    expect(result.totalToolCalls).toBeGreaterThan(0);
    expect(result.toolCounter['read']).toBeGreaterThanOrEqual(1);
  });

  it('detects tool errors', () => {
    const result = analyzeSession(content, 'sess-oc-001', 'opencode');
    expect(result.toolErrorsTotal).toBeGreaterThan(0);
    expect(result.errorRate).toBeGreaterThan(0);
  });

  it('calculates duration', () => {
    const result = analyzeSession(content, 'sess-oc-001', 'opencode');
    expect(result.duration).toBeGreaterThan(0);
  });
});

describe('analyzeSession - auto detection', () => {
  it('auto-detects Claude Code format', () => {
    const content = readFileSync(join(fixturesDir, 'claude-rich-sample.jsonl'), 'utf-8');
    const result = analyzeSession(content, 'test-session');
    expect(result.toolName).toBe('claude-code');
  });

  it('auto-detects OpenCode format', () => {
    const content = readFileSync(join(fixturesDir, 'opencode-rich-sample.jsonl'), 'utf-8');
    const result = analyzeSession(content, 'test-session');
    expect(result.toolName).toBe('opencode');
  });
});

describe('extractSubagentStats', () => {
  const mainContent = readFileSync(join(fixturesDir, 'claude-subagent-sample.jsonl'), 'utf-8');
  const subContent = readFileSync(join(fixturesDir, 'subagent-explore.jsonl'), 'utf-8');

  function makeTree(): SessionTree {
    return {
      mainJsonlContent: mainContent,
      subagents: [{
        agentId: 'explore-1',
        jsonlContent: subContent,
        meta: { agentType: 'Explore', description: 'Find config files' },
      }],
    };
  }

  it('computes basic subagent stats', () => {
    const tree = makeTree();
    const { subagentMessages } = normalizeSessionTree(tree, 'claude-code');
    const stats = extractSubagentStats(subagentMessages, tree.subagents);

    expect(stats.totalSubagents).toBe(1);
    expect(stats.subagents).toHaveLength(1);
    expect(stats.subagents[0]!.agentId).toBe('explore-1');
    expect(stats.subagents[0]!.agentType).toBe('Explore');
    expect(stats.subagents[0]!.description).toBe('Find config files');
    expect(stats.subagents[0]!.messageCount).toBeGreaterThan(0);
  });

  it('extracts slug from first JSONL line', () => {
    const tree = makeTree();
    const { subagentMessages } = normalizeSessionTree(tree, 'claude-code');
    const stats = extractSubagentStats(subagentMessages, tree.subagents);

    // subagent-explore.jsonl has slug: "curious-fox" in first line
    expect(stats.subagents[0]!.slug).toBe('curious-fox');
  });

  it('tracks token and cost totals', () => {
    const tree = makeTree();
    const { subagentMessages } = normalizeSessionTree(tree, 'claude-code');
    const stats = extractSubagentStats(subagentMessages, tree.subagents);

    // The subagent fixture has 3 assistant messages with usage data
    expect(stats.totalSubagentTokens).toBeGreaterThan(0);
    expect(stats.totalSubagentCost).toBeGreaterThan(0);
    expect(stats.subagents[0]!.totalTokens).toBe(stats.totalSubagentTokens);
    expect(stats.subagents[0]!.totalCost).toBe(stats.totalSubagentCost);
  });

  it('counts tool calls across subagents', () => {
    const tree = makeTree();
    const { subagentMessages } = normalizeSessionTree(tree, 'claude-code');
    const stats = extractSubagentStats(subagentMessages, tree.subagents);

    // subagent-explore.jsonl has Glob and Read tool calls
    expect(stats.subagentToolCounter['Glob']).toBe(1);
    expect(stats.subagentToolCounter['Read']).toBe(1);
  });

  it('computes agent type distribution', () => {
    const tree = makeTree();
    const { subagentMessages } = normalizeSessionTree(tree, 'claude-code');
    const stats = extractSubagentStats(subagentMessages, tree.subagents);

    expect(stats.agentTypeDistribution['Explore']).toBe(1);
  });

  it('handles empty subagent map', () => {
    const stats = extractSubagentStats(new Map(), []);
    expect(stats.totalSubagents).toBe(0);
    expect(stats.totalSubagentTokens).toBe(0);
    expect(stats.totalSubagentCost).toBe(0);
    expect(stats.subagentErrorRate).toBe(0);
  });
});

describe('analyzeSessionTree', () => {
  const mainContent = readFileSync(join(fixturesDir, 'claude-subagent-sample.jsonl'), 'utf-8');
  const subContent = readFileSync(join(fixturesDir, 'subagent-explore.jsonl'), 'utf-8');

  it('produces analysis with subagentStats', () => {
    const tree: SessionTree = {
      mainJsonlContent: mainContent,
      subagents: [{
        agentId: 'explore-1',
        jsonlContent: subContent,
        meta: { agentType: 'Explore', description: 'Find config files' },
      }],
    };

    const analysis = analyzeSessionTree(tree, 'sess-sub-001', 'claude-code');

    // Main session metrics should be present
    expect(analysis.sessionId).toBe('sess-sub-001');
    expect(analysis.toolName).toBe('claude-code');
    expect(analysis.assistantMsgCount).toBeGreaterThan(0);
    expect(analysis.totalToolCalls).toBeGreaterThan(0);

    // Subagent stats should be attached
    expect(analysis.subagentStats).toBeDefined();
    expect(analysis.subagentStats!.totalSubagents).toBe(1);
    expect(analysis.subagentStats!.subagents[0]!.agentType).toBe('Explore');
  });

  it('returns no subagentStats when tree has no subagents', () => {
    const tree: SessionTree = {
      mainJsonlContent: mainContent,
      subagents: [],
    };

    const analysis = analyzeSessionTree(tree, 'sess-sub-001', 'claude-code');
    expect(analysis.subagentStats).toBeUndefined();
  });

  it('produces same main metrics as analyzeSession', () => {
    const tree: SessionTree = {
      mainJsonlContent: mainContent,
      subagents: [],
    };

    const fromTree = analyzeSessionTree(tree, 'sess-sub-001', 'claude-code');
    const fromFlat = analyzeSession(mainContent, 'sess-sub-001', 'claude-code');

    expect(fromTree.assistantMsgCount).toBe(fromFlat.assistantMsgCount);
    expect(fromTree.userMsgCount).toBe(fromFlat.userMsgCount);
    expect(fromTree.totalToolCalls).toBe(fromFlat.totalToolCalls);
    expect(fromTree.totalCost).toBe(fromFlat.totalCost);
  });
});
