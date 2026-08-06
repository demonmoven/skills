import type {
  AnalyzeResult,
  ContentBlock,
  LogEntry,
  ProcessedEntry,
  ProcessedEntryType,
  Summary,
  SubagentTrace,
  TokenMetrics,
  ToolCall,
  UsageStats,
} from './types.js';

/**
 * The tool name used by Claude Code for its subagent (Task) tool.
 * Note: in the JSONL this appears as `name: "Agent"`, NOT `"Task"` — this tripped
 * me up during research. See research.md §1.5.
 */
const AGENT_TOOL_NAME = 'Agent';

/**
 * Transform raw LogEntries into the processed tree consumed by the renderer.
 *
 * Key steps (borrowed-and-simplified from cclogviewer's processor):
 *   1. Flatten each LogEntry.message.content[] into ToolCalls + text
 *   2. Pair tool_use with tool_result via tool_use_id (result lives in a later user event)
 *   3. For Agent (Task) tool calls, use toolUseResult.agentId → subagent trace
 *      (ByteDance fork's deterministic link — no heuristic matching needed)
 *   4. Recursively process subagent entries at depth+1
 */
export function processSession(
  mainEntries: LogEntry[],
  subagents: Map<string, SubagentTrace>,
  sourceFile: string,
): AnalyzeResult {
  const processed = processEntries(mainEntries, 0, subagents, new Set());
  const cwd = extractCwd(mainEntries);

  return {
    mainEntries: processed,
    subagents,
    summary: {} as Summary, // populated later by computeSummary()
    sourceFile,
    cwd,
  };
}

/** First cwd seen in any entry wins. */
function extractCwd(entries: LogEntry[]): string | undefined {
  for (const e of entries) {
    if (typeof e.cwd === 'string' && e.cwd.length > 0) return e.cwd;
  }
  return undefined;
}

function processEntries(
  raw: LogEntry[],
  depth: number,
  subagents: Map<string, SubagentTrace>,
  visitedSubagents: Set<string>,
): ProcessedEntry[] {
  // Phase 1: raw → ProcessedEntry (one-to-one, minus hidden types)
  const processed: ProcessedEntry[] = [];
  for (const rawEntry of raw) {
    const p = rawToProcessed(rawEntry, depth);
    if (p) processed.push(p);
  }

  // Phase 2: build tool-call index from assistant entries
  const toolCallIndex = new Map<string, ToolCall>();
  for (const p of processed) {
    for (const tc of p.toolCalls) {
      toolCallIndex.set(tc.id, tc);
    }
  }

  // Phase 3: walk raw entries again to find tool_results and pair them.
  // We need the raw entry here (not the processed one) because toolUseResult is
  // on the raw entry envelope, not inside the message content.
  for (const rawEntry of raw) {
    if (rawEntry.type !== 'user') continue;
    const content = rawEntry.message?.content;
    if (!Array.isArray(content)) continue;

    for (const block of content) {
      if (block.type !== 'tool_result') continue;
      const tc = toolCallIndex.get(block.tool_use_id);
      if (!tc) continue;

      tc.result = {
        content: block.content,
        isError: block.is_error === true,
        rawPayload: rawEntry.toolUseResult,
      };

      // Subagent drill-down: Agent tool → toolUseResult.agentId → subagent trace
      if (tc.name === AGENT_TOOL_NAME) {
        const agentId = rawEntry.toolUseResult?.agentId;
        if (agentId && subagents.has(agentId) && !visitedSubagents.has(agentId)) {
          visitedSubagents.add(agentId);
          const sa = subagents.get(agentId)!;
          const aggregates = {
            totalDurationMs: rawEntry.toolUseResult?.totalDurationMs,
            totalTokens: rawEntry.toolUseResult?.totalTokens,
            totalToolUseCount: rawEntry.toolUseResult?.totalToolUseCount,
          };
          sa.aggregates = aggregates;
          tc.subagent = {
            agentId,
            agentType: sa.meta.agentType,
            description: sa.meta.description,
            aggregates,
            // Recursive: process the subagent's entries at depth+1.
            // v1 assumes only 1 level of nesting but the recursion supports arbitrary depth.
            processed: processEntries(sa.entries, depth + 1, subagents, visitedSubagents),
          };
        }
      }
    }
  }

  return processed;
}

function rawToProcessed(raw: LogEntry, depth: number): ProcessedEntry | null {
  const uuid = raw.uuid ?? '';
  const parentUuid = raw.parentUuid ?? null;
  const timestamp = raw.timestamp ?? '';

  // v1 hides non-message plumbing events. They're noise for a trajectory view.
  if (
    raw.type === 'file-history-snapshot' ||
    raw.type === 'queue-operation' ||
    raw.type === 'last-prompt'
  ) {
    return null;
  }

  if (raw.type === 'system') {
    return {
      uuid,
      parentUuid,
      timestamp,
      depth,
      type: 'system',
      role: 'system',
      text: extractSystemText(raw),
      toolCalls: [],
    };
  }

  const msg = raw.message;
  if (!msg) return null;
  const role = msg.role;
  const content = msg.content;

  // Handle string content (plain user text)
  if (typeof content === 'string') {
    return {
      uuid,
      parentUuid,
      timestamp,
      depth,
      type: role === 'user' ? 'user-text' : 'assistant-text',
      role,
      text: content,
      toolCalls: [],
    };
  }

  if (!Array.isArray(content)) return null;

  // Array content — flatten blocks into one ProcessedEntry with toolCalls + concatenated text
  const toolCalls: ToolCall[] = [];
  const textParts: string[] = [];
  let hasThinking = false;
  let hasToolResult = false;
  let toolResultFor: string | undefined;

  for (const block of content as ContentBlock[]) {
    if (block.type === 'text') {
      textParts.push(block.text);
    } else if (block.type === 'thinking') {
      textParts.push(block.thinking);
      hasThinking = true;
    } else if (block.type === 'tool_use') {
      toolCalls.push({
        id: block.id,
        name: block.name,
        input: block.input,
      });
    } else if (block.type === 'tool_result') {
      hasToolResult = true;
      toolResultFor = block.tool_use_id;
      // tool_result text is attached to the ToolCall in phase 2, not kept here
    }
  }

  let type: ProcessedEntryType;
  if (role === 'user' && hasToolResult) type = 'user-tool-result';
  else if (role === 'user') type = 'user-text';
  else if (role === 'assistant' && hasThinking && toolCalls.length === 0 && textParts.length > 0) {
    type = 'assistant-thinking';
  } else type = 'assistant-text';

  const text = textParts.length > 0 ? textParts.join('\n\n') : undefined;
  const tokens = extractTokens(msg.usage);

  return {
    uuid,
    parentUuid,
    timestamp,
    depth,
    type,
    role,
    text,
    toolCalls,
    toolResultFor,
    tokens,
  };
}

function extractTokens(usage?: UsageStats): TokenMetrics | undefined {
  if (!usage) return undefined;
  return {
    inputTokens: usage.input_tokens ?? 0,
    outputTokens: usage.output_tokens ?? 0,
    cacheCreationTokens: usage.cache_creation_input_tokens ?? 0,
    cacheReadTokens: usage.cache_read_input_tokens ?? 0,
  };
}

function extractSystemText(raw: LogEntry): string {
  // system events have varied shapes; best-effort extraction
  const msg = raw.message as unknown;
  if (typeof msg === 'string') return msg;
  if (msg && typeof msg === 'object' && 'content' in msg) {
    const c = (msg as { content: unknown }).content;
    if (typeof c === 'string') return c;
  }
  return `[system: ${raw.type}]`;
}
