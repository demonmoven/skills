package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

// ==================== Quest 状态 ====================

type QuestStatus string

const (
	QuestStatusPending      QuestStatus = "pending"       // 已创建，等待开始
	QuestStatusRunning      QuestStatus = "running"       // 剑士执行中
	QuestStatusReviewing    QuestStatus = "reviewing"     // 法师评审中
	QuestStatusWaitingInput QuestStatus = "waiting_input" // 等待用户回答具体问题
	QuestStatusUserReview   QuestStatus = "user_review"   // 用户终审中
	QuestStatusSuccess      QuestStatus = "success"       // 成功完成
	QuestStatusFailed       QuestStatus = "failed"        // 失败
	QuestStatusCancelled    QuestStatus = "cancelled"     // 已取消
	QuestStatusBlocked      QuestStatus = "blocked"       // 阻塞（超预算、agent 失败等）
)

// terminalQuestStatuses 列出终态（不能再迁移出去）。
var terminalQuestStatuses = map[QuestStatus]bool{
	QuestStatusSuccess:   true,
	QuestStatusFailed:    true,
	QuestStatusCancelled: true,
}

// IsTerminal 返回该状态是否为终态。
func (qs QuestStatus) IsTerminal() bool { return terminalQuestStatuses[qs] }

// ==================== Quest 来源 ====================

type QuestSource string

const (
	SourceUser       QuestSource = "user"
	SourceAutomation QuestSource = "automation"

	QuestSourcePrefix = "automation:"
)

// ==================== Phase 状态 ====================

type PhaseStatus string

const (
	PhasePending PhaseStatus = "pending"
	PhaseRunning PhaseStatus = "running"
	PhaseDone    PhaseStatus = "done"
	PhaseFailed  PhaseStatus = "failed"
)

type QuestType string

const (
	QuestTypeExecute QuestType = "execute" // 执行型：直接动手做
	QuestTypeDesign  QuestType = "design"  // 方案型：出方案、做设计
)

// QuestMode 用户面向的委托模式（v0.3.6+）。
// 内部映射到 QuestType + Pipeline 选择。
type QuestMode string

const (
	ModeRun    QuestMode = "run"    // 直接执行，无 review
	ModeCheck  QuestMode = "check"  // 执行 + review
	ModeDesign QuestMode = "design" // 设计 + 评审 + 执行 + 评审
)

// WorkflowMode 是 v0.5 面向用户的委托工作流模式。
//
// 旧版 QuestMode 是表单/API 的输入枚举（run/check/design），WorkflowMode
// 是 thread ledger 的持久语义（Direct/Checked/Goal）。
type WorkflowMode string

const (
	WorkflowModeDirect  WorkflowMode = "direct"
	WorkflowModeChecked WorkflowMode = "checked"
	WorkflowModeGoal    WorkflowMode = "goal"
)

// QuestModeToWorkflowMode 将旧表单模式映射到 v0.5 workflow 模式。
func QuestModeToWorkflowMode(m QuestMode) WorkflowMode {
	switch m {
	case ModeRun:
		return WorkflowModeDirect
	case ModeDesign:
		return WorkflowModeGoal
	default:
		return WorkflowModeChecked
	}
}

// WorkflowModeFromQuest 将存储字段推导为 v0.5 workflow 模式。
func WorkflowModeFromQuest(questType QuestType, skipReview bool) WorkflowMode {
	if questType == QuestTypeDesign {
		return WorkflowModeGoal
	}
	if skipReview {
		return WorkflowModeDirect
	}
	return WorkflowModeChecked
}

// WorkflowModeRank defines the one-way upgrade order: direct < checked < goal.
func WorkflowModeRank(m WorkflowMode) int {
	switch m {
	case WorkflowModeDirect:
		return 1
	case WorkflowModeChecked:
		return 2
	case WorkflowModeGoal:
		return 3
	default:
		return 0
	}
}

func IsValidWorkflowMode(m WorkflowMode) bool {
	return WorkflowModeRank(m) > 0
}

// ModeToQuestType 将用户 Mode 映射到内部 QuestType。
func ModeToQuestType(m QuestMode) QuestType {
	if m == ModeDesign {
		return QuestTypeDesign
	}
	return QuestTypeExecute
}

// ModeSkipReview 返回该模式是否跳过 review 阶段。
func ModeSkipReview(m QuestMode) bool {
	return m == ModeRun
}

type QuestIntensity string

const (
	QuestIntensityQuick       QuestIntensity = "quick"       // 快速：小任务、低预算
	QuestIntensityStandard    QuestIntensity = "standard"    // 标准：默认通用任务预算
	QuestIntensityDeep        QuestIntensity = "deep"        // 深入：复杂任务，更多执行预算
	QuestIntensityAdversarial QuestIntensity = "adversarial" // 严审：高风险/高价值任务，更多验证预算
)

type QuestVerdict string

const (
	VerdictPass          QuestVerdict = "pass"
	VerdictRequestChange QuestVerdict = "request_changes"
	VerdictReject        QuestVerdict = "reject"
)

type ApplyStatus string

const (
	ApplyStatusNone    ApplyStatus = ""
	ApplyStatusPending ApplyStatus = "pending"
	ApplyStatusApplied ApplyStatus = "applied"
	ApplyStatusFailed  ApplyStatus = "failed"
)

// ==================== 职业 ====================

type AdventurerClass string

const (
	ClassWarrior AdventurerClass = "warrior" // 剑士：生产者
	ClassMage    AdventurerClass = "mage"    // 法师：评审者
)

// PostAuthorRole 是 v0.5 Post ledger 的作者角色。UI 只有一种 Post/Reply，
// 但权限由 role + artifact_type 共同约束。
type PostAuthorRole string

const (
	PostRoleMaker      PostAuthorRole = "maker"
	PostRoleChecker    PostAuthorRole = "checker"
	PostRoleSystem     PostAuthorRole = "system"
	PostRoleHuman      PostAuthorRole = "human"
	PostRoleAutomation PostAuthorRole = "automation"
)

// ArtifactType 是 Post 附带 artifact 的权限账本类型。
type ArtifactType string

const (
	ArtifactTypeDelivery             ArtifactType = "DeliveryArtifact"
	ArtifactTypeMakerReport          ArtifactType = "MakerReport"
	ArtifactTypeReviewReport         ArtifactType = "ReviewReport"
	ArtifactTypeEvidence             ArtifactType = "Evidence"
	ArtifactTypeVerificationArtifact ArtifactType = "VerificationArtifact"
	ArtifactTypeDecisionNote         ArtifactType = "DecisionNote"
)

type AdventurerStatus string

const (
	AdventurerActive       AdventurerStatus = "active"
	AdventurerPendingSetup AdventurerStatus = "pending_setup" // 待用户确认配置后激活
	AdventurerRetired      AdventurerStatus = "retired"
)

// ==================== Agent ====================

type AgentType string

const (
	AgentTypeACP  AgentType = "acp"  // ACP 协议接入
	AgentTypeCLI  AgentType = "cli"  // CLI 命令行接入
	AgentTypeMock AgentType = "mock" // 测试用 mock 执行器
)

// ==================== 工作区 ====================

type WorkspaceMode string

const (
	WorkspaceAuto     WorkspaceMode = "auto"     // 自动探测（git worktree 存在就用 worktree，否则 copy）
	WorkspaceWorktree WorkspaceMode = "worktree" // git worktree 隔离
	WorkspaceCopy     WorkspaceMode = "copy"     // 目录拷贝隔离
	WorkspaceReadOnly WorkspaceMode = "readonly" // 只读，直接用原目录
)

// ==================== 影响声明 ====================

// ImpactSummary 是 agent 在 quest 收尾时主动声明的影响。
type ImpactSummary struct {
	WhatChanged string   `json:"what_changed,omitempty"`
	Affected    []string `json:"affected,omitempty"`
	NotTouched  []string `json:"not_touched,omitempty"`
	Caveats     []string `json:"caveats,omitempty"`
}

// IsEmpty 报告 ImpactSummary 是否完全空。
func (i ImpactSummary) IsEmpty() bool {
	return i.WhatChanged == "" && len(i.Affected) == 0 && len(i.NotTouched) == 0 && len(i.Caveats) == 0
}

// ==================== 事件类型 ====================

type EventType string

const (
	// Quest 生命周期
	EvtQuestCreated    EventType = "quest.created"
	EvtQuestStarted    EventType = "quest.started"
	EvtPhaseChanged    EventType = "quest.phase_changed"
	EvtReviewSubmitted EventType = "quest.review_submitted"
	EvtRework          EventType = "quest.rework"
	EvtUserReview      EventType = "quest.user_review"
	EvtQuestSuccess    EventType = "quest.success"
	EvtQuestFailed     EventType = "quest.failed"
	EvtQuestCancelled  EventType = "quest.cancelled"
	EvtQuestApplied    EventType = "quest.applied"
	EvtQuestDiscarded  EventType = "quest.discarded"
	EvtQuestNote       EventType = "quest.note"

	// 工作区
	EvtWorkspaceCreated   EventType = "workspace.created"
	EvtWorkspaceDiffReady EventType = "workspace.diff_ready"

	// Micro
	EvtMicroTurn    EventType = "micro.turn"
	EvtTokenDelta   EventType = "micro.token_delta"
	EvtAssistantMsg EventType = "micro.assistant_msg"
	EvtToolStart    EventType = "micro.tool_start"
	EvtToolEnd      EventType = "micro.tool_end"
)

// ==================== 时间工具 ====================

func NowMs() int64 { return time.Now().UnixMilli() }

// ==================== 提问质量校验 ====================

// lowQualityQuestionPatterns 是已知的劣质提问占位符。
// agent 偶尔会把"提问工具"当万能退路，question 只填占位符（如 "need input"），
// 用户看到后完全不知道被问了什么。命中即拒绝，逼 agent 写清真正的问题。
var lowQualityQuestionPatterns = []string{
	"need input",
	"need more input",
	"input needed",
	"input required",
	"require input",
	"waiting for input",
	"ask user",
	"ask the user",
	"user input",
	"需要输入",
	"需要用户输入",
	"需要用户回答",
	"等待输入",
	"等待用户输入",
	"问用户",
	"询问用户",
	"待输入",
	"提问",
	"?",
	"??",
	"???",
	"？？？",
}

// ValidateQuestionText 校验 agent 向用户提问的文本质量。
// 返回的 error 非空时，调用方应拒绝该提问并把错误反馈给 agent。
//
// 规则（窄而稳，避免误伤）：
//  1. trim 后为空 → 拒绝
//  2. trim 后命中占位符黑名单（大小写不敏感）→ 拒绝
//  3. trim 后长度 < 6 且不含问号 → 拒绝（过短且非疑问，几乎不可能是有效问题）
func ValidateQuestionText(question string) error {
	q := strings.TrimSpace(question)
	if q == "" {
		return errQuestionEmpty
	}
	lower := strings.ToLower(q)
	for _, p := range lowQualityQuestionPatterns {
		if lower == p {
			return errQuestionPlaceholder
		}
	}
	// 用 rune 计数避免中文按字节计长导致误判（"继续"=2 rune 但 6 byte）。
	if utf8.RuneCountInString(q) < 6 && !strings.ContainsAny(q, "?？") {
		return errQuestionTooShort
	}
	return nil
}

var (
	errQuestionEmpty       = errQuestion("question 不能为空")
	errQuestionPlaceholder = errQuestion("question 内容无信息量（疑似占位符），请写清楚到底要问用户什么")
	errQuestionTooShort    = errQuestion("question 过短且非疑问句，请写清楚问题背景和具体要问的内容")
)

type errQuestion string

func (e errQuestion) Error() string { return string(e) }
