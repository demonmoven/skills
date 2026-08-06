import type { NormalizedMessage, SessionAnalysis, PerTurnData, SubagentStats, SubagentMeta, SessionTree } from './types.js';
import { parseTimestamp } from './parser.js';

// ─── Pricing ($/MTok) ──────────────────────────────────────────────

const PRICING: Record<string, Record<string, number>> = {
  opus:   { input: 15.0,  output: 75.0,  cache_write: 18.75, cache_read: 1.50 },
  sonnet: { input: 3.0,   output: 15.0,  cache_write: 3.75,  cache_read: 0.30 },
  haiku:  { input: 0.80,  output: 4.0,   cache_write: 1.0,   cache_read: 0.08 },
  // Trae / ByteDance models — pricing is approximate; update when official rates are available
  doubao: { input: 0.80,  output: 2.0,   cache_write: 0,     cache_read: 0 },
  deepseek: { input: 0.27, output: 1.10,  cache_write: 0,     cache_read: 0 },
};

function getPricing(model: string): Record<string, number> {
  const m = model.toLowerCase();
  if (m.includes('opus')) return PRICING['opus']!;
  if (m.includes('haiku')) return PRICING['haiku']!;
  if (m.includes('doubao')) return PRICING['doubao']!;
  if (m.includes('deepseek')) return PRICING['deepseek']!;
  // Sonnet as default for Anthropic; doubao as default for unknown (conservative)
  if (m.includes('sonnet')) return PRICING['sonnet']!;
  return PRICING['sonnet']!; // default
}

function getModelFamily(model: string): string {
  const m = model.toLowerCase();
  if (m.includes('opus')) return 'Opus';
  if (m.includes('sonnet')) return 'Sonnet';
  if (m.includes('haiku')) return 'Haiku';
  if (m.includes('doubao')) return 'Doubao';
  if (m.includes('deepseek')) return 'DeepSeek';
  return model;
}

// ─── Main Analysis ─────────────────────────────────────────────────

export function extractMetrics(
  messages: NormalizedMessage[],
  sessionId: string,
  toolName: string,
): SessionAnalysis {
  let userMsgCount = 0;
  let assistantMsgCount = 0;
  let systemMsgCount = 0;
  let thinkingBlocks = 0;
  let hasUsageData = false;

  const timestamps: Date[] = [];
  const modelsUsed: Record<string, number> = {};
  const toolCounter: Record<string, number> = {};
  const filesEdited: Record<string, number> = {};
  const toolCallSequence: Array<{ name: string; timestamp: string }> = [];

  // Token/cost tracking
  let totalInput = 0;
  let totalOutput = 0;
  let totalCacheWrite = 0;
  let totalCacheRead = 0;
  let totalCost = 0;
  const cumulativeCosts: Array<{ turn: number; cost: number }> = [];
  const perTurnData: PerTurnData[] = [];

  // Latency tracking
  const latencies: number[] = [];
  let lastUserTs: Date | null = null;

  // Tool error tracking: tool_use_id → toolName mapping via sequence
  const toolErrorMap: Record<string, { calls: number; errors: number }> = {};

  let turnIndex = 0;

  for (const msg of messages) {
    // Parse timestamp
    if (msg.timestamp) {
      try {
        timestamps.push(parseTimestamp(msg.timestamp));
      } catch {
        // skip invalid timestamps
      }
    }

    if (msg.role === 'user') {
      userMsgCount++;

      // Track latency (user → next assistant)
      if (msg.timestamp) {
        try { lastUserTs = parseTimestamp(msg.timestamp); } catch { /* skip */ }
      }

      // Track tool results for error rate
      if (msg.toolResults) {
        for (const tr of msg.toolResults) {
          const name = tr.toolName;
          if (!toolErrorMap[name]) toolErrorMap[name] = { calls: 0, errors: 0 };
          // tool_result doesn't count as a "call" — the tool_use does
          if (tr.isError) toolErrorMap[name]!.errors++;
        }
      }
    } else if (msg.role === 'assistant') {
      assistantMsgCount++;

      // Latency calculation
      if (lastUserTs && msg.timestamp) {
        try {
          const ts = parseTimestamp(msg.timestamp);
          const delta = (ts.getTime() - lastUserTs.getTime()) / 1000;
          if (delta > 0 && delta < 600) latencies.push(Math.round(delta * 100) / 100);
        } catch { /* skip */ }
        lastUserTs = null;
      }

      // Model tracking
      const model = msg.model ?? 'unknown';
      modelsUsed[model] = (modelsUsed[model] ?? 0) + 1;

      // Thinking blocks
      if (msg.thinking) thinkingBlocks++;

      // Tool calls
      if (msg.toolCalls) {
        for (const tc of msg.toolCalls) {
          toolCounter[tc.name] = (toolCounter[tc.name] ?? 0) + 1;
          toolCallSequence.push({ name: tc.name, timestamp: msg.timestamp });

          if (!toolErrorMap[tc.name]) toolErrorMap[tc.name] = { calls: 0, errors: 0 };
          toolErrorMap[tc.name]!.calls++;

          // Track file edits
          if (['Edit', 'Write', 'Read', 'Glob', 'Grep', 'edit', 'write', 'read', 'glob', 'grep'].includes(tc.name)) {
            const fp = tc.input;
            if (fp) filesEdited[fp] = (filesEdited[fp] ?? 0) + 1;
          }
        }
      }

      // Token/cost calculation
      if (msg.usage) {
        hasUsageData = true;
        const u = msg.usage;
        const pricing = getPricing(model);
        const cost = (
          u.inputTokens * pricing['input']! +
          u.outputTokens * pricing['output']! +
          u.cacheWriteTokens * pricing['cache_write']! +
          u.cacheReadTokens * pricing['cache_read']!
        ) / 1_000_000;

        totalInput += u.inputTokens;
        totalOutput += u.outputTokens;
        totalCacheWrite += u.cacheWriteTokens;
        totalCacheRead += u.cacheReadTokens;
        totalCost += cost;

        turnIndex++;
        cumulativeCosts.push({ turn: turnIndex, cost: Math.round(totalCost * 10000) / 10000 });
        perTurnData.push({
          turn: turnIndex,
          timestamp: msg.timestamp,
          model: getModelFamily(model),
          inputTokens: u.inputTokens,
          outputTokens: u.outputTokens,
          cacheWrite: u.cacheWriteTokens,
          cacheRead: u.cacheReadTokens,
          cost: Math.round(cost * 10000) / 10000,
        });
      }
    } else if (msg.role === 'system') {
      systemMsgCount++;
    }
  }

  // Derived metrics
  const totalToolCalls = Object.values(toolCounter).reduce((a, b) => a + b, 0);
  const toolErrorsTotal = Object.values(toolErrorMap).reduce((a, b) => a + b.errors, 0);

  const errorRate = totalToolCalls > 0
    ? Math.round((toolErrorsTotal / totalToolCalls) * 10000) / 100
    : 0;

  const retries = detectRetries(toolCallSequence);
  const retryRate = totalToolCalls > 0
    ? Math.round((retries.length / totalToolCalls) * 10000) / 100
    : 0;

  const thinkingRatio = assistantMsgCount > 0
    ? Math.round((thinkingBlocks / assistantMsgCount) * 10000) / 100
    : 0;

  const toolDiversity = totalToolCalls > 0
    ? Math.round((Object.keys(toolCounter).length / totalToolCalls) * 10000) / 100
    : 0;

  const cacheEfficiency = (totalCacheRead + totalInput) > 0
    ? Math.round((totalCacheRead / (totalCacheRead + totalInput)) * 10000) / 100
    : 0;

  // Efficiency score
  const efficiencyScore = Math.max(0, Math.min(100, Math.round(
    (1 - errorRate / 100) * 30 +
    (1 - retryRate / 100) * 25 +
    Math.min(thinkingRatio / 100, 1) * 15 +
    Math.min(toolDiversity / 100 * 5, 1) * 15 +
    15 // base completion score
  )));

  // Duration
  let duration = 0;
  let startTime = '';
  let endTime = '';
  if (timestamps.length > 0) {
    const min = new Date(Math.min(...timestamps.map(t => t.getTime())));
    const max = new Date(Math.max(...timestamps.map(t => t.getTime())));
    duration = (max.getTime() - min.getTime()) / 1000;
    startTime = min.toISOString();
    endTime = max.toISOString();
  }

  // Sort and limit filesEdited
  const sortedFiles = Object.entries(filesEdited)
    .sort(([, a], [, b]) => b - a)
    .slice(0, 20);
  const filesEditedLimited = Object.fromEntries(sortedFiles);

  return {
    sessionId,
    toolName,
    duration,
    durationStr: formatDuration(duration),
    startTime,
    endTime,
    userMsgCount,
    assistantMsgCount,
    systemMsgCount,
    totalToolCalls,
    toolCounter,
    toolErrorMap,
    toolErrorsTotal,
    filesEdited: filesEditedLimited,
    thinkingBlocks,
    retries,
    modelsUsed,
    totalInput,
    totalOutput,
    totalCacheWrite,
    totalCacheRead,
    totalCost: Math.round(totalCost * 10000) / 10000,
    cacheEfficiency,
    hasUsageData,
    cumulativeCosts,
    perTurnData,
    latencies,
    errorRate,
    retryRate,
    thinkingRatio,
    toolDiversity,
    efficiencyScore,
    conversation: messages,
  };
}

// ─── Subagent Stats ───────────────────────────────────────────────

/**
 * Extract aggregated statistics from subagent normalized messages.
 */
export function extractSubagentStats(
  subagentMessages: Map<string, NormalizedMessage[]>,
  subagentInfos: SessionTree['subagents'],
): SubagentStats {
  const subagents: SubagentMeta[] = [];
  let totalSubagentTokens = 0;
  let totalSubagentCost = 0;
  const agentTypeDistribution: Record<string, number> = {};
  const subagentToolCounter: Record<string, number> = {};
  let totalSubagentToolCalls = 0;
  let totalSubagentToolErrors = 0;

  for (const info of subagentInfos) {
    const msgs = subagentMessages.get(info.agentId) ?? [];
    let tokens = 0;
    let cost = 0;
    let slug = info.agentId;

    for (const msg of msgs) {
      // Extract slug from the first message (subagent JSONL entries carry a slug field
      // but it's lost during normalization — we'll look for it in the raw content below)
      if (msg.role === 'assistant' && msg.usage) {
        const u = msg.usage;
        const model = msg.model ?? '';
        const pricing = getPricing(model);
        const msgCost = (
          u.inputTokens * pricing['input']! +
          u.outputTokens * pricing['output']! +
          u.cacheWriteTokens * pricing['cache_write']! +
          u.cacheReadTokens * pricing['cache_read']!
        ) / 1_000_000;

        tokens += u.inputTokens + u.outputTokens;
        cost += msgCost;
      }

      if (msg.toolCalls) {
        for (const tc of msg.toolCalls) {
          subagentToolCounter[tc.name] = (subagentToolCounter[tc.name] ?? 0) + 1;
          totalSubagentToolCalls++;
        }
      }
      if (msg.toolResults) {
        for (const tr of msg.toolResults) {
          if (tr.isError) totalSubagentToolErrors++;
        }
      }
    }

    // Try to extract slug from raw JSONL content (first line often has it)
    try {
      const firstLine = info.jsonlContent.slice(0, info.jsonlContent.indexOf('\n'));
      const raw = JSON.parse(firstLine) as Record<string, unknown>;
      if (typeof raw['slug'] === 'string') slug = raw['slug'];
    } catch { /* use agentId as fallback */ }

    subagents.push({
      agentId: info.agentId,
      slug,
      agentType: info.meta.agentType,
      description: info.meta.description,
      messageCount: msgs.length,
      totalTokens: tokens,
      totalCost: Math.round(cost * 10000) / 10000,
    });

    totalSubagentTokens += tokens;
    totalSubagentCost += cost;
    agentTypeDistribution[info.meta.agentType] =
      (agentTypeDistribution[info.meta.agentType] ?? 0) + 1;
  }

  return {
    totalSubagents: subagents.length,
    subagents,
    totalSubagentTokens,
    totalSubagentCost: Math.round(totalSubagentCost * 10000) / 10000,
    agentTypeDistribution,
    subagentToolCounter,
    subagentErrorRate: totalSubagentToolCalls > 0
      ? Math.round((totalSubagentToolErrors / totalSubagentToolCalls) * 10000) / 100
      : 0,
  };
}

// ─── Helpers ───────────────────────────────────────────────────────

function detectRetries(
  sequence: Array<{ name: string; timestamp: string }>,
): Array<{ tool: string; count: number }> {
  const retries: Array<{ tool: string; count: number }> = [];
  if (sequence.length < 3) return retries;

  let i = 0;
  while (i < sequence.length) {
    const toolName = sequence[i]!.name;
    let j = i + 1;
    while (j < sequence.length && sequence[j]!.name === toolName) {
      j++;
    }
    const count = j - i;
    if (count >= 3) {
      retries.push({ tool: toolName, count });
    }
    i = j;
  }
  return retries;
}

export function formatDuration(seconds: number): string {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  const parts: string[] = [];
  if (h > 0) parts.push(`${h}h`);
  if (m > 0) parts.push(`${m}m`);
  parts.push(`${s}s`);
  return parts.join(' ');
}
