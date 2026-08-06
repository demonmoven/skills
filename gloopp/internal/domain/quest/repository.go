package quest

import (
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== Quest Repository 接口 ====================
//
// 这是领域层定义的存储抽象。
// 所有业务逻辑（orchestrator、service、API 层）都应该依赖这个接口，
// 而不是直接依赖 fsstore 的具体实现。
//
// 这样设计的好处：
// 1. 业务逻辑与存储解耦，容易测试（可以用 memory repository）
// 2. 未来可以平滑替换存储后端（SQLite、PostgreSQL 等）
// 3. 存储层的实现细节不会泄露到业务层
//
// Phase 1 目标：定义接口，fsstore 作为第一个实现。
// 注意：接口还不完整，会随着迁移逐步完善。

// Repository 定义 Quest 的存储接口。
// 这是一个"集合式"接口，参考 DDD 的 Repository 模式。
type Repository interface {
	// ===== 基本读写 =====

	// Load 加载一个 Quest。找不到返回错误。
	Load(id string) (*QuestMeta, error)

	// Save 保存 Quest（新建或更新）。
	Save(q *QuestMeta) error

	// Delete 删除 Quest（可选，当前可能未实现）。
	Delete(id string) error

	// ===== 列表查询 =====

	// List 列出所有 Quest（按创建时间倒序）。
	List() ([]*QuestMeta, error)

	// ListByStatus 按状态筛选。
	ListByStatus(status model.QuestStatus) ([]*QuestMeta, error)

	// ===== 事件与日志 =====

	// AppendEvent 追加事件（events.jsonl）。
	AppendEvent(qid string, eventType model.EventType, payload any) error

	// AppendReview 追加评审记录。
	AppendReview(qid string, review *ReviewRecord) error

	// LoadReviews 加载所有评审记录。
	LoadReviews(qid string) ([]*ReviewRecord, error)

	// ===== 工作区 =====

	// WorkspacePath 返回工作区路径。
	WorkspacePath(qid string) string

	// ===== 统计 =====

	// CountByStatus 按状态统计数量。
	CountByStatus() (map[model.QuestStatus]int, error)
}

// QuestMeta 是领域层定义的 Quest 持久化结构。
//
// Phase 1: 这是 fsstore.QuestMeta 的领域层镜像，用于解耦。
// 未来 Phase 2/3 会逐步演化为完整的领域聚合根。
//
// 注意：为了降低迁移成本，字段布局与 fsstore.QuestMeta 基本一致，
// 但领域层不包含存储实现细节（如文件路径等）。
type QuestMeta struct {
	ID      string            `json:"id"`
	ShortID string            `json:"short_id"`
	Query   string            `json:"query"`
	Type    model.QuestType   `json:"type"`
	Status  model.QuestStatus `json:"status"`

	// v0.5 Post/thread root fields. Query is the legacy compatibility text;
	// OriginalRequest is the immutable user-goal anchor.
	OriginalRequest      string             `json:"original_request,omitempty"`
	IntentSummary        string             `json:"intent_summary,omitempty"`
	Constraints          []string           `json:"constraints,omitempty"`
	ExpectedArtifacts    []string           `json:"expected_artifacts,omitempty"`
	WorkflowMode         model.WorkflowMode `json:"workflow_mode,omitempty"`
	PinnedOutcomeSummary string             `json:"pinned_outcome_summary,omitempty"`

	// 强度与预算
	Intensity                model.QuestIntensity `json:"intensity,omitempty"`
	MaxRework                int                  `json:"max_rework"`
	MaxTurnsPerPhaseOverride int                  `json:"max_turns_per_phase_override,omitempty"`
	MaxDurationMsOverride    int64                `json:"max_duration_per_quest_ms_override,omitempty"`

	// 冒险者
	WarriorID      string `json:"warrior_id"`
	MageID         string `json:"mage_id"`
	ExecuteAgentID string `json:"execute_agent_id,omitempty"`
	ReviewAgentID  string `json:"review_agent_id,omitempty"`

	// 阶段管道
	// Pipeline 是阶段定义列表（静态配置）
	Pipeline Pipeline `json:"-"` // 不序列化，运行时从默认值或 quest 类型派生
	// Pipeline persistence fields mirror fsstore.QuestMeta for recovery.
	PipelineVersion int        `json:"pipeline_version,omitempty"`
	PipelineName    string     `json:"pipeline_name,omitempty"`
	PhaseCount      int        `json:"phase_count,omitempty"`
	PipelineDefHash string     `json:"pipeline_def_hash,omitempty"`
	PipelineDef     []PhaseDef `json:"pipeline_def,omitempty"`
	// CurrentPhaseIdx 当前执行到第几个阶段
	CurrentPhaseIdx int `json:"current_phase_idx,omitempty"`
	// Phases 各阶段的运行时状态
	Phases []PhaseRun `json:"phases,omitempty"`

	// 工作区
	WorkspaceMode  model.WorkspaceMode `json:"workspace_mode"`
	WorkspacePath  string              `json:"workspace_path"`
	BaseWorkingDir string              `json:"base_working_dir"`
	BaseBranch     string              `json:"base_branch,omitempty"`
	BaseCommit     string              `json:"base_commit,omitempty"`

	// 过程
	ReworkCount            int                `json:"rework_count"`
	ResumeCount            int                `json:"resume_count,omitempty"`
	ReviewHints            string             `json:"review_hints,omitempty"`
	CommentCursor          int                `json:"comment_cursor,omitempty"` // 已处理的用户评论游标
	DesignSummary          string             `json:"design_summary,omitempty"`
	ParentQuestID          string             `json:"parent_quest_id,omitempty"`
	AutoSpawnExecute       bool               `json:"auto_spawn_execute,omitempty"`
	ChildExecuteQuestID    string             `json:"child_execute_quest_id,omitempty"`
	SpawnError             string             `json:"spawn_error,omitempty"`
	GroupID                string             `json:"group_id,omitempty"`
	FanoutLeafID           string             `json:"fanout_leaf_id,omitempty"`
	OwnershipScopes        []string           `json:"ownership_scopes,omitempty"`
	MergeStrategy          string             `json:"merge_strategy,omitempty"`
	MergeOwnerLeafID       string             `json:"merge_owner_leaf_id,omitempty"`
	AcceptanceCriteria     string             `json:"acceptance_criteria,omitempty"`
	AllowQuickAutoComplete bool               `json:"allow_quick_auto_complete,omitempty"`
	TriageMode             string             `json:"triage_mode,omitempty"`
	WaitingInput           *WaitingInputState `json:"waiting_input,omitempty"`

	// 阻塞时长核算：blocked 期间 quest 不在做实际工作，这段时间不计入总时长
	// 预算（与 waiting_input 暂停同理）。进 blocked 记 BlockedAtMs，出 blocked
	// 把本次时长累加进 AccumulatedBlockedMs（见 ApplyTransitionResult）。
	BlockedAtMs          int64 `json:"blocked_at_ms,omitempty"`
	AccumulatedBlockedMs int64 `json:"accumulated_blocked_ms,omitempty"`

	// 结果
	FinalVerdict          model.QuestVerdict  `json:"final_verdict,omitempty"`
	FinalComment          string              `json:"final_comment,omitempty"`
	ImpactSummary         model.ImpactSummary `json:"impact_summary,omitempty"` // HOTL: agent 主动声明的影响
	FinalizedBy           string              `json:"finalized_by,omitempty"`
	AutoPassedByPolicy    string              `json:"auto_passed_by_policy,omitempty"`
	AutoCompletedByPolicy string              `json:"auto_completed_by_policy,omitempty"`
	PolicyDecisionID      string              `json:"policy_decision_id,omitempty"`
	Applied               bool                `json:"applied,omitempty"`
	ApplyStatus           model.ApplyStatus   `json:"apply_status,omitempty"`
	ApplyError            string              `json:"apply_error,omitempty"`
	ApplyFailedAtMs       int64               `json:"apply_failed_at_ms,omitempty"`
	BlockedReason         string              `json:"blocked_reason,omitempty"`
	BlockedReasonCode     string              `json:"blocked_reason_code,omitempty"`
	BlockedCategory       string              `json:"blocked_category,omitempty"`
	FailureAttribution    *FailureAttribution `json:"failure_attribution,omitempty"`
	MageScore             int                 `json:"mage_score,omitempty"`

	// Diff 摘要
	DiffStat             string `json:"diff_stat,omitempty"`
	DiffChangedFiles     int    `json:"diff_changed_files,omitempty"`
	DiffAdditions        int    `json:"diff_additions,omitempty"`
	DiffDeletions        int    `json:"diff_deletions,omitempty"`
	WorkspaceCleaned     bool   `json:"workspace_cleaned,omitempty"`
	WorkspaceCleanedAtMs int64  `json:"workspace_cleaned_at_ms,omitempty"`

	// 元信息
	CreatedBy     string `json:"created_by"`
	Official      bool   `json:"official,omitempty"`
	CreatedAtMs   int64  `json:"created_at_ms"`
	UpdatedAtMs   int64  `json:"updated_at_ms,omitempty"`
	StartedAtMs   int64  `json:"started_at_ms,omitempty"`
	CompletedAtMs int64  `json:"completed_at_ms,omitempty"`
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
}

// ReviewRecord 评审记录。
type ReviewRecord struct {
	Ts           int64              `json:"ts"`
	Verdict      model.QuestVerdict `json:"verdict"`
	Comment      string             `json:"comment"`
	RewriteHints string             `json:"rewrite_hints"`
	ReviewedBy   string             `json:"reviewed_by"`
	Score        int                `json:"score,omitempty"`
	// ReviewSource 由调用方在写入时构造（见 mage-review-spec v0.2.2 §0.2.1）。
	// fsstore 只持久化，绝不反向读 quest/phases 做推断。
	Source *ReviewSource `json:"review_source,omitempty"`
	// StructuredReview typed 协议（Phase 1.5+，见 spec §4.1）。
	// 为 nil 时前端降级展示原始 comment。校验失败时不持久化（丢弃 + 发 event）。
	StructuredReview *StructuredReview `json:"structured_review,omitempty"`
}

// ReviewSource 标记一条 review 的来源身份和溯源信息。
// 前端 selectLatestMageReview 优先按此字段判断 Mage Review（Phase 1.5+）；
// 历史数据无此字段时回退到 ReviewedBy 兼容路径。
type ReviewSource struct {
	SourceRole         string `json:"source_role"`                    // warrior / mage / user / policy / automation — 永远必填
	SourceClass        string `json:"source_class,omitempty"`         // 冒险者职业：warrior/mage。非冒险者角色留空
	SourcePhaseIdx     int    `json:"source_phase_idx"`               // 阶段索引；非 phase 内决策填 -1（0 是 warrior phase）
	SourceAdventurerID string `json:"source_adventurer_id,omitempty"` // 冒险者 ID。user/policy/automation 留空
	SourceSessionID    string `json:"source_session_id,omitempty"`    // phase 对应 session id。非 phase 内决策留空
	SourceIncomplete   bool   `json:"source_incomplete,omitempty"`    // true = 溯源字段不全，身份可信任但 phase/session 不可靠
}

// ===== QuestMeta 的领域方法 =====

// IsTerminal 返回是否为终态。
func (q *QuestMeta) IsTerminal() bool {
	return IsTerminal(q.Status)
}

// CanTransitionTo 检查能否迁移到目标状态。
func (q *QuestMeta) CanTransitionTo(next model.QuestStatus) bool {
	return CanTransition(q.Status, next)
}

// TransitionTo 执行状态迁移。
// 注意：这是在领域对象上的操作，只修改内存状态，不持久化。
// 调用方需要显式调用 Repository.Save() 持久化。
func (q *QuestMeta) TransitionTo(next model.QuestStatus) error {
	if !q.CanTransitionTo(next) {
		return nil
	}
	q.Status = next
	return nil
}

// CanRework 检查是否还能返工（未达最大返工次数）。
func (q *QuestMeta) CanRework() bool {
	return q.ReworkCount < q.MaxRework
}

// Source 返回 quest 来源。
func (q *QuestMeta) Source() model.QuestSource {
	return QuestSource(q.CreatedBy)
}

// AttributeFailure 对该 quest 进行失败归因。
func (q *QuestMeta) AttributeFailure() *FailureAttribution {
	return AttributeFailure(&questAttributable{q})
}

// questAttributable 适配 QuestMeta 到 Attributable 接口。
type questAttributable struct {
	q *QuestMeta
}

func (a *questAttributable) GetStatus() model.QuestStatus        { return a.q.Status }
func (a *questAttributable) GetBlockedReason() string            { return a.q.BlockedReason }
func (a *questAttributable) GetFinalVerdict() model.QuestVerdict { return a.q.FinalVerdict }
func (a *questAttributable) GetFinalComment() string             { return a.q.FinalComment }
func (a *questAttributable) GetApplyStatus() model.ApplyStatus   { return a.q.ApplyStatus }
func (a *questAttributable) GetApplyError() string               { return a.q.ApplyError }

// ApplyTransitionResult 把状态迁移结果应用到 QuestMeta。
func (q *QuestMeta) ApplyTransitionResult(r TransitionResult) {
	// 阻塞时长核算（在改 Status 前判断旧状态）：
	//   - 进入 blocked：记录 BlockedAtMs 起点。
	//   - 离开 blocked（ClearBlocked / 转其它态）：把本次 blocked 时长累加进
	//     AccumulatedBlockedMs，由 questDurationElapsedMs 从墙钟 elapsed 扣除。
	if r.NewStatus == model.QuestStatusBlocked && q.Status != model.QuestStatusBlocked {
		q.BlockedAtMs = model.NowMs()
	} else if q.Status == model.QuestStatusBlocked && r.NewStatus != model.QuestStatusBlocked {
		if q.BlockedAtMs > 0 {
			if now := model.NowMs(); now > q.BlockedAtMs {
				q.AccumulatedBlockedMs += now - q.BlockedAtMs
			}
			q.BlockedAtMs = 0
		}
	}

	q.Status = r.NewStatus
	if r.FinalVerdict != "" {
		q.FinalVerdict = r.FinalVerdict
	}
	if r.FinalComment != "" {
		q.FinalComment = r.FinalComment
	}
	if r.BlockedReason != "" {
		q.BlockedReason = r.BlockedReason
	}
	if r.BlockedReasonCode != "" {
		q.BlockedReasonCode = r.BlockedReasonCode
	}
	if r.BlockedCategory != "" {
		q.BlockedCategory = r.BlockedCategory
	}
	if r.ReviewHints != "" {
		q.ReviewHints = r.ReviewHints
	}
	if r.ReworkCount > 0 {
		q.ReworkCount = r.ReworkCount
	}
	if r.ResumeCount > 0 {
		q.ResumeCount = r.ResumeCount
	}
	if r.ClearBlocked {
		q.BlockedReason = ""
		q.BlockedReasonCode = ""
		q.BlockedCategory = ""
		// FailureAttribution 是 blocked 状态的伴生字段（进 blocked 时写、出 blocked 时清）。
		// 离开 blocked 即归因过期，与阻塞原因一并清，避免半清理状态污染后续阶段。
		q.FailureAttribution = nil
	}
	if r.SetCompleted && q.CompletedAtMs == 0 {
		q.CompletedAtMs = model.NowMs()
	}
	if r.SetStarted && q.StartedAtMs == 0 {
		q.StartedAtMs = model.NowMs()
	}
}

// ===== Phase 相关辅助方法 =====

// EnsurePipeline 确保 Pipeline 已初始化。
// 如果 Pipeline 为空，使用默认管道。
func (q *QuestMeta) EnsurePipeline() {
	if len(q.Pipeline) > 0 {
		return
	}
	q.Pipeline = DefaultPipeline()
}

// EnsurePhases 确保 Phases 运行时列表已初始化。
// 如果 Phases 为空，根据 Pipeline 创建对应数量的 pending 阶段。
func (q *QuestMeta) EnsurePhases() {
	q.EnsurePipeline()
	if len(q.Phases) == len(q.Pipeline) {
		return
	}
	// 如果已有 phases 但数量不匹配，保留已有数据，补充缺失的
	if len(q.Phases) < len(q.Pipeline) {
		for i := len(q.Phases); i < len(q.Pipeline); i++ {
			def := q.Pipeline[i]
			q.Phases = append(q.Phases, PhaseRun{
				PhaseIdx: def.Index,
				Status:   PhasePending,
			})
		}
	}
}

// CurrentPhaseDef 返回当前阶段的定义。
// 如果没有当前阶段，返回 nil。
func (q *QuestMeta) CurrentPhaseDef() *PhaseDef {
	q.EnsurePipeline()
	return q.Pipeline.At(q.CurrentPhaseIdx)
}

// CurrentPhaseRun 返回当前阶段的运行时状态。
// 如果没有当前阶段，返回 nil。
func (q *QuestMeta) CurrentPhaseRun() *PhaseRun {
	q.EnsurePhases()
	if q.CurrentPhaseIdx < 0 || q.CurrentPhaseIdx >= len(q.Phases) {
		return nil
	}
	return &q.Phases[q.CurrentPhaseIdx]
}

// PhaseRunAt 返回指定索引的阶段运行时状态。
func (q *QuestMeta) PhaseRunAt(idx int) *PhaseRun {
	q.EnsurePhases()
	if idx < 0 || idx >= len(q.Phases) {
		return nil
	}
	return &q.Phases[idx]
}

// StartCurrentPhase 标记当前阶段为运行中。
func (q *QuestMeta) StartCurrentPhase(nowMs int64) {
	pr := q.CurrentPhaseRun()
	if pr != nil {
		pr.Start(nowMs)
	}
}

// CompleteCurrentPhase 标记当前阶段为已完成。
func (q *QuestMeta) CompleteCurrentPhase(nowMs int64) {
	pr := q.CurrentPhaseRun()
	if pr != nil {
		pr.Complete(nowMs)
	}
}

// AdvanceToNextPhase 推进到下一阶段。
// 返回新的阶段索引，如果已经是最后阶段返回 -1。
func (q *QuestMeta) AdvanceToNextPhase() int {
	q.EnsurePipeline()
	next := q.Pipeline.NextPhaseIdx(q.CurrentPhaseIdx)
	if next < 0 {
		return -1
	}
	q.CurrentPhaseIdx = next
	return next
}

// ReworkToPhase 重置并回到指定阶段（返工）。
// 重置目标阶段及其之后所有阶段的状态。
func (q *QuestMeta) ReworkToPhase(targetIdx int, nowMs int64) {
	q.EnsurePhases()
	if targetIdx < 0 || targetIdx >= len(q.Phases) {
		return
	}
	// 重置从目标阶段开始的所有后续阶段
	for i := targetIdx; i < len(q.Phases); i++ {
		q.Phases[i].Reset()
	}
	q.CurrentPhaseIdx = targetIdx
}

// SetPhaseAdventurer 设置指定阶段的冒险者 ID。
func (q *QuestMeta) SetPhaseAdventurer(phaseIdx int, advID string) {
	q.EnsurePhases()
	if phaseIdx >= 0 && phaseIdx < len(q.Phases) {
		q.Phases[phaseIdx].AdventurerID = advID
	}
}

// SetPhaseSessionID 设置指定阶段的会话 ID。
func (q *QuestMeta) SetPhaseSessionID(phaseIdx int, sessionID string) {
	q.EnsurePhases()
	if phaseIdx >= 0 && phaseIdx < len(q.Phases) {
		q.Phases[phaseIdx].SessionID = sessionID
	}
}

// SetPhaseTurns 设置指定阶段的回合数。
func (q *QuestMeta) SetPhaseTurns(phaseIdx int, turns int) {
	q.EnsurePhases()
	if phaseIdx >= 0 && phaseIdx < len(q.Phases) {
		q.Phases[phaseIdx].Turns = turns
	}
}
