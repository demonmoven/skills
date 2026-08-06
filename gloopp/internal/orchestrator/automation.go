package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== Automation 管理 ====================

// ListAutomations 列出所有 automation 配置。
func (e *Engine) ListAutomations() ([]*fsstore.AutomationConfig, error) {
	return e.root.ListAutomations()
}

func (e *Engine) ListAutomationTemplates() ([]*fsstore.AutomationConfig, error) {
	return e.root.ListAutomationTemplates()
}

// GetAutomation 获取单个 automation 配置。
func (e *Engine) GetAutomation(id string) (*fsstore.AutomationConfig, error) {
	return e.root.GetAutomation(id)
}

// SaveAutomation 保存 automation 配置（新建或更新）。
func (e *Engine) SaveAutomation(cfg *fsstore.AutomationConfig) error {
	if cfg != nil && cfg.AutoApply && cfg.AllowL2 {
		return fmt.Errorf("auto_apply 与 allow_l2 不能同时开启")
	}
	return e.root.SaveAutomation(cfg)
}

func (e *Engine) SetAutomationEnabled(id string, enabled bool) (*fsstore.AutomationConfig, error) {
	cfg, err := e.root.GetAutomation(id)
	if err != nil {
		return nil, fmt.Errorf("automation 不存在: %w", err)
	}
	cfg.Enabled = enabled
	if err := e.root.SaveAutomation(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

type AutomationUpdate struct {
	Name             *string
	Description      *string
	Query            *string
	QuestType        *string
	Intensity        *string
	WorkingDir       *string
	WorkspaceMode    *model.WorkspaceMode
	WarriorID        *string
	MageID           *string
	Trigger          *fsstore.AutomationTrigger
	Cron             *string
	Priority         *int
	AutoStart        *bool
	TriageMode       *string
	AutoApply        *bool
	AllowHOTL        *bool
	AllowL2          *bool
	WithDesignPhase  *bool
	AutoSpawnExecute *bool
	TrustTier        *string
	TrustTierLocked  *bool
	Connectors       *[]string
}

func (e *Engine) UpdateAutomation(id string, upd AutomationUpdate) (*fsstore.AutomationConfig, error) {
	cfg, err := e.root.GetAutomation(id)
	if err != nil {
		return nil, fmt.Errorf("automation 不存在: %w", err)
	}
	if upd.Name != nil {
		cfg.Name = *upd.Name
	}
	if upd.Description != nil {
		cfg.Description = *upd.Description
	}
	if upd.Query != nil {
		cfg.Query = *upd.Query
	}
	if upd.QuestType != nil {
		qt := model.QuestType(*upd.QuestType)
		if qt != model.QuestTypeExecute && qt != model.QuestTypeDesign {
			return nil, fmt.Errorf("quest_type 必须是 execute 或 design")
		}
		cfg.QuestType = string(qt)
	}
	if upd.Intensity != nil {
		qi := model.QuestIntensity(*upd.Intensity)
		if _, err := questIntensityBudget(e.cfg, qi); err != nil {
			return nil, err
		}
		cfg.Intensity = string(qi)
	}
	if upd.WorkingDir != nil {
		cfg.WorkingDir = *upd.WorkingDir
	}
	if upd.WorkspaceMode != nil {
		cfg.WorkspaceMode = string(*upd.WorkspaceMode)
	}
	if upd.WarriorID != nil {
		cfg.WarriorID = *upd.WarriorID
	}
	if upd.MageID != nil {
		cfg.MageID = *upd.MageID
	}
	if upd.Trigger != nil {
		if *upd.Trigger != fsstore.TriggerManual && *upd.Trigger != fsstore.TriggerSchedule {
			return nil, fmt.Errorf("trigger 必须是 manual 或 schedule")
		}
		cfg.Trigger = *upd.Trigger
	}
	if upd.Cron != nil {
		cfg.Cron = fsstore.NormalizeCronPreset(*upd.Cron)
	}
	if upd.Priority != nil {
		cfg.Priority = *upd.Priority
	}
	if upd.AutoStart != nil {
		cfg.AutoStart = *upd.AutoStart
		cfg.RuntimePolicyUserOverride = true
	}
	if upd.TriageMode != nil {
		switch *upd.TriageMode {
		case "", "candidate", "direct":
			cfg.TriageMode = *upd.TriageMode
		default:
			return nil, fmt.Errorf("triage_mode 必须是 candidate 或 direct")
		}
	}
	if upd.AutoApply != nil {
		cfg.AutoApply = *upd.AutoApply
		cfg.RuntimePolicyUserOverride = true
	}
	if upd.AllowHOTL != nil {
		cfg.RuntimePolicyUserOverride = true
	}
	if upd.AllowL2 != nil {
		cfg.AllowL2 = *upd.AllowL2
		cfg.RuntimePolicyUserOverride = true
	}
	if upd.WithDesignPhase != nil {
		cfg.WithDesignPhase = *upd.WithDesignPhase
	}
	if upd.AutoSpawnExecute != nil {
		cfg.AutoSpawnExecute = *upd.AutoSpawnExecute
	}
	if upd.TrustTier != nil {
		if !fsstore.IsValidTrustTier(*upd.TrustTier) {
			return nil, fmt.Errorf("trust_tier 必须是 tier_0 | tier_1 | tier_2 | tier_3")
		}
		cfg.TrustTier = *upd.TrustTier
	}
	if upd.TrustTierLocked != nil {
		cfg.TrustTierLocked = *upd.TrustTierLocked
	}
	if upd.Connectors != nil {
		cfg.Connectors = *upd.Connectors
	}
	if cfg.AutoApply && cfg.AllowL2 {
		return nil, fmt.Errorf("auto_apply 与 allow_l2 不能同时开启")
	}
	if err := e.root.SaveAutomation(cfg); err != nil {
		return nil, err
	}
	if cfg.TrustTier != "" && cfg.TrustTierLocked {
		_, _ = e.root.LockAutomationTrustTier(cfg.ID, cfg.TrustTier, "manual update")
	}
	return cfg, nil
}

// DeleteAutomation 删除 automation 配置。
func (e *Engine) DeleteAutomation(id string) error {
	return e.root.DeleteAutomation(id)
}

func (e *Engine) recordAutomationTrustOutcome(q *fsstore.QuestMeta, outcome, verificationType string, independent bool) {
	e.recordAutomationTrustOutcomeTyped(q, outcome, verificationType, independent, fsstore.TrustOutcomeTypeTerminal)
}

// recordAutomationTrustStageOutcome 记录一条 stage 级（非终态）信任事件。
// 用于 user_review 超时这类"非质量失败"信号：既不计入 ConsecutiveFailures
// （不是 automation 的错），也不占用 questID:terminal 去重槽（后续若恢复到
// 真正终态仍可记录 terminal outcome），只让 Recent outcomes 反映这件事发生过。
func (e *Engine) recordAutomationTrustStageOutcome(q *fsstore.QuestMeta, outcome, verificationType string) {
	e.recordAutomationTrustOutcomeTyped(q, outcome, verificationType, true, fsstore.TrustOutcomeTypeStage)
}

func (e *Engine) recordAutomationTrustOutcomeTyped(q *fsstore.QuestMeta, outcome, verificationType string, independent bool, outcomeType string) {
	if q == nil || q.Source() != model.SourceAutomation {
		return
	}
	autoID := q.AutomationID()
	if autoID == "" {
		return
	}
	state, err := e.root.EnsureAutomationTrustState(autoID)
	if err != nil {
		e.log.Warn("加载 automation trust state 失败", "automation", autoID, "err", err)
		return
	}
	record := fsstore.AutomationTrustOutcome{
		QuestID:          q.ID,
		OutcomeType:      outcomeType,
		Outcome:          outcome,
		VerificationType: verificationType,
		Independent:      independent,
		TsMs:             fsstore.NowMs(),
	}
	if !state.AppendOutcome(record) {
		e.log.Debug("automation trust outcome 已存在，跳过重复统计", "automation", autoID, "qid", q.ID, "verification_type", verificationType)
		return
	}
	if err := e.root.SaveAutomationTrustState(state); err != nil {
		e.log.Warn("保存 automation trust state 失败", "automation", autoID, "err", err)
		return
	}
	e.publish(q.ID, "", events.EventType("automation.trust"), map[string]any{
		"automation":        autoID,
		"tier":              state.Tier,
		"locked":            state.Locked,
		"outcome":           outcome,
		"verification_type": verificationType,
		"independent":       independent,
		"outcome_type":      outcomeType,
	})
}

func (e *Engine) settleAutomationTrustForEvent(qid string, typ events.EventType) {
	switch typ {
	case events.EvtQuestSuccess, events.EvtQuestFailed, events.EvtQuestCancelled, events.EvtQuestApplyFailed, events.EvtQuestBlocked:
	default:
		return
	}
	q, err := fsstore.NewQuestStore(e.root).LoadQuest(qid)
	if err != nil || q == nil || q.Source() != model.SourceAutomation {
		return
	}
	switch typ {
	case events.EvtQuestSuccess:
		// 事件由 questService 同步发出，先于 macro_loop/quest_lifecycle 里的显式
		// recordAutomationTrustOutcome 调用。这里给出具体 verification_type，使
		// 得胜出去重的那条 outcome 已带最细标签（显式调用之后会被安全 dedup）。
		e.recordAutomationTrustOutcome(q, "success", trustVerificationTypeForSuccess(q), false)
	case events.EvtQuestFailed:
		e.recordAutomationTrustOutcome(q, "failure", trustVerificationTypeForFailure(q), true)
	case events.EvtQuestCancelled:
		e.recordAutomationTrustOutcome(q, "failure", "quest_cancelled", true)
	case events.EvtQuestApplyFailed:
		e.recordAutomationTrustOutcome(q, "failure", "apply_failed", true)
	case events.EvtQuestBlocked:
		// 仅 user_review 超时记 stage 信号——它是终审未到场的"非质量失败"，
		// 不计 ConsecutiveFailures，也不占 terminal 去重槽。其余 block 原因
		// （agent 错误/预算超支等）后续会走向 retry 或真正的终态，由对应
		// 事件记录 terminal outcome，这里不重复记 stage 噪音。
		if q.BlockedReasonCode == "user_confirm_timeout" {
			e.recordAutomationTrustStageOutcome(q, "user_review_timeout", "user_review_timeout")
		}
	}
}

// trustVerificationTypeForSuccess 根据 quest 终态来源给出具体 verification_type。
// 之前所有成功都落 "quest_success"，丢失了 human_approved / policy_auto_complete
// 等标签（显式调用被同步事件的 dedup 吞掉）。
func trustVerificationTypeForSuccess(q *fsstore.QuestMeta) string {
	switch q.FinalizedBy {
	case "user":
		return "human_approved"
	case "policy":
		if q.AutoPassedByPolicy != "" {
			return "policy_auto_pass"
		}
		return "policy_auto_complete"
	}
	return "quest_success"
}

// trustVerificationTypeForFailure 区分失败原因：
//   - human_rejected：用户终审拒绝（FinalizedBy=user 且 FinalVerdict=reject）
//   - mage_rejected：法师评审拒绝（failQuest 带"法师评审拒绝"理由）
//   - 其余按缓存的 FailureAttribution.Reason，兜底 quest_failed（操作性失败）
//
// 这避免 mage 质量拒绝与 agent 崩溃都落同一 "quest.failed"，无法区分——
// 质量信号和操作性失败对信任分级权重应不同。
func trustVerificationTypeForFailure(q *fsstore.QuestMeta) string {
	if q.FinalizedBy == "user" && q.FinalVerdict == model.VerdictReject {
		return "human_rejected"
	}
	if strings.Contains(q.FinalComment, "法师评审拒绝") {
		return "mage_rejected"
	}
	if q.FailureAttribution != nil && q.FailureAttribution.Reason != "" {
		return "quest_failed:" + q.FailureAttribution.Reason
	}
	return "quest_failed"
}

// RunAutomation 触发一个 automation，创建 quest。
// 返回创建的 quest id。
func (e *Engine) RunAutomation(ctx context.Context, id string) (string, error) {
	cfg, err := e.root.GetAutomation(id)
	if err != nil {
		return "", fmt.Errorf("automation 不存在: %w", err)
	}
	if !cfg.Enabled {
		return "", fmt.Errorf("automation 已禁用: %s", id)
	}
	if cfg.HasTag("triage") {
		if archived, reason := e.archiveNoFindingTriageIfEmpty(cfg); archived {
			e.root.MarkAutomationRun(id)
			e.publishSystem(events.EventType("automation.discovery_archived"), map[string]any{
				"automation": cfg.ID,
				"outcome":    "no_finding",
				"reason":     reason,
			})
			// v0.5.4: no_finding 创建轻量 quest 进 Feed，让 automation 像账号发"无发现"动态。
			// 不启动执行（RequireAgent=false），只记录扫描结果。
			workDir := cfg.WorkingDir
			if workDir == "" {
				workDir = e.workDir
			}
			noFindingQuery := fmt.Sprintf("Automation %s 扫描完成：无发现（%s）", cfg.Name, reason)
			q, qErr := e.CreateQuest(ctx, noFindingQuery, model.QuestTypeDesign, workDir, CreateQuestOptions{
				CreatedBy:     model.QuestSourcePrefix + cfg.ID,
				WorkspaceMode: model.WorkspaceReadOnly,
				RequireAgent:  false,
			})
			if qErr == nil {
				e.projectAutomationNoFindingPost(q, cfg, reason)
				return q.ID, nil
			}
			e.log.Warn("no_finding quest 创建失败，仅归档", "automation", cfg.ID, "err", qErr)
			return "", nil
		}
	}

	// Fan-out: 多 quest 并行
	if len(cfg.FanOut) > 0 {
		return e.runAutomationFanOut(ctx, cfg)
	}

	// 确定工作目录
	workDir := cfg.WorkingDir
	if workDir == "" {
		workDir = e.workDir
	}

	// 创建 quest
	questType := model.QuestType(cfg.QuestType)
	if questType == "" {
		questType = model.QuestTypeExecute
	}
	query := cfg.Query
	if cfg.HasTag("audit") {
		query = e.buildAuditQuery(query)
	}
	workspaceMode := model.WorkspaceMode(cfg.WorkspaceMode)
	if workspaceMode == "auto" {
		workspaceMode = ""
	}
	if workspaceMode == "" && (cfg.HasTag("context") || cfg.HasTag("knowledge")) {
		workspaceMode = model.WorkspaceReadOnly
	}
	triageMode := fsstore.AutomationTriageMode(cfg)
	q, err := e.CreateQuest(ctx, query, questType, workDir, CreateQuestOptions{
		WarriorID:          cfg.WarriorID,
		MageID:             cfg.MageID,
		WorkspaceMode:      workspaceMode,
		Intensity:          model.QuestIntensity(cfg.Intensity),
		CreatedBy:          model.QuestSourcePrefix + cfg.ID,
		TriageMode:         triageMode,
		RequireAgent:       true,
		AcceptanceCriteria: cfg.AcceptanceCriteria,
		WithDesignPhase:    cfg.WithDesignPhase,
		AutoSpawnExecute:   cfg.AutoSpawnExecute,
	})
	if err != nil {
		return "", fmt.Errorf("创建委托失败: %w", err)
	}

	qs := fsstore.NewQuestStore(e.root)
	qMeta, err := qs.LoadQuest(q.ID)
	if err != nil || qMeta == nil {
		qMeta = q
	}

	// 如果 auto_start，启动执行
	if cfg.AutoStart {
		if err := e.StartQuest(ctx, q.ID); err != nil {
			return q.ID, fmt.Errorf("启动委托失败: %w", err)
		}
	}

	// 更新 automation 运行统计
	e.root.MarkAutomationRun(id)

	automationRunPayload := map[string]any{
		"automation":  cfg.ID,
		"name":        cfg.Name,
		"qid":         q.ID,
		"auto_start":  cfg.AutoStart,
		"auto_apply":  cfg.AutoApply,
		"triage_mode": triageMode,
	}
	automationRunEvent, err := e.recordQuestEventRequired(q.ID, "", events.EventType("automation.run"), automationRunPayload)
	if err != nil {
		return "", fmt.Errorf("写入 automation.run event 失败: %w", err)
	}
	e.projectAutomationRunThreadPost(qMeta, cfg, automationRunEvent.ID, triageMode)
	e.publishRecordedEvent(automationRunEvent)

	e.publish(q.ID, "", events.EvtQuestNote, map[string]any{
		"note":        "由 automation 创建: " + cfg.Name,
		"automation":  cfg.ID,
		"auto_start":  cfg.AutoStart,
		"auto_apply":  cfg.AutoApply,
		"triage_mode": triageMode,
	})

	return q.ID, nil
}

func (e *Engine) projectAutomationRunThreadPost(q *fsstore.QuestMeta, cfg *fsstore.AutomationConfig, sourceEventID int64, triageMode string) {
	if q == nil || cfg == nil || strings.TrimSpace(q.ID) == "" {
		return
	}
	if strings.TrimSpace(q.CreatedBy) == "" {
		q.CreatedBy = model.QuestSourcePrefix + cfg.ID
	}
	content := fmt.Sprintf("Automation %s created this quest", cfg.Name)
	if strings.TrimSpace(triageMode) != "" {
		content += " | triage_mode=" + strings.TrimSpace(triageMode)
	}
	if cfg.AutoStart {
		content += " | auto_start=true"
	}
	if cfg.AutoApply {
		content += " | auto_apply=true"
	}
	if _, err := fsstore.NewQuestStore(e.root).AppendThreadPost(q.ID, fsstore.AppendThreadPostOptions{
		PostID:         fmt.Sprintf("auto_run_%s_%s", q.ID, cfg.ID),
		ThreadID:       q.ID,
		ParentReplyID:  fsstore.RootPostID(q.ID),
		RootPostID:     fsstore.RootPostID(q.ID),
		CausalRefs:     []string{fsstore.RootPostID(q.ID)},
		AuthorIdentity: model.QuestSourcePrefix + cfg.ID,
		AuthorRole:     model.PostRoleAutomation,
		SourceEventID:  sourceEventID,
		Kind:           "automation_run",
		Content:        content,
	}); err != nil {
		e.log.Warn("automation run thread post 投影失败", "qid", q.ID, "automation", cfg.ID, "err", err)
	}
}

// projectAutomationNoFindingPost 把 no_finding 结果投影成 automation reply，进 Feed。
// 不伪造 outcome：基于 archiveNoFindingTriageIfEmpty 的真实归档事实。
func (e *Engine) projectAutomationNoFindingPost(q *fsstore.QuestMeta, cfg *fsstore.AutomationConfig, reason string) {
	if q == nil || cfg == nil || strings.TrimSpace(q.ID) == "" {
		return
	}
	content := fmt.Sprintf("Automation %s 扫描完成：无发现。%s", cfg.Name, reason)
	payload := map[string]any{
		"automation_id": cfg.ID,
		"automation":    cfg.Name,
		"reason":        reason,
	}
	ev, evErr := e.recordQuestEventRequired(q.ID, "", events.EvtQuestNote, payload)
	if evErr != nil {
		e.log.Warn("automation no_finding: record event failed", "qid", q.ID, "automation", cfg.ID, "err", evErr)
		return
	}
	if _, err := fsstore.NewQuestStore(e.root).AppendThreadPost(q.ID, fsstore.AppendThreadPostOptions{
		PostID:         fmt.Sprintf("auto_no_finding_%d", ev.ID),
		ThreadID:       q.ID,
		ParentReplyID:  fsstore.RootPostID(q.ID),
		RootPostID:     fsstore.RootPostID(q.ID),
		CausalRefs:     []string{fsstore.RootPostID(q.ID)},
		AuthorIdentity: model.QuestSourcePrefix + cfg.ID,
		AuthorRole:     model.PostRoleAutomation,
		SourceEventID:  ev.ID,
		Kind:           "automation_no_finding",
		Content:        content,
	}); err != nil {
		e.log.Warn("automation no_finding thread post 投影失败", "qid", q.ID, "automation", cfg.ID, "err", err)
	}
}

func (e *Engine) archiveNoFindingTriageIfEmpty(cfg *fsstore.AutomationConfig) (bool, string) {
	if cfg == nil || !cfg.HasTag("triage") {
		return false, ""
	}
	inbox, err := e.root.ListInbox()
	if err != nil {
		return false, ""
	}
	quests, err := e.ListQuests()
	if err != nil {
		return false, ""
	}
	for _, q := range quests {
		if fsstore.HumanExceptionFromQuest(q) != nil {
			return false, ""
		}
	}
	if len(inbox) > 0 {
		return false, ""
	}
	reason := "no candidates or human exceptions"
	if err := e.root.ArchiveAutomationDiscovery(&fsstore.AutomationDiscoveryArchive{
		AutomationID: cfg.ID,
		Outcome:      "no_finding",
		Reason:       reason,
		Summary:      "triage scan found no actionable items",
	}); err != nil {
		e.log.Warn("automation no-finding 归档失败", "automation", cfg.ID, "err", err)
		return false, ""
	}
	return true, reason
}

// runAutomationFanOut 从 FanOut 规格创建 N 个并行 quest，共享 GroupID。
func (e *Engine) runAutomationFanOut(ctx context.Context, cfg *fsstore.AutomationConfig) (string, error) {
	groupID := fmt.Sprintf("grp_%s_%08x", cfg.ID, time.Now().UnixNano()&0xFFFFFFFF)

	workDir := cfg.WorkingDir
	if workDir == "" {
		workDir = e.workDir
	}
	workspaceMode := model.WorkspaceMode(cfg.WorkspaceMode)
	if workspaceMode == "auto" {
		workspaceMode = ""
	}

	var qids []string
	contracts := make([]fsstore.FanoutContract, 0, len(cfg.FanOut))
	for _, spec := range cfg.FanOut {
		contracts = append(contracts, fsstore.FanoutContract{
			LeafID:           spec.LeafID,
			OwnershipScopes:  spec.OwnershipScopes,
			MergeStrategy:    spec.MergeStrategy,
			MergeOwnerLeafID: spec.MergeOwnerLeafID,
		})
	}
	if err := fsstore.ValidateFanoutBatch(contracts); err != nil {
		return "", err
	}
	for _, spec := range cfg.FanOut {
		warriorID := spec.WarriorID
		if warriorID == "" {
			warriorID = cfg.WarriorID
		}
		mageID := spec.MageID
		if mageID == "" {
			mageID = cfg.MageID
		}

		q, err := e.CreateQuest(ctx, spec.Query, model.QuestType(cfg.QuestType), workDir, CreateQuestOptions{
			WarriorID:     warriorID,
			MageID:        mageID,
			WorkspaceMode: workspaceMode,
			Intensity:     model.QuestIntensity(cfg.Intensity),
			CreatedBy:     model.QuestSourcePrefix + cfg.ID,
			TriageMode:    fsstore.AutomationTriageMode(cfg),
			RequireAgent:  true,
			GroupID:       groupID,
			FanoutContract: fsstore.FanoutContract{
				LeafID:           spec.LeafID,
				OwnershipScopes:  spec.OwnershipScopes,
				MergeStrategy:    spec.MergeStrategy,
				MergeOwnerLeafID: spec.MergeOwnerLeafID,
			},
			AcceptanceCriteria: cfg.AcceptanceCriteria,
		})
		if err != nil {
			return "", fmt.Errorf("fan-out quest 创建失败: %w", err)
		}

		if cfg.AutoStart {
			if err := e.StartQuest(ctx, q.ID); err != nil {
				e.log.Warn("fan-out quest 启动失败", "qid", q.ID, "err", err)
			}
		}
		qids = append(qids, q.ID)
	}

	e.root.MarkAutomationRun(cfg.ID)

	e.publish("", "", events.EventType("automation.fan_out"), map[string]any{
		"automation":      cfg.ID,
		"group_id":        groupID,
		"quest_ids":       qids,
		"count":           len(qids),
		"fanout_contract": true,
	})

	if len(qids) > 0 {
		return qids[0], nil
	}
	return "", nil
}

func (e *Engine) buildAuditQuery(base string) string {
	qs, err := e.ListQuests()
	if err != nil {
		return base
	}
	lines := []string{base, "", "## 最近完成委托"}
	count := 0
	for _, q := range qs {
		if q.Status != model.QuestStatusSuccess {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s [%s] %s | verdict=%s | comment=%s", q.ID, q.Type, q.Query, q.FinalVerdict, q.FinalComment))
		count++
		if count >= 10 {
			break
		}
	}
	if count == 0 {
		lines = append(lines, "- 暂无已完成委托；请说明当前没有可审计对象，并给出后续审计建议。")
	}
	return strings.Join(lines, "\n")
}

// ==================== Inbox ====================

// ListInbox 列出收件箱中的委托（来自 automation、待用户确认）。
func (e *Engine) ListInbox() ([]*fsstore.QuestMeta, error) {
	return e.root.ListInbox()
}

func (e *Engine) RecoverAutoStartInboxItems(ctx context.Context) (recovered int, failed int, total int) {
	items, err := e.root.ListInbox()
	if err != nil {
		e.log.Error("恢复 auto-start inbox：列表加载失败", "err", err)
		return 0, 0, 0
	}
	for _, q := range items {
		if q == nil || q.Status != model.QuestStatusPending || q.Source() != model.SourceAutomation {
			continue
		}
		cfg, err := e.root.GetAutomation(q.AutomationID())
		if err != nil || !fsstore.AutomationShouldAutoStartByOfficialPolicy(cfg) {
			continue
		}
		total++
		e.log.Info("恢复 auto-start inbox 委托", "qid", q.ID, "automation", cfg.ID)
		if err := e.StartQuest(ctx, q.ID); err != nil {
			e.log.Warn("恢复 auto-start inbox 委托失败", "qid", q.ID, "automation", cfg.ID, "err", err)
			failed++
			continue
		}
		recovered++
	}
	return recovered, failed, total
}

type InboxUpdate struct {
	Query         *string
	QuestType     *string
	WorkDir       *string
	WorkspaceMode *model.WorkspaceMode
}

func (e *Engine) UpdateInboxItem(qid string, upd InboxUpdate) (*fsstore.QuestMeta, error) {
	qs := fsstore.NewQuestStore(e.root)
	q, err := qs.LoadQuest(qid)
	if err != nil {
		return nil, fmt.Errorf("加载委托失败: %w", err)
	}
	if q.Status != model.QuestStatusPending || q.Source() != model.SourceAutomation {
		return nil, fmt.Errorf("委托 %s 不在 inbox 中", qid)
	}
	if upd.Query != nil {
		if *upd.Query == "" {
			return nil, fmt.Errorf("query 不能为空")
		}
		q.Query = *upd.Query
	}
	if upd.QuestType != nil {
		qt := model.QuestType(*upd.QuestType)
		if qt != model.QuestTypeExecute && qt != model.QuestTypeDesign {
			return nil, fmt.Errorf("type 必须是 execute 或 design")
		}
		q.Type = qt
	}
	if upd.WorkDir != nil {
		q.BaseWorkingDir = *upd.WorkDir
	}
	if upd.WorkspaceMode != nil {
		q.WorkspaceMode = *upd.WorkspaceMode
	}
	if err := qs.SaveQuest(q); err != nil {
		return nil, fmt.Errorf("保存委托失败: %w", err)
	}
	e.publish(qid, "", events.EvtQuestNote, map[string]any{
		"note": "inbox item updated",
	})
	return q, nil
}

// hasAutoApply 检查 quest 是否来自开启了 auto_apply 的 automation。
func (e *Engine) hasAutoApply(q *fsstore.QuestMeta) bool {
	if q.Source() != model.SourceAutomation {
		return false
	}
	autoID := strings.TrimPrefix(q.CreatedBy, model.QuestSourcePrefix)
	cfg, err := e.root.GetAutomation(autoID)
	if err != nil {
		return false
	}
	return cfg.AutoApply
}

func (e *Engine) isContextAutomationQuest(q *fsstore.QuestMeta) bool {
	if q == nil || q.Source() != model.SourceAutomation {
		return false
	}
	autoID := strings.TrimPrefix(q.CreatedBy, model.QuestSourcePrefix)
	if autoID == "auto_context_refresh" {
		return true
	}
	cfg, err := e.root.GetAutomation(autoID)
	return err == nil && cfg != nil && cfg.HasTag("context")
}

func (e *Engine) l2Allowed(q *fsstore.QuestMeta) bool {
	if q == nil || q.Source() != model.SourceAutomation {
		return false
	}
	autoID := strings.TrimPrefix(q.CreatedBy, model.QuestSourcePrefix)
	cfg, err := e.root.GetAutomation(autoID)
	return err == nil && cfg.AllowL2 && !cfg.AutoApply
}

// AcceptInboxItem 接受收件箱里的委托（启动执行）。
func (e *Engine) AcceptInboxItem(ctx context.Context, qid string) error {
	qs := fsstore.NewQuestStore(e.root)
	q, err := qs.LoadQuest(qid)
	if err != nil {
		return fmt.Errorf("加载委托失败: %w", err)
	}
	if q.Status != model.QuestStatusPending {
		return fmt.Errorf("委托状态 %s，不在收件箱中", q.Status)
	}
	return e.StartQuest(ctx, qid)
}

// RejectInboxItem 拒绝收件箱里的委托（取消并标记）。
// 状态变更通过 QuestService 执行。
func (e *Engine) RejectInboxItem(qid, reason string) error {
	comment := "收件箱拒绝: " + reason
	_, err := e.questService.CancelQuest(qid, comment)
	if err != nil {
		return fmt.Errorf("取消委托失败: %w", err)
	}
	e.cleanupWorkspaceIfTerminal(qid)
	return nil
}
