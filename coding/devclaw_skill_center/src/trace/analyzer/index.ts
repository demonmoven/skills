import { parseJsonl } from './parser.js';
import { normalize, normalizeSessionTree, detectFormat } from './normalizer.js';
import { extractMetrics, extractSubagentStats } from './metrics.js';
import type { SessionAnalysis, SessionTree } from './types.js';

export type { SessionAnalysis, NormalizedMessage, PerTurnData, EvalResult, SubagentStats, SubagentMeta, SessionTree } from './types.js';

/**
 * Analyze a JSONL session file content.
 *
 * @internal Prefer {@link analyzeSessionTree} for all external callers.
 * This function is kept for internal use within `analyzeSessionTree` and tests.
 *
 * @param jsonlContent - Raw JSONL text content
 * @param sessionId - Session ID (from TOS object key or file name)
 * @param toolName - Optional tool name to skip format auto-detection
 * @returns Complete session analysis result
 */
export function analyzeSession(
  jsonlContent: string,
  sessionId: string,
  toolName?: string,
): SessionAnalysis {
  const rawMessages = parseJsonl(jsonlContent);
  const format = toolName ?? detectFormat(rawMessages);
  const normalizedMessages = normalize(rawMessages, format);
  return extractMetrics(normalizedMessages, sessionId, format);
}

/**
 * Analyze a full session tree (main session + subagent sessions).
 * Falls back to analyzeSession behavior when no subagents are present.
 *
 * @param tree - SessionTree containing main JSONL and subagent data
 * @param sessionId - Session ID
 * @param toolName - Optional tool name to skip format auto-detection
 * @returns Session analysis with optional subagentStats
 */
export function analyzeSessionTree(
  tree: SessionTree,
  sessionId: string,
  toolName?: string,
): SessionAnalysis {
  const rawMessages = parseJsonl(tree.mainJsonlContent);
  const format = toolName ?? detectFormat(rawMessages);

  // Normalize the full tree
  const { mainMessages, subagentMessages } = normalizeSessionTree(tree, format);

  // Extract main session metrics
  const analysis = extractMetrics(mainMessages, sessionId, format);

  // Extract subagent statistics and conversations if subagents exist
  if (subagentMessages.size > 0) {
    analysis.subagentStats = extractSubagentStats(subagentMessages, tree.subagents);

    // Attach subagent conversations for inline display
    const subagentConversations: SessionAnalysis['subagentConversations'] = {};
    for (const sub of tree.subagents) {
      const msgs = subagentMessages.get(sub.agentId);
      if (msgs && msgs.length > 0) {
        subagentConversations[sub.agentId] = {
          agentId: sub.agentId,
          agentType: sub.meta.agentType,
          description: sub.meta.description,
          messages: msgs,
        };
      }
    }
    analysis.subagentConversations = subagentConversations;
  }

  return analysis;
}
