package orchestrator

import (
	"fmt"
	"strings"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/prompt"
)

// ==================== B 方案可行性验证 ====================
//
// 问题：剑士侧能力优先效果差，核心原因是 execution_instruction 太大。
// 如果把 execution_instruction 也变成"可查询"，能省多少？
// 代价是什么？

// 实验 B-1：激进模式 — 只留 quest_user_intent，其他全部可查询
func TestBScheme_RadicalMode(t *testing.T) {
	scenarios := []struct {
		name        string
		quest       *fsstore.QuestMeta
		reworkHints string
		prevSummary string
	}{
		{
			name: "Round 0 标准任务",
			quest: &fsstore.QuestMeta{
				Query:       "实现用户认证模块，包含登录注册功能",
				Type:        model.QuestTypeExecute,
				Intensity:   model.QuestIntensityStandard,
				ReworkCount: 0,
				MaxRework:   3,
			},
		},
		{
			name: "Round 1 返工",
			quest: &fsstore.QuestMeta{
				Query:       "实现用户认证模块，包含登录注册功能",
				Type:        model.QuestTypeExecute,
				Intensity:   model.QuestIntensityStandard,
				ReworkCount: 1,
				MaxRework:   3,
				ReviewHints: "请补充单元测试，覆盖边界条件；密码需要哈希存储。",
			},
			reworkHints: "请补充单元测试，覆盖边界条件；密码需要哈希存储。",
			prevSummary: "上一轮交付：基础登录注册接口，明文密码，无单测。",
		},
		{
			name: "Round 2 返工（有验收标准）",
			quest: &fsstore.QuestMeta{
				Query:              "实现用户认证模块",
				Type:               model.QuestTypeExecute,
				Intensity:          model.QuestIntensityDeep,
				ReworkCount:        2,
				MaxRework:          3,
				ReviewHints:        "密码哈希强度不够，使用 bcrypt；错误处理需要更细致。",
				AcceptanceCriteria: "1. 邮箱注册登录\n2. 密码 bcrypt 哈希\n3. JWT 鉴权\n4. 单元测试覆盖率>80%",
				DiffStat:           "3 files changed, 150 insertions(+), 30 deletions(-)",
				DiffChangedFiles:   3,
				DiffAdditions:      150,
				DiffDeletions:      30,
			},
			reworkHints: "密码哈希强度不够，使用 bcrypt；错误处理需要更细致。",
			prevSummary: "上一轮交付：SHA1 哈希密码，基础错误处理。",
		},
	}

	t.Log("========== B-1 激进模式：只留 quest_user_intent ==========")
	t.Logf("%-25s %-10s %-10s %-10s", "场景", "全量 chars", "激进 chars", "节省比例")
	t.Log(strings.Repeat("-", 60))

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			fullPack := prompt.BuildQuestContextPack(sc.quest, sc.reworkHints, sc.prevSummary)
			fullChars := len(fullPack.RenderXMLish())

			radPack := fullPack.ToCapabilityMode(prompt.CapabilityModeConfig{
				CoreBlocks: []string{"quest_user_intent"},
			})
			radChars := len(radPack.RenderXMLish())

			savings := float64(fullChars-radChars) / float64(fullChars) * 100

			t.Logf("%-25s %-10d %-10d %-10.1f%%",
				sc.name, fullChars, radChars, savings)

			queryable := fullPack.QueryableBlocks([]string{"quest_user_intent"})
			t.Logf("  可查询 blocks (%d): %v", len(queryable), queryable)
		})
	}
}

// 实验 B-2：拆分 execution_instruction
//
// execution_instruction 目前是"一锅端"，里面有：
//   1. 数据信任警示（必选，协议级）
//   2. phase_checkpoint 接口契约（必选，协议级）
//   3. skill 加载提示（可选，信息级）
//
// 如果把 1+2 精简后保留为核心，3 和其他细节移到可查询块，
// 就能在不破坏协议的前提下大幅瘦身。
//
// 这个实验模拟"瘦身版" execution_instruction 的效果。

func TestBScheme_SplitExecutionInstruction(t *testing.T) {
	quest := &fsstore.QuestMeta{
		Query:       "实现用户认证模块",
		Type:        model.QuestTypeExecute,
		Intensity:   model.QuestIntensityStandard,
		ReworkCount: 1,
		MaxRework:   3,
		ReviewHints: "请补充单元测试。",
	}
	reworkHints := "请补充单元测试。"
	prevSummary := "上一轮交付摘要。"

	fullPack := prompt.BuildQuestContextPack(quest, reworkHints, prevSummary)
	fullChars := len(fullPack.RenderXMLish())

	// 找到 execution_instruction 块的大小
	execBlock, ok := fullPack.QueryBlock("execution_instruction")
	if !ok {
		t.Fatal("找不到 execution_instruction 块")
	}
	execChars := len(execBlock.Content)

	t.Log("========== B-2 execution_instruction 拆分分析 ==========")
	t.Logf("全量 prompt: %d chars", fullChars)
	t.Logf("execution_instruction 大小: %d chars (占 %.1f%%)",
		execChars, float64(execChars)/float64(fullChars)*100)

	// 模拟三种拆分方案
	scenarios := []struct {
		name        string
		coreKeepPct float64 // 保留百分之多少作为核心协议
		description string
	}{
		{"保守拆分", 0.6, "保留大部分协议，只移走 skill 提示和例子"},
		{"平衡拆分", 0.4, "保留核心接口契约，细节移到可查询"},
		{"激进拆分", 0.2, "只留一句话协议，全部细节可查询"},
	}

	t.Log("")
	t.Logf("%-15s %-12s %-12s %-12s %-12s",
		"拆分方案", "核心保留", "核心 exec chars", "总 prompt", "节省比例")
	t.Log(strings.Repeat("-", 65))

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			coreExecChars := int(float64(execChars) * sc.coreKeepPct)
			movedChars := execChars - coreExecChars

			// 模拟：execution_instruction 瘦身 + 新增一个 execution_details 可查询块
			// 总 token = 全量 - 移走的 + catalog 增量
			// catalog 增量很小，暂时忽略
			estimatedTotal := fullChars - movedChars
			savings := float64(fullChars-estimatedTotal) / float64(fullChars) * 100

			t.Logf("%-15s %-12.0f%% %-14d %-12d %-11.1f%%",
				sc.name, sc.coreKeepPct*100, coreExecChars, estimatedTotal, savings)
			t.Logf("  描述: %s", sc.description)
			t.Logf("  移至可查询: %d chars", movedChars)
		})
	}
}

// 实验 B-3：拆分后能力优先的效果
//
// 结合 B-2 的"平衡拆分"，再加上能力优先模式，看看整体效果。

func TestBScheme_CombinedEffect(t *testing.T) {
	scenarios := []struct {
		name        string
		quest       *fsstore.QuestMeta
		reworkHints string
		prevSummary string
	}{
		{
			name: "Round 0 标准任务",
			quest: &fsstore.QuestMeta{
				Query:       "实现用户认证模块",
				Type:        model.QuestTypeExecute,
				Intensity:   model.QuestIntensityStandard,
				ReworkCount: 0,
				MaxRework:   3,
			},
		},
		{
			name: "Round 1 返工",
			quest: &fsstore.QuestMeta{
				Query:       "实现用户认证模块",
				Type:        model.QuestTypeExecute,
				Intensity:   model.QuestIntensityStandard,
				ReworkCount: 1,
				MaxRework:   3,
				ReviewHints: "请补充单元测试。",
			},
			reworkHints: "请补充单元测试。",
			prevSummary: "上一轮交付摘要。",
		},
		{
			name: "Round 2 返工（全量）",
			quest: &fsstore.QuestMeta{
				Query:              "实现用户认证模块",
				Type:               model.QuestTypeExecute,
				Intensity:          model.QuestIntensityDeep,
				ReworkCount:        2,
				MaxRework:          3,
				ReviewHints:        "密码哈希强度不够。",
				AcceptanceCriteria: "1. 邮箱注册\n2. 密码哈希\n3. JWT",
				DiffStat:           "3 files changed",
				DiffChangedFiles:   3,
			},
			reworkHints: "密码哈希强度不够。",
			prevSummary: "上一轮交付摘要。",
		},
	}

	t.Log("========== B-3 组合效果：拆分 exec + 能力优先 ==========")
	t.Log("（假设 execution_instruction 平衡拆分，60% 移至可查询）")
	t.Log("")
	t.Logf("%-25s %-10s %-10s %-10s %-10s",
		"场景", "全量 chars", "现状能力", "拆分+能力", "总节省")
	t.Log(strings.Repeat("-", 65))

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			fullPack := prompt.BuildQuestContextPack(sc.quest, sc.reworkHints, sc.prevSummary)
			fullChars := len(fullPack.RenderXMLish())

			// 现状能力优先（默认核心块）
			currentCapPack := fullPack.ToCapabilityMode(prompt.CapabilityModeConfig{
				CoreBlocks: prompt.DefaultWarriorCoreBlocks,
			})
			currentCapChars := len(currentCapPack.RenderXMLish())

			// 模拟拆分后：execution_instruction 瘦身 40%，其余 60% 变成 execution_details 可查询
			// 这里用估算方式
			execBlock, _ := fullPack.QueryBlock("execution_instruction")
			execOrigChars := len(execBlock.Content)
			execCoreChars := int(float64(execOrigChars) * 0.4) // 保留 40%
			savedFromSplit := execOrigChars - execCoreChars

			// 拆分后的全量大小
			//（其实差不多，因为只是拆分，内容还在；但能力优先模式下可以不注入）
			// 这里我们直接算"拆分后 + 能力优先"的大小：
			// 核心 = quest_user_intent + 瘦身的 execution_instruction + catalog
			// catalog 会多一个 execution_details 条目

			// 简化计算：在当前能力优先基础上，再减去 exec 移走的部分
			splitCapChars := currentCapChars - savedFromSplit + 50 // +50 是 catalog 新增条目

			currentSavings := float64(fullChars-currentCapChars) / float64(fullChars) * 100
			totalSavings := float64(fullChars-splitCapChars) / float64(fullChars) * 100

			t.Logf("%-25s %-10d %-10d %-10d %-10.1f%%",
				sc.name, fullChars, currentCapChars, splitCapChars, totalSavings)
			t.Logf("  现状能力优先节省: %.1f%% → 拆分后额外节省: %.1f%%",
				currentSavings, totalSavings-currentSavings)
		})
	}
}

// 实验 B-4：代价分析 — agent 需要查询多少次才能回本？
//
// 能力优先不是免费的。每次查询都有 token 开销（查询请求 + 返回内容）。
// 这个实验计算"查询几个 block 后，总 token 会超过全量注入"。

func TestBScheme_BreakEvenAnalysis(t *testing.T) {
	quest := &fsstore.QuestMeta{
		Query:              "实现用户认证模块",
		Type:               model.QuestTypeExecute,
		Intensity:          model.QuestIntensityDeep,
		ReworkCount:        2,
		MaxRework:          3,
		ReviewHints:        "密码哈希强度不够，使用 bcrypt；错误处理需要更细致。",
		AcceptanceCriteria: "1. 邮箱注册登录\n2. 密码 bcrypt 哈希\n3. JWT 鉴权\n4. 单元测试覆盖率>80%",
		DiffStat:           "3 files changed, 150 insertions(+), 30 deletions(-)",
		DiffChangedFiles:   3,
		DiffAdditions:      150,
		DiffDeletions:      30,
	}
	reworkHints := "密码哈希强度不够，使用 bcrypt；错误处理需要更细致。"
	prevSummary := "上一轮交付：SHA1 哈希密码，基础错误处理。"

	fullPack := prompt.BuildQuestContextPack(quest, reworkHints, prevSummary)
	fullChars := len(fullPack.RenderXMLish())

	// 三种核心块方案
	configs := []struct {
		name       string
		coreBlocks []string
	}{
		{"默认核心（现状）", prompt.DefaultWarriorCoreBlocks},
		{"+拆分 exec（保留40%）", []string{"quest_user_intent", "execution_instruction"}}, // 简化，假设拆分后 exec 小很多
		{"仅 intent（激进）", []string{"quest_user_intent"}},
	}

	t.Log("========== B-4 回本分析：查询几个 block 后总 token 超过全量？ ==========")
	t.Logf("全量基准: %d chars", fullChars)
	t.Log("")

	for _, cfg := range configs {
		t.Run(cfg.name, func(t *testing.T) {
			capPack := fullPack.ToCapabilityMode(prompt.CapabilityModeConfig{
				CoreBlocks: cfg.coreBlocks,
			})
			baseChars := len(capPack.RenderXMLish())

			queryable := fullPack.QueryableBlocks(cfg.coreBlocks)

			t.Logf("--- %s ---", cfg.name)
			t.Logf("  首轮: %d chars (节省 %.1f%%)",
				baseChars, float64(fullChars-baseChars)/float64(fullChars)*100)

			// 累加查询，看查询几个后超过全量
			runningChars := baseChars
			breakEvenIdx := -1
			for i, name := range queryable {
				block, _ := fullPack.QueryBlock(name)
				// 查询开销 = block 内容 + 查询请求本身的固定开销（估算 30 chars）
				queryCost := len(block.Content) + 30
				runningChars += queryCost
				if breakEvenIdx < 0 && runningChars >= fullChars {
					breakEvenIdx = i + 1
				}
				t.Logf("  查 %d 个 (%s): %d chars (超全量 %+.0f%%)",
					i+1, name, runningChars,
					float64(runningChars-fullChars)/float64(fullChars)*100)
			}

			if breakEvenIdx > 0 {
				t.Logf("  ⚠️  查询 %d 个 block 后总 token 超过全量注入", breakEvenIdx)
			} else {
				t.Logf("  ✅  全部查完也没超全量（不可能，因为内容就是全量的）")
			}
		})
	}
}

// 实验 B-5：实际查询需求估算
//
// 理论上"查几个就超全量"，但实践中 agent 不会每个 block 都查。
// 我们估算一下实际场景中 agent 可能需要查几个 block。

func TestBScheme_RealisticQueryEstimate(t *testing.T) {
	// 场景：一个标准的返工任务，agent 会查什么？
	//
	// 基于对 agent 行为的理解：
	// - 返工场景：几乎肯定会查 latest_review_blockers（不看怎么改？）
	// - 返工场景：很可能查 previous_round_summary（确认上一轮做了什么）
	// - 有验收标准时：很可能查 acceptance_criteria
	// - diff 快照：可能查，也可能不查（如果摘要已经够清楚）
	// - rework_progress：大概率不查（不重要）
	// - intensity_note：大概率不查（不重要）
	// - execution_instruction：首轮不查，但执行中可能需要查协议细节

	t.Log("========== B-5 实际查询需求估算（剑士返工场景） ==========")
	t.Log("")

	quest := &fsstore.QuestMeta{
		Query:              "实现用户认证模块",
		Type:               model.QuestTypeExecute,
		Intensity:          model.QuestIntensityDeep,
		ReworkCount:        2,
		MaxRework:          3,
		ReviewHints:        "密码哈希强度不够，使用 bcrypt；错误处理需要更细致。",
		AcceptanceCriteria: "1. 邮箱注册登录\n2. 密码 bcrypt 哈希\n3. JWT 鉴权",
		DiffStat:           "3 files changed, 150 insertions(+), 30 deletions(-)",
		DiffChangedFiles:   3,
		DiffAdditions:      150,
		DiffDeletions:      30,
	}
	reworkHints := "密码哈希强度不够，使用 bcrypt；错误处理需要更细致。"
	prevSummary := "上一轮交付：SHA1 哈希密码，基础错误处理。"

	fullPack := prompt.BuildQuestContextPack(quest, reworkHints, prevSummary)
	fullChars := len(fullPack.RenderXMLish())

	// 四种 agent 画像
	agents := []struct {
		name        string
		queryBlocks []string
		description string
	}{
		{
			"保守型",
			[]string{"latest_review_blockers", "previous_round_summary", "acceptance_criteria", "workspace_diff_snapshot"},
			"保险起见，相关的都查",
		},
		{
			"平衡型",
			[]string{"latest_review_blockers", "previous_round_summary", "acceptance_criteria"},
			"查核心信息，diff 用摘要判断",
		},
		{
			"高效型",
			[]string{"latest_review_blockers", "acceptance_criteria"},
			"只查必须的，其他靠记忆和推理",
		},
		{
			"极简型",
			[]string{"latest_review_blockers"},
			"只看改动要求，其他不看",
		},
	}

	// 两种核心配置
	configs := []struct {
		name       string
		coreBlocks []string
	}{
		{"现状默认核心", prompt.DefaultWarriorCoreBlocks},
		// 假设拆分 exec 后，核心 exec 只有原来的 20%
	}

	t.Logf("全量基准: %d chars", fullChars)
	t.Log("")

	for _, cfg := range configs {
		t.Run(cfg.name, func(t *testing.T) {
			capPack := fullPack.ToCapabilityMode(prompt.CapabilityModeConfig{
				CoreBlocks: cfg.coreBlocks,
			})
			baseChars := len(capPack.RenderXMLish())

			t.Logf("【核心配置: %s】首轮 %d chars (节省 %.1f%%)",
				cfg.name, baseChars, float64(fullChars-baseChars)/float64(fullChars)*100)
			t.Log("")

			for _, agent := range agents {
				totalChars := baseChars
				for _, name := range agent.queryBlocks {
					block, ok := fullPack.QueryBlock(name)
					if ok {
						totalChars += len(block.Content) + 30 // +查询开销
					}
				}
				savings := float64(fullChars-totalChars) / float64(fullChars) * 100

				symbol := "✅"
				if savings < 0 {
					symbol = "⚠️"
				}

				t.Logf("  %s %-10s: 查 %d 个 → %d chars (节省 %+.1f%%)",
					symbol, agent.name, len(agent.queryBlocks), totalChars, savings)
				t.Logf("    描述: %s", agent.description)
			}
			t.Log("")
		})
	}
}

// ==================== B 方案架构风险分析 ====================

func TestBScheme_ArchitectureRisks(t *testing.T) {
	t.Log("========== B 方案架构风险与代价分析 ==========")
	t.Log("")

	risks := []struct {
		risk       string
		severity   string
		mitigation string
	}{
		{
			"首轮协议不完整风险",
			"中",
			"核心协议必须保留在首轮，只把详细说明、例子、tips 移到可查询",
		},
		{
			"agent 不知道该查什么",
			"高",
			"catalog 需要有高质量的预览和描述，让 agent 能判断是否需要查",
		},
		{
			"查询增加交互轮次，更慢",
			"中",
			"首轮更轻但多一轮查询；总耗时取决于查询次数和网络延迟",
		},
		{
			"质量下降（信息不足导致理解偏差）",
			"高",
			"需要真实 LLM 实验验证质量影响；可设置 fallback 阈值",
		},
		{
			"实现复杂度上升",
			"低",
			"ContextPack 已有 ToCapabilityMode/QueryBlock，改动不大",
		},
		{
			"调试困难（上下文分散在多轮）",
			"中",
			"需要在日志中记录查询历史，完整还原上下文",
		},
	}

	for i, r := range risks {
		severityColor := "🟢"
		if r.severity == "中" {
			severityColor = "🟡"
		} else if r.severity == "高" {
			severityColor = "🔴"
		}
		t.Logf("%d. %s %s [%s 风险]", i+1, severityColor, r.risk, r.severity)
		t.Logf("   缓解: %s", r.mitigation)
		t.Log("")
	}

	t.Log("=== 建议 ===")
	t.Log("1. 先做阶段二：真实 LLM 验证质量影响，这是最大的风险")
	t.Log("2. 执行协议拆分要保守：核心契约必留，只移方法论文档")
	t.Log("3. 分阶段上线：先法师（收益确定）→ 再剑士（需要验证）")
	t.Log("4. 可配置开关：按 quest 强度或用户选择决定是否启用能力优先")
}

// ==================== 辅助函数 ====================

func fmtPct(f float64) string {
	return fmt.Sprintf("%.1f%%", f)
}
