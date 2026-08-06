package orchestrator

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/domain/quest"
	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
	"code.byted.org/lihuanyu.0w0/gloop/internal/executor"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
)

// ==================== 微循环：单阶段回合制 ====================
//
// 每 turn: 发消息 → 流式推理 → 拦截平台工具 → 结果回灌
// 微循环：单阶段内反复调用 executor，直到阶段结束工具触发 / guardrail 触发（时长/无进展/连续错误）/ 安全网 turn 上限 / 触发 stop 信号。

// PhaseRunConfig 把 runMicroLoop 所需的一大捆参数打包，避免 11 元参数列表
// （Go 社区约定：参数 >5 个时用结构体；参数语义相关，struct 更可读、可演进）。
type PhaseRunConfig struct {
	Qid         string
	Sid         string // 文件系统/事件用的 session ID（quest 内唯一，如 warrior_0）
	ExecSID     string // executor 内存 map 用的 session key（全局唯一，如 qst_xxx/warrior_0）
	Quest       *fsstore.QuestMeta
	Adv         *fsstore.AdventurerFile
	Agent       *fsstore.AgentConfig
	PhaseRole   quest.PhaseRole
	Ex          executor.Executor
	ModelChoice string
	Phase       int // 0 = warrior, 1 = mage
	MaxTurns    int
	StopOnJSON  bool
	InitialMsgs []executor.Message
}

// collectNewUserComments 收集 quest 中自 lastCount 之后新增的用户评论。
// 返回评论内容列表和最新的评论计数。
func (e *Engine) collectNewUserComments(qid string, lastCount int) ([]string, int) {
	qs := fsstore.NewQuestStore(e.root)
	events, err := qs.ReadEvents(qid, 0)
	if err != nil {
		return nil, lastCount
	}

	var comments []string
	count := 0
	for _, ev := range events {
		if ev.Type != "quest.note" {
			continue
		}
		payload, ok := ev.Payload.(map[string]interface{})
		if !ok {
			continue
		}
		source, _ := payload["source"].(string)
		if source != "user" {
			continue
		}
		count++
		if count > lastCount {
			content, _ := payload["content"].(string)
			if content != "" {
				comments = append(comments, content)
			}
		}
	}
	return comments, count
}

// formatUserCommentMessage 将用户评论格式化成一条 user message。
// 多条评论合并为一条，用 XMLish 标签包裹，保持与 context pack 一致的风格。
func formatUserCommentMessage(comments []string) executor.Message {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<user_comments count=\"%d\">\n", len(comments)))
	for i, c := range comments {
		sb.WriteString(fmt.Sprintf("评论 %d:\n", i+1))
		sb.WriteString(c)
		sb.WriteString("\n\n")
	}
	sb.WriteString("</user_comments>")
	sb.WriteString("\n\n请结合以上用户评论继续任务。")

	return executor.Message{
		Role:    "user",
		Content: sb.String(),
	}
}

func (e *Engine) collectWaitingInputAnswers(qid string) []fsstore.AnswerRecord {
	qs := fsstore.NewQuestStore(e.root)
	q, err := qs.LoadQuest(qid)
	if err != nil || q.WaitingInput == nil || q.WaitingInput.LastAnswerID == "" {
		return nil
	}
	answers, err := qs.LoadAnswers(qid)
	if err != nil {
		return nil
	}
	var out []fsstore.AnswerRecord
	for _, answer := range answers {
		if answer.AnswerID == q.WaitingInput.LastAnswerID {
			out = append(out, answer)
			break
		}
	}
	return out
}

func (e *Engine) clearWaitingInputAfterAnswerInject(qid string) {
	qs := fsstore.NewQuestStore(e.root)
	q, err := qs.LoadQuest(qid)
	if err != nil || q.WaitingInput == nil {
		return
	}
	q.WaitingInput = nil
	q.UpdatedAtMs = fsstore.NowMs()
	if err := qs.SaveQuest(q); err != nil {
		e.log.Warn("清理 waiting_input 状态失败", "qid", qid, "err", err)
	}
}

func formatWaitingInputAnswerMessage(answers []fsstore.AnswerRecord) executor.Message {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<user_answers count=\"%d\">\n", len(answers)))
	for _, answer := range answers {
		sb.WriteString(fmt.Sprintf("<answer id=\"%s\" question_id=\"%s\" source=\"%s\">\n", answer.AnswerID, answer.QuestionID, answer.Source))
		sb.WriteString(answer.Content)
		sb.WriteString("\n</answer>\n")
	}
	sb.WriteString("</user_answers>")
	sb.WriteString("\n\n请结合以上用户回答继续任务。")
	return executor.Message{
		Role:    "user",
		Content: sb.String(),
		Meta:    map[string]any{"_waiting_answer_inject": true, "_answer_count": len(answers)},
	}
}

// watchPhaseSignal 启动一个后台 goroutine，轮询 phase 信号文件。
// 检测到 phase-end 信号后，等待宽限期，然后调用 Interrupt 强制中断 agent 进程，
// 使 SendMessage 提前返回。阶段结束的正式处理（consumePhaseSignal）仍由
// 主循环在 turn 边界完成，保证日志、事件、状态变更都在主 goroutine 内进行。
//
// 返回 stopFn：调用它停止监控并阻塞到 goroutine 退出。
func (e *Engine) watchPhaseSignal(
	ctx context.Context,
	qid string,
	sid, execSID, workDir string,
	ex executor.Executor,
	onSignalSeen func(map[string]any),
	pollInterval time.Duration,
	gracePeriod time.Duration,
) func() {
	stop := make(chan struct{})
	done := make(chan struct{})

	go func() {
		defer close(done)
		if pollInterval <= 0 {
			return
		}
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-ticker.C:
				sig, err := fsstore.ReadPhaseSignal(workDir, sid)
				if err != nil || sig == nil {
					continue
				}
				if !sig.PhaseEnded {
					// 非 phase-end 信号（如 quest_ask），忽略 —— 主循环会处理
					continue
				}
				if onSignalSeen != nil {
					onSignalSeen(map[string]any{
						"message":       sig.Message,
						"phase_verdict": sig.PhaseVerdict,
						"qid":           qid,
					})
				}
				// 检测到 phase-end 信号 → 宽限期到了就强制中断
				if gracePeriod > 0 {
					select {
					case <-time.After(gracePeriod):
						_ = ex.Interrupt(context.Background(), execSID, "phase_signal_grace_expired")
					case <-stop:
					case <-ctx.Done():
					}
				}
				return
			}
		}
	}()

	return func() {
		close(stop)
		<-done
	}
}

func (e *Engine) runMicroLoop(
	ctx context.Context,
	cfg *PhaseRunConfig,
) (finalContent string, phaseEnded PhaseSignal, err error) {
	qid := cfg.Qid
	sid := cfg.Sid
	execSID := cfg.ExecSID // executor 内存 map key（全局唯一，由 executor 从 QuestID+sessionID 构造）
	q := cfg.Quest
	adv := cfg.Adv
	agent := cfg.Agent
	ex := cfg.Ex
	modelChoice := cfg.ModelChoice
	phase := cfg.Phase
	stopOnJSON := cfg.StopOnJSON
	pendingMsgs := append([]executor.Message{}, cfg.InitialMsgs...)

	// 每次 micro loop 启动时，将入队游标对齐到已发送游标
	// 前一阶段/hint 轮入队但未发送的评论会被重新检测并入队，避免 phase 提前结束导致丢失
	e.syncCommentEnqueuedToCursor(qid)

	// 预算追踪
	phaseStart := fsstore.NowMs()
	var totalTokens int64 = 0
	consecutiveErrors := 0
	retryableErrorStreak := 0
	authRecoveryAttempted := false
	noProgressStreak := 0
	prevContent := ""
	maxNoProgress := e.cfg.MaxNoProgressTurns

	maxTurns := e.cfg.MaxTurnsPerPhase
	if maxTurns <= 0 {
		maxTurns = 1000
	}
	for turn := 1; turn <= maxTurns; turn++ {
		select {
		case <-ctx.Done():
			return finalContent, phaseEnded, ctx.Err()
		default:
		}
		if sig, ok := e.consumePhaseSignal(qid, sid, q.WorkspacePath, phase); ok {
			return finalContent, sig, nil
		}

		if answers := e.collectWaitingInputAnswers(qid); len(answers) > 0 {
			pendingMsgs = append(pendingMsgs, formatWaitingInputAnswerMessage(answers))
		}

		// 检查新的用户评论，注入到 pending 消息队列末尾
		// 评论排在已有 tool results 之后，保证对话顺序正确
		// 双游标：commentEnqueued 防止重复入队，commentCursor（已发送）在发送时才推进并持久化
		enqueued := e.getCommentEnqueued(qid)
		newComments, newCount := e.collectNewUserComments(qid, enqueued)
		if len(newComments) > 0 {
			e.setCommentEnqueued(qid, newCount)
			commentMsg := formatUserCommentMessage(newComments)
			// 在 Meta 里标记这是一条评论注入消息，发送时推进持久化 cursor
			commentMsg.Meta = map[string]any{
				"_comment_inject":     true,
				"_comment_count":      len(newComments),
				"_comment_new_cursor": newCount,
			}
			pendingMsgs = append(pendingMsgs, commentMsg)
		}

		if len(pendingMsgs) == 0 {
			// 会话继续，agent 自行决定下一步。系统不提供引导。
			pendingMsgs = append(pendingMsgs, executor.Message{
				Role:    "system",
				Content: "[session continued]",
			})
		}

		// 取一条 pending 消息发给 executor
		msg := pendingMsgs[0]
		pendingMsgs = pendingMsgs[1:]
		// _retry 标记：transient/重试塞回的同一条消息，不重复记录注入类 session row。
		// 语义：重试是"把同一条消息再发一次"，不是"注入新上下文/评论"。
		isRetry := false
		if metaMap, ok := msg.Meta.(map[string]any); ok {
			if r, ok := metaMap["_retry"].(bool); ok {
				isRetry = r
			}
		}

		// 记录 turn 开始
		turnMeta := map[string]any{"status": "thinking"}
		if isRetry {
			turnMeta["retry"] = true
		}
		e.appendSessionRow(qid, sid, &fsstore.QuestSessionRow{
			Seq: turn, Timestamp: fsstore.NowMs(), Kind: "turn_summary", SessionID: sid,
			Role: "system", Content: fmt.Sprintf("turn %d start", turn),
			Phase: phase, Meta: turnMeta,
		})
		turnEvent := map[string]any{
			"turn":   turn,
			"phase":  phase,
			"step":   "think",
			"status": "thinking",
		}
		if isRetry {
			turnEvent["retry"] = true
		}
		e.publish(qid, sid, events.EvtMicroTurn, turnEvent)

		if metaMap, ok := msg.Meta.(map[string]any); ok {
			if packSummary, ok := metaMap["context_pack"]; ok {
				// 重试不重复记录 context_pack（内容与首次完全相同）
				if !isRetry {
					// 同时保存摘要、完整 blocks 结构化数据和渲染后的原始文本，
					// 前端可以直接用 blocks 做结构化展示，也能切换到原始文本视图。
					meta := map[string]any{"context_pack": packSummary}
					if blocks, ok := metaMap["context_pack_blocks"]; ok {
						meta["context_pack_blocks"] = blocks
					}
					rendered := ""
					if r, ok := metaMap["context_pack_rendered"].(string); ok {
						rendered = r
					}
					e.appendSessionRow(qid, sid, &fsstore.QuestSessionRow{
						Seq: turn, Timestamp: fsstore.NowMs(), Kind: "context_pack", SessionID: sid,
						Role: msg.Role, Phase: phase, Content: rendered, Meta: meta,
					})
					e.publish(qid, sid, events.EvtProgressUpdated, map[string]any{
						"step":         "context_pack",
						"turn":         turn,
						"phase":        phase,
						"context_pack": packSummary,
					})
				}
			}
			// 评论注入消息：实际发送时才推进 cursor，避免 phase 提前结束导致评论丢失
			if isInject, _ := metaMap["_comment_inject"].(bool); isInject && !isRetry {
				if newCursor, _ := metaMap["_comment_new_cursor"].(int); newCursor > 0 {
					e.setCommentCursor(qid, newCursor)
					e.persistCommentCursor(qid, newCursor)
				}
				count, _ := metaMap["_comment_count"].(int)
				e.appendSessionRow(qid, sid, &fsstore.QuestSessionRow{
					Seq: turn, Timestamp: fsstore.NowMs(), Kind: "user_comment", SessionID: sid,
					Role: "user", Phase: phase,
					Content: fmt.Sprintf("%d 条新用户评论已注入", count),
					Meta:    map[string]any{"count": count},
				})
			}
			if isInject, _ := metaMap["_waiting_answer_inject"].(bool); isInject && !isRetry {
				e.clearWaitingInputAfterAnswerInject(qid)
				count, _ := metaMap["_answer_count"].(int)
				e.appendSessionRow(qid, sid, &fsstore.QuestSessionRow{
					Seq: turn, Timestamp: fsstore.NowMs(), Kind: "user_answer", SessionID: sid,
					Role: "user", Phase: phase,
					Content: fmt.Sprintf("%d 条用户回答已注入", count),
					Meta:    map[string]any{"count": count},
				})
			}
		}

		// 流式通道：边收边 SSE，同时累积 delta 用于信号中断时恢复 assistant 内容
		stream := make(chan executor.StreamChunk, 256)
		streamDone := make(chan struct{})
		var streamDelta strings.Builder
		outputObserved := false
		var lastRuntimeActivityMs atomic.Int64
		lastRuntimeActivityMs.Store(fsstore.NowMs())
		markRuntimeActivity := func() {
			lastRuntimeActivityMs.Store(fsstore.NowMs())
		}
		go func() {
			defer close(streamDone)
			for c := range stream {
				if c.Event != "" {
					if c.Event == executor.RuntimeEventOutputFirst {
						outputObserved = true
					}
					switch c.Event {
					case executor.RuntimeEventOutputFirst, executor.RuntimeEventOutputHeartbeat, executor.RuntimeEventProcessStarted:
						markRuntimeActivity()
					}
					e.publishRuntimeHealthEvent(qid, sid, c.Event, c.Data, map[string]any{
						"turn":  turn,
						"phase": phase,
					})
					continue
				}
				if c.Error != "" {
					continue
				}
				if c.Delta != "" {
					if !outputObserved {
						outputObserved = true
						markRuntimeActivity()
						e.publishRuntimeHealthEvent(qid, sid, executor.RuntimeEventOutputFirst, map[string]any{
							"bytes": len(c.Delta),
						}, map[string]any{
							"turn":  turn,
							"phase": phase,
						})
					}
					streamDelta.WriteString(c.Delta)
					e.publish(qid, sid, events.EvtTokenDelta, map[string]any{
						"delta":  c.Delta,
						"finish": c.Finish,
					})
				}
			}
		}()

		turnStart := fsstore.NowMs()
		turnCtx, cancelTurn := e.turnContext(ctx, phaseStart)

		sessionMeta := map[string]any{
			"turn":            turn,
			"phase":           phase,
			"model":           modelChoice,
			"capability_tier": string(ex.CapabilityTier()),
			"phase_role":      string(cfg.PhaseRole),
		}
		if adv != nil {
			sessionMeta["adventurer_id"] = adv.ID
			sessionMeta["adventurer_name"] = adv.Name
		}
		if agent != nil {
			sessionMeta["agent_id"] = agent.Name
		}
		e.publishRuntimeHealthEvent(qid, sid, executor.RuntimeEventSessionCreated, nil, sessionMeta)

		stopIdleWatch := e.watchRuntimeIdle(turnCtx, qid, sid, turn, phase, &lastRuntimeActivityMs)
		// 启动实时信号监听：检测到 phase-end 信号后给宽限期再强制中断
		stopWatch := e.watchPhaseSignal(
			turnCtx, qid, sid, execSID, q.WorkspacePath, ex,
			func(data map[string]any) {
				markRuntimeActivity()
				e.publishRuntimeHealthEvent(qid, sid, executor.RuntimeEventProtocolSignalSeen, data, map[string]any{
					"turn":  turn,
					"phase": phase,
				})
			},
			time.Duration(e.cfg.SignalPollIntervalMs)*time.Millisecond,
			time.Duration(e.cfg.SignalGracePeriodMs)*time.Millisecond,
		)

		resp, sErr := ex.SendMessage(turnCtx, execSID, msg, modelChoice, stream)
		stopWatch()
		stopIdleWatch()
		cancelTurn()

		// SendMessage 返回后立刻检查信号：如果 agent 被信号中断了，
		// 直接结束阶段，不进入错误分类或重试逻辑。
		if sig, ok := e.consumePhaseSignal(qid, sid, q.WorkspacePath, phase); ok {
			close(stream)
			<-streamDone
			// 信号中断时补记 assistant message：agent 的 streaming 内容不应丢失。
			// resp 可能因 context canceled 为零值，用累积的 stream delta 兜底。
			acontent := ""
			if sErr == nil {
				acontent = resp.Message.Content
			}
			if strings.TrimSpace(acontent) == "" {
				acontent = streamDelta.String()
			}
			if strings.TrimSpace(acontent) != "" {
				finalContent = acontent
				e.appendSessionRow(qid, sid, &fsstore.QuestSessionRow{
					Seq: turn, Timestamp: fsstore.NowMs(), Kind: "message", SessionID: sid,
					Role: "assistant", Content: acontent, Phase: phase,
					Meta: map[string]any{
						"finish_reason": "phase_signal",
						"token_input":   resp.TokenInput,
						"token_output":  resp.TokenOutput,
						"duration_ms":   resp.DurationMs,
						"recovered":     sErr != nil,
					},
				})
				e.publish(qid, sid, events.EvtAssistantMsg, map[string]any{
					"turn":         turn,
					"content":      truncate(acontent, 1600),
					"has_tools":    len(resp.Message.ToolCalls) > 0,
					"finish":       "phase_signal",
					"token_input":  resp.TokenInput,
					"token_output": resp.TokenOutput,
				})
			}
			return finalContent, sig, nil
		}

		if sErr != nil && turnCtx.Err() != nil {
			if iErr := ex.Interrupt(context.Background(), execSID, "turn_context_done"); iErr != nil {
				e.log.Warn("中断 agent 失败", "qid", qid, "sid", sid, "err", iErr)
			}
		}
		close(stream)
		<-streamDone

		if sErr != nil {
			consecutiveErrors++
			category := classifyAgentError(sErr)
			e.publishRuntimeHealthEvent(qid, sid, string(events.EvtAgentErrorClassified), map[string]any{
				"category":           category,
				"consecutive_errors": consecutiveErrors,
				"error":              sErr.Error(),
			}, map[string]any{"turn": turn, "phase": phase})
			e.appendSessionRow(qid, sid, &fsstore.QuestSessionRow{
				Seq: turn, Timestamp: fsstore.NowMs(), Kind: "error", SessionID: sid,
				Role: "system", Content: sErr.Error(), Phase: phase, Error: sErr.Error(),
				Meta: map[string]any{"category": category},
			})
			if category == "auth" {
				if !authRecoveryAttempted {
					authRecoveryAttempted = true
					if hint, ok := tryNonInteractiveAuthRecovery(ctx, ex); ok {
						e.appendSessionRow(qid, sid, &fsstore.QuestSessionRow{
							Seq: turn, Timestamp: fsstore.NowMs(), Kind: "agent_auth", SessionID: sid,
							Role: "system", Content: hint, Phase: phase,
							Meta: map[string]any{"category": category, "retry": true, "mode": "noninteractive"},
						})
						pendingMsgs = append([]executor.Message{msg}, pendingMsgs...)
						continue
					}
				}
				return finalContent, agentErrorBlockedResult(sErr, category, consecutiveErrors, ex), nil
			}
			if category == "configuration" {
				return finalContent, agentErrorBlockedResult(sErr, category, consecutiveErrors, ex), nil
			}
			if category == "rate_limit" || category == "transient" || category == "session_lost" {
				retryableErrorStreak++
				if retryableErrorStreak < maxRetryableAgentErrors() {
					time.Sleep(agentRetryDelay(retryableErrorStreak))
					// 标记为重试：塞回的消息不重复记录注入类 session row
					if msg.Meta == nil {
						msg.Meta = map[string]any{}
					}
					if mm, ok := msg.Meta.(map[string]any); ok {
						mm["_retry"] = true
					}
					pendingMsgs = append([]executor.Message{msg}, pendingMsgs...)
					continue
				}
				return finalContent, agentErrorBlockedResult(sErr, category, consecutiveErrors, ex), nil
			}
			if maxErrs := e.cfg.MaxConsecutiveAgentErrors; maxErrs > 0 && consecutiveErrors >= maxErrs {
				return finalContent, PhaseSignal{
					OK:           false,
					Message:      fmt.Sprintf("agent 连续错误达到阈值：%d", consecutiveErrors),
					PhaseEnded:   true,
					PhaseVerdict: "blocked",
					PhaseComment: fmt.Sprintf("agent 连续 %d 次执行失败，最后错误：%s", consecutiveErrors, sErr.Error()),
					Data: map[string]any{
						"reason":             "agent_consecutive_errors",
						"consecutive_errors": consecutiveErrors,
						"last_error":         sErr.Error(),
					},
				}, nil
			}
			continue
		}
		consecutiveErrors = 0
		retryableErrorStreak = 0

		// 记录 adventurer turn 指标。agent-direct 阶段可能没有 adventurer overlay。
		if adv != nil {
			stats := fsstore.NewStatsStore(e.root)
			if sErr := stats.RecordTurn(adv.ID, modelChoice,
				resp.TokenInput, resp.TokenOutput, 0.0, fsstore.NowMs()-turnStart); sErr != nil {
				e.log.Warn("RecordTurn 失败", "adv", adv.ID, "err", sErr)
			}
		}

		// 累计 token
		totalTokens += resp.TokenInput + resp.TokenOutput

		e.publishExternalStateWriteObservations(qid, sid, q.WorkspacePath, ex, resp)

		// 记录 assistant 消息（过滤 agent runtime 协议消息，不写入用户可见 trace）
		acontent := resp.Message.Content
		if !isSystemProtocolMessage(acontent) {
			finalContent = acontent
			e.appendSessionRow(qid, sid, &fsstore.QuestSessionRow{
				Seq: turn, Timestamp: fsstore.NowMs(), Kind: "message", SessionID: sid,
				Role: "assistant", Content: acontent, Phase: phase,
				Meta: map[string]any{
					"finish_reason": resp.FinishReason,
					"token_input":   resp.TokenInput,
					"token_output":  resp.TokenOutput,
					"duration_ms":   resp.DurationMs,
				},
			})
			e.publish(qid, sid, events.EvtAssistantMsg, map[string]any{
				"turn":         turn,
				"content":      truncate(acontent, 1600),
				"has_tools":    len(resp.Message.ToolCalls) > 0,
				"finish":       resp.FinishReason,
				"token_input":  resp.TokenInput,
				"token_output": resp.TokenOutput,
			})
		}

		// stopOnJSON 检查
		if stopOnJSON {
			if _, ok := extractJSON(acontent); ok {
				return finalContent, phaseEnded, nil
			}
		}

		// mock executor 通过 ChatResponse.PhaseSignal 模拟 CLI 阶段信号。
		// 真实 executor（ACP/CLI）不设置此字段——它们通过 gloop CLI 写信号文件。
		if resp.PhaseSignal != nil && resp.PhaseSignal.PhaseEnded {
			e.publish(qid, sid, events.EvtToolEnd, map[string]any{
				"tool":   "gloop_cli_signal",
				"id":     sid,
				"result": truncate(resp.PhaseSignal.PhaseComment, 400),
			})
			return finalContent, PhaseSignal{
				OK:           resp.PhaseSignal.OK,
				Message:      resp.PhaseSignal.Message,
				PhaseEnded:   true,
				PhaseVerdict: resp.PhaseSignal.PhaseVerdict,
				PhaseComment: resp.PhaseSignal.PhaseComment,
				PhaseHints:   resp.PhaseSignal.PhaseHints,
				PhaseScore:   resp.PhaseSignal.PhaseScore,
				Data:         resp.PhaseSignal.Data,
			}, nil
		}

		// ===== ACT：agent 的 ReAct 循环完全自治 =====
		// gloop 不感知、不拦截、不代执行 agent 的 tool calls。
		// agent → gloop 的唯一通道是 gloop CLI（phase done / review / post / quest ask 等），
		// 这些 CLI 命令写信号文件或调 server API，由 consumePhaseSignal 在下一轮 turn 开头消费。
		if len(resp.Message.ToolCalls) > 0 {
			e.publish(qid, sid, events.EvtProgressUpdated, map[string]any{
				"step":       "tool_calls",
				"turn":       turn,
				"tool_count": len(resp.Message.ToolCalls),
			})
		}
		if sig, ok := e.consumePhaseSignal(qid, sid, q.WorkspacePath, phase); ok {
			return finalContent, sig, nil
		}
		if !phaseSignalExpected(resp) {
			e.publishRuntimeHealthEvent(qid, sid, executor.RuntimeEventProtocolWarning, map[string]any{
				"reason":        "missing_phase_signal",
				"finish_reason": resp.FinishReason,
			}, map[string]any{"turn": turn, "phase": phase})
		}

		// ===== 无进展检测 =====
		// 有工具调用，或本轮文本相对上一轮有新内容，都算有进展；否则累计 streak。
		// 系统只做确定性判断（有无工具调用、文本是否变化），不做语义质量评估。
		if maxNoProgress > 0 {
			madeProgress := len(resp.Message.ToolCalls) > 0 ||
				(strings.TrimSpace(acontent) != "" && acontent != prevContent)
			if madeProgress {
				noProgressStreak = 0
			} else {
				noProgressStreak++
			}
			prevContent = acontent
			if noProgressStreak >= maxNoProgress {
				e.log.Warn("连续无进展，阻塞阶段", "qid", qid, "sid", sid,
					"streak", noProgressStreak, "max", maxNoProgress)
				e.appendSessionRow(qid, sid, &fsstore.QuestSessionRow{
					Seq: turn, Timestamp: fsstore.NowMs(), Kind: "no_progress", SessionID: sid,
					Role: "system", Content: fmt.Sprintf("连续 %d 轮无进展", noProgressStreak),
					Phase: phase,
				})
				return finalContent, PhaseSignal{
					OK:           false,
					Message:      fmt.Sprintf("连续 %d 轮无进展，阶段阻塞", noProgressStreak),
					PhaseEnded:   true,
					PhaseVerdict: "blocked",
					PhaseComment: fmt.Sprintf("连续 %d 轮无新工具调用且无新推理产出", noProgressStreak),
					Data: map[string]any{
						"reason":             "no_progress",
						"no_progress_streak": noProgressStreak,
					},
				}, nil
			}
		}

		// ===== OBSERVE：如果 finish_reason=stop 但是回合还有剩余，下回合发 continue user 消息 =====
		if resp.FinishReason == "stop" {
			// 下一轮循环自动插入「继续推进」user 消息
		}
	}
	return finalContent, phaseEnded, nil
}

func (e *Engine) publishExternalStateWriteObservations(qid, sid, workspacePath string, ex executor.Executor, resp *executor.ChatResponse) {
	if resp == nil {
		return
	}
	observations := traexFileChangeObservationsFromMeta(resp.Meta)
	if len(observations) == 0 {
		return
	}
	workspaceAbs, err := filepath.Abs(workspacePath)
	if err != nil || strings.TrimSpace(workspaceAbs) == "" {
		return
	}
	for _, obs := range observations {
		path := strings.TrimSpace(obs.Path)
		if path == "" {
			continue
		}
		absPath := path
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(workspaceAbs, absPath)
		}
		absPath, err = filepath.Abs(absPath)
		if err != nil {
			continue
		}
		if pathWithinDir(absPath, workspaceAbs) {
			continue
		}
		payload := map[string]any{
			"path":      absPath,
			"raw_path":  path,
			"kind":      obs.Kind,
			"tool":      obs.Tool,
			"status":    obs.Status,
			"transport": "traex",
		}
		if ex != nil {
			payload["executor_id"] = ex.ID()
			payload["executor_name"] = ex.Name()
		}
		e.publish(qid, sid, events.EvtAgentExternalStateWriteObserved, payload)
	}
}

func traexFileChangeObservationsFromMeta(meta any) []executor.FileChangeObservation {
	m, ok := meta.(map[string]any)
	if !ok {
		return nil
	}
	raw, ok := m["traex_file_changes"]
	if !ok {
		return nil
	}
	switch changes := raw.(type) {
	case []executor.FileChangeObservation:
		return changes
	case []any:
		out := make([]executor.FileChangeObservation, 0, len(changes))
		for _, item := range changes {
			if obs, ok := item.(executor.FileChangeObservation); ok {
				out = append(out, obs)
				continue
			}
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			path, _ := m["path"].(string)
			kind, _ := m["kind"].(string)
			tool, _ := m["tool"].(string)
			status, _ := m["status"].(string)
			if strings.TrimSpace(path) == "" {
				continue
			}
			out = append(out, executor.FileChangeObservation{
				Path:   path,
				Kind:   kind,
				Tool:   tool,
				Status: status,
			})
		}
		return out
	default:
		return nil
	}
}

func pathWithinDir(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func (e *Engine) turnContext(parent context.Context, _ int64) (context.Context, context.CancelFunc) {
	// v0.3.6: 不再有 per-phase duration 限制，直接返回 cancelable context
	return context.WithCancel(parent)
}

func (e *Engine) watchRuntimeIdle(
	ctx context.Context,
	qid, sid string,
	turn, phase int,
	lastActivityMs *atomic.Int64,
) func() {
	thresholdMs := e.cfg.RuntimeIdleWarningMs
	if thresholdMs <= 0 || lastActivityMs == nil {
		return func() {}
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		timer := time.NewTimer(time.Duration(thresholdMs) * time.Millisecond)
		defer timer.Stop()
		warned := false
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-timer.C:
				last := lastActivityMs.Load()
				idleForMs := fsstore.NowMs() - last
				if idleForMs >= thresholdMs && !warned {
					warned = true
					e.publishRuntimeHealthEvent(qid, sid, executor.RuntimeEventIdleWarning, map[string]any{
						"threshold_ms":        thresholdMs,
						"idle_for_ms":         idleForMs,
						"last_activity_at_ms": last,
					}, map[string]any{"turn": turn, "phase": phase})
				}
				if warned {
					return
				}
				next := time.Duration(thresholdMs-idleForMs) * time.Millisecond
				if next <= 0 {
					next = time.Duration(thresholdMs) * time.Millisecond
				}
				timer.Reset(next)
			}
		}
	}()
	return func() {
		close(stop)
		<-done
	}
}

func phaseSignalExpected(resp *executor.ChatResponse) bool {
	if resp == nil {
		return false
	}
	if resp.PhaseSignal != nil && resp.PhaseSignal.PhaseEnded {
		return true
	}
	if len(resp.Message.ToolCalls) > 0 {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(resp.FinishReason)) {
	case "tool_calls", "phase_signal", "blocked", "error":
		return true
	default:
		return false
	}
}

// isSystemProtocolMessage 判断内容是否为 agent runtime 的系统协议消息。
// 这些消息是 agent 和 runtime 之间的握手（hook lifecycle、session init），
// 对用户没有价值，不应写入 quest trace。
func isSystemProtocolMessage(content string) bool {
	s := strings.TrimSpace(content)
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, `{"type":"system"`) || strings.HasPrefix(s, `[{"type":"system"`) {
		return true
	}
	if strings.HasPrefix(s, "{") && strings.Contains(s, `"hook_id"`) {
		return true
	}
	return false
}

func classifyAgentError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "not logged in"),
		strings.Contains(msg, "please run /login"),
		strings.Contains(msg, "authentication required"),
		strings.Contains(msg, "auth required"),
		strings.Contains(msg, "login required"),
		strings.Contains(msg, "unauthorized"),
		// 某些 CLI 二进制按构建形态裁剪了鉴权后端（如 relay 在特定
		// 构建里不含 Anthropic 鉴权），报错形如
		// "Anthropic authentication is not supported in this build"。
		// 这类错误不是瞬态的——重试或加 hint 轮次都无效，必须走 auth
		// fail-fast：一次非交互恢复尝试后直接 blocked，避免在
		// consecutiveErrors 循环里空耗 turn/duration 预算。
		strings.Contains(msg, "authentication is not supported"),
		strings.Contains(msg, "not supported in this build"):
		return "auth"
	case strings.Contains(msg, "no api key"),
		strings.Contains(msg, "api key not found"),
		strings.Contains(msg, "missing api key"),
		strings.Contains(msg, "model metadata"),
		strings.Contains(msg, "cannot be routed"),
		strings.Contains(msg, "unknown model"),
		strings.Contains(msg, "invalid model"):
		return "configuration"
	case strings.Contains(msg, "executable file not found"),
		strings.Contains(msg, "no such file or directory"),
		strings.Contains(msg, "command not found"),
		strings.Contains(msg, "not found in $path"):
		return "configuration"
	case strings.Contains(msg, "429"),
		strings.Contains(msg, "rate limit"),
		strings.Contains(msg, "rate_limit"),
		strings.Contains(msg, "too many requests"),
		strings.Contains(msg, "overloaded"),
		strings.Contains(msg, "quota"):
		return "rate_limit"
	case
		// session 不存在：recovery retry / 并发清理 / client 连接断开都可能触发。
		// 本质 transient——retry 会幂等重建 session（P0-2 保证），不该当
		// unknown 空耗预算后落 require_user。本地 4 个 quest 因它 blocked。
		strings.Contains(msg, "session 不存在"),
		strings.Contains(msg, "session not found"),
		strings.Contains(msg, "no such session"):
		return "session_lost"
	case strings.Contains(msg, "timeout"),
		strings.Contains(msg, "timed out"),
		strings.Contains(msg, "deadline exceeded"),
		strings.Contains(msg, "context canceled"),
		strings.Contains(msg, "temporarily unavailable"),
		strings.Contains(msg, "connection reset"),
		strings.Contains(msg, "eof"):
		return "transient"
	default:
		return "unknown"
	}
}

func maxRetryableAgentErrors() int {
	return 3
}

func agentRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	// 指数退避：2s/4s/8s。网络超时后上游可能需要 10-30s 恢复，
	// 线性 1-3s 太短会撞同样的超时；指数退避给上游恢复时间，总耗时 14s 可接受。
	return time.Duration(1<<attempt) * time.Second
}

func (e *Engine) publishRuntimeHealthEvent(qid, sid string, event string, data any, extra map[string]any) {
	payload := map[string]any{}
	if m, ok := data.(map[string]any); ok {
		for k, v := range m {
			payload[k] = v
		}
	} else if data != nil {
		payload["data"] = data
	}
	for k, v := range extra {
		payload[k] = v
	}
	e.updateAgentSessionState(qid, sid, event, payload)
	e.appendSessionRow(qid, sid, &fsstore.QuestSessionRow{
		Timestamp: fsstore.NowMs(),
		Kind:      "runtime_health",
		SessionID: sid,
		Role:      "system",
		Content:   event,
		Meta:      payload,
	})
	e.publish(qid, sid, events.EventType(event), payload)
}

func (e *Engine) updateAgentSessionState(qid, sid string, event string, payload map[string]any) {
	if e == nil || e.root == nil || qid == "" || sid == "" {
		return
	}
	_, err := e.root.PatchAgentSessionState(qid, sid, func(state *fsstore.AgentSessionState) {
		if v, ok := payload["phase"].(int); ok {
			state.Phase = v
		}
		if v, ok := payload["agent_id"].(string); ok && v != "" {
			state.AgentID = v
		}
		if v, ok := payload["capability_tier"].(string); ok && v != "" {
			state.CapabilityTier = v
		}
		now := fsstore.NowMs()
		switch event {
		case executor.RuntimeEventSessionCreated:
			state.HealthStatus = "created"
			if state.StartedAtMs == 0 {
				state.StartedAtMs = now
			}
		case executor.RuntimeEventProcessStarted:
			state.HealthStatus = "process_started"
			if state.StartedAtMs == 0 {
				state.StartedAtMs = now
			}
		case executor.RuntimeEventOutputFirst:
			state.HealthStatus = "active"
			if state.FirstOutputAtMs == 0 {
				state.FirstOutputAtMs = now
			}
			state.LastOutputAtMs = now
		case executor.RuntimeEventOutputHeartbeat:
			state.HealthStatus = "active"
			state.LastOutputAtMs = now
		case executor.RuntimeEventProtocolSignalSeen:
			state.HealthStatus = "waiting_signal"
		case executor.RuntimeEventIdleWarning:
			state.HealthStatus = "idle"
		case executor.RuntimeEventProtocolWarning:
			state.HealthStatus = "protocol_risk"
		case executor.RuntimeEventProcessExited:
			state.HealthStatus = "exited"
			if code, ok := intFromAny(payload["exit_code"]); ok {
				state.ExitCode = &code
			}
		case string(events.EvtAgentErrorClassified):
			state.HealthStatus = "error"
		}
	})
	if err != nil {
		e.log.Warn("保存 agent session state 失败", "qid", qid, "sid", sid, "event", event, "err", err)
	}
}

func intFromAny(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	case float32:
		return int(x), true
	default:
		return 0, false
	}
}

func agentErrorBlockedResult(err error, category string, consecutiveErrors int, ex executor.Executor) PhaseSignal {
	actions := recoveryActionsForExecutor(ex)
	data := map[string]any{
		"reason":             "agent_error_" + category,
		"category":           category,
		"consecutive_errors": consecutiveErrors,
		"last_error":         err.Error(),
	}
	if len(actions) > 0 {
		data["actions"] = actions
	}
	return PhaseSignal{
		OK:           false,
		Message:      "agent_error_" + category,
		PhaseEnded:   true,
		PhaseVerdict: "blocked",
		PhaseComment: fmt.Sprintf("agent 执行错误（%s），已阻塞；连续错误 %d 次。最后错误：%s", category, consecutiveErrors, err.Error()),
		Data:         data,
	}
}

func (e *Engine) consumePhaseSignal(qid, sid, workDir string, phase int) (PhaseSignal, bool) {
	sig, err := fsstore.ReadPhaseSignal(workDir, sid)
	if err != nil || sig == nil {
		return PhaseSignal{}, false
	}
	_ = fsstore.RemovePhaseSignal(workDir, sid)
	result := PhaseSignal{
		OK:           sig.OK,
		Message:      sig.Message,
		Data:         sig.Data,
		PhaseEnded:   sig.PhaseEnded,
		PhaseVerdict: sig.PhaseVerdict,
		PhaseComment: sig.PhaseComment,
		PhaseHints:   sig.PhaseHints,
		PhaseScore:   sig.PhaseScore,
	}
	resultJSON := PhaseSignalToJSON(result)
	e.appendSessionRow(qid, sid, &fsstore.QuestSessionRow{
		Seq: 0, Timestamp: fsstore.NowMs(), Kind: "tool_result",
		SessionID: sid, Role: "tool", ToolName: "gloop_cli_signal",
		Content: resultJSON, Phase: phase,
		Meta: map[string]any{"source": "cli_signal"},
	})
	e.publish(qid, sid, events.EvtToolEnd, map[string]any{
		"tool":   "gloop_cli_signal",
		"id":     sid,
		"result": truncate(resultJSON, 400),
	})
	return result, true
}
