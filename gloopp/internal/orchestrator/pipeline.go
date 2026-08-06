package orchestrator

import (
	"context"
	"fmt"
	"os"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/domain/quest"
	"code.byted.org/lihuanyu.0w0/gloop/internal/executor"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/prompt"
)

// ==================== Pipeline 阶段管道适配 ====================
//
// Phase 2: 用阶段管道替代硬编码两阶段。
// 每个 Agent 阶段共享同一个执行引擎（runPhase），差异通过 phaseConfig 注入。
//
// 设计原则：数据驱动，非接口多态。
// 阶段索引决定了配置的构建方式，
// 而不是通过不同的 Phase 接口实现。

// phaseInput 是阶段输入的统一结构。
// 不同阶段使用不同的字段，保持结构简单。
type phaseInput struct {
	// 上一阶段的输出（评审阶段用）
	PreviousOutput string
}

func (e *Engine) contextFileModeEnabled() bool {
	return e != nil && e.cfg != nil && e.cfg.ContextFileMode || os.Getenv("GLOOP_CONTEXT_FILE_MODE") != ""
}

func (e *Engine) injectLoopStatePack(qid string, pack *prompt.ContextPack, phase string) {
	if e == nil || e.root == nil || pack == nil || qid == "" {
		return
	}
	state, err := e.root.LoadLoopStateSpine(qid)
	if err != nil || state == nil || len(state.Attempts) == 0 && state.NextExpectedAction == "" {
		return
	}
	pack.Blocks = append(pack.Blocks, prompt.ContextBlock{
		Name:           "loop_state_spine",
		Source:         "gloop",
		Trust:          prompt.TrustPlatform,
		Phase:          phase,
		Role:           "history",
		Staleness:      "live",
		Priority:       9,
		UserControlled: false,
		Content:        renderLoopStateSpine(state),
	})
}

func renderLoopStateSpine(state *fsstore.LoopStateSpine) string {
	if state == nil {
		return ""
	}
	lines := []string{
		"Loop State Spine:",
		"current_goal: " + state.CurrentGoal,
		"current_phase: " + state.CurrentPhase,
	}
	if state.NextExpectedAction != "" {
		lines = append(lines, "next_expected_action: "+state.NextExpectedAction)
	}
	if len(state.Attempts) > 0 {
		last := state.Attempts[len(state.Attempts)-1]
		lines = append(lines, "last_attempt:")
		lines = append(lines, "- result: "+last.Result)
		if last.Summary != "" {
			lines = append(lines, "- summary: "+last.Summary)
		}
		if last.Reason != "" {
			lines = append(lines, "- reason: "+last.Reason)
		}
	}
	if len(state.OpenBlockers) > 0 {
		lines = append(lines, "open_blockers:")
		for _, item := range state.OpenBlockers {
			lines = append(lines, "- "+item)
		}
	}
	return strings.Join(lines, "\n")
}

// buildPhaseConfig 为指定阶段构建 phaseConfig。
//
// 这是管道化的核心：根据 PhaseDef 动态构建执行配置。
// 阶段角色决定配置策略；阶段名称、索引、只读性、结束信号等从 PhaseDef 读取。
func (e *Engine) buildPhaseConfig(q *fsstore.QuestMeta, adv *fsstore.AdventurerFile, phaseDef *quest.PhaseDef, input phaseInput) *phaseConfig {
	if phaseDef == nil {
		phaseDef = quest.DefaultPipeline().At(0)
	}
	switch phaseDef.Role {
	case quest.PhaseRoleReview:
		return e.buildReviewPhaseConfig(q, adv, phaseDef, input.PreviousOutput)
	case quest.PhaseRoleExecute:
		return e.buildExecutePhaseConfig(q, adv, phaseDef)
	default:
		return e.buildExecutePhaseConfig(q, adv, phaseDef)
	}
}

// buildExecutePhaseConfig 构建执行阶段（剑士）的配置。
func (e *Engine) buildExecutePhaseConfig(q *fsstore.QuestMeta, adv *fsstore.AdventurerFile, phaseDef *quest.PhaseDef) *phaseConfig {
	phaseIdx := phaseDef.Index
	phaseName := phaseDef.Name
	if phaseName == "" {
		phaseName = phaseNameForIdx(phaseIdx)
	}
	allowedTools := []string(nil)
	if adv != nil {
		allowedTools = adv.Tools
	}
	if len(phaseDef.AllowedTools) > 0 {
		allowedTools = phaseDef.AllowedTools
	}
	maxHints := phaseDef.DefaultHints
	if maxHints == 0 {
		maxHints = 2
	}
	return &phaseConfig{
		PhaseName:     phaseName,
		PhaseIdx:      phaseIdx,
		SessionPrefix: sessionPrefixForPhase(*phaseDef),
		ReadOnly:      phaseDef.ReadOnly,
		AllowedTools:  allowedTools,
		MaxHints:      maxHints,
		HintNote:      "execution_hint_injected",
		HintContent:   "[protocol] 执行阶段必须通过 Bash 执行 gloop CLI 提交交付，未调用则阶段未结束。\n命令：`gloop phase done --summary \"你的摘要\" --deliverable name=<显示名>,kind=document,path=<相对当前目录的产物路径>,description=<说明>`（--deliverable 可重复，声明产出物否则评审看不到产物）\nverdict: done（提交交付，进入评审）/ fail（放弃，转用户终审）。",
		BuildUserMsg: func() executor.Message {
			// 返工轮次：带上上一轮剑士的交付摘要，避免重复工作
			var prevRoundSummary string
			if q.ReworkCount > 0 {
				prevSID := fmt.Sprintf("warrior_%d", q.ReworkCount-1)
				if artifact, err := fsstore.NewQuestStore(e.root).PhaseArtifactForReview(q.ID, prevSID); err == nil {
					prevRoundSummary = artifact.Assistant
				}
				if reviews, err := fsstore.NewQuestStore(e.root).LoadReviews(q.ID); err == nil {
					q.Reviews = reviews
				}
			}
			pack := prompt.BuildQuestContextPackWith(q, q.ReviewHints, prevRoundSummary, e.templates)
			e.injectLoopStatePack(q.ID, &pack, "warrior")
			prompt.InjectHOTLMode(&pack, e.hotlAutoClose(), "warrior")
			if phaseName == "warrior_execute" {
				if design, ok, err := fsstore.NewQuestStore(e.root).FindPhaseArtifactByKind(q.ID, "design_plan"); err == nil && ok {
					prompt.InjectDesignPlan(&pack, design.Assistant, "warrior")
				}
			}

			if e.contextFileModeEnabled() && q.WorkspacePath != "" {
				var err error
				pack, err = pack.WriteFileContext(q.WorkspacePath, fsstore.GloopDir, prompt.DefaultWarriorCoreBlocks)
				if err != nil {
					e.log.Warn("文件式上下文写入失败，降级为全量注入", "qid", q.ID, "error", err)
				}
			}

			// 按预算裁剪，确保 summary / 渲染文本 / blocks 都来自同一份裁剪后的数据
			trimmed := pack.WithBudget(prompt.DefaultRenderOptions)
			rendered := trimmed.RenderXMLish()
			summary := pack.Summary(prompt.DefaultRenderOptions)

			return executor.Message{
				Role:    "user",
				Content: rendered,
				Meta: map[string]any{
					"context_pack":          summary,
					"context_pack_blocks":   trimmed.Blocks,
					"context_pack_rendered": rendered,
				},
			}
		},
		IsDone: func(pe PhaseSignal) bool {
			return pe.PhaseEnded && pe.PhaseVerdict != ""
		},
		BuildFallback: func(finalContent string, maxHints int) PhaseSignal {
			return PhaseSignal{
				PhaseEnded:   true,
				PhaseVerdict: "blocked",
				PhaseComment: fmt.Sprintf("剑士执行超时：%d 次补 hint 后仍未调用 phase_checkpoint，已转由用户终审。最终输出摘要：%s",
					maxHints, truncate(finalContent, 800)),
				Message: "execution_phase_timeout_no_checkpoint",
			}
		},
	}
}

// buildReviewPhaseConfig 构建评审阶段（法师）的配置。
func (e *Engine) buildReviewPhaseConfig(q *fsstore.QuestMeta, adv *fsstore.AdventurerFile, phaseDef *quest.PhaseDef, warriorOutput string) *phaseConfig {
	allowedTools := []string(nil)
	if adv != nil {
		allowedTools = adv.Tools
	}
	if len(phaseDef.AllowedTools) > 0 {
		allowedTools = phaseDef.AllowedTools
	}
	if len(allowedTools) == 0 {
		allowedTools = []string{"Bash", "Read", "Glob", "Grep"}
	}
	phaseIdx := phaseDef.Index
	phaseName := phaseDef.Name
	if phaseName == "" {
		phaseName = phaseNameForIdx(phaseIdx)
	}
	maxHints := phaseDef.DefaultHints
	if maxHints == 0 {
		maxHints = 2
	}
	return &phaseConfig{
		PhaseName:     phaseName,
		PhaseIdx:      phaseIdx,
		SessionPrefix: sessionPrefixForPhase(*phaseDef),
		ReadOnly:      phaseDef.ReadOnly,
		AllowedTools:  allowedTools,
		HintNote:      "review_hint_injected",
		HintContent:   "[protocol] 评审阶段必须通过 Bash 执行 gloop CLI 提交结论（verdict=pass / request_changes / reject），未调用则评审未完成。\n命令：`gloop review pass --comment \"总评\" --score 9`\n（request_changes 需带 --hints，reject 需带 --comment）",
		BuildUserMsg: func() executor.Message {
			qs := fsstore.NewQuestStore(e.root)
			// 当前轮剑士的 artifact（主要评审对象）
			curWarriorSID := phaseSessionID(q, previousExecutePhaseDef(q, phaseIdx))
			artifact := fsstore.PhaseReviewArtifact{Assistant: warriorOutput}
			if q != nil {
				if stored, err := qs.PhaseArtifactForReview(q.ID, curWarriorSID); err == nil {
					artifact = stored
				}
			}
			// 上一轮剑士的交付摘要（用于 review_history 对比改进脉络）
			var prevRoundWarriorSummary string
			if q != nil && q.ReworkCount > 0 {
				prevSID := fmt.Sprintf("warrior_%d", q.ReworkCount-1)
				if prevArtifact, err := qs.PhaseArtifactForReview(q.ID, prevSID); err == nil {
					prevRoundWarriorSummary = prevArtifact.Assistant
				}
			}
			// P0-3: 自动证据注入——法师评审前，平台先跑一批轻量验证命令，
			// 结果作为 platform_tool_evidence 注入，可信等级最高。
			if q != nil && q.WorkspacePath != "" {
				sid := phaseSessionID(q, phaseDef)
				autoEv, refs := e.collectAutoEvidencePack(context.Background(), q.ID, sid, q)
				if autoEv != "" {
					injectAutoEvidence(&artifact, autoEv)
					for i := range refs {
						if err := e.root.AppendEvidenceRef(q.ID, &refs[i]); err != nil {
							e.log.Warn("保存 EvidenceRef 失败", "qid", q.ID, "sid", sid, "err", err)
						}
					}
					e.log.Info("自动证据已注入", "qid", q.ID, "sid", sid)
				}
			}
			pack := prompt.BuildReviewContextPackFromArtifactWith(q, artifact, prevRoundWarriorSummary, e.templates)
			prompt.InjectHOTLMode(&pack, e.hotlAutoClose(), "mage_review")
			if phaseName == "mage_implementation_review" {
				if design, ok, err := qs.FindPhaseArtifactByKind(q.ID, "design_plan"); err == nil && ok {
					prompt.InjectDesignPlan(&pack, design.Assistant, "mage_review")
					prompt.InjectDesignAlignmentReview(&pack, "mage_review")
				}
			}

			if e.contextFileModeEnabled() && q.WorkspacePath != "" {
				var err error
				pack, err = pack.WriteFileContext(q.WorkspacePath, fsstore.GloopDir, prompt.DefaultMageCoreBlocks)
				if err != nil {
					e.log.Warn("文件式上下文写入失败（法师），降级为全量注入", "qid", q.ID, "error", err)
				}
			}

			// 按预算裁剪，确保 summary / 渲染文本 / blocks 都来自同一份裁剪后的数据
			trimmed := pack.WithBudget(prompt.DefaultRenderOptions)
			rendered := trimmed.RenderXMLish()
			summary := pack.Summary(prompt.DefaultRenderOptions)

			return executor.Message{
				Role:    "user",
				Content: rendered,
				Meta: map[string]any{
					"context_pack":          summary,
					"context_pack_blocks":   trimmed.Blocks,
					"context_pack_rendered": rendered,
				},
			}
		},
		IsDone: isValidReviewVerdict,
		BuildFallback: func(finalContent string, maxHints int) PhaseSignal {
			return PhaseSignal{
				PhaseEnded:   true,
				PhaseVerdict: "", // 空 verdict，宏循环拿到后按用户终审处理
				PhaseComment: fmt.Sprintf("法师评审超时：%d 次补 hint 后仍未调用 review_quest，已转由用户终审。最终输出摘要：%s",
					maxHints, truncate(finalContent, 800)),
				PhaseHints: "请用户人工评审法师的最终输出内容，不要依赖平台推断。",
				Message:    "review_phase_timeout_no_verdict",
			}
		},
	}
}

// runAgentPhase 运行指定索引的 Agent 阶段。
//
// 这是管道化后的统一入口，替代原来的 runWarriorPhase 和 runMagePhase。
// 内部根据阶段索引构建对应配置，然后调用通用 runPhase 引擎。
func (e *Engine) runAgentPhase(ctx context.Context, q *fsstore.QuestMeta, adv *fsstore.AdventurerFile, phaseIdx int, input phaseInput) phaseOutcome {
	actor, err := e.resolvePhaseActorForQuestPhase(q, phaseIdx, adv)
	if err != nil {
		return phaseOutcome{Err: err}
	}
	cfg := e.buildPhaseConfig(q, actor.Adventurer, phaseDefForQuest(q, phaseIdx), input)
	return e.runPhase(ctx, q, actor, cfg)
}

// ===== 辅助函数 =====

func phaseDefForQuest(q *fsstore.QuestMeta, phaseIdx int) *quest.PhaseDef {
	if q != nil && len(q.PipelineDef) > 0 {
		defs := fsstore.QuestMetaToDomain(q).PipelineDef
		if phaseIdx >= 0 && phaseIdx < len(defs) {
			return &defs[phaseIdx]
		}
	}
	if def := quest.DefaultPipeline().At(phaseIdx); def != nil {
		return def
	}
	return &quest.PhaseDef{
		Index:     phaseIdx,
		Name:      phaseNameForIdx(phaseIdx),
		Role:      phaseRoleForIdx(phaseIdx),
		ReadOnly:  false,
		EndSignal: endSignalForRole(phaseRoleForIdx(phaseIdx)),
		ReworkTo:  0,
	}
}

func previousExecutePhaseDef(q *fsstore.QuestMeta, reviewPhaseIdx int) *quest.PhaseDef {
	for idx := reviewPhaseIdx - 1; idx >= 0; idx-- {
		def := phaseDefForQuest(q, idx)
		if def != nil && def.Role == quest.PhaseRoleExecute {
			return def
		}
	}
	return phaseDefForQuest(q, 0)
}

func phaseSessionID(q *fsstore.QuestMeta, def *quest.PhaseDef) string {
	rework := 0
	if q != nil {
		rework = q.ReworkCount
	}
	if def == nil {
		return fmt.Sprintf("phase_%d_%d", 0, rework)
	}
	return fmt.Sprintf("%s_%d", sessionPrefixForPhase(*def), rework)
}

func sessionPrefixForPhase(def quest.PhaseDef) string {
	if def.Name != "" && def.Name != "warrior" && def.Name != "mage" && def.Name != "mage_review" {
		return def.Name
	}
	switch def.Role {
	case quest.PhaseRoleReview:
		if def.Index <= 1 {
			return "mage"
		}
		return fmt.Sprintf("mage_%d", def.Index)
	case quest.PhaseRoleExecute:
		if def.Index == 0 {
			return "warrior"
		}
		return fmt.Sprintf("warrior_%d", def.Index)
	default:
		return fmt.Sprintf("phase_%d", def.Index)
	}
}

func endSignalForRole(_ quest.PhaseRole) string {
	return "phase_checkpoint"
}

// phaseRoleForIdx 根据阶段索引返回角色分类。
// 目前两阶段固定：0=execute, 1=review
// 未来可以从 PhaseDef 获取。
func phaseRoleForIdx(idx int) quest.PhaseRole {
	switch idx {
	case 0:
		return quest.PhaseRoleExecute
	case 1:
		return quest.PhaseRoleReview
	default:
		return quest.PhaseRoleReview
	}
}

// phaseNameForIdx 返回阶段名称（事件/日志用）。
func phaseNameForIdx(idx int) string {
	switch idx {
	case 0:
		return "warrior"
	case 1:
		return "mage_review"
	default:
		return fmt.Sprintf("phase_%d", idx)
	}
}

// phaseDisplayNameForIdx 返回阶段展示名。
func phaseDisplayNameForIdx(idx int) string {
	switch idx {
	case 0:
		return "剑士执行"
	case 1:
		return "法师评审"
	default:
		return fmt.Sprintf("阶段 %d", idx)
	}
}
