package fsstore

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/executor"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== Quest（委托）结构：workspace/quests/<qid>/meta.json 等 ====================

const (
	DefaultPipelineName    = "default"
	DefaultPipelineVersion = 1
)

// PhaseTask 追踪单个阶段的执行状态（v2: warrior → mage 两个固定阶段）
type PhaseTask struct {
	PhaseIdx     int                   `json:"phase_idx"`
	Role         string                `json:"role"`  // warrior / mage
	Class        model.AdventurerClass `json:"class"` // 对应冒险者职业
	Goal         string                `json:"goal"`
	Name         string                `json:"name,omitempty"`
	DisplayName  string                `json:"display_name,omitempty"`
	ReadOnly     bool                  `json:"read_only,omitempty"`
	EndSignal    string                `json:"end_signal,omitempty"`
	ReworkTo     int                   `json:"rework_to,omitempty"` // 返工目标阶段索引，-1 表示不支持返工
	AllowedTools []string              `json:"allowed_tools,omitempty"`
	AdventurerID string                `json:"adventurer_id"`
	AgentID      string                `json:"agent_id,omitempty"`
	SessionID    string                `json:"session_id"`
	Status       model.PhaseStatus     `json:"status"` // pending / running / done / failed
	Turns        int                   `json:"turns"`
	ReworkCount  int                   `json:"rework_count"`
	StartedAtMs  int64                 `json:"started_at_ms,omitempty"`
	EndedAtMs    int64                 `json:"ended_at_ms,omitempty"`
}

type QuestMeta struct {
	ID      string            `json:"id"`
	ShortID string            `json:"short_id"`
	Query   string            `json:"query"` // 用户输入的原始文本（legacy root post body）
	Type    model.QuestType   `json:"type"`  // execute / design
	Status  model.QuestStatus `json:"status"`
	Inputs  []QuestArtifact   `json:"inputs,omitempty"` // 委托输入资产；meta 只存引用和校验信息，不存二进制

	// v0.5 thread-aware Post root fields. Query remains the compatibility body,
	// OriginalRequest is the immutable goal anchor used by the Post ledger.
	OriginalRequest      string             `json:"original_request,omitempty"`
	IntentSummary        string             `json:"intent_summary,omitempty"`
	Constraints          []string           `json:"constraints,omitempty"`
	ExpectedArtifacts    []string           `json:"expected_artifacts,omitempty"`
	WorkflowMode         model.WorkflowMode `json:"workflow_mode,omitempty"`
	PinnedOutcomeSummary string             `json:"pinned_outcome_summary,omitempty"`

	// 强度是平台内部预算档位；Gloop 不做语义判断，默认 standard，用户/automation 可显式覆盖。
	Intensity model.QuestIntensity `json:"intensity,omitempty"`

	// 冒险者
	WarriorID      string `json:"warrior_id"`
	MageID         string `json:"mage_id"`
	ExecuteAgentID string `json:"execute_agent_id,omitempty"`
	ReviewAgentID  string `json:"review_agent_id,omitempty"`

	// 阶段运行时状态（v2 阶段管道）
	// 与 WarriorID/MageID 双写双读，保持向后兼容。
	PipelineVersion int         `json:"pipeline_version,omitempty"`
	CurrentPhase    int         `json:"current_phase_idx,omitempty"`
	PipelineName    string      `json:"pipeline_name,omitempty"`
	PhaseCount      int         `json:"phase_count,omitempty"`
	PipelineDefHash string      `json:"pipeline_def_hash,omitempty"`
	PipelineDef     []PhaseTask `json:"pipeline_def,omitempty"`
	Phases          []PhaseTask `json:"phases,omitempty"`

	// 工作区
	WorkspaceMode      model.WorkspaceMode     `json:"workspace_mode"` // worktree/copy/readonly
	WorkspacePath      string                  `json:"workspace_path"`
	BaseWorkingDir     string                  `json:"base_working_dir"`
	BaseBranch         string                  `json:"base_branch,omitempty"`
	BaseCommit         string                  `json:"base_commit,omitempty"`
	WorkspaceDowngrade *WorkspaceDowngradeInfo `json:"workspace_downgrade,omitempty"`

	// 过程
	ReworkCount         int                `json:"rework_count"`
	MaxRework           int                `json:"max_rework"`
	GoalIterations      int                `json:"goal_iterations,omitempty"`     // HOTL v0.2: goal 模式当前迭代次数
	GoalMaxIterations   int                `json:"goal_max_iterations,omitempty"` // HOTL v0.2: goal 模式最大迭代次数
	GoalCondition       string             `json:"goal_condition,omitempty"`      // HOTL v0.2: goal 模式完成条件描述
	ReviewHints         string             `json:"review_hints,omitempty"`
	CommentCursor       int                `json:"comment_cursor,omitempty"`         // 已处理的用户评论数，跨运行周期去重用
	DesignSummary       string             `json:"design_summary,omitempty"`         // 方案型 quest 的方案摘要
	ParentQuestID       string             `json:"parent_quest_id,omitempty"`        // 从方案 quest 发起的执行 quest
	AutoSpawnExecute    bool               `json:"auto_spawn_execute,omitempty"`     // design 成功后自动创建执行委托
	ChildExecuteQuestID string             `json:"child_execute_quest_id,omitempty"` // design 自动/手动 spawn 的执行委托
	SpawnError          string             `json:"spawn_error,omitempty"`            // 最近一次自动 spawn 失败原因
	GroupID             string             `json:"group_id,omitempty"`               // fan-out 分组标签
	FanoutLeafID        string             `json:"fanout_leaf_id,omitempty"`         // fan-out leaf 标识；同一 group 内唯一
	OwnershipScopes     []string           `json:"ownership_scopes,omitempty"`       // leaf 声明的文件/模块/问题域 ownership
	MergeStrategy       string             `json:"merge_strategy,omitempty"`         // single_leaf | sequential | no_merge
	MergeOwnerLeafID    string             `json:"merge_owner_leaf_id,omitempty"`    // single_leaf / sequential 的合并责任 leaf
	AcceptanceCriteria  string             `json:"acceptance_criteria,omitempty"`    // 验收标准；满足时法师按此评审
	Connectors          []string           `json:"connectors,omitempty"`             // HOTL v0.2 环 4: quest 自主闭环后触发的 connector（如 "git"）；空=不触发。与 automation 继承的合并。
	TriageMode          string             `json:"triage_mode,omitempty"`            // candidate/direct；automation no-auto-start 候选投影
	ResumeCount         int                `json:"resume_count,omitempty"`           // blocked 后人工恢复次数
	WaitingInput        *WaitingInputState `json:"waiting_input,omitempty"`          // 等待用户回答的问题状态

	// 阻塞时长核算：quest 处于 blocked 期间不在做实际工作，这段时间不计入总
	// 时长预算（与 waiting_input 暂停同理）。BlockedAtMs 记录最近一次进入
	// blocked 的时刻；恢复时把本次 blocked 时长累加进 AccumulatedBlockedMs，
	// 由 questDurationElapsedMs 从墙钟 elapsed 中扣除。
	BlockedAtMs          int64 `json:"blocked_at_ms,omitempty"`
	AccumulatedBlockedMs int64 `json:"accumulated_blocked_ms,omitempty"`

	// Quest 级预算覆盖。0 表示跟随全局 config。
	MaxTurnsPerPhaseOverride      int   `json:"max_turns_per_phase_override,omitempty"`
	MaxDurationPerQuestMsOverride int64 `json:"max_duration_per_quest_ms_override,omitempty"`

	// 结果
	WarriorSummary        string              `json:"warrior_summary,omitempty"` // 执行者的 phase done summary（HOTL Feed 兜底来源）
	FinalVerdict          model.QuestVerdict  `json:"final_verdict,omitempty"`
	FinalComment          string              `json:"final_comment,omitempty"`
	ImpactSummary         model.ImpactSummary `json:"impact_summary,omitempty"`           // HOTL: agent 主动声明的影响
	FinalizedBy           string              `json:"finalized_by,omitempty"`             // user / policy / automation
	AutoPassedByPolicy    string              `json:"auto_passed_by_policy,omitempty"`    // policy name for HOTL auto-pass (review phase)
	AutoCompletedByPolicy string              `json:"auto_completed_by_policy,omitempty"` // policy name for HOTL auto-complete (execute phase)
	PolicyDecisionID      string              `json:"policy_decision_id,omitempty"`       // linked policy.decision event
	Outputs               []QuestArtifact     `json:"outputs,omitempty"`                  // 委托产出物（剑士声明的交付物）
	Reviews               []ReviewRecord      `json:"-"`                                  // prompt 构建时临时注入，不写入 meta.json
	Applied               bool                `json:"applied,omitempty"`
	ApplyStatus           model.ApplyStatus   `json:"apply_status,omitempty"`
	ApplyError            string              `json:"apply_error,omitempty"`
	ApplyFailedAtMs       int64               `json:"apply_failed_at_ms,omitempty"`
	BlockedReason         string              `json:"blocked_reason,omitempty"`      // 阻塞原因（超预算等）
	BlockedReasonCode     string              `json:"blocked_reason_code,omitempty"` // 稳定阻塞原因代码（给 policy / filter 用）
	BlockedCategory       string              `json:"blocked_category,omitempty"`    // 错误性质分类（transient/auth/configuration，恢复策略二维匹配用）
	FailureAttribution    *FailureAttribution `json:"failure_attribution,omitempty"` // 最新结构化失败归因
	MageScore             int                 `json:"mage_score,omitempty"`          // 法师给的质量评分（1-10，0=未评分）

	// Diff 摘要（进入 user_review 时缓存，避免重复计算）
	DiffStat             string `json:"diff_stat,omitempty"`
	DiffChangedFiles     int    `json:"diff_changed_files,omitempty"`
	DiffAdditions        int    `json:"diff_additions,omitempty"`
	DiffDeletions        int    `json:"diff_deletions,omitempty"`
	WorkspaceCleaned     bool   `json:"workspace_cleaned,omitempty"`
	WorkspaceCleanedAtMs int64  `json:"workspace_cleaned_at_ms,omitempty"`

	// 元信息
	CreatedBy     string `json:"created_by"` // user / automation:xxx
	Official      bool   `json:"official,omitempty"`
	CreatedAtMs   int64  `json:"created_at_ms"`
	UpdatedAtMs   int64  `json:"updated_at_ms,omitempty"`
	StartedAtMs   int64  `json:"started_at_ms,omitempty"`
	CompletedAtMs int64  `json:"completed_at_ms,omitempty"`

	// Prompt 版本快照（quest 创建时记录，用于归因排查）
	PromptVersion     string            `json:"prompt_version,omitempty"`      // prompt 版本短标识（base hash 前 8 位）
	PromptOverrides   []string          `json:"prompt_overrides,omitempty"`    // 被用户目录覆盖的模板文件列表
	PromptOverrideMap map[string]string `json:"prompt_override_map,omitempty"` // 覆盖模板的内容哈希（细粒度对比用）
}

type FailureAttribution struct {
	Stage       string         `json:"stage"`
	Reason      string         `json:"reason"`
	Category    string         `json:"category,omitempty"`
	Message     string         `json:"message,omitempty"`
	Recoverable bool           `json:"recoverable"`
	Actions     []string       `json:"actions,omitempty"`
	Details     map[string]any `json:"details,omitempty"`
}

type WaitingInputState struct {
	QuestionID         string    `json:"question_id"`
	QuestionText       string    `json:"question_text"`
	PhaseIdx           int       `json:"phase_idx"`
	SessionID          string    `json:"session_id,omitempty"`
	PhaseType          string    `json:"phase_type,omitempty"`
	Asker              string    `json:"asker,omitempty"`
	TimeoutMs          int64     `json:"timeout_ms,omitempty"`
	TimeoutAction      string    `json:"timeout_action,omitempty"`
	ResumePhaseIdx     int       `json:"resume_phase_idx,omitempty"`
	LastCommentSeq     int       `json:"last_comment_seq,omitempty"`
	LastAnswerID       string    `json:"last_answer_id,omitempty"`
	ProcessedAnswerIDs []string  `json:"processed_answer_ids,omitempty"`
	AskedAt            time.Time `json:"asked_at"`
	// AccumulatedPausedMs 是本次 quest 累计因 waiting_input 暂停的毫秒数。
	// 每次 answer 恢复时把"本次等待时长"累加进来，供 questDurationElapsedMs
	// 直接读取，避免依赖 events.jsonl 扫描时序（answer 落盘 vs duration 判断
	// 的竞态会导致等待时间未被扣减，quest 被误判 duration_exceeded）。
	AccumulatedPausedMs int64 `json:"accumulated_paused_ms,omitempty"`
}

// Source 返回该 quest 的来源。
func (q *QuestMeta) Source() model.QuestSource {
	if strings.HasPrefix(q.CreatedBy, model.QuestSourcePrefix) {
		return model.SourceAutomation
	}
	return model.SourceUser
}

// AutomationID 返回创建该 quest 的 automation ID；如果不是 automation 创建的，返回空字符串。
func (q *QuestMeta) AutomationID() string {
	if strings.HasPrefix(q.CreatedBy, model.QuestSourcePrefix) {
		return strings.TrimPrefix(q.CreatedBy, model.QuestSourcePrefix)
	}
	return ""
}

// ===== Phase 兼容层（双写双读） =====
//
// Phase 2: 用 Phases 列表替代硬编码的 WarriorID/MageID。
// 保持双写双读，确保向后兼容：
//   - 加载时：如果 Phases 为空，从 WarriorID/MageID 反推
//   - 保存时：从 Phases 同步回写 WarriorID/MageID

// EnsurePhases 确保 Phases 字段已初始化。
// 如果 Phases 为空，从 WarriorID/MageId 反推构建两阶段列表。
func (q *QuestMeta) EnsurePhases() {
	q.EnsurePipelineFields()
	if len(q.Phases) > 0 {
		return
	}
	q.Phases = instantiatePhaseTasks(q.PipelineDef, q.WarriorID, q.MageID, q.ExecuteAgentID, q.ReviewAgentID)
	// 如果 quest 已经启动/完成，同步状态到 phases
	// running → phase 0 running, phase 1 pending
	// reviewing → phase 0 done, phase 1 running
	// user_review/success/failed → phase 0 done, phase 1 done
	switch q.Status {
	case model.QuestStatusRunning, model.QuestStatusWaitingInput:
		q.Phases[0].Status = model.PhaseRunning
	case model.QuestStatusReviewing:
		q.Phases[0].Status = model.PhaseDone
		q.Phases[1].Status = model.PhaseRunning
	case model.QuestStatusUserReview, model.QuestStatusSuccess, model.QuestStatusFailed, model.QuestStatusCancelled:
		q.Phases[0].Status = model.PhaseDone
		q.Phases[1].Status = model.PhaseDone
	case model.QuestStatusBlocked:
		// blocked 可能发生在任何阶段，保守都设为 running
		q.Phases[0].Status = model.PhaseDone
		q.Phases[1].Status = model.PhaseRunning
	}
}

func (q *QuestMeta) EnsurePipelineFields() {
	if q.PipelineName == "" {
		q.PipelineName = DefaultPipelineName
	}
	if q.PipelineVersion == 0 {
		q.PipelineVersion = DefaultPipelineVersion
	}
	if q.PhaseCount == 0 {
		if len(q.PipelineDef) > 0 {
			q.PhaseCount = len(q.PipelineDef)
		} else if len(q.Phases) > 0 {
			q.PhaseCount = len(q.Phases)
		} else {
			q.PhaseCount = len(defaultPhaseTasks("", ""))
		}
	}
	if len(q.PipelineDef) == 0 {
		q.PipelineDef = defaultPhaseTasks("", "")
	}
	if q.PipelineDefHash == "" {
		q.PipelineDefHash = phaseDefHash(q.PipelineDef)
	}
	if q.Status == model.QuestStatusRunning || q.Status == model.QuestStatusWaitingInput || q.Status == model.QuestStatusReviewing {
		if q.CurrentPhase == 0 {
			q.CurrentPhase = q.inferCurrentPhaseIdx()
		}
	}
}

func (q *QuestMeta) ValidatePipelineState() error {
	q.EnsurePipelineFields()
	if q.PhaseCount <= 0 {
		return fmt.Errorf("phase_count must be positive")
	}
	if len(q.Phases) > 0 && q.PhaseCount != len(q.Phases) {
		return fmt.Errorf("phase_count %d != phases length %d", q.PhaseCount, len(q.Phases))
	}
	if q.CurrentPhase >= q.PhaseCount {
		return fmt.Errorf("current_phase_idx %d out of range for phase_count %d", q.CurrentPhase, q.PhaseCount)
	}
	if len(q.PipelineDef) > 0 {
		if q.PhaseCount != len(q.PipelineDef) {
			return fmt.Errorf("phase_count %d != pipeline_def length %d", q.PhaseCount, len(q.PipelineDef))
		}
		if q.PipelineDefHash != phaseDefHash(q.PipelineDef) {
			return fmt.Errorf("pipeline_def_hash mismatch")
		}
	}
	return nil
}

// SyncPhaseIDs 从 Phases 同步回 WarriorID/MageID（双写兼容）。
// 保存前调用，确保老字段仍然有效。
func (q *QuestMeta) SyncPhaseIDs() {
	if len(q.Phases) == 0 {
		return
	}
	if len(q.Phases) > 0 && q.Phases[0].AdventurerID != "" {
		q.WarriorID = q.Phases[0].AdventurerID
	}
	if len(q.Phases) > 0 && q.Phases[0].AgentID != "" {
		q.ExecuteAgentID = q.Phases[0].AgentID
	}
	if len(q.Phases) > 1 && q.Phases[1].AdventurerID != "" {
		q.MageID = q.Phases[1].AdventurerID
	}
	if len(q.Phases) > 1 && q.Phases[1].AgentID != "" {
		q.ReviewAgentID = q.Phases[1].AgentID
	}
}

// PhaseForIdx 返回指定索引的阶段任务指针。
func (q *QuestMeta) PhaseForIdx(idx int) *PhaseTask {
	q.EnsurePhases()
	if idx < 0 || idx >= len(q.Phases) {
		return nil
	}
	return &q.Phases[idx]
}

// CurrentPhaseIdx 根据 quest 状态推断当前阶段索引。
// 返回 -1 表示不在任何 agent 阶段。
func (q *QuestMeta) CurrentPhaseIdx() int {
	q.EnsurePipelineFields()
	if q.CurrentPhase != 0 || q.Status == model.QuestStatusRunning || q.Status == model.QuestStatusWaitingInput {
		return q.CurrentPhase
	}
	return q.inferCurrentPhaseIdx()
}

func (q *QuestMeta) inferCurrentPhaseIdx() int {
	switch q.Status {
	case model.QuestStatusRunning, model.QuestStatusWaitingInput:
		return 0
	case model.QuestStatusReviewing:
		return 1
	default:
		return -1
	}
}

func defaultPhaseTasks(warriorID, mageID string) []PhaseTask {
	return []PhaseTask{
		{
			PhaseIdx:     0,
			Role:         "warrior",
			Class:        model.ClassWarrior,
			Goal:         "按照用户需求完成任务，交付可验证的产出",
			AdventurerID: warriorID,
			ReworkTo:     0,
			Status:       model.PhasePending,
		},
		{
			PhaseIdx:     1,
			Role:         "mage",
			Class:        model.ClassMage,
			Goal:         "评审剑士的产出，给出通过/返工/拒绝结论",
			AdventurerID: mageID,
			ReworkTo:     0,
			Status:       model.PhasePending,
		},
	}
}

func instantiatePhaseTasks(defs []PhaseTask, warriorID, mageID, executeAgentID, reviewAgentID string) []PhaseTask {
	if len(defs) == 0 {
		defs = defaultPhaseTasks("", "")
	}
	out := make([]PhaseTask, len(defs))
	for i, def := range defs {
		out[i] = PhaseTask{
			PhaseIdx: def.PhaseIdx,
			Role:     def.Role,
			Class:    def.Class,
			Goal:     def.Goal,
			AgentID:  def.AgentID,
			ReworkTo: def.ReworkTo,
			Status:   model.PhasePending,
		}
		switch i {
		case 0:
			out[i].AdventurerID = warriorID
			if out[i].AgentID == "" {
				out[i].AgentID = executeAgentID
			}
		case 1:
			out[i].AdventurerID = mageID
			if out[i].AgentID == "" {
				out[i].AgentID = reviewAgentID
			}
		}
	}
	return out
}

func phaseDefHash(phases []PhaseTask) string {
	type phaseDef struct {
		PhaseIdx int                   `json:"phase_idx"`
		Role     string                `json:"role"`
		Class    model.AdventurerClass `json:"class"`
		Goal     string                `json:"goal"`
		AgentID  string                `json:"agent_id,omitempty"`
	}
	defs := make([]phaseDef, 0, len(phases))
	for _, phase := range phases {
		defs = append(defs, phaseDef{
			PhaseIdx: phase.PhaseIdx,
			Role:     phase.Role,
			Class:    phase.Class,
			Goal:     phase.Goal,
			AgentID:  phase.AgentID,
		})
	}
	b, _ := json.Marshal(defs)
	sum := sha256.Sum256(b)
	return fmt.Sprintf("%x", sum[:])
}

// IsOfficial 通过反查 automation 配置来判断 quest 是否来自官方自动化。
// 如果找不到对应 automation 配置，回退到 quest 自身的 Official 字段（兼容旧数据）。
func (r *Root) IsQuestOfficial(q *QuestMeta) bool {
	if q == nil {
		return false
	}
	if autoID := q.AutomationID(); autoID != "" {
		if cfg, err := r.GetAutomation(autoID); err == nil && cfg != nil {
			return cfg.IsOfficial()
		}
	}
	// fallback: old quests may have the field stored directly
	return q.Official
}

// SetSource 设置 created_by 字段；SourceUser 时写入 "user"，SourceAutomation 时写入 "automation:<id>"。
func (q *QuestMeta) SetSource(s model.QuestSource, id string) {
	switch s {
	case model.SourceAutomation:
		if id == "" {
			q.CreatedBy = model.QuestSourcePrefix
		} else {
			q.CreatedBy = model.QuestSourcePrefix + id
		}
	default:
		if id == "" {
			q.CreatedBy = string(model.SourceUser)
		} else {
			q.CreatedBy = id
		}
	}
}

// QuestSessionRow 对应 sessions/<sid>.jsonl 的一行
type QuestSessionRow struct {
	Seq        int            `json:"seq"`
	Timestamp  int64          `json:"ts"`
	Kind       string         `json:"kind"` // message|tool_call|tool_result|turn_summary|phase|review|error
	SessionID  string         `json:"sid,omitempty"`
	Role       string         `json:"role,omitempty"`    // message: system/user/assistant/tool
	Content    string         `json:"content,omitempty"` // message: 文本
	ToolName   string         `json:"tool_name,omitempty"`
	ToolID     string         `json:"tool_id,omitempty"`
	ToolArgs   string         `json:"tool_args,omitempty"` // json 字符串
	ToolResult string         `json:"tool_result,omitempty"`
	Error      string         `json:"error,omitempty"`
	Phase      int            `json:"phase,omitempty"`
	Status     string         `json:"status,omitempty"`
	Meta       map[string]any `json:"meta,omitempty"` // 额外信息（token 数、duration 等）
}

type PhaseReviewArtifact struct {
	Kind                 string
	PhaseName            string
	SessionID            string
	Assistant            string
	PlatformToolEvidence string
	Outputs              []QuestArtifact
	OutputChecks         map[string]ArtifactVerification
	Evidence             PhaseEvidence
}

type ArtifactVerification struct {
	Verified bool   `json:"verified"`
	Status   string `json:"status"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message,omitempty"`
}

type PhaseEvidence struct {
	AutoCollected      string `json:"auto_collected,omitempty"`
	PlatformMechanisms string `json:"platform_mechanisms,omitempty"`
	NativeToolCalls    string `json:"native_tool_calls,omitempty"`
}

// QuestEventRow 对应 events.jsonl 的一行（SSE 重放）
type QuestEventRow struct {
	ID        int64       `json:"id,omitempty"`
	Timestamp int64       `json:"ts"`
	Type      string      `json:"type"`
	QuestID   string      `json:"qid,omitempty"`
	SessionID string      `json:"sid,omitempty"`
	Payload   interface{} `json:"payload,omitempty"`
}

type GlobalEventRow struct {
	GlobalID  int64       `json:"global_id,omitempty"`
	EventID   int64       `json:"event_id,omitempty"`
	Timestamp int64       `json:"ts"`
	Type      string      `json:"type"`
	QuestID   string      `json:"qid,omitempty"`
	SessionID string      `json:"sid,omitempty"`
	Payload   interface{} `json:"payload,omitempty"`
}

type DesignDoc struct {
	QuestID             string   `json:"quest_id"`
	SchemaVersion       string   `json:"schema_version"`
	OriginalQuery       string   `json:"original_query"`
	Summary             string   `json:"summary"`
	ReviewComment       string   `json:"review_comment,omitempty"`
	Goals               []string `json:"goals,omitempty"`
	NonGoals            []string `json:"non_goals,omitempty"`
	ImplementationSteps []string `json:"implementation_steps,omitempty"`
	Risks               []string `json:"risks,omitempty"`
	AcceptanceCriteria  []string `json:"acceptance_criteria,omitempty"`
	ExecutePrompt       string   `json:"execute_prompt"`
	CreatedAtMs         int64    `json:"created_at_ms"`
}

// ==================== 目录 & 读写 ====================

type QuestStore struct {
	root *Root
}

func NewQuestStore(root *Root) *QuestStore { return &QuestStore{root: root} }

func (qs *QuestStore) dir(qid string) string {
	return qs.root.Sub(SubdirQuests, qid)
}
func (qs *QuestStore) sessionsDir(qid string) string {
	return filepath.Join(qs.dir(qid), "sessions")
}

// WorkDir 返回 per-quest 隔离工作目录（自动创建）。
// 返回 (path, error)，MkdirAll 失败时透传错误。
func (qs *QuestStore) WorkDir(qid string) (string, error) {
	wd := filepath.Join(qs.dir(qid), "work")
	if err := os.MkdirAll(wd, 0o755); err != nil {
		return "", err
	}
	return wd, nil
}

var questMu sync.Mutex // 保护 qid 并发生成

// NewQuestID 生成 quest id：qst_<日期YYMMDD>_<纳秒末4位>
func (qs *QuestStore) NewQuestID() (string, string) {
	questMu.Lock()
	defer questMu.Unlock()
	n := time.Now()
	day := fmt.Sprintf("%02d%02d%02d", n.Year()%100, int(n.Month()), n.Day())
	sfx := strconv.FormatInt(n.UnixNano()%10000, 10)
	short := day + sfx
	return "qst_" + short, short
}

// CreateQuest 新建委托 & 所有子目录
func (qs *QuestStore) CreateQuest(q *QuestMeta) error {
	qid := qs.dir(q.ID)
	if err := os.MkdirAll(filepath.Join(qid, "sessions"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(qid, "work"), 0o755); err != nil {
		return err
	}
	now := model.NowMs()
	q.CreatedAtMs = now
	q.UpdatedAtMs = now
	// 确保 phases 已初始化
	q.EnsurePhases()
	if err := q.ValidatePipelineState(); err != nil {
		return err
	}
	// 双写兼容
	q.SyncPhaseIDs()
	return WriteJSON(filepath.Join(qid, "meta.json"), q)
}

// LoadQuest 读 meta.json
func (qs *QuestStore) LoadQuest(qid string) (*QuestMeta, error) {
	qid = qs.resolveQuestID(qid)
	m, err := ReadJSON[QuestMeta](filepath.Join(qs.dir(qid), "meta.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("委托不存在: %s", qid)
		}
		return nil, err
	}
	// 统一展开 base_working_dir（~ 和 $VAR）。
	// 单点收敛：所有读取者（Prepare/apply/diff/cleanup/automation）拿到的都是绝对路径，
	// 无需各自展开；存储保留用户原始输入不丢失。
	m.BaseWorkingDir = ExpandDir(m.BaseWorkingDir)
	// 兼容老数据：如果 phases 为空，从 warrior_id/mage_id 反推
	m.EnsurePhases()
	if err := m.ValidatePipelineState(); err != nil {
		return nil, fmt.Errorf("委托 pipeline 状态不一致: %w", err)
	}
	return m, nil
}

func (qs *QuestStore) resolveQuestID(qid string) string {
	qid = strings.TrimSpace(qid)
	if qid == "" || strings.HasPrefix(qid, "qst_") {
		return qid
	}
	candidate := "qst_" + qid
	if _, err := os.Stat(filepath.Join(qs.dir(candidate), "meta.json")); err == nil {
		return candidate
	}
	return qid
}

// SaveQuest 覆盖写 meta.json
func (qs *QuestStore) SaveQuest(q *QuestMeta) error {
	if q.CreatedAtMs == 0 {
		q.CreatedAtMs = model.NowMs()
	}
	previous, _ := qs.loadQuestRaw(q.ID)
	q.UpdatedAtMs = model.NowMs()
	// 双写兼容：从 phases 同步回 warrior_id/mage_id
	q.EnsurePipelineFields()
	if err := q.ValidatePipelineState(); err != nil {
		return err
	}
	q.SyncPhaseIDs()
	projections, err := qs.prepareThreadStateTransitionProjection(previous, q)
	if err != nil {
		return err
	}
	if err := WriteJSON(filepath.Join(qs.dir(q.ID), "meta.json"), q); err != nil {
		return err
	}
	if err := qs.appendThreadStateProjections(q, projections); err != nil {
		return err
	}
	if err := qs.appendFanoutLeafDoneNotification(q); err != nil {
		return err
	}
	if q.GroupID != "" && isTerminalStatus(q.Status) && (previous == nil || !isTerminalStatus(previous.Status)) {
		return qs.AppendFanoutSummarySnapshot(q.GroupID)
	}
	return nil
}

func (qs *QuestStore) loadQuestRaw(qid string) (*QuestMeta, error) {
	qid = qs.resolveQuestID(qid)
	m, err := ReadJSON[QuestMeta](filepath.Join(qs.dir(qid), "meta.json"))
	if err != nil {
		return nil, err
	}
	return m, nil
}

// ListQuests 读所有委托，按创建时间倒序
func (qs *QuestStore) ListQuests() ([]*QuestMeta, error) {
	dir := qs.root.Sub(SubdirQuests)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []*QuestMeta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		m, err := qs.LoadQuest(e.Name())
		if err != nil {
			fmt.Fprintf(os.Stderr, "[warn] skip bad quest %s: %v\n", e.Name(), err)
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAtMs > out[j].CreatedAtMs })
	return out, nil
}

// FindFanoutRoot finds the canonical root quest for a fanout group.
// The root is the quest with matching GroupID and empty FanoutLeafID.
// Returns ("", false) if no root found.
func (qs *QuestStore) FindFanoutRoot(groupID string) (*QuestMeta, bool) {
	if strings.TrimSpace(groupID) == "" {
		return nil, false
	}
	quests, err := qs.ListQuests()
	if err != nil {
		return nil, false
	}
	for _, q := range quests {
		if q.GroupID == groupID && q.FanoutLeafID == "" {
			return q, true
		}
	}
	return nil, false
}

// ListFanoutGroupLeaves returns all leaf quests in a fanout group.
func (qs *QuestStore) ListFanoutGroupLeaves(groupID string) []*QuestMeta {
	if strings.TrimSpace(groupID) == "" {
		return nil
	}
	quests, err := qs.ListQuests()
	if err != nil {
		return nil
	}
	var leaves []*QuestMeta
	for _, q := range quests {
		if q.GroupID == groupID && q.FanoutLeafID != "" {
			leaves = append(leaves, q)
		}
	}
	return leaves
}

type WorkspaceCleanupCandidate struct {
	QuestID       string            `json:"quest_id"`
	Status        model.QuestStatus `json:"status"`
	WorkspacePath string            `json:"workspace_path"`
	CompletedAtMs int64             `json:"completed_at_ms"`
	Reason        string            `json:"reason"`
}

func (qs *QuestStore) CleanupExpiredWorkspaces(retentionDays int, includeFailed bool, nowMs int64, dryRun bool) ([]WorkspaceCleanupCandidate, error) {
	if retentionDays < 0 {
		return nil, fmt.Errorf("retentionDays must be >= 0")
	}
	if nowMs == 0 {
		nowMs = NowMs()
	}
	items, err := qs.ListQuests()
	if err != nil {
		return nil, err
	}
	cutoff := nowMs - int64(retentionDays)*24*60*60*1000
	var out []WorkspaceCleanupCandidate
	for _, q := range items {
		if q.WorkspaceCleaned || q.WorkspacePath == "" || q.CompletedAtMs == 0 || q.CompletedAtMs > cutoff {
			continue
		}
		reason := ""
		switch q.Status {
		case model.QuestStatusSuccess:
			reason = "success_retention_expired"
		case model.QuestStatusFailed, model.QuestStatusCancelled:
			if !includeFailed {
				continue
			}
			reason = "terminal_retention_expired"
		default:
			continue
		}
		// 安全护栏：只清理 gloop 自己创建的隔离工作目录（位于数据目录
		// <dataDir>/workspace/quests/ 之下）。任何指向该区域之外的 workspace_path
		// 都视为用户真实工作区，绝不自动删除——历史上 quest 的 workspace_path 曾被
		// 记录为用户 cwd（如源码仓库目录），过期清理曾因此误删用户代码。
		if !qs.isManagedWorkDir(q.WorkspacePath) {
			fmt.Fprintf(os.Stderr, "[warn] 跳过清理 %s：workspace_path %q 不在 gloop 管理的隔离工作目录内，疑似用户工作区\n", q.ID, q.WorkspacePath)
			continue
		}
		candidate := WorkspaceCleanupCandidate{
			QuestID:       q.ID,
			Status:        q.Status,
			WorkspacePath: q.WorkspacePath,
			CompletedAtMs: q.CompletedAtMs,
			Reason:        reason,
		}
		out = append(out, candidate)
		if dryRun {
			continue
		}
		if err := os.RemoveAll(q.WorkspacePath); err != nil {
			return out, fmt.Errorf("cleanup workspace %s: %w", q.ID, err)
		}
		q.WorkspaceCleaned = true
		q.WorkspaceCleanedAtMs = nowMs
		if err := qs.SaveQuest(q); err != nil {
			return out, fmt.Errorf("save cleanup meta %s: %w", q.ID, err)
		}
	}
	return out, nil
}

// isManagedWorkDir 判断 path 是否严格位于 gloop 数据目录的 quests 隔离区之内
// （<dataDir>/workspace/quests/...）。只有这类路径才是 gloop 自己创建、可安全
// 自动删除的隔离工作目录；用户的真实工作区（cwd）绝不应被自动清理。
func (qs *QuestStore) isManagedWorkDir(path string) bool {
	if path == "" {
		return false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	managedRoot, err := filepath.Abs(qs.root.Sub(SubdirQuests))
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(managedRoot, abs)
	if err != nil {
		return false
	}
	// rel 不为 "." 且不以 ".." 开头 → 严格位于 managedRoot 之下
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// SavePlan 写 plan.json（阶段任务拆解）
func (qs *QuestStore) SavePlan(qid string, plan []PhaseTask) error {
	return WriteJSON(filepath.Join(qs.dir(qid), "plan.json"), plan)
}

func (qs *QuestStore) LoadPlan(qid string) ([]PhaseTask, error) {
	p, err := ReadJSON[[]PhaseTask](filepath.Join(qs.dir(qid), "plan.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return *p, nil
}

// SaveChecks 写 checks.json（支持任意类型）。
func (qs *QuestStore) SaveChecks(qid string, checks any) error {
	return WriteJSON(filepath.Join(qs.dir(qid), "checks.json"), checks)
}

// SaveChecksT 泛型版本：以具体类型写 checks.json，避免 any 擦除。
func SaveChecksT[T any](qs *QuestStore, qid string, checks *T) error {
	return WriteJSON(filepath.Join(qs.dir(qid), "checks.json"), checks)
}

// LoadChecks 读 checks.json 到 dst（dst 必须是指针）。
func (qs *QuestStore) LoadChecks(qid string, dst any) error {
	raw, err := ReadJSON[json.RawMessage](filepath.Join(qs.dir(qid), "checks.json"))
	if err != nil {
		return err
	}
	return json.Unmarshal(*raw, dst)
}

// LoadChecksT 泛型版本：以具体类型读 checks.json，返回 *T。
func LoadChecksT[T any](qs *QuestStore, qid string) (*T, error) {
	return ReadJSON[T](filepath.Join(qs.dir(qid), "checks.json"))
}

func (qs *QuestStore) SaveDesignDoc(qid string, doc *DesignDoc) error {
	if doc.CreatedAtMs == 0 {
		doc.CreatedAtMs = model.NowMs()
	}
	return WriteJSON(filepath.Join(qs.dir(qid), "design.json"), doc)
}

func (qs *QuestStore) LoadDesignDoc(qid string) (*DesignDoc, error) {
	return ReadJSON[DesignDoc](filepath.Join(qs.dir(qid), "design.json"))
}

// AppendSessionRow 往 sessions/<sid>.jsonl 追加一行
func (qs *QuestStore) AppendSessionRow(qid, sid string, row *QuestSessionRow) error {
	if row.Timestamp == 0 {
		row.Timestamp = model.NowMs()
	}
	if row.SessionID == "" {
		row.SessionID = sid
	}
	return AppendJSONL(filepath.Join(qs.sessionsDir(qid), sid+".jsonl"), row)
}

// ReadSessionRows 读 session 最后 n 行；n<=0 读全部
func (qs *QuestStore) ReadSessionRows(qid, sid string, n int) ([]QuestSessionRow, error) {
	return ReadJSONL[QuestSessionRow](filepath.Join(qs.sessionsDir(qid), sid+".jsonl"), n)
}

// ReadContextPackRows returns context-pack trace rows for a session. These rows
// contain summaries, not full prompt text, so they are suitable for debug views.
func (qs *QuestStore) ReadContextPackRows(qid, sid string, n int) ([]QuestSessionRow, error) {
	rows, err := qs.ReadSessionRows(qid, sid, n)
	if err != nil {
		return nil, err
	}
	out := make([]QuestSessionRow, 0, len(rows))
	for _, row := range rows {
		if row.Kind == "context_pack" {
			out = append(out, row)
		}
	}
	return out, nil
}

// LastAssistantMessage 返回该 session 最后一条 assistant 消息（用作下阶段输入）
func (qs *QuestStore) LastAssistantMessage(qid, sid string) (string, error) {
	rows, err := qs.ReadSessionRows(qid, sid, 200)
	if err != nil {
		return "", err
	}
	for i := len(rows) - 1; i >= 0; i-- {
		if rows[i].Kind == "message" && rows[i].Role == "assistant" && rows[i].Content != "" {
			return rows[i].Content, nil
		}
	}
	return "", nil
}

// PhaseArtifactForReview returns the assistant-facing phase artifact plus
// platform tool-call evidence that a reviewer needs to verify phase contracts.
// PhaseArtifactForReview returns the assistant-facing phase artifact plus
// platform tool-call evidence that a reviewer needs to verify phase contracts.
//
// artifact 选取优先级：
//  1. phase_checkpoint / review_quest 的 CLI signal summary（最准确，结构化交付）
//  2. 最后一条非空 assistant message（兼容旧数据 / 未正常结束阶段的情况）
func (qs *QuestStore) PhaseArtifactForReview(qid, sid string) (PhaseReviewArtifact, error) {
	rows, err := qs.ReadSessionRows(qid, sid, 200)
	if err != nil {
		return PhaseReviewArtifact{}, err
	}
	q, _ := qs.LoadQuest(qid)

	// 优先级 1：找 CLI signal 记录的 phase summary（最可靠）
	var assistant string
	for i := len(rows) - 1; i >= 0; i-- {
		if rows[i].Kind == "tool_result" && rows[i].ToolName == "gloop_cli_signal" {
			if summary := extractPhaseSummaryFromSignal(rows[i].Content); summary != "" {
				assistant = summary
				break
			}
		}
	}
	// 优先级 2：回退到最后一条非空 assistant message
	if assistant == "" {
		for i := len(rows) - 1; i >= 0; i-- {
			if rows[i].Kind == "message" && rows[i].Role == "assistant" && rows[i].Content != "" {
				assistant = rows[i].Content
				break
			}
		}
	}
	var platformMechanisms []string
	var nativeToolCalls []string
	for _, row := range rows {
		switch row.Kind {
		case "tool_call":
			if row.ToolName != "" {
				platformMechanisms = append(platformMechanisms, fmt.Sprintf("- tool_call %s args=%s", row.ToolName, row.ToolArgs))
			}
		case "tool_result":
			if row.ToolName != "" {
				platformMechanisms = append(platformMechanisms, fmt.Sprintf("- tool_result %s result=%s", row.ToolName, row.Content))
			}
		case "native_tool_call_observed":
			if row.ToolName != "" {
				status, _ := row.Meta["status"].(string)
				result, _ := row.Meta["result"].(string)
				if result == "" {
					result = "result_unavailable"
				}
				if status != "" {
					nativeToolCalls = append(nativeToolCalls, fmt.Sprintf("- native_tool_observed %s status=%s args=%s result=%s", row.ToolName, status, row.ToolArgs, result))
				} else {
					nativeToolCalls = append(nativeToolCalls, fmt.Sprintf("- native_tool_observed %s args=%s result=%s", row.ToolName, row.ToolArgs, result))
				}
			}
		}
	}
	evidence := PhaseEvidence{
		PlatformMechanisms: strings.Join(platformMechanisms, "\n"),
		NativeToolCalls:    strings.Join(nativeToolCalls, "\n"),
	}
	outputs, checks := qs.outputsForReview(qid, q)
	kind, phaseName := phaseArtifactMetadata(q, sid)
	return PhaseReviewArtifact{
		Kind:                 kind,
		PhaseName:            phaseName,
		SessionID:            sid,
		Assistant:            assistant,
		PlatformToolEvidence: evidence.LegacyString(),
		Outputs:              outputs,
		OutputChecks:         checks,
		Evidence:             evidence,
	}, nil
}

func (qs *QuestStore) FindPhaseArtifactByKind(qid, kind string) (PhaseReviewArtifact, bool, error) {
	if kind == "" {
		return PhaseReviewArtifact{}, false, nil
	}
	q, err := qs.LoadQuest(qid)
	if err != nil {
		return PhaseReviewArtifact{}, false, err
	}
	q.EnsurePipelineFields()
	for idx := len(q.PipelineDef) - 1; idx >= 0; idx-- {
		def := q.PipelineDef[idx]
		sid := fmt.Sprintf("%s_%d", phaseSessionPrefix(def), q.ReworkCount)
		artifact, err := qs.PhaseArtifactForReview(qid, sid)
		if err != nil {
			continue
		}
		if artifact.Kind == kind && strings.TrimSpace(artifact.Assistant) != "" {
			return artifact, true, nil
		}
	}
	return PhaseReviewArtifact{}, false, nil
}

func phaseArtifactMetadata(q *QuestMeta, sid string) (kind string, phaseName string) {
	if q == nil {
		return "", ""
	}
	q.EnsurePipelineFields()
	for _, def := range q.PipelineDef {
		prefix := phaseSessionPrefix(def)
		if sid == fmt.Sprintf("%s_%d", prefix, q.ReworkCount) || strings.HasPrefix(sid, prefix+"_") {
			phaseName = def.Name
			if phaseName == "" {
				phaseName = def.Role
			}
			if phaseName == "warrior_design" {
				kind = "design_plan"
			}
			return kind, phaseName
		}
	}
	return "", ""
}

func phaseSessionPrefix(def PhaseTask) string {
	if def.Name != "" && def.Name != "warrior" && def.Name != "mage" && def.Name != "mage_review" {
		return def.Name
	}
	switch def.Role {
	case "mage", "review":
		if def.PhaseIdx <= 1 {
			return "mage"
		}
		return fmt.Sprintf("mage_%d", def.PhaseIdx)
	case "warrior", "execute":
		if def.PhaseIdx == 0 {
			return "warrior"
		}
		return fmt.Sprintf("warrior_%d", def.PhaseIdx)
	default:
		return fmt.Sprintf("phase_%d", def.PhaseIdx)
	}
}

func (e PhaseEvidence) LegacyString() string {
	var sections []string
	if strings.TrimSpace(e.AutoCollected) != "" {
		sections = append(sections, e.AutoCollected)
	}
	if strings.TrimSpace(e.PlatformMechanisms) != "" {
		sections = append(sections, e.PlatformMechanisms)
	}
	if strings.TrimSpace(e.NativeToolCalls) != "" {
		sections = append(sections, e.NativeToolCalls)
	}
	return strings.Join(sections, "\n")
}

func (qs *QuestStore) outputsForReview(qid string, q *QuestMeta) ([]QuestArtifact, map[string]ArtifactVerification) {
	if q == nil || len(q.Outputs) == 0 {
		return nil, nil
	}
	outputs := make([]QuestArtifact, 0, len(q.Outputs))
	checks := map[string]ArtifactVerification{}
	for _, out := range q.Outputs {
		if out.Source != "warrior_phase" {
			continue
		}
		outputs = append(outputs, out)
		checks[out.ID] = qs.verifyOutputArtifact(qid, out)
	}
	if len(outputs) == 0 {
		return nil, nil
	}
	return outputs, checks
}

func (qs *QuestStore) verifyOutputArtifact(qid string, art QuestArtifact) ArtifactVerification {
	if strings.TrimSpace(art.StoragePath) == "" {
		return ArtifactVerification{Status: "unverified", Message: "no storage path or URL declared"}
	}
	if strings.HasPrefix(art.StoragePath, "http://") || strings.HasPrefix(art.StoragePath, "https://") {
		return ArtifactVerification{Status: "external", Path: art.StoragePath, Message: "external artifact; existence not checked locally"}
	}
	path := art.StoragePath
	if !filepath.IsAbs(path) {
		path = qs.ArtifactPath(qid, art)
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ArtifactVerification{Status: "missing", Path: path, Message: "file does not exist"}
		}
		return ArtifactVerification{Status: "error", Path: path, Message: err.Error()}
	}
	return ArtifactVerification{Verified: true, Status: "verified", Path: path, Message: "file exists"}
}

// extractPhaseSummaryFromSignal 从 gloop_cli_signal 的 JSON 结果中提取 phase_comment（即交付摘要）。
// 解析失败或没有 phase_comment 时返回空字符串。
func extractPhaseSummaryFromSignal(content string) string {
	if content == "" {
		return ""
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return ""
	}
	if comment, ok := data["phase_comment"].(string); ok && comment != "" {
		return comment
	}
	// fallback: 从 data.summary 里取
	if dataMap, ok := data["data"].(map[string]interface{}); ok {
		if summary, ok := dataMap["summary"].(string); ok && summary != "" {
			return summary
		}
	}
	return ""
}

// AppendEvent 往 events.jsonl 追加事件
func (qs *QuestStore) AppendEvent(qid string, ev *QuestEventRow) error {
	if ev.Timestamp == 0 {
		ev.Timestamp = model.NowMs()
	}
	if ev.QuestID == "" {
		ev.QuestID = qid
	}
	path := filepath.Join(qs.dir(qid), "events.jsonl")
	if ev.ID == 0 {
		ev.ID = nextQuestEventID(path)
	}
	return AppendJSONL(path, ev)
}

// ReadEvents 最后 n 条事件（SSE 重连补流）
func (qs *QuestStore) ReadEvents(qid string, n int) ([]QuestEventRow, error) {
	rows, err := ReadJSONL[QuestEventRow](filepath.Join(qs.dir(qid), "events.jsonl"), n)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if rows[i].ID == 0 {
			rows[i].ID = int64(i + 1)
		}
	}
	return rows, nil
}

func nextQuestEventID(path string) int64 {
	rows, err := ReadJSONL[QuestEventRow](path, 0)
	if err != nil || len(rows) == 0 {
		return 1
	}
	maxID := int64(0)
	for i, row := range rows {
		id := row.ID
		if id == 0 {
			id = int64(i + 1)
		}
		if id > maxID {
			maxID = id
		}
	}
	return maxID + 1
}

func (r *Root) AppendGlobalEvent(ev *GlobalEventRow) error {
	if ev == nil {
		return nil
	}
	if ev.Timestamp == 0 {
		ev.Timestamp = model.NowMs()
	}
	path := r.Sub(SubdirWorkspace, "events", "global.jsonl")
	if ev.GlobalID == 0 {
		ev.GlobalID = nextGlobalEventID(path)
	}
	return AppendJSONL(path, ev)
}

func (r *Root) ReadGlobalEvents(n int) ([]GlobalEventRow, error) {
	rows, err := ReadJSONL[GlobalEventRow](r.Sub(SubdirWorkspace, "events", "global.jsonl"), n)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if rows[i].GlobalID == 0 {
			rows[i].GlobalID = int64(i + 1)
		}
	}
	return rows, nil
}

func nextGlobalEventID(path string) int64 {
	rows, err := ReadJSONL[GlobalEventRow](path, 0)
	if err != nil || len(rows) == 0 {
		return 1
	}
	maxID := int64(0)
	for i, row := range rows {
		id := row.GlobalID
		if id == 0 {
			id = int64(i + 1)
		}
		if id > maxID {
			maxID = id
		}
	}
	return maxID + 1
}

// AppendReview 追加 reviews.jsonl（多轮返工，可累积）
type ReviewRecord struct {
	Ts              int64              `json:"ts"`
	Verdict         model.QuestVerdict `json:"verdict"`
	Comment         string             `json:"comment"`
	RewriteHints    string             `json:"rewrite_hints"`
	ReviewedBy      string             `json:"reviewed_by"`
	Score           int                `json:"score,omitempty"`             // 质量评分（1-10，仅 pass 时有意义）
	RepairCount     int                `json:"repair_count,omitempty"`      // 已修复问题数（从 comment 解析，P1-1 修复追踪）
	TotalIssueCount int                `json:"total_issue_count,omitempty"` // 总问题数（从 comment 解析）
	// ReviewSource 由调用方构造后传入（见 mage-review-spec v0.2.2 §0.2.1）。
	// fsstore 只持久化，绝不反向读 quest/phases 做推断。
	Source *ReviewSource `json:"review_source,omitempty"`
	// StructuredReview typed 协议（Phase 1.5+）。fsstore 只持久化，校验在 domain 层做。
	StructuredReview json.RawMessage `json:"structured_review,omitempty"`
}

// ReviewSource 标记一条 review 的来源身份和溯源信息。
// 结构与 domain/quest.ReviewSource 完全一致，由 quest_repository_adapter.go 做字段搬运。
type ReviewSource struct {
	SourceRole         string `json:"source_role"`
	SourceClass        string `json:"source_class,omitempty"`
	SourcePhaseIdx     int    `json:"source_phase_idx"`
	SourceAdventurerID string `json:"source_adventurer_id,omitempty"`
	SourceSessionID    string `json:"source_session_id,omitempty"`
	SourceIncomplete   bool   `json:"source_incomplete,omitempty"`
}

type AnswerRecord struct {
	AnswerID   string    `json:"answer_id"`
	QuestionID string    `json:"question_id,omitempty"`
	Content    string    `json:"content"`
	Source     string    `json:"source,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// RepairProgress 是从法师评审 comment 中提取的修复进度信息。
// 来源：P1-1 返工效果追踪，法师按"修复进度: X/Y"格式标注。
type RepairProgress struct {
	Repaired int  // 已修复问题数
	Total    int  // 总问题数
	HasData  bool // 是否成功解析到数据
}

// ParseRepairProgress 从评审 comment 中提取"修复进度: X/Y"格式的修复率。
// 用于 P1-1 返工效果追踪和 P1-3 僵局检测。
func ParseRepairProgress(comment string) RepairProgress {
	if comment == "" {
		return RepairProgress{}
	}
	// 匹配"修复进度: X/Y"格式，容错前后空白和中文括号
	// 支持：修复进度: 2/5、修复进度：3/6、修复进度:  1 / 3 等
	re := regexp.MustCompile(`修复进度[：:]\s*(\d+)\s*/\s*(\d+)`)
	matches := re.FindStringSubmatch(comment)
	if len(matches) < 3 {
		return RepairProgress{}
	}
	repaired, err1 := strconv.Atoi(matches[1])
	total, err2 := strconv.Atoi(matches[2])
	if err1 != nil || err2 != nil || total <= 0 {
		return RepairProgress{}
	}
	return RepairProgress{
		Repaired: repaired,
		Total:    total,
		HasData:  true,
	}
}

// Rate 返回修复率（0-1）。如果没有数据返回 -1。
func (p RepairProgress) Rate() float64 {
	if !p.HasData || p.Total <= 0 {
		return -1
	}
	return float64(p.Repaired) / float64(p.Total)
}

func (qs *QuestStore) AppendReview(qid string, r *ReviewRecord) error {
	if r.Ts == 0 {
		r.Ts = model.NowMs()
	}
	return AppendJSONL(filepath.Join(qs.dir(qid), "reviews.jsonl"), r)
}

func (qs *QuestStore) LoadReviews(qid string) ([]ReviewRecord, error) {
	return ReadJSONL[ReviewRecord](filepath.Join(qs.dir(qid), "reviews.jsonl"), 0)
}

func (qs *QuestStore) AppendAnswer(qid string, r *AnswerRecord) error {
	if r == nil {
		return fmt.Errorf("answer record 不能为空")
	}
	if strings.TrimSpace(r.Content) == "" {
		return fmt.Errorf("answer content 不能为空")
	}
	if r.AnswerID == "" {
		r.AnswerID = "answer_" + NewIDShort()
	}
	if r.Source == "" {
		r.Source = "api"
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	return AppendJSONL(filepath.Join(qs.dir(qid), "answers.jsonl"), r)
}

func (qs *QuestStore) LoadAnswers(qid string) ([]AnswerRecord, error) {
	return ReadJSONL[AnswerRecord](filepath.Join(qs.dir(qid), "answers.jsonl"), 0)
}

// ExportMessagesForExecutor 读 session 所有行 → 组装 executor.Message 数组
// （用于 rework 时 ImportContext）
func (qs *QuestStore) ExportMessagesForExecutor(qid, sid string) ([]executor.Message, error) {
	rows, err := qs.ReadSessionRows(qid, sid, 0)
	if err != nil {
		return nil, err
	}
	out := make([]executor.Message, 0, len(rows))
	for _, row := range rows {
		if row.Kind != "message" {
			continue
		}
		m := executor.Message{
			Role:       row.Role,
			Content:    row.Content,
			ToolCallID: row.ToolID,
		}
		if row.ToolArgs != "" {
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(row.ToolArgs), &args); err != nil {
				fmt.Fprintf(os.Stderr, "[warn] ExportMessagesForExecutor qid=%s sid=%s: bad tool_args: %v\n", qid, sid, err)
				continue
			}
			m.ToolCalls = []executor.ToolCall{{
				ID:        row.ToolID,
				ToolName:  row.ToolName,
				Arguments: args,
			}}
		}
		out = append(out, m)
	}
	return out, nil
}

// ==================== Quest 状态机（集中校验合法迁移） ====================

// ValidQuestTransitions 定义合法的 v2 状态迁移
// 主线: pending → running → reviewing → user_review → success/failed/cancelled
var ValidQuestTransitions = map[model.QuestStatus][]model.QuestStatus{
	"": {
		model.QuestStatusPending,
	},
	model.QuestStatusPending: {
		model.QuestStatusRunning,
		model.QuestStatusCancelled,
	},
	model.QuestStatusRunning: {
		model.QuestStatusReviewing,    // 剑士完成 → 法师评审
		model.QuestStatusWaitingInput, // 等待用户回答具体问题
		model.QuestStatusFailed,       // 致命错误
		model.QuestStatusBlocked,      // 阻塞
		model.QuestStatusCancelled,    // 用户取消
	},
	model.QuestStatusWaitingInput: {
		model.QuestStatusRunning,    // 收到回复 / 超时继续
		model.QuestStatusBlocked,    // 超时且策略要求阻塞
		model.QuestStatusUserReview, // 超时提交用户终审
		model.QuestStatusCancelled,  // 用户取消
	},
	model.QuestStatusReviewing: {
		model.QuestStatusRunning,    // rework → 剑士返工
		model.QuestStatusUserReview, // 法师通过 → 用户终审
		model.QuestStatusSuccess,    // 平台托管副作用已生效的 automation → 审计通过后直接完成
		model.QuestStatusFailed,     // 评审拒绝
		model.QuestStatusCancelled,  // 用户取消
	},
	model.QuestStatusUserReview: {
		model.QuestStatusRunning,   // 用户要求返工
		model.QuestStatusSuccess,   // 用户通过
		model.QuestStatusFailed,    // 用户拒绝
		model.QuestStatusBlocked,   // 用户确认超时 / 等待外部输入
		model.QuestStatusCancelled, // 用户取消
	},
	model.QuestStatusBlocked: {
		model.QuestStatusRunning,    // 解除阻塞
		model.QuestStatusUserReview, // 直接提交用户终审
		model.QuestStatusCancelled,  // 取消
	},
}

// terminalQuestStatuses 列出终态（不能再迁移出去）。
var terminalQuestStatuses = map[model.QuestStatus]bool{
	model.QuestStatusSuccess:   true,
	model.QuestStatusFailed:    true,
	model.QuestStatusCancelled: true,
}

// IsTerminal 返回该状态是否为终态。
func (q *QuestMeta) IsTerminal() bool {
	return terminalQuestStatuses[q.Status]
}

// CanTransitionTo 校验是否能迁移到 next 状态。
func (q *QuestMeta) CanTransitionTo(next model.QuestStatus) bool {
	cur := q.Status
	// 自环一律不合法（v2 没有自环概念）
	if cur == next {
		return false
	}
	// 终态不允许迁移出去
	if terminalQuestStatuses[cur] {
		return false
	}
	allowed, ok := ValidQuestTransitions[cur]
	if !ok {
		// 未知当前状态：只要 next 不是终态就允许（兼容老数据）
		return !terminalQuestStatuses[next]
	}
	for _, s := range allowed {
		if s == next {
			return true
		}
	}
	return false
}

// TransitionTo 执行状态迁移；不合法时返回 error（状态不改变）。
func (q *QuestMeta) TransitionTo(next model.QuestStatus) error {
	if !q.CanTransitionTo(next) {
		return fmt.Errorf("非法 Quest 状态迁移: %s → %s", q.Status, next)
	}
	q.Status = next
	return nil
}

func (q *QuestMeta) ResumeFromBlocked(comment string) error {
	if err := q.TransitionTo(model.QuestStatusRunning); err != nil {
		return err
	}
	q.ResumeCount++
	q.BlockedReason = ""
	q.BlockedReasonCode = ""
	q.BlockedCategory = ""
	if comment != "" {
		q.ReviewHints = comment
	}
	return nil
}

func (q *QuestMeta) MoveBlockedToUserReview(comment string) error {
	if err := q.TransitionTo(model.QuestStatusUserReview); err != nil {
		return err
	}
	q.FinalComment = comment
	if q.FinalComment == "" {
		q.FinalComment = "blocked 后由用户转入终审"
	}
	q.BlockedReason = ""
	q.BlockedReasonCode = ""
	q.BlockedCategory = ""
	return nil
}

func (q *QuestMeta) CancelFromBlocked(comment string) error {
	if err := q.TransitionTo(model.QuestStatusCancelled); err != nil {
		return err
	}
	q.CompletedAtMs = NowMs()
	q.FinalComment = comment
	return nil
}

func (q *QuestMeta) RequestUserReviewRework(comment string) error {
	if err := q.TransitionTo(model.QuestStatusRunning); err != nil {
		return err
	}
	q.ReworkCount++
	q.ReviewHints = comment
	return nil
}

func (q *QuestMeta) CompleteUserReview(verdict model.QuestVerdict, comment string) error {
	return q.CompleteReviewed(verdict, comment)
}

func (q *QuestMeta) CompleteReviewed(verdict model.QuestVerdict, comment string) error {
	var next model.QuestStatus
	switch verdict {
	case model.VerdictPass:
		next = model.QuestStatusSuccess
	case model.VerdictReject:
		next = model.QuestStatusFailed
	default:
		return fmt.Errorf("用户终审完成不支持 verdict: %s", verdict)
	}
	if err := q.TransitionTo(next); err != nil {
		return err
	}
	q.FinalVerdict = verdict
	q.FinalComment = comment
	q.CompletedAtMs = NowMs()
	return nil
}

func (q *QuestMeta) BuildDesignDoc(summary string, reviewComment string) *DesignDoc {
	return BuildDesignDoc(q, summary, reviewComment)
}

func BuildDesignDoc(q *QuestMeta, summary string, reviewComment string) *DesignDoc {
	if summary == "" {
		summary = reviewComment
	}
	return &DesignDoc{
		QuestID:       q.ID,
		SchemaVersion: "gloop.design.v1",
		OriginalQuery: q.Query,
		Summary:       summary,
		ReviewComment: reviewComment,
		Goals: []string{
			"按方案摘要完成可运行实现",
		},
		NonGoals: []string{
			"不要扩展到方案未覆盖的额外产品范围",
		},
		ImplementationSteps: []string{
			"阅读 design summary 与原始需求",
			"拆分实现步骤并逐步落地",
			"验证关键风险和验收标准",
		},
		Risks: []string{
			"方案摘要可能仍不完整，执行前需要主动补齐关键不确定点",
		},
		AcceptanceCriteria: []string{
			"实现应覆盖方案摘要中的核心设计目标",
			"实现边界应与 design quest 的非目标保持一致",
			"关键风险需要在执行阶段被验证或显式记录",
		},
		ExecutePrompt: fmt.Sprintf("基于方案委托 %s 执行落地。\n\n原始需求：\n%s\n\n已通过方案摘要：\n%s\n\n执行要求：\n- 按方案拆解并实现\n- 保持与现有代码风格一致\n- 对风险点给出验证结果\n- 完成后提交阶段总结", q.ID, q.Query, summary),
		CreatedAtMs:   NowMs(),
	}
}
