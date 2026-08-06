package fsstore

import (
	"encoding/json"

	"code.byted.org/lihuanyu.0w0/gloop/internal/domain/quest"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== Quest Repository 适配器 ====================
//
// QuestStoreAdapter 让 QuestStore 适配 domain/quest.Repository 接口。
// 这是 Phase 1 的过渡层：领域层通过 Repository 接口访问存储，
// 底层仍然使用现有的 fsstore 实现。
//
// 后续 Phase 2/3 可以替换为其他存储实现（如 SQLite），
// 而领域层和服务层代码不需要改动。

// NewQuestRepository 创建一个 quest.Repository 实现。
// 这是领域层的入口点，所有新代码应该通过 Repository 接口访问存储。
func NewQuestRepository(root *Root) quest.Repository {
	return &QuestRepositoryAdapter{
		store: NewQuestStore(root),
	}
}

// QuestRepositoryAdapter 是 quest.Repository 的 fsstore 实现。
type QuestRepositoryAdapter struct {
	store *QuestStore
}

// ===== 基本读写 =====

func (a *QuestRepositoryAdapter) Load(id string) (*quest.QuestMeta, error) {
	q, err := a.store.LoadQuest(id)
	if err != nil {
		return nil, err
	}
	return QuestMetaToDomain(q), nil
}

func (a *QuestRepositoryAdapter) Save(q *quest.QuestMeta) error {
	fs := QuestMetaFromDomain(q)
	a.preserveFsstoreOnlyFields(fs, q.ID)
	return a.store.SaveQuest(fs)
}

// preserveFsstoreOnlyFields 保留 fsstore 独有的附属字段。
//
// domain QuestMeta 只建模状态机字段，不认识 Outputs/Connectors/PromptVersion/
// Goal*/WorkspaceDowngrade 等 fsstore 附属数据。QuestMetaFromDomain 重建时这些字段
// 全是零值，直接 SaveQuest 会把磁盘已有值覆盖成空——导致剑士声明的产出物、HOTL
// connector、prompt 快照等在 domain round-trip 后丢失。
//
// 语义：这些字段归 fsstore 直接路径管理（AddDeclaredOutput/ClearOutputs 等），
// domain 状态机无权修改也无权清空，round-trip 时无条件保留磁盘已有值。
func (a *QuestRepositoryAdapter) preserveFsstoreOnlyFields(fs *QuestMeta, qid string) {
	existing, err := a.store.LoadQuest(qid)
	if err != nil || existing == nil {
		return
	}
	fs.Outputs = existing.Outputs
	fs.Connectors = existing.Connectors
	fs.PromptVersion = existing.PromptVersion
	fs.PromptOverrides = existing.PromptOverrides
	fs.PromptOverrideMap = existing.PromptOverrideMap
	fs.WorkspaceDowngrade = existing.WorkspaceDowngrade
	fs.GoalIterations = existing.GoalIterations
	fs.GoalMaxIterations = existing.GoalMaxIterations
	fs.GoalCondition = existing.GoalCondition
}

func (a *QuestRepositoryAdapter) Delete(id string) error {
	// TODO: 实现删除逻辑（当前 fsstore 未提供删除功能）
	return nil
}

// ===== 列表查询 =====

func (a *QuestRepositoryAdapter) List() ([]*quest.QuestMeta, error) {
	items, err := a.store.ListQuests()
	if err != nil {
		return nil, err
	}
	out := make([]*quest.QuestMeta, len(items))
	for i, q := range items {
		out[i] = QuestMetaToDomain(q)
	}
	return out, nil
}

func (a *QuestRepositoryAdapter) ListByStatus(status model.QuestStatus) ([]*quest.QuestMeta, error) {
	all, err := a.List()
	if err != nil {
		return nil, err
	}
	var out []*quest.QuestMeta
	for _, q := range all {
		if q.Status == status {
			out = append(out, q)
		}
	}
	return out, nil
}

// ===== 事件与日志 =====

func (a *QuestRepositoryAdapter) AppendEvent(qid string, eventType model.EventType, payload any) error {
	ev := &QuestEventRow{
		Timestamp: NowMs(),
		Type:      string(eventType),
		QuestID:   qid,
		Payload:   payload,
	}
	return a.store.AppendEvent(qid, ev)
}

func (a *QuestRepositoryAdapter) AppendReview(qid string, r *quest.ReviewRecord) error {
	rr := &ReviewRecord{
		Ts:           r.Ts,
		Verdict:      r.Verdict,
		Comment:      r.Comment,
		RewriteHints: r.RewriteHints,
		ReviewedBy:   r.ReviewedBy,
		Score:        r.Score,
	}
	if r.Source != nil {
		rr.Source = &ReviewSource{
			SourceRole:         r.Source.SourceRole,
			SourceClass:        r.Source.SourceClass,
			SourcePhaseIdx:     r.Source.SourcePhaseIdx,
			SourceAdventurerID: r.Source.SourceAdventurerID,
			SourceSessionID:    r.Source.SourceSessionID,
			SourceIncomplete:   r.Source.SourceIncomplete,
		}
	}
	if r.StructuredReview != nil {
		if raw, err := json.Marshal(r.StructuredReview); err == nil {
			rr.StructuredReview = raw
		}
	}
	return a.store.AppendReview(qid, rr)
}

func (a *QuestRepositoryAdapter) LoadReviews(qid string) ([]*quest.ReviewRecord, error) {
	records, err := a.store.LoadReviews(qid)
	if err != nil {
		return nil, err
	}
	out := make([]*quest.ReviewRecord, len(records))
	for i, r := range records {
		rec := &quest.ReviewRecord{
			Ts:           r.Ts,
			Verdict:      r.Verdict,
			Comment:      r.Comment,
			RewriteHints: r.RewriteHints,
			ReviewedBy:   r.ReviewedBy,
			Score:        r.Score,
		}
		if r.Source != nil {
			rec.Source = &quest.ReviewSource{
				SourceRole:         r.Source.SourceRole,
				SourceClass:        r.Source.SourceClass,
				SourcePhaseIdx:     r.Source.SourcePhaseIdx,
				SourceAdventurerID: r.Source.SourceAdventurerID,
				SourceSessionID:    r.Source.SourceSessionID,
				SourceIncomplete:   r.Source.SourceIncomplete,
			}
		}
		if len(r.StructuredReview) > 0 {
			var sr quest.StructuredReview
			if err := json.Unmarshal(r.StructuredReview, &sr); err == nil {
				rec.StructuredReview = &sr
			}
		}
		out[i] = rec
	}
	return out, nil
}

// ===== 工作区 =====

func (a *QuestRepositoryAdapter) WorkspacePath(qid string) string {
	// 工作区路径在 quest meta 中，需要先加载
	// 注意：这是一个便捷方法，效率不高；频繁调用应该直接用 Load()
	q, err := a.store.LoadQuest(qid)
	if err != nil {
		return ""
	}
	return q.WorkspacePath
}

// ===== 统计 =====

func (a *QuestRepositoryAdapter) CountByStatus() (map[model.QuestStatus]int, error) {
	all, err := a.List()
	if err != nil {
		return nil, err
	}
	counts := make(map[model.QuestStatus]int)
	for _, q := range all {
		counts[q.Status]++
	}
	return counts, nil
}

// ==================== 类型转换 ====================
//
// 在 fsstore.QuestMeta 和 domain quest.QuestMeta 之间转换。
// 这是一层薄薄的转换层是必要的架构成本，它让领域层和存储层解耦。

// questMetaToDomain 把 fsstore.QuestMeta 转为 domain quest.QuestMeta。
func QuestMetaToDomain(q *QuestMeta) *quest.QuestMeta {
	if q == nil {
		return nil
	}
	// 确保 phases 已初始化（老数据兼容）
	q.EnsurePhases()

	d := &quest.QuestMeta{
		ID:      q.ID,
		ShortID: q.ShortID,
		Query:   q.Query,
		Type:    q.Type,
		Status:  q.Status,

		OriginalRequest:      q.OriginalRequest,
		IntentSummary:        q.IntentSummary,
		Constraints:          append([]string(nil), q.Constraints...),
		ExpectedArtifacts:    append([]string(nil), q.ExpectedArtifacts...),
		WorkflowMode:         q.WorkflowMode,
		PinnedOutcomeSummary: q.PinnedOutcomeSummary,

		Intensity:                q.Intensity,
		MaxRework:                q.MaxRework,
		MaxTurnsPerPhaseOverride: q.MaxTurnsPerPhaseOverride,
		MaxDurationMsOverride:    q.MaxDurationPerQuestMsOverride,

		WarriorID:      q.WarriorID,
		MageID:         q.MageID,
		ExecuteAgentID: q.ExecuteAgentID,
		ReviewAgentID:  q.ReviewAgentID,

		PipelineVersion: q.PipelineVersion,
		PipelineName:    q.PipelineName,
		PhaseCount:      q.PhaseCount,
		PipelineDefHash: q.PipelineDefHash,
		PipelineDef:     phaseDefsToDomain(q.PipelineDef),
		CurrentPhaseIdx: q.CurrentPhaseIdx(),

		WorkspaceMode:  q.WorkspaceMode,
		WorkspacePath:  q.WorkspacePath,
		BaseWorkingDir: q.BaseWorkingDir,
		BaseBranch:     q.BaseBranch,
		BaseCommit:     q.BaseCommit,

		ReworkCount:          q.ReworkCount,
		ResumeCount:          q.ResumeCount,
		ReviewHints:          q.ReviewHints,
		CommentCursor:        q.CommentCursor,
		DesignSummary:        q.DesignSummary,
		ParentQuestID:        q.ParentQuestID,
		AutoSpawnExecute:     q.AutoSpawnExecute,
		ChildExecuteQuestID:  q.ChildExecuteQuestID,
		SpawnError:           q.SpawnError,
		GroupID:              q.GroupID,
		FanoutLeafID:         q.FanoutLeafID,
		OwnershipScopes:      append([]string(nil), q.OwnershipScopes...),
		MergeStrategy:        q.MergeStrategy,
		MergeOwnerLeafID:     q.MergeOwnerLeafID,
		AcceptanceCriteria:   q.AcceptanceCriteria,
		TriageMode:           q.TriageMode,
		WaitingInput:         waitingInputToDomain(q.WaitingInput),
		BlockedAtMs:          q.BlockedAtMs,
		AccumulatedBlockedMs: q.AccumulatedBlockedMs,

		FinalVerdict:          q.FinalVerdict,
		FinalComment:          q.FinalComment,
		ImpactSummary:         q.ImpactSummary,
		FinalizedBy:           q.FinalizedBy,
		AutoPassedByPolicy:    q.AutoPassedByPolicy,
		AutoCompletedByPolicy: q.AutoCompletedByPolicy,
		PolicyDecisionID:      q.PolicyDecisionID,
		Applied:               q.Applied,
		ApplyStatus:           q.ApplyStatus,
		ApplyError:            q.ApplyError,
		ApplyFailedAtMs:       q.ApplyFailedAtMs,
		BlockedReason:         q.BlockedReason,
		BlockedReasonCode:     q.BlockedReasonCode,
		BlockedCategory:       q.BlockedCategory,
		MageScore:             q.MageScore,

		DiffStat:             q.DiffStat,
		DiffChangedFiles:     q.DiffChangedFiles,
		DiffAdditions:        q.DiffAdditions,
		DiffDeletions:        q.DiffDeletions,
		WorkspaceCleaned:     q.WorkspaceCleaned,
		WorkspaceCleanedAtMs: q.WorkspaceCleanedAtMs,

		CreatedBy:     q.CreatedBy,
		Official:      q.Official,
		CreatedAtMs:   q.CreatedAtMs,
		UpdatedAtMs:   q.UpdatedAtMs,
		StartedAtMs:   q.StartedAtMs,
		CompletedAtMs: q.CompletedAtMs,
	}

	// 转换 phases
	if len(q.Phases) > 0 {
		d.Phases = make([]quest.PhaseRun, len(q.Phases))
		for i, p := range q.Phases {
			d.Phases[i] = phaseTaskToDomain(p)
		}
	}

	if len(d.PipelineDef) > 0 {
		d.Pipeline = quest.Pipeline(d.PipelineDef)
	} else {
		d.Pipeline = quest.DefaultPipeline()
	}

	if q.FailureAttribution != nil {
		d.FailureAttribution = &quest.FailureAttribution{
			Stage:       quest.FailureStage(q.FailureAttribution.Stage),
			Reason:      quest.FailureReason(q.FailureAttribution.Reason),
			Category:    q.FailureAttribution.Category,
			Message:     q.FailureAttribution.Message,
			Recoverable: q.FailureAttribution.Recoverable,
			Actions:     q.FailureAttribution.Actions,
			Details:     q.FailureAttribution.Details,
		}
	}
	return d
}

// questMetaFromDomain 把 domain quest.QuestMeta 转回 fsstore.QuestMeta。
func QuestMetaFromDomain(d *quest.QuestMeta) *QuestMeta {
	if d == nil {
		return nil
	}
	q := &QuestMeta{
		ID:      d.ID,
		ShortID: d.ShortID,
		Query:   d.Query,
		Type:    d.Type,
		Status:  d.Status,

		OriginalRequest:      d.OriginalRequest,
		IntentSummary:        d.IntentSummary,
		Constraints:          append([]string(nil), d.Constraints...),
		ExpectedArtifacts:    append([]string(nil), d.ExpectedArtifacts...),
		WorkflowMode:         d.WorkflowMode,
		PinnedOutcomeSummary: d.PinnedOutcomeSummary,

		Intensity:                     d.Intensity,
		MaxRework:                     d.MaxRework,
		MaxTurnsPerPhaseOverride:      d.MaxTurnsPerPhaseOverride,
		MaxDurationPerQuestMsOverride: d.MaxDurationMsOverride,

		WarriorID:      d.WarriorID,
		MageID:         d.MageID,
		ExecuteAgentID: d.ExecuteAgentID,
		ReviewAgentID:  d.ReviewAgentID,

		PipelineVersion: d.PipelineVersion,
		PipelineName:    d.PipelineName,
		PhaseCount:      d.PhaseCount,
		PipelineDefHash: d.PipelineDefHash,
		PipelineDef:     phaseDefsFromDomain(d.PipelineDef),
		CurrentPhase:    d.CurrentPhaseIdx,

		WorkspaceMode:  d.WorkspaceMode,
		WorkspacePath:  d.WorkspacePath,
		BaseWorkingDir: d.BaseWorkingDir,
		BaseBranch:     d.BaseBranch,
		BaseCommit:     d.BaseCommit,

		ReworkCount:          d.ReworkCount,
		ResumeCount:          d.ResumeCount,
		ReviewHints:          d.ReviewHints,
		CommentCursor:        d.CommentCursor,
		DesignSummary:        d.DesignSummary,
		ParentQuestID:        d.ParentQuestID,
		AutoSpawnExecute:     d.AutoSpawnExecute,
		ChildExecuteQuestID:  d.ChildExecuteQuestID,
		SpawnError:           d.SpawnError,
		GroupID:              d.GroupID,
		FanoutLeafID:         d.FanoutLeafID,
		OwnershipScopes:      append([]string(nil), d.OwnershipScopes...),
		MergeStrategy:        d.MergeStrategy,
		MergeOwnerLeafID:     d.MergeOwnerLeafID,
		AcceptanceCriteria:   d.AcceptanceCriteria,
		TriageMode:           d.TriageMode,
		WaitingInput:         waitingInputFromDomain(d.WaitingInput),
		BlockedAtMs:          d.BlockedAtMs,
		AccumulatedBlockedMs: d.AccumulatedBlockedMs,

		FinalVerdict:          d.FinalVerdict,
		FinalComment:          d.FinalComment,
		ImpactSummary:         d.ImpactSummary,
		FinalizedBy:           d.FinalizedBy,
		AutoPassedByPolicy:    d.AutoPassedByPolicy,
		AutoCompletedByPolicy: d.AutoCompletedByPolicy,
		PolicyDecisionID:      d.PolicyDecisionID,
		Applied:               d.Applied,
		ApplyStatus:           d.ApplyStatus,
		ApplyError:            d.ApplyError,
		ApplyFailedAtMs:       d.ApplyFailedAtMs,
		BlockedReason:         d.BlockedReason,
		BlockedReasonCode:     d.BlockedReasonCode,
		BlockedCategory:       d.BlockedCategory,
		MageScore:             d.MageScore,

		DiffStat:             d.DiffStat,
		DiffChangedFiles:     d.DiffChangedFiles,
		DiffAdditions:        d.DiffAdditions,
		DiffDeletions:        d.DiffDeletions,
		WorkspaceCleaned:     d.WorkspaceCleaned,
		WorkspaceCleanedAtMs: d.WorkspaceCleanedAtMs,

		CreatedBy:     d.CreatedBy,
		Official:      d.Official,
		CreatedAtMs:   d.CreatedAtMs,
		UpdatedAtMs:   d.UpdatedAtMs,
		StartedAtMs:   d.StartedAtMs,
		CompletedAtMs: d.CompletedAtMs,
	}

	// 转换 phases
	if len(d.Phases) > 0 {
		q.Phases = make([]PhaseTask, len(d.Phases))
		for i, p := range d.Phases {
			q.Phases[i] = phaseTaskFromDomain(p)
		}
	}

	// 双写兼容
	q.SyncPhaseIDs()

	if d.FailureAttribution != nil {
		q.FailureAttribution = &FailureAttribution{
			Stage:       string(d.FailureAttribution.Stage),
			Reason:      string(d.FailureAttribution.Reason),
			Category:    d.FailureAttribution.Category,
			Message:     d.FailureAttribution.Message,
			Recoverable: d.FailureAttribution.Recoverable,
			Actions:     d.FailureAttribution.Actions,
			Details:     d.FailureAttribution.Details,
		}
	}
	return q
}

// ===== PhaseTask <-> quest.PhaseRun 转换 =====

func phaseTaskToDomain(p PhaseTask) quest.PhaseRun {
	return quest.PhaseRun{
		PhaseIdx:     p.PhaseIdx,
		AdventurerID: p.AdventurerID,
		AgentID:      p.AgentID,
		SessionID:    p.SessionID,
		Status:       p.Status,
		Turns:        p.Turns,
		ReworkCount:  p.ReworkCount,
		StartedAtMs:  p.StartedAtMs,
		EndedAtMs:    p.EndedAtMs,
	}
}

func phaseTasksToDomain(items []PhaseTask) []quest.PhaseRun {
	if len(items) == 0 {
		return nil
	}
	out := make([]quest.PhaseRun, len(items))
	for i, p := range items {
		out[i] = phaseTaskToDomain(p)
	}
	return out
}

func phaseDefsToDomain(items []PhaseTask) []quest.PhaseDef {
	if len(items) == 0 {
		return nil
	}
	out := make([]quest.PhaseDef, len(items))
	for i, p := range items {
		role := phaseRoleToDomain(p.Role)
		out[i] = quest.PhaseDef{
			Index:        p.PhaseIdx,
			Name:         firstNonEmpty(p.Name, p.Role),
			DisplayName:  firstNonEmpty(p.DisplayName, p.Role),
			Role:         role,
			Class:        p.Class,
			Goal:         p.Goal,
			ReadOnly:     p.ReadOnly,
			EndSignal:    firstNonEmpty(p.EndSignal, endSignalForPhaseRole(role)),
			ReworkTo:     p.ReworkTo,
			AllowedTools: append([]string(nil), p.AllowedTools...),
			AgentID:      p.AgentID,
		}
	}
	return out
}

func endSignalForPhaseRole(_ quest.PhaseRole) string {
	return "phase_checkpoint"
}

func phaseRoleToDomain(role string) quest.PhaseRole {
	switch role {
	case "warrior", string(quest.PhaseRoleExecute):
		return quest.PhaseRoleExecute
	case "mage", string(quest.PhaseRoleReview):
		return quest.PhaseRoleReview
	case string(quest.PhaseRoleHuman):
		return quest.PhaseRoleHuman
	default:
		return quest.PhaseRole(role)
	}
}

func phaseTaskFromDomain(p quest.PhaseRun) PhaseTask {
	return PhaseTask{
		PhaseIdx:     p.PhaseIdx,
		Role:         "", // 从 pipeline 配置获取更可靠，这里留空
		Class:        "", // 同上
		AdventurerID: p.AdventurerID,
		AgentID:      p.AgentID,
		SessionID:    p.SessionID,
		Status:       p.Status,
		Turns:        p.Turns,
		ReworkCount:  p.ReworkCount,
		StartedAtMs:  p.StartedAtMs,
		EndedAtMs:    p.EndedAtMs,
	}
}

func phaseTasksFromDomain(items []quest.PhaseRun) []PhaseTask {
	if len(items) == 0 {
		return nil
	}
	out := make([]PhaseTask, len(items))
	for i, p := range items {
		out[i] = phaseTaskFromDomain(p)
	}
	return out
}

func phaseDefsFromDomain(items []quest.PhaseDef) []PhaseTask {
	return PhaseDefsFromDomain(items)
}

func PhaseDefsFromDomain(items []quest.PhaseDef) []PhaseTask {
	if len(items) == 0 {
		return nil
	}
	out := make([]PhaseTask, len(items))
	for i, p := range items {
		out[i] = PhaseTask{
			PhaseIdx:     p.Index,
			Name:         p.Name,
			DisplayName:  p.DisplayName,
			Role:         phaseRoleFromDomain(p.Role),
			Class:        p.Class,
			Goal:         p.Goal,
			ReadOnly:     p.ReadOnly,
			EndSignal:    p.EffectiveEndSignal(),
			ReworkTo:     p.ReworkTo,
			AllowedTools: append([]string(nil), p.AllowedTools...),
			AgentID:      p.AgentID,
			Status:       model.PhasePending,
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func phaseRoleFromDomain(role quest.PhaseRole) string {
	switch role {
	case quest.PhaseRoleExecute:
		return "warrior"
	case quest.PhaseRoleReview:
		return "mage"
	default:
		return string(role)
	}
}

func waitingInputToDomain(w *WaitingInputState) *quest.WaitingInputState {
	if w == nil {
		return nil
	}
	return &quest.WaitingInputState{
		QuestionID:         w.QuestionID,
		QuestionText:       w.QuestionText,
		PhaseIdx:           w.PhaseIdx,
		SessionID:          w.SessionID,
		PhaseType:          w.PhaseType,
		Asker:              w.Asker,
		TimeoutMs:          w.TimeoutMs,
		TimeoutAction:      w.TimeoutAction,
		ResumePhaseIdx:     w.ResumePhaseIdx,
		LastCommentSeq:     w.LastCommentSeq,
		LastAnswerID:       w.LastAnswerID,
		ProcessedAnswerIDs: append([]string(nil), w.ProcessedAnswerIDs...),
		AskedAt:            w.AskedAt,
	}
}

func waitingInputFromDomain(w *quest.WaitingInputState) *WaitingInputState {
	if w == nil {
		return nil
	}
	return &WaitingInputState{
		QuestionID:         w.QuestionID,
		QuestionText:       w.QuestionText,
		PhaseIdx:           w.PhaseIdx,
		SessionID:          w.SessionID,
		PhaseType:          w.PhaseType,
		Asker:              w.Asker,
		TimeoutMs:          w.TimeoutMs,
		TimeoutAction:      w.TimeoutAction,
		ResumePhaseIdx:     w.ResumePhaseIdx,
		LastCommentSeq:     w.LastCommentSeq,
		LastAnswerID:       w.LastAnswerID,
		ProcessedAnswerIDs: append([]string(nil), w.ProcessedAnswerIDs...),
		AskedAt:            w.AskedAt,
	}
}
