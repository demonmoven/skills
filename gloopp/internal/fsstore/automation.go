package fsstore

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

const legacyContextRefreshQuery = "从本地工作区和飞书（群聊/文档/日程）提炼用户近期工作上下文，输出结构化摘要。用 context_write_dim 工具写入各维度（workspace/lark_im/lark_doc/lark_calendar），最后用 context_write_summary 写总览摘要。按信息量写短摘要，宁可短也不要注水；只留主题级摘要，不留原始聊天、群名、链接、精确日程、个人安排或可识别参会信息。"

const defaultContextRefreshQuery = `从本地工作区和飞书（群聊/文档/日程）提炼用户近期工作上下文，生成给后续 agent 使用的结构化摘要。

前置判断（节省 token）：
先读 context_show dim=activity_snapshot。如果过去 24h 内无任何 quest 创建/完成/失败，且飞书数据源无新消息（群聊无新内容、文档无更新、日程无变化），则直接 phase done 说明"数据源无变化，跳过本次刷新"，不要浪费 token 重写所有维度。只有确认有新活动时才继续以下流程。

目标不是写长总结，而是减少后续任务的无效提问、重复检索和旧坑复现。请用 context_write_dim 写入以下维度：

必写维度：
- identity — 用户身份、角色、团队归属、权限边界。从飞书通讯录/群聊角色/项目权限等推断。
- workspace — 本地 Git 仓库和项目文件的项目上下文。
- tooling — 用户可用的工具链、CLI、服务地址、部署环境与基础设施。从 workspace 配置文件和历史命令推断。
- lark_im — 从飞书聊天记录（群聊/私聊/话题群）提炼的近期讨论主题和决策。
  - Phase 0 实验：如 user 身份和飞书任务权限可用，可在 lark_im 的 Current Focus 下新增三级标题 "### Candidate Commitment Signals (Experimental)"，只从飞书任务提取最多 3 条候选承诺信号。
  - 命令：lark-cli task +get-my-tasks --as user --complete=false --page-all。若命令失败、无授权或任务数为 0，写入 Gaps 说明，不要编造信号。
  - 字段：Action 摘要、时间暗示、来源类型、置信度、不确定原因、核验关键词。来源类型固定为 lark_task，置信度可为 高/中高。
  - 边界：候选信号仅用于二次核验，不作为行动事实；不得创建 dim_commitment_signals 独立维度；不得写状态、优先级、task_guid、URL、open_id、群名、消息链接或原始标题全文。
  - 展示顺序仅用于可读性：时间暗示更近、置信度更高的信号靠前；不得写入 priority 字段或暗示已确认承诺。
- lark_doc — 从飞书云文档提炼的相关文档摘要。
- lark_calendar — 从飞书日历提炼的工作节奏、会议模式与协作规律。注意：不要存具体日程/参会人/时间，只存持久模式（如"每周一有XX会""偏好XX时段处理深度工作"）。

可选维度（有可靠信息时才写入）：
- gloop_history — 从 Gloop 历史委托中提炼的个人工作模式偏好（任务拆解风格、验证习惯、验收口径）。注意：不要写"上次做了什么"这种进展类信息，只写可复用的模式偏好。

最后用 context_write_summary 写总览摘要。

每个维度必须使用稳定 Markdown 结构（identity 和 tooling 维度可省略不适用的章节，如 Current Focus / Recent Decisions）：

# <维度标题>

## Scope
- 数据来源范围、时间新鲜度和主要覆盖对象。

## Current Focus
- 用户近期正在推进的项目、战役、风险或阻塞点。

## Stable Preferences
- 长期稳定的用户偏好、技术判断原则、沟通风格和验收口径。

## Project Contracts
- 与项目/仓库/服务相关的事实、边界、常用验证命令、API/前后端契约。

## Recent Decisions
- 近期明确拍板的结论。每条尽量写成「结论 - 来源类型 - 新鲜度 - 置信度」。

## Anti-Patterns
- 已被用户纠正或明确不希望重复发生的行为。

## Retrieval Index
- 需要细查时应去哪里查：仓库、命令、文档主题、群聊主题或上下文维度名。只写可行动线索，不写敏感原文。

## Gaps
- 本轮无法确认、可能过时、存在冲突或需要后续二次确认的信息。

语言要求：
- 章节标题必须保持上述英文名称，作为稳定 schema anchor。
- 正文默认使用中文撰写，方便中文用户和飞书场景直接消费。
- 技术名词、命令、文件路径、API、状态枚举、产品名可以保留英文。

刷新/沉淀语义：
- context_write_dim 和 context_write_summary 是覆盖写入，不是无限追加日志。
- identity / tooling 属于低频变动维度，除非有明确变化证据否则保持原样即可。
- Current Focus / Recent Decisions / lark_calendar 等近期事实按本轮时间窗口刷新，旧的短期事实不应长期保留。
- lark_calendar 只沉淀持久模式（工作节奏、会议规律），不保留具体日程和参会人。
- Stable Preferences / Project Contracts / Anti-Patterns / Retrieval Index 可以跨天保留，但必须根据本轮证据修正、合并或删除过时项。
- gloop_history 只写可复用的工作模式偏好，不写具体任务进展或结论。
- 需要长期沉淀的稳定规则应写成可复用原则；不要把每日聊天、日程和临时任务堆成历史流水账。

总览 summary.md 写 2-4 段，必须覆盖：Current Focus、Stable Preferences、Decision Rules、Freshness/Gaps。summary 只做路由入口，不要复制所有维度正文。

隐私和安全硬约束：
- 只留主题级摘要，不存原始聊天、群名、链接、精确日程、个人安排、可识别参会信息或凭证。
- 不把摘要当事实权威；对低置信度或可能过时的信息显式标注。
- 遇到多源冲突时保留冲突说明，不要强行合并成单一事实。
- 不修改项目代码，不执行外部副作用命令。`

const defaultMorningBriefingQuery = `基于 Gloop 用户上下文和近期委托历史，生成今日早间简报。

步骤：
1. 先读 context_show dim=activity_snapshot 获取当前活跃/阻塞/近期完成状态（避免重新遍历 quest 列表）
2. 用 context_summary 快速读取当前工作重心
3. 如 activity_snapshot 显示有失败/阻塞 quest，重点分析原因
4. 按优先级输出今日建议行动清单（3-5 条）

输出格式：
- 每条建议用一句话说明 WHY + WHAT，不要泛泛而谈
- 标注优先级（P0/P1/P2）和预估时间
- 如有阻塞点或风险明确标出
- 如果有失败的 quest，分析失败原因并建议是否 retry

约束：
- 不修改项目代码，不执行外部副作用命令
- 不重新做 context refresh（那是另一个 automation 的事）
- 只读取信息，输出建议`

const defaultStaleQuestCleanupQuery = `检查 Gloop 中是否存在僵尸/滞留委托，并输出清理建议。

检查规则：
1. status=running 且已运行超过 2 小时的 quest → 可能挂死
2. status=inbox 且创建超过 7 天未处理的 quest → 可能被遗忘
3. status=reviewing 且停留超过 24 小时的 quest → 可能需要人工介入
4. status=failed 且未被 retry 的 quest（最近 3 天内）→ 是否需要重试

输出格式：
- 按严重程度排序（挂死 > 遗忘 > 待介入 > 可重试）
- 每条给出 quest ID、query 摘要、建议动作（cancel/retry/remind）
- 如果没有异常，明确说"当前无僵尸委托"

约束：
- 只读取和分析，不要自动 cancel 或修改任何 quest
- 不修改项目代码，不执行外部副作用命令`

const defaultContextDecayAlertQuery = `检查 Gloop 用户上下文各维度的新鲜度，发现过期维度时发出预警。

步骤：
1. 先读 context_show dim=activity_snapshot 判断平台近期是否活跃（有 quest 创建/完成）
2. 用 context_list 获取所有维度的 UpdatedAt
3. 如果任何维度超过 3 天未更新 → 标记为 stale
4. 如果 summary 超过 3 天未更新 → 标记为 stale
5. 检查上次 context refresh automation 是否正常完成

判断逻辑：
- 如果平台本身不活跃（24h 内无 quest 活动），维度 stale 是正常的，降级为 info
- 如果平台活跃但某维度 stale，说明 refresh 可能失败了，优先级更高

输出格式：
- 列出每个维度的最后更新时间和新鲜度状态（fresh/stale/critical）
- 对 stale 维度说明可能原因（lark token 过期？数据源无变化？refresh 失败？）
- 给出修复建议（重新 auth、手动触发 refresh、检查 agent 日志等）
- 如果全部 fresh 或平台不活跃，说"上下文健康，无需操作"

约束：
- 只读取和诊断，不要自动触发 refresh
- 不修改项目代码，不执行外部副作用命令`

const defaultWorkspaceDriftQuery = `检查当前工作区的 Git 状态，发现漂移问题时报告。

检查项目：
1. 未提交改动超过 3 天（git status + file mtime）
2. 本地分支落后远程 main/master 超过 10 个 commit
3. go.mod / package.json 中是否有 replace 指令或本地 path 依赖未清理
4. 是否存在未 push 的 commit

输出格式：
- 按风险程度排序
- 每条说明问题 + 建议动作（commit/push/rebase/clean）
- 如果工作区干净，说"工作区状态健康"

约束：
- 只读取和分析，不要自动执行 git 操作
- 不修改项目代码`

const defaultCodebaseMapRefreshQuery = `扫描当前工作区的代码结构，生成代码地图（codebase map）上下文维度，供后续 agent 快速定位代码、减少重复扫描。

前置判断（节省 token）：
先读 context_show dim=codebase_map（如果存在），对比最近一次 git commit 的时间戳。如果自上次刷新后没有新的 commit 且改动文件数 < 3，则直接 phase done 说明"代码无显著变化，跳过刷新"。只有确认有新改动或首次生成时才继续。

目标不是写长文档，而是减少后续任务的无效扫代码成本。请用 context_write_dim 写入 codebase_map 维度。

维度内容使用以下稳定 Markdown 结构：

# 代码地图

## Scope
- 仓库路径、主要语言、框架、构建系统
- 最后更新时间、覆盖范围（是否包含子模块/第三方依赖）
- 本次扫描深度（顶层 / 核心模块 / 全量）

## Directory Tree
- 顶层目录结构树（2-3 层深度，.gloop/ 等系统目录和 node_modules/ 等依赖目录可省略）
- 每个目录的一句话职责说明

## Core Modules
- 核心模块/包列表，每个包含：
  - 模块名/路径
  - 一句话职责
  - 关键类型/接口/函数名（不是代码，是名字）
  - 与其他模块的关系

## Key Interfaces
- 对外暴露的核心接口、API、CLI 命令
- 入口文件路径 + 一句话用途

## Architecture Patterns
- 项目采用的主要架构模式（如 MVC、分层、事件驱动、插件化等）
- 数据流/调用链的关键路径

## Build & Test
- 构建命令、测试命令
- 关键配置文件位置（go.mod / package.json / Makefile 等）

## Knowledge Gaps
- 本次扫描未覆盖的区域
- 不确定或需要进一步探索的模块

语言要求：
- 章节标题保持英文作为稳定 schema anchor。
- 正文默认中文，技术名词、命令、文件路径、API 保持英文。

生成原则：
- 只存索引和摘要，不复制代码原文。
- 核心模块写深，边缘模块写浅；不重要的目录可以一笔带过。
- 优先给出"在哪里"和"干什么"，"具体怎么实现"留给 agent 自己读代码。
- context_write_dim 是覆盖写入，不是追加。

约束：
- 不修改项目代码，不执行外部副作用命令。
- 只读取文件结构和文件名，不做全量代码分析（节省 token）。
- 忽略 .gloopignore / .git / node_modules / vendor / dist / build / 等系统和产物目录。`

const defaultQuestPatternReportQuery = `统计过去一周的 Gloop 委托执行情况，生成模式分析报告。

统计维度：
1. 总 quest 数量、成功/失败/取消分布
2. 平均执行时长（按 intensity 分组）
3. 失败原因 Top 3（agent 超时？执行错误？review 不通过？）
4. 最常用的 agent/warrior 组合及其成功率
5. 自动化 vs 手动委托的比例

输出格式：
- 用表格或列表清晰展示
- 给出 1-2 条可行动的优化建议（如"某 agent 失败率高，建议切换"或"deep intensity 平均耗时过长，建议拆分"）
- 如果数据量太少（< 5 条），说明样本不足，建议下周再看

约束：
- 只读取和分析，不修改任何数据
- 不修改项目代码，不执行外部副作用命令`

const defaultInboxTriageQuery = `扫描所有需要用户处理的 Gloop 委托，进行分类并给出处理建议。

扫描范围：
1. **Inbox 待确认**：status=pending 且 source=automation 的委托（等待用户确认是否启动）
2. **待用户终审**：status=user_review 的委托（等待用户评审通过/返工/拒绝）
3. **执行中阻塞**：status=blocked 的委托（等待用户处理阻塞）
4. **Reviewing 超时**：status=reviewing 且停留超过 4 小时的委托（可能卡住了需要关注）

分类标准：
- 按优先级分组：高优（P0）/ 中优（P1）/ 普通（P2）
- P0：blocked 状态、高风险改动、超过 24 小时未处理的 user_review
- P1：普通 user_review、超过 12 小时未处理的 inbox 项
- P2：新的 inbox 项、低风险改动

输出格式：
- 总数统计：每类多少个，总数多少
- 按优先级排序的待办列表，每条包含：
  - quest ID / short_id
  - 类型（execute / design）
  - 一句话摘要（从 query 提炼）
  - 当前状态 + 停留时长
  - 建议动作（accept / review / resolve-blocked / 关注）
  - 风险级别（P0/P1/P2）
- 如果总数为 0，明确说"当前没有待处理委托"

通知策略：
- 如果有待处理项，调用 notify_user 发送飞书通知
- 通知标题："待处理委托：N 项等你处理"
- 通知正文：简要列出前 3-5 条最重要的，以及总数
- 如果没有待处理项，不发通知（避免打扰）
- 同一 automation 运行周期内最多发 1 条通知

约束：
- 只读取和分析，不要自动 accept/review/apply 任何委托
- 不修改项目代码，不执行外部副作用命令
- quest 状态变更必须由用户手动操作`

// ==================== Automation 配置（automation/*.json） ====================
//
// 每条 automation 定义一个「自动委托」：什么条件下、给谁、发什么委托。
// 支持手动触发和由 scheduler 按 cron 自动触发。

type AutomationTrigger string

const (
	TriggerManual   AutomationTrigger = "manual"   // 手动触发
	TriggerSchedule AutomationTrigger = "schedule" // 定时触发
	TriggerEvent    AutomationTrigger = "event"    // 事件触发（per-quest event log）
	TriggerGoal     AutomationTrigger = "goal"     // HOTL v0.2: goal 模式，持续运行直到条件满足
)

// AutomationSource 表示 automation 的来源。
// official = 官方预置，受保护（不可修改/删除），版本升级时会自动更新
// user = 用户自建，完全可控
type AutomationSource string

const (
	SourceOfficial AutomationSource = "official"
	SourceUser     AutomationSource = "user"
)

const CurrentOfficialAutomationPolicyVersion = 1

// FanOutSpec 描述一个并行 quest 的参数。
type FanOutSpec struct {
	Query            string   `json:"query"`
	WarriorID        string   `json:"warrior_id,omitempty"`
	MageID           string   `json:"mage_id,omitempty"`
	LeafID           string   `json:"leaf_id,omitempty"`
	OwnershipScopes  []string `json:"ownership_scopes,omitempty"`
	MergeStrategy    string   `json:"merge_strategy,omitempty"`
	MergeOwnerLeafID string   `json:"merge_owner_leaf_id,omitempty"`
}

type AutomationConfig struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Enabled         bool              `json:"enabled"`
	Source          AutomationSource  `json:"source"`                     // official | user
	Trigger         AutomationTrigger `json:"trigger"`                    // manual | schedule
	Cron            string            `json:"cron,omitempty"`             // schedule 模式的 cron 表达式
	Event           string            `json:"event,omitempty"`            // event 模式匹配的事件类型
	EventScope      string            `json:"event_scope,omitempty"`      // per_quest | global；空=per_quest
	EventAction     string            `json:"event_action,omitempty"`     // HOTL v0.2: note | spawn_quest；空=note
	SpawnAutomation string            `json:"spawn_automation,omitempty"` // HOTL v0.2: event_action=spawn_quest 时触发的 automation ID
	MaxIterations   int               `json:"max_iterations,omitempty"`   // HOTL v0.2: goal 模式最大迭代次数；空=10
	Connectors      []string          `json:"connectors,omitempty"`       // HOTL v0.2 环 4: quest 自主闭环后触发的 connector（如 "git"）；空=不触发
	Priority        int               `json:"priority,omitempty"`
	// 官方策略元信息。
	OfficialPolicyVersion     int  `json:"official_policy_version,omitempty"`
	RuntimePolicyUserOverride bool `json:"runtime_policy_user_override,omitempty"`

	// 委托参数
	Query              string       `json:"query"`                         // 委托内容
	QuestType          string       `json:"quest_type"`                    // execute | design
	Intensity          string       `json:"intensity,omitempty"`           // quick | standard | deep | adversarial；空=standard
	WorkingDir         string       `json:"working_dir"`                   // 工作目录
	WorkspaceMode      string       `json:"workspace_mode,omitempty"`      // auto/worktree/copy/readonly；空=跟随全局默认
	WarriorID          string       `json:"warrior_id"`                    // 指定剑士（空=自动选）
	MageID             string       `json:"mage_id"`                       // 指定法师（空=自动选）
	AutoStart          bool         `json:"auto_start"`                    // 创建后自动开始（true = 直接跑，不进 inbox）
	TriageMode         string       `json:"triage_mode,omitempty"`         // candidate | direct；auto_start=false 默认 candidate
	AutoApply          bool         `json:"auto_apply"`                    // 通过后自动 apply（慎用）
	AllowL2            bool         `json:"allow_l2,omitempty"`            // 允许 L2 外部副作用命令；与 auto_apply 互斥
	WithDesignPhase    bool         `json:"with_design_phase,omitempty"`   // 创建 quest 时启用可选设计阶段
	AutoSpawnExecute   bool         `json:"auto_spawn_execute,omitempty"`  // design 成功后自动创建执行委托
	TrustTier          string       `json:"trust_tier,omitempty"`          // tier_0..tier_3；空=未启用信任分级
	TrustTierLocked    bool         `json:"trust_tier_locked,omitempty"`   // 人工锁定后不自动升降级
	Tags               []string     `json:"tags,omitempty"`                // 功能分类标签：context/audit/hygiene/briefing/report/knowledge 等
	AcceptanceCriteria string       `json:"acceptance_criteria,omitempty"` // 验收标准
	FanOut             []FanOutSpec `json:"fan_out,omitempty"`             // 多 quest 并发规格

	// 元信息
	CreatedAtMs int64 `json:"created_at_ms"`
	UpdatedAtMs int64 `json:"updated_at_ms"`
	LastRunMs   int64 `json:"last_run_ms,omitempty"`
	RunCount    int   `json:"run_count"`
}

type AutomationDiscoveryArchive struct {
	ID           string `json:"id"`
	AutomationID string `json:"automation_id"`
	Outcome      string `json:"outcome"` // no_finding
	Reason       string `json:"reason,omitempty"`
	Summary      string `json:"summary,omitempty"`
	CreatedAtMs  int64  `json:"created_at_ms"`
}

// ==================== CRUD ====================

func (r *Root) ListAutomations() ([]*AutomationConfig, error) {
	dir := r.Sub(SubdirAutomation)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []*AutomationConfig
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		c, err := ReadJSON[AutomationConfig](filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		normalizeAutomationAfterLoad(c)
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority > out[j].Priority
		}
		if out[i].CreatedAtMs != out[j].CreatedAtMs {
			return out[i].CreatedAtMs > out[j].CreatedAtMs
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (r *Root) ListAutomationTemplates() ([]*AutomationConfig, error) {
	all, err := r.ListAutomations()
	if err != nil {
		return nil, err
	}
	out := []*AutomationConfig{}
	for _, a := range all {
		if a.IsOfficial() {
			out = append(out, a)
		}
	}
	return out, nil
}

// IsOfficial 返回该 automation 是否为官方预置。
func (c *AutomationConfig) IsOfficial() bool {
	return c.Source == SourceOfficial
}

// HasTag 检查是否包含指定标签。
func (c *AutomationConfig) HasTag(tag string) bool {
	for _, t := range c.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

func (r *Root) GetAutomation(id string) (*AutomationConfig, error) {
	if id == "" {
		return nil, fmt.Errorf("automation id 为空")
	}
	path := r.Sub(SubdirAutomation, id+".json")
	c, err := ReadJSON[AutomationConfig](path)
	if err != nil {
		return nil, fmt.Errorf("automation 不存在 %s: %w", id, err)
	}
	normalizeAutomationAfterLoad(c)
	return c, nil
}

func (r *Root) SaveAutomation(c *AutomationConfig) error {
	if c.ID == "" {
		return fmt.Errorf("automation id 为空")
	}
	if c.CreatedAtMs == 0 {
		c.CreatedAtMs = model.NowMs()
	}
	c.UpdatedAtMs = model.NowMs()
	if c.Trigger == "" {
		c.Trigger = TriggerManual
	}
	if c.QuestType == "" {
		c.QuestType = string(model.QuestTypeExecute)
	}
	c.Cron = NormalizeCronPreset(c.Cron)
	normalizeAutomationBeforeSave(c)
	return WriteJSON(r.Sub(SubdirAutomation, c.ID+".json"), c)
}

// normalizeAutomationAfterLoad 在读取 automation 配置后做兼容归一化：
// - 如果 source 为空，从 tags 推断（有 official tag → official，否则 user）
func normalizeAutomationAfterLoad(c *AutomationConfig) {
	if c.Source == "" {
		if c.HasTag("official") {
			c.Source = SourceOfficial
		} else {
			c.Source = SourceUser
		}
	}
}

// normalizeAutomationBeforeSave 在保存 automation 配置前做归一化和降级兼容：
// - source 为空时先从 tags 推断（兼容旧代码只设 tag 的情况），再默认 user
// - 降级兼容：source=official 时确保 tags 里有 official，老版本代码读了也能识别
func normalizeAutomationBeforeSave(c *AutomationConfig) {
	if c.Source == "" {
		if c.HasTag("official") {
			c.Source = SourceOfficial
		} else {
			c.Source = SourceUser
		}
	}
	// 降级兼容：同步 official tag
	if c.Source == SourceOfficial {
		if !c.HasTag("official") {
			c.Tags = append(c.Tags, "official")
		}
	}
}

// NormalizeCronPreset accepts frontend-friendly schedule presets at backend boundaries.
func NormalizeCronPreset(cron string) string {
	switch cron {
	case "daily":
		return "0 0 * * *"
	case "daily9":
		return "0 9 * * *"
	case "weekday":
		return "0 9 * * 1-5"
	case "weekly":
		return "0 0 * * 1"
	default:
		return cron
	}
}

func (r *Root) DeleteAutomation(id string) error {
	if id == "" {
		return fmt.Errorf("automation id 为空")
	}
	path := r.Sub(SubdirAutomation, id+".json")
	return os.Remove(path)
}

// MarkAutomationRun 更新最后运行时间和次数
func (r *Root) MarkAutomationRun(id string) error {
	c, err := r.GetAutomation(id)
	if err != nil {
		return err
	}
	c.LastRunMs = model.NowMs()
	c.RunCount++
	c.UpdatedAtMs = model.NowMs()
	return r.SaveAutomation(c)
}

// ==================== 默认值 ====================

// InitDefaultAutomations 确保官方预置 automation 存在。
func (r *Root) InitDefaultAutomations() ([]string, error) {
	defaults := []AutomationConfig{
		{
			ID:          "auto_context_refresh",
			Name:        "用户工作上下文刷新",
			Description: "从本地工作区和飞书提炼用户上下文摘要，供所有任务参考",
			Enabled:     true,
			Trigger:     TriggerSchedule,
			Cron:        "0 9 * * *",
			Priority:    100,
			Query:       defaultContextRefreshQuery,
			QuestType:   string(model.QuestTypeExecute),
			AutoStart:   true,
			AutoApply:   false,
			Tags:        []string{"official", "template", "context"},
		},
		{
			ID:          "auto_morning_briefing",
			Name:        "每日早间简报",
			Description: "综合上下文和近期委托，生成今日可行动建议清单",
			Enabled:     true,
			Trigger:     TriggerSchedule,
			Cron:        "30 9 * * 1-5",
			Priority:    80,
			Query:       defaultMorningBriefingQuery,
			QuestType:   string(model.QuestTypeDesign),
			AutoStart:   true,
			AutoApply:   false,
			Tags:        []string{"official", "template", "briefing"},
		},
		{
			ID:          "auto_stale_quest_cleanup",
			Name:        "僵尸委托清理",
			Description: "检测长时间运行或滞留的委托，提醒用户决策",
			Enabled:     true,
			Trigger:     TriggerSchedule,
			Cron:        "0 10 * * *",
			Priority:    70,
			Query:       defaultStaleQuestCleanupQuery,
			QuestType:   string(model.QuestTypeDesign),
			AutoStart:   true,
			AutoApply:   false,
			Tags:        []string{"official", "template", "hygiene"},
		},
		{
			ID:          "auto_context_decay_alert",
			Name:        "上下文过期预警",
			Description: "检查上下文维度是否有长时间未更新的情况",
			Enabled:     true,
			Trigger:     TriggerSchedule,
			Cron:        "0 11 * * *",
			Priority:    60,
			Query:       defaultContextDecayAlertQuery,
			QuestType:   string(model.QuestTypeDesign),
			AutoStart:   true,
			AutoApply:   false,
			Tags:        []string{"official", "template", "context", "hygiene"},
		},
		{
			ID:          "auto_codebase_map_refresh",
			Name:        "代码地图刷新",
			Description: "扫描工作区代码结构，生成代码地图上下文，减少后续任务重复扫代码",
			Enabled:     false,
			Trigger:     TriggerManual,
			Priority:    50,
			Query:       defaultCodebaseMapRefreshQuery,
			QuestType:   string(model.QuestTypeExecute),
			AutoStart:   false,
			AutoApply:   false,
			Tags:        []string{"official", "template", "context", "knowledge"},
		},
		{
			ID:          "auto_daily_lint",
			Name:        "每日代码检查",
			Description: "扫描项目中的 lint 错误和明显 bug",
			Enabled:     false,
			Trigger:     TriggerManual,
			Query:       "扫描这个项目中的常见代码问题、潜在 bug 和可改进点，输出结构化的检查报告，按严重程度分类。",
			QuestType:   string(model.QuestTypeExecute),
			AutoStart:   false, // 进 inbox 等用户确认
			AutoApply:   false,
			Tags:        []string{"official", "template"},
		},
		{
			ID:          "auto_code_review",
			Name:        "代码评审助手",
			Description: "对当前工作区的改动进行代码审查",
			Enabled:     false,
			Trigger:     TriggerManual,
			Query:       "审查当前工作区与主分支的差异，给出代码质量评审意见：正确性、可读性、性能、安全。不用太啰嗦，重点说问题。",
			QuestType:   string(model.QuestTypeDesign),
			AutoStart:   false,
			AutoApply:   false,
			Tags:        []string{"official", "template"},
		},
		{
			ID:          "auto_dependency_check",
			Name:        "依赖安全检查",
			Description: "检查项目依赖是否有已知漏洞",
			Enabled:     false,
			Trigger:     TriggerManual,
			Query:       "检查项目依赖的第三方库是否有已知安全漏洞，给出升级建议。",
			QuestType:   string(model.QuestTypeExecute),
			AutoStart:   false,
			AutoApply:   false,
			Tags:        []string{"official", "template"},
		},
		{
			ID:          "auto_workspace_drift_detect",
			Name:        "工作区漂移检测",
			Description: "检查工作区未提交改动、分支落后等漂移问题",
			Enabled:     false,
			Trigger:     TriggerManual,
			Query:       defaultWorkspaceDriftQuery,
			QuestType:   string(model.QuestTypeDesign),
			AutoStart:   false,
			AutoApply:   false,
			Tags:        []string{"official", "template", "hygiene"},
		},
		{
			ID:          "auto_audit_completed",
			Name:        "审计已完成委托",
			Description: "二次审查最近完成的委托",
			Enabled:     false,
			Trigger:     TriggerManual,
			Query:       "二次审查最近完成的 Gloop 委托，聚焦遗漏风险、验收质量和可复用经验，输出结构化审计报告。",
			QuestType:   string(model.QuestTypeDesign),
			AutoStart:   false,
			AutoApply:   false,
			Tags:        []string{"official", "template", "audit"},
		},
		{
			ID:          "auto_quest_pattern_report",
			Name:        "周度委托模式报告",
			Description: "统计过去一周委托的成功率、耗时和失败模式",
			Enabled:     false,
			Trigger:     TriggerManual,
			Query:       defaultQuestPatternReportQuery,
			QuestType:   string(model.QuestTypeDesign),
			AutoStart:   false,
			AutoApply:   false,
			Tags:        []string{"official", "template", "report"},
		},
		{
			ID:          "auto_inbox_triage",
			Name:        "收件箱智能分类",
			Description: "扫描待处理委托（inbox/user_review/blocked），按优先级分类并发送飞书通知",
			Enabled:     true,
			Trigger:     TriggerSchedule,
			Cron:        "30 9 * * 1-5",
			Priority:    75,
			Query:       defaultInboxTriageQuery,
			QuestType:   string(model.QuestTypeDesign),
			Intensity:   "quick",
			AutoStart:   true,
			AutoApply:   false,
			Tags:        []string{"official", "template", "triage", "hygiene"},
		},
		{
			ID:          "auto_knowledge_pack_refresh",
			Name:        "知识包刷新",
			Description: "整理 Gloop context 为可携带的 OKF-style knowledge bundle",
			Enabled:     false,
			Trigger:     TriggerManual,
			Priority:    90,
			Query:       "检查当前 Gloop 用户上下文摘要和维度内容，评估是否适合作为可携带 knowledge bundle 被其他 agent 消费。不要修改项目代码；必要时只使用 context_write_dim / context_write_summary 补充缺失的摘要结构、来源说明和可复用索引。重点保证知识短、可追溯、可读，避免写入原始聊天、精确日程、隐私信息或外部系统副作用。完成后说明可以用 `gloop context export --format okf --out <dir>` 导出。",
			QuestType:   string(model.QuestTypeDesign),
			AutoStart:   false,
			AutoApply:   false,
			AllowL2:     false,
			Tags:        []string{"official", "template", "context", "knowledge"},
		},
	}

	created := []string{}
	ts := model.NowMs()
	for _, a := range defaults {
		a.Source = SourceOfficial
		a.OfficialPolicyVersion = CurrentOfficialAutomationPolicyVersion
		relPath := "automation/" + a.ID + ".json"
		path := r.Sub(SubdirAutomation, a.ID+".json")
		if _, err := os.Stat(path); err == nil {
			if upgraded, upErr := r.upgradeOfficialAutomation(path, &a); upErr != nil {
				return created, upErr
			} else if upgraded {
				created = append(created, relPath)
			}
			continue
		} else if !os.IsNotExist(err) {
			return created, err
		}
		a.CreatedAtMs = ts
		a.UpdatedAtMs = ts
		if err := r.SaveAutomation(&a); err == nil {
			created = append(created, relPath)
		}
	}
	return created, nil
}

func (r *Root) upgradeOfficialAutomation(path string, def *AutomationConfig) (bool, error) {
	cfg, err := ReadJSON[AutomationConfig](path)
	if err != nil {
		return false, err
	}
	normalizeAutomationAfterLoad(cfg)
	if def == nil || cfg.ID != def.ID || !cfg.IsOfficial() {
		return false, nil
	}
	changed := false
	policyOutdated := cfg.OfficialPolicyVersion < CurrentOfficialAutomationPolicyVersion
	if cfg.ID == "auto_context_refresh" && shouldUpgradeContextRefreshQuery(cfg.Query) {
		cfg.Query = defaultContextRefreshQuery
		cfg.Description = "从本地工作区和飞书提炼结构化用户上下文，供所有任务参考"
		changed = true
	}
	if policyOutdated {
		if !cfg.RuntimePolicyUserOverride && shouldUpgradeOfficialAutomationRuntimePolicy(def) {
			if cfg.AutoStart != def.AutoStart {
				cfg.AutoStart = def.AutoStart
				changed = true
			}
			if cfg.AutoApply != def.AutoApply {
				cfg.AutoApply = def.AutoApply
				changed = true
			}
		}
		cfg.OfficialPolicyVersion = CurrentOfficialAutomationPolicyVersion
		changed = true
	}
	if !changed {
		return false, nil
	}
	if err := r.SaveAutomation(cfg); err != nil {
		return false, err
	}
	return true, nil
}

func shouldUpgradeOfficialAutomationRuntimePolicy(def *AutomationConfig) bool {
	return def != nil && def.Enabled && def.Trigger == TriggerSchedule && def.AutoStart && !def.AutoApply
}

func AutomationShouldAutoStartByOfficialPolicy(cfg *AutomationConfig) bool {
	return cfg != nil &&
		cfg.IsOfficial() &&
		cfg.OfficialPolicyVersion >= CurrentOfficialAutomationPolicyVersion &&
		!cfg.RuntimePolicyUserOverride &&
		cfg.Enabled &&
		cfg.Trigger == TriggerSchedule &&
		cfg.AutoStart &&
		!cfg.AutoApply
}

func AutomationTriageMode(cfg *AutomationConfig) string {
	if cfg == nil {
		return "candidate"
	}
	if cfg.TriageMode != "" {
		return cfg.TriageMode
	}
	if !cfg.AutoStart {
		return "candidate"
	}
	return "direct"
}

func (r *Root) ArchiveAutomationDiscovery(item *AutomationDiscoveryArchive) error {
	if item == nil {
		return fmt.Errorf("automation discovery archive 为空")
	}
	if item.AutomationID == "" {
		return fmt.Errorf("automation id 为空")
	}
	if item.ID == "" {
		item.ID = "disc_" + NewIDShort()
	}
	if item.Outcome == "" {
		item.Outcome = "no_finding"
	}
	if item.CreatedAtMs == 0 {
		item.CreatedAtMs = model.NowMs()
	}
	return AppendJSONL(r.Sub(SubdirAutomation, "discovery_archive.jsonl"), item)
}

func (r *Root) LoadAutomationDiscoveryArchive() ([]AutomationDiscoveryArchive, error) {
	items, err := ReadJSONL[AutomationDiscoveryArchive](r.Sub(SubdirAutomation, "discovery_archive.jsonl"), 0)
	if err != nil {
		if os.IsNotExist(err) {
			return []AutomationDiscoveryArchive{}, nil
		}
		return nil, err
	}
	return items, nil
}

func shouldUpgradeContextRefreshQuery(query string) bool {
	if query == legacyContextRefreshQuery {
		return true
	}
	if strings.Contains(query, "每个维度必须使用稳定 Markdown 结构") &&
		!strings.Contains(query, "正文默认使用中文撰写") {
		return true
	}
	if strings.Contains(query, "正文默认使用中文撰写") &&
		!strings.Contains(query, "覆盖写入，不是无限追加日志") {
		return true
	}
	if strings.Contains(query, "lark_im — 从飞书聊天记录") &&
		!strings.Contains(query, "Candidate Commitment Signals (Experimental)") {
		return true
	}
	return false
}

// ==================== Inbox ====================
//
// Inbox 不是独立实体，就是「等待用户确认的 quest」。
// 筛选条件：status = pending 且 created_by 以 "automation:" 开头。

// ListInbox 列出收件箱中的委托（待确认的 automation 委托）。
func (r *Root) ListInbox() ([]*QuestMeta, error) {
	qs := NewQuestStore(r)
	all, err := qs.ListQuests()
	if err != nil {
		return nil, err
	}
	var out []*QuestMeta
	for _, q := range all {
		if q.Status != model.QuestStatusPending {
			continue
		}
		if q.Source() == model.SourceAutomation {
			out = append(out, q)
		}
	}
	return out, nil
}
