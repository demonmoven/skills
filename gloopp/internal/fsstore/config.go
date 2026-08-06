package fsstore

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/version"
)

// ==================== 全局配置（config.json） ====================

type GlobalConfig struct {
	Version                 string `json:"version"`
	Host                    string `json:"host"`
	Port                    int    `json:"port"`
	MaxTurnsPerPhase        int    `json:"max_turns_per_phase"` // 单阶段最大回合数（技术安全网，不作为主预算）
	MaxReworkPerQuest       int    `json:"max_rework_per_quest"`
	MaxConcurrent           int    `json:"max_concurrent"` // 最大并发 quest 数，0 = 不限
	DefaultModel            string `json:"default_model"`
	DefaultWorkingDir       string `json:"default_working_dir"`
	DefaultWorkspaceMode    string `json:"default_workspace_mode"` // auto/worktree/copy/readonly；空或 auto = 自动探测
	DefaultWarriorID        string `json:"default_warrior_id"`     // 默认剑士 ID；创建 quest 时未指定则用此值
	DefaultMageID           string `json:"default_mage_id"`        // 默认法师 ID；创建 quest 时未指定则用此值
	DefaultExecuteAgentID   string `json:"default_execute_agent_id,omitempty"`
	DefaultReviewAgentID    string `json:"default_review_agent_id,omitempty"`
	ContextFileMode         bool   `json:"context_file_mode"` // 文件式上下文模式；默认关闭，保留环境变量覆盖
	WorkspaceRetentionDays  int    `json:"workspace_retention_days"`
	AutoCleanupFailedQuests bool   `json:"auto_cleanup_failed_quests"`

	// Apply 策略（仅 worktree 模式生效）：
	//   - patch            ：默认，用 git diff 生成 patch 后 git apply --index，快且精确
	//   - patch_then_merge ：patch 冲突时 fallback 到 git merge，成功率更高但可能带入多余提交
	//   - merge            ：直接用 git merge --no-ff，适合 base 前进较多的场景
	ApplyStrategy string `json:"apply_strategy"`

	// 预算控制（0 = 不限）。Token 字段仅用于记录兼容旧配置，不作为平台停止条件。
	MaxTokensPerPhase     int64 `json:"max_tokens_per_phase"`      // Deprecated: token 是 agent/agent 内部预算，平台不据此停止
	MaxDurationPerQuestMs int64 `json:"max_duration_per_quest_ms"` // 单 quest 总时长上限（毫秒）
	MaxTokensPerQuest     int64 `json:"max_tokens_per_quest"`      // Deprecated: token 是 agent/agent 内部预算，平台不据此停止
	UserConfirmTimeoutMs  int64 `json:"user_confirm_timeout_ms"`   // 等待用户终审超时（毫秒）

	// 评审提示：法师忘调 review_quest 工具时补提示的次数
	// 执行提示：剑士忘调 phase_checkpoint 工具时补提示的次数
	// Agent 连续返回 executor error 时的阶段阻塞阈值。
	MaxConsecutiveAgentErrors int `json:"max_consecutive_agent_errors"`
	// 连续无进展（无工具调用且无新增文本）轮数阈值，超过则阻塞阶段。0 = 不检测。
	MaxNoProgressTurns int `json:"max_no_progress_turns"`
	// 恢复重试退避冷却的封顶秒数。0 = 不封顶，用 policy 的指数退避；
	// >0 时实际冷却 = min(policy 冷却, 此值)。供低延迟环境/测试收窄退避。
	RecoveryCooldownMaxSeconds int `json:"recovery_cooldown_max_seconds"`

	// 阶段信号监听（CLI syscall 触发 phase-end 的实时响应）
	SignalPollIntervalMs int64 `json:"signal_poll_interval_ms"` // 轮询间隔（毫秒），默认 500
	SignalGracePeriodMs  int64 `json:"signal_grace_period_ms"`  // 检测到 phase-end 信号后的宽限期（毫秒），默认 3000
	RuntimeIdleWarningMs int64 `json:"runtime_idle_warning_ms"` // AgentSession 无输出/信号多久后发 idle warning；0=关闭

	// Mage 可审计验证命令白名单。法师不能拿通用 Shell，只能通过平台执行这里显式允许的命令。
	CommandAllowlist []AllowedCommand `json:"mage_command_allowlist"`

	// HOTL v0.2 环 4：Connectors 配置。quest 自主闭环后把影响落到外部系统。
	// 默认全 OFF，需显式启用。详见 internal/connectors。
	Connectors ConnectorsConfig `json:"connectors"`

	// HOTL 自主闭环开关。nil 表示老配置缺字段，加载时迁移为默认 true；
	// 显式 false 用于 doctor / 测试验证人工 user_review 链路。
	HotlAutoClose *bool `json:"hotl_auto_close,omitempty"`
}

// ConnectorsConfig 汇总所有 connector 的配置。
type ConnectorsConfig struct {
	Git GitConnectorConfig `json:"git"`
}

// GitConnectorConfig 是 GitConnector 的持久化配置。
// 默认 Enabled=false：即使 automation 声明了 connectors:["git"]，也不会真的提交/推送，
// 必须在这里把 Enabled 设 true 才生效。
type GitConnectorConfig struct {
	Enabled     bool   `json:"enabled"`
	Remote      string `json:"remote"`      // 空=origin
	BaseBranch  string `json:"base_branch"` // 空=main；只读参考，永不推送
	AuthorName  string `json:"author_name"` // 空=用仓库默认
	AuthorEmail string `json:"author_email"`
	Push        bool   `json:"push"` // false=只本地 commit，不 push
}

func DefaultGlobalConfig() *GlobalConfig {
	hotlAutoClose := true
	return &GlobalConfig{
		Version:                   version.Version,
		Host:                      version.DefaultHost,
		Port:                      version.DefaultPort,
		MaxTurnsPerPhase:          50, // 技术安全网，主预算由 guardrails（时长/无进展/连续错误）决定
		MaxReworkPerQuest:         3,
		MaxConcurrent:             6,
		DefaultModel:              "auto",
		DefaultWorkspaceMode:      "auto",
		WorkspaceRetentionDays:    7,
		AutoCleanupFailedQuests:   false,
		ApplyStrategy:             ApplyStrategyPatch,
		MaxTokensPerPhase:         0,                   // 默认不限 token
		MaxDurationPerQuestMs:     24 * 60 * 60 * 1000, // 24 小时（wall-clock 硬兜底）
		MaxTokensPerQuest:         0,                   // 默认不限总 token
		UserConfirmTimeoutMs:      72 * 60 * 60 * 1000,
		MaxConsecutiveAgentErrors: 3,
		MaxNoProgressTurns:        3,
		SignalPollIntervalMs:      500,
		SignalGracePeriodMs:       3000,
		RuntimeIdleWarningMs:      60 * 1000,
		CommandAllowlist:          DefaultCommandAllowlist(),
		HotlAutoClose:             &hotlAutoClose,
	}
}

func (r *Root) LoadConfig() (*GlobalConfig, error) {
	p := r.Sub(FileConfig)
	c, err := ReadJSON[GlobalConfig](p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			def := DefaultGlobalConfig()
			if err := WriteJSON(p, def); err != nil {
				return nil, err
			}
			return def, nil
		}
		return nil, err
	}
	// 补默认值：老版本配置缺字段时不 panic
	def := DefaultGlobalConfig()
	applyDefaults(c, def)
	return c, nil
}

// applyDefaults 将 def 中的值填充到 dst，仅当 dst 字段为零值时才覆盖。
// 不包含 DefaultWorkingDir（用户手写字段）和 MaxTokensPer(Phase|Quest)（旧字段仅观测兼容，零值有效）。
func applyDefaults(dst, def *GlobalConfig) {
	if dst.Version == "" {
		dst.Version = def.Version
	}
	if dst.Host == "" {
		dst.Host = def.Host
	}
	if dst.Port == 0 {
		dst.Port = def.Port
	}
	if dst.MaxTurnsPerPhase == 0 {
		dst.MaxTurnsPerPhase = def.MaxTurnsPerPhase
	}
	if dst.MaxReworkPerQuest == 0 {
		dst.MaxReworkPerQuest = def.MaxReworkPerQuest
	}
	if dst.MaxConcurrent == 0 {
		dst.MaxConcurrent = def.MaxConcurrent
	}
	if dst.DefaultModel == "" {
		dst.DefaultModel = def.DefaultModel
	}
	if dst.DefaultWorkspaceMode == "" {
		dst.DefaultWorkspaceMode = def.DefaultWorkspaceMode
	}
	if dst.WorkspaceRetentionDays == 0 {
		dst.WorkspaceRetentionDays = def.WorkspaceRetentionDays
	}
	if dst.MaxConsecutiveAgentErrors == 0 {
		dst.MaxConsecutiveAgentErrors = def.MaxConsecutiveAgentErrors
	}
	if dst.MaxNoProgressTurns == 0 {
		dst.MaxNoProgressTurns = def.MaxNoProgressTurns
	}
	if dst.SignalPollIntervalMs == 0 {
		dst.SignalPollIntervalMs = def.SignalPollIntervalMs
	}
	if dst.SignalGracePeriodMs == 0 {
		dst.SignalGracePeriodMs = def.SignalGracePeriodMs
	}
	if dst.RuntimeIdleWarningMs == 0 {
		dst.RuntimeIdleWarningMs = def.RuntimeIdleWarningMs
	}
	if dst.CommandAllowlist == nil {
		dst.CommandAllowlist = def.CommandAllowlist
	} else if len(dst.CommandAllowlist) > 0 {
		// 用户配置非空时，按 ID 与默认白名单合并：
		// - 用户配置的同 ID 条目覆盖默认参数
		// - 默认中存在但用户没配置的条目，自动补入
		// 这样平台升级新增默认命令时，老用户也能自动获得，
		// 同时保留用户自定义条目的优先级。
		merged := make([]AllowedCommand, 0, len(dst.CommandAllowlist)+len(def.CommandAllowlist))
		seen := map[string]bool{}
		for _, item := range dst.CommandAllowlist {
			item = item.withDefaults()
			merged = append(merged, item)
			seen[item.ID] = true
		}
		for _, item := range def.CommandAllowlist {
			if !seen[item.ID] {
				merged = append(merged, item)
				seen[item.ID] = true
			}
		}
		dst.CommandAllowlist = merged
	}
	if dst.MaxDurationPerQuestMs == 0 {
		dst.MaxDurationPerQuestMs = def.MaxDurationPerQuestMs
	}
	if dst.UserConfirmTimeoutMs == 0 {
		dst.UserConfirmTimeoutMs = def.UserConfirmTimeoutMs
	}
	// HOTL v0.2 迁移：老 config 没有 hotl_auto_close 字段。用 *bool 区分
	// "显式 false" 和 "缺失"，缺失时迁移为生产默认 true。
	if dst.HotlAutoClose == nil {
		v := true
		dst.HotlAutoClose = &v
	}

	// v0.2.12 预算迁移：早期 config 把 max_turns_per_phase / max_duration_per_phase_ms
	// 设得过紧（6 回合 / 10 分钟），导致稍复杂的 phase 被频繁切断 failed。
	// 对 < 0.2.12 的旧配置，把低于合理下限的值向上提升。
	if semverLess(dst.Version, "0.2.12") {
		if dst.MaxTurnsPerPhase > 0 && dst.MaxTurnsPerPhase < 20 {
			dst.MaxTurnsPerPhase = 20
		}
		// 早期 config 把 max_consecutive_agent_errors 设为 2，对 relay/aiden CLI
		// 偶发的 transient 错误过于敏感。提升到 3 容忍一次抖动。
		if dst.MaxConsecutiveAgentErrors > 0 && dst.MaxConsecutiveAgentErrors < 3 {
			dst.MaxConsecutiveAgentErrors = 3
		}
	}
}

// isOldConfig 判断 config version 是否早于 v0.2.0（HOTL 引入版本）。
func isOldConfig(v string) bool {
	return v != "" && semverLess(v, "0.2.0")
}

// semverLess 判断 a < b（简化版 semver 比较，只支持 X.Y.Z）。
func semverLess(a, b string) bool {
	var aMaj, aMin, aPat, bMaj, bMin, bPat int
	fmt.Sscanf(a, "%d.%d.%d", &aMaj, &aMin, &aPat)
	fmt.Sscanf(b, "%d.%d.%d", &bMaj, &bMin, &bPat)
	if aMaj != bMaj {
		return aMaj < bMaj
	}
	if aMin != bMin {
		return aMin < bMin
	}
	return aPat < bPat
}

func (r *Root) SaveConfig(c *GlobalConfig) error {
	return WriteJSON(r.Sub(FileConfig), c)
}

// NewIDShort 生成一个 8 位十六进制短 ID（给前端显示用）。
func NewIDShort() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Validate 对 GlobalConfig 做基本合法性校验，0 值字段视为"不限"而不是错误。
func (c *GlobalConfig) Validate() error {
	if c == nil {
		return errors.New("config is nil")
	}
	if c.MaxConcurrent < 0 {
		return fmt.Errorf("max_concurrent 不能为负 (%d)", c.MaxConcurrent)
	}
	if c.MaxTurnsPerPhase < 0 {
		return fmt.Errorf("max_turns_per_phase 不能为负 (%d)", c.MaxTurnsPerPhase)
	}
	if c.MaxReworkPerQuest < 0 {
		return fmt.Errorf("max_rework_per_quest 不能为负 (%d)", c.MaxReworkPerQuest)
	}
	if c.WorkspaceRetentionDays < 0 {
		return fmt.Errorf("workspace_retention_days 不能为负 (%d)", c.WorkspaceRetentionDays)
	}
	if c.MaxDurationPerQuestMs < 0 {
		return fmt.Errorf("max_duration_per_quest_ms 不能为负 (%d)", c.MaxDurationPerQuestMs)
	}
	if c.MaxConsecutiveAgentErrors < 0 {
		return fmt.Errorf("max_consecutive_agent_errors 不能为负 (%d)", c.MaxConsecutiveAgentErrors)
	}
	if c.MaxNoProgressTurns < 0 {
		return fmt.Errorf("max_no_progress_turns 不能为负 (%d)", c.MaxNoProgressTurns)
	}
	if c.SignalPollIntervalMs < 0 {
		return fmt.Errorf("signal_poll_interval_ms 不能为负 (%d)", c.SignalPollIntervalMs)
	}
	if c.SignalGracePeriodMs < 0 {
		return fmt.Errorf("signal_grace_period_ms 不能为负 (%d)", c.SignalGracePeriodMs)
	}
	if c.RuntimeIdleWarningMs < 0 {
		return fmt.Errorf("runtime_idle_warning_ms 不能为负 (%d)", c.RuntimeIdleWarningMs)
	}
	if c.UserConfirmTimeoutMs < 0 {
		return fmt.Errorf("user_confirm_timeout_ms 不能为负 (%d)", c.UserConfirmTimeoutMs)
	}
	if err := ValidateAllowedCommands(c.CommandAllowlist); err != nil {
		return err
	}
	return nil
}

// GlobalConfigPatch 是 PATCH /api/settings 的部分更新结构。
// 每个字段都是指针：nil = 不修改，非 nil = 覆盖。
type GlobalConfigPatch struct {
	MaxTurnsPerPhase          *int              `json:"max_turns_per_phase"`
	MaxReworkPerQuest         *int              `json:"max_rework_per_quest"`
	MaxConcurrent             *int              `json:"max_concurrent"`
	DefaultModel              *string           `json:"default_model"`
	DefaultWorkingDir         *string           `json:"default_working_dir"`
	DefaultWorkspaceMode      *string           `json:"default_workspace_mode"`
	DefaultWarriorID          *string           `json:"default_warrior_id"`
	DefaultMageID             *string           `json:"default_mage_id"`
	DefaultExecuteAgentID     *string           `json:"default_execute_agent_id"`
	DefaultReviewAgentID      *string           `json:"default_review_agent_id"`
	ContextFileMode           *bool             `json:"context_file_mode"`
	WorkspaceRetentionDays    *int              `json:"workspace_retention_days"`
	AutoCleanupFailedQuests   *bool             `json:"auto_cleanup_failed_quests"`
	MaxDurationPerPhaseMs     *int64            `json:"max_duration_per_phase_ms"`
	MaxDurationPerQuestMs     *int64            `json:"max_duration_per_quest_ms"`
	UserConfirmTimeoutMs      *int64            `json:"user_confirm_timeout_ms"`
	MaxReviewHints            *int              `json:"max_review_hints"`
	MaxExecutionHints         *int              `json:"max_execution_hints"`
	MaxConsecutiveAgentErrors *int              `json:"max_consecutive_agent_errors"`
	MaxNoProgressTurns        *int              `json:"max_no_progress_turns"`
	SignalPollIntervalMs      *int64            `json:"signal_poll_interval_ms"`
	SignalGracePeriodMs       *int64            `json:"signal_grace_period_ms"`
	RuntimeIdleWarningMs      *int64            `json:"runtime_idle_warning_ms"`
	CommandAllowlist          *[]AllowedCommand `json:"mage_command_allowlist"`
	Connectors                *ConnectorsConfig `json:"connectors"`
	HotlAutoClose             *bool             `json:"hotl_auto_close"`
}

// Apply 把 patch 里非 nil 字段合并进 base，返回新对象（base 不被修改）。
func (p *GlobalConfigPatch) Apply(base *GlobalConfig) *GlobalConfig {
	if base == nil {
		base = DefaultGlobalConfig()
	}
	merged := *base
	if p == nil {
		return &merged
	}
	if p.MaxTurnsPerPhase != nil {
		merged.MaxTurnsPerPhase = *p.MaxTurnsPerPhase
	}
	if p.MaxReworkPerQuest != nil {
		merged.MaxReworkPerQuest = *p.MaxReworkPerQuest
	}
	if p.MaxConcurrent != nil {
		merged.MaxConcurrent = *p.MaxConcurrent
	}
	if p.DefaultModel != nil {
		merged.DefaultModel = *p.DefaultModel
	}
	if p.DefaultWorkingDir != nil {
		merged.DefaultWorkingDir = *p.DefaultWorkingDir
	}
	if p.DefaultWorkspaceMode != nil {
		merged.DefaultWorkspaceMode = *p.DefaultWorkspaceMode
	}
	if p.DefaultWarriorID != nil {
		merged.DefaultWarriorID = *p.DefaultWarriorID
	}
	if p.DefaultMageID != nil {
		merged.DefaultMageID = *p.DefaultMageID
	}
	if p.DefaultExecuteAgentID != nil {
		merged.DefaultExecuteAgentID = *p.DefaultExecuteAgentID
	}
	if p.DefaultReviewAgentID != nil {
		merged.DefaultReviewAgentID = *p.DefaultReviewAgentID
	}
	if p.ContextFileMode != nil {
		merged.ContextFileMode = *p.ContextFileMode
	}
	if p.WorkspaceRetentionDays != nil {
		merged.WorkspaceRetentionDays = *p.WorkspaceRetentionDays
	}
	if p.AutoCleanupFailedQuests != nil {
		merged.AutoCleanupFailedQuests = *p.AutoCleanupFailedQuests
	}
	if p.MaxDurationPerQuestMs != nil {
		merged.MaxDurationPerQuestMs = *p.MaxDurationPerQuestMs
	}
	if p.UserConfirmTimeoutMs != nil {
		merged.UserConfirmTimeoutMs = *p.UserConfirmTimeoutMs
	}
	if p.MaxConsecutiveAgentErrors != nil {
		merged.MaxConsecutiveAgentErrors = *p.MaxConsecutiveAgentErrors
	}
	if p.MaxNoProgressTurns != nil {
		merged.MaxNoProgressTurns = *p.MaxNoProgressTurns
	}
	if p.SignalPollIntervalMs != nil {
		merged.SignalPollIntervalMs = *p.SignalPollIntervalMs
	}
	if p.SignalGracePeriodMs != nil {
		merged.SignalGracePeriodMs = *p.SignalGracePeriodMs
	}
	if p.RuntimeIdleWarningMs != nil {
		merged.RuntimeIdleWarningMs = *p.RuntimeIdleWarningMs
	}
	if p.CommandAllowlist != nil {
		merged.CommandAllowlist = *p.CommandAllowlist
	}
	if p.Connectors != nil {
		merged.Connectors = *p.Connectors
	}
	if p.HotlAutoClose != nil {
		v := *p.HotlAutoClose
		merged.HotlAutoClose = &v
	}
	return &merged
}

// ==================== Agent 配置（agents/<name>.json） ====================

// AgentConfig v2: 统一 ACP 和 CLI 两种接入方式
type AgentConfig struct {
	Name                string            `json:"name"`
	Type                model.AgentType   `json:"type"`          // acp / cli
	Command             string            `json:"command"`       // 启动命令
	Args                []string          `json:"args"`          // 启动参数
	Env                 map[string]string `json:"env,omitempty"` // 环境变量
	EnvFiles            []string          `json:"env_files,omitempty"`
	DefaultModel        string            `json:"default_model"` // 默认模型
	DefaultAllowedTools []string          `json:"default_allowed_tools,omitempty"`
	SupportsReadOnly    *bool             `json:"supports_readonly,omitempty"`
	ExecutionTraits     []string          `json:"execution_traits,omitempty"`
	Enabled             bool              `json:"enabled"`
	Official            bool              `json:"official,omitempty"`
}

func BuiltinAgentType(name string) (model.AgentType, bool) {
	switch name {
	case "traex", "relay", "pi", "codex", "aiden_x_claude", "aiden_x_codex", "hermes":
		return model.AgentTypeCLI, true
	default:
		return "", false
	}
}

func IsBuiltinAgentName(name string) bool {
	_, ok := BuiltinAgentType(name)
	return ok
}

func (p *AgentConfig) EffectiveEnv() (map[string]string, error) {
	if p == nil {
		return nil, nil
	}
	env := map[string]string{}
	if isClaudeAgent(p) {
		items, err := DiscoverClaudeEnv()
		if err != nil {
			return nil, err
		}
		for k, v := range items {
			env[k] = v
		}
	}
	for _, path := range p.EnvFiles {
		items, err := ReadEnvFile(path)
		if err != nil {
			return nil, err
		}
		for k, v := range items {
			env[k] = v
		}
	}
	for k, v := range p.Env {
		env[k] = v
	}
	if len(env) == 0 {
		return nil, nil
	}
	return env, nil
}

func isClaudeAgent(p *AgentConfig) bool {
	if p == nil {
		return false
	}
	if strings.EqualFold(p.Name, "claude") {
		return true
	}
	if p.Command == "npx" {
		for _, arg := range p.Args {
			if strings.Contains(arg, "claude-agent-acp") {
				return true
			}
		}
	}
	return strings.Contains(filepath.Base(p.Command), "claude")
}

func DiscoverClaudeEnv() (map[string]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	candidates := []string{
		filepath.Join(home, ".profile"),
		filepath.Join(home, ".bash_profile"),
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".zprofile"),
		filepath.Join(home, ".zshrc"),
	}
	out := map[string]string{}
	for _, path := range candidates {
		items, err := readEnvFileAllowed(path, isClaudeEnvKey, false)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		for k, v := range items {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func isClaudeEnvKey(k string) bool {
	if k == "ANTHROPIC_API_KEY" || k == "ANTHROPIC_AUTH_TOKEN" || k == "ANTHROPIC_BASE_URL" || k == "ANTHROPIC_MODEL" {
		return true
	}
	return strings.HasPrefix(k, "ANTHROPIC_DEFAULT_") || strings.HasPrefix(k, "CLAUDE_CODE_")
}

func ReadEnvFile(path string) (map[string]string, error) {
	return readEnvFileAllowed(path, func(string) bool { return true }, true)
}

func readEnvFileAllowed(path string, allow func(string) bool, strict bool) (map[string]string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	expanded, err := expandUserPath(path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(expanded)
	if err != nil {
		return nil, fmt.Errorf("读取 env file %s 失败: %w", path, err)
	}
	defer f.Close()

	out := map[string]string{}
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			if !strict {
				continue
			}
			return nil, fmt.Errorf("解析 env file %s:%d 失败: 缺少 KEY=VALUE", path, lineNo)
		}
		k = strings.TrimSpace(k)
		if k == "" || strings.ContainsAny(k, " \t") {
			if !strict {
				continue
			}
			return nil, fmt.Errorf("解析 env file %s:%d 失败: key 非法", path, lineNo)
		}
		v = strings.TrimSpace(v)
		if unquoted, err := strconv.Unquote(v); err == nil {
			v = unquoted
		}
		if allow == nil || allow(k) {
			out[k] = v
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 env file %s 失败: %w", path, err)
	}
	return out, nil
}

func expandUserPath(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}
	return path, nil
}

func (r *Root) ListAgents() ([]*AgentConfig, error) {
	return listJSONFiles[AgentConfig](r.Sub(SubdirAgents))
}

func (r *Root) SaveAgent(p *AgentConfig) error {
	if p.Name == "" {
		return fmt.Errorf("agent name 为空")
	}
	return WriteJSON(r.Sub(SubdirAgents, p.Name+".json"), p)
}

func (r *Root) DeleteAgent(name string) error {
	if name == "" {
		return fmt.Errorf("agent name 为空")
	}
	return os.Remove(r.Sub(SubdirAgents, name+".json"))
}

// GetAgent 按名字获取 agent 配置
func (r *Root) GetAgent(name string) (*AgentConfig, error) {
	p, err := ReadJSON[AgentConfig](r.Sub(SubdirAgents, name+".json"))
	if err != nil {
		return nil, fmt.Errorf("agent 不存在 %s: %w", name, err)
	}
	return p, nil
}

// ListAgentsByType 按类型枚举 agent
func (r *Root) ListAgentsByType(ptype model.AgentType) ([]*AgentConfig, error) {
	all, err := r.ListAgents()
	if err != nil {
		return nil, err
	}
	var out []*AgentConfig
	for _, p := range all {
		if p.Enabled && p.Type == ptype {
			out = append(out, p)
		}
	}
	return out, nil
}

// ==================== 冒险者（adventurers/*.json） ====================

// AdventurerFile v2: 职业 + agent + 经验等级
type AdventurerFile struct {
	// 基础信息
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Class       model.AdventurerClass  `json:"class"`       // warrior / mage
	Description string                 `json:"description"` // 简介
	Status      model.AdventurerStatus `json:"status"`      // pending_setup / active / retired

	// Agent 配置
	Agent        string   `json:"agent"`                  // 对应 agents/*.json 的 name
	Model        string   `json:"model,omitempty"`        // 冒险者级模型覆盖；为空则使用 agent.default_model
	CustomPrompt string   `json:"custom_prompt"`          // 自定义人设
	Tools        []string `json:"tools"`                  // 允许的工具列表
	ExtraSkills  []string `json:"extra_skills,omitempty"` // 额外技能名称列表（在职业默认技能之外追加）

	// 经验与战绩
	Level     int   `json:"level"`
	Exp       int64 `json:"exp"`
	WinCount  int   `json:"win_count"`
	LoseCount int   `json:"lose_count"`

	// 元信息
	CreatedAtMs int64 `json:"created_at_ms"`

	// 计算字段（运行时填充，不落盘）
	// Deprecated: use GetTitle()
	Title string `json:"title,omitempty"`
}

func (r *Root) ListAdventurers() ([]*AdventurerFile, error) {
	out, err := listJSONFiles[AdventurerFile](r.Sub(SubdirAdventurers))
	if err != nil {
		return nil, err
	}
	for _, a := range out {
		a.populateComputed()
	}
	return out, nil
}

func (r *Root) GetAdventurer(id string) (*AdventurerFile, error) {
	a, err := ReadJSON[AdventurerFile](r.Sub(SubdirAdventurers, id+".json"))
	if err != nil {
		return nil, fmt.Errorf("冒险者不存在 %s: %w", id, err)
	}
	a.populateComputed()
	return a, nil
}

// populateComputed 填充计算字段（title 等）。
func (a *AdventurerFile) populateComputed() {
	a.Title = LevelTitle(a.Class, a.Level)
}

func (r *Root) SaveAdventurer(a *AdventurerFile) error {
	if a.ID == "" {
		return fmt.Errorf("adventurer id 为空")
	}
	return WriteJSON(r.Sub(SubdirAdventurers, a.ID+".json"), a)
}

// Title 返回该冒险者的等级称号（如 "大剑士" / "大法师"）。
// 字段 Title 由 populateComputed 填充，方法 Title() 提供程序化访问。
func (a *AdventurerFile) GetTitle() string {
	if a.Title != "" {
		return a.Title
	}
	return LevelTitle(a.Class, a.Level)
}

// ActivateAdventurer 激活冒险者（pending_setup → active）。
// 可选传入更新字段（name/agent/custom_prompt/tools），激活时一并写入。
func (r *Root) ActivateAdventurer(id string, updates *AdventurerUpdate) (*AdventurerFile, error) {
	a, err := r.GetAdventurer(id)
	if err != nil {
		return nil, err
	}
	if a.Status == model.AdventurerActive {
		return a, nil // 已经是 active，幂等
	}
	if a.Status != model.AdventurerPendingSetup {
		return nil, fmt.Errorf("冒险者状态 %s，无法激活（仅 pending_setup 可激活）", a.Status)
	}

	if updates != nil {
		if updates.Name != "" {
			a.Name = updates.Name
		}
		if updates.Agent != "" {
			a.Agent = updates.Agent
		}
		if updates.Model != nil {
			a.Model = *updates.Model
		}
		if updates.CustomPrompt != nil {
			a.CustomPrompt = *updates.CustomPrompt
		}
		if updates.Tools != nil {
			a.Tools = updates.Tools
		}
		if updates.Description != nil {
			a.Description = *updates.Description
		}
	}

	a.Status = model.AdventurerActive
	if err := r.SaveAdventurer(a); err != nil {
		return nil, err
	}
	return a, nil
}

// AdventurerUpdate 激活/更新冒险者时的可选字段。
type AdventurerUpdate struct {
	Name         string   `json:"name,omitempty"`
	Agent        string   `json:"agent,omitempty"`
	Model        *string  `json:"model,omitempty"`
	Description  *string  `json:"description,omitempty"`
	CustomPrompt *string  `json:"custom_prompt,omitempty"`
	Tools        []string `json:"tools,omitempty"`
}

// FindByClass 在可用冒险者中挑选最合适的冒险者：胜率优先，等级次之。
func (r *Root) FindByClass(class model.AdventurerClass) (*AdventurerFile, error) {
	all, err := r.ListAdventurers()
	if err != nil {
		return nil, err
	}
	var pick *AdventurerFile
	for _, a := range all {
		if a.Status != model.AdventurerActive {
			continue
		}
		if a.Class != class {
			continue
		}
		if pick == nil || BetterAdventurer(a, pick) {
			pick = a
		}
	}
	if pick == nil {
		return nil, fmt.Errorf("没找到职业=%s 的可用冒险者（跑 `gloop init` 生成基础冒险者）", class)
	}
	return pick, nil
}

func BetterAdventurer(a, b *AdventurerFile) bool {
	aRate := winRate(a)
	bRate := winRate(b)
	if aRate != bRate {
		return aRate > bRate
	}
	if a.Level != b.Level {
		return a.Level > b.Level
	}
	if a.Exp != b.Exp {
		return a.Exp > b.Exp
	}
	return a.CreatedAtMs < b.CreatedAtMs
}

func winRate(a *AdventurerFile) float64 {
	total := a.WinCount + a.LoseCount
	if total == 0 {
		return 0
	}
	return float64(a.WinCount) / float64(total)
}
