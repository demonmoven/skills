package orchestrator

import (
	"fmt"
	"strings"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/prompt"
)

// ==================== CLI 能力优先 AB 实验（修正版）====================
//
// 核心假设：平台预消化的那些上下文（diff、返工进度、验收标准……）
// agent 自己用 CLI 工具（git diff、grep、gloop quest show 等）就能查到，
// 平台反而多此一举，还可能摘要失真、信息过时、浪费 token。
//
// 对照组（Control）：全量 ContextPack — 平台把所有信息喂给 agent
// 实验组（Treatment）：极简上下文 + agent 用 CLI 自查询

// --- 信息来源分析 ---
//
// 先分析：当前 ContextPack 里每个 block 的信息来源是什么？
// agent 自己能不能查到？用什么工具查？

type contextBlockInfo struct {
	name       string
	sourceKind string // "user_input" | "platform_mechanism" | "cross_session" | "workspace_state" | "quest_metadata" | "protocol"
	selfServe  bool   // agent 自己能否查到
	cliCommand string // 用什么命令查
	outputSize string // 输出量级：small / medium / large
	notes      string
}

// 剑士侧 blocks 信息来源分析
var warriorBlocksAnalysis = []contextBlockInfo{
	{"quest_user_intent", "user_input", false, "-", "-", "用户原始输入，必须给"},
	{"intensity_note", "platform_mechanism", true, "gloop quest show", "small", "quest 元数据的一部分，CLI 能查"},
	{"design_quest_note", "protocol", true, "gloop skill show gloop-quest-execution", "small", "平台约定，skill 里有"},
	{"latest_review_blockers", "cross_session", false, "-", "-", "法师评审意见，跨 session，agent 查不到"},
	{"previous_round_summary", "workspace_state", true, "git log + 读文件", "medium", "agent 可以通过 git 历史 + 文件内容推断"},
	{"workspace_diff_snapshot", "workspace_state", true, "git diff", "medium-large", "agent 直接 git diff 就行"},
	{"rework_progress", "quest_metadata", true, "gloop quest show", "small", "第几次返工，CLI 能查"},
	{"acceptance_criteria", "quest_metadata", true, "gloop quest show", "small-medium", "验收标准，CLI 能查"},
	{"execution_instruction", "protocol", true, "gloop skill show gloop-quest-execution", "medium", "平台协议，skill 里有"},
}

// 法师侧 blocks 信息来源分析
var mageBlocksAnalysis = []contextBlockInfo{
	{"quest_user_intent", "user_input", false, "-", "-", "用户原始输入，必须给"},
	{"warrior_artifact", "cross_session", false, "-", "-", "剑士交付物，跨 session，必须给"},
	{"platform_tool_evidence", "platform_mechanism", false, "-", "small", "平台内部日志，agent 查不到"},
	{"acceptance_criteria", "quest_metadata", true, "gloop quest show", "small-medium", "CLI 能查"},
	{"review_history", "quest_metadata", true, "gloop quest show", "medium", "返工历史，CLI 能查"},
	{"review_evidence_pack", "platform_mechanism", true, "gloop quest show --evidence", "small", "平台证据包，CLI 能查"},
	{"review_requirements", "protocol", true, "gloop skill show gloop-quest-review", "small", "评审要求，skill 里有"},
	{"review_protocol", "protocol", true, "gloop skill show gloop-quest-review", "small", "评审协议，skill 里有"},
}

func TestCLICapability_BlockSourceAnalysis(t *testing.T) {
	t.Log("========== 上下文块信息来源分析 ==========")
	t.Log("")

	t.Log("--- 剑士侧 ---")
	t.Logf("%-25s %-15s %-8s %-20s", "Block 名称", "信息类型", "可自查", "查询方式")
	t.Log(strings.Repeat("-", 70))

	selfServeCount := 0
	mustProvideCount := 0
	for _, b := range warriorBlocksAnalysis {
		status := "✅ 可自查"
		if !b.selfServe {
			status = "❌ 必须给"
			mustProvideCount++
		} else {
			selfServeCount++
		}
		t.Logf("%-25s %-15s %-8s %-20s", b.name, b.sourceKind, status, b.cliCommand)
	}
	t.Logf("可自查: %d 个, 必须给: %d 个 (共 %d 个)",
		selfServeCount, mustProvideCount, len(warriorBlocksAnalysis))
	t.Log("")

	t.Log("--- 法师侧 ---")
	t.Logf("%-25s %-15s %-8s %-20s", "Block 名称", "信息类型", "可自查", "查询方式")
	t.Log(strings.Repeat("-", 70))

	selfServeCount = 0
	mustProvideCount = 0
	for _, b := range mageBlocksAnalysis {
		status := "✅ 可自查"
		if !b.selfServe {
			status = "❌ 必须给"
			mustProvideCount++
		} else {
			selfServeCount++
		}
		t.Logf("%-25s %-15s %-8s %-20s", b.name, b.sourceKind, status, b.cliCommand)
	}
	t.Logf("可自查: %d 个, 必须给: %d 个 (共 %d 个)",
		selfServeCount, mustProvideCount, len(mageBlocksAnalysis))
}

// --- 实验 1：Token 节省潜力 ---
//
// 如果把所有"可自查"的 block 都移除，首轮 prompt 能轻多少？
// 这是理论最大节省量（假设 agent 一个都不查）。

func TestCLICapability_MaxTokenSavings(t *testing.T) {
	scenarios := []struct {
		name        string
		side        string // "warrior" | "mage"
		quest       *fsstore.QuestMeta
		reworkHints string
		prevSummary string
		artifact    string
	}{
		{
			name: "剑士 · Round 0",
			side: "warrior",
			quest: &fsstore.QuestMeta{
				Query:       "实现用户认证模块，包含登录注册功能",
				Type:        model.QuestTypeExecute,
				Intensity:   model.QuestIntensityStandard,
				ReworkCount: 0,
				MaxRework:   3,
			},
		},
		{
			name: "剑士 · Round 2 返工（全量信息）",
			side: "warrior",
			quest: &fsstore.QuestMeta{
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
			},
			reworkHints: "密码哈希强度不够，使用 bcrypt；错误处理需要更细致。",
			prevSummary: "上一轮交付：SHA1 哈希密码，基础错误处理。",
		},
		{
			name: "法师 · Round 0 首次评审",
			side: "mage",
			quest: &fsstore.QuestMeta{
				Query:       "实现用户认证模块",
				Type:        model.QuestTypeExecute,
				Intensity:   model.QuestIntensityStandard,
				ReworkCount: 0,
				MaxRework:   3,
			},
			artifact: "已完成用户认证模块：\n1. 注册接口 POST /api/auth/register\n2. 登录接口 POST /api/auth/login\n3. JWT 鉴权中间件\n\n代码位于 auth.go 和 auth_test.go。",
		},
		{
			name: "法师 · Round 2 返工评审",
			side: "mage",
			quest: &fsstore.QuestMeta{
				Query:              "实现用户认证模块",
				Type:               model.QuestTypeExecute,
				Intensity:          model.QuestIntensityDeep,
				ReworkCount:        2,
				MaxRework:          3,
				ReviewHints:        "bcrypt cost 太低，建议 12；缺少刷新 token 机制。",
				AcceptanceCriteria: "1. 邮箱注册登录\n2. 密码 bcrypt 哈希\n3. JWT + refresh token\n4. 单元测试覆盖率>80%",
			},
			artifact: "已将 bcrypt cost 提升至 12，新增 refresh token 接口。测试覆盖率 85%。",
		},
	}

	t.Log("========== CLI 能力优先：首轮 Token 最大节省潜力 ==========")
	t.Log("（假设：所有可自查的 block 都移除，agent 一个都不查）")
	t.Log("")
	t.Logf("%-25s %-12s %-12s %-12s", "场景", "全量 chars", "极简 chars", "节省比例")
	t.Log(strings.Repeat("-", 65))

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			var fullPack prompt.ContextPack
			var coreBlocks []string // 必须保留的 block 名称

			if sc.side == "warrior" {
				fullPack = prompt.BuildQuestContextPack(sc.quest, sc.reworkHints, sc.prevSummary)
				// 必须给的：quest_user_intent, latest_review_blockers（跨 session）
				coreBlocks = []string{"quest_user_intent", "latest_review_blockers"}
			} else {
				fullPack = prompt.BuildReviewContextPackFromArtifact(
					sc.quest,
					fsstore.PhaseReviewArtifact{Assistant: sc.artifact},
					sc.prevSummary,
				)
				// 必须给的：quest_user_intent, warrior_artifact, platform_tool_evidence
				coreBlocks = []string{"quest_user_intent", "warrior_artifact", "platform_tool_evidence"}
			}

			fullChars := len(fullPack.RenderXMLish())

			minimalPack := fullPack.ToCapabilityMode(prompt.CapabilityModeConfig{
				CoreBlocks:       coreBlocks,
				CatalogBlockName: "available_cli_tools",
			})
			// 注意：这里用 ToCapabilityMode 来模拟"移除可自查 block"
			// 但 catalog 变成了"可用 CLI 工具列表"，而不是 block 目录
			minimalChars := len(minimalPack.RenderXMLish())

			savings := float64(fullChars-minimalChars) / float64(fullChars) * 100

			t.Logf("%-25s %-12d %-12d %-12.1f%%",
				sc.name, fullChars, minimalChars, savings)

			removed := fullPack.QueryableBlocks(coreBlocks)
			t.Logf("  移除的 blocks (%d): %v", len(removed), removed)
		})
	}
}

// --- 实验 2：实际总 Token 估算 ---
//
// 首轮省了，但 agent 要花 token 查命令。
// 模拟不同"勤奋程度"的 agent，估算总 token。

func TestCLICapability_TotalTokenEstimate(t *testing.T) {
	// 用 Round 2 返工场景（信息最多）做基准
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

	// 定义每个"可自查" block 的自查成本（命令 + 输出）
	// 注意：CLI 输出通常比平台预消化的更"原始"，信息量更大，token 更多
	queryCosts := map[string]int{
		"intensity_note":          200,  // gloop quest show → 小
		"previous_round_summary":  1500, // git log + 读几个文件 → 中
		"workspace_diff_snapshot": 2000, // git diff → 中大
		"rework_progress":         150,  // gloop quest show → 小
		"acceptance_criteria":     300,  // gloop quest show → 小
		"execution_instruction":   800,  // gloop skill show → 中
		// "design_quest_note" 算在 execution skill 里
	}

	// 不同 agent 画像
	agents := []struct {
		name        string
		queries     []string
		description string
	}{
		{
			"极简型",
			[]string{},
			"什么都不查，只看必须给的",
		},
		{
			"高效型",
			[]string{"workspace_diff_snapshot"},
			"只查 git diff，了解改动范围",
		},
		{
			"平衡型",
			[]string{"workspace_diff_snapshot", "acceptance_criteria"},
			"查 diff + 验收标准",
		},
		{
			"谨慎型",
			[]string{"workspace_diff_snapshot", "acceptance_criteria", "previous_round_summary", "rework_progress"},
			"把工作区相关的都查一遍",
		},
		{
			"全查型",
			[]string{"intensity_note", "workspace_diff_snapshot", "acceptance_criteria",
				"previous_round_summary", "rework_progress", "execution_instruction"},
			"所有能查的都查一遍",
		},
	}

	t.Log("========== CLI 能力优先：总 Token 估算（剑士 Round 2 返工）==========")
	t.Logf("全量上下文基准: %d chars", fullChars)
	t.Log("")
	t.Logf("%-10s %-8s %-12s %-12s %-10s",
		"Agent 类型", "查询数", "首轮 chars", "总 chars", "相对全量")
	t.Log(strings.Repeat("-", 55))

	// 极简首轮 = 必须给的 blocks
	coreBlocks := []string{"quest_user_intent", "latest_review_blockers"}
	// 重新计算：只保留核心 blocks 的渲染
	minimalChars := 0
	for _, name := range coreBlocks {
		if block, ok := fullPack.QueryBlock(name); ok {
			minimalChars += len(block.Content) + 100 // +标签开销
		}
	}
	minimalChars += 100 // gloop_context 标签开销

	for _, agent := range agents {
		t.Run(agent.name, func(t *testing.T) {
			queryCost := 0
			for _, q := range agent.queries {
				if cost, ok := queryCosts[q]; ok {
					queryCost += cost
				}
			}
			totalChars := minimalChars + queryCost
			ratio := float64(totalChars) / float64(fullChars) * 100

			symbol := "✅"
			if ratio > 100 {
				symbol = "⚠️"
			}

			t.Logf("%-10s %-8d %-12d %-12d %-10.1f%% %s",
				agent.name, len(agent.queries), minimalChars, totalChars, ratio, symbol)
		})
	}

	t.Log("")
	t.Log("💡 关键观察：")
	t.Log("  - 只要 agent 查了 git diff，总 token 就接近全量注入")
	t.Log("  - 因为 git diff 的原始输出比平台摘要大得多")
	t.Log("  - 但平台摘要是有损压缩，原始输出信息更完整")
}

// --- 实验 3：质量维度对比 ---
//
// Token 只是一方面，信息质量更重要。
// 对比"平台预消化"和"原始 CLI 输出"的信息质量。

func TestCLICapability_QualityComparison(t *testing.T) {
	t.Log("========== 信息质量维度对比 ==========")
	t.Log("")

	dimensions := []struct {
		dimension    string
		platformPre  string // 平台预消化的表现
		cliSelfServe string // CLI 自查的表现
		winner       string // "platform" | "cli" | "depends"
		notes        string
	}{
		{
			"信息准确性",
			"中（摘要可能失真）",
			"高（原始数据无损失）",
			"cli",
			"平台摘要可能出错，原始 CLI 输出是事实",
		},
		{
			"信息时效性",
			"低（首轮快照，可能过时）",
			"高（实时查询）",
			"cli",
			"执行过程中文件会变，平台快照是静态的",
		},
		{
			"信息密度",
			"高（精选摘要）",
			"低（原始输出有噪声）",
			"platform",
			"平台给的是精华，CLI 输出要自己筛",
		},
		{
			"理解成本",
			"低（结构清晰）",
			"高（需要解析原始输出）",
			"platform",
			"Agent 需要理解 CLI 输出格式，增加认知负担",
		},
		{
			"可追溯性",
			"低（来源不透明）",
			"高（命令可复现）",
			"cli",
			"CLI 命令明确，结果可验证",
		},
		{
			"协议一致性",
			"高（平台强制）",
			"中（agent 可能漏看协议）",
			"platform",
			"关键协议必须确保 agent 看到",
		},
		{
			"灵活度",
			"低（平台决定给什么）",
			"高（agent 按需查询）",
			"cli",
			"agent 可以深入查自己关心的细节",
		},
		{
			"首次响应速度",
			"快（信息都在首轮）",
			"慢（要先查再干活）",
			"platform",
			"CLI 查询增加额外轮次",
		},
	}

	t.Logf("%-15s %-20s %-20s %-8s",
		"维度", "平台预消化", "CLI 自查", "胜出")
	t.Log(strings.Repeat("-", 65))

	platformWins := 0
	cliWins := 0
	depends := 0

	for _, d := range dimensions {
		winner := d.winner
		if winner == "platform" {
			winner = "🏛️ 平台"
			platformWins++
		} else if winner == "cli" {
			winner = "💻 CLI"
			cliWins++
		} else {
			winner = "🤔 看情况"
			depends++
		}
		t.Logf("%-15s %-20s %-20s %-8s",
			d.dimension, d.platformPre, d.cliSelfServe, winner)
		t.Logf("  → %s", d.notes)
	}

	t.Log("")
	t.Logf("汇总: 平台 %d 胜, CLI %d 胜, 看情况 %d",
		platformWins, cliWins, depends)

	t.Log("")
	t.Log("=== 结论 ===")
	t.Log("  平台预消化的核心价值：信息密度高、理解成本低、协议有保障")
	t.Log("  CLI 自查的核心价值：信息准确实时、灵活可追溯")
	t.Log("  两者不是替代关系，是互补关系")
}

// --- 实验 4：混合模式设计 ---
//
// 最优解可能不是二选一，而是混合模式：
// - 必须给的：用户意图、跨 session 信息
// - 平台预消化的：关键协议、核心摘要（一句话版本）
// - 可深查的：详细信息通过 CLI 获取

func TestCLICapability_HybridMode(t *testing.T) {
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

	// 三种模式对比
	modes := []struct {
		name        string
		coreBlocks  []string
		description string
	}{
		{
			"全量注入（现状）",
			[]string{}, // 空 = 全量
			"所有 block 直接注入",
		},
		{
			"纯 CLI 能力优先",
			[]string{"quest_user_intent", "latest_review_blockers"},
			"只给必须的，其他全靠 CLI 查",
		},
		{
			"混合模式（推荐）",
			[]string{"quest_user_intent", "latest_review_blockers",
				"execution_instruction", "workspace_diff_snapshot"},
			"核心协议 + diff 概览直接给，细节靠 CLI 深查",
		},
	}

	t.Log("========== 混合模式设计 ==========")
	t.Logf("全量基准: %d chars", fullChars)
	t.Log("")

	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			var chars int
			var blockCount int

			if len(mode.coreBlocks) == 0 {
				// 全量
				chars = fullChars
				blockCount = len(fullPack.Blocks)
			} else {
				// 只保留核心 blocks
				corePack := fullPack.ToCapabilityMode(prompt.CapabilityModeConfig{
					CoreBlocks: mode.coreBlocks,
				})
				// 去掉 catalog 块，模拟"纯核心 + CLI 说明"
				// （实际实现中 catalog 会被替换为 "可用 CLI 工具" 提示）
				chars = len(corePack.RenderXMLish())
				blockCount = len(corePack.Blocks) - 1 // 减去 catalog
			}

			savings := float64(fullChars-chars) / float64(fullChars) * 100

			t.Logf("【%s】", mode.name)
			t.Logf("  描述: %s", mode.description)
			t.Logf("  Block 数: %d, 总 chars: %d (节省 %.1f%%)",
				blockCount, chars, savings)

			if len(mode.coreBlocks) > 0 {
				kept := strings.Join(mode.coreBlocks, ", ")
				t.Logf("  直接注入: %s", kept)

				queryable := fullPack.QueryableBlocks(mode.coreBlocks)
				if len(queryable) > 0 {
					t.Logf("  可深查: %v", queryable)
				}
			}
			t.Log("")
		})
	}
}

// ==================== 实验结论 ====================
//
// 运行：go test -v -run "TestCLICapability" ./internal/orchestrator/
//
// 核心结论：
//
// 1. 你的 insight 是对的——大部分信息 agent 确实能自己查
//    剑士侧：9 个 block 中 7 个可自查 (78%)
//    法师侧：8 个 block 中 5 个可自查 (63%)
//
// 2. 但 Token 账不是这么算的
//    - 平台预消化是"精选摘要"，密度高
//    - CLI 原始输出信息量大，查一个 git diff 就几乎等于全量注入
//    - 首轮确实轻了，但总 token 不一定省
//
// 3. 真正的价值不在省 token，而在：
//    - 信息更准确（原始数据 vs 摘要）
//    - 信息更实时（动态查询 vs 首轮快照）
//    - agent 可以按需深挖（平台不知道 agent 需要什么细节）
//
// 4. 推荐方向：混合模式
//    - 必须给：用户意图、跨 session 信息
//    - 轻量摘要直接给：diff 概览、核心协议一句话
//    - 详细信息靠 CLI 深查：完整 diff、历史记录、验收标准详情
//
// 5. 最大的风险：
//    - Agent 不知道该查什么 → 漏看关键信息
//    - 查询增加轮次 → 更慢
//    - 需要真实 LLM 实验验证质量影响

// helper
func blockInfo(b contextBlockInfo) string {
	return fmt.Sprintf("%s (%s)", b.name, b.sourceKind)
}
