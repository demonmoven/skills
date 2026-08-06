package fsstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
	"code.byted.org/lihuanyu.0w0/gloop/internal/version"
)

// Root 封装 `~/.gloop/` 的根路径与子目录常量。
// 数据分三层：
//
//	配置层（用户可手改 + git 友好）：config.json / agents/*.json / adventurers/*.json / prompts/*
//	运行态层（机器生成，加 .gitignore）：workspace/quests/<qid>/*、workspace/stats/*.json
//	本地绑定密钥层（敏感、权限 0600）：token.json
//
// 注意：Root 本身是 thread-unsafe 的。进程内共享时，quest 目录之间无竞争；
// 若需要并发保护，锁应放在 QuestStore/StatsStore 等子 store 层级。
type Root struct {
	path string
}

const (
	SubdirAgents      = "agents"
	SubdirAdventurers = "adventurers"
	SubdirPrompts     = "prompts"
	SubdirAutomation  = "automation"
	SubdirWorkspace   = "workspace"
	SubdirQuests      = "workspace/quests"
	SubdirStats       = "workspace/stats"
	SubdirContext     = "context" // 用户上下文摘要（agent 启动时注入索引）

	FileConfig    = "config.json"
	FileToken     = "token.json"
	FileGitIgnore = ".gitignore"

	GitIgnoreContent = "# Gloop runtime data — 不要提交\nworkspace/\n*.swp\n.DS_Store\n"
)

var claudeACPArgs = []string{"--yes", "--package", "@agentclientprotocol/claude-agent-acp", "claude-agent-acp"}
var legacyClaudeACPArgs = []string{"@agentclientprotocol/claude-agent-acp"}

const relayDefaultModel = "alwaysday1"

// DefaultPath 返回 `~/.gloop` 绝对路径
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("读取 HOME 失败: %w", err)
	}
	return filepath.Join(home, version.DirName()), nil
}

// Open 打开（或初始化）根目录。root 为空时用 DefaultPath。
func Open(root string) (*Root, error) {
	if root == "" {
		p, err := DefaultPath()
		if err != nil {
			return nil, err
		}
		root = p
	}
	subs := []string{SubdirAgents, SubdirAdventurers, SubdirPrompts, SubdirAutomation, SubdirQuests, SubdirStats, SubdirContext}
	for _, s := range subs {
		if err := os.MkdirAll(filepath.Join(root, s), 0o755); err != nil {
			return nil, fmt.Errorf("创建子目录 %s 失败: %w", s, err)
		}
	}
	r := &Root{path: root}
	if err := r.RenameContextDim("loop_state", "activity_snapshot"); err != nil {
		fmt.Fprintf(os.Stderr, "[Gloop] 迁移 context/loop_state 到 activity_snapshot 失败: %v\n", err)
	}
	gi := filepath.Join(root, FileGitIgnore)
	if _, err := os.Stat(gi); errors.Is(err, os.ErrNotExist) {
		if wErr := os.WriteFile(gi, []byte(GitIgnoreContent), 0o644); wErr != nil {
			// 非关键文件，失败不阻塞，仅 stderr 提示
			fmt.Fprintf(os.Stderr, "[Gloop] 写入 .gitignore 失败: %v\n", wErr)
		}
	}
	return r, nil
}

// Path 返回根目录的绝对路径。
func (r *Root) Path() string { return r.path }

// Sub 以根目录为基础拼接子路径，返回任意层级的绝对路径。
func (r *Root) Sub(parts ...string) string {
	return filepath.Join(append([]string{r.path}, parts...)...)
}

// ExpandDir 展开路径里的 ~（home 目录）和环境变量。
// 用户在创建委托时可能填 ~/agent-workspace/xxx，git/cp 不认 ~ 字面量，必须展开。
func ExpandDir(dir string) string {
	if dir == "" {
		return dir
	}
	// 先展开 ~，再展开 $VAR
	if strings.HasPrefix(dir, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			if dir == "~" {
				dir = home
			} else if strings.HasPrefix(dir, "~/") {
				dir = filepath.Join(home, dir[2:])
			}
		}
	}
	return os.ExpandEnv(dir)
}

// InitDefaultFiles 首次启动时生成默认文件：
//   - v2 agents (traex/omp/relay/pi/codex/claude)
//   - 全局配置默认工作目录
//     配置存在就不覆盖，用户手改过的不丢
//
// AgentProbe 是 CLI 启动时探测到的各 agent 可执行文件路径。
// 有探测到的路径会优先使用，否则回退到裸命令名（依赖 PATH）。
type AgentProbe struct {
	TraeX  string
	Relay  string
	Pi     string
	Codex  string
	Aiden  string
	Hermes string
}

// InitDefaultFiles 首次启动时生成默认文件（v2 agent 配置、默认工作目录、默认 automation 模板）。
// 用户已存在的配置不覆盖，以保留手改。返回本次实际新建的文件相对路径列表。
func (r *Root) InitDefaultFiles(probe AgentProbe, workingDir string) (created []string, err error) {
	created = []string{}

	// ---------- v2 agents ----------
	v2Agents := []AgentConfig{
		{Name: "traex", Type: model.AgentTypeCLI, Command: pickProbeCommand(probe.TraeX, "traex"), DefaultModel: "auto", Enabled: false, Official: true},
		{Name: "relay", Type: model.AgentTypeCLI, Command: pickProbeCommand(probe.Relay, "relay"), Args: []string{"-p", "--verbose", "--output-format=stream-json"}, DefaultModel: relayDefaultModel, Enabled: false, Official: true},
		{Name: "pi", Type: model.AgentTypeCLI, Command: pickProbeCommand(probe.Pi, "pi"), Args: []string{"--mode", "json", "--no-session"}, DefaultModel: "auto", Enabled: false, Official: true},
		{Name: "codex", Type: model.AgentTypeCLI, Command: pickProbeCommand(probe.Codex, "codex"), DefaultModel: "auto", Enabled: false, Official: true},
		{Name: "aiden_x_claude", Type: model.AgentTypeCLI, Command: pickProbeCommand(probe.Aiden, "aiden"), Args: []string{"x", "claude", "--stream-json"}, DefaultModel: "auto", Enabled: false, Official: true},
		{Name: "aiden_x_codex", Type: model.AgentTypeCLI, Command: pickProbeCommand(probe.Aiden, "aiden"), Args: []string{"x", "codex", "--stream-json"}, DefaultModel: "auto", Enabled: false, Official: true},
		{Name: "hermes", Type: model.AgentTypeCLI, Command: pickProbeCommand(probe.Hermes, "hermes"), DefaultModel: "auto", Enabled: false, Official: true},
	}
	for _, p := range v2Agents {
		path := r.Sub(SubdirAgents, p.Name+".json")
		if _, err2 := os.Stat(path); err2 == nil {
			if p.Name == "relay" {
				if existing, readErr := ReadJSON[AgentConfig](path); readErr == nil && shouldRepairLegacyRelayDefaultModel(existing) {
					existing.DefaultModel = relayDefaultModel
					existing.Official = true
					if saveErr := r.SaveAgent(existing); saveErr != nil {
						fmt.Fprintf(os.Stderr, "[warn] repair %s: %v\n", path, saveErr)
					}
				}
			}
			continue
		} else if !errors.Is(err2, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "[warn] stat %s: %v\n", path, err2)
			continue
		}
		if e2 := r.SaveAgent(&p); e2 != nil {
			fmt.Fprintf(os.Stderr, "[warn] save %s: %v\n", path, e2)
			continue
		}
		created = append(created, "agents/"+p.Name+".json")
	}

	// ---------- legacy repairs（已从官方列表移除但用户可能仍有残留配置） ----------
	r.repairLegacyAgentIfExists("claude", func(p *AgentConfig) bool {
		return shouldRepairLegacyClaudeACP(p)
	}, func(p *AgentConfig) {
		p.Args = append([]string(nil), claudeACPArgs...)
		p.Official = true
	})

	// traex_cli → traex 重命名迁移
	r.migrateLegacyAgentName("traex_cli", "traex")

	// ---------- 全局配置（只在缺失时写默认工作目录） ----------
	cfg, e := r.LoadConfig()
	if e == nil && cfg.DefaultWorkingDir == "" {
		cfg.DefaultWorkingDir = workingDir
		if e := r.SaveConfig(cfg); e != nil {
			fmt.Fprintf(os.Stderr, "[warn] save %s: %v\n", FileConfig, e)
		} else {
			created = append(created, FileConfig)
		}
	}

	// ---------- 默认 automation 配置 ----------
	if autoFiles, err := r.InitDefaultAutomations(); err == nil && len(autoFiles) > 0 {
		created = append(created, autoFiles...)
	}

	return created, nil
}

func shouldRepairLegacyClaudeACP(p *AgentConfig) bool {
	if p == nil {
		return false
	}
	return p.Type == model.AgentTypeACP &&
		strings.EqualFold(p.Name, "claude") &&
		strings.EqualFold(strings.TrimSpace(p.Command), "npx") &&
		reflect.DeepEqual(p.Args, legacyClaudeACPArgs)
}

func shouldRepairLegacyRelayDefaultModel(p *AgentConfig) bool {
	if p == nil {
		return false
	}
	return p.Type == model.AgentTypeCLI &&
		strings.EqualFold(p.Name, "relay") &&
		strings.EqualFold(strings.TrimSuffix(filepath.Base(p.Command), filepath.Ext(p.Command)), "relay") &&
		(p.DefaultModel == "" || p.DefaultModel == "auto")
}

// repairLegacyAgentIfExists 检查一个已从官方列表移除的 agent 是否仍有残留配置，
// 如果需要修复则就地修改并保存。用于升级兼容。
func (r *Root) repairLegacyAgentIfExists(name string, needsRepair func(*AgentConfig) bool, apply func(*AgentConfig)) {
	path := r.Sub(SubdirAgents, name+".json")
	existing, readErr := ReadJSON[AgentConfig](path)
	if readErr != nil {
		return
	}
	if !needsRepair(existing) {
		return
	}
	apply(existing)
	if saveErr := r.SaveAgent(existing); saveErr != nil {
		fmt.Fprintf(os.Stderr, "[warn] repair legacy %s: %v\n", name, saveErr)
	}
}

// migrateLegacyAgentName 把旧名 agent 配置迁移到新名（仅当新名不存在时执行）。
func (r *Root) migrateLegacyAgentName(oldName, newName string) {
	oldPath := r.Sub(SubdirAgents, oldName+".json")
	newPath := r.Sub(SubdirAgents, newName+".json")
	if _, err := os.Stat(newPath); err == nil {
		return // 新名已存在，不覆盖
	}
	existing, readErr := ReadJSON[AgentConfig](oldPath)
	if readErr != nil {
		return
	}
	existing.Name = newName
	existing.Official = true
	if saveErr := r.SaveAgent(existing); saveErr != nil {
		fmt.Fprintf(os.Stderr, "[warn] migrate %s→%s: %v\n", oldName, newName, saveErr)
		return
	}
	os.Remove(oldPath)
}

func pickProbeCommand(probed, fallback string) string {
	if strings.TrimSpace(probed) != "" {
		return probed
	}
	return fallback
}

// ==================== 通用 JSON 读写 ====================

func ReadJSON[T any](path string) (*T, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("解析 %s 失败: %w", filepath.Base(path), err)
	}
	return &out, nil
}

// WriteJSON 写 JSON 用 temp + rename，保证崩溃安全。
func WriteJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// AppendJSONL 追加单行 JSON。O_APPEND 语义，崩溃安全。
func AppendJSONL(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(raw, '\n')); err != nil {
		return err
	}
	return nil
}

// RotateJSONLIfBig 在 jsonl 文件超过 maxBytes 时轮转：
// 当前文件重命名为 events.<ts>.jsonl（保留最近 keep 份，超出的删最老），
// 原路径重新开始写入。maxBytes<=0 或文件未超阈值时 no-op。
func RotateJSONLIfBig(path string, maxBytes int64, keep int) error {
	if maxBytes <= 0 || keep <= 0 {
		return nil
	}
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if fi.Size() < maxBytes {
		return nil
	}
	dir := filepath.Dir(path)
	base := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	// 轮转：当前 → events.<unix>.jsonl
	rotated := filepath.Join(dir, base+"."+strconv.FormatInt(time.Now().Unix(), 10)+".jsonl")
	if err := os.Rename(path, rotated); err != nil {
		return err
	}
	// 清理超出 keep 份的旧轮转文件
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var rotFiles []os.DirEntry
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, base+".") && strings.HasSuffix(name, ".jsonl") {
			rotFiles = append(rotFiles, e)
		}
	}
	// 按名字里的时间戳排序（降序=新→老），删超出 keep 的
	for i := 0; i < len(rotFiles); i++ {
		for j := i + 1; j < len(rotFiles); j++ {
			if rotFiles[j].Name() > rotFiles[i].Name() {
				rotFiles[i], rotFiles[j] = rotFiles[j], rotFiles[i]
			}
		}
	}
	for i := keep; i < len(rotFiles); i++ {
		_ = os.Remove(filepath.Join(dir, rotFiles[i].Name()))
	}
	return nil
}

// ReadJSONL 读取最后 n 行；n<=0 读全部。
func ReadJSONL[T any](path string, n int) ([]T, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []T
	start := 0
	// 如果要最后 n 行，先切 byte 倒着找换行
	if n > 0 {
		count := 0
		i := len(raw) - 1
		for ; i >= 0 && count < n; i-- {
			if raw[i] == '\n' {
				count++
				if count == n {
					start = i + 1
					break
				}
			}
		}
		if i < 0 {
			start = 0
		}
	}
	rest := raw[start:]
	lineStart := 0
	for i := 0; i < len(rest); i++ {
		if rest[i] == '\n' {
			if i > lineStart {
				var t T
				if err := json.Unmarshal(rest[lineStart:i], &t); err == nil {
					out = append(out, t)
				}
			}
			lineStart = i + 1
		}
	}
	if lineStart < len(rest) {
		var t T
		if err := json.Unmarshal(rest[lineStart:], &t); err == nil {
			out = append(out, t)
		}
	}
	return out, nil
}

// ==================== 小工具 ====================

// listJSONFiles 遍历目录中所有 .json 文件并反序列化为 *T。
// 坏文件或读取失败 emit 一条 stderr warning 后跳过。
func listJSONFiles[T any](dir string) ([]*T, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []*T
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		p := filepath.Join(dir, e.Name())
		c, err := ReadJSON[T](p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[warn] skip bad json %s: %v\n", p, err)
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

// FromMs 毫秒 → time.Time
func FromMs(ms int64) time.Time { return time.UnixMilli(ms) }

// NowMs 返回当前时间的毫秒时间戳。
func NowMs() int64 { return model.NowMs() }
