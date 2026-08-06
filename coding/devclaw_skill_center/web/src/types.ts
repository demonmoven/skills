/** Session list item returned by GET /api/sessions */
export interface SessionListItem {
  sessionId: string;
  toolName: string;
  userId: string;
  date: string;
  objectKey: string;
  hasAnalysis: boolean;
  department?: string;
  nickname?: string;
}

export interface SessionListResponse {
  sessions: SessionListItem[];
  total: number;
  page: number;
  pageSize: number;
  departments?: string[];
}

/** Per-turn data in analysis result */
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

/** Normalized message for conversation tab */
export interface NormalizedMessage {
  role: 'user' | 'assistant' | 'system';
  timestamp: string;
  content: string;
  toolCalls?: Array<{ name: string; input: string; toolUseId?: string }>;
  toolResults?: Array<{ toolName: string; isError: boolean; content: string; agentId?: string }>;
  thinking?: string;
  model?: string;
  subagentId?: string;
}

/** Metadata for a single subagent */
export interface SubagentMeta {
  agentId: string;
  slug: string;
  agentType: string;
  description: string;
  messageCount: number;
  totalTokens: number;
  totalCost: number;
}

/** Aggregated subagent statistics */
export interface SubagentStats {
  totalSubagents: number;
  subagents: SubagentMeta[];
  totalSubagentTokens: number;
  totalSubagentCost: number;
  agentTypeDistribution: Record<string, number>;
  subagentToolCounter: Record<string, number>;
  subagentErrorRate: number;
}

/** LLM-generated harness analysis */
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
  turn: string;
  behavior: string;
  expectation: string;
  gap: string;
  severity: 'critical' | 'high' | 'medium';
}

/** An inefficiency pattern detected during evaluation */
export interface Inefficiency {
  pattern: string;
  description: string;
  wastedTurns: number;
  suggestion: string;
}

/** LLM-as-Judge evaluation result */
export interface EvalResult {
  score: number;
  taskCompletion: number;
  userAlignment: number;
  efficiency: number;
  errorHandling: number;
  decisionQuality: number;
  communication: number;
  codeQuality: number;
  harnessUsage: number;
  comment: string;
  userCorrections?: UserCorrection[];
  inefficiencies?: Inefficiency[];
  evaluatedAt: string;
  evaluatedBy: string;
  harnessAnalysis?: HarnessAnalysis;
  harnessStats?: {
    totalArtifacts: number;
    usedArtifacts: number;
    unusedArtifacts: number;
    repoScanned: boolean;
  };
}

/** Full session analysis result */
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
  subagentConversations?: Record<string, {
    agentId: string;
    agentType: string;
    description: string;
    messages: NormalizedMessage[];
  }>;
}
