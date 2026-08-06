/**
 * Types for `xdev trace analyze`.
 *
 * Two layers:
 *   1. LogEntry — raw JSONL line shape (matches Claude Code's on-disk format)
 *   2. ProcessedEntry — enriched model used by renderer (depth, tool pairing, subagent linking)
 */

// ============ Raw JSONL (Layer 1) ============

export type LogEntryType =
  | 'user'
  | 'assistant'
  | 'system'
  | 'file-history-snapshot'
  | 'queue-operation'
  | 'last-prompt'
  | string; // tolerate unknown future types

export interface LogEntry {
  type: LogEntryType;
  uuid?: string;
  parentUuid?: string | null;
  sessionId?: string;
  timestamp?: string;
  cwd?: string;
  gitBranch?: string;
  version?: string;
  isSidechain?: boolean;
  userType?: string;
  entrypoint?: string;
  message?: RawMessage;
  toolUseResult?: ToolUseResultPayload;
  sourceToolAssistantUUID?: string;
  promptId?: string;
  permissionMode?: string;
  slug?: string;
  // Catch-all for fields we don't care about
  [key: string]: unknown;
}

export interface RawMessage {
  role: 'user' | 'assistant';
  content: string | ContentBlock[];
  model?: string;
  id?: string;
  usage?: UsageStats;
  stop_reason?: string | null;
  stop_sequence?: string | null;
}

export type ContentBlock =
  | { type: 'text'; text: string }
  | { type: 'thinking'; thinking: string; signature?: string }
  | { type: 'tool_use'; id: string; name: string; input: unknown; caller?: { type: string } }
  | {
      type: 'tool_result';
      tool_use_id: string;
      content: string | Array<{ type: 'text' | 'image'; text?: string }>;
      is_error?: boolean;
    };

export interface UsageStats {
  input_tokens?: number;
  output_tokens?: number;
  cache_creation_input_tokens?: number;
  cache_read_input_tokens?: number;
  cache_creation?: {
    ephemeral_5m_input_tokens?: number;
    ephemeral_1h_input_tokens?: number;
  };
  service_tier?: string;
}

export interface ToolUseResultPayload {
  status?: string;
  agentId?: string;
  agentType?: string;
  description?: string;
  prompt?: string;
  content?: unknown;
  totalDurationMs?: number;
  totalTokens?: number;
  totalToolUseCount?: number;
  usage?: UsageStats;
  // For file read/edit results
  type?: string;
  file?: { filePath?: string; content?: string; numLines?: number };
}

// ============ Subagent (Layer 1.5) ============

export interface SubagentMeta {
  agentType: string;
  description: string;
}

export interface SubagentTrace {
  agentId: string;
  meta: SubagentMeta;
  entries: LogEntry[];
  // Populated by processor from main-session toolUseResult
  aggregates?: {
    totalDurationMs?: number;
    totalTokens?: number;
    totalToolUseCount?: number;
  };
}

// ============ Processed (Layer 2 — what the renderer consumes) ============

export type ProcessedEntryType =
  | 'user-text'         // user typed text
  | 'user-tool-result'  // user event wrapping a tool_result
  | 'assistant-text'    // assistant text block
  | 'assistant-thinking'
  | 'system'
  | 'other';            // file-history / queue-operation / etc.

export interface TokenMetrics {
  inputTokens: number;
  outputTokens: number;
  cacheCreationTokens: number;
  cacheReadTokens: number;
}

export interface ToolCall {
  id: string;
  name: string;
  input: unknown;
  /**
   * Result block once paired. Null until paired.
   */
  result?: {
    content: unknown;
    isError: boolean;
    /** Raw toolUseResult sidecar on the user event that carried this result. */
    rawPayload?: ToolUseResultPayload;
  };
  /**
   * If this tool call is an `Agent` (Task) tool, link to the subagent's processed tree.
   */
  subagent?: {
    agentId: string;
    agentType: string;
    description: string;
    aggregates: {
      totalDurationMs?: number;
      totalTokens?: number;
      totalToolUseCount?: number;
    };
    /** Processed entries of the subagent's own trace, depth-shifted. */
    processed: ProcessedEntry[];
  };
}

export interface ProcessedEntry {
  uuid: string;
  parentUuid: string | null;
  timestamp: string;
  depth: number;                // 0 = main trunk, 1+ = inside subagent
  type: ProcessedEntryType;
  role?: 'user' | 'assistant' | 'system';

  /** Text content (for user-text / assistant-text / assistant-thinking). */
  text?: string;

  /** Tool calls attached to an assistant entry. */
  toolCalls: ToolCall[];

  /** If this is a user event wrapping a tool_result, which tool_use it replies to. */
  toolResultFor?: string;

  /** Token metrics for assistant entries. */
  tokens?: TokenMetrics;

  /** Whether this entry is an error. */
  isError?: boolean;
}

// ============ Summary ============

export interface SkillInvocation {
  skillName: string;
  args?: string;
  timestamp: string;
  toolUseId: string;
  /** 0 = main thread, 1+ = inside a subagent */
  depth: number;
  isError?: boolean;
}

export interface Summary {
  sessionId: string;
  model?: string;
  startedAt?: string;
  endedAt?: string;
  durationMs: number;
  totalEntries: number;
  totalTurns: number;            // count of user-text entries (approx)
  totalToolCalls: number;
  toolDistribution: Record<string, number>;
  tokenTotals: {
    input: number;
    output: number;
    cacheRead: number;
    cacheCreation: number;
  };
  errorCount: number;
  subagentCount: number;
  subagentTypes: Record<string, number>;
  skillInvocations: SkillInvocation[];
  skillDistribution: Record<string, number>;
}

// ============ Work Directory ============

export interface WorkDirNode {
  /** Base filename (or dir name). */
  name: string;
  /** Path relative to the docs/ root of the cwd. */
  relativePath: string;
  isDir: boolean;
  /** Child nodes for directories. */
  children?: WorkDirNode[];
  /** Embedded file content (for text files under size limit). */
  content?: string;
  /** File size in bytes. */
  size?: number;
  /** If content was skipped, the reason (too large / binary / error). */
  contentSkipped?: string;
}

// ============ Final Analyze Result ============

export interface AnalyzeResult {
  mainEntries: ProcessedEntry[];
  subagents: Map<string, SubagentTrace>;
  summary: Summary;
  /** Source file path */
  sourceFile: string;
  /** Session working directory (from JSONL entries). */
  cwd?: string;
  /** Tree of files under <cwd>/docs/, or undefined if not present. */
  workDir?: WorkDirNode;
}
