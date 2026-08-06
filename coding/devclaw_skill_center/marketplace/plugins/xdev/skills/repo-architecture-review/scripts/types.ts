// scripts/types.ts

export type Language = 'go' | 'python' | 'typescript' | 'javascript' | 'java' | 'kotlin' | 'rust' | 'ruby' | 'php' | 'csharp' | 'cpp' | 'other';

/** Layering convention discovered by LLM during L0 pre-scan. */
export interface LayeringConvention {
  /** Whether the repo has a recognisable layering convention. */
  hasLayering: boolean;
  /** Ordered layer names from outermost (entry) to innermost (infra). */
  layers: string[];
  /** Layers whose siblings should be isolated (horizontal coupling check). */
  businessLayers: string[];
  /** Layers where sibling sharing is legitimate (dal/dao/util). */
  infraLayers: string[];
  /** Human-readable summary of the convention, e.g. "handler → service → dao → dal". */
  conventionSummary: string;
  /** LLM's reasoning for the detected convention. */
  reasoning: string;
}

export interface RepoProfile {
  rootPath: string;
  languages: Array<{ lang: Language; fileCount: number; locTotal: number }>;
  hasGoMod: boolean;
  goModulePath?: string;                  // e.g. "code.byted.org/flow/creativity"
  hasPyProject: boolean;
  hasSetupCfg: boolean;
  hasRequirementsTxt: boolean;
  dockerfiles: string[];                  // relative paths
  ghaWorkflows: string[];                 // relative paths
  topLevelPackages: {                     // per-language root packages
    go?: string[];                        // e.g. ['internal/domain', 'internal/infra', 'cmd/api']
    python?: string[];                    // e.g. ['mypkg', 'mypkg.domain', 'mypkg.infra']
  };
  workspaces: string[];                   // for monorepos; empty for single-module repos
  totalFiles: number;
  totalLocMillions: number;
  gitLogLineEstimate: number;
  detectedConfigs: {                      // which tool configs already exist
    importLinter?: string;                // path to pyproject.toml / .importlinter
    archGo?: string;                      // path to arch-go.yml
    hadolint?: string;                    // path to .hadolint.yaml
    actionlint?: string;                  // path to .actionlint.yaml
  };
  /** Layering convention discovered via LLM analysis of the package structure. */
  layering?: LayeringConvention;
}

export type Category =
  | 'structure'
  | 'dependency'
  | 'boundary'
  | 'layering'
  | 'coupling'
  | 'build-deploy'
  | 'test-org'
  | 'config'
  | 'data-layer'
  | 'security-boundary'
  | 'dead-code'
  | 'doc-drift';

export type Severity = 'critical' | 'high' | 'medium' | 'low';

export type RuleSource = 'repo-config' | 'skill-generated' | 'skill-builtin';

export interface Location {
  file: string;                           // relative to repo root
  line?: number;
  symbol?: string;
  module?: string;
}

export interface Evidence {
  metric?: { name: string; value: number; threshold: number; percentile?: number };
  signals?: string[];
  rawRef?: string;                        // path under .architecture-review/tmp/
}

export interface Violation {
  id: string;                             // stable hash
  source: string;                         // 'l1-tool:import-linter' / 'l2-graph:modularity' / ...
  category: Category;
  ruleId: string;
  ruleSource: RuleSource;
  severity: Severity;
  locations: Location[];                  // never empty
  evidence: Evidence;
  message: string;
}

export interface Finding {
  id: string;                             // = groupId
  category: Category;
  title: string;                          // 1-line
  rootCause: string;                      // fuser output
  impact: string;
  actions: string[];
  confidence: 'high' | 'medium' | 'low';
  confidenceScore: number;                // 0..1 after multipliers
  severity: Severity;
  signals: Array<{ source: string; description: string }>;
  locations: Location[];
  violationIds: string[];                 // member violations
}

export interface ToolRunResult {
  toolId: string;
  status: 'ok' | 'skipped' | 'failed';
  reason?: 'no-config' | 'no-binary' | 'not-applicable' | 'timeout' | 'install-failed' | 'exec-failed';
  durationMs: number;
  generatedConfigPath?: string;           // if auto-config wrote a file
  installedVia?: 'pipx' | 'go-install' | 'docker' | 'binary' | 'pre-existing';
  rawOutputPath?: string;                 // .architecture-review/tmp/<tool>.json
  violations: Violation[];
  error?: string;
}

export interface RunMeta {
  startedAt: string;                      // ISO
  finishedAt: string;
  wallClockMs: number;
  toolRuns: ToolRunResult[];
  generatedFiles: string[];
  coverageDowngrades: string[];           // human-readable reasons
  stackDetection: string[];               // short strings like "Go + Python + Docker + GHA"
  mapReduceUsed: boolean;
  resumeSource?: string;
}
