package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/domain/quest"
	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/policy"
)

// ==================== Quest 生命周期 API ====================

// CreateQuest 创建新委托。
type CreateQuestOptions struct {
	WarriorID          string
	MageID             string
	ExecuteAgentID     string
	ReviewAgentID      string
	WorkspaceMode      model.WorkspaceMode
	Intensity          model.QuestIntensity
	WorkflowMode       model.WorkflowMode
	CreatedBy          string
	TriageMode         string
	RequireAgent       bool
	Inputs             []fsstore.QuestArtifact
	GroupID            string
	FanoutContract     fsstore.FanoutContract
	AcceptanceCriteria string
	WithDesignPhase    bool
	SkipReview         bool
	AutoSpawnExecute   bool
	Connectors         []string // HOTL v0.2 环 4: quest 自主闭环后触发的 connector
}

func (e *Engine) CreateQuest(ctx context.Context, query string, questType model.QuestType, workDir string, opts ...CreateQuestOptions) (*fsstore.QuestMeta, error) {
	if query == "" {
		return nil, errors.New("query 不能为空")
	}
	if questType == "" {
		questType = model.QuestTypeExecute
	}
	intensity := model.QuestIntensityStandard
	if len(opts) > 0 && opts[0].Intensity != "" {
		intensity = opts[0].Intensity
	}
	budget, err := questIntensityBudget(e.cfg, intensity)
	if err != nil {
		return nil, err
	}

	qs := fsstore.NewQuestStore(e.root)
	qid, short := qs.NewQuestID()

	wd := workDir
	if wd == "" {
		wd = e.workDir
	}

	workspaceMode := model.WorkspaceMode(e.cfg.DefaultWorkspaceMode)
	if workspaceMode == "auto" {
		workspaceMode = ""
	}

	q := &fsstore.QuestMeta{
		ID:              qid,
		ShortID:         short,
		Query:           query,
		Type:            questType,
		Status:          model.QuestStatusPending,
		Inputs:          nil,
		OriginalRequest: query,
		IntentSummary:   intentSummaryFallback(query),
		Intensity:       intensity,
		WorkspaceMode:   workspaceMode,
		BaseWorkingDir:  wd,
		ReworkCount:     0,
		MaxRework:       budget.MaxRework,
		CreatedBy:       string(model.SourceUser),
		CreatedAtMs:     fsstore.NowMs(),
	}
	if budget.OverrideQuestBudget {
		q.MaxTurnsPerPhaseOverride = budget.MaxTurnsPerPhase
		q.MaxDurationPerQuestMsOverride = budget.MaxDurationPerQuestMs
	}
	if len(opts) > 0 {
		if opts[0].CreatedBy != "" {
			q.CreatedBy = opts[0].CreatedBy
		}
		if opts[0].TriageMode != "" {
			q.TriageMode = opts[0].TriageMode
		}
		q.WarriorID = opts[0].WarriorID
		q.MageID = opts[0].MageID
		q.ExecuteAgentID = opts[0].ExecuteAgentID
		q.ReviewAgentID = opts[0].ReviewAgentID
		q.WorkflowMode = opts[0].WorkflowMode
		q.GroupID = opts[0].GroupID
		q.ApplyFanoutContract(opts[0].FanoutContract)
		q.AcceptanceCriteria = opts[0].AcceptanceCriteria
		q.AutoSpawnExecute = opts[0].AutoSpawnExecute
		if len(opts[0].Connectors) > 0 {
			q.Connectors = append([]string{}, opts[0].Connectors...)
		}
		if opts[0].WithDesignPhase {
			q.PipelineName = "design"
			q.PipelineDef = fsstore.PhaseDefsFromDomain(quest.DesignPipeline())
			q.PhaseCount = len(q.PipelineDef)
			q.PipelineDefHash = ""
			q.Phases = nil
		}
		if opts[0].SkipReview && !opts[0].WithDesignPhase {
			q.PipelineName = "run"
			q.PipelineDef = fsstore.PhaseDefsFromDomain(quest.DefaultPipeline()[:1])
			q.PhaseCount = 1
			q.MageID = ""
			q.PipelineDefHash = ""
			q.Phases = nil
		}
		if len(opts[0].Inputs) > 0 {
			q.Inputs = append([]fsstore.QuestArtifact{}, opts[0].Inputs...)
		}
		if opts[0].WorkspaceMode != "" {
			q.WorkspaceMode = opts[0].WorkspaceMode
		}
		if opts[0].RequireAgent {
			if q.ExecuteAgentID == "" {
				warrior, err := e.pickRunnableAdventurer(q.WarriorID, model.ClassWarrior)
				if err != nil {
					return nil, fmt.Errorf("选剑士失败: %w", err)
				}
				q.WarriorID = warrior.ID
			}
			// quick 强度跳过法师评审，不分配法师
			if intensity != model.QuestIntensityQuick && q.ReviewAgentID == "" {
				mage, err := e.pickRunnableAdventurer(q.MageID, model.ClassMage)
				if err != nil {
					return nil, fmt.Errorf("选法师失败: %w", err)
				}
				q.MageID = mage.ID
			}
		}
	}
	if q.WorkflowMode == "" {
		q.WorkflowMode = model.WorkflowModeFromQuest(q.Type, q.PipelineName == "run")
	}
	if err := fsstore.ValidateFanoutContract(q.FanoutContract()); err != nil {
		return nil, err
	}
	if err := qs.ValidateFanoutSpawn(q.GroupID, q.FanoutContract()); err != nil {
		return nil, err
	}

	// 记录 prompt 版本快照（quest 归因用）
	if e.templates != nil {
		promptVer := e.templates.Version()
		q.PromptVersion = promptVer.Short()
		if len(promptVer.Overrides) > 0 {
			q.PromptOverrides = promptVer.Overrides
			q.PromptOverrideMap = promptVer.OverrideHashes
		}
	}

	if err := qs.CreateQuest(q); err != nil {
		return nil, fmt.Errorf("创建 quest 失败: %w", err)
	}

	createdPayload := map[string]any{
		"qid":           qid,
		"short_id":      short,
		"query":         query,
		"type":          questType,
		"intensity":     intensity,
		"root_post_id":  fsstore.RootPostID(qid),
		"workflow_mode": q.WorkflowMode,
	}
	createdEv, err := e.recordQuestEventRequired(qid, "", events.EvtQuestCreated, createdPayload)
	if err != nil {
		return nil, fmt.Errorf("记录 quest.created event 失败: %w", err)
	}

	rootContent := q.OriginalRequest
	if strings.TrimSpace(rootContent) == "" {
		rootContent = q.Query
	}
	if strings.TrimSpace(rootContent) == "" {
		rootContent = "Quest " + qid
	}
	rootAuthorRole := model.PostRoleHuman
	if q.Source() == model.SourceAutomation {
		rootAuthorRole = model.PostRoleAutomation
	}
	if _, err := qs.AppendThreadPost(qid, fsstore.AppendThreadPostOptions{
		PostID:         fsstore.RootPostID(qid),
		ThreadID:       qid,
		RootPostID:     fsstore.RootPostID(qid),
		AuthorIdentity: q.CreatedBy,
		AuthorRole:     rootAuthorRole,
		Kind:           "root",
		Content:        rootContent,
		SourceEventID:  createdEv.ID,
		CreatedAtMs:    q.CreatedAtMs,
	}); err != nil {
		return nil, fmt.Errorf("创建 root post 失败: %w", err)
	}
	e.publishRecordedEvent(createdEv)
	return q, nil
}

func intentSummaryFallback(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}
	line := query
	if idx := strings.IndexAny(line, "\r\n"); idx >= 0 {
		line = strings.TrimSpace(line[:idx])
	}
	runes := []rune(line)
	if len(runes) > 80 {
		return string(runes[:80]) + "..."
	}
	return line
}

type intensityBudget struct {
	MaxTurnsPerPhase      int
	MaxRework             int
	MaxDurationPerQuestMs int64
	OverrideQuestBudget   bool
}

func questIntensityBudget(cfg *fsstore.GlobalConfig, intensity model.QuestIntensity) (intensityBudget, error) {
	switch intensity {
	case "", model.QuestIntensityStandard:
		return intensityBudget{MaxRework: cfg.MaxReworkPerQuest}, nil
	case model.QuestIntensityQuick:
		return intensityBudget{MaxTurnsPerPhase: 10, MaxRework: 1, MaxDurationPerQuestMs: 30 * 60 * 1000, OverrideQuestBudget: true}, nil
	case model.QuestIntensityDeep:
		return intensityBudget{MaxTurnsPerPhase: 80, MaxRework: 4, MaxDurationPerQuestMs: 8 * 60 * 60 * 1000, OverrideQuestBudget: true}, nil
	case model.QuestIntensityAdversarial:
		return intensityBudget{MaxTurnsPerPhase: 100, MaxRework: 5, MaxDurationPerQuestMs: 12 * 60 * 60 * 1000, OverrideQuestBudget: true}, nil
	default:
		return intensityBudget{}, fmt.Errorf("intensity 必须是 quick | standard | deep | adversarial")
	}
}

// SpawnQuest 创建并启动一个独立委托（fan-out 扇出）。
func (e *Engine) SpawnQuest(ctx context.Context, query, groupID, workDir string, parentQuestID string, opts ...fsstore.FanoutContract) (string, error) {
	if groupID == "" {
		groupID = fmt.Sprintf("grp_%08x", time.Now().UnixNano()&0xFFFFFFFF)
	}
	var contract fsstore.FanoutContract
	if len(opts) > 0 {
		contract = opts[0]
	}

	// Fanout root identity: first spawn from a normal quest promotes parent to root.
	var rootQuestID string
	var isFirstSpawn bool // true when this spawn promotes parent to root (opens group)
	qs := fsstore.NewQuestStore(e.root)
	if parentQuestID != "" {
		parent, err := e.GetQuest(parentQuestID)
		if err != nil {
			return "", fmt.Errorf("fanout: 获取父委托失败: %w", err)
		}
		if parent == nil {
			return "", fmt.Errorf("fanout: 父委托不存在: %s", parentQuestID)
		}
		if parent.GroupID == "" {
			// First spawn: promote parent to fanout root.
			parent.GroupID = groupID
			if err := qs.SaveQuest(parent); err != nil {
				return "", fmt.Errorf("fanout: 提升父委托为 root 失败: %w", err)
			}
			rootQuestID = parent.ID
			isFirstSpawn = true
			e.log.Info("fanout: promoted parent to root", "parent", parentQuestID, "group", groupID)
		} else if parent.FanoutLeafID == "" {
			// Parent is already a root — reuse its group.
			rootQuestID = parent.ID
			groupID = parent.GroupID
		} else {
			// Parent is a leaf; find the group root and reuse its group.
			if root, found := qs.FindFanoutRoot(parent.GroupID); found {
				rootQuestID = root.ID
				groupID = root.GroupID
			}
		}
	}

	q, err := e.CreateQuest(ctx, query, model.QuestTypeExecute, workDir, CreateQuestOptions{
		GroupID:        groupID,
		FanoutContract: contract,
		RequireAgent:   true,
	})
	if err != nil {
		return "", fmt.Errorf("spawn quest 创建失败: %w", err)
	}

	// Set ParentQuestID on the leaf for fanout lifecycle notifications.
	if rootQuestID != "" {
		q.ParentQuestID = rootQuestID
		if err := qs.SaveQuest(q); err != nil {
			return "", fmt.Errorf("fanout: 保存 leaf ParentQuestID 失败: %w", err)
		}
	}

	// Fanout lifecycle ThreadPosts: write durable narrative on root thread.
	if rootQuestID != "" {
		rootQ, rootErr := e.GetQuest(rootQuestID)
		if rootErr != nil {
			return "", fmt.Errorf("fanout: 获取 root 委托失败: %w", rootErr)
		}
		if rootQ == nil {
			return "", fmt.Errorf("fanout: root 委托不存在: %s", rootQuestID)
		}
		if isFirstSpawn {
			if err := e.appendFanoutGroupOpenedPost(rootQ, groupID); err != nil {
				return "", fmt.Errorf("fanout: 写入 group_opened post 失败: %w", err)
			}
		}
		// Write summary snapshot BEFORE leaf_spawned event publish, so SSE
		// subscribers that refetch on leaf_spawned will already see the
		// updated summary in the same fetch.
		if sumErr := fsstore.NewQuestStore(e.root).AppendFanoutSummarySnapshot(groupID); sumErr != nil {
			e.log.Warn("fanout: 写入 summary snapshot 失败", "group_id", groupID, "err", sumErr)
		}
		if err := e.appendFanoutLeafSpawnedPost(rootQ, q, query); err != nil {
			return "", fmt.Errorf("fanout: 写入 leaf_spawned post 失败: %w", err)
		}
		if err := e.appendFanoutMergeDecisionPost(rootQ, q, groupID); err != nil {
			return "", fmt.Errorf("fanout: 写入 merge_decision post 失败: %w", err)
		}
	}

	if err := e.StartQuest(ctx, q.ID); err != nil {
		return q.ID, fmt.Errorf("spawn quest 启动失败: %w", err)
	}
	e.publish(q.ID, "", events.EvtQuestSpawned, map[string]any{
		"qid":                 q.ID,
		"group_id":            groupID,
		"query":               query,
		"fanout_leaf_id":      q.FanoutLeafID,
		"ownership_scopes":    q.OwnershipScopes,
		"merge_strategy":      q.MergeStrategy,
		"merge_owner_leaf_id": q.MergeOwnerLeafID,
		"parent_quest_id":     rootQuestID,
	})
	return q.ID, nil
}

// appendFanoutGroupOpenedPost writes a fanout_group_opened system post on the root's thread.
// Ordering: record event → ensure root → write post → publish event (avoids SSE refetch race).
func (e *Engine) appendFanoutGroupOpenedPost(rootQ *fsstore.QuestMeta, groupID string) error {
	qs := fsstore.NewQuestStore(e.root)
	openedPayload := map[string]any{
		"group_id": groupID,
		"root_qid": rootQ.ID,
	}
	ev, err := e.recordQuestEventRequired(rootQ.ID, "", events.EvtQuestNote, openedPayload)
	if err != nil {
		return fmt.Errorf("record group_opened event: %w", err)
	}
	if ev.ID == 0 {
		return fmt.Errorf("group_opened event has zero ID")
	}
	if _, err := qs.EnsureRootThreadPost(rootQ); err != nil {
		return fmt.Errorf("ensure root thread post: %w", err)
	}
	postID := fmt.Sprintf("fanout_group_opened_%d", ev.ID)
	if _, err := qs.AppendThreadPost(rootQ.ID, fsstore.AppendThreadPostOptions{
		PostID:        postID,
		ThreadID:      rootQ.ID,
		ParentReplyID: "",
		RootPostID:    fsstore.RootPostID(rootQ.ID),
		CausalRefs:    []string{fsstore.RootPostID(rootQ.ID)},
		AuthorRole:    model.PostRoleSystem,
		SourceEventID: ev.ID,
		Kind:          "fanout_group_opened",
		Content:       "Fanout group opened: " + groupID,
	}); err != nil {
		return fmt.Errorf("append group_opened post: %w", err)
	}
	e.publishRecordedEvent(ev)
	return nil
}

// appendFanoutLeafSpawnedPost writes a fanout_leaf_spawned system post on the root's thread.
func (e *Engine) appendFanoutLeafSpawnedPost(rootQ *fsstore.QuestMeta, leafQ *fsstore.QuestMeta, query string) error {
	qs := fsstore.NewQuestStore(e.root)
	leafPayload := map[string]any{
		"group_id":       rootQ.GroupID,
		"leaf_qid":       leafQ.ID,
		"fanout_leaf_id": leafQ.FanoutLeafID,
		"query":          query,
	}
	ev, err := e.recordQuestEventRequired(rootQ.ID, "", events.EvtQuestNote, leafPayload)
	if err != nil {
		return fmt.Errorf("record leaf_spawned event: %w", err)
	}
	if ev.ID == 0 {
		return fmt.Errorf("leaf_spawned event has zero ID")
	}
	if _, err := qs.EnsureRootThreadPost(rootQ); err != nil {
		return fmt.Errorf("ensure root thread post: %w", err)
	}
	leafContent := fmt.Sprintf("Leaf spawned: %s", leafQ.ID)
	if leafQ.FanoutLeafID != "" {
		leafContent = fmt.Sprintf("Leaf spawned: %s (%s)", leafQ.FanoutLeafID, leafQ.ID)
	}
	postID := fmt.Sprintf("fanout_leaf_spawned_%d", ev.ID)
	if _, err := qs.AppendThreadPost(rootQ.ID, fsstore.AppendThreadPostOptions{
		PostID:        postID,
		ThreadID:      rootQ.ID,
		ParentReplyID: "",
		RootPostID:    fsstore.RootPostID(rootQ.ID),
		CausalRefs:    []string{fsstore.RootPostID(rootQ.ID)},
		AuthorRole:    model.PostRoleSystem,
		SourceEventID: ev.ID,
		Kind:          "fanout_leaf_spawned",
		Content:       leafContent,
	}); err != nil {
		return fmt.Errorf("append leaf_spawned post: %w", err)
	}
	e.publishRecordedEvent(ev)
	return nil
}

// appendFanoutMergeDecisionPost writes a fanout_merge_decision post on the root
// thread when a leaf with merge contract (MergeStrategy or MergeOwnerLeafID) is spawned.
// Idempotent: one merge post per group (stable post_id = sys_fanout_merge_{group_id}).
func (e *Engine) appendFanoutMergeDecisionPost(rootQ *fsstore.QuestMeta, leafQ *fsstore.QuestMeta, groupID string) error {
	if leafQ.MergeStrategy == "" && leafQ.MergeOwnerLeafID == "" {
		return nil
	}
	qs := fsstore.NewQuestStore(e.root)

	// Check if merge post already exists (idempotent).
	existingPosts, err := qs.LoadThreadPosts(rootQ.ID)
	if err == nil {
		for _, p := range existingPosts {
			if p.Kind == "fanout_merge_decision" {
				return nil
			}
		}
	}

	mergePayload := map[string]any{
		"group_id":            groupID,
		"merge_strategy":      leafQ.MergeStrategy,
		"merge_owner_leaf_id": leafQ.MergeOwnerLeafID,
		"leaf_qid":            leafQ.ID,
	}
	ev, err := e.recordQuestEventRequired(rootQ.ID, "", events.EvtQuestNote, mergePayload)
	if err != nil {
		return fmt.Errorf("record merge_decision event: %w", err)
	}
	if _, err := qs.EnsureRootThreadPost(rootQ); err != nil {
		return fmt.Errorf("ensure root thread post: %w", err)
	}

	var parts []string
	parts = append(parts, "Merge decision for group "+groupID)
	if leafQ.MergeStrategy != "" {
		parts = append(parts, "strategy="+leafQ.MergeStrategy)
	}
	if leafQ.MergeOwnerLeafID != "" {
		parts = append(parts, "owner_leaf="+leafQ.MergeOwnerLeafID)
	}

	postID := fmt.Sprintf("sys_fanout_merge_%s", groupID)
	if _, err := qs.AppendThreadPost(rootQ.ID, fsstore.AppendThreadPostOptions{
		PostID:        postID,
		ThreadID:      rootQ.ID,
		ParentReplyID: "",
		RootPostID:    fsstore.RootPostID(rootQ.ID),
		CausalRefs:    []string{fsstore.RootPostID(rootQ.ID)},
		AuthorRole:    model.PostRoleSystem,
		SourceEventID: ev.ID,
		Kind:          "fanout_merge_decision",
		Content:       strings.Join(parts, " | "),
	}); err != nil {
		return fmt.Errorf("append merge_decision post: %w", err)
	}
	e.publishRecordedEvent(ev)
	return nil
}
// type/pipeline when the stored field is empty (legacy compat).
func (e *Engine) resolveWorkflowMode(q *fsstore.QuestMeta) model.WorkflowMode {
	if q.WorkflowMode != "" {
		return q.WorkflowMode
	}
	return model.WorkflowModeFromQuest(q.Type, q.PipelineName == "run")
}

func (e *Engine) UpgradeWorkflowMode(ctx context.Context, qid string, target model.WorkflowMode, reason string) (*fsstore.QuestMeta, error) {
	return e.upgradeWorkflowMode(ctx, qid, target, reason, false)
}

func (e *Engine) upgradeWorkflowModeDuringRuntime(ctx context.Context, qid string, target model.WorkflowMode, reason string) (*fsstore.QuestMeta, error) {
	return e.upgradeWorkflowMode(ctx, qid, target, reason, true)
}

func (e *Engine) upgradeWorkflowMode(ctx context.Context, qid string, target model.WorkflowMode, reason string, allowActiveRuntime bool) (*fsstore.QuestMeta, error) {
	if !model.IsValidWorkflowMode(target) {
		return nil, workflowModeUpgradeErrorf("workflow_mode must be direct | checked | goal")
	}
	qs := fsstore.NewQuestStore(e.root)
	q, err := qs.LoadQuest(qid)
	if err != nil {
		return nil, err
	}
	current := e.resolveWorkflowMode(q)
	if !model.IsValidWorkflowMode(current) {
		return nil, workflowModeUpgradeErrorf("current workflow_mode %q is invalid", current)
	}
	if model.WorkflowModeRank(target) <= model.WorkflowModeRank(current) {
		return nil, workflowModeUpgradeErrorf("workflow_mode can only upgrade direct → checked → goal")
	}
	if !allowActiveRuntime && e.isWorkflowModeUpgradeRuntimeActive(q) {
		return nil, workflowModeUpgradeErrorf("workflow_mode cannot be upgraded while quest runtime is active")
	}
	q.WorkflowMode = target
	if err := qs.SaveQuest(q); err != nil {
		return nil, err
	}
	upgradePayload := map[string]any{
		"from":   current,
		"to":     target,
		"reason": strings.TrimSpace(reason),
	}
	upgradeEv, err := e.recordQuestEventRequired(q.ID, "", events.EvtWorkflowUpgraded, upgradePayload)
	if err != nil {
		return nil, fmt.Errorf("记录 workflow_upgraded event 失败: %w", err)
	}
	content := workflowUpgradeContent(current, target, reason)
	if _, err := qs.AppendThreadPost(q.ID, fsstore.AppendThreadPostOptions{
		PostID:         fmt.Sprintf("sys_workflow_upgrade_%s_%s", q.ID, target),
		ThreadID:       q.ID,
		ParentReplyID:  fsstore.RootPostID(q.ID),
		RootPostID:     fsstore.RootPostID(q.ID),
		CausalRefs:     []string{fsstore.RootPostID(q.ID)},
		AuthorIdentity: "system",
		AuthorRole:     model.PostRoleSystem,
		Kind:           "system_workflow_upgrade",
		Content:        content,
		SourceEventID:  upgradeEv.ID,
		CreatedAtMs:    fsstore.NowMs(),
	}); err != nil {
		return nil, err
	}
	e.publishRecordedEvent(upgradeEv)
	return qs.LoadQuest(q.ID)
}

func (e *Engine) isWorkflowModeUpgradeRuntimeActive(q *fsstore.QuestMeta) bool {
	if q == nil {
		return false
	}
	if q.Status == model.QuestStatusRunning || q.Status == model.QuestStatusReviewing {
		return true
	}
	if q.Status == model.QuestStatusPending {
		if e.isQueued(q.ID) || e.IsQuestStarting(q.ID) {
			return true
		}
	}
	return false
}

type WorkflowModeUpgradeError struct {
	Message string
}

func (e *WorkflowModeUpgradeError) Error() string { return e.Message }

func workflowModeUpgradeErrorf(format string, args ...any) error {
	return &WorkflowModeUpgradeError{Message: fmt.Sprintf(format, args...)}
}

func workflowUpgradeContent(from, to model.WorkflowMode, reason string) string {
	content := fmt.Sprintf("Workflow mode upgraded: %s → %s.", from, to)
	if strings.TrimSpace(reason) != "" {
		content += " Reason: " + strings.TrimSpace(reason)
	}
	return content
}

// SpawnExecuteFromDesign 基于已通过的 design quest 创建 execute quest。
func (e *Engine) SpawnExecuteFromDesign(ctx context.Context, designQID string, autoStart bool) (*fsstore.QuestMeta, error) {
	qs := fsstore.NewQuestStore(e.root)
	parent, err := qs.LoadQuest(designQID)
	if err != nil {
		return nil, fmt.Errorf("加载方案委托失败: %w", err)
	}
	if parent.Type != model.QuestTypeDesign {
		return nil, fmt.Errorf("委托 %s 不是 design 类型", designQID)
	}
	if parent.Status != model.QuestStatusSuccess {
		return nil, fmt.Errorf("design 委托状态 %s，尚未成功通过，不能发起执行", parent.Status)
	}
	if parent.ChildExecuteQuestID != "" {
		if child, err := qs.LoadQuest(parent.ChildExecuteQuestID); err == nil && child != nil {
			return child, nil
		}
	}
	// 检查是否已有活跃的子执行委托
	all, _ := qs.ListQuests()
	for _, q := range all {
		if q.ParentQuestID == designQID && q.Type == model.QuestTypeExecute && q.Status != model.QuestStatusFailed && q.Status != model.QuestStatusCancelled {
			parent.ChildExecuteQuestID = q.ID
			parent.SpawnError = ""
			_ = qs.SaveQuest(parent)
			return q, nil
		}
	}
	designDoc, _ := qs.LoadDesignDoc(parent.ID)
	if designDoc == nil {
		designDoc = parent.BuildDesignDoc(parent.DesignSummary, parent.FinalComment)
	}
	if designDoc.Summary == "" {
		return nil, fmt.Errorf("design 委托缺少方案摘要")
	}
	child, err := e.CreateQuest(ctx, designDoc.ExecutePrompt, model.QuestTypeExecute, parent.BaseWorkingDir, CreateQuestOptions{
		WarriorID:        parent.WarriorID,
		MageID:           parent.MageID,
		WorkspaceMode:    parent.WorkspaceMode,
		Intensity:        parent.Intensity,
		AutoSpawnExecute: false,
	})
	if err != nil {
		return nil, err
	}
	child.ParentQuestID = parent.ID
	if len(parent.Inputs) > 0 {
		for _, input := range parent.Inputs {
			f, source, err := qs.OpenArtifact(parent.ID, input.ID)
			if err != nil {
				return nil, fmt.Errorf("继承输入资产失败: %w", err)
			}
			artifact, err := qs.SaveArtifact(child.ID, fsstore.ArtifactInput{
				Name:   source.Name,
				MIME:   source.MIME,
				Source: "quest:" + parent.ID,
				Reader: f,
			})
			_ = f.Close()
			if err != nil {
				return nil, fmt.Errorf("保存继承输入资产失败: %w", err)
			}
			child.Inputs = append(child.Inputs, artifact)
		}
	}
	if err := qs.SaveQuest(child); err != nil {
		return nil, fmt.Errorf("保存执行委托失败: %w", err)
	}
	parent.ChildExecuteQuestID = child.ID
	parent.SpawnError = ""
	if err := qs.SaveQuest(parent); err != nil {
		return nil, fmt.Errorf("保存 design child 关联失败: %w", err)
	}
	e.publish(child.ID, "", events.EvtQuestCreated, map[string]any{
		"qid":             child.ID,
		"parent_quest_id": parent.ID,
		"type":            child.Type,
		"spawned_from":    "design",
	})
	if autoStart {
		if err := e.StartQuest(ctx, child.ID); err != nil {
			return child, err
		}
	}
	return child, nil
}

func (e *Engine) maybeAutoSpawnExecuteFromDesign(ctx context.Context, qid string) {
	qs := fsstore.NewQuestStore(e.root)
	parent, err := qs.LoadQuest(qid)
	if err != nil || parent == nil {
		return
	}
	if parent.Type != model.QuestTypeDesign || parent.Status != model.QuestStatusSuccess || !parent.AutoSpawnExecute {
		return
	}
	if parent.Intensity == model.QuestIntensityQuick {
		parent.SpawnError = "quick design 暂不支持自动创建执行委托"
		parent.UpdatedAtMs = fsstore.NowMs()
		_ = qs.SaveQuest(parent)
		e.publish(parent.ID, "", events.EvtQuestNote, map[string]any{
			"note":  parent.SpawnError,
			"scope": "auto_spawn_execute",
		})
		return
	}
	child, err := e.SpawnExecuteFromDesign(ctx, parent.ID, true)
	if err != nil {
		latest, loadErr := qs.LoadQuest(parent.ID)
		if loadErr == nil && latest != nil {
			latest.SpawnError = err.Error()
			latest.UpdatedAtMs = fsstore.NowMs()
			_ = qs.SaveQuest(latest)
		}
		e.publish(parent.ID, "", events.EvtQuestNote, map[string]any{
			"note":  "自动创建执行委托失败：" + err.Error(),
			"scope": "auto_spawn_execute",
		})
		return
	}
	e.publish(parent.ID, "", events.EvtQuestNote, map[string]any{
		"note":                   "已自动创建执行委托",
		"scope":                  "auto_spawn_execute",
		"child_execute_quest_id": child.ID,
	})
}

// StartQuest 启动委托执行（异步）。
// 如果已达最大并发数，则加入队列等待，返回 nil（不返回错误，排队算成功）。
func (e *Engine) StartQuest(ctx context.Context, qid string) error {
	qs := fsstore.NewQuestStore(e.root)
	q, err := qs.LoadQuest(qid)
	if err != nil {
		return fmt.Errorf("加载委托失败: %w", err)
	}
	if q.Status != model.QuestStatusPending {
		return fmt.Errorf("委托状态 %s，无法启动", q.Status)
	}

	// 同步校验冒险者：mismatched class 等快速失败不需要 workspace，
	// 应在 HTTP 请求内返回错误，而非异步后才发现。
	if q.WarriorID != "" {
		if _, err := e.pickAdventurer(q.WarriorID, model.ClassWarrior); err != nil {
			return fmt.Errorf("选剑士失败: %w", err)
		}
	}
	if q.Intensity != model.QuestIntensityQuick && q.MageID != "" {
		if _, err := e.pickAdventurer(q.MageID, model.ClassMage); err != nil {
			return fmt.Errorf("选法师失败: %w", err)
		}
	}

	rt, queued, queuePos := e.reserveQuestRuntime(qid, context.Background())
	if queued {
		e.publish(qid, "", events.EvtQuestNote, map[string]any{
			"note":      "已加入执行队列，等待空闲",
			"queue_pos": queuePos,
		})
		return nil
	}
	if rt == nil {
		// 已在 running 或已入队——幂等，不重复启动
		return nil
	}

	// workspace 准备异步化：立即返回，Prepare 在 goroutine 里跑。
	// HTTP 创建委托秒回，前端通过 SSE 感知 workspace.created / quest.started。
	// Prepare 失败在 goroutine 里标记 blocked 并 SSE 通知。
	go e.launchQuestAsync(qid, q, rt)
	return nil
}

func (e *Engine) reserveQuestRuntime(qid string, parent context.Context) (*questRuntime, bool, int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.running[qid]; ok {
		return nil, false, 0
	}
	if e.queueIndex(qid) >= 0 {
		return nil, true, e.queueIndex(qid) + 1
	}
	if maxCon := e.cfg.MaxConcurrent; maxCon > 0 && len(e.running) >= maxCon {
		e.queue = append(e.queue, qid)
		return nil, true, len(e.queue)
	}
	runCtx, cancel := context.WithCancel(parent)

	// 从持久化加载 comment cursor（跨运行周期去重）
	commentCursor := 0
	if qs := fsstore.NewQuestStore(e.root); qs != nil {
		if q, err := qs.LoadQuest(qid); err == nil {
			commentCursor = q.CommentCursor
		}
	}

	rt := &questRuntime{
		ctx:             runCtx,
		cancel:          cancel,
		done:            make(chan struct{}),
		commentCursor:   commentCursor, // 已发送
		commentEnqueued: commentCursor, // 已入队（防止重复入队）
	}
	e.running[qid] = rt
	return rt, false, 0
}

func (e *Engine) releaseQuestRuntime(qid string, rt *questRuntime, startQueued bool) {
	// 先从 running map 移除，消除与 AppendUserAnswer 的竞态窗口：
	// 此前 delete 在 flush/settle 之后，期间 answer 调 reserveQuestRuntime 会因
	// running map 有残留返回 nil，导致 quest 卡 running 无 goroutine 推进。
	e.mu.Lock()
	if cur := e.running[qid]; cur == rt {
		delete(e.running, qid)
	}
	e.mu.Unlock()

	// 持久化 comment cursor（用已 delete 的 rt，不依赖 running map）
	e.flushCommentCursor(qid, rt)
	e.settleRunningRecoveryState(qid)

	rt.cancel()
	var nextQid string
	shouldRetry := false
	e.mu.Lock()
	if !e.stopping {
		shouldRetry = e.shouldAutoRetryBlockedLocked(qid)
	}
	if startQueued && len(e.queue) > 0 {
		nextQid = e.queue[0]
		e.queue = e.queue[1:]
	}
	e.mu.Unlock()
	close(rt.done)
	if shouldRetry {
		go e.autoRetryBlockedQuest(context.Background(), qid)
	}
	if nextQid != "" {
		go e.startNextQueued(context.Background(), nextQid)
	}
}

func (e *Engine) dropReservedRuntime(qid string, rt *questRuntime) {
	e.releaseQuestRuntime(qid, rt, true)
}

// launchQuest 实际启动一个 quest（假设已有并发槽位）。
// launchQuestAsync 异步执行 workspace 准备 + 状态机启动 + macro loop。
// Prepare 失败时标记 quest blocked 并 SSE 通知，不阻塞调用方。
func (e *Engine) launchQuestAsync(qid string, q *fsstore.QuestMeta, rt *questRuntime) {
	defer e.releaseQuestRuntime(qid, rt, true)

	qs := fsstore.NewQuestStore(e.root)

	// 准备工作区
	wm := fsstore.NewWorkspaceManager(e.root)
	wsInfo, wsErr := wm.Prepare(context.Background(), qid, q.BaseWorkingDir, q.WorkspaceMode)
	if wsErr != nil {
		e.log.Error("异步准备工作区失败", "qid", qid, "err", wsErr)
		e.publish(qid, "", events.EvtQuestNote, map[string]any{
			"note":  "准备工作区失败: " + wsErr.Error(),
			"error": wsErr.Error(),
		})
		// 标记 blocked，让前端能看到失败
		if _, err := e.questService.BlockQuest(qid, quest.BlockQuestOptions{
			Reason:     "准备工作区失败: " + wsErr.Error(),
			ReasonCode: "workspace_prepare_failed",
		}); err != nil {
			e.log.Error("标记 blocked 失败", "qid", qid, "err", err)
		}
		return
	}
	q.WorkspaceMode = wsInfo.Mode
	q.WorkspacePath = wsInfo.Path
	q.BaseBranch = wsInfo.BaseBranch
	q.BaseCommit = wsInfo.BaseCommit
	if wsInfo.DowngradedFrom != "" {
		q.WorkspaceDowngrade = &fsstore.WorkspaceDowngradeInfo{
			From:   wsInfo.DowngradedFrom,
			Reason: wsInfo.DowngradeReason,
		}
	}

	// 选冒险者。agent-direct phase binding 允许没有 adventurer。
	if q.WarriorID != "" || (q.ExecuteAgentID == "" && phaseTaskForActor(q, 0).AgentID == "") {
		warrior, wErr := e.pickAdventurer(q.WarriorID, model.ClassWarrior)
		if wErr != nil {
			e.log.Error("异步选剑士失败", "qid", qid, "err", wErr)
			e.publish(qid, "", events.EvtQuestNote, map[string]any{
				"note":  "选剑士失败: " + wErr.Error(),
				"error": wErr.Error(),
			})
			if _, err := e.questService.BlockQuest(qid, quest.BlockQuestOptions{
				Reason:     "选剑士失败: " + wErr.Error(),
				ReasonCode: "adventurer_pick_failed",
			}); err != nil {
				e.log.Error("标记 blocked 失败", "qid", qid, "err", err)
			}
			return
		}
		q.WarriorID = warrior.ID
	}

	// quick 强度跳过法师评审，不分配法师
	if q.Intensity != model.QuestIntensityQuick && (q.MageID != "" || (q.ReviewAgentID == "" && phaseTaskForActor(q, 1).AgentID == "")) {
		mage, mErr := e.pickAdventurer(q.MageID, model.ClassMage)
		if mErr != nil {
			e.log.Error("异步选法师失败", "qid", qid, "err", mErr)
			e.publish(qid, "", events.EvtQuestNote, map[string]any{
				"note":  "选法师失败: " + mErr.Error(),
				"error": mErr.Error(),
			})
			if _, err := e.questService.BlockQuest(qid, quest.BlockQuestOptions{
				Reason:     "选法师失败: " + mErr.Error(),
				ReasonCode: "adventurer_pick_failed",
			}); err != nil {
				e.log.Error("标记 blocked 失败", "qid", qid, "err", err)
			}
			return
		}
		q.MageID = mage.ID
	}

	// 先保存工作区和冒险者信息（字段更新，非状态变更）
	if err := qs.SaveQuest(q); err != nil {
		e.log.Error("异步保存 quest 失败", "qid", qid, "err", err)
		e.publish(qid, "", events.EvtQuestNote, map[string]any{
			"note":  "保存 quest 失败: " + err.Error(),
			"error": err.Error(),
		})
		return
	}

	e.publish(qid, "", events.EvtWorkspaceCreated, map[string]any{
		"mode":             wsInfo.Mode,
		"path":             wsInfo.Path,
		"base_dir":         wsInfo.BaseDir,
		"base_branch":      wsInfo.BaseBranch,
		"base_commit":      wsInfo.BaseCommit,
		"downgraded_from":  wsInfo.DowngradedFrom,
		"downgrade_reason": wsInfo.DowngradeReason,
	})
	// 通过服务层启动 quest（pending → running，统一入口）
	qStartedDomain, err := e.questService.StartQuest(qid, e.cfg.MaxReworkPerQuest)
	if err != nil {
		e.log.Error("异步启动 quest 失败", "qid", qid, "err", err)
		e.publish(qid, "", events.EvtQuestNote, map[string]any{
			"note":  "启动 quest 失败: " + err.Error(),
			"error": err.Error(),
		})
		return
	}
	_ = fsstore.QuestMetaFromDomain(qStartedDomain)

	e.runMacroLoop(rt.ctx, qid)
}

// launchQuest 同步版本：供 startNextQueued 等已有 goroutine 的路径使用。
func (e *Engine) launchQuest(ctx context.Context, qid string, q *fsstore.QuestMeta, rt *questRuntime) error {
	qs := fsstore.NewQuestStore(e.root)

	// 准备工作区
	wm := fsstore.NewWorkspaceManager(e.root)
	wsInfo, wsErr := wm.Prepare(ctx, qid, q.BaseWorkingDir, q.WorkspaceMode)
	if wsErr != nil {
		e.dropReservedRuntime(qid, rt)
		return fmt.Errorf("准备工作区失败: %w", wsErr)
	}
	q.WorkspaceMode = wsInfo.Mode
	q.WorkspacePath = wsInfo.Path
	q.BaseBranch = wsInfo.BaseBranch
	q.BaseCommit = wsInfo.BaseCommit
	if wsInfo.DowngradedFrom != "" {
		q.WorkspaceDowngrade = &fsstore.WorkspaceDowngradeInfo{
			From:   wsInfo.DowngradedFrom,
			Reason: wsInfo.DowngradeReason,
		}
	}

	// 选冒险者
	warrior, wErr := e.pickAdventurer(q.WarriorID, model.ClassWarrior)
	if wErr != nil {
		e.dropReservedRuntime(qid, rt)
		return fmt.Errorf("选剑士失败: %w", wErr)
	}
	q.WarriorID = warrior.ID

	// quick 强度跳过法师评审，不分配法师
	if q.Intensity != model.QuestIntensityQuick {
		mage, mErr := e.pickAdventurer(q.MageID, model.ClassMage)
		if mErr != nil {
			e.dropReservedRuntime(qid, rt)
			return fmt.Errorf("选法师失败: %w", mErr)
		}
		q.MageID = mage.ID
	}

	// 先保存工作区和冒险者信息（字段更新，非状态变更）
	if err := qs.SaveQuest(q); err != nil {
		e.dropReservedRuntime(qid, rt)
		return fmt.Errorf("保存 quest 失败: %w", err)
	}

	e.publish(qid, "", events.EvtWorkspaceCreated, map[string]any{
		"mode":             wsInfo.Mode,
		"path":             wsInfo.Path,
		"base_dir":         wsInfo.BaseDir,
		"base_branch":      wsInfo.BaseBranch,
		"base_commit":      wsInfo.BaseCommit,
		"downgraded_from":  wsInfo.DowngradedFrom,
		"downgrade_reason": wsInfo.DowngradeReason,
	})
	// 通过服务层启动 quest（pending → running，统一入口）
	qStartedDomain, err := e.questService.StartQuest(qid, e.cfg.MaxReworkPerQuest)
	if err != nil {
		e.dropReservedRuntime(qid, rt)
		return fmt.Errorf("启动 quest 失败: %w", err)
	}
	q = fsstore.QuestMetaFromDomain(qStartedDomain)

	go func() {
		defer e.releaseQuestRuntime(qid, rt, true)
		e.runMacroLoop(rt.ctx, qid)
	}()
	return nil
}

// startNextQueued 启动一个刚出队的 quest。
func (e *Engine) startNextQueued(ctx context.Context, qid string) {
	qs := fsstore.NewQuestStore(e.root)
	q, err := qs.LoadQuest(qid)
	if err != nil {
		e.log.Error("启动排队 quest 失败：加载失败", "qid", qid, "err", err)
		return
	}
	if q.Status != model.QuestStatusPending {
		// running / reviewing 状态的排队 quest（如恢复场景、blocked 恢复后重新入队），
		// 直接走宏循环恢复路径。
		if q.Status == model.QuestStatusRunning || q.Status == model.QuestStatusReviewing {
			e.launchRunningLoop(context.Background(), qid)
		}
		return
	}
	rt, queued, _ := e.reserveQuestRuntime(qid, context.Background())
	if queued || rt == nil {
		e.log.Error("启动排队 quest 失败：出队后仍无法获得运行槽", "qid", qid)
		return
	}
	if err := e.launchQuest(ctx, qid, q, rt); err != nil {
		e.log.Error("启动排队 quest 失败", "qid", qid, "err", err)
	}
}

// isQueued 检查 quest 是否在队列中。
func (e *Engine) isQueued(qid string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.queueIndex(qid) >= 0
}

// queueIndex 返回 quest 在队列中的索引，不存在返回 -1。（调用方需持有 mu）
func (e *Engine) queueIndex(qid string) int {
	for i, id := range e.queue {
		if id == qid {
			return i
		}
	}
	return -1
}

// removeFromQueue 从队列中移除 quest，返回是否移除成功。
func (e *Engine) removeFromQueue(qid string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	idx := e.queueIndex(qid)
	if idx < 0 {
		return false
	}
	e.queue = append(e.queue[:idx], e.queue[idx+1:]...)
	return true
}

// QueueSize 返回当前队列长度。
func (e *Engine) QueueSize() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.queue)
}

func (e *Engine) QueuePosition(qid string) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	idx := e.queueIndex(qid)
	if idx < 0 {
		return 0
	}
	return idx + 1
}

func (e *Engine) IsQuestStarting(qid string) bool {
	qs := fsstore.NewQuestStore(e.root)
	q, err := qs.LoadQuest(qid)
	if err != nil || q.Status != model.QuestStatusPending {
		return false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ok := e.running[qid]
	return ok
}

// RunningCount 返回当前运行中的 quest 数。
func (e *Engine) RunningCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.running)
}

// StopQuest 停止运行中或排队中的委托。
// 编排逻辑（队列管理、运行时取消）保留在 Engine，状态变更通过 QuestService 执行。
func (e *Engine) StopQuest(qid, reason string) error {
	// 先检查是否在队列中（在队列里 → 直接移除并标记取消）。
	if e.removeFromQueue(qid) {
		// 通过服务层执行取消（状态校验 + 持久化 + 事件发布）
		comment := fmt.Sprintf("已取消（排队）：%s", reason)
		_, err := e.questService.CancelQuest(qid, comment)
		if err != nil {
			e.log.Error("cancel queued quest failed", "qid", qid, "err", err)
		}
		e.cleanupWorkspaceIfTerminal(qid)
		return nil
	}

	e.mu.Lock()
	rt, ok := e.running[qid]
	e.mu.Unlock()
	if !ok {
		qs := fsstore.NewQuestStore(e.root)
		q, err := qs.LoadQuest(qid)
		if err != nil {
			return fmt.Errorf("加载委托失败: %w", err)
		}
		if q.Status == model.QuestStatusPending && q.Source() == model.SourceAutomation {
			return e.RejectInboxItem(qid, reason)
		}
		return fmt.Errorf("委托未在运行: %s", qid)
	}
	rt.cancel()
	select {
	case <-rt.done:
	case <-time.After(3 * time.Second):
	}

	// 通过服务层执行取消（幂等：已是终态则直接返回）
	_, err := e.questService.CancelQuest(qid, reason)
	if err != nil {
		e.log.Error("cancel running quest failed", "qid", qid, "err", err)
	}
	e.cleanupWorkspaceIfTerminal(qid)
	return nil
}

type ResumeBlockedOptions struct {
	Action             string
	Comment            string
	AddTurns           int
	AddDurationMinutes int
}

func applyAndSaveQuest(qs *fsstore.QuestStore, q *fsstore.QuestMeta, action func() error) error {
	if err := action(); err != nil {
		return fmt.Errorf("状态迁移失败: %w", err)
	}
	if err := qs.SaveQuest(q); err != nil {
		return fmt.Errorf("保存 quest 失败: %w", err)
	}
	return nil
}

func normalizeBlockedAction(action string) string {
	switch action {
	case "", "continue":
		return "continue"
	case "user-review", "user_review":
		return "user-review"
	case "cancel":
		return "cancel"
	default:
		return action
	}
}

func (e *Engine) ResolveBlockedQuest(ctx context.Context, qid string, opts ResumeBlockedOptions) error {
	// 先加载 quest 以计算预算基值（编排层逻辑，不属于领域层）
	qs := fsstore.NewQuestStore(e.root)
	q, err := qs.LoadQuest(qid)
	if err != nil {
		return fmt.Errorf("加载委托失败: %w", err)
	}
	if q.Status != model.QuestStatusBlocked {
		return fmt.Errorf("委托状态 %s，不在 blocked 阶段", q.Status)
	}

	// 计算绝对预算值（编排层负责"加多少"的决策，领域层只负责状态变更）
	serviceOpts := quest.ResolveBlockedOptions{
		Action:  opts.Action,
		Comment: opts.Comment,
	}
	if opts.AddTurns > 0 {
		base := e.maxTurnsPerPhase(q)
		serviceOpts.SetMaxTurnsOverride = base + opts.AddTurns
	}
	if opts.AddDurationMinutes > 0 {
		base := e.maxDurationPerQuestMs(q)
		if base <= 0 {
			base = e.cfg.MaxDurationPerQuestMs
		}
		serviceOpts.SetMaxDurationMsOverride = base + int64(opts.AddDurationMinutes)*60*1000
	}

	// 通过 QuestService 执行状态变更（统一入口：校验 + 持久化 + 事件发布）
	result, err := e.questService.ResolveBlocked(qid, serviceOpts)
	if err != nil {
		return err
	}

	// 编排逻辑：根据 action 决定后续行为
	action := questAction(serviceOpts.Action)
	switch action {
	case "continue":
		e.launchRunningLoop(context.Background(), qid)
	case "user-review":
		e.runAfterCommitSubscribers(qid, "diff_summary")
	case "cancel":
		// 终态，无需后续操作
	}

	_ = result
	return nil
}

// questAction 归一化 action 字符串（与 quest 包的 normalizeBlockedAction 保持一致）
func questAction(action string) string {
	switch action {
	case "", "continue":
		return "continue"
	case "user-review", "user_review":
		return "user-review"
	case "cancel":
		return "cancel"
	default:
		return action
	}
}

func (e *Engine) RecoverBlockedQuest(ctx context.Context, qid string, action string, opts ResumeBlockedOptions) error {
	if action == "" {
		return fmt.Errorf("recovery action 不能为空")
	}
	qs := fsstore.NewQuestStore(e.root)
	q, err := qs.LoadQuest(qid)
	if err != nil {
		return fmt.Errorf("加载委托失败: %w", err)
	}
	if q.Status != model.QuestStatusBlocked {
		return fmt.Errorf("委托状态 %s，不在 blocked 阶段", q.Status)
	}
	if _, err := runUserRecoveryAction(ctx, action); err != nil {
		return fmt.Errorf("恢复动作失败: %w", err)
	}
	opts.Action = "continue"
	if opts.Comment == "" {
		opts.Comment = "agent recovery action: " + action
	}
	return e.ResolveBlockedQuest(ctx, qid, opts)
}

func (e *Engine) recordRecoveryDecision(q *fsstore.QuestMeta) {
	if q == nil {
		return
	}
	recoveryCount := 0
	if state, err := e.root.GetRecoveryState(q.ID); err == nil && state != nil {
		recoveryCount = state.AttemptIndex + 1
	}
	facts := policy.NewFactBuilder(e.root).BuildRecoveryFacts(q, recoveryCount)
	decision := (policy.RecoveryPolicy{}).Decide(facts)
	addTurns, _ := intCondition(decision.Conditions, "add_turns")
	addDurationMinutes, _ := intCondition(decision.Conditions, "add_duration_minutes")
	cooldownSeconds, _ := intCondition(decision.Conditions, "cooldown_seconds")
	maxAttempts, _ := intCondition(decision.Conditions, "max_attempts")
	// 配置封顶：低延迟环境/测试可收窄退避冷却。0 = 不封顶。
	if cap := e.cfg.RecoveryCooldownMaxSeconds; cap > 0 && cooldownSeconds > cap {
		cooldownSeconds = cap
	}
	state := &fsstore.RecoveryState{
		QuestID:            q.ID,
		PhaseIdx:           q.CurrentPhaseIdx(),
		PolicyName:         decision.PolicyName,
		AttemptIndex:       recoveryCount,
		ActionIndex:        0,
		InputHash:          decision.InputHash,
		AddTurns:           addTurns,
		AddDurationMinutes: addDurationMinutes,
		CooldownSeconds:    cooldownSeconds,
		MaxAttempts:        maxAttempts,
		Status:             fsstore.RecoveryStatusPending,
		LastAction:         decision.Action,
		BlockReason:        q.BlockedReasonCode,
		OriginalError:      q.BlockedReason,
	}
	// 退避：若策略给了 cooldown，设定 NextRetryAt。autoRetryBlockedQuest
	// 会等到该时刻再恢复，避免立即重试撞上同一个瞬态故障。
	if decision.Action == policy.ActionRetry && cooldownSeconds > 0 {
		state.NextRetryAt = time.Now().Add(time.Duration(cooldownSeconds) * time.Second)
	}
	if err := e.root.SaveRecoveryState(state); err != nil {
		e.log.Warn("保存 recovery state 失败", "qid", q.ID, "err", err)
	}
	e.publish(q.ID, "", events.EventType("policy.decision"), map[string]any{
		"decision_id":     decision.DecisionID,
		"quest_id":        q.ID,
		"phase_idx":       q.CurrentPhaseIdx(),
		"policy_name":     decision.PolicyName,
		"policy_action":   decision.Action,
		"input_hash":      decision.InputHash,
		"safe_by_default": decision.SafeByDefault,
		"reason":          decision.Reason,
		"conditions":      decision.Conditions,
		"input_facts":     facts.Map(),
		"policy_kind":     "recovery",
	})
}

func intCondition(conditions map[string]any, key string) (int, bool) {
	if len(conditions) == 0 {
		return 0, false
	}
	switch v := conditions[key].(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int(n), true
	default:
		return 0, false
	}
}

func (e *Engine) shouldAutoRetryBlockedLocked(qid string) bool {
	qs := fsstore.NewQuestStore(e.root)
	q, err := qs.LoadQuest(qid)
	if err != nil || q.Status != model.QuestStatusBlocked {
		return false
	}
	state, err := e.root.GetRecoveryState(qid)
	if err != nil || state == nil {
		return false
	}
	if state.LastAction != policy.ActionRetry || state.Status != fsstore.RecoveryStatusPending {
		return false
	}
	if state.MaxAttempts <= 0 {
		return false
	}
	if state.AttemptIndex >= state.MaxAttempts {
		e.recordRecoveryAttemptLimit(q, state)
		return false
	}
	return true
}

func (e *Engine) autoRetryBlockedQuest(ctx context.Context, qid string) {
	state, err := e.root.GetRecoveryState(qid)
	if err != nil || state == nil || state.LastAction != policy.ActionRetry || state.Status != fsstore.RecoveryStatusPending {
		return
	}
	if state.MaxAttempts <= 0 || state.AttemptIndex >= state.MaxAttempts {
		return
	}
	// 退避：策略设了 NextRetryAt 时，等到该时刻再恢复。立即重试可能撞上
	// 同一个瞬态故障（短时网络分区 / rate limit），退避给故障自愈窗口。
	// 等待期间用户/调度器可能已手动解决 quest，醒来后复查状态。
	if !state.NextRetryAt.IsZero() {
		if wait := time.Until(state.NextRetryAt); wait > 0 {
			e.log.Debug("recovery retry 退避等待", "qid", qid, "wait", wait)
			timer := time.NewTimer(wait)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return
			}
			// 醒来复查：engine 可能正在停，或 quest 已被手动恢复/取消。
			e.mu.Lock()
			stopping := e.stopping
			e.mu.Unlock()
			if stopping {
				return
			}
			q, err := fsstore.NewQuestStore(e.root).LoadQuest(qid)
			if err != nil || q.Status != model.QuestStatusBlocked {
				return
			}
			state, err = e.root.GetRecoveryState(qid)
			if err != nil || state == nil || state.LastAction != policy.ActionRetry || state.Status != fsstore.RecoveryStatusPending {
				return
			}
			if state.MaxAttempts <= 0 || state.AttemptIndex >= state.MaxAttempts {
				return
			}
		}
	}
	state.Status = fsstore.RecoveryStatusRunning
	if err := e.root.SaveRecoveryState(state); err != nil {
		e.log.Warn("保存 recovery retry running 状态失败", "qid", qid, "err", err)
		return
	}
	if err := e.ResolveBlockedQuest(ctx, qid, ResumeBlockedOptions{
		Action:             "continue",
		Comment:            "recovery policy retry: " + state.PolicyName,
		AddTurns:           state.AddTurns,
		AddDurationMinutes: state.AddDurationMinutes,
	}); err != nil {
		e.log.Warn("自动 recovery retry 失败", "qid", qid, "err", err)
		state.Status = fsstore.RecoveryStatusFailed
		state.LastError = err.Error()
		_ = e.root.SaveRecoveryState(state)
		return
	}
	state.Status = fsstore.RecoveryStatusRunning
	state.LastError = ""
	_ = e.root.SaveRecoveryState(state)
}

func (e *Engine) recordRecoveryAttemptLimit(q *fsstore.QuestMeta, state *fsstore.RecoveryState) {
	if q == nil || state == nil {
		return
	}
	facts := policy.NewFactBuilder(e.root).BuildRecoveryFacts(q, state.AttemptIndex)
	decision := policy.RecoveryAttemptLimitDecision(facts, state.MaxAttempts)
	e.publish(q.ID, "", events.EventType("policy.decision"), map[string]any{
		"decision_id":     decision.DecisionID,
		"quest_id":        q.ID,
		"phase_idx":       q.CurrentPhaseIdx(),
		"policy_name":     decision.PolicyName,
		"policy_action":   decision.Action,
		"input_hash":      decision.InputHash,
		"safe_by_default": decision.SafeByDefault,
		"reason":          decision.Reason,
		"conditions":      decision.Conditions,
		"input_facts":     facts.Map(),
		"policy_kind":     "recovery",
	})
}

func (e *Engine) settleRunningRecoveryState(qid string) {
	e.settleRecoveryForEvent(qid, "")
}

func (e *Engine) settleRecoveryForEvent(qid string, typ events.EventType) {
	state, err := e.root.GetRecoveryState(qid)
	if err != nil || state == nil || state.LastAction != policy.ActionRetry || state.Status != fsstore.RecoveryStatusRunning {
		return
	}
	q, err := fsstore.NewQuestStore(e.root).LoadQuest(qid)
	if err != nil {
		return
	}
	switch q.Status {
	case model.QuestStatusUserReview, model.QuestStatusSuccess:
		state.Status = fsstore.RecoveryStatusSucceeded
		state.LastError = ""
	case model.QuestStatusFailed, model.QuestStatusCancelled:
		state.Status = fsstore.RecoveryStatusFailed
		state.LastError = string(q.Status)
	case model.QuestStatusBlocked:
		e.settleBlockedRecoveryState(q, state)
	default:
		return
	}
	_ = e.root.SaveRecoveryState(state)
}

func (e *Engine) settleBlockedRecoveryState(q *fsstore.QuestMeta, state *fsstore.RecoveryState) {
	if q == nil || state == nil {
		return
	}
	reasonCode := q.BlockedReasonCode
	if reasonCode == "" {
		state.Status = fsstore.RecoveryStatusFailed
		state.LastError = "blocked_without_reason_code"
		return
	}
	if reasonCode == state.BlockReason || isHighRiskRecoveryBlockReason(reasonCode) {
		state.Status = fsstore.RecoveryStatusFailed
		state.LastError = reasonCode
		return
	}
	if state.MaxAttempts > 0 && state.AttemptIndex < state.MaxAttempts {
		state.Status = fsstore.RecoveryStatusPending
		state.LastError = reasonCode
		return
	}
	state.Status = fsstore.RecoveryStatusFailed
	state.LastError = reasonCode
}

func isHighRiskRecoveryBlockReason(reasonCode string) bool {
	switch reasonCode {
	case "duration_exceeded", "user_confirm_timeout":
		return true
	default:
		return strings.Contains(reasonCode, "auth") ||
			strings.Contains(reasonCode, "l2") ||
			strings.Contains(reasonCode, "external_side_effect")
	}
}

func (e *Engine) launchRunningLoop(ctx context.Context, qid string) {
	rt, queued, queuePos := e.reserveQuestRuntime(qid, ctx)
	if queued {
		e.log.Debug("launchRunningLoop quest 已入队", "qid", qid, "queue_pos", queuePos)
		return
	}
	if rt == nil {
		// reserveQuestRuntime 返回 nil 且未入队 = quest 已在 e.running map
		// （旧 goroutine 未 release）。rework 路径上这会导致 quest 状态 running
		// 但无新 goroutine 推进，in-loop 超时检查失效，quest 卡死。
		// 告警暴露这个状态，scheduler 的 blockExpiredRunningQuests 兜底。
		e.log.Warn("launchRunningLoop 无可用 runtime（旧 goroutine 未释放），quest 可能卡死，由 scheduler 超时兜底", "qid", qid)
		return
	}
	go func() {
		defer e.releaseQuestRuntime(qid, rt, false)
		e.runMacroLoop(rt.ctx, qid)
	}()
}

// ==================== 启动恢复（孤儿 Quest 回收） ====================
//
// 服务重启后，磁盘上可能残留处于 running / reviewing 状态的 quest，
// 但内存里没有 goroutine 推进它们，表现为"卡住"。
// RecoverOrphanedQuests 在引擎启动后调用一次，扫描并恢复这些孤儿 quest。
//
// 恢复策略（全局最优权衡）：
//   - reviewing: 剑士已完成，工作区是最终状态 → 直接续跑法师阶段（安全、零损耗）
//   - running:   剑士可能做了一半 → 保留工作区，重新发起剑士 prompt，由 agent 自行判断进度
//     （系统做机制，agent 做决策。重跑的开销远小于引入 turn 级 checkpoint 的复杂度）
//   - 无法安全恢复的（工作区丢失、冒险者不可用）→ 转 blocked，用户手动处理
//
// 返回值：recovered=已恢复执行数, blocked=降级为阻塞数, total=发现的孤儿总数

const orphanAutoRecoveryWindowMs = int64(6 * 60 * 60 * 1000) // 6h

// RecoverOrphanedQuests 扫描磁盘上处于 running/reviewing 状态的孤儿 quest 并尝试恢复。
// 并发约束：受 MaxConcurrent 限制，超出的入队等待。
func (e *Engine) RecoverOrphanedQuests(ctx context.Context) (recovered int, blocked int, total int) {
	qs := fsstore.NewQuestStore(e.root)
	quests, err := qs.ListQuests()
	if err != nil {
		e.log.Error("恢复孤儿 quest：列表加载失败", "err", err)
		return 0, 0, 0
	}

	var orphans []*fsstore.QuestMeta
	for _, q := range quests {
		if q.Status == model.QuestStatusRunning || q.Status == model.QuestStatusReviewing {
			orphans = append(orphans, q)
		}
	}
	total = len(orphans)
	if total == 0 {
		return 0, 0, 0
	}

	e.log.Info("发现孤儿 quest，尝试恢复", "count", total)

	for _, q := range orphans {
		// 基本健全性检查：无法安全恢复的降级为 blocked
		if reason := e.recoverySanityCheck(q); reason != "" {
			e.log.Warn("孤儿 quest 无法自动恢复，转 blocked",
				"qid", q.ID, "status", q.Status, "reason", reason)
			e.markOrphanBlocked(qs, q, reason)
			blocked++
			continue
		}
		if reason := orphanRecoveryWindowReason(q, fsstore.NowMs()); reason != "" {
			e.log.Warn("孤儿 quest 超过自动恢复窗口，转 blocked",
				"qid", q.ID, "status", q.Status, "reason", reason)
			e.markOrphanBlocked(qs, q, reason)
			blocked++
			continue
		}

		// 获取运行槽位；并发满了就入队，由 releaseQuestRuntime → startNextQueued 出队
		rt, queued, queuePos := e.reserveQuestRuntime(q.ID, ctx)
		if queued {
			e.log.Info("孤儿 quest 已加入恢复队列", "qid", q.ID, "queue_pos", queuePos)
			e.publish(q.ID, "", events.EvtQuestNote, map[string]any{
				"note":        "服务中断后恢复中，已加入执行队列",
				"queue_pos":   queuePos,
				"mode":        "recovery",
				"from_status": q.Status,
			})
			continue
		}
		if rt == nil {
			// 已经在 running map 里（并发调用等），跳过
			continue
		}

		// 拿到槽位 → 直接启动宏循环
		e.log.Info("孤儿 quest 恢复执行", "qid", q.ID, "from_status", q.Status)
		e.publish(q.ID, "", events.EvtQuestNote, map[string]any{
			"note":        "服务中断后自动恢复执行",
			"mode":        "recovery",
			"from_status": q.Status,
		})

		go func(qid string, rt *questRuntime) {
			defer e.releaseQuestRuntime(qid, rt, true)
			e.runMacroLoop(rt.ctx, qid)
		}(q.ID, rt)
		recovered++
	}

	return recovered, blocked, total
}

func orphanRecoveryWindowReason(q *fsstore.QuestMeta, nowMs int64) string {
	if q == nil || nowMs <= 0 {
		return ""
	}
	anchor := q.UpdatedAtMs
	if anchor == 0 {
		anchor = q.StartedAtMs
	}
	if anchor == 0 {
		anchor = q.CreatedAtMs
	}
	if anchor == 0 || nowMs-anchor <= orphanAutoRecoveryWindowMs {
		return ""
	}
	return fmt.Sprintf("状态停留超过自动恢复窗口（%d 小时），需要用户确认后继续", orphanAutoRecoveryWindowMs/(60*60*1000))
}

// recoverySanityCheck 检查 quest 是否具备自动恢复的基本条件。
// 返回空字符串表示可以恢复，否则返回原因。
func (e *Engine) recoverySanityCheck(q *fsstore.QuestMeta) string {
	if q.WorkspacePath == "" {
		return "工作区路径为空"
	}
	if q.WarriorID == "" {
		return "剑士冒险者未设置"
	}
	// 检查冒险者是否存在且可用
	if warrior, err := e.root.GetAdventurer(q.WarriorID); err != nil || warrior.Status != model.AdventurerActive {
		return "剑士冒险者不可用"
	}
	if q.MageID == "" {
		if q.Intensity == model.QuestIntensityQuick || q.Status == model.QuestStatusRunning {
			return ""
		}
		return "法师冒险者未设置"
	}
	if mage, err := e.root.GetAdventurer(q.MageID); err != nil || mage.Status != model.AdventurerActive {
		return "法师冒险者不可用"
	}
	return ""
}

// markOrphanBlocked 将孤儿 quest 标记为 blocked 并发布事件。
func (e *Engine) markOrphanBlocked(qs *fsstore.QuestStore, q *fsstore.QuestMeta, reason string) {
	q.Status = model.QuestStatusBlocked
	q.BlockedReason = "服务中断后无法自动恢复：" + reason
	q.BlockedReasonCode = "service_interruption_cannot_recover"
	q.UpdatedAtMs = fsstore.NowMs()
	if err := qs.SaveQuest(q); err != nil {
		e.log.Error("保存 orphan blocked 状态失败", "qid", q.ID, "err", err)
		return
	}
	e.publish(q.ID, "", events.EvtQuestBlocked, map[string]any{
		"reason": "service_interruption_cannot_recover",
		"phase":  "recovery",
		"detail": reason,
	})
	if blocked, err := qs.LoadQuest(q.ID); err == nil {
		e.recordRecoveryDecision(blocked)
	}
}

// ResolveUserReview 用户终审裁决。
func (e *Engine) ResolveUserReview(ctx context.Context, qid string, verdict model.QuestVerdict, comment string) error {
	// 通过 QuestService 执行状态变更（统一入口：校验 + 持久化 + 事件发布）
	q, err := e.questService.ResolveUserReview(qid, quest.ResolveUserReviewOptions{
		Verdict: verdict,
		Comment: comment,
	})
	if err != nil {
		return err
	}

	// 编排逻辑：返工的话重新启动宏循环
	if verdict == model.VerdictRequestChange {
		e.launchRunningLoop(context.Background(), qid)
	}

	// 编排逻辑：通过的话更新 diff summary 缓存（自动化 quest 可能需要）
	if verdict == model.VerdictPass {
		// pass 状态不需要更新 diff（已经是最终态了）
		_ = q
		if q != nil {
			qMeta := fsstore.QuestMetaFromDomain(q)
			e.recordAutomationTrustOutcome(qMeta, "success", "human_approved", true)
			if qMeta != nil && qMeta.Type == model.QuestTypeDesign && qMeta.AutoSpawnExecute {
				go e.maybeAutoSpawnExecuteFromDesign(context.Background(), qMeta.ID)
			}
		}
	}
	if verdict == model.VerdictReject {
		if q != nil {
			e.recordAutomationTrustOutcome(fsstore.QuestMetaFromDomain(q), "failure", "human_rejected", true)
		}
	}

	return nil
}

func (e *Engine) completeUserReviewedQuest(qs *fsstore.QuestStore, q *fsstore.QuestMeta, verdict model.QuestVerdict, comment string, eventType events.EventType) error {
	return e.completeReviewedQuest(qs, q, verdict, comment, eventType, nil)
}

func (e *Engine) loadQuestAdventurers(q *fsstore.QuestMeta) (*fsstore.AdventurerFile, *fsstore.AdventurerFile) {
	var warrior *fsstore.AdventurerFile
	if q.WarriorID != "" {
		adv, err := e.root.GetAdventurer(q.WarriorID)
		if err == nil {
			warrior = adv
		} else {
			e.log.Warn("加载冒险者失败", "adv_id", q.WarriorID, "err", err)
		}
	}

	var mage *fsstore.AdventurerFile
	if q.MageID != "" {
		adv, err := e.root.GetAdventurer(q.MageID)
		if err == nil {
			mage = adv
		} else {
			e.log.Warn("加载冒险者失败", "adv_id", q.MageID, "err", err)
		}
	}
	return warrior, mage
}

// IsQuestRunning 检查委托是否在运行。
func (e *Engine) IsQuestRunning(qid string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ok := e.running[qid]
	return ok
}

// ==================== Workspace: Diff / Apply / Discard ====================

// ComputeDiff 计算工作区相对 base 的差异。
func (e *Engine) ComputeDiff(ctx context.Context, qid string) (fsstore.DiffResult, error) {
	qs := fsstore.NewQuestStore(e.root)
	q, err := qs.LoadQuest(qid)
	if err != nil {
		return fsstore.DiffResult{}, fmt.Errorf("加载委托失败: %w", err)
	}
	if q.WorkspacePath == "" {
		return fsstore.DiffResult{}, fmt.Errorf("工作区未初始化")
	}
	wm := fsstore.NewWorkspaceManager(e.root)
	diff, err := wm.ComputeDiff(ctx, qid, q)
	if err != nil {
		return diff, err
	}
	// 缓存到 quest 元信息
	e.publish(qid, "", events.EvtWorkspaceDiffReady, map[string]any{
		"changed_files": diff.ChangedFiles,
		"additions":     diff.Additions,
		"deletions":     diff.Deletions,
		"stat":          diff.Stat,
	})
	return diff, nil
}

func (e *Engine) ListApplyBackups(qid string) ([]*fsstore.ApplyBackupMeta, error) {
	if _, err := e.GetQuest(qid); err != nil {
		return nil, err
	}
	return fsstore.NewWorkspaceManager(e.root).ListApplyBackups(qid)
}

func (e *Engine) GetApplyBackup(qid string, backupID string) (*fsstore.ApplyBackupMeta, error) {
	if _, err := e.GetQuest(qid); err != nil {
		return nil, err
	}
	return fsstore.NewWorkspaceManager(e.root).GetApplyBackup(qid, backupID)
}

// ApplyQuest 把工作区改动合回 base 目录。
// force=true 时跳过高严重级别的安全检查。
// 返回安全警告列表（即使 apply 成功也可能有警告）。
func (e *Engine) ApplyQuest(ctx context.Context, qid string, force bool) ([]fsstore.SafetyWarning, error) {
	qs := fsstore.NewQuestStore(e.root)
	q, err := qs.LoadQuest(qid)
	if err != nil {
		return nil, fmt.Errorf("加载委托失败: %w", err)
	}
	if q.Status != model.QuestStatusSuccess {
		return nil, fmt.Errorf("只有 success 状态的委托才能 apply（当前: %s）", q.Status)
	}
	if q.Applied {
		return nil, fmt.Errorf("该委托已经 apply 过了")
	}
	if q.WorkspaceMode == model.WorkspaceReadOnly {
		// readonly 模式无隔离工作区，改动直接在基准目录，无需 apply → no-op 成功。
		// 不报错失败，避免 readonly 调研委托卡在 apply_failed。
		q.Applied = true
		q.ApplyStatus = model.ApplyStatusApplied
		q.ApplyError = ""
		q.ApplyFailedAtMs = 0
		q.UpdatedAtMs = fsstore.NowMs()
		if err := qs.SaveQuest(q); err != nil {
			return nil, fmt.Errorf("保存 quest 失败: %w", err)
		}
		e.publish(q.ID, "", events.EvtQuestApplied, map[string]any{
			"skipped":  true,
			"reason":   "readonly 模式无隔离工作区，无需 apply",
			"warnings": nil,
		})
		e.recordAutomationTrustOutcome(q, "success", "apply_success", true)
		return nil, nil
	}
	q.ApplyStatus = model.ApplyStatusPending
	q.ApplyError = ""
	q.ApplyFailedAtMs = 0
	q.UpdatedAtMs = fsstore.NowMs()
	if err := qs.SaveQuest(q); err != nil {
		return nil, fmt.Errorf("保存 apply intent 失败: %w", err)
	}

	// 如果 quest 只有外部产出物（如飞书文档），没有实质的 workspace 文件变更，
	// apply 直接 no-op 成功 —— 不需要合并任何东西回 base。
	// 这避免了"写飞书文档的 quest 也有 apply 按钮且失败"的问题。
	if hasOnlyExternalOutputs(q) {
		q.Applied = true
		q.ApplyStatus = model.ApplyStatusApplied
		q.ApplyError = ""
		q.ApplyFailedAtMs = 0
		if err := qs.SaveQuest(q); err != nil {
			return nil, fmt.Errorf("保存 quest 失败: %w", err)
		}
		e.publish(q.ID, "", events.EvtQuestApplied, map[string]any{
			"skipped":  true,
			"reason":   "只有外部产出物，无 workspace 变更需要合并",
			"warnings": nil,
		})
		e.recordAutomationTrustOutcome(q, "success", "apply_success", true)
		return nil, nil
	}

	warnings, backup, err := e.applyWorkspace(ctx, qid, q, force)
	if err != nil {
		e.recordApplyFailure(qs, q, err)
		return warnings, err
	}

	q.Applied = true
	q.ApplyStatus = model.ApplyStatusApplied
	q.ApplyError = ""
	q.ApplyFailedAtMs = 0
	if err := qs.SaveQuest(q); err != nil {
		return warnings, fmt.Errorf("保存 quest 失败: %w", err)
	}

	e.cleanupAppliedWorkspace(ctx, qid, q)
	e.publishQuestApplied(qid, q, warnings, force, backup)
	e.recordAutomationTrustOutcome(q, "success", "apply_success", true)
	return warnings, nil
}

func (e *Engine) recordApplyFailure(qs *fsstore.QuestStore, q *fsstore.QuestMeta, err error) {
	if q == nil || err == nil {
		return
	}
	q.ApplyStatus = model.ApplyStatusFailed
	q.ApplyError = err.Error()
	q.ApplyFailedAtMs = fsstore.NowMs()
	q.UpdatedAtMs = q.ApplyFailedAtMs
	if saveErr := qs.SaveQuest(q); saveErr != nil {
		e.log.Warn("保存 apply 失败状态失败", "qid", q.ID, "err", saveErr)
	}
	e.publish(q.ID, "", events.EvtQuestApplyFailed, map[string]any{
		"error": err.Error(),
	})
	e.recordAutomationTrustOutcome(q, "failure", "apply_failed", true)
}

func (e *Engine) RecoverPendingApplies(ctx context.Context) (recovered int, failed int, total int) {
	qs := fsstore.NewQuestStore(e.root)
	quests, err := qs.ListQuests()
	if err != nil {
		e.log.Error("恢复 pending apply：列表加载失败", "err", err)
		return 0, 0, 0
	}
	for _, q := range quests {
		if q.Status != model.QuestStatusSuccess || q.Applied || q.ApplyStatus != model.ApplyStatusPending {
			continue
		}
		total++
		e.log.Info("恢复 pending apply", "qid", q.ID)
		if _, err := e.ApplyQuest(ctx, q.ID, false); err != nil {
			e.log.Warn("恢复 pending apply 失败", "qid", q.ID, "err", err)
			failed++
			continue
		}
		recovered++
	}
	return recovered, failed, total
}

func (e *Engine) RecoverWaitingInputTimeouts(ctx context.Context) (continued int, blocked int, escalated int, total int) {
	qs := fsstore.NewQuestStore(e.root)
	quests, err := qs.ListQuests()
	if err != nil {
		e.log.Error("恢复 waiting_input：列表加载失败", "err", err)
		return 0, 0, 0, 0
	}
	now := time.Now().UTC()
	for _, q := range quests {
		if q.Status != model.QuestStatusWaitingInput || q.WaitingInput == nil || q.WaitingInput.TimeoutMs <= 0 {
			continue
		}
		deadline := q.WaitingInput.AskedAt.Add(time.Duration(q.WaitingInput.TimeoutMs) * time.Millisecond)
		// automation quest 无人 answer，24h 超时太长会僵尸。用 5min 短超时
		// 让 timeout_action（默认 continue=自主判断继续）尽快生效。
		if q.Source() == model.SourceAutomation {
			autoDeadline := q.WaitingInput.AskedAt.Add(5 * time.Minute)
			if autoDeadline.Before(deadline) {
				deadline = autoDeadline
			}
		}
		if now.Before(deadline) {
			continue
		}
		total++
		action := q.WaitingInput.TimeoutAction
		if action == "" {
			action = "continue"
		}
		// Circuit-breaker: automation quest 连续 timeout-continue 超过 3 次，
		// 强制 block，防止 agent 反复请求 user_input 导致无限循环。
		const maxAutoTimeoutContinues = 3
		if action == "continue" && q.Source() == model.SourceAutomation &&
			len(q.WaitingInput.ProcessedAnswerIDs) >= maxAutoTimeoutContinues {
			action = "block"
			e.log.Warn("automation quest timeout-continue 次数超限，强制 block",
				"qid", q.ID, "continues", len(q.WaitingInput.ProcessedAnswerIDs))
		}
		switch action {
		case "continue":
			_, err := e.AppendUserAnswer(q.ID, q.WaitingInput.QuestionID, "用户未在规定时间内回复，请基于现有信息继续。", "timeout")
			if err != nil {
				e.log.Warn("waiting_input 超时继续失败", "qid", q.ID, "err", err)
				continue
			}
			continued++
		case "block":
			q.BlockedReason = "waiting_input_timeout"
			q.WaitingInput = nil
			if err := q.TransitionTo(model.QuestStatusBlocked); err != nil {
				e.log.Warn("waiting_input 超时转 blocked 失败", "qid", q.ID, "err", err)
				continue
			}
			q.UpdatedAtMs = fsstore.NowMs()
			if err := qs.SaveQuest(q); err != nil {
				e.log.Warn("保存 waiting_input blocked 状态失败", "qid", q.ID, "err", err)
				continue
			}
			e.publish(q.ID, "", events.EvtQuestBlocked, map[string]any{"reason": "waiting_input_timeout"})
			blocked++
		case "escalate":
			q.FinalComment = "waiting_input 超时，提交用户终审"
			q.WaitingInput = nil
			if err := q.TransitionTo(model.QuestStatusUserReview); err != nil {
				e.log.Warn("waiting_input 超时提交终审失败", "qid", q.ID, "err", err)
				continue
			}
			q.UpdatedAtMs = fsstore.NowMs()
			if err := qs.SaveQuest(q); err != nil {
				e.log.Warn("保存 waiting_input user_review 状态失败", "qid", q.ID, "err", err)
				continue
			}
			e.publish(q.ID, "", events.EvtUserReview, map[string]any{"from": "waiting_input_timeout"})
			escalated++
		default:
			e.log.Warn("未知 waiting_input timeout_action，保持等待", "qid", q.ID, "action", action)
		}
	}
	return continued, blocked, escalated, total
}

func (e *Engine) applyWorkspace(ctx context.Context, qid string, q *fsstore.QuestMeta, force bool) ([]fsstore.SafetyWarning, *fsstore.ApplyBackupMeta, error) {
	wm := fsstore.NewWorkspaceManager(e.root)

	// 安全检查
	warnings, sErr := wm.SafetyCheck(ctx, qid, q)
	if sErr != nil {
		e.log.Warn("安全检查失败，继续", "qid", qid, "err", sErr)
	}
	if !force && fsstore.HasHighSeverity(warnings) {
		return warnings, nil, fmt.Errorf("存在高严重级别安全警告，使用 force 跳过（警告数: %d）", len(warnings))
	}

	backup, err := wm.Apply(ctx, qid, q, e.cfg.ApplyStrategy)
	if err != nil {
		return warnings, nil, fmt.Errorf("apply 失败: %w", err)
	}

	return warnings, backup, nil
}

func (e *Engine) cleanupAppliedWorkspace(ctx context.Context, qid string, q *fsstore.QuestMeta) {
	// apply 成功后自动清理工作区（改动已合回 base，workspace 不再需要）
	wm := fsstore.NewWorkspaceManager(e.root)
	if err := wm.Cleanup(ctx, qid, q); err != nil {
		e.log.Warn("apply 后清理工作区失败", "qid", qid, "err", err)
	}
}

func (e *Engine) publishQuestApplied(qid string, q *fsstore.QuestMeta, warnings []fsstore.SafetyWarning, force bool, backup *fsstore.ApplyBackupMeta) {
	payload := map[string]any{
		"mode":     q.WorkspaceMode,
		"warnings": len(warnings),
		"force":    force,
	}
	if backup != nil {
		if backup.ID != "" {
			payload["backup_id"] = backup.ID
		}
		if backup.PatchPath != "" {
			payload["patch_path"] = backup.PatchPath
		}
		payload["files_changed"] = backup.FilesChanged
	}
	e.publish(qid, "", events.EvtQuestApplied, payload)
}

// DiscardQuest 丢弃工作区改动（不影响 base）。
// quest 仍然保留记录，只是工作区被清空。
func (e *Engine) DiscardQuest(ctx context.Context, qid string, reason string) error {
	qs := fsstore.NewQuestStore(e.root)
	q, err := qs.LoadQuest(qid)
	if err != nil {
		return fmt.Errorf("加载委托失败: %w", err)
	}
	if q.WorkspaceMode == model.WorkspaceReadOnly {
		return fmt.Errorf("readonly 模式不支持 discard")
	}

	wm := fsstore.NewWorkspaceManager(e.root)
	if err := wm.Discard(ctx, qid, q); err != nil {
		return fmt.Errorf("discard 失败: %w", err)
	}

	e.publish(qid, "", events.EvtQuestDiscarded, map[string]any{
		"reason": reason,
	})
	return nil
}

// cleanupWorkspaceIfTerminal 在 quest 进入终态后清理工作区。
// 幂等：已清理、非终态或 readonly 模式都不报错。
// worktree 模式：git worktree remove + branch -D + 删 work 目录。
// copy 模式：删 work 目录。
// readonly 模式：跳过（无独立 workspace，直接用 baseDir）。
func (e *Engine) cleanupWorkspaceIfTerminal(qid string) {
	qs := fsstore.NewQuestStore(e.root)
	q, err := qs.LoadQuest(qid)
	if err != nil || q == nil {
		return
	}
	if !q.Status.IsTerminal() {
		return
	}
	if q.WorkspaceMode == model.WorkspaceReadOnly {
		return
	}
	wm := fsstore.NewWorkspaceManager(e.root)
	if err := wm.Cleanup(context.Background(), qid, q); err != nil {
		e.log.Warn("终态清理工作区失败", "qid", qid, "err", err)
	}
}

// CleanupOrphanWorkspaces 清理孤儿 worktree 和分支（quest 不存在或已终态的）。
// daemon 启动时和 scheduler 定时调用。返回被清理的条目数。
func (e *Engine) CleanupOrphanWorkspaces(ctx context.Context) int {
	wm := fsstore.NewWorkspaceManager(e.root)
	cleaned, err := wm.CleanupOrphanWorktrees(ctx)
	if err != nil {
		e.log.Warn("清理孤儿 worktree 失败", "err", err)
		return 0
	}
	if len(cleaned) > 0 {
		e.log.Info("清理孤儿 worktree 完成", "count", len(cleaned))
	}
	return len(cleaned)
}

// updateDiffSummary 计算 diff 并把摘要缓存到 quest meta 里。
// 计算失败不影响 quest 状态（打 warn 日志）。
// 以 fire-and-forget goroutine 启动，内部自带 panic 恢复，避免带崩主进程。
func (e *Engine) updateDiffSummary(qid string) {
	defer func() {
		if r := recover(); r != nil {
			e.log.Error("updateDiffSummary panic 已恢复", "qid", qid, "panic", r)
		}
	}()
	qs := fsstore.NewQuestStore(e.root)
	q, err := qs.LoadQuest(qid)
	if err != nil {
		e.log.Warn("更新 diff 摘要失败：加载 quest 失败", "qid", qid, "err", err)
		return
	}
	if q.WorkspacePath == "" {
		return
	}
	if q.WorkspaceMode == model.WorkspaceReadOnly {
		return
	}
	status := q.Status

	wm := fsstore.NewWorkspaceManager(e.root)
	diff, err := wm.ComputeDiff(context.Background(), qid, q)
	if err != nil {
		e.log.Warn("计算 diff 失败", "qid", qid, "err", err)
		return
	}

	q.DiffStat = diff.Stat
	q.DiffChangedFiles = diff.ChangedFiles
	q.DiffAdditions = diff.Additions
	q.DiffDeletions = diff.Deletions

	latest, err := qs.LoadQuest(qid)
	if err != nil {
		e.log.Warn("保存 diff 摘要失败：重新加载 quest 失败", "qid", qid, "err", err)
		return
	}
	if latest.Status != status {
		e.log.Debug("跳过 diff 摘要保存：quest 状态已变化", "qid", qid, "old_status", status, "new_status", latest.Status)
		return
	}
	latest.DiffStat = q.DiffStat
	latest.DiffChangedFiles = q.DiffChangedFiles
	latest.DiffAdditions = q.DiffAdditions
	latest.DiffDeletions = q.DiffDeletions

	if err := qs.SaveQuest(latest); err != nil {
		e.log.Warn("保存 diff 摘要失败", "qid", qid, "err", err)
		return
	}

	e.publish(qid, "", events.EvtWorkspaceDiffReady, map[string]any{
		"changed_files": diff.ChangedFiles,
		"additions":     diff.Additions,
		"deletions":     diff.Deletions,
		"stat":          diff.Stat,
	})
}

func (e *Engine) refreshDiffSummaryForPolicy(ctx context.Context, qid string) bool {
	qs := fsstore.NewQuestStore(e.root)
	q, err := qs.LoadQuest(qid)
	if err != nil {
		e.log.Warn("刷新 policy diff 摘要失败：加载 quest 失败", "qid", qid, "err", err)
		return false
	}
	if q.WorkspacePath == "" || q.WorkspaceMode == model.WorkspaceReadOnly {
		return true
	}
	wm := fsstore.NewWorkspaceManager(e.root)
	diff, err := wm.ComputeDiff(ctx, qid, q)
	if err != nil {
		e.log.Warn("刷新 policy diff 摘要失败：计算 diff 失败", "qid", qid, "err", err)
		return false
	}
	q.DiffStat = diff.Stat
	q.DiffChangedFiles = diff.ChangedFiles
	q.DiffAdditions = diff.Additions
	q.DiffDeletions = diff.Deletions
	q.UpdatedAtMs = fsstore.NowMs()
	if err := qs.SaveQuest(q); err != nil {
		e.log.Warn("刷新 policy diff 摘要失败：保存 quest 失败", "qid", qid, "err", err)
		return false
	}
	e.publish(qid, "", events.EvtWorkspaceDiffReady, map[string]any{
		"changed_files": diff.ChangedFiles,
		"additions":     diff.Additions,
		"deletions":     diff.Deletions,
		"stat":          diff.Stat,
	})
	return true
}

// safeLoadStatus 安全加载 quest 状态（用于事件中）。
func (e *Engine) safeLoadStatus(qid string) string {
	qs := fsstore.NewQuestStore(e.root)
	if q, err := qs.LoadQuest(qid); err == nil {
		return string(q.Status)
	}
	return ""
}

// hasOnlyExternalOutputs 检查 quest 是否只有外部产出物（如飞书文档链接）。
// 当只有外部产出物时，workspace diff 被视为临时文件，不需要 apply。
func hasOnlyExternalOutputs(q *fsstore.QuestMeta) bool {
	if q == nil || len(q.Outputs) == 0 {
		return false
	}
	hasExternal := false
	hasFile := false
	for _, out := range q.Outputs {
		if strings.HasPrefix(out.StoragePath, "http://") || strings.HasPrefix(out.StoragePath, "https://") {
			hasExternal = true
		} else if out.StoragePath != "" {
			hasFile = true
		}
	}
	return hasExternal && !hasFile
}
