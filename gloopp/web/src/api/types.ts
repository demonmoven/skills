export type QuestStatus =
  | 'pending'
  | 'running'
  | 'reviewing'
  | 'waiting_input'
  | 'user_review'
  | 'success'
  | 'failed'
  | 'cancelled'
  | 'blocked'

export type QuestType = 'execute' | 'design'
export type QuestIntensity = 'quick' | 'standard' | 'deep' | 'adversarial'
export type WorkspaceMode = 'auto' | 'worktree' | 'copy' | 'readonly'
export type Verdict = 'pass' | 'request_changes' | 'reject'
export type ApplyStatus = '' | 'pending' | 'applied' | 'failed'
export type PhaseStatus = 'pending' | 'running' | 'done' | 'failed' | string
export type FinalizedBy = 'user' | 'policy' | 'automation' | '' | string
export type TrustTier = 'tier_0' | 'tier_1' | 'tier_2' | 'tier_3' | string

export interface ImpactSummary {
  what_changed?: string
  affected?: string[]
  not_touched?: string[]
  caveats?: string[]
}

export interface QuestInlineNote {
  time?: number
  content?: string
  author?: string
  [key: string]: unknown
}

export interface QuestPhaseTask {
  phase_idx?: number
  role?: string
  class?: string
  goal?: string
  name?: string
  display_name?: string
  read_only?: boolean
  end_signal?: string
  allowed_tools?: string[]
  adventurer_id?: string
  session_id?: string
  status?: PhaseStatus
  turns?: number
  rework_count?: number
  started_at_ms?: number
  ended_at_ms?: number
}

export interface WaitingInputState {
  question_id: string
  question_text: string
  phase_idx: number
  session_id?: string
  phase_type?: string
  asker?: string
  timeout_ms?: number
  timeout_action?: string
  resume_phase_idx?: number
  last_comment_seq?: number
  last_answer_id?: string
  processed_answer_ids?: string[]
  asked_at?: string
}

export interface QuestPolicyDecisionView {
  id?: string
  policy_name?: string
  action?: string
  reason?: string
  at_ms?: number
  [key: string]: unknown
}

export interface QuestRecoveryStateSummary {
  strategy?: string
  attempt?: number
  max_attempts?: number
  next_attempt_at_ms?: number
  status?: 'active' | 'exhausted' | 'waiting' | 'pending' | 'running' | 'succeeded' | 'failed' | string
  policy_name?: string
  last_error?: string
  [key: string]: unknown
}

export interface QuestPolicyView {
  effective_trust_tier?: TrustTier
  safety_floor_reason?: string
  last_policy_decision?: QuestPolicyDecisionView | null
  recovery_state_summary?: QuestRecoveryStateSummary | null
  [key: string]: unknown
}

export interface HumanExceptionItem {
  id: string
  quest_id: string
  reason: string
  recommended_action: string
  evidence_summary?: string
  risk_level: string
  priority?: number
  available_actions: string[]
  audit_ref: string
  source_status: string
  created_at_ms?: number
}

export interface AgentSessionState {
  session_id: string
  quest_id: string
  phase?: number
  agent_id?: string
  capability_tier?: string
  health_status?: string
  started_at_ms?: number
  first_output_at_ms?: number
  last_output_at_ms?: number
  exit_code?: number
  updated_at_ms?: number
}

export interface QuestMeta {
  id: string
  short_id?: string
  query: string
  inputs?: QuestArtifact[]
  outputs?: QuestArtifact[]
  type: QuestType
  original_request?: string
  intent_summary?: string
  constraints?: string[]
  expected_artifacts?: string[]
  workflow_mode?: 'direct' | 'checked' | 'goal' | string
  pinned_outcome_summary?: string
  intensity?: QuestIntensity
  status: QuestStatus
  warrior_id?: string
  mage_id?: string
  execute_agent_id?: string
  review_agent_id?: string
  pipeline_version?: number
  current_phase_idx?: number
  pipeline_name?: string
  phase_count?: number
  pipeline_def_hash?: string
  pipeline_def?: QuestPhaseTask[]
  phases?: QuestPhaseTask[]
  workspace_mode?: string
  connectors?: string[]
  workspace_path?: string
  base_working_dir?: string
  rework_count: number
  max_rework: number
  design_summary?: string
  parent_quest_id?: string
  auto_spawn_execute?: boolean
  child_execute_quest_id?: string
  spawn_error?: string
  group_id?: string
  fanout_leaf_id?: string
  ownership_scopes?: string[]
  merge_strategy?: 'single_leaf' | 'sequential' | 'no_merge' | string
  merge_owner_leaf_id?: string
  final_verdict?: string
  final_comment?: string
  warrior_summary?: string
  impact_summary?: ImpactSummary
  finalized_by?: FinalizedBy
  auto_passed_by_policy?: string
  auto_completed_by_policy?: string
  policy_decision_id?: string
  applied?: boolean
  apply_status?: ApplyStatus
  apply_error?: string
  apply_failed_at_ms?: number
  apply_failed?: boolean
  blocked_reason?: string
  blocked_reason_code?: string
  failure_attribution?: FailureAttribution
  resume_count?: number
  waiting_input?: WaitingInputState | null
  mage_score?: number
  max_turns_per_phase_override?: number
  max_duration_per_quest_ms_override?: number
  /** @deprecated Backend no longer returns this; derive from phases[].turns. */
  turns_used?: number
  /** @deprecated Backend no longer returns this; use config or max_turns_per_phase_override. */
  max_turns?: number
  /** @deprecated Backend no longer returns this; derive from timestamps. */
  duration_used_ms?: number
  /** @deprecated Backend no longer returns this; use config or max_duration_per_quest_ms_override. */
  max_duration_ms?: number
  /** @deprecated Notes live in quest events/trace, not quest meta. */
  notes?: QuestInlineNote[]
  /** @deprecated Comments live in quest events/trace, not quest meta. */
  comments?: QuestInlineNote[]
  diff_stat?: string
  diff_changed_files?: number
  diff_additions?: number
  diff_deletions?: number
  effect_type?: 'workspace_diff' | 'context_store' | 'external_side_effect' | 'none' | string
  policy_view?: QuestPolicyView
  human_exception?: HumanExceptionItem
  agent_sessions?: AgentSessionState[]
  workspace_diff_pending?: boolean
  queued?: boolean
  queue_position?: number
  starting?: boolean
  created_by: string
  official?: boolean
  created_at_ms: number
  updated_at_ms?: number
  started_at_ms?: number
  completed_at_ms?: number
  [key: string]: unknown
}

export interface FanoutLeaf {
  quest_id: string
  short_id?: string
  query: string
  status: QuestStatus | string
  fanout_leaf_id: string
  ownership_scopes?: string[]
  merge_strategy?: 'single_leaf' | 'sequential' | 'no_merge' | string
  merge_owner_leaf_id?: string
  warrior_id?: string
  mage_id?: string
  execute_agent_id?: string
  review_agent_id?: string
  workflow_mode?: 'direct' | 'checked' | 'goal' | string
  pinned_outcome?: string
  created_at_ms?: number
  updated_at_ms?: number
  completed_at_ms?: number
  is_current?: boolean
}

export interface FanoutGroup {
  group_id: string
  leaves: FanoutLeaf[]
}

export type ThreadAuthorRole = 'maker' | 'checker' | 'system' | 'human' | 'automation' | string

export interface ThreadArtifactRef {
  id: string
  artifact_type: 'DeliveryArtifact' | 'MakerReport' | 'ReviewReport' | 'Evidence' | 'VerificationArtifact' | 'DecisionNote' | string
  ref?: string
  role?: ThreadAuthorRole
}

export interface ThreadPost {
  post_id: string
  thread_id: string
  parent_reply_id?: string
  root_post_id: string
  causal_refs?: string[]
  author_identity?: string
  author_role: ThreadAuthorRole
  artifact_refs?: ThreadArtifactRef[]
  source_event_id?: number
  kind?: string
  content?: string
  created_at_ms: number
}

export interface QuestArtifact {
  id: string
  schema_version?: string
  kind: 'image' | 'file' | 'log' | 'document' | 'archive' | 'unknown' | string
  mime?: string
  name: string
  size: number
  sha256: string
  storage_path: string
  source?: string
  created_at_ms: number
}

export interface AdventurerFile {
  id: string
  name: string
  class: 'warrior' | 'mage'
  title?: string
  description?: string
  status: 'pending_setup' | 'active' | 'retired'
  agent: string
  model?: string
  custom_prompt?: string
  tools?: string[] | string
  level: number
  exp: number
  win_count: number
  lose_count: number
  created_at_ms: number
  updated_at_ms?: number
  [key: string]: unknown
}

export interface ExecutorInfo {
  id: string
  agent?: string
  name: string
  type: 'acp' | 'cli' | 'mock'
  command?: string
  args?: string[]
  env?: Record<string, string>
  default_model?: string
  enabled: boolean
  official?: boolean
  capabilities?: string[]
  capability_tier?: string
  [key: string]: unknown
}

export type AutomationTrigger = 'manual' | 'schedule'
export type AutomationSource = 'official' | 'user'

export interface FanOutSpec {
  query: string
  warrior_id?: string
  mage_id?: string
  execute_agent_id?: string
  review_agent_id?: string
  leaf_id?: string
  ownership_scopes?: string[]
  merge_strategy?: 'single_leaf' | 'sequential' | 'no_merge' | string
  merge_owner_leaf_id?: string
}

export interface AutomationConfig {
  id: string
  name: string
  description?: string
  query: string
  quest_type?: QuestType
  intensity?: QuestIntensity
  working_dir?: string
  workspace_mode?: WorkspaceMode
  trigger?: AutomationTrigger
  cron?: string
  auto_start?: boolean
  triage_mode?: 'candidate' | 'direct' | string
  auto_apply?: boolean
  allow_quick_auto_complete?: boolean
  allow_l2?: boolean
  with_design_phase?: boolean
  auto_spawn_execute?: boolean
  trust_tier?: TrustTier
  trust_tier_locked?: boolean
  enabled: boolean
  priority?: number
  source?: AutomationSource
  official_policy_version?: number
  runtime_policy_user_override?: boolean
  tags?: string[]
  fan_out?: FanOutSpec[]
  run_count?: number
  last_run_ms?: number
  created_at_ms?: number
  updated_at_ms?: number
  warrior_id?: string
  mage_id?: string
  execute_agent_id?: string
  review_agent_id?: string
  connectors?: string[]
  [key: string]: unknown
}

export interface AutomationDiscoveryArchive {
  id: string
  automation_id: string
  outcome: 'no_finding' | string
  reason?: string
  summary?: string
  created_at_ms?: number
  [key: string]: unknown
}

export interface SkillManifest {
  name: string
  description?: string
  version?: string
  category?: string
  warrior_available?: boolean
  mage_available?: boolean
  requires_bins?: string[]
  source?: string
  path?: string
  size_bytes?: number
  updated_at_ms?: number
  [key: string]: unknown
}

export interface PromptTemplate {
  name: string
  source: string
  size_bytes: number
  preview?: string
  is_overridden: boolean
  content?: string
}

export interface PersonalStats {
  total_quests?: number
  success_quests?: number
  failed_quests?: number
  applied_quests?: number
  total_turns?: number
  loop_health?: PersonalLoopHealth
  anomaly_signals?: PersonalAnomalySignal[]
  [key: string]: unknown
}

export interface PersonalLoopHealth {
  started_sessions?: number
  first_output_sessions?: number
  first_output_coverage?: number
  late_failure_count?: number
  human_exception_open_count?: number
  human_exception_open_age_ms?: number
  maker_phase_done_count?: number
  maker_report_count?: number
  maker_report_coverage?: number
  review_phase_done_count?: number
  review_report_count?: number
  review_report_coverage?: number
  evidence_backed_review_count?: number
  evidence_backed_review_coverage?: number
  automation_no_finding_count?: number
  automation_candidate_count?: number
  automation_no_finding_rate?: number
  tier_c_session_count?: number
  tier_c_usage?: number
}

export interface PersonalDrilldown {
  query?: Record<string, string[]>
}

export interface PersonalAnomalySignal {
  key: string
  label: string
  severity: 'ok' | 'warn' | 'bad' | string
  count: number
  drilldown?: PersonalDrilldown
}

export interface QuestEvent {
  type: string
  quest_id?: string
  session_id?: string
  payload?: unknown
  ts?: number
  app?: string
  qid?: string
  mascot?: string
  timestamp?: number
}

export interface FailureAttributionPayload {
  stage?: string
  reason?: string
  category?: string
  message?: string
  recoverable?: boolean
  actions?: string[]
  details?: Record<string, unknown>
  [key: string]: unknown
}

export type FailureAttribution = FailureAttributionPayload

export interface DiffFileStat {
  name: string
  additions?: number
  deletions?: number
  status?: string
  [key: string]: unknown
}

export interface DiffResult {
  diff?: string
  stat?: string
  source?: string
  backup_id?: string
  changed_files?: number
  additions?: number
  deletions?: number
  files?: DiffFileStat[]
}

export interface SafetyWarning {
  severity: 'high' | 'medium' | 'low' | string
  category: string
  message: string
  path?: string
}

export interface AllowedCommand {
  id: string
  description?: string
  command: string
  args?: string[]
  allow_extra_args?: boolean
  timeout_ms?: number
  side_effect_level?: 'L0' | 'L1' | 'L2' | string
}

export interface CommandRecommendation extends AllowedCommand {
  reason?: string
  source?: string
  present?: boolean
}

export interface ApplyBackupMeta {
  id: string
  quest_id: string
  mode: string
  base_dir: string
  base_branch?: string
  base_commit?: string
  apply_commit?: string
  work_commit?: string
  files_changed?: number
  created_at_ms: number
  bundle_path?: string
  patch_path?: string
}

export interface WorkspaceCleanupItem {
  quest_id: string
  short_id?: string
  status?: QuestStatus | string
  workspace_path?: string
  completed_at_ms?: number
  age_days?: number
  removed?: boolean
  error?: string
  [key: string]: unknown
}

export interface QuestSessionRow {
  seq: number
  ts: number
  kind: string
  sid?: string
  role?: string
  content?: string
  tool_name?: string
  tool_id?: string
  tool_args?: string
  tool_result?: string
  error?: string
  phase?: number
  status?: string
  meta?: Record<string, unknown>
}

// ===== Context Pack 相关类型 =====
// 对应后端 prompt.ContextBlock
export interface ContextBlock {
  name: string
  source?: string
  trust?: 'platform' | 'user' | 'agent_output' | 'mixed' | string
  phase?: string
  role?: 'control' | 'primary_input' | 'evidence' | 'history' | 'protocol' | 'supporting' | string
  staleness?: 'live' | 'phase_snapshot' | 'previous_review' | 'previous_round' | 'original' | 'historical' | string
  priority?: number
  user_controlled?: boolean
  truncated?: boolean
  content: string
}

// 对应后端 prompt.ContextBlockSummary
export interface ContextBlockSummary {
  name: string
  source?: string
  trust?: 'platform' | 'user' | 'agent_output' | 'mixed' | string
  phase?: string
  role?: 'control' | 'primary_input' | 'evidence' | 'history' | 'protocol' | 'supporting' | string
  staleness?: 'live' | 'phase_snapshot' | 'previous_review' | 'previous_round' | 'original' | 'historical' | string
  priority?: number
  user_controlled?: boolean
  chars: number
  truncated?: boolean
  preview?: string
}

// 对应后端 prompt.ContextPackSummary
export interface ContextPackSummary {
  kind: string
  blocks: ContextBlockSummary[]
  original_chars: number
  rendered_chars: number
  truncated?: boolean
}

// Context pack 完整数据（summary + blocks + rendered 原文）
export interface ContextPackData {
  summary?: ContextPackSummary
  blocks?: ContextBlock[]
  rendered?: string
}

export interface DesignDoc {
  quest_id: string
  schema_version: string
  original_query: string
  summary: string
  review_comment?: string
  goals?: string[]
  non_goals?: string[]
  implementation_steps?: string[]
  risks?: string[]
  acceptance_criteria?: string[]
  execute_prompt: string
  created_at_ms: number
}

export type QuestCheck = Record<string, unknown>

export interface QuestReview {
  ts: number
  verdict: Verdict
  comment: string
  reviewed_by: string
  score?: number
  rewrite_hints?: string
  repair_count?: number
  total_issue_count?: number
  // Phase 1.5 起由调用方在写入时构造（见 mage-review-spec v0.2.2 §0.2.1）。
  // 历史数据 / Phase 1 数据此字段为 undefined。
  review_source?: {
    source_role?: string
    source_class?: string
    source_phase_idx?: number
    source_adventurer_id?: string
    source_session_id?: string
    source_incomplete?: boolean
  }
  // Phase 1.5+ typed 评审协议（见 spec §4.1）。校验失败时后端不持久化此字段。
  structured_review?: StructuredReview
}

export interface MakerReport {
  schema_version: string
  report_id: string
  quest_id: string
  session_id?: string
  phase: number
  verdict: string
  summary: string
  artifacts?: string[]
  changeset?: string
  evidence?: string[]
  impact?: Record<string, unknown>
  open_questions?: string[]
  transition?: boolean
  created_at_ms: number
}

export interface ReviewReport {
  schema_version: string
  report_id: string
  quest_id: string
  session_id?: string
  phase: number
  verdict: Verdict
  score?: number
  checked_against: string[]
  evidence_refs?: ReportEvidenceRef[]
  findings: {
    disagreements: unknown[]
    risks_extra: unknown[]
    endorsements: unknown[]
    referenced_checks: unknown[]
  }
  required_changes?: string[]
  residual_risks?: string[]
  confidence: string
  transition_note?: string
  comment?: string
  rewrite_hints?: string
  structured_review?: StructuredReview
  created_at_ms: number
}

export interface ReportEvidenceRef {
  id: string
  kind: string
  trust_tier: string
  source: string
  command_id?: string
  exit_code?: number
  duration_ms?: number
  timed_out?: boolean
  output_path?: string
  snippet?: string
  created_at_ms: number
}

export interface QuestReports {
  maker_reports: MakerReport[]
  review_reports: ReviewReport[]
}

export interface LoopStateSpine {
  schema_version: string
  quest_id: string
  current_goal?: string
  current_phase?: string
  attempts?: Array<{
    run_id: string
    summary?: string
    result?: string
    reason?: string
    created_at_ms: number
  }>
  confirmed_facts?: string[]
  failed_paths?: string[]
  open_blockers?: string[]
  next_expected_action?: string
  human_exceptions?: string[]
  updated_at_ms: number
}

// ---- StructuredReview 类型（spec §4.1）----

export type EvidenceTargetKind =
  | 'warrior_message' | 'artifact' | 'check' | 'diff_file'
  | 'workspace_effect' | 'policy_target' | 'quest' | 'design_doc' | 'other'

export type MageDimension =
  | 'correctness' | 'completeness' | 'boundary' | 'risk'
  | 'validation' | 'coherence' | 'scope' | 'style' | 'other'

export type MageSeverity = 'high' | 'med' | 'low' | 'info'

export interface EvidenceRef {
  id: string
  kind: EvidenceTargetKind
  anchor?: string
  field?: string
  snippet?: string
}

export interface MageDisagreement {
  id: string
  dimension: MageDimension
  severity: MageSeverity
  target_kind: EvidenceTargetKind
  claim: string
  evidence: EvidenceRef[]
  suggestion?: string
}

export interface MageExtraRisk {
  id: string
  kind: string
  severity: MageSeverity
  detail: string
  evidence: EvidenceRef[]
}

export interface MageEndorsement {
  id: string
  dimension: MageDimension
  reason: string
  evidence: EvidenceRef[]
}

export interface MageCheckReference {
  check_id: string
  expected?: string
  comment?: string
}

export interface FreeCommentRange {
  start_char: number
  end_char: number
}

export interface StructuredReview {
  schema_version: string
  reviewed_warrior_phase_idx: number
  reviewed_warrior_session_id?: string
  reviewed_warrior_latest_turn?: number
  coverage_ratio?: [number, number]
  confidence?: string
  disagreements: MageDisagreement[]
  risks_extra: MageExtraRisk[]
  endorsements: MageEndorsement[]
  referenced_checks?: MageCheckReference[]
  free_comment_range?: FreeCommentRange
}

export type UpdateState = 'outdated' | 'up-to-date' | 'unknown'

export interface UpdateStatus {
  current: string
  latest?: string
  state: UpdateState
  last_checked_at?: number
  next_check_at?: number
  upgrade_hint?: string
  last_error?: string
}

// ==================== Context (用户上下文) ====================

export interface ContextDimInfo {
  name: string
  title: string
  description: string
  updated_at_ms: number
  size_bytes: number
  source?: string
}

export interface ContextDim extends ContextDimInfo {
  body: string
}

export interface ContextMeta {
  updated_at_ms: number
  generated_by?: string
  dimensions: ContextDimInfo[]
  total_size_bytes: number
}

export interface KnowledgeExportFile {
  path: string
  size_bytes: number
}

export interface KnowledgeExportPreview {
  format: string
  generated_at_ms: number
  files: KnowledgeExportFile[]
  file_count: number
  total_size_bytes: number
  dimension_count: number
  summary_included: boolean
  diff?: KnowledgeExportDiff
}

export interface KnowledgeExportMeta {
  format: string
  exported_at_ms: number
  filename?: string
  file_count: number
  total_size_bytes: number
  dimension_count: number
  files?: KnowledgeExportFile[]
}

export interface KnowledgeExportDiff {
  added: KnowledgeExportFile[]
  changed: KnowledgeExportFile[]
  removed: KnowledgeExportFile[]
  unchanged: number
}

export interface ConfigPackagePreviewSection {
  name: string
  creates: number
  updates: number
  skips: number
  warnings?: string[]
}

export interface ConfigPackagePreview {
  ok: boolean
  sections: ConfigPackagePreviewSection[]
  warnings?: string[]
}

// ==================== Activity Feed（首页 Feed 流） ====================

export type ActivityKind = 'user' | 'post' | 'summary' | 'attention'

export interface ActivityItem {
  ts: number
  kind: ActivityKind
  quest_id: string
  thread_id?: string
  root_post_id?: string
  short_id?: string
  query: string
  project: string
  warrior_id?: string
  warrior_name?: string
  warrior_level?: number
  warrior_title?: string
  warrior_class?: string
  warrior_exp?: number
  status: string
  summary?: string
  impact?: ImpactSummary | null
  source?: string
  post_id?: string
  reply_to?: string
  author_role?: 'maker' | 'checker' | 'system' | 'human' | 'automation' | string
  causal_refs?: string[]
  post_kind?: string // post / milestone / blocker
  source_event_id?: number // v0.5: 事件来源标识（先用 existing event row id 过渡）
  // 委托帖上下文（kind=user）：@ 了哪些冒险者、派生自/生成了哪些委托
  quest_type?: string
  warrior_adv_id?: string
  mage_adv_id?: string
  warrior_adv_name?: string
  mage_adv_name?: string
  parent_quest_id?: string
  parent_short_id?: string
  child_quest_id?: string
  child_short_id?: string
  group_id?: string
  fanout_fold?: boolean
  fanout_leaf_count?: number
  fanout_leaf_ids?: string[]
  fanout_quest_ids?: string[]
  merge_owner_leaf_id?: string
  reply_count?: number
  reply_previews?: ActivityReplyPreview[]
  triage_mode?: 'candidate' | 'direct' | string // v0.5.4: candidate quest 在 Feed 暴露 triage_mode，供确认入口
  human_exception_id?: string
  available_actions?: string[]
  recommended_action?: string
  risk_level?: 'high' | 'medium' | 'low' | string
  attention_priority?: number
  blocked_reason_code?: string
  blocked_category?: string
  action_endpoint_hint?: 'resolve-blocked' | 'answer' | 'resolve-user-review' | 'apply-discard' | string
}

export interface ActivityReplyPreview {
  ts: number
  post_id: string
  reply_to?: string
  author_role?: 'maker' | 'checker' | 'system' | 'human' | 'automation' | string
  post_kind?: string
  source_event_id?: number
  summary?: string
  causal_refs?: string[]
}

export interface ActivityResponse {
  items: ActivityItem[]
  projects: string[]
}

// ==================== ThreadView (P0b-4b) ====================

export interface ThreadActor {
  identity: string
  role: 'maker' | 'checker' | 'system' | 'human' | 'automation' | string
  name: string
  class?: string
  source_type: 'adventurer' | 'system' | 'automation' | string
}

export interface ThreadFanoutLeafPreview {
  quest_id: string
  short_id?: string
  status: string
  leaf_id: string
  latest_post_ts?: number
  latest_post_preview?: string
}

export interface ThreadFanoutSummary {
  group_id: string
  is_root: boolean
  is_leaf: boolean
  root_quest_id?: string
  leaves?: ThreadFanoutLeafPreview[]
}

export interface ThreadActionEntrypoint {
  action_type: string
  target_section: string
  evidence_post_id?: string
  priority: number
  hint?: string
}

export interface ThreadViewResponse {
  ok: boolean
  quest: {
    id: string
    short_id?: string
    query: string
    status: QuestStatus
    created_by: string
    warrior_id?: string
    mage_id?: string
    created_at_ms: number
    updated_at_ms?: number
  }
  posts: ThreadPost[]
  actors: ThreadActor[]
  fanout_summary?: ThreadFanoutSummary | null
  action_entrypoints?: ThreadActionEntrypoint[]
}
