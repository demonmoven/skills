package quest

import (
	"fmt"
	"log"

	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// ==================== Quest 应用服务层 ====================
//
// QuestService 是所有 Quest 状态变更的单一入口。
// 它编排领域逻辑（状态机、失败归因）和基础设施（存储、事件总线）。
//
// 设计原则：
// - 所有修改 Quest 状态的操作都必须通过 QuestService
// - Service 层不包含业务规则，只做协调
// - 业务规则在领域层（status_machine.go, failure.go）
// - 存储通过 Repository 接口访问
//
// Phase 1: 这是过渡设计。服务层直接封装了常见的状态变更操作，
// 确保所有入口都经过统一的校验和事件发布。

// EventPublisher 发布事件的接口（最小化依赖）。
type EventPublisher interface {
	Publish(qid, sessionID string, typ events.EventType, payload map[string]any)
}

// Service 是 Quest 应用服务。
type Service struct {
	repo   Repository
	events EventPublisher
	nowMs  func() int64 // 可注入的时间函数（方便测试）
}

// NewService 创建 Quest 服务。
func NewService(repo Repository, evts EventPublisher) *Service {
	return &Service{
		repo:   repo,
		events: evts,
		nowMs:  model.NowMs,
	}
}

// ===== 阻塞相关操作 =====

// ResolveBlockedOptions 解除阻塞的选项。
type ResolveBlockedOptions struct {
	Action             string // continue | user-review | cancel
	Comment            string
	AddTurns           int // 相对增加的回合数
	AddDurationMinutes int // 相对增加的时长（分钟）
	// 绝对预算覆盖（如果设置，优先使用；否则用 AddXxx 相对增加）
	SetMaxTurnsOverride      int
	SetMaxDurationMsOverride int64
}

// ResolveBlocked 处理 blocked 状态的 quest。
// 返回迁移后的 quest 和错误。
func (s *Service) ResolveBlocked(qid string, opts ResolveBlockedOptions) (*QuestMeta, error) {
	q, err := s.repo.Load(qid)
	if err != nil {
		return nil, fmt.Errorf("加载委托失败: %w", err)
	}
	if q.Status != model.QuestStatusBlocked {
		return nil, fmt.Errorf("委托状态 %s，不在 blocked 阶段", q.Status)
	}

	// 调整预算（绝对覆盖优先，否则相对增加）
	if opts.SetMaxTurnsOverride > 0 {
		q.MaxTurnsPerPhaseOverride = opts.SetMaxTurnsOverride
	} else if opts.AddTurns > 0 {
		q.MaxTurnsPerPhaseOverride += opts.AddTurns
	}
	if opts.SetMaxDurationMsOverride > 0 {
		q.MaxDurationMsOverride = opts.SetMaxDurationMsOverride
	} else if opts.AddDurationMinutes > 0 {
		q.MaxDurationMsOverride += int64(opts.AddDurationMinutes) * 60 * 1000
	}

	action := normalizeBlockedAction(opts.Action)
	var result TransitionResult
	resumePhaseIdx := -1
	if action == "continue" {
		q.EnsurePhases()
		resumePhaseIdx = q.CurrentPhaseIdx
		if resumePhaseIdx < 0 || resumePhaseIdx >= len(q.Phases) {
			for i := range q.Phases {
				if !q.Phases[i].IsTerminal() {
					resumePhaseIdx = i
					break
				}
			}
		}
		if resumePhaseIdx < 0 && len(q.Phases) > 0 {
			resumePhaseIdx = 0
		}
	}

	switch action {
	case "continue":
		result, err = ResumeFromBlocked(q.Status, opts.Comment, q.ResumeCount)
		if err != nil {
			return nil, err
		}
	case "user-review":
		result, err = MoveBlockedToUserReview(q.Status, opts.Comment)
		if err != nil {
			return nil, err
		}
	case "cancel":
		result, err = CancelFromBlocked(q.Status, opts.Comment)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("action 必须是 continue | user-review | cancel")
	}

	q.ApplyTransitionResult(result)
	now := s.nowMs()
	if action == "continue" && resumePhaseIdx >= 0 {
		q.CurrentPhaseIdx = resumePhaseIdx
		q.StartCurrentPhase(now)
	}
	q.UpdatedAtMs = now

	if err := s.repo.Save(q); err != nil {
		return nil, fmt.Errorf("保存 quest 失败: %w", err)
	}

	// 发布事件
	s.publishBlockedResolved(q, action, opts)

	return q, nil
}

func (s *Service) publishBlockedResolved(q *QuestMeta, action string, opts ResolveBlockedOptions) {
	if s.events == nil {
		return
	}
	switch action {
	case "continue":
		s.events.Publish(q.ID, "", events.EvtRework, map[string]any{
			"from":                  "blocked",
			"action":                action,
			"comment":               opts.Comment,
			"resume_count":          q.ResumeCount,
			"max_turns_per_phase":   q.MaxTurnsPerPhaseOverride,
			"max_duration_quest_ms": q.MaxDurationMsOverride,
		})
	case "user-review":
		s.events.Publish(q.ID, "", events.EvtUserReview, map[string]any{
			"from":    "blocked",
			"comment": q.FinalComment,
		})
	case "cancel":
		s.events.Publish(q.ID, "", events.EvtQuestCancelled, map[string]any{
			"from":    "blocked",
			"comment": opts.Comment,
		})
	}
}

// ===== 用户终审相关操作 =====

// ResolveUserReviewOptions 用户终审选项。
type ResolveUserReviewOptions struct {
	Verdict model.QuestVerdict
	Comment string
	// Phase 1.5：用户终审溯源（见 mage-review-spec v0.2.2 §0.2.1）。
	// phase=-1 表示"非 phase 内决策"（user review 在 phase 管道外部）。
	ActorUserID string
}

// ResolveUserReview 处理用户终审。
// 根据 verdict 同步更新 phase 管道状态。
func (s *Service) ResolveUserReview(qid string, opts ResolveUserReviewOptions) (*QuestMeta, error) {
	q, err := s.repo.Load(qid)
	if err != nil {
		return nil, fmt.Errorf("加载委托失败: %w", err)
	}
	if q.Status != model.QuestStatusUserReview {
		return nil, fmt.Errorf("委托状态 %s，不在用户评审阶段", q.Status)
	}

	// 记录评审
	rv := &ReviewRecord{
		Ts:         s.nowMs(),
		Verdict:    opts.Verdict,
		Comment:    opts.Comment,
		ReviewedBy: "user",
	}
	// ReviewSource：user review = phase 外部决策，PhaseIdx=-1 是 sentinel（见 spec §0.2.1）。
	rv.Source = &ReviewSource{
		SourceRole:         "user",
		SourceAdventurerID: opts.ActorUserID,
		SourcePhaseIdx:     -1,
	}
	// StructuredReview 两层校验（spec §8.4）：user review 一般无 structured，但保持一致
	ApplyStructuredValidation(rv, nil, s.events, qid)
	if err := s.repo.AppendReview(qid, rv); err != nil {
		log.Printf("[error] append review for quest %s: %v", qid, err)
	}

	var result TransitionResult

	switch opts.Verdict {
	case model.VerdictPass:
		result, err = CompleteUserReview(q.Status, model.VerdictPass, opts.Comment)
		if err != nil {
			return nil, err
		}
	case model.VerdictRequestChange:
		if !q.CanRework() {
			return nil, fmt.Errorf("已达最大返工次数(%d)，无法继续返工", q.MaxRework)
		}
		result, err = RequestUserReviewRework(q.Status, opts.Comment, q.ReworkCount)
		if err != nil {
			return nil, err
		}
	case model.VerdictReject:
		result, err = CompleteUserReview(q.Status, model.VerdictReject, opts.Comment)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("未知 verdict: %s", opts.Verdict)
	}

	q.ApplyTransitionResult(result)
	if opts.Verdict == model.VerdictPass || opts.Verdict == model.VerdictReject {
		q.FinalizedBy = "user"
	}
	q.UpdatedAtMs = s.nowMs()

	// ===== Phase 管道同步 =====
	now := s.nowMs()
	switch opts.Verdict {
	case model.VerdictPass, model.VerdictReject:
		// 终态：确保所有 phase 都标记为完成
		q.EnsurePhases()
		for i := range q.Phases {
			if !q.Phases[i].IsTerminal() {
				q.Phases[i].Complete(now)
			}
		}
	case model.VerdictRequestChange:
		// 返工：回到 phase 0，重置并重新启动
		q.EnsurePhases()
		q.ReworkToPhase(0, now)
		q.StartCurrentPhase(now)
	}

	if err := s.repo.Save(q); err != nil {
		return nil, fmt.Errorf("保存 quest 失败: %w", err)
	}

	// 发布事件
	s.publishUserReviewResolved(q, opts)

	return q, nil
}

func (s *Service) publishUserReviewResolved(q *QuestMeta, opts ResolveUserReviewOptions) {
	if s.events == nil {
		return
	}
	switch opts.Verdict {
	case model.VerdictPass:
		s.events.Publish(q.ID, "", events.EvtQuestSuccess, map[string]any{
			"verdict": string(opts.Verdict),
			"comment": opts.Comment,
		})
	case model.VerdictRequestChange:
		s.events.Publish(q.ID, "", events.EvtRework, map[string]any{
			"rework_count": q.ReworkCount,
			"hints":        opts.Comment,
			"from":         "user_review",
		})
	case model.VerdictReject:
		s.events.Publish(q.ID, "", events.EvtQuestFailed, map[string]any{
			"verdict": string(opts.Verdict),
			"comment": opts.Comment,
		})
	}
}

// ===== 通用取消/失败操作 =====

// CancelQuest 取消一个 quest（从任何非终态取消）。
func (s *Service) CancelQuest(qid, reason string) (*QuestMeta, error) {
	q, err := s.repo.Load(qid)
	if err != nil {
		return nil, fmt.Errorf("加载委托失败: %w", err)
	}
	if q.IsTerminal() {
		return q, nil // 已经是终态，幂等返回
	}

	result, err := CancelQuest(q.Status, reason)
	if err != nil {
		return nil, err
	}

	q.ApplyTransitionResult(result)
	q.UpdatedAtMs = s.nowMs()

	// ===== Phase 管道同步 =====
	// 终态：标记所有未完成阶段为 failed
	q.EnsurePhases()
	now := s.nowMs()
	for i := range q.Phases {
		if !q.Phases[i].IsTerminal() {
			q.Phases[i].Fail(now)
		}
	}

	if err := s.repo.Save(q); err != nil {
		return nil, fmt.Errorf("保存 quest 失败: %w", err)
	}

	if s.events != nil {
		s.events.Publish(qid, "", events.EvtQuestCancelled, map[string]any{
			"reason": reason,
		})
	}

	return q, nil
}

// FailQuest 标记 quest 失败。
func (s *Service) FailQuest(qid, reason string) (*QuestMeta, error) {
	q, err := s.repo.Load(qid)
	if err != nil {
		return nil, fmt.Errorf("加载委托失败: %w", err)
	}
	if q.IsTerminal() {
		return q, nil // 已经是终态，幂等返回
	}

	result, err := FailQuest(q.Status, reason)
	if err != nil {
		return nil, err
	}

	q.ApplyTransitionResult(result)
	q.UpdatedAtMs = s.nowMs()

	// ===== Phase 管道同步 =====
	// 失败：标记所有未完成阶段为 failed
	q.EnsurePhases()
	now := s.nowMs()
	for i := range q.Phases {
		if !q.Phases[i].IsTerminal() {
			q.Phases[i].Fail(now)
		}
	}

	if err := s.repo.Save(q); err != nil {
		return nil, fmt.Errorf("保存 quest 失败: %w", err)
	}

	if s.events != nil {
		s.events.Publish(qid, "", events.EvtQuestFailed, map[string]any{
			"reason": reason,
		})
	}

	return q, nil
}

// ===== 超时阻塞操作（供 scheduler 使用） =====

// BlockUserReviewTimeout 将超时的 user_review quest 转为 blocked。
// 这是 scheduler 专用的操作，直接执行状态迁移。
func (s *Service) BlockUserReviewTimeout(qid string, timeoutMs int64) (bool, error) {
	q, err := s.repo.Load(qid)
	if err != nil {
		return false, err
	}
	if q.Status != model.QuestStatusUserReview {
		return false, nil
	}

	result, err := BlockQuest(q.Status, "等待用户终审超时")
	if err != nil {
		return false, nil // 迁移不合法，静默跳过
	}
	result.BlockedReasonCode = "user_confirm_timeout"

	q.ApplyTransitionResult(result)
	q.UpdatedAtMs = s.nowMs()

	if err := s.repo.Save(q); err != nil {
		return false, fmt.Errorf("保存 quest 失败: %w", err)
	}

	if s.events != nil {
		s.events.Publish(qid, "", events.EvtQuestBlocked, map[string]any{
			"reason":     "user_confirm_timeout",
			"timeout_ms": timeoutMs,
		})
	}

	return true, nil
}

// BlockRunningTimeout 将执行超时的 running/reviewing quest 转为 blocked。
// 这是 scheduler 专用的兜底操作——覆盖 rework 路径等 goroutine 失踪场景
// （launchRunningLoop 因 reserveQuestRuntime 返回 nil 静默失败，quest 状态
// running 但无 goroutine 推进，in-loop 的 max_duration/max_no_progress 检查
// 无从触发）。实测 qst_2606199899 因此卡死 127h。
//
// 与 BlockUserReviewTimeout 对称：scheduler 每分钟扫描 running/reviewing，
// 扣除 waiting_input 暂停时长后若超过 max_duration_per_quest_ms 则 block。
func (s *Service) BlockRunningTimeout(qid string, elapsedMs, limitMs int64) (bool, error) {
	q, err := s.repo.Load(qid)
	if err != nil {
		return false, err
	}
	if q.Status != model.QuestStatusRunning && q.Status != model.QuestStatusReviewing {
		return false, nil
	}
	if elapsedMs < limitMs {
		return false, nil
	}

	result, err := BlockQuest(q.Status, fmt.Sprintf("执行超时（已用 %dms 超过上限 %dms）", elapsedMs, limitMs))
	if err != nil {
		return false, nil // 迁移不合法，静默跳过
	}
	result.BlockedReasonCode = "duration_exceeded"

	q.ApplyTransitionResult(result)
	q.UpdatedAtMs = s.nowMs()

	if err := s.repo.Save(q); err != nil {
		return false, fmt.Errorf("保存 quest 失败: %w", err)
	}

	if s.events != nil {
		s.events.Publish(qid, "", events.EvtQuestBlocked, map[string]any{
			"reason":     "duration_exceeded",
			"elapsed_ms": elapsedMs,
			"limit_ms":   limitMs,
		})
	}

	return true, nil
}

// ===== 法师评审相关操作 =====

// MoveToUserReviewOptions 提交用户终审的选项。
type MoveToUserReviewOptions struct {
	Verdict   model.QuestVerdict // 法师评审结论（pass / request_changes / 空）
	Comment   string             // 评审意见
	Reason    string             // 为什么进入用户终审（事件 payload 用）
	FromPhase string             // 来源阶段（warrior / mage_review / blocked 等）
}

// MoveToUserReview 将 quest 从 reviewing 转入 user_review。
// 同时完成当前 agent phase。
func (s *Service) MoveToUserReview(qid string, opts MoveToUserReviewOptions) (*QuestMeta, error) {
	q, err := s.repo.Load(qid)
	if err != nil {
		return nil, fmt.Errorf("加载委托失败: %w", err)
	}

	// 支持从 reviewing 或 running 转入
	//   - reviewing: 正常评审路径（法师评审后 → 用户终审）
	//   - running: quick 模式跳过评审（剑士完成后直接 → 用户终审）
	if q.Status != model.QuestStatusReviewing && q.Status != model.QuestStatusRunning {
		return nil, fmt.Errorf("委托状态 %s，不能转入用户终审", q.Status)
	}

	q.FinalVerdict = opts.Verdict
	q.FinalComment = opts.Comment

	// 使用通用状态迁移（reviewing → user_review）
	result, err := MoveToUserReview(q.Status)
	if err != nil {
		return nil, err
	}

	q.ApplyTransitionResult(result)
	q.UpdatedAtMs = s.nowMs()

	// ===== Phase 管道同步 =====
	// - 从 reviewing 来：当前是法师阶段（phase 1），完成它
	// - 从 running 来（quick 模式）：当前是剑士阶段（phase 0），完成它
	q.EnsurePhases()
	if q.CurrentPhaseIdx >= 0 {
		q.CompleteCurrentPhase(s.nowMs())
	}
	// 推进到 user_review 阶段（phase 2）
	// 注意：如果 pipeline 只有 2 个 agent 阶段，user_review 可能不在 pipeline 中
	// 这里先不推进 CurrentPhaseIdx，保持与 quest 状态的一致性

	if err := s.repo.Save(q); err != nil {
		return nil, fmt.Errorf("保存 quest 失败: %w", err)
	}

	if s.events != nil {
		payload := map[string]any{
			"comment": opts.Comment,
		}
		if opts.Verdict != "" {
			payload["verdict"] = string(opts.Verdict)
		}
		if opts.Reason != "" {
			payload["reason"] = opts.Reason
		}
		if opts.FromPhase != "" {
			payload["from"] = opts.FromPhase
		}
		s.events.Publish(qid, "", events.EvtUserReview, payload)
	}

	return q, nil
}

// RequestReviewRework 法师评审要求返工（reviewing → running）。
// 同时重置 phases 并回到 phase 0。
func (s *Service) RequestReviewRework(qid string, hints string) (*QuestMeta, error) {
	return s.requestReviewReworkToPhase(qid, 0, hints)
}

// CompleteReviewOptions 完成法师评审的选项。
type CompleteReviewOptions struct {
	Verdict            model.QuestVerdict
	Comment            string
	FinalizedBy        string
	AutoPassedByPolicy string
	PolicyDecisionID   string
	// Phase 1.5：ReviewSource 溯源字段（见 mage-review-spec v0.2.2 §0.2.1 方案 A）。
	// 调用方有上下文就填；不填则 SourceIncomplete=true。
	AdventurerID string
	PhaseIdx     *int // 指针类型：nil 表示未提供（区别于 0 = warrior phase）
	SessionID    string
}

// CompleteReview 完成法师评审，直接得出最终结果（reviewing → success/failed）。
// 用于不需要用户终审的场景（如 context automation）。
func (s *Service) CompleteReview(qid string, opts CompleteReviewOptions) (*QuestMeta, error) {
	q, err := s.repo.Load(qid)
	if err != nil {
		return nil, fmt.Errorf("加载委托失败: %w", err)
	}

	// 记录评审
	rv := &ReviewRecord{
		Ts:         s.nowMs(),
		Verdict:    opts.Verdict,
		Comment:    opts.Comment,
		ReviewedBy: "mage",
	}
	// ReviewSource：service 路径（legacy 入口），按方案 A 填齐（见 mage-review-spec v0.2.2 §0.2.1）。
	rv.Source = &ReviewSource{
		SourceRole:         "mage",
		SourceClass:        "mage",
		SourceAdventurerID: opts.AdventurerID,
		SourceSessionID:    opts.SessionID,
	}
	if opts.PhaseIdx != nil {
		rv.Source.SourcePhaseIdx = *opts.PhaseIdx
	} else {
		rv.Source.SourcePhaseIdx = -1
		rv.Source.SourceIncomplete = true
	}
	if opts.AdventurerID == "" || opts.SessionID == "" {
		rv.Source.SourceIncomplete = true
	}
	// StructuredReview 两层校验（spec §8.4）：失败则丢弃 structured 部分 + 发 event
	ApplyStructuredValidation(rv, nil, s.events, qid)
	if err := s.repo.AppendReview(qid, rv); err != nil {
		log.Printf("[error] append review for quest %s: %v", qid, err)
	}

	result, err := CompleteReview(q.Status, opts.Verdict, opts.Comment)
	if err != nil {
		return nil, err
	}

	q.ApplyTransitionResult(result)
	if opts.FinalizedBy != "" {
		q.FinalizedBy = opts.FinalizedBy
	}
	q.AutoPassedByPolicy = opts.AutoPassedByPolicy
	q.PolicyDecisionID = opts.PolicyDecisionID
	q.UpdatedAtMs = s.nowMs()

	if err := s.repo.Save(q); err != nil {
		return nil, fmt.Errorf("保存 quest 失败: %w", err)
	}

	// 发布对应事件
	if s.events != nil {
		eventType := events.EvtQuestSuccess
		if opts.Verdict == model.VerdictReject {
			eventType = events.EvtQuestFailed
		}
		s.events.Publish(qid, "", eventType, map[string]any{
			"verdict": string(opts.Verdict),
			"comment": opts.Comment,
		})
	}

	return q, nil
}

// ===== 阶段流转操作 =====

type CompletePhaseOptions struct {
	PhaseIdx int
}

// CompletePhase 完成当前 agent 阶段并推进到下一阶段。
// 当前状态机仍只有 running/reviewing 两个 agent 状态；因此 phase 0 完成等价于进入 reviewing。
func (s *Service) CompletePhase(qid string, opts CompletePhaseOptions) (*QuestMeta, error) {
	q, err := s.repo.Load(qid)
	if err != nil {
		return nil, fmt.Errorf("加载委托失败: %w", err)
	}
	if opts.PhaseIdx != q.CurrentPhaseIdx {
		return nil, fmt.Errorf("phase %d 不是当前阶段 %d", opts.PhaseIdx, q.CurrentPhaseIdx)
	}
	q.EnsurePhases()
	if q.CurrentPhaseIdx != opts.PhaseIdx {
		return nil, fmt.Errorf("phase %d 不是当前阶段 %d", opts.PhaseIdx, q.CurrentPhaseIdx)
	}
	now := s.nowMs()
	if prev := q.PhaseRunAt(opts.PhaseIdx); prev != nil && !prev.IsTerminal() {
		prev.Complete(now)
	}
	if q.AdvanceToNextPhase() < 0 {
		return nil, fmt.Errorf("phase %d 完成后没有自动下一阶段", opts.PhaseIdx)
	}
	nextPhase := q.CurrentPhaseDef()
	if nextPhase == nil {
		return nil, fmt.Errorf("phase %d 完成后没有有效下一阶段定义", opts.PhaseIdx)
	}
	nextStatus := model.QuestStatusRunning
	if nextPhase.Role == PhaseRoleReview {
		nextStatus = model.QuestStatusReviewing
	}
	if q.Status != nextStatus {
		if err := checkTransition(q.Status, nextStatus); err != nil {
			return nil, err
		}
	}
	q.Status = nextStatus
	q.StartCurrentPhase(now)
	q.UpdatedAtMs = now
	if err := s.repo.Save(q); err != nil {
		return nil, fmt.Errorf("保存 quest 失败: %w", err)
	}
	if s.events != nil {
		s.events.Publish(qid, "", events.EvtPhaseChanged, map[string]any{
			"phase": q.CurrentPhaseIdx,
		})
	}
	return q, nil
}

type RequestReworkToPhaseOptions struct {
	TargetPhaseIdx int
	Hints          string
}

// RequestReworkToPhase 从评审阶段返工到指定阶段。
func (s *Service) RequestReworkToPhase(qid string, opts RequestReworkToPhaseOptions) (*QuestMeta, error) {
	return s.requestReviewReworkToPhase(qid, opts.TargetPhaseIdx, opts.Hints)
}

func (s *Service) requestReviewReworkToPhase(qid string, targetPhaseIdx int, hints string) (*QuestMeta, error) {
	q, err := s.repo.Load(qid)
	if err != nil {
		return nil, fmt.Errorf("加载委托失败: %w", err)
	}
	q.EnsurePhases()
	if targetPhaseIdx < 0 || targetPhaseIdx >= len(q.Phases) {
		return nil, fmt.Errorf("返工目标 phase %d 越界", targetPhaseIdx)
	}
	targetDef := q.Pipeline.At(targetPhaseIdx)
	if targetDef == nil {
		return nil, fmt.Errorf("返工目标 phase %d 没有定义", targetPhaseIdx)
	}
	if targetDef.Role == PhaseRoleReview || targetDef.Role == PhaseRoleHuman {
		return nil, fmt.Errorf("返工目标 phase %d 必须是执行阶段，got %s", targetPhaseIdx, targetDef.Role)
	}

	result, err := RequestReviewRework(q.Status, hints, q.ReworkCount)
	if err != nil {
		return nil, err
	}

	q.ApplyTransitionResult(result)
	now := s.nowMs()
	q.ReworkToPhase(targetPhaseIdx, now)
	q.StartCurrentPhase(now)
	q.UpdatedAtMs = now

	if err := s.repo.Save(q); err != nil {
		return nil, fmt.Errorf("保存 quest 失败: %w", err)
	}

	if s.events != nil {
		s.events.Publish(qid, "", events.EvtRework, map[string]any{
			"rework_count":     q.ReworkCount,
			"hints":            hints,
			"from":             "reviewing",
			"target_phase_idx": targetPhaseIdx,
			"target_phase":     targetDef.Name,
		})
	}

	return q, nil
}

type CompleteQuestOptions struct {
	Verdict            model.QuestVerdict
	Comment            string
	FinalizedBy        string
	AutoPassedByPolicy string
	PolicyDecisionID   string
	// Phase 1.5 溯源（透传给 CompleteReview）
	AdventurerID string
	PhaseIdx     *int
	SessionID    string
}

// CompleteQuest 直接完成 quest，用于平台托管路径（如 context automation）。
func (s *Service) CompleteQuest(qid string, opts CompleteQuestOptions) (*QuestMeta, error) {
	return s.CompleteReview(qid, CompleteReviewOptions{
		Verdict:            opts.Verdict,
		Comment:            opts.Comment,
		FinalizedBy:        opts.FinalizedBy,
		AutoPassedByPolicy: opts.AutoPassedByPolicy,
		PolicyDecisionID:   opts.PolicyDecisionID,
		AdventurerID:       opts.AdventurerID,
		PhaseIdx:           opts.PhaseIdx,
		SessionID:          opts.SessionID,
	})
}

type CompleteExecuteOptions struct {
	Comment               string
	FinalizedBy           string
	AutoCompletedByPolicy string
	PolicyDecisionID      string
}

// CompleteExecute 直接从运行状态完成 quest（跳过评审和用户终审）。
// 用于 quick 模式无副作用 auto-complete 等场景。
func (s *Service) CompleteExecute(qid string, opts CompleteExecuteOptions) (*QuestMeta, error) {
	q, err := s.repo.Load(qid)
	if err != nil {
		return nil, fmt.Errorf("加载委托失败: %w", err)
	}

	result, err := CompleteExecute(q.Status, opts.Comment)
	if err != nil {
		return nil, err
	}

	q.ApplyTransitionResult(result)
	if opts.FinalizedBy != "" {
		q.FinalizedBy = opts.FinalizedBy
	}
	if opts.AutoCompletedByPolicy != "" {
		q.AutoCompletedByPolicy = opts.AutoCompletedByPolicy
	}
	q.PolicyDecisionID = opts.PolicyDecisionID
	q.UpdatedAtMs = s.nowMs()

	// 完成当前执行阶段
	q.EnsurePhases()
	now := s.nowMs()
	if phase := q.PhaseRunAt(q.CurrentPhaseIdx); phase != nil && !phase.IsTerminal() {
		phase.Complete(now)
	}

	if err := s.repo.Save(q); err != nil {
		return nil, fmt.Errorf("保存 quest 失败: %w", err)
	}

	if s.events != nil {
		s.events.Publish(qid, "", events.EvtQuestSuccess, map[string]any{
			"verdict":                  string(model.VerdictPass),
			"comment":                  opts.Comment,
			"auto_completed_by_policy": opts.AutoCompletedByPolicy,
		})
	}

	return q, nil
}

// StartReviewPhase 从 running 转入 reviewing（剑士阶段结束）。
// 同时完成 phase 0，启动 phase 1。
func (s *Service) StartReviewPhase(qid string) (*QuestMeta, error) {
	q, err := s.repo.Load(qid)
	if err != nil {
		return nil, fmt.Errorf("加载委托失败: %w", err)
	}

	result, err := StartReviewPhase(q.Status)
	if err != nil {
		return nil, err
	}

	q.ApplyTransitionResult(result)
	q.UpdatedAtMs = s.nowMs()

	// ===== Phase 管道同步 =====
	// phase 0 完成，推进到 phase 1 并启动
	q.EnsurePhases()
	now := s.nowMs()
	if q.CurrentPhaseIdx == 0 {
		q.CompleteCurrentPhase(now)
	}
	q.AdvanceToNextPhase()
	q.StartCurrentPhase(now)

	if err := s.repo.Save(q); err != nil {
		return nil, fmt.Errorf("保存 quest 失败: %w", err)
	}

	if s.events != nil {
		s.events.Publish(qid, "", events.EvtPhaseChanged, map[string]any{
			"phase": "reviewing",
		})
	}

	return q, nil
}

// StartQuest 启动 quest（pending → running）。
// 设置 started_at 和 max_rework 默认值。
// 同时初始化 phases 并启动 phase 0。
func (s *Service) StartQuest(qid string, maxRework int) (*QuestMeta, error) {
	q, err := s.repo.Load(qid)
	if err != nil {
		return nil, fmt.Errorf("加载委托失败: %w", err)
	}

	result, err := StartQuest(q.Status)
	if err != nil {
		return nil, err
	}

	q.ApplyTransitionResult(result)
	q.UpdatedAtMs = s.nowMs()
	if maxRework > 0 && q.MaxRework == 0 {
		q.MaxRework = maxRework
	}

	// ===== Phase 管道同步 =====
	// 确保 phases 初始化，启动 phase 0
	q.EnsurePhases()
	q.CurrentPhaseIdx = 0
	q.StartCurrentPhase(s.nowMs())
	// 同步冒险者 ID（从 legacy 字段写入 phase 0）
	if q.WarriorID != "" {
		q.SetPhaseAdventurer(0, q.WarriorID)
	}
	if q.MageID != "" {
		q.SetPhaseAdventurer(1, q.MageID)
	}

	if err := s.repo.Save(q); err != nil {
		return nil, fmt.Errorf("保存 quest 失败: %w", err)
	}

	if s.events != nil {
		s.events.Publish(qid, "", events.EvtQuestStarted, map[string]any{
			"query":      q.Query,
			"type":       q.Type,
			"warrior_id": q.WarriorID,
			"mage_id":    q.MageID,
		})
	}

	return q, nil
}

// BlockQuestOptions 阻塞选项。
type BlockQuestOptions struct {
	Reason     string // 阻塞原因（写入 BlockedReason 字段）
	ReasonCode string // 原因代码（事件 payload 用，如 duration_exceeded）
	Phase      string // 发生在哪个阶段
	Category   string // 错误性质分类（transient/auth/configuration 等，恢复策略用）
	Details    map[string]any
}

// BlockQuest 将 quest 转为 blocked 状态。
// 适用于预算超限、超时等场景。
func (s *Service) BlockQuest(qid string, opts BlockQuestOptions) (*QuestMeta, error) {
	q, err := s.repo.Load(qid)
	if err != nil {
		return nil, fmt.Errorf("加载委托失败: %w", err)
	}
	if q.IsTerminal() {
		return nil, fmt.Errorf("委托已处于终态 %s，不能阻塞", q.Status)
	}

	result, err := BlockQuest(q.Status, opts.Reason)
	if err != nil {
		return nil, err
	}
	result.BlockedReasonCode = opts.ReasonCode
	result.BlockedCategory = opts.Category

	q.ApplyTransitionResult(result)
	q.UpdatedAtMs = s.nowMs()

	if err := s.repo.Save(q); err != nil {
		return nil, fmt.Errorf("保存 quest 失败: %w", err)
	}

	if s.events != nil {
		payload := map[string]any{
			"reason": opts.ReasonCode,
		}
		if opts.Phase != "" {
			payload["phase"] = opts.Phase
		}
		if opts.Category != "" {
			payload["category"] = opts.Category
		}
		if opts.Details != nil {
			for k, v := range opts.Details {
				payload[k] = v
			}
		}
		s.events.Publish(qid, "", events.EvtQuestBlocked, payload)
	}

	return q, nil
}

// CompleteAutoApplySuccess 自动应用成功，直接完成任务。
// 从 user_review → success，并设置 apply 相关字段。
// 这是一个组合操作：状态迁移 + 字段更新 + 事件发布。
func (s *Service) CompleteAutoApplySuccess(qid string, comment string) (*QuestMeta, error) {
	q, err := s.repo.Load(qid)
	if err != nil {
		return nil, fmt.Errorf("加载委托失败: %w", err)
	}

	result, err := CompleteUserReview(q.Status, model.VerdictPass, comment)
	if err != nil {
		return nil, err
	}

	q.ApplyTransitionResult(result)
	q.UpdatedAtMs = s.nowMs()

	// 设置 apply 相关字段
	q.Applied = true
	q.ApplyStatus = model.ApplyStatusApplied
	q.ApplyError = ""
	q.ApplyFailedAtMs = 0

	if err := s.repo.Save(q); err != nil {
		return nil, fmt.Errorf("保存 quest 失败: %w", err)
	}

	if s.events != nil {
		s.events.Publish(qid, "", events.EvtQuestSuccess, map[string]any{
			"verdict":    string(model.VerdictPass),
			"comment":    comment,
			"auto_apply": true,
		})
	}

	return q, nil
}

// ===== 查询操作 =====

// GetQuest 获取单个 quest。
func (s *Service) GetQuest(qid string) (*QuestMeta, error) {
	return s.repo.Load(qid)
}

// ListQuests 列出所有 quest。
func (s *Service) ListQuests() ([]*QuestMeta, error) {
	return s.repo.List()
}

// ListByStatus 按状态列出 quest。
func (s *Service) ListByStatus(status model.QuestStatus) ([]*QuestMeta, error) {
	return s.repo.ListByStatus(status)
}

// ===== 字段更新（非状态机变更） =====

// UpdateFailureAttribution 更新 quest 的失败归因摘要。
// 这是字段更新，不涉及状态机迁移。
func (s *Service) UpdateFailureAttribution(qid string, attr *FailureAttribution) error {
	q, err := s.repo.Load(qid)
	if err != nil {
		return fmt.Errorf("加载委托失败: %w", err)
	}
	q.FailureAttribution = attr
	q.UpdatedAtMs = s.nowMs()
	if err := s.repo.Save(q); err != nil {
		return fmt.Errorf("保存失败归因失败: %w", err)
	}
	return nil
}

// ===== 辅助函数 =====

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
