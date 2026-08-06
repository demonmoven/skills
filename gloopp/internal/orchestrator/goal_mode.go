package orchestrator

import (
	"context"
	"fmt"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
)

// goalModeMaxIterations 是 goal 模式的默认最大迭代次数。
// 超过后 quest 正常完成（不再继续迭代），让人感知到"目标未在预算内达成"。
const goalModeDefaultMaxIterations = 10

// checkGoalCompletion 是 HOTL v0.2 /goal 模式的完成度检查。
//
// 制造者/检查者分离延伸到停止条件：quest 自主闭环（checker pass + apply）后，
// 独立检查 goal 条件是否达成。done → 正常完成；continue → 自动迭代。
//
// 当前实现用规则检查（acceptance criteria / goal condition 文本匹配），
// 未来可接入独立小模型做语义级完成度判断。
func (e *Engine) checkGoalCompletion(ctx context.Context, qid string) {
	q, err := fsstore.NewQuestStore(e.root).LoadQuest(qid)
	if err != nil {
		e.log.Error("goal check: 加载 quest 失败", "qid", qid, "err", err)
		return
	}
	if q.GoalCondition == "" && q.GoalMaxIterations == 0 {
		return // 非 goal 模式
	}

	maxIter := q.GoalMaxIterations
	if maxIter == 0 {
		maxIter = goalModeDefaultMaxIterations
	}

	// 迭代次数已达上限 → 正常完成，不再迭代
	if q.GoalIterations >= maxIter {
		e.log.Info("goal mode: 达到最大迭代次数，停止迭代", "qid", qid, "iterations", q.GoalIterations, "max", maxIter)
		e.publish(qid, "", "goal.iteration_limit", map[string]any{
			"iterations":     q.GoalIterations,
			"max_iterations": maxIter,
			"message":        "goal 模式达到最大迭代次数，quest 正常完成",
		})
		return
	}

	// 完成度检查：当前用 acceptance criteria 是否满足做判断
	// （如果 quest 有验收标准且产出里包含通过信号，视为 done）
	done := e.evaluateGoalCondition(q)
	if done {
		e.log.Info("goal mode: 目标达成", "qid", qid, "iterations", q.GoalIterations)
		e.publish(qid, "", "goal.achieved", map[string]any{
			"iterations": q.GoalIterations,
			"message":    "goal 模式目标达成",
		})
		return
	}

	// 未达成 → 迭代计数 +1，重新触发 quest
	q.GoalIterations++
	if err := fsstore.NewQuestStore(e.root).SaveQuest(q); err != nil {
		e.log.Error("goal mode: 保存迭代计数失败", "qid", qid, "err", err)
		return
	}
	e.log.Info("goal mode: 目标未达成，迭代", "qid", qid, "iterations", q.GoalIterations, "max", maxIter)
	e.publish(qid, "", "goal.continue", map[string]any{
		"iterations":     q.GoalIterations,
		"max_iterations": maxIter,
		"message":        "goal 模式目标未达成，继续迭代",
	})

	// 重新创建 quest（继承 goal 配置）
	if q.CreatedBy != "" && strings.HasPrefix(q.CreatedBy, "automation:") {
		autoID := strings.TrimPrefix(q.CreatedBy, "automation:")
		e.log.Info("goal mode: 重新触发 automation", "qid", qid, "automation", autoID)
		if _, err := e.RunAutomation(ctx, autoID); err != nil {
			e.log.Error("goal mode: 重新触发失败", "qid", qid, "automation", autoID, "err", err)
		}
	}
}

// evaluateGoalCondition 评估 goal 条件是否达成。
//
// 当前实现（规则级）：
// - 如果 quest 有 AcceptanceCriteria，检查产出中是否包含通过信号
// - 如果没有明确验收标准，默认 done（让 checker pass 作为完成信号）
//
// 未来可替换为独立小模型做语义级判断。
func (e *Engine) evaluateGoalCondition(q *fsstore.QuestMeta) bool {
	// 无验收标准 → checker pass 即视为 done
	if strings.TrimSpace(q.AcceptanceCriteria) == "" {
		return true
	}

	// 有验收标准 → 检查最近的事件里是否有 "pass" / "done" 信号
	// （简化实现：如果 quest 已经 success 且 checker pass，视为达成）
	// 更精确的检查需要读 quest 产出并匹配验收标准，留给后续独立模型实现
	return q.Status == "success" || q.Status == "applied"
}

// setupGoalMode 在创建 quest 时初始化 goal 模式配置。
// 由 CreateQuest 调用，从 automation 配置继承 goal 参数。
func setupGoalMode(q *fsstore.QuestMeta, autoCfg *fsstore.AutomationConfig) {
	if autoCfg == nil || autoCfg.Trigger != fsstore.TriggerGoal {
		return
	}
	maxIter := autoCfg.MaxIterations
	if maxIter == 0 {
		maxIter = goalModeDefaultMaxIterations
	}
	q.GoalMaxIterations = maxIter
	// goal 条件用 Query 描述（automation 没有独立 Goal 字段，Query 即目标）
	q.GoalCondition = autoCfg.Query
}

// RunGoalAutomation 创建一个 goal 模式的 quest。
func (e *Engine) RunGoalAutomation(ctx context.Context, id string) (string, error) {
	cfg, err := e.root.GetAutomation(id)
	if err != nil {
		return "", fmt.Errorf("automation 不存在: %w", err)
	}
	if cfg.Trigger != fsstore.TriggerGoal {
		return "", fmt.Errorf("automation %s 不是 goal 模式", id)
	}
	qid, err := e.RunAutomation(ctx, id)
	if err != nil {
		return "", err
	}
	// 补写 goal 配置到 quest
	q, err := fsstore.NewQuestStore(e.root).LoadQuest(qid)
	if err != nil {
		return qid, err
	}
	setupGoalMode(q, cfg)
	if err := fsstore.NewQuestStore(e.root).SaveQuest(q); err != nil {
		e.log.Error("goal mode: 初始化配置失败", "qid", qid, "err", err)
	}
	return qid, nil
}
