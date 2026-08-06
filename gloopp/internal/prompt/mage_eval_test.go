package prompt

import (
	"fmt"
	"strings"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

func TestPrintFullMagePrompts(t *testing.T) {
	adv := &fsstore.AdventurerFile{
		Name:  "Test Mage",
		Class: model.ClassMage,
		Level: 1,
	}
	sys, err := BuildSystem(adv)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("=== MAGE SYSTEM PROMPT ===")
	fmt.Println(sys)
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	// ====== 场景1: code_change 有埋雷 ======
	quest1 := &fsstore.QuestMeta{
		ID:        "qst_code_001",
		ShortID:   "code_001",
		Query:     "给 gloop 添加一个 gloop stats 命令，用于统计用户的冒险者战绩和任务完成率。要求：1) 支持按职业筛选 2) 支持按时间范围筛选 3) 输出 JSON 和表格两种格式 4) 单元测试覆盖核心逻辑",
		Type:      model.QuestTypeExecute,
		Intensity: model.QuestIntensityStandard,
		Status:    model.QuestStatusReviewing,
		MaxRework: 3,
	}
	warrior1 := "## 任务完成总结\n\n**task_type**: code_change\n**side_effect_level**: L1 本地可回滚变更\n\n### 实际做的动作\n1. 新增 internal/cli/stats_cmd.go — 实现 stats 子命令\n2. 新增 internal/cli/stats_cmd_test.go — 单元测试，覆盖率约 85%\n3. 修改 internal/cli/root.go — 注册 stats 子命令\n4. 新增 internal/fsstore/stats.go — 统计数据聚合逻辑\n\n### 产物\n- 新增 3 个文件，修改 1 个文件\n- 新增约 320 行代码\n\n### 已完成验证\n- go test ./internal/cli/... -v 全部通过（12 个测试用例）\n- go build ./cmd/gloop 编译成功\n- 手动运行 gloop stats --class warrior --format json 输出正确\n\n### 残余风险\n- 时间范围筛选的边界条件测试覆盖有限\n- 大数量级下的性能未测试\n\n### 验收标准满足情况\n1. 按职业筛选：已实现 --class 参数\n2. 按时间范围筛选：已实现 --since 和 --until 参数\n3. JSON 和表格格式：已实现 --format 参数\n4. 单元测试：12 个用例，覆盖率 85%"

	pack1 := BuildReviewPrompt(quest1, warrior1)
	fmt.Println("=== SCENARIO 1: code_change (埋雷场景) ===")
	fmt.Println(pack1)
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	// ====== 场景2: research_analysis 有埋雷 ======
	quest2 := &fsstore.QuestMeta{
		ID:        "qst_research_001",
		ShortID:   "research_001",
		Query:     "调研当前主流的 Agent Loop Engineering 框架，对比 LangGraph、AutoGen、Gloop 三者的架构设计、适用场景和性能表现，输出一份可作为技术选型参考的调研报告。",
		Type:      model.QuestTypeExecute,
		Intensity: model.QuestIntensityDeep,
		Status:    model.QuestStatusReviewing,
		MaxRework: 4,
	}
	warrior2 := "## 调研报告：主流 Agent Loop 框架对比\n\n**task_type**: research_analysis\n**side_effect_level**: L0 纯读/分析\n\n### 背景\n随着 AI Agent 技术的快速发展，Agent Loop Engineering 框架层出不穷。本文对比三款主流框架：LangGraph、AutoGen 和 Gloop。\n\n### 架构设计对比\n\n| 维度 | LangGraph | AutoGen | Gloop |\n|------|-----------|---------|-------|\n| 范式 | 状态机图 | 多智能体对话 | 双职业编排 |\n| 语言 | Python | Python | Go |\n| 核心抽象 | Node/Edge | Agent/GroupChat | Warrior/Mage |\n| 可观测性 | LangSmith | 有限 | Dashboard + events.jsonl |\n\n### 适用场景\n- LangGraph：复杂多步骤工作流、需要精细控制状态流转的场景\n- AutoGen：多智能体协作、需要对话式交互的场景\n- Gloop：本地开发场景、代码任务、需要 Maker-Checker 分离的场景\n\n### 性能表现\n根据官方 benchmark 数据：\n- LangGraph：单任务平均 2.3s 延迟\n- AutoGen：多 agent 场景平均 5.1s 延迟\n- Gloop：单 quest 平均 1.8s 延迟（本地执行）\n\n### 推荐选型建议\n- 如果是 Python 生态且需要复杂工作流 → LangGraph\n- 如果需要多 agent 对话协作 → AutoGen\n- 如果是本地开发、代码任务优先 → Gloop\n\n### 验证\n对比了三个框架的官方文档和 GitHub README，信息准确。\n\n### 残余风险\n- 性能数据来自官方，可能有偏差\n- 三个框架的最新版本可能有变化"

	pack2 := BuildReviewPrompt(quest2, warrior2)
	fmt.Println("=== SCENARIO 2: research_analysis (埋雷场景) ===")
	fmt.Println(pack2)
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	// ====== 场景3: design 方案设计 / adversarial 强度 有埋雷 ======
	quest3 := &fsstore.QuestMeta{
		ID:                 "qst_design_001",
		ShortID:            "design_001",
		Query:              "设计一个分布式任务队列系统，支持延迟消息、死信队列、优先级队列和水平扩展。产出可执行的技术方案。",
		Type:               model.QuestTypeDesign,
		Intensity:          model.QuestIntensityAdversarial,
		Status:             model.QuestStatusReviewing,
		MaxRework:          5,
		AcceptanceCriteria: "1. 方案必须支持至少 10k QPS 的吞吐量\n2. 必须有明确的故障转移和恢复机制\n3. 必须包含数据一致性方案\n4. 必须有完整的监控和告警方案\n5. 必须评估成本",
	}
	warrior3 := "# 分布式任务队列系统设计方案\n\n## 1. 背景与问题定义\n当前系统需要一个高可用、可扩展的分布式任务队列，支撑异步任务处理、延迟消息投递等场景。\n\n## 2. 目标与非目标\n### 目标\n- 支持延迟消息、死信队列、优先级队列\n- 支持水平扩展\n- 高可用，99.9% SLA\n\n### 非目标\n- 不支持事务消息\n- 不支持消息回溯\n\n## 3. 推荐方案\n基于 Redis Streams 实现分布式任务队列。\n\n### 架构\n- Producer：业务服务写入 Redis Streams\n- Consumer Worker：多个 worker 实例消费\n- Redis Cluster：3 主 3 从，保证高可用\n- Dead Letter Queue：消费失败超过 N 次的消息转移到死信队列\n\n### 关键接口\n```go\ntype TaskQueue interface {\n    Enqueue(task Task) error\n    EnqueueDelayed(task Task, delay time.Duration) error\n    Dequeue() (Task, error)\n    Ack(taskID string) error\n}\n```\n\n## 4. 迁移/实施步骤\n1. 搭建 Redis Cluster 集群\n2. 实现 TaskQueue 接口\n3. 接入 producer 端\n4. 接入 consumer 端\n5. 压测验证\n\n## 5. 风险、取舍与回滚策略\n- 风险：Redis 数据可能丢失 → 缓解：开启 AOF 持久化\n- 回滚：切换回原有队列方案\n\n## 6. 验收标准\n- 延迟消息误差 < 1s\n- 系统支持水平扩展\n- 死信队列功能正常\n- 优先级队列按优先级消费\n\n## 7. 总结\n本方案基于成熟的 Redis Streams 技术，简单可靠，易于落地。"

	pack3 := BuildReviewPrompt(quest3, warrior3)
	fmt.Println("=== SCENARIO 3: design / adversarial 强度 (埋雷场景) ===")
	fmt.Println(pack3)
}
