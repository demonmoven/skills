/** Raw JSONL line — untyped, any tool format */
export type RawJsonlLine = Record<string, unknown>;

/** Normalized message — unified model across all tool formats */
export interface NormalizedMessage {
  role: 'user' | 'assistant' | 'system';
  timestamp: string;
  content: string;
  toolCalls?: Array<{ name: string; input: string; toolUseId?: string }>;
  toolResults?: Array<{ toolName: string; isError: boolean; content: string; agentId?: string }>;
  thinking?: string;
  model?: string;
  usage?: {
    inputTokens: number;
    outputTokens: number;
    cacheWriteTokens: number;
    cacheReadTokens: number;
  };
  /** If this message belongs to a subagent, the agentId. Undefined for main session messages. */
  subagentId?: string;
}

// ─── Subagent Types ───────────────────────────────────────────────

/** Metadata for a single subagent within a session */
export interface SubagentMeta {
  agentId: string;
  slug: string;               // human-readable name, e.g. "expressive-cooking-owl"
  agentType: string;           // "general-purpose", "Explore", "Plan", etc.
  description: string;         // from .meta.json
  messageCount: number;
  totalTokens: number;
  totalCost: number;
}

/** Aggregated subagent statistics for a session */
export interface SubagentStats {
  totalSubagents: number;
  subagents: SubagentMeta[];
  totalSubagentTokens: number;
  totalSubagentCost: number;
  agentTypeDistribution: Record<string, number>;
  subagentToolCounter: Record<string, number>;
  subagentErrorRate: number;
}

/** Input container representing a full session with subagent data */
export interface SessionTree {
  mainJsonlContent: string;
  subagents: Array<{
    agentId: string;
    jsonlContent: string;
    meta: { agentType: string; description: string };
  }>;
}

export interface PerTurnData {
  turn: number;
  timestamp: string;
  model: string;
  inputTokens: number;
  outputTokens: number;
  cacheWrite: number;
  cacheRead: number;
  cost: number;
}

// ─── Harness Metrics ───────────────────────────────────────────────

/** A single harness artifact detected in the session trace */
export interface HarnessArtifact {
  path: string;
  category: 'doc-root' | 'doc-rules' | 'doc-guidance' | 'doc-reference' | 'doc-quality' | 'doc-other' | 'skill' | 'hook';
  readCount: number;
  mentionCount: number;
  /** Number of times the artifact was injected into the prompt via system/user messages
   * (e.g. "Instructions from:" patterns in Claude Code, or AGENTS.md auto-loaded as system context) */
  injectedCount: number;
  firstReadTurn: number | null;
}

/** Programmatic harness usage statistics extracted from NormalizedMessage[] */
export interface HarnessMetrics {
  artifacts: HarnessArtifact[];
  totalArtifacts: number;
  usedArtifacts: number;
  unusedArtifacts: number;
  skillInvocations: Array<{ name: string; count: number }>;
  execPlanUsed: boolean;
  docReadBeforeEdit: boolean;
  /** Whether the full repo was scanned for harness files (vs. trace-only discovery) */
  repoScanned: boolean;
}

/** LLM-generated harness analysis (part of EvalResult) */
export interface HarnessAnalysis {
  summary: string;
  usedArtifacts: Array<{
    path: string;
    readCount: number;
    mechanism: string;
    evidence: string;
  }>;
  unusedArtifacts: Array<{
    path: string;
    intendedPurpose: string;
  }>;
  mechanismEffectiveness: {
    autoInjected: string;
    hardConstraints: string;
    explicitInvocation: string;
    passiveDocumentation: string;
  };
}

/** A user correction detected during evaluation */
export interface UserCorrection {
  turn: string;           // approximate position in conversation
  behavior: string;       // what assistant did
  expectation: string;    // what user wanted
  gap: string;            // generalized pattern
  severity: 'critical' | 'high' | 'medium';
}

/** An inefficiency pattern detected during evaluation */
export interface Inefficiency {
  pattern: string;        // generalized pattern name
  description: string;    // what happened
  wastedTurns: number;    // estimated wasted turns
  suggestion: string;     // what should have been done
}

/** LLM-as-Judge evaluation result */
export interface EvalResult {
  score: number;          // 0-1 overall score
  taskCompletion: number; // 0-1
  userAlignment: number;  // 0-1
  efficiency: number;     // 0-1
  errorHandling: number;  // 0-1
  decisionQuality: number; // 0-1
  communication: number;  // 0-1
  codeQuality: number;    // 0-1
  harnessUsage: number;   // 0-1
  comment: string;        // 200-500 char Chinese commentary
  userCorrections?: UserCorrection[];
  inefficiencies?: Inefficiency[];
  evaluatedAt: string;    // ISO timestamp
  evaluatedBy: string;    // 'claude' | 'opencode'
  harnessAnalysis?: HarnessAnalysis;
  /** Programmatic harness stats (attached server-side, not from LLM output) */
  harnessStats?: {
    totalArtifacts: number;
    usedArtifacts: number;
    unusedArtifacts: number;
    repoScanned: boolean;
  };
}

export interface SessionAnalysis {
  sessionId: string;
  toolName: string;
  duration: number;
  durationStr: string;
  startTime: string;
  endTime: string;
  userMsgCount: number;
  assistantMsgCount: number;
  systemMsgCount: number;
  totalToolCalls: number;
  toolCounter: Record<string, number>;
  toolErrorMap: Record<string, { calls: number; errors: number }>;
  toolErrorsTotal: number;
  filesEdited: Record<string, number>;
  thinkingBlocks: number;
  retries: Array<{ tool: string; count: number }>;
  modelsUsed: Record<string, number>;
  totalInput: number;
  totalOutput: number;
  totalCacheWrite: number;
  totalCacheRead: number;
  totalCost: number;
  cacheEfficiency: number;
  hasUsageData: boolean;
  cumulativeCosts: Array<{ turn: number; cost: number }>;
  perTurnData: PerTurnData[];
  latencies: number[];
  errorRate: number;
  retryRate: number;
  thinkingRatio: number;
  toolDiversity: number;
  efficiencyScore: number;
  conversation: NormalizedMessage[];
  evaluation?: EvalResult;
  subagentStats?: SubagentStats;
  /** Subagent conversations keyed by agentId, for inline expansion in the conversation view */
  subagentConversations?: Record<string, {
    agentId: string;
    agentType: string;
    description: string;
    messages: NormalizedMessage[];
  }>;
}
