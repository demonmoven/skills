package orchestrator

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/domain/quest"
	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
	"code.byted.org/lihuanyu.0w0/gloop/internal/executor"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/policy"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"strconv"
)

const defaultWaitingInputTimeoutMs int64 = 24 * 60 * 60 * 1000

func agentContextEnv(workDir string, ctx fsstore.AgentContext) []string {
	return []string{
		"GLOOP_DATA_DIR=" + ctx.DataDir,
		"GLOOP_QUEST_ID=" + ctx.QuestID,
		"GLOOP_SESSION_ID=" + ctx.SessionID,
		"GLOOP_PHASE=" + ctx.PhaseName,
		"GLOOP_PHASE_INDEX=" + fmt.Sprintf("%d", ctx.Phase),
		"GLOOP_ADVENTURER_ID=" + ctx.AdventurerID,
		"GLOOP_ADVENTURER_CLASS=" + ctx.AdventurerClass,
		"GLOOP_AGENT_ID=" + ctx.AgentID,
		"GLOOP_PHASE_ROLE=" + ctx.PhaseRole,
		"GLOOP_WORKSPACE_PATH=" + workDir,
		"GLOOP_SIGNAL_WORKSPACE_PATH=" + ctx.SignalWorkspacePath,
		"GLOOP_CONTEXT=" + filepath.Join(workDir, fsstore.AgentContextRel),
	}
}

func attachQuestInputs(root *fsstore.Root, q *fsstore.QuestMeta, msg *executor.Message) {
	if q == nil || msg == nil || len(q.Inputs) == 0 {
		return
	}
	for _, artifact := range q.Inputs {
		msg.Parts = append(msg.Parts, executor.ContentBlock{
			Type:         "artifact",
			ArtifactID:   artifact.ID,
			Kind:         artifact.Kind,
			MIME:         artifact.MIME,
			Name:         artifact.Name,
			Size:         artifact.Size,
			SHA256:       artifact.SHA256,
			StoragePath:  artifact.StoragePath,
			AbsolutePath: questArtifactAbsPath(root, q, artifact),
		})
	}
}

func questArtifactAbsPath(root *fsstore.Root, q *fsstore.QuestMeta, artifact fsstore.QuestArtifact) string {
	if root == nil || q == nil || artifact.StoragePath == "" {
		return ""
	}
	return filepath.Join(root.Path(), fsstore.SubdirQuests, q.ID, filepath.FromSlash(artifact.StoragePath))
}

// ==================== 宏循环：Quest 级状态机 ====================
//
//  pending → running → reviewing → user_review → success / failed
//                ↑           ↓             ↓
//                └── rework ─┘── rework ───┘
//
// 每轮循环：剑士执行 → 法师评审 → 决定下一步
//
// Phase 2 管道化：阶段由 pipeline 索引驱动，
// 每个 Agent 阶段通过统一的 runAgentPhase(phaseIdx) 执行。
// 目前仍保持两阶段硬编码顺序，但底层已经是统一引擎。

func (e *Engine) runMacroLoop(ctx context.Context, qid string) {
	qs := fsstore.NewQuestStore(e.root)
	q, err := qs.LoadQuest(qid)
	if err != nil {
		e.log.Error("宏循环：加载 quest 失败", "qid", qid, "err", err)
		return
	}

	// 加载冒险者（legacy 路径）；agent-direct phase binding 允许没有 adventurer。
	var warrior *fsstore.AdventurerFile
	if q.WarriorID != "" {
		var wErr error
		warrior, wErr = e.root.GetAdventurer(q.WarriorID)
		if wErr != nil {
			e.log.Error("加载剑士失败", "qid", qid, "err", wErr)
			e.failQuest(qid, "剑士冒险者不存在")
			return
		}
	} else if q.ExecuteAgentID == "" && phaseTaskForActor(q, 0).AgentID == "" {
		e.failQuest(qid, "剑士冒险者不存在")
		return
	}
	// quick 强度或 MageID 为空时跳过法师评审（单阶段交付）
	skipMageReview := (q.MageID == "" && q.ReviewAgentID == "" && phaseTaskForActor(q, 1).AgentID == "") || q.Intensity == model.QuestIntensityQuick
	var mage *fsstore.AdventurerFile
	if !skipMageReview && q.MageID != "" {
		var mErr error
		mage, mErr = e.root.GetAdventurer(q.MageID)
		if mErr != nil {
			e.log.Error("加载法师失败", "qid", qid, "err", mErr)
			e.failQuest(qid, "法师冒险者不存在")
			return
		}
	}

	// 主循环：执行 → 评审 → 返工
	//
	// 恢复语义：如果 quest 已经处于 reviewing 状态（如服务中断后恢复），
	// 则跳过第一轮剑士阶段，直接进入法师评审。返工循环不受影响。
	skipWarrior := q.Status == model.QuestStatusReviewing
	if skipWarrior {
		e.log.Info("宏循环恢复：从 reviewing 状态继续，跳过剑士阶段", "qid", qid, "rework_count", q.ReworkCount)
	}

	for {
		// Slice 3: 升档后可能补分配了 mage / 解除了 quick，每次迭代重新评估。
		skipMageReview = (q.MageID == "" && q.ReviewAgentID == "" && phaseTaskForActor(q, 1).AgentID == "") || q.Intensity == model.QuestIntensityQuick
		if !skipMageReview && (mage == nil || mage.ID != q.MageID) {
			if q.MageID != "" {
				if m, mErr := e.root.GetAdventurer(q.MageID); mErr == nil {
					mage = m
				}
			}
		}
		nextQ, keepGoing := e.runMacroIteration(ctx, qs, q, warrior, mage, skipWarrior, skipMageReview)
		if nextQ != nil {
			q = nextQ
		}
		skipWarrior = q.Status == model.QuestStatusReviewing
		if !keepGoing {
			return
		}
	}
}

func (e *Engine) runMacroIteration(
	ctx context.Context,
	qs *fsstore.QuestStore,
	q *fsstore.QuestMeta,
	warrior, mage *fsstore.AdventurerFile,
	skipWarrior bool,
	skipMageReview bool,
) (*fsstore.QuestMeta, bool) {
	qid := q.ID
	select {
	case <-ctx.Done():
		return q, false
	default:
	}

	maxQuestDuration := e.maxDurationPerQuestMs(q)
	if maxQuestDuration > 0 && q.StartedAtMs > 0 {
		nowMs := fsstore.NowMs()
		elapsed := questDurationElapsedMs(qs, q, nowMs)
		if elapsed >= maxQuestDuration {
			e.log.Warn("Quest 总时长超限，进入阻塞状态", "qid", qid,
				"elapsed_ms", elapsed, "max_ms", maxQuestDuration)
			_, _ = e.questService.BlockQuest(qid, quest.BlockQuestOptions{
				Reason:     fmt.Sprintf("任务总时长超限：%dms / %dms", elapsed, maxQuestDuration),
				ReasonCode: "duration_exceeded",
				Phase:      "quest",
				Details: map[string]any{
					"elapsed_ms":      elapsed,
					"wall_elapsed_ms": nowMs - q.StartedAtMs,
					"max_ms":          maxQuestDuration,
				},
			})
			if blocked, err := qs.LoadQuest(qid); err == nil {
				e.recordRecoveryDecision(blocked)
			}
			return q, false
		}
	}

	currentPhase := phaseDefForQuest(q, q.CurrentPhaseIdx())
	if currentPhase == nil {
		e.failQuest(qid, "当前阶段定义不存在")
		return q, false
	}
	if currentPhase.Role == quest.PhaseRoleReview && mage == nil && q.ReviewAgentID == "" && phaseTaskForActor(q, currentPhase.Index).AgentID == "" {
		e.failQuest(qid, "法师冒险者不存在")
		return q, false
	}
	switch currentPhase.Role {
	case quest.PhaseRoleExecute:
		_, nextQ, done := e.runWarriorStep(ctx, qs, q, warrior, currentPhase, skipWarrior, skipMageReview)
		if done {
			return nextQ, false
		}
		if nextQ != nil {
			q = nextQ
		}
		return q, true
	case quest.PhaseRoleReview:
		warriorOutput := ""
		if prev := previousExecutePhaseDef(q, currentPhase.Index); prev != nil {
			if artifact, err := qs.PhaseArtifactForReview(qid, phaseSessionID(q, prev)); err == nil && strings.TrimSpace(artifact.Assistant) != "" {
				warriorOutput = artifact.Assistant
			}
		}
		if !skipMageReview {
			e.runPreCompletionProcessors(ctx, qid, q, currentPhase, "before_review")
		}
		verdict, comment, hints, score, rv, done := e.runMageReviewStep(ctx, qs, q, mage, currentPhase, warriorOutput)
		if done {
			return q, false
		}
		return e.handleReviewOutcome(ctx, qs, q, warrior, mage, rv, verdict, comment, hints, score)
	default:
		e.failQuest(qid, fmt.Sprintf("不支持的阶段角色: %s", currentPhase.Role))
		return q, false
	}
}

// escalateDirectToChecked 把 Direct quest 升档至 Checked：升 workflow_mode、
// 补分配 mage、恢复两阶段 pipeline、解除 quick intensity。任一步失败返回 error，
// 调用方负责 block 委托（绝不回退 quick auto-complete，避免静默放行外部副作用）。
func (e *Engine) escalateDirectToChecked(ctx context.Context, qs *fsstore.QuestStore, q *fsstore.QuestMeta) (*fsstore.QuestMeta, error) {
	qid := q.ID
	if _, err := e.upgradeWorkflowModeDuringRuntime(ctx, qid, model.WorkflowModeChecked,
		"Direct 委托检测到外部写副作用，自动升档至 Checked 审查"); err != nil {
		return nil, fmt.Errorf("UpgradeWorkflowMode: %w", err)
	}
	// reload：UpgradeWorkflowMode 已保存
	fresh, err := qs.LoadQuest(qid)
	if err != nil {
		return nil, fmt.Errorf("reload after upgrade: %w", err)
	}
	// 补分配 mage（quick/run 模式下 MageID 为空）
	if fresh.MageID == "" {
		mageAdv, mErr := e.pickAdventurer("", model.ClassMage)
		if mErr != nil {
			return nil, fmt.Errorf("assign mage: %w", mErr)
		}
		fresh.MageID = mageAdv.ID
		e.log.Info("外部写升档：补分配法师", "qid", qid, "mage_id", mageAdv.ID)
	}
	// 恢复两阶段 pipeline（run 模式是单阶段 PipelineName=run + PhaseCount=1）
	if fresh.PhaseCount < 2 || fresh.PipelineName == "run" {
		fresh.PipelineName = fsstore.DefaultPipelineName
		fresh.PipelineDef = fsstore.PhaseDefsFromDomain(quest.DefaultPipeline())
		fresh.PhaseCount = len(fresh.PipelineDef)
		fresh.PipelineDefHash = ""
		// 清空 Phases，让 EnsurePhases（LoadQuest 时调用）按新 pipeline 重建。
		// quest.Status=running → phase 0=running, phase 1=pending。
		fresh.Phases = nil
	}
	fresh.Intensity = model.QuestIntensityStandard
	fresh.WorkflowMode = model.WorkflowModeChecked
	if err := qs.SaveQuest(fresh); err != nil {
		return nil, fmt.Errorf("save quest: %w", err)
	}
	return fresh, nil
}

func (e *Engine) runWarriorStep(
	ctx context.Context,
	qs *fsstore.QuestStore,
	q *fsstore.QuestMeta,
	warrior *fsstore.AdventurerFile,
	phase *quest.PhaseDef,
	skipWarrior bool,
	skipMageReview bool,
) (string, *fsstore.QuestMeta, bool) {
	qid := q.ID
	if skipWarrior {
		artifactKey := fmt.Sprintf("warrior_%d", q.ReworkCount)
		if artifact, err := qs.PhaseArtifactForReview(qid, artifactKey); err == nil && strings.TrimSpace(artifact.Assistant) != "" {
			e.publish(qid, "", events.EvtQuestNote, map[string]any{
				"note":  "服务中断后自动恢复，继续法师评审阶段",
				"phase": "reviewing",
				"mode":  "recovery",
			})
			return artifact.Assistant, q, false
		}
		e.log.Warn("恢复执行：未找到剑士阶段 artifact", "qid", qid, "artifact_key", artifactKey)
		e.publish(qid, "", events.EvtQuestNote, map[string]any{
			"note":  "服务中断后自动恢复，继续法师评审阶段",
			"phase": "reviewing",
			"mode":  "recovery",
		})
		return "(剑士阶段已完成，但输出摘要不可用；请基于工作区实际文件状态进行评审)", q, false
	}

	phaseIdx := 0
	if phase != nil {
		phaseIdx = phase.Index
	}
	warriorOutcome := e.runAgentPhase(ctx, q, warrior, phaseIdx, phaseInput{})
	warriorOutput := warriorOutcome.Content
	warriorPhaseEnd := warriorOutcome.PhaseEnded
	if warriorOutcome.Err != nil {
		if ctx.Err() != nil {
			e.log.Info("剑士阶段因上下文取消而停止", "qid", qid, "err", warriorOutcome.Err)
			return "", q, true
		}
		e.log.Error("剑士阶段失败", "qid", qid, "err", warriorOutcome.Err)
		blockedResult := PhaseSignal{
			OK:           false,
			Message:      "agent_phase_error",
			PhaseEnded:   true,
			PhaseVerdict: "blocked",
			PhaseComment: fmt.Sprintf("剑士执行失败：%v", warriorOutcome.Err),
			Data:         map[string]any{"reason": "agent_phase_error", "category": classifyAgentError(warriorOutcome.Err), "error": warriorOutcome.Err.Error()},
		}
		blocked := e.blockQuestOrFail(qs, qid, quest.BlockQuestOptions{
			Reason:     blockedResult.PhaseComment,
			ReasonCode: blockedReasonCode(blockedResult),
			Phase:      "warrior",
			Category:   blockedCategory(blockedResult),
		}, blockedResult, "warrior")
		return warriorOutput, blocked, true
	}
	curWarriorSID := fmt.Sprintf("warrior_%d", q.ReworkCount)
	if artifact, err := qs.PhaseArtifactForReview(qid, curWarriorSID); err == nil && strings.TrimSpace(artifact.Assistant) != "" {
		warriorOutput = artifact.Assistant
	}
	if warriorPhaseEnd.PhaseEnded && warriorPhaseEnd.PhaseVerdict == string(model.QuestStatusWaitingInput) {
		e.log.Info("剑士请求用户输入，进入 waiting_input", "qid", qid)
		q.WaitingInput = waitingInputStateFromSignal(warriorPhaseEnd, phaseIdx)
		q.Status = model.QuestStatusWaitingInput
		q.UpdatedAtMs = fsstore.NowMs()
		if err := qs.SaveQuest(q); err != nil {
			e.log.Error("保存 waiting_input 状态失败", "qid", qid, "err", err)
		}
		// Record event durably, then write ThreadPost, then publish (avoid SSE refetch race).
		evPayload := map[string]any{
			"question_id": q.WaitingInput.QuestionID,
			"question":    q.WaitingInput.QuestionText,
			"phase_idx":   q.WaitingInput.PhaseIdx,
		}
		ev, evErr := e.recordQuestEventRequired(qid, "", events.EvtQuestWaitingInput, evPayload)
		if evErr != nil {
			e.log.Error("record waiting_input event failed (macro loop)", "qid", qid, "err", evErr)
		} else {
			e.projectWaitingInputRaised(qid, q.WaitingInput.QuestionText, ev.ID, "")
			e.publishRecordedEvent(ev)
		}
		return warriorOutput, q, true
	}
	if warriorPhaseEnd.PhaseEnded && warriorPhaseEnd.PhaseVerdict == "blocked" {
		e.log.Warn("剑士阶段预算超限，进入阻塞状态", "qid", qid, "reason", warriorPhaseEnd.Message)
		blocked := e.blockQuestOrFail(qs, qid, quest.BlockQuestOptions{
			Reason:     warriorPhaseEnd.PhaseComment,
			ReasonCode: blockedReasonCode(warriorPhaseEnd),
			Phase:      "warrior",
			Category:   blockedCategory(warriorPhaseEnd),
		}, warriorPhaseEnd, "warrior")
		return warriorOutput, blocked, true
	}
	if warriorPhaseEnd.PhaseEnded {
		e.persistMakerReport(qid, curWarriorSID, phaseIdx, warriorPhaseEnd)
		e.extractWarriorDeliverables(qs, q, warriorPhaseEnd)
		// HOTL Feed 兜底：把 warrior 的 phase done summary 提升为一等公民字段
		if ws := warriorPhaseEnd.PhaseComment; ws != "" && q.WarriorSummary != ws {
			q.WarriorSummary = ws
			_ = qs.SaveQuest(q)
		}
		impact := extractImpactFromSignal(warriorPhaseEnd)
		if !impact.IsEmpty() {
			// 返工后新 impact 覆盖旧值：HOTL 常态是返工，人需要看到最新一轮的影响声明
			q.ImpactSummary = impact
			if err := qs.SaveQuest(q); err != nil {
				e.log.Warn("保存 ImpactSummary 失败", "qid", qid, "err", err)
			} else {
				e.log.Info("ImpactSummary 已记录", "qid", qid, "what_changed_len", len(impact.WhatChanged))
			}
		} else {
			// HOTL 要求 agent 收尾必填 --impact；空值是异常信号，log warn 让人能在日志发现
			e.log.Warn("warrior 阶段结束但未声明影响", "qid", qid, "hint", "HOTL 要求 gloop phase done --impact")
		}

		// Slice 3: Direct 外部写自动升档 — 检测到 external_side_effect 时，
		// 升档至 Checked + 补分配 mage + 恢复两阶段 pipeline + 解除 quick，
		// 让 Checker 先审。不 MoveToUserReview：后续走 CompletePhase → reviewing → mage review。
		// 安全约束：一旦确认 external_side_effect，绝不回退 quick auto-complete；
		// 升档任一步失败 → block 委托（reason_code=checked_escalation_unavailable）。
		if freshQ, _ := qs.LoadQuest(qid); freshQ != nil {
			resolvedMode := e.resolveWorkflowMode(freshQ)
			effectType := policy.NewFactBuilder(e.root).EffectType(freshQ)
			if effectType == policy.EffectExternalSideEffect && resolvedMode == model.WorkflowModeDirect {
				e.log.Info("Direct 委托检测到外部写副作用，升档至 Checked", "qid", qid, "effect_type", effectType)
				escalated, escErr := e.escalateDirectToChecked(ctx, qs, freshQ)
				if escErr != nil {
					e.log.Error("外部写升档失败，block 委托（不回退 auto-complete）", "qid", qid, "err", escErr)
					blocked := e.blockQuestOrFail(qs, qid, quest.BlockQuestOptions{
						Reason:     "外部写升档失败，无法进入 Checker 审查: " + escErr.Error(),
						ReasonCode: "checked_escalation_unavailable",
						Phase:      "warrior",
					}, PhaseSignal{OK: false, Message: "checked_escalation_unavailable"}, "system")
					return warriorOutput, blocked, true
				}
				q = escalated
				skipMageReview = false
			}
		}
	}
	if skipMageReview {
		e.log.Info("quick 模式：跳过法师评审，直接进入用户终审", "qid", qid)
		summary := warriorOutput
		if warriorPhaseEnd.PhaseEnded && warriorPhaseEnd.PhaseComment != "" {
			summary = warriorPhaseEnd.PhaseComment
		}

		// run 模式：warrior done = quest done，无需 review 或 policy
		e.refreshDiffSummaryForPolicy(ctx, qid)

		qUpdated, cErr := e.questService.CompleteExecute(qid, quest.CompleteExecuteOptions{
			Comment:     summary,
			FinalizedBy: "run_mode_auto",
		})
		if cErr != nil {
			e.log.Warn("run mode auto-complete 失败，blocked", "qid", qid, "err", cErr)
			_, _ = e.questService.BlockQuest(qid, quest.BlockQuestOptions{
				Reason: "auto-complete 失败: " + cErr.Error(), ReasonCode: "complete_failed",
			})
		} else {
			e.awardExp(q, warrior, nil, model.VerdictPass)
			e.runAfterCommitSubscribers(qid, "diff_summary")
			e.cleanupWorkspaceIfTerminal(qid)
			q = fsstore.QuestMetaFromDomain(qUpdated)
		}
		return warriorOutput, q, true
	}
	qUpdatedDomain, err := e.questService.CompletePhase(qid, quest.CompletePhaseOptions{PhaseIdx: phaseIdx})
	if err != nil {
		e.log.Warn("状态迁移失败", "qid", qid, "from", q.Status, "to", model.QuestStatusReviewing, "err", err)
		return warriorOutput, q, false
	}
	return warriorOutput, fsstore.QuestMetaFromDomain(qUpdatedDomain), false
}

func (e *Engine) runMageReviewStep(
	ctx context.Context,
	qs *fsstore.QuestStore,
	q *fsstore.QuestMeta,
	mage *fsstore.AdventurerFile,
	phase *quest.PhaseDef,
	warriorOutput string,
) (model.QuestVerdict, string, string, int, *fsstore.ReviewRecord, bool) {
	qid := q.ID
	phaseIdx := 1
	if phase != nil {
		phaseIdx = phase.Index
	}
	mageOutcome := e.runAgentPhase(ctx, q, mage, phaseIdx, phaseInput{PreviousOutput: warriorOutput})
	magePhaseEnd := mageOutcome.PhaseEnded
	if magePhaseEnd.PhaseEnded && magePhaseEnd.PhaseVerdict == "blocked" {
		e.log.Warn("法师阶段被阻塞，进入阻塞状态", "qid", qid, "reason", magePhaseEnd.Message)
		e.blockQuestOrFail(qs, qid, quest.BlockQuestOptions{
			Reason:     magePhaseEnd.PhaseComment,
			ReasonCode: blockedReasonCode(magePhaseEnd),
			Phase:      "mage_review",
			Category:   blockedCategory(magePhaseEnd),
		}, magePhaseEnd, "mage")
		return "", "", "", 0, nil, true
	}
	verdict, comment, hints, score, structuredRaw, reviewErr := e.reviewVerdictFromOutcome(mageOutcome)
	if reviewErr != nil {
		if ctx.Err() != nil {
			e.log.Info("法师阶段因上下文取消而停止", "qid", qid, "err", reviewErr)
			return "", "", "", 0, nil, true
		}
		e.log.Error("法师阶段失败", "qid", qid, "err", reviewErr)
		blockedResult := PhaseSignal{
			OK:           false,
			Message:      "agent_phase_error",
			PhaseEnded:   true,
			PhaseVerdict: "blocked",
			PhaseComment: fmt.Sprintf("法师评审失败：%v", reviewErr),
			Data:         map[string]any{"reason": "agent_phase_error", "category": classifyAgentError(reviewErr), "error": reviewErr.Error()},
		}
		e.blockQuestOrFail(qs, qid, quest.BlockQuestOptions{
			Reason:     blockedResult.PhaseComment,
			ReasonCode: blockedReasonCode(blockedResult),
			Phase:      "mage_review",
			Category:   blockedCategory(blockedResult),
		}, blockedResult, "mage")
		return "", "", "", 0, nil, true
	}
	reviewedBy := ""
	if mage != nil {
		reviewedBy = mage.ID
	} else if agentID := reviewAgentIDForPhase(q, phaseIdx); agentID != "" {
		reviewedBy = "agent:" + agentID
	}
	rv := &fsstore.ReviewRecord{
		Ts:           fsstore.NowMs(),
		Verdict:      verdict,
		Comment:      comment,
		RewriteHints: hints,
		ReviewedBy:   reviewedBy,
		Score:        score,
	}
	if progress := fsstore.ParseRepairProgress(comment); progress.HasData {
		rv.RepairCount = progress.Repaired
		rv.TotalIssueCount = progress.Total
	}
	// ReviewSource：macro 路径是唯一能全量填齐 5 字段的路径（见 mage-review-spec v0.2.2 §0.2.1）。
	rv.Source = buildMageReviewSource(q, mage, phaseIdx)
	// StructuredReview 两层校验（spec §8.4）：解析 → 校验 → 通过则保留 raw JSON，失败则丢弃 + 发 event
	if len(structuredRaw) > 0 {
		if sr, pErr := quest.ParseStructuredReview(structuredRaw); pErr == nil && sr != nil {
			if vErrs := quest.ValidateStructuredReview(sr); len(vErrs) == 0 {
				rv.StructuredReview = structuredRaw
			} else {
				payload := quest.BuildStructuredInvalidPayload(structuredRaw, sr, vErrs, rv.Ts, rv.ReviewedBy)
				e.publish(qid, "", quest.StructuredInvalidEventType, quest.PayloadToMap(payload))
				e.log.Warn("法师 structured_review 校验失败，已丢弃 structured 部分", "qid", qid, "errors", len(vErrs))
			}
		} else if pErr != nil {
			// JSON 解析失败：无法解析成 StructuredReview，发 event
			payload := quest.BuildStructuredInvalidPayload(structuredRaw, nil, []quest.StructuredValidationError{
				{Code: quest.ErrTypeMismatch, Path: "", Message: "JSON parse failed: " + pErr.Error()},
			}, rv.Ts, rv.ReviewedBy)
			e.publish(qid, "", quest.StructuredInvalidEventType, quest.PayloadToMap(payload))
			e.log.Warn("法师 structured_review JSON 解析失败", "qid", qid, "err", pErr)
		}
	}
	if err := qs.AppendReview(qid, rv); err != nil {
		e.log.Error("宏循环：保存 review 失败", "qid", qid, "err", err)
	}
	e.persistReviewReport(qid, fmt.Sprintf("mage_%d", q.ReworkCount), phaseIdx, rv)
	if score > 0 {
		q.MageScore = score
	}
	e.publish(qid, "", events.EvtReviewSubmitted, map[string]any{
		"verdict": verdict,
		"comment": comment,
		"hints":   hints,
	})
	return verdict, comment, hints, score, rv, false
}

// buildMageReviewSource 构造法师 review 的 ReviewSource（Phase 1.5）。
// macro 路径是唯一能全量填齐 5 字段的路径（见 mage-review-spec v0.2.2 §0.2.1）。
// session_id 从 quest.phases[phaseIdx] 取；若 phases 为空则标记 SourceIncomplete。
func buildMageReviewSource(q *fsstore.QuestMeta, mage *fsstore.AdventurerFile, phaseIdx int) *fsstore.ReviewSource {
	src := &fsstore.ReviewSource{
		SourceRole:     "mage",
		SourcePhaseIdx: phaseIdx,
	}
	if mage != nil {
		src.SourceClass = string(mage.Class)
		src.SourceAdventurerID = mage.ID
	} else if q != nil {
		src.SourceClass = string(model.ClassMage)
		if agentID := reviewAgentIDForPhase(q, phaseIdx); agentID != "" {
			src.SourceAdventurerID = "agent:" + agentID
		}
	}
	if phaseIdx >= 0 && phaseIdx < len(q.Phases) {
		src.SourceSessionID = q.Phases[phaseIdx].SessionID
	} else {
		src.SourceIncomplete = true
	}
	return src
}

func reviewAgentIDForPhase(q *fsstore.QuestMeta, phaseIdx int) string {
	if q == nil {
		return ""
	}
	if q.ReviewAgentID != "" {
		return q.ReviewAgentID
	}
	return phaseTaskForActor(q, phaseIdx).AgentID
}

func (e *Engine) blockQuestOrFail(
	qs *fsstore.QuestStore,
	qid string,
	opts quest.BlockQuestOptions,
	attribution PhaseSignal,
	actor string,
) *fsstore.QuestMeta {
	blockedDomain, err := e.questService.BlockQuest(qid, opts)
	if err != nil {
		e.log.Error("阻塞 quest 失败", "qid", qid, "phase", opts.Phase, "err", err, "reason", opts.Reason)
		e.failQuest(qid, fmt.Sprintf("阻塞任务失败：%v；原始原因：%s", err, opts.Reason))
		if loaded, loadErr := qs.LoadQuest(qid); loadErr == nil {
			return loaded
		}
		return nil
	}
	blocked := fsstore.QuestMetaFromDomain(blockedDomain)
	if blocked == nil {
		if loaded, err := qs.LoadQuest(qid); err == nil {
			blocked = loaded
		}
	}
	if blocked != nil {
		e.recordRecoveryDecision(blocked)
	}
	e.publishFailureAttribution(qid, attribution, actor)
	return blocked
}

func (e *Engine) handleReviewOutcome(
	ctx context.Context,
	qs *fsstore.QuestStore,
	q *fsstore.QuestMeta,
	warrior, mage *fsstore.AdventurerFile,
	rv *fsstore.ReviewRecord,
	verdict model.QuestVerdict,
	comment string,
	hints string,
	score int,
) (*fsstore.QuestMeta, bool) {
	qid := q.ID
	// 空 verdict：法师未显式调用 review_quest（超时兜底）。
	// 平台不猜结论，直接进入用户终审，避免越界做语义判断。
	if verdict == "" {
		_, _ = e.questService.BlockQuest(qid, quest.BlockQuestOptions{
			Reason:     "法师评审超时未提交结论",
			ReasonCode: "mage_timeout_no_review",
			Phase:      "mage_review",
			Details:    map[string]any{"comment": comment},
		})
		e.runAfterCommitSubscribers(qid, "diff_summary")
		return q, false
	}

	switch verdict {
	case model.VerdictPass:
		return e.handleReviewPass(ctx, qs, q, warrior, mage, rv, verdict, comment, hints, score)
	case model.VerdictReject:
		e.failQuest(qid, fmt.Sprintf("法师评审拒绝：%s", comment))
		return q, false
	case model.VerdictRequestChange:
		return e.handleReviewRequestChanges(qs, q, verdict, comment, hints)
	default:
		e.log.Error("未知评审结论", "qid", qid, "verdict", verdict)
		e.failQuest(qid, fmt.Sprintf("未知评审结论: %s", verdict))
		return q, false
	}
}

func (e *Engine) handleReviewPass(
	ctx context.Context,
	qs *fsstore.QuestStore,
	q *fsstore.QuestMeta,
	warrior, mage *fsstore.AdventurerFile,
	rv *fsstore.ReviewRecord,
	verdict model.QuestVerdict,
	comment string,
	hints string,
	score int,
) (*fsstore.QuestMeta, bool) {
	qid := q.ID
	if score > 0 && score < lowPassScoreSignalThreshold() {
		e.publishReviewSignal(qid, map[string]any{
			"signal":    "low_score_pass",
			"severity":  "warning",
			"score":     score,
			"verdict":   verdict,
			"intensity": q.Intensity,
			"message":   "mage returned pass with a low score; platform records the signal without rewriting the verdict",
		})
	}

	// HOTL v0.2: 所有 quest 一视同仁走 policy 决策。
	// checker pass = 最优解，policy 返回 auto_pass → 自主 apply → success。
	decision, facts := e.decideReviewPolicy(q, verdict, score)
	e.publishPolicyDecision(q, decision, facts)

	// 多阶段 quest：非最后阶段先推进
	currentPhase := q.CurrentPhaseIdx()
	if q.PhaseCount > 0 && currentPhase >= 1 && currentPhase < q.PhaseCount-1 {
		qUpdatedDomain, err := e.questService.CompletePhase(qid, quest.CompletePhaseOptions{PhaseIdx: currentPhase})
		if err != nil {
			e.log.Error("推进评审阶段失败", "qid", qid, "phase", currentPhase, "err", err)
			return q, false
		}
		return fsstore.QuestMetaFromDomain(qUpdatedDomain), true
	}

	if decision.Action == policy.ActionAutoPass {
		if e.hotlAutoClose() {
			// HOTL v0.2: 自主闭环（apply + success + 影响通知）
			return e.runAutoCloseLoop(ctx, qs, q, warrior, mage, verdict, comment, &decision)
		}
		// HOTL 关闭时，policy auto-pass 只作为审查信号记录；仍进入显式用户终审链路。
	}

	// v0.1 兼容 fallback：policy 未放行 → auto_apply 或 user_review
	if e.hasAutoApply(q) {
		return e.runAutoApplyProcessor(ctx, qs, q, warrior, mage, verdict, comment)
	}
	qUpdatedDomain, err := e.questService.MoveToUserReview(qid, quest.MoveToUserReviewOptions{
		Verdict:   verdict,
		Comment:   comment,
		FromPhase: "mage_review",
	})
	if err == nil {
		q = fsstore.QuestMetaFromDomain(qUpdatedDomain)
		if q.Type == model.QuestTypeDesign && q.DesignSummary == "" {
			q.DesignSummary = comment
			if err := qs.SaveDesignDoc(q.ID, q.BuildDesignDoc(q.DesignSummary, comment)); err != nil {
				e.log.Error("SaveDesignDoc(user_review) 失败", "qid", qid, "err", err)
			}
			_ = qs.SaveQuest(q)
		}
	}
	e.runAfterCommitSubscribers(qid, "diff_summary")
	return q, false
}

func (e *Engine) handleReviewRequestChanges(
	qs *fsstore.QuestStore,
	q *fsstore.QuestMeta,
	verdict model.QuestVerdict,
	comment string,
	hints string,
) (*fsstore.QuestMeta, bool) {
	qid := q.ID
	e.recordLoopStateRework(q, verdict, comment, hints)
	if q.ReworkCount >= q.MaxRework {
		finalComment := fmt.Sprintf("已达最大返工次数(%d)，提交用户终审。最后意见：%s", q.MaxRework, comment)
		_, _ = e.questService.BlockQuest(qid, quest.BlockQuestOptions{
			Reason:     "达到最大返工次数",
			ReasonCode: "max_rework_exceeded",
			Phase:      "mage_review",
			Details:    map[string]any{"comment": finalComment},
		})
		e.runAfterCommitSubscribers(qid, "diff_summary")
		return q, false
	}

	if q.ReworkCount >= 2 {
		if isStuck, reason, _ := e.detectReworkStalemate(qs, qid, q); isStuck {
			e.log.Info("返工僵局信号已记录", "qid", qid, "reason", reason, "rework_count", q.ReworkCount)
			e.publishReviewSignal(qid, map[string]any{
				"signal":       "rework_stalemate",
				"severity":     "warning",
				"reason":       reason,
				"rework_count": q.ReworkCount,
				"intensity":    q.Intensity,
				"message":      "platform records rework-stalemate as an audit signal without changing intensity or overriding mage verdict",
			})
		}
	}

	currentPhaseIdx := q.CurrentPhaseIdx()
	targetPhaseIdx := 0
	if currentPhaseIdx >= 0 {
		if domainDef := phaseDefForQuest(q, currentPhaseIdx); domainDef != nil {
			targetPhaseIdx = domainDef.ReworkTo
		}
	}
	if currentPhaseIdx > 1 && targetPhaseIdx == 0 {
		e.log.Warn("非首个评审阶段返工目标为 0，按上一执行阶段兜底", "qid", qid, "phase", currentPhaseIdx)
		if prev := previousExecutePhaseDef(q, currentPhaseIdx); prev != nil {
			targetPhaseIdx = prev.Index
		}
	}
	qUpdatedDomain, err := e.questService.RequestReworkToPhase(qid, quest.RequestReworkToPhaseOptions{TargetPhaseIdx: targetPhaseIdx, Hints: hints})
	if err != nil {
		e.log.Error("返工失败", "qid", qid, "err", err)
	} else {
		q = fsstore.QuestMetaFromDomain(qUpdatedDomain)
	}
	e.runAfterCommitSubscribers(qid, "diff_summary")
	return q, true
}

func (e *Engine) recordLoopStateRework(q *fsstore.QuestMeta, verdict model.QuestVerdict, comment, hints string) {
	if e == nil || e.root == nil || q == nil {
		return
	}
	qid := q.ID
	_, err := e.root.PatchLoopStateSpine(qid, func(state *fsstore.LoopStateSpine) {
		state.CurrentGoal = q.Query
		state.CurrentPhase = "maker_rework"
		state.Attempts = append(state.Attempts, fsstore.LoopStateAttempt{
			RunID:     fmt.Sprintf("run_%d", q.ReworkCount),
			Summary:   comment,
			Result:    string(verdict),
			Reason:    hints,
			CreatedAt: fsstore.NowMs(),
		})
		if strings.TrimSpace(comment) != "" {
			state.FailedPaths = appendIfMissing(state.FailedPaths, comment)
		}
		if strings.TrimSpace(hints) != "" {
			state.OpenBlockers = appendIfMissing(state.OpenBlockers, hints)
			state.NextExpectedAction = hints
		} else {
			state.NextExpectedAction = "Address latest ReviewReport required changes."
		}
	})
	if err != nil {
		e.log.Warn("保存 Loop State Spine 失败", "qid", qid, "err", err)
	}
}

func appendIfMissing(items []string, item string) []string {
	item = strings.TrimSpace(item)
	if item == "" {
		return items
	}
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}

func (e *Engine) runWorkspaceCommitProcessor(ctx context.Context, qid string, q *fsstore.QuestMeta) {
	wm := fsstore.NewWorkspaceManager(e.root)
	msg := fmt.Sprintf("gloop: warrior phase %d snapshot (%s)", q.ReworkCount, qid)
	if commitHash, err := wm.CommitWorkspaceChanges(ctx, qid, q, msg); err != nil {
		e.log.Warn("剑士阶段后自动提交工作区改动失败", "qid", qid, "err", err)
	} else if commitHash != "" {
		e.log.Debug("剑士阶段后自动提交工作区改动成功", "qid", qid, "commit", commitHash)
	}
}

func (e *Engine) runPreCompletionProcessors(ctx context.Context, qid string, q *fsstore.QuestMeta, phase *quest.PhaseDef, timing string) {
	if phase == nil {
		return
	}
	for _, name := range phase.PreCompletionProcessors {
		switch name {
		case "workspace_commit":
			if timing == "before_review" {
				e.runWorkspaceCommitProcessor(ctx, qid, q)
			}
		case "auto_apply":
			// auto_apply is handled after a pass verdict because it needs review output.
		default:
			e.log.Warn("未知 pre-completion processor，跳过", "qid", qid, "processor", name, "phase", phase.Name)
		}
	}
}

func (e *Engine) runAfterCommitSubscribers(qid string, names ...string) {
	for _, name := range names {
		switch name {
		case "diff_summary":
			e.bgWG.Add(1)
			go func() {
				defer e.bgWG.Done()
				e.updateDiffSummary(qid)
			}()
		case "review_notify":
			// System notifications are already handled by notifications.EventSubscriber.
		case "":
		default:
			e.log.Warn("未知 after-commit subscriber，跳过", "qid", qid, "subscriber", name)
		}
	}
}

func (e *Engine) runAutoApplyProcessor(
	ctx context.Context,
	qs *fsstore.QuestStore,
	q *fsstore.QuestMeta,
	warrior, mage *fsstore.AdventurerFile,
	verdict model.QuestVerdict,
	comment string,
) (*fsstore.QuestMeta, bool) {
	qid := q.ID
	e.log.Info("auto_apply: 自动应用改动", "qid", qid)

	// readonly 模式无隔离工作区，无需 apply，直接完成
	if q.WorkspaceMode == model.WorkspaceReadOnly {
		e.log.Info("auto_apply: readonly quest 无需 apply，直接闭环", "qid", qid)
		qUpdatedDomain, err := e.questService.MoveToUserReview(qid, quest.MoveToUserReviewOptions{
			Verdict:   verdict,
			Comment:   comment,
			FromPhase: "mage_review",
		})
		if err != nil {
			e.log.Error("auto_apply: readonly 转入用户终审失败", "qid", qid, "err", err)
			return q, false
		}
		q = fsstore.QuestMetaFromDomain(qUpdatedDomain)
		qUpdatedDomain, err = e.questService.CompleteAutoApplySuccess(qid, comment)
		if err != nil {
			e.log.Error("auto_apply: readonly 闭环完成失败", "qid", qid, "err", err)
			return q, false
		}
		q = fsstore.QuestMetaFromDomain(qUpdatedDomain)
		e.publishQuestApplied(qid, q, nil, false, nil)
		e.recordAutomationTrustOutcome(q, "success", "auto_apply_success", false)
		e.awardExp(q, warrior, mage, verdict)
		return q, false
	}

	qUpdatedDomain, err := e.questService.MoveToUserReview(qid, quest.MoveToUserReviewOptions{
		Verdict:   verdict,
		Comment:   comment,
		FromPhase: "mage_review",
	})
	if err != nil {
		e.log.Error("auto_apply: 转入用户终审失败", "qid", qid, "err", err)
		return q, false
	}
	q = fsstore.QuestMetaFromDomain(qUpdatedDomain)

	// apply 前同步计算 diff summary：work 目录在 apply 后会被 cleanup，
	// 不提前存 meta 就再也算不出来（回归：qst_2606253040 diff_stat 为空）。
	e.updateDiffSummary(qid)

	warnings, backup, applyErr := e.applyWorkspace(ctx, qid, q, false)
	if applyErr != nil {
		e.log.Error("auto_apply 失败", "qid", qid, "err", applyErr)
		e.recordApplyFailure(qs, q, applyErr)
		e.publish(qid, "", events.EvtUserReview, map[string]any{
			"verdict":           verdict,
			"comment":           comment,
			"auto_apply_failed": true,
			"error":             applyErr.Error(),
		})
		e.runAfterCommitSubscribers(qid, "diff_summary")
		return q, false
	}
	qUpdatedDomain, err = e.questService.CompleteAutoApplySuccess(qid, comment)
	if err != nil {
		e.log.Error("auto_apply success 失败", "qid", qid, "err", err)
		return q, false
	}
	q = fsstore.QuestMetaFromDomain(qUpdatedDomain)
	e.cleanupAppliedWorkspace(ctx, qid, q)
	e.publishQuestApplied(qid, q, warnings, false, backup)
	e.recordAutomationTrustOutcome(q, "success", "auto_apply_success", false)
	e.awardExp(q, warrior, mage, verdict)
	return q, false
}

// runAutoCloseLoop 是 HOTL v0.2 的自主闭环入口：checker pass → apply → success。
// 与 runAutoApplyProcessor 的区别：由 policy auto_pass 驱动（所有 quest），
// 而非仅 automation 配置了 AutoApply 的 quest。
// apply 失败时进 user_review（技术问题需人处理，不是质量审批）。
func (e *Engine) runAutoCloseLoop(
	ctx context.Context,
	qs *fsstore.QuestStore,
	q *fsstore.QuestMeta,
	warrior, mage *fsstore.AdventurerFile,
	verdict model.QuestVerdict,
	comment string,
	decision *policy.Decision,
) (*fsstore.QuestMeta, bool) {
	qid := q.ID
	e.log.Info("hotl v0.2: checker pass → 自主闭环", "qid", qid, "policy", decision.PolicyName)

	// design quest 无 workspace diff，直接完成
	if q.Type == model.QuestTypeDesign {
		if err := e.completeReviewedQuest(qs, q, verdict, comment, events.EvtQuestSuccess, decision); err != nil {
			e.log.Error("hotl v0.2: design quest 自主完成失败", "qid", qid, "err", err)
		}
		return q, false
	}

	// readonly 模式无隔离工作区，改动直接在基准目录发生，无需 apply → 直接完成
	if q.WorkspaceMode == model.WorkspaceReadOnly {
		e.log.Info("hotl v0.2: readonly quest 无需 apply，直接闭环", "qid", qid)
		if err := e.completeReviewedQuest(qs, q, verdict, comment, events.EvtQuestSuccess, decision); err != nil {
			e.log.Error("hotl v0.2: readonly quest 自主完成失败", "qid", qid, "err", err)
		}
		return q, false
	}

	// 有 workspace 改动 → apply
	qUpdatedDomain, err := e.questService.MoveToUserReview(qid, quest.MoveToUserReviewOptions{
		Verdict:   verdict,
		Comment:   comment,
		FromPhase: "mage_review",
	})
	if err != nil {
		e.log.Error("hotl v0.2: 转入 user_review 失败", "qid", qid, "err", err)
		return q, false
	}
	q = fsstore.QuestMetaFromDomain(qUpdatedDomain)

	// apply 前同步计算 diff summary：work 目录在 apply 后会被 cleanup，
	// 不提前存 meta 就再也算不出来（回归：qst_2606253040 diff_stat 为空）。
	e.updateDiffSummary(qid)

	warnings, backup, applyErr := e.applyWorkspace(ctx, qid, q, false)
	if applyErr != nil {
		e.log.Error("hotl v0.2: 自主 apply 失败，进 user_review", "qid", qid, "err", applyErr)
		e.recordApplyFailure(qs, q, applyErr)
		e.publish(qid, "", events.EvtUserReview, map[string]any{
			"verdict":           verdict,
			"comment":           comment,
			"auto_apply_failed": true,
			"error":             applyErr.Error(),
		})
		e.runAfterCommitSubscribers(qid, "diff_summary")
		return q, false
	}
	qUpdatedDomain, err = e.questService.CompleteAutoApplySuccess(qid, comment)
	if err != nil {
		e.log.Error("hotl v0.2: 自主闭环完成失败", "qid", qid, "err", err)
		return q, false
	}
	q = fsstore.QuestMetaFromDomain(qUpdatedDomain)
	if q.Type == model.QuestTypeDesign && q.DesignSummary == "" {
		q.DesignSummary = comment
		_ = qs.SaveDesignDoc(q.ID, q.BuildDesignDoc(q.DesignSummary, comment))
		_ = qs.SaveQuest(q)
	}
	e.cleanupAppliedWorkspace(ctx, qid, q)
	e.publishQuestApplied(qid, q, warnings, false, backup)
	e.recordAutomationTrustOutcome(q, "success", "policy_auto_pass", false)
	e.awardExp(q, warrior, mage, verdict)
	// HOTL v0.2: goal 模式完成度检查
	if q.GoalMaxIterations > 0 || q.GoalCondition != "" {
		go e.checkGoalCompletion(ctx, qid)
	}
	return q, false
}

func (e *Engine) decideReviewPolicy(q *fsstore.QuestMeta, verdict model.QuestVerdict, score int) (policy.Decision, policy.InputFacts) {
	facts := policy.NewFactBuilder(e.root).BuildReviewFacts(q, string(verdict), score)
	return (policy.ReviewPolicy{}).Decide(facts), facts
}

func (e *Engine) hotlAutoClose() bool {
	if e.cfg == nil || e.cfg.HotlAutoClose == nil {
		return true
	}
	return *e.cfg.HotlAutoClose
}

func (e *Engine) publishPolicyDecision(q *fsstore.QuestMeta, decision policy.Decision, facts policy.InputFacts) {
	if q == nil {
		return
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
		"input_facts":     facts.Map(),
	})
}

func (e *Engine) publishReviewSignal(qid string, payload map[string]any) {
	e.publish(qid, "", events.EventType("review.signal"), payload)
}

func lowPassScoreSignalThreshold() int {
	return 5
}

// reviewVerdictFromOutcome 从阶段输出中提取评审结论（verdict/comment/hints/score/structuredReview/err）。
//
// 这是 review 类阶段的专用解析函数。对于 execute 阶段，直接使用 phaseOutcome 结构体即可。
// 未来如果有更多阶段类型（如 user_review、QA 等），可以新增对应的解析函数。
func (e *Engine) reviewVerdictFromOutcome(outcome phaseOutcome) (model.QuestVerdict, string, string, int, json.RawMessage, error) {
	if outcome.Err != nil {
		return model.VerdictRequestChange, outcome.Content, outcome.Err.Error(), 0, nil, outcome.Err
	}
	pe := outcome.PhaseEnded
	// 正常或兜底空 verdict 都交给宏循环处理
	if isValidReviewVerdict(pe) {
		verdict := model.QuestVerdict(strings.ToLower(pe.PhaseVerdict))
		return verdict, pe.PhaseComment, pe.PhaseHints, pe.PhaseScore, pe.StructuredReview, nil
	}
	// 兜底超时：空 verdict
	return "", pe.PhaseComment, pe.PhaseHints, 0, pe.StructuredReview, nil
}

func waitingInputStateFromSignal(sig PhaseSignal, phaseIdx int) *fsstore.WaitingInputState {
	questionID := ""
	timeoutAction := "continue"
	var timeoutMs int64
	if data, ok := sig.Data.(map[string]any); ok {
		if v, ok := data["question_id"].(string); ok {
			questionID = v
		}
		if v, ok := data["timeout_action"].(string); ok && v != "" {
			timeoutAction = v
		}
		switch v := data["timeout_ms"].(type) {
		case int64:
			timeoutMs = v
		case int:
			timeoutMs = int64(v)
		case float64:
			timeoutMs = int64(v)
		}
	}
	if questionID == "" {
		questionID = "question_" + fsstore.NewIDShort()
	}
	if timeoutMs == 0 {
		timeoutMs = defaultWaitingInputTimeoutMs
	}
	return &fsstore.WaitingInputState{
		QuestionID:     questionID,
		QuestionText:   sig.PhaseComment,
		PhaseIdx:       phaseIdx,
		PhaseType:      "execute",
		Asker:          "warrior",
		TimeoutMs:      timeoutMs,
		TimeoutAction:  timeoutAction,
		ResumePhaseIdx: phaseIdx,
		AskedAt:        time.Now().UTC(),
	}
}

func blockedReasonCode(result PhaseSignal) string {
	if data, ok := result.Data.(map[string]any); ok {
		if reason, ok := data["reason"].(string); ok && reason != "" {
			return reason
		}
	}
	if result.Message != "" {
		return result.Message
	}
	return string(result.PhaseVerdict)
}

func blockedCategory(result PhaseSignal) string {
	if data, ok := result.Data.(map[string]any); ok {
		if cat, ok := data["category"].(string); ok {
			return cat
		}
	}
	return ""
}

func (e *Engine) completeReviewedQuest(qs *fsstore.QuestStore, q *fsstore.QuestMeta, verdict model.QuestVerdict, comment string, eventType events.EventType, decision *policy.Decision) error {
	qid := q.ID

	opts := quest.CompleteQuestOptions{
		Verdict: verdict,
		Comment: comment,
	}
	if decision != nil && decision.Action == policy.ActionAutoPass {
		opts.FinalizedBy = "policy"
		opts.AutoPassedByPolicy = decision.PolicyName
		opts.PolicyDecisionID = decision.DecisionID
	}

	// 通过服务层完成评审并持久化
	qUpdatedDomain, err := e.questService.CompleteQuest(qid, opts)
	if err != nil {
		return fmt.Errorf("完成评审失败: %w", err)
	}
	q = fsstore.QuestMetaFromDomain(qUpdatedDomain)

	// 保存 design 文档（文件系统层面的操作，不属于领域状态机）
	if q.Type == model.QuestTypeDesign && verdict == model.VerdictPass && q.DesignSummary == "" {
		q.DesignSummary = comment
		if err := qs.SaveDesignDoc(q.ID, q.BuildDesignDoc(q.DesignSummary, comment)); err != nil {
			return fmt.Errorf("保存 design 文档失败: %w", err)
		}
		_ = qs.SaveQuest(q)
	}

	warrior, mage := e.loadQuestAdventurers(q)
	if decision != nil && decision.Action == policy.ActionAutoPass {
		e.recordAutomationTrustOutcome(q, "success", "policy_auto_pass", false)
	}
	e.awardExp(q, warrior, mage, verdict)
	if q.Type == model.QuestTypeDesign && q.AutoSpawnExecute {
		go e.maybeAutoSpawnExecuteFromDesign(context.Background(), q.ID)
	}
	e.cleanupWorkspaceIfTerminal(qid)
	return nil
}

// ==================== 阶段公共骨架 ====================
//
// 剑士 / 法师阶段结构完全相同（getExecutor → BuildSystem → CreateSession →
// defer Close → publish phase → runMicroLoop → hint-loop → fallback）。
// 差异项通过 phaseConfig 注入，消除重复。
//
// 注：runWarriorPhase / runMagePhase 已在 Phase 2 管道化中被移除，
// 统一由 runAgentPhase(phaseIdx) 替代，见 pipeline.go。

type phaseConfig struct {
	PhaseName     string // 事件 payload 用：warrior / mage_review
	PhaseIdx      int    // PhaseRunConfig.Phase
	SessionPrefix string // session ID 前缀（不包含 _%d）
	ReadOnly      bool
	AllowedTools  []string
	MaxHints      int    // 配置取数（函数，已做默认值）
	HintNote      string // hint 事件的 note 字段
	HintContent   string // hint 轮次注入的 system 消息
	BuildUserMsg  func() executor.Message
	// IsDone 判断首轮或 hint 轮次后阶段是否已正常结束
	IsDone func(PhaseSignal) bool
	// BuildFallback 在所有 hint 用完仍未结束时，构造兜底 ToolResult
	BuildFallback func(finalContent string, maxHints int) PhaseSignal
}

// phaseOutcome 是 runPhase 的统一返回值：最终文本 + 阶段结果 + 错误。
type phaseOutcome struct {
	Content    string
	PhaseEnded PhaseSignal
	Err        error
}

// runPhase 执行一段阶段：首轮微循环 → N 轮 hint → 兜底。
func (e *Engine) runPhase(ctx context.Context, q *fsstore.QuestMeta, actor *phaseActor, cfg *phaseConfig) phaseOutcome {
	sessionID := fmt.Sprintf("%s_%d", cfg.SessionPrefix, q.ReworkCount)

	if actor == nil {
		return phaseOutcome{Err: fmt.Errorf("phase actor 为空")}
	}

	ex, modelName, err := e.executorForPhaseActor(actor)
	if err != nil {
		return phaseOutcome{Err: fmt.Errorf("获取执行器失败: %w", err)}
	}

	sysPrompt, err := buildSystemForPhaseActor(actor, e.skills, e.templates)
	if err != nil {
		return phaseOutcome{Err: fmt.Errorf("构建 system prompt 失败: %w", err)}
	}

	sessCfg := executor.SessionConfig{
		SystemPrompt: sysPrompt,
		Model:        modelName,
		WorkingDir:   q.WorkspacePath,
		ReadOnly:     cfg.ReadOnly,
		QuestID:      q.ID,
	}
	advID := ""
	advClass := ""
	if actor.Adventurer != nil {
		advID = actor.Adventurer.ID
		advClass = string(actor.Adventurer.Class)
	}
	agentID := ""
	if actor.Agent != nil {
		agentID = actor.Agent.Name
	}
	agentCtx := fsstore.AgentContext{
		DataDir:             e.root.Path(),
		QuestID:             q.ID,
		SessionID:           sessionID,
		Phase:               cfg.PhaseIdx,
		PhaseName:           cfg.PhaseName,
		AdventurerID:        advID,
		AdventurerClass:     advClass,
		AgentID:             agentID,
		PhaseRole:           string(actor.Role),
		SignalWorkspacePath: q.WorkspacePath,
	}
	qs := fsstore.NewQuestStore(e.root)
	var sandboxDir string
	if cfg.ReadOnly && q.WorkspacePath != "" {
		var sandboxErr error
		sandboxDir, sandboxErr = qs.PrepareSandboxCopy(q.ID, sessionID, q.WorkspacePath)
		if sandboxErr != nil {
			return phaseOutcome{Err: fmt.Errorf("准备 mage sandbox 失败: %w", sandboxErr)}
		}
		sessCfg.WorkingDir = sandboxDir
		sessCfg.Env = agentContextEnv(sandboxDir, agentCtx)
		sessCfg.Env = append(sessCfg.Env, "GLOOP_COMMAND_WORKDIR="+sandboxDir)
		e.publish(q.ID, sessionID, events.EvtPhaseChanged, map[string]any{
			"phase":   cfg.PhaseName,
			"sandbox": "copy",
		})
		defer func() {
			if files, add, del, err := fsstore.SandboxDiff(q.WorkspacePath, sandboxDir); err == nil && len(files) > 0 {
				e.appendSessionRow(q.ID, sessionID, &fsstore.QuestSessionRow{
					Timestamp: fsstore.NowMs(),
					Kind:      "mage_sandbox_writes_captured",
					SessionID: sessionID,
					Role:      "system",
					Phase:     cfg.PhaseIdx,
					Content:   fmt.Sprintf("%d sandbox file changes captured (+%d -%d)", len(files), add, del),
					Meta: map[string]any{
						"files":     files,
						"additions": add,
						"deletions": del,
						"mode":      "copy",
					},
				})
				e.publish(q.ID, sessionID, events.EvtToolEnd, map[string]any{
					"tool":      "mage_sandbox",
					"mode":      "copy",
					"files":     len(files),
					"additions": add,
					"deletions": del,
				})
			}
			if err := qs.RemoveSandbox(q.ID, sessionID); err != nil {
				e.log.Warn("清理 mage sandbox 失败", "qid", q.ID, "sid", sessionID, "err", err)
			}
		}()
	} else {
		sessCfg.Env = agentContextEnv(q.WorkspacePath, agentCtx)
	}
	if len(cfg.AllowedTools) > 0 {
		sessCfg.AllowedTools = cfg.AllowedTools
	}
	handle, err := ex.CreateSession(ctx, sessionID, sessCfg)
	if err != nil {
		return phaseOutcome{Err: fmt.Errorf("创建 session 失败: %w", err)}
	}
	e.publish(q.ID, sessionID, events.EvtAgentTransportSelected, agentTransportSelectedPayload(actor, ex, cfg, handle))
	defer ex.CloseSession(ctx, handle.SessionID)
	agentContextDir := q.WorkspacePath
	if sandboxDir != "" {
		agentContextDir = sandboxDir
	}
	if err := fsstore.WriteAgentContext(agentContextDir, agentCtx); err != nil {
		e.log.Warn("写 agent context 失败", "qid", q.ID, "sid", sessionID, "err", err)
	}
	_ = fsstore.RemovePhaseSignal(q.WorkspacePath, sessionID)

	e.publish(q.ID, sessionID, events.EvtPhaseChanged, map[string]any{
		"phase":  cfg.PhaseName,
		"adv_id": advID,
		"agent":  agentID,
	})

	maxHints := cfg.MaxHints
	if maxHints < 0 {
		maxHints = 0
	}

	userMsg := cfg.BuildUserMsg()
	attachQuestInputs(e.root, q, &userMsg)

	// 复用同一个 PhaseRunConfig，让 CommentCursor 跨 hint 轮次保持去重状态
	runCfg := &PhaseRunConfig{
		Qid: q.ID, Sid: sessionID, ExecSID: handle.SessionID, Quest: q, Adv: actor.Adventurer, Agent: actor.Agent, PhaseRole: actor.Role,
		Ex: ex, ModelChoice: modelName, Phase: cfg.PhaseIdx,
		MaxTurns: e.maxTurnsPerPhase(q), StopOnJSON: false,
	}

	runCfg.InitialMsgs = []executor.Message{userMsg}
	finalContent, phaseEnded, lastErr := e.runMicroLoop(ctx, runCfg)
	if lastErr != nil {
		return phaseOutcome{Content: finalContent, PhaseEnded: phaseEnded, Err: lastErr}
	}
	if phaseShouldStop(cfg, phaseEnded) {
		return phaseOutcome{Content: finalContent, PhaseEnded: phaseEnded}
	}

	for hintRound := 1; hintRound <= maxHints; hintRound++ {
		e.log.Info(cfg.PhaseName+" 未提交阶段结论，补提示",
			"qid", q.ID, "hint_round", hintRound)

		hintMsg := executor.Message{Role: "system", Content: cfg.HintContent}

		e.publish(q.ID, sessionID, events.EvtPhaseChanged, map[string]any{
			"phase":      cfg.PhaseName,
			"hint_round": hintRound,
			"note":       cfg.HintNote,
		})

		runCfg.InitialMsgs = []executor.Message{hintMsg}
		hintContent, pe, perr := e.runMicroLoop(ctx, runCfg)
		if hintContent != "" {
			finalContent = hintContent
		}
		phaseEnded = pe
		if perr != nil {
			return phaseOutcome{Content: finalContent, PhaseEnded: phaseEnded, Err: perr}
		}
		if phaseShouldStop(cfg, phaseEnded) {
			return phaseOutcome{Content: finalContent, PhaseEnded: phaseEnded}
		}
	}

	// ---------- structured phase-end observation ----------
	// Text that looks like a phase-ending payload is useful evidence, but it is
	// not a platform action. The phase only ends when the agent uses the syscall
	// path (or a registered platform tool in tests/legacy executors).
	if !phaseShouldStop(cfg, phaseEnded) && strings.TrimSpace(finalContent) != "" {
		if parsed, ok := ParseStructuredPhaseEnd(cfg, finalContent); ok {
			decisionID := fmt.Sprintf("structured_observed_%s_r%d_%s",
				strings.ReplaceAll(cfg.PhaseName, " ", "_"), q.ReworkCount,
				fsstore.NewIDShort())
			inputHash := sha1Hex(finalContent + "|" + strconv.Itoa(cfg.PhaseIdx) + "|" + strconv.Itoa(q.ReworkCount))
			e.appendSessionRow(q.ID, sessionID, &fsstore.QuestSessionRow{
				Seq: 0, Timestamp: fsstore.NowMs(),
				Kind: "structured_phase_end_observed", SessionID: sessionID,
				Role: "system", Phase: cfg.PhaseIdx,
				ToolName: cfg.PhaseName + "_structured_text",
				Content:  parsed.PhaseEnded.PhaseComment,
				Meta: map[string]any{
					"decision_id":          decisionID,
					"input_hash":           inputHash,
					"parser":               parsed.Parser,
					"phase_ended":          false,
					"platform_executed":    false,
					"verdict":              parsed.PhaseEnded.PhaseVerdict,
					"phase_comment_length": len(parsed.PhaseEnded.PhaseComment),
					"hints_length":         len(parsed.PhaseEnded.PhaseHints),
					"score":                parsed.PhaseEnded.PhaseScore,
				},
			})
			e.log.Info(cfg.PhaseName+" 结构化阶段文本已记录",
				"qid", q.ID, "decision_id", decisionID,
				"parser", parsed.Parser, "verdict", parsed.PhaseEnded.PhaseVerdict)
		}
	}
	if !phaseShouldStop(cfg, phaseEnded) {
		e.log.Warn(cfg.PhaseName+" 超时未提交阶段结论，走兜底", "qid", q.ID)
		phaseEnded = cfg.BuildFallback(finalContent, maxHints)
	}
	return phaseOutcome{Content: finalContent, PhaseEnded: phaseEnded}
}

func agentTransportSelectedPayload(actor *phaseActor, ex executor.Executor, cfg *phaseConfig, handle *executor.SessionHandle) map[string]any {
	payload := map[string]any{
		"phase":          "",
		"phase_idx":      -1,
		"agent_id":       "",
		"agent":          "",
		"adventurer_id":  "",
		"executor_id":    "",
		"executor_name":  "",
		"executor_type":  "",
		"transport":      "unknown",
		"resume_capable": false,
	}
	if cfg != nil {
		payload["phase"] = cfg.PhaseName
		payload["phase_idx"] = cfg.PhaseIdx
	}
	if actor != nil {
		if actor.Agent != nil {
			payload["agent_id"] = actor.Agent.Name
			payload["agent"] = actor.Agent.Name
		}
		if actor.Adventurer != nil {
			payload["adventurer_id"] = actor.Adventurer.ID
			payload["adventurer_class"] = string(actor.Adventurer.Class)
			if payload["agent"] == "" {
				payload["agent"] = actor.Adventurer.Agent
			}
		}
		payload["phase_role"] = string(actor.Role)
		payload["binding_source"] = string(actor.Binding.Source)
	}
	if ex != nil {
		payload["executor_id"] = ex.ID()
		payload["executor_name"] = ex.Name()
		payload["executor_type"] = string(ex.Type())
		transport, resumeCapable := transportInfoForExecutor(ex)
		payload["transport"] = transport
		payload["resume_capable"] = resumeCapable
	}
	if handle != nil {
		payload["executor_session_id"] = handle.SessionID
	}
	return payload
}

func transportInfoForExecutor(ex executor.Executor) (transport string, resumeCapable bool) {
	if ex == nil {
		return "unknown", false
	}
	switch ex.Type() {
	case model.AgentTypeACP:
		return "acp", true
	case model.AgentTypeCLI:
		name := strings.ToLower(ex.Name())
		id := strings.ToLower(ex.ID())
		if strings.Contains(name, "traex") || strings.Contains(id, "traex") {
			return "traex", true
		}
		if strings.Contains(name, "codex") || strings.Contains(id, "codex") {
			return "codex_cli", false
		}
		if strings.Contains(name, "relay") || strings.Contains(name, "claude") || strings.Contains(id, "relay") || strings.Contains(id, "claude") {
			return "claude_cli", false
		}
		if strings.Contains(name, "aiden") || strings.Contains(id, "aiden") {
			return "aiden_cli", false
		}
		if strings.Contains(name, "pi") || strings.Contains(id, "pi") {
			return "pi_cli", false
		}
		return "cli", false
	default:
		return string(ex.Type()), false
	}
}

func phaseShouldStop(cfg *phaseConfig, phaseEnded PhaseSignal) bool {
	return cfg.IsDone(phaseEnded) || (phaseEnded.PhaseEnded && phaseEnded.PhaseVerdict == "blocked")
}

// ==================== 旧阶段函数（已移除，留存档参考） ====================
//
// runWarriorPhase → runAgentPhase(ctx, q, adv, 0, phaseInput{})
// runMagePhase    → runAgentPhase(ctx, q, adv, 1, phaseInput{PreviousOutput: warriorOutput})
//                  + reviewVerdictFromOutcome() 提取评审结论

func (e *Engine) maxTurnsPerPhase(q *fsstore.QuestMeta) int {
	if q != nil && q.MaxTurnsPerPhaseOverride > 0 {
		return q.MaxTurnsPerPhaseOverride
	}
	return e.cfg.MaxTurnsPerPhase
}

func (e *Engine) maxDurationPerQuestMs(q *fsstore.QuestMeta) int64 {
	if q != nil && q.MaxDurationPerQuestMsOverride > 0 {
		return q.MaxDurationPerQuestMsOverride
	}
	return e.cfg.MaxDurationPerQuestMs
}

func questDurationElapsedMs(qs *fsstore.QuestStore, q *fsstore.QuestMeta, nowMs int64) int64 {
	if q == nil || q.StartedAtMs <= 0 || nowMs <= q.StartedAtMs {
		return 0
	}
	elapsed := nowMs - q.StartedAtMs
	// 扣除暂停时间：等待用户输入 + 阻塞期间都不是在做实际工作，不计入总时长预算。
	paused := questWaitingInputDurationMs(qs, q, nowMs) + questBlockedDurationMs(q, nowMs)
	if paused >= elapsed {
		return 0
	}
	return elapsed - paused
}

// questBlockedDurationMs 返回 quest 累计 blocked 的毫秒数：读已累加的
// AccumulatedBlockedMs（出 blocked 时由 ApplyTransitionResult 写入），若当前仍
// 处于 blocked，补上本次进行中的 blocked 时长。与 waiting_input 暂停同构——
// blocked 期间 quest 没在跑，这段墙钟时间从 duration 预算扣除，避免一条 quest
// 因人工迟迟未恢复（blocked 静置）被误判 duration_exceeded。
func questBlockedDurationMs(q *fsstore.QuestMeta, nowMs int64) int64 {
	if q == nil {
		return 0
	}
	total := q.AccumulatedBlockedMs
	if q.Status == model.QuestStatusBlocked && q.BlockedAtMs > 0 && nowMs > q.BlockedAtMs {
		total += nowMs - q.BlockedAtMs
	}
	if total < 0 {
		return 0
	}
	return total
}

func questWaitingInputDurationMs(qs *fsstore.QuestStore, q *fsstore.QuestMeta, nowMs int64) int64 {
	if qs == nil || q == nil || q.ID == "" || nowMs <= q.StartedAtMs {
		return 0
	}
	// 优先读显式累加字段（AppendUserAnswer 恢复时写入），避免依赖
	// events.jsonl 落盘时序导致的竞态。
	if q.WaitingInput != nil && q.WaitingInput.AccumulatedPausedMs > 0 {
		total := q.WaitingInput.AccumulatedPausedMs
		// 若当前仍处于 waiting_input（尚未恢复），补上本次进行中的等待时长。
		if q.Status == model.QuestStatusWaitingInput && !q.WaitingInput.AskedAt.IsZero() {
			askedMs := q.WaitingInput.AskedAt.UnixMilli()
			if askedMs > q.StartedAtMs && askedMs < nowMs {
				total += nowMs - askedMs
			}
		}
		if total < 0 {
			return 0
		}
		return total
	}
	// 降级：历史 quest 没有 AccumulatedPausedMs 字段，扫描 events 兜底。
	rows, err := qs.ReadEvents(q.ID, 0)
	if err != nil {
		return 0
	}
	var total int64
	var waitingStart int64
	for _, row := range rows {
		ts := row.Timestamp
		if ts <= 0 || ts < q.StartedAtMs {
			continue
		}
		if ts > nowMs {
			ts = nowMs
		}
		switch events.EventType(row.Type) {
		case events.EvtQuestWaitingInput:
			if waitingStart == 0 {
				waitingStart = ts
			}
		case events.EvtQuestNote:
			if waitingStart > 0 && questEventPayloadHasAnswer(row.Payload) {
				total += ts - waitingStart
				waitingStart = 0
			}
		case events.EvtQuestBlocked, events.EvtUserReview, events.EvtQuestCancelled:
			if waitingStart > 0 {
				total += ts - waitingStart
				waitingStart = 0
			}
		}
	}
	if waitingStart > 0 {
		total += nowMs - waitingStart
	}
	if total < 0 {
		return 0
	}
	return total
}

func questEventPayloadHasAnswer(payload any) bool {
	m, ok := payload.(map[string]any)
	if !ok {
		return false
	}
	if answerID, ok := m["answer_id"].(string); ok && answerID != "" {
		return true
	}
	if _, hasQuestion := m["question_id"].(string); hasQuestion {
		_, hasContent := m["content"].(string)
		_, hasSource := m["source"].(string)
		return hasContent || hasSource
	}
	return false
}

// isValidReviewVerdict 检查 phaseEnded 是否包含有效的评审结论。
func isValidReviewVerdict(pe PhaseSignal) bool {
	if !pe.PhaseEnded || pe.PhaseVerdict == "" {
		return false
	}
	verdict := model.QuestVerdict(strings.ToLower(pe.PhaseVerdict))
	return verdict == model.VerdictPass || verdict == model.VerdictRequestChange || verdict == model.VerdictReject
}

// failQuest 标记 quest 失败并发布事件。
// 通过 QuestService 执行，确保状态校验、持久化和事件发布一致。
func (e *Engine) failQuest(qid string, reason string) {
	_, err := e.questService.FailQuest(qid, reason)
	if err != nil {
		e.log.Error("标记 quest 失败", "qid", qid, "err", err)
		return
	}
	e.cleanupWorkspaceIfTerminal(qid)
}

// ==================== 经验值与等级成长 ====================

// awardExp 发放经验值给剑士和法师（用户终审或受控 automation 审计通过后调用）。
func (e *Engine) awardExp(
	q *fsstore.QuestMeta,
	warrior, mage *fsstore.AdventurerFile,
	verdict model.QuestVerdict,
) {
	stats := fsstore.NewStatsStore(e.root)
	warriorExp, mageExp := calcExpGain(verdict, q.ReworkCount, q.MageScore)

	recordOne := func(adv *fsstore.AdventurerFile, role string, exp int64) {
		if adv == nil {
			return
		}
		_, modelName, err := e.getExecutor(adv)
		if err != nil {
			e.log.Warn("获取 "+role+" executor 失败（跳过经验统计）", "adv", adv.ID, "err", err)
			return
		}
		if modelName == "" {
			modelName = "unknown"
		}
		if err := stats.RecordVerdict(adv, verdict, exp, modelName); err != nil {
			e.log.Warn("记录 "+role+" 经验失败", "adv", adv.ID, "err", err)
			return
		}
		e.log.Info(role+"获得经验", "adv", adv.ID, "exp", exp, "level", adv.Level)
	}

	recordOne(warrior, "剑士", warriorExp)
	recordOne(mage, "法师", mageExp)
}

// calcExpGain 根据评审结论、返工次数和质量评分计算剑士和法师的经验值。
// 公式：基础值 × 返工惩罚系数 × 质量系数
//   - 返工惩罚：每次返工 -20%，最低 0.5 倍
//   - 质量系数（仅 pass 且 score > 0 时生效，作用于剑士）：
//     ≥8 分 → ×1.5
//     5-7 分 → ×1.0
//     <5 分 → ×0.5
func calcExpGain(verdict model.QuestVerdict, reworkCount int, mageScore int) (int64, int64) {
	// 返工惩罚：max(0.5, 1.0 - 0.2 * reworkCount)
	reworkMult := 1.0 - 0.2*float64(reworkCount)
	if reworkMult < 0.5 {
		reworkMult = 0.5
	}

	var wBase, mBase float64
	switch verdict {
	case model.VerdictPass:
		wBase, mBase = 100.0, 60.0
	case model.VerdictRequestChange:
		wBase, mBase = 20.0, 25.0
	case model.VerdictReject:
		wBase, mBase = 5.0, 40.0
	default:
		wBase, mBase = 10.0, 10.0
	}

	// 质量系数（仅 pass 且有评分时应用于剑士）
	qualityMult := 1.0
	if verdict == model.VerdictPass && mageScore > 0 {
		switch {
		case mageScore >= 8:
			qualityMult = 1.5
		case mageScore >= 5:
			qualityMult = 1.0
		default:
			qualityMult = 0.5
		}
	}

	return int64(wBase * reworkMult * qualityMult), int64(mBase * reworkMult)
}

// extractWarriorDeliverables 从剑士阶段结束结果中提取产出物声明，写入 quest meta。
// 返工轮次会先清空上一轮剑士产出物，再写入新的。
func (e *Engine) extractWarriorDeliverables(qs *fsstore.QuestStore, q *fsstore.QuestMeta, phaseEnd PhaseSignal) {
	if phaseEnd.Data == nil {
		return
	}
	dataMap, ok := phaseEnd.Data.(map[string]any)
	if !ok {
		return
	}
	deliverablesRaw, ok := dataMap["deliverables"]
	if !ok {
		return
	}
	deliverables, ok := deliverablesRaw.([]any)
	if !ok || len(deliverables) == 0 {
		return
	}

	// 先清空上一轮剑士声明的产出物（返工场景）
	if err := qs.ClearOutputs(q.ID, "warrior_phase"); err != nil {
		e.log.Warn("清空剑士产出物失败", "qid", q.ID, "err", err)
		return
	}

	added := 0
	for _, item := range deliverables {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		if strings.TrimSpace(name) == "" {
			continue
		}
		kind, _ := m["kind"].(string)
		// path 指向工作区内相对 quest 目录的本地文件（后端 OpenArtifact 按此路径读取并 inline 展示）；
		// url 指向外部链接。两者优先 path，均存在时 path 优先。
		path, _ := m["path"].(string)
		url, _ := m["url"].(string)
		storagePath := path
		if strings.TrimSpace(storagePath) == "" {
			storagePath = url
		}
		description, _ := m["description"].(string)

		decl := fsstore.DeclaredOutput{
			Name:        name,
			Kind:        kind,
			URL:         storagePath,
			Description: description,
			Source:      "warrior_phase",
		}
		if _, err := qs.AddDeclaredOutput(q.ID, decl); err != nil {
			e.log.Warn("添加剑士产出物失败", "qid", q.ID, "name", name, "err", err)
			continue
		}
		added++
	}
	if added > 0 {
		e.log.Info("剑士产出物已登记", "qid", q.ID, "count", added)
	}
}

func (e *Engine) persistMakerReport(qid, sid string, phaseIdx int, phaseEnd PhaseSignal) {
	if e == nil || e.root == nil || qid == "" {
		return
	}
	reportID := "maker_report_" + fsstore.NewIDShort()
	report := &fsstore.MakerReport{
		ReportID:   reportID,
		QuestID:    qid,
		SessionID:  sid,
		Phase:      phaseIdx,
		Verdict:    phaseEnd.PhaseVerdict,
		Summary:    phaseEnd.PhaseComment,
		Artifacts:  artifactIDsFromPhaseSignal(phaseEnd),
		Impact:     extractImpactFromSignal(phaseEnd),
		Transition: true,
	}
	if strings.TrimSpace(report.Verdict) == "" {
		report.Verdict = "done"
	}
	makerEv, eventErr := e.recordQuestEventRequired(qid, sid, events.EventType("report.maker_submitted"), map[string]any{
		"report_id":  reportID,
		"phase":      phaseIdx,
		"verdict":    report.Verdict,
		"session_id": sid,
	})
	if eventErr == nil && makerEv.ID > 0 {
		report.SourceEventID = makerEv.ID
	}
	appendErr := e.root.AppendMakerReport(qid, report)
	if appendErr != nil {
		e.log.Warn("保存 MakerReport 失败", "qid", qid, "sid", sid, "err", appendErr)
		return
	}
	if eventErr == nil && makerEv.ID > 0 {
		e.publishRecordedEvent(makerEv)
	}
}

func (e *Engine) persistReviewReport(qid, sid string, phaseIdx int, rv *fsstore.ReviewRecord) {
	if e == nil || e.root == nil || qid == "" || rv == nil {
		return
	}
	evidenceRefs := e.reviewEvidenceRefs(qid)
	reportID := "review_report_" + fsstore.NewIDShort()
	report := &fsstore.ReviewReport{
		ReportID:         reportID,
		QuestID:          qid,
		SessionID:        sid,
		Phase:            phaseIdx,
		Verdict:          rv.Verdict,
		Score:            rv.Score,
		CheckedAgainst:   evidenceRefIDs(evidenceRefs),
		EvidenceRefs:     evidenceRefs,
		Findings:         findingsFromStructuredReview(rv.StructuredReview),
		Confidence:       "low",
		TransitionNote:   "evidence pack not available in phase 2",
		Comment:          rv.Comment,
		RewriteHints:     rv.RewriteHints,
		StructuredReview: rv.StructuredReview,
	}
	if len(rv.StructuredReview) > 0 {
		report.Confidence = "med"
	}
	if len(evidenceRefs) > 0 {
		report.Confidence = "high"
		report.TransitionNote = ""
	}
	if rv.Verdict == model.VerdictRequestChange && strings.TrimSpace(rv.RewriteHints) != "" {
		report.RequiredChanges = []string{rv.RewriteHints}
	}
	if rv.Verdict == model.VerdictPass && rv.Score > 0 && rv.Score < lowPassScoreSignalThreshold() {
		report.ResidualRisks = []string{"low score pass"}
	}
	reviewEv, eventErr := e.recordQuestEventRequired(qid, sid, events.EventType("report.review_submitted"), map[string]any{
		"report_id":  reportID,
		"phase":      phaseIdx,
		"verdict":    string(rv.Verdict),
		"score":      rv.Score,
		"session_id": sid,
	})
	if eventErr == nil && reviewEv.ID > 0 {
		report.SourceEventID = reviewEv.ID
	}
	appendErr := e.root.AppendReviewReport(qid, report)
	if appendErr != nil {
		e.log.Warn("保存 ReviewReport 失败", "qid", qid, "sid", sid, "err", appendErr)
		return
	}
	if eventErr == nil && reviewEv.ID > 0 {
		e.publishRecordedEvent(reviewEv)
	}
}

func (e *Engine) reviewEvidenceRefs(qid string) []fsstore.EvidenceRef {
	if e == nil || e.root == nil || qid == "" {
		return nil
	}
	refs, err := e.root.LoadEvidenceRefs(qid)
	if err != nil {
		e.log.Warn("读取 EvidenceRef 失败", "qid", qid, "err", err)
		return nil
	}
	return refs
}

func evidenceRefIDs(refs []fsstore.EvidenceRef) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		if strings.TrimSpace(ref.ID) != "" {
			out = append(out, ref.ID)
		}
	}
	return out
}

func artifactIDsFromPhaseSignal(phaseEnd PhaseSignal) []string {
	dataMap, ok := phaseEnd.Data.(map[string]any)
	if !ok || dataMap == nil {
		return nil
	}
	deliverables, ok := dataMap["deliverables"].([]any)
	if !ok || len(deliverables) == 0 {
		return nil
	}
	out := make([]string, 0, len(deliverables))
	for _, item := range deliverables {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if id, _ := m["id"].(string); strings.TrimSpace(id) != "" {
			out = append(out, id)
			continue
		}
		if path, _ := m["path"].(string); strings.TrimSpace(path) != "" {
			out = append(out, path)
			continue
		}
		if name, _ := m["name"].(string); strings.TrimSpace(name) != "" {
			out = append(out, name)
		}
	}
	return out
}

func findingsFromStructuredReview(raw json.RawMessage) fsstore.ReviewFindings {
	findings := fsstore.ReviewFindings{
		Disagreements:    []any{},
		RisksExtra:       []any{},
		Endorsements:     []any{},
		ReferencedChecks: []any{},
	}
	if len(raw) == 0 {
		return findings
	}
	sr, err := quest.ParseStructuredReview(raw)
	if err != nil || sr == nil {
		return findings
	}
	findings.Disagreements = anySlice(sr.Disagreements)
	findings.RisksExtra = anySlice(sr.RisksExtra)
	findings.Endorsements = anySlice(sr.Endorsements)
	findings.ReferencedChecks = anySlice(sr.ReferencedChecks)
	return findings
}

func anySlice[T any](items []T) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out
}

// detectReworkStalemate 检测返工是否出现低进展信号。
//
// 判定条件（满足任意一条即判定僵局）：
//  1. 已经返工 ≥ 2 轮，且最新修复率 < 50%（修复不到一半问题）
//  2. 连续两轮修复率都为 0（完全没进展）
//
// 返回：是否僵局、原因描述、最近的修复进度
//
// 该信号仅用于审计和展示，不直接改变强度、verdict 或状态流。
func (e *Engine) detectReworkStalemate(qs *fsstore.QuestStore, qid string, q *fsstore.QuestMeta) (bool, string, fsstore.RepairProgress) {
	reviews, err := qs.LoadReviews(qid)
	if err != nil {
		return false, "", fsstore.RepairProgress{}
	}

	// 收集所有 request_changes 评审的修复进度
	var progressList []fsstore.RepairProgress
	for _, r := range reviews {
		if r.Verdict == model.VerdictRequestChange && r.TotalIssueCount > 0 {
			progressList = append(progressList, fsstore.RepairProgress{
				Repaired: r.RepairCount,
				Total:    r.TotalIssueCount,
				HasData:  true,
			})
		}
	}

	// 至少需要 2 轮返工数据才做检测
	if len(progressList) < 2 {
		return false, "", fsstore.RepairProgress{}
	}

	latest := progressList[len(progressList)-1]

	// 条件 2：连续两轮修复率都为 0
	if len(progressList) >= 2 {
		prev := progressList[len(progressList)-2]
		if prev.HasData && latest.HasData && prev.Repaired == 0 && latest.Repaired == 0 {
			return true, "连续 2 轮返工修复率为 0，完全没有进展", latest
		}
	}

	// 条件 1：最新修复率 < 50%
	if latest.HasData && latest.Rate() < 0.5 {
		return true, fmt.Sprintf("最新轮修复率 %.0f%%（%d/%d），低于 50%% 阈值",
			latest.Rate()*100, latest.Repaired, latest.Total), latest
	}

	return false, "", latest
}

// parsedPhaseEnd is the result of a successful structured-fallback parse.
type parsedPhaseEnd struct {
	Parser     string
	PhaseEnded PhaseSignal
}

// ParseStructuredPhaseEnd scans the final assistant content for deterministic,
// phase-ending KV fields and returns a ToolResult equivalent to what
// `gloop review …` / `gloop phase done …` would normally produce. Rules are
// strictly deterministic and ordered from highest to lowest credibility:
//
//  1. ```json code blocks with all required schema fields
//  2. <phase_end verdict=… …></phase_end> XML-ish blocks
//  3. Markdown KV table rows (| verdict | pass | etc.) plus inline **key**: value
//  4. Full-text regex KV extraction (last resort)
func ParseStructuredPhaseEnd(cfg *phaseConfig, content string) (parsedPhaseEnd, bool) {
	if cfg == nil {
		return parsedPhaseEnd{}, false
	}
	isReview := false
	switch cfg.PhaseIdx {
	case 1:
		isReview = true
	default:
		name := strings.ToLower(cfg.PhaseName)
		isReview = strings.Contains(name, "review") || strings.Contains(name, "mage")
	}
	normalized := strings.ReplaceAll(content, "\r\n", "\n")

	// 1) fenced ```json
	for _, block := range parseFenced(normalized) {
		if strings.ToLower(block.lang) != "json" {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(block.body), &obj); err == nil {
			if pe, ok := buildStructuredFromMap(isReview, obj); ok {
				return parsedPhaseEnd{Parser: "fenced_json_schema", PhaseEnded: pe}, true
			}
		}
	}

	// 2) <phase_end …></phase_end>
	for _, m := range reStructXML.FindAllStringSubmatch(normalized, -1) {
		attrs := parseAttrs(m[1])
		if pe, ok := buildStructuredFromMap(isReview, attrs); ok {
			return parsedPhaseEnd{Parser: "xml_phase_end_attrs", PhaseEnded: pe}, true
		}
	}

	// 3) Markdown / inline KV tables
	table := parseKVTable(normalized)
	if len(table) > 0 {
		if pe, ok := buildStructuredFromMap(isReview, table); ok {
			return parsedPhaseEnd{Parser: "kv_table_rows", PhaseEnded: pe}, true
		}
	}

	// 4) regex KV (lowest credibility)
	kv := extractRegexKV(normalized)
	if len(kv) > 0 {
		if pe, ok := buildStructuredFromMap(isReview, kv); ok {
			return parsedPhaseEnd{Parser: "regex_kv", PhaseEnded: pe}, true
		}
	}

	return parsedPhaseEnd{}, false
}

// ---------- helpers ----------

// fbBlock represents one fenced code block.
type fbBlock struct {
	lang string
	body string
}

func parseFenced(raw string) []fbBlock {
	var out []fbBlock
	inBlock := false
	lang := ""
	var buf strings.Builder
	for _, line := range strings.SplitAfter(raw, "\n") {
		trim := strings.TrimSpace(strings.TrimSuffix(line, "\n"))
		if strings.HasPrefix(trim, "```") {
			if !inBlock {
				inBlock = true
				lang = strings.TrimSpace(strings.TrimPrefix(trim, "```"))
				buf.Reset()
			} else {
				out = append(out, fbBlock{lang: lang, body: buf.String()})
				inBlock = false
				lang = ""
				buf.Reset()
			}
			continue
		}
		if inBlock {
			buf.WriteString(line)
		}
	}
	return out
}

var reStructXML = regexp.MustCompile(`(?s)<phase_end([^>]*)>(.*?)</phase_end>`)

func parseAttrs(raw string) map[string]any {
	out := map[string]any{}
	for _, m := range regexp.MustCompile(`([a-zA-Z_][\w-]*)\s*=\s*"([^"]*)"`).FindAllStringSubmatch(raw, -1) {
		out[strings.ToLower(m[1])] = m[2]
	}
	return out
}

// parseKVTable extracts | key | value | rows, Markdown table header/value rows,
// **key**: value pairs, and key = value lines into a flat map.
func parseKVTable(raw string) map[string]any {
	out := map[string]any{}
	for _, line := range strings.Split(raw, "\n") {
		t := strings.TrimSpace(line)
		// Markdown pipe table rows
		if strings.HasPrefix(t, "|") && strings.Count(t, "|") >= 2 {
			parts := strings.Split(t, "|")
			for len(parts) > 0 && strings.TrimSpace(parts[0]) == "" {
				parts = parts[1:]
			}
			for len(parts) > 0 && strings.TrimSpace(parts[len(parts)-1]) == "" {
				parts = parts[:len(parts)-1]
			}
			if len(parts) < 2 {
				continue
			}
			// skip header separator rows
			allDashes := true
			for _, p := range parts {
				s := strings.Trim(strings.TrimSpace(p), "-:")
				if s != "" {
					allDashes = false
					break
				}
			}
			if allDashes {
				continue
			}
			key := strings.ToLower(strings.Trim(strings.TrimSpace(parts[0]), "*"))
			val := strings.TrimSpace(parts[1])
			if key == "" || val == "" {
				continue
			}
			out[key] = val
			continue
		}
		// inline **key**: value / key: value / key = value
		for _, m := range regexp.MustCompile(`(?:\*\*)?([a-zA-Z_][\w-]*)(?:\*\*)?\s*[:=]\s*(.+)$`).FindAllStringSubmatch(t, -1) {
			key := strings.ToLower(m[1])
			val := strings.Trim(strings.TrimSpace(m[2]), "\"'`|")
			if key == "" || val == "" {
				continue
			}
			if _, exists := out[key]; !exists {
				out[key] = val
			}
		}
	}
	return out
}

// extractRegexKV performs a last-resort per-line regex scan for well-known
// fields. Existing structured parsers are always preferred; this is only used
// when the assistant repeats the same KV pairs in plain prose.
func extractRegexKV(raw string) map[string]any {
	out := map[string]any{}
	fields := map[string][]string{
		"verdict": {"verdict"},
		"comment": {"comment"},
		"hints":   {"hints", "hint"},
		"score":   {"score"},
		"summary": {"summary"},
		"status":  {"status"},
	}
	for canonical, keys := range fields {
		for _, k := range keys {
			pat := regexp.MustCompile(`(?im)^\s*\**\s*` + regexp.QuoteMeta(k) + `\s*\**\s*[:=]\s*(.+?)\s*$`)
			for _, m := range pat.FindAllStringSubmatch(raw, -1) {
				val := strings.Trim(strings.TrimSpace(m[1]), "\"'`|")
				if val == "" {
					continue
				}
				if _, exists := out[canonical]; !exists {
					out[canonical] = val
				}
			}
		}
	}
	return out
}

// buildStructuredFromMap converts a flat KV map into the phase-ending
// ToolResult. Validation is strict and mirrors CLI validation (verdict must
// be one of the three allowed values, hints required for request_changes,
// score must be in range for pass, summary required for phase_done, etc.).
func buildStructuredFromMap(isReview bool, obj map[string]any) (PhaseSignal, bool) {
	getStr := func(keys ...string) (string, bool) {
		for _, k := range keys {
			if v, ok := obj[k]; ok {
				switch x := v.(type) {
				case string:
					if strings.TrimSpace(x) != "" {
						return strings.TrimSpace(x), true
					}
				case float64:
					return fmt.Sprintf("%v", x), true
				case int:
					return fmt.Sprintf("%d", x), true
				}
			}
		}
		return "", false
	}
	getInt := func(keys ...string) int {
		for _, k := range keys {
			if v, ok := obj[k]; ok {
				switch x := v.(type) {
				case float64:
					return int(x)
				case int:
					return x
				case string:
					n, _ := strconv.Atoi(strings.TrimSpace(x))
					return n
				}
			}
		}
		return 0
	}
	res := PhaseSignal{OK: true, PhaseEnded: true}
	if isReview {
		v, ok := getStr("verdict")
		if !ok {
			return PhaseSignal{}, false
		}
		v = strings.ToLower(strings.TrimSpace(v))
		switch v {
		case "pass":
		case "request_changes", "request-changes", "changes", "rework":
			v = "request_changes"
		case "reject":
		default:
			return PhaseSignal{}, false
		}
		c, ok := getStr("comment")
		if !ok {
			return PhaseSignal{}, false
		}
		res.PhaseVerdict = v
		res.PhaseComment = c
		if h, ok := getStr("hints", "hint"); ok {
			res.PhaseHints = h
		}
		if v == "request_changes" && strings.TrimSpace(res.PhaseHints) == "" {
			return PhaseSignal{}, false
		}
		res.PhaseScore = getInt("score")
		if v == "pass" && (res.PhaseScore < 0 || res.PhaseScore > 10) {
			return PhaseSignal{}, false
		}
		return res, true
	}
	// execution (warrior) phase: status + summary
	summary, ok := getStr("summary")
	if !ok {
		return PhaseSignal{}, false
	}
	status, _ := getStr("status")
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "done":
		status = "done"
	case "blocked":
	default:
		return PhaseSignal{}, false
	}
	res.PhaseVerdict = status
	res.PhaseComment = summary
	return res, true
}

// sha1Hex returns a short, deterministic, non-cryptographic fingerprint used
// exclusively for input-hashing in audit trails.
func sha1Hex(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])[:16]
}
