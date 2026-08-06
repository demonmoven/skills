import { basename, extname } from 'node:path';
import type { AnalyzeResult, ProcessedEntry, Summary, SkillInvocation } from './types.js';

const SKILL_TOOL_NAME = 'Skill';

/**
 * Walk the processed trajectory (including subagent drill-downs) and aggregate.
 */
export function computeSummary(result: AnalyzeResult): Summary {
  const summary: Summary = {
    sessionId: basename(result.sourceFile, extname(result.sourceFile)),
    durationMs: 0,
    totalEntries: 0,
    totalTurns: 0,
    totalToolCalls: 0,
    toolDistribution: {},
    tokenTotals: { input: 0, output: 0, cacheRead: 0, cacheCreation: 0 },
    errorCount: 0,
    subagentCount: result.subagents.size,
    subagentTypes: {},
    skillInvocations: [],
    skillDistribution: {},
  };

  for (const sa of result.subagents.values()) {
    const type = sa.meta.agentType || 'unknown';
    summary.subagentTypes[type] = (summary.subagentTypes[type] ?? 0) + 1;
  }

  let firstTs = '';
  let lastTs = '';
  let model: string | undefined;

  const walk = (entries: ProcessedEntry[]): void => {
    for (const e of entries) {
      summary.totalEntries++;

      if (e.type === 'user-text') summary.totalTurns++;

      if (e.tokens) {
        summary.tokenTotals.input += e.tokens.inputTokens;
        summary.tokenTotals.output += e.tokens.outputTokens;
        summary.tokenTotals.cacheRead += e.tokens.cacheReadTokens;
        summary.tokenTotals.cacheCreation += e.tokens.cacheCreationTokens;
      }

      if (e.timestamp) {
        if (!firstTs || e.timestamp < firstTs) firstTs = e.timestamp;
        if (!lastTs || e.timestamp > lastTs) lastTs = e.timestamp;
      }

      for (const tc of e.toolCalls) {
        summary.totalToolCalls++;
        summary.toolDistribution[tc.name] = (summary.toolDistribution[tc.name] ?? 0) + 1;
        if (tc.result?.isError) summary.errorCount++;

        if (tc.name === SKILL_TOOL_NAME) {
          const input = tc.input as Record<string, unknown> | undefined;
          const skillName =
            input && typeof input['skill'] === 'string'
              ? (input['skill'] as string)
              : 'unknown';
          const args =
            input && typeof input['args'] === 'string'
              ? (input['args'] as string)
              : undefined;
          summary.skillDistribution[skillName] =
            (summary.skillDistribution[skillName] ?? 0) + 1;
          const invocation: SkillInvocation = {
            skillName,
            args,
            timestamp: e.timestamp,
            toolUseId: tc.id,
            depth: e.depth,
            isError: tc.result?.isError === true,
          };
          summary.skillInvocations.push(invocation);
        }

        if (tc.subagent?.processed) walk(tc.subagent.processed);
      }
    }
  };

  walk(result.mainEntries);

  // Try to grab model from the first assistant raw entry we can find.
  // (We don't keep model in ProcessedEntry for simplicity; peek at subagents if main has none.)
  // Leaving model undefined for v1 is fine — the renderer will display "-".

  summary.startedAt = firstTs || undefined;
  summary.endedAt = lastTs || undefined;
  if (firstTs && lastTs) {
    summary.durationMs = new Date(lastTs).getTime() - new Date(firstTs).getTime();
  }
  summary.model = model;

  return summary;
}
