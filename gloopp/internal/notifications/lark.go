package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// LarkCliNotifier 通过 lark-cli 发送飞书通知。
// 依赖系统中已安装并登录的 lark-cli 命令行工具。
// best-effort：发送失败不返回 error 给上层，只记日志。
type LarkCliNotifier struct {
	mu         sync.RWMutex
	available  bool
	userOpenID string
	userName   string
	lastCheck  time.Time
	checkError string

	baseURL string // dashboard 基础 URL，如 "http://127.0.0.1:37317"
}

type larkAuthStatus struct {
	Identities struct {
		Bot  larkIdentity `json:"bot"`
		User larkIdentity `json:"user"`
	} `json:"identities"`
	Brand string `json:"brand"`
}

type larkIdentity struct {
	Status    string `json:"status"`
	Available bool   `json:"available"`
	Message   string `json:"message"`
	OpenID    string `json:"openId"`
	UserName  string `json:"userName"`
}

type larkSendResult struct {
	OK   bool `json:"ok"`
	Data struct {
		MessageID string `json:"message_id"`
	} `json:"data"`
	Error *larkApiError `json:"error,omitempty"`
}

type larkApiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// NewLarkCliNotifier 创建飞书通知器，并立即检测可用性。
func NewLarkCliNotifier() *LarkCliNotifier {
	n := &LarkCliNotifier{}
	n.checkAvailability()
	return n
}

// SetBaseURL 设置 dashboard 基础 URL，用于通知里的跳转链接。
// 格式如 "http://127.0.0.1:37317"，不带尾斜杠。
func (n *LarkCliNotifier) SetBaseURL(url string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.baseURL = strings.TrimRight(url, "/")
}

// BaseURL 返回当前 dashboard 基础 URL。
func (n *LarkCliNotifier) BaseURL() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.baseURL
}

// Available 返回通知是否可用。
func (n *LarkCliNotifier) Available() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	// 缓存超过 5 分钟就重新检测（惰性）
	if time.Since(n.lastCheck) > 5*time.Minute {
		go n.checkAvailability()
	}
	return n.available
}

// Recipient 返回当前接收人信息（open_id + 姓名）。
// 不可用时返回空字符串。
func (n *LarkCliNotifier) Recipient() (openID, name string) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.userOpenID, n.userName
}

// StatusError 如果不可用，返回原因描述；可用时返回空字符串。
func (n *LarkCliNotifier) StatusError() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.available {
		return ""
	}
	return n.checkError
}

// Send 发送一条简单的飞书 Markdown 消息给当前登录用户。
// 用于 agent 通过 notify_user 工具主动发送的通知。
// best-effort：发送失败不会返回 error（只在内部日志里记录），
// 但会返回 ok=false 和错误描述，方便调用方告知 agent。
func (n *LarkCliNotifier) Send(ctx context.Context, title, body string) (ok bool, message string) {
	if !n.Available() {
		return false, "飞书通知不可用：" + n.StatusError()
	}

	n.mu.RLock()
	openID := n.userOpenID
	baseURL := n.baseURL
	n.mu.RUnlock()

	// 组装消息内容：标题加粗 + 正文 + Gloop 标识
	text := "**Gloop · " + title + "**"
	if body != "" {
		text += "\n" + body
	}

	// 末尾加上 Gloop 标识和链接
	signature := "\n\n---\n*来自 Gloop 自动化助手*"
	if baseURL != "" {
		signature += " · [打开控制台](" + baseURL + "/dashboard)"
	}
	text += signature

	cmd := exec.CommandContext(ctx, "lark-cli",
		"im", "+messages-send",
		"--as", "bot",
		"--user-id", openID,
		"--markdown", text,
		"--format", "json",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// 执行失败，可能是 lark-cli 没装或 token 过期，重新检测
		go n.checkAvailability()
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return false, "发送失败：" + strings.TrimSpace(errMsg)
	}

	var result larkSendResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err == nil && !result.OK {
		if result.Error != nil {
			return false, "API 错误：" + result.Error.Message
		}
		return false, "发送失败（未知原因）"
	}

	return true, "已发送"
}

// SendCard 发送一张交互式卡片消息。
// 用于系统级通知（quest 状态变更等）。
// card 是飞书卡片 JSON 的 map 结构，调用方负责组装。
func (n *LarkCliNotifier) SendCard(ctx context.Context, card map[string]any) (ok bool, message string) {
	if !n.Available() {
		return false, "飞书通知不可用：" + n.StatusError()
	}

	n.mu.RLock()
	openID := n.userOpenID
	n.mu.RUnlock()

	cardJSON, err := json.Marshal(card)
	if err != nil {
		return false, "卡片序列化失败：" + err.Error()
	}

	cmd := exec.CommandContext(ctx, "lark-cli",
		"im", "+messages-send",
		"--as", "bot",
		"--user-id", openID,
		"--msg-type", "interactive",
		"--content", string(cardJSON),
		"--format", "json",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		go n.checkAvailability()
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return false, "发送失败：" + strings.TrimSpace(errMsg)
	}

	var result larkSendResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err == nil && !result.OK {
		if result.Error != nil {
			return false, "API 错误：" + result.Error.Message
		}
		return false, "发送失败（未知原因）"
	}

	return true, "已发送"
}

// checkAvailability 检测 lark-cli 是否安装并登录。
// 写操作，调用方需持有写锁或确保并发安全。
func (n *LarkCliNotifier) checkAvailability() {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.lastCheck = time.Now()

	// 1. 检查 lark-cli 是否存在
	path, err := exec.LookPath("lark-cli")
	if err != nil || path == "" {
		n.available = false
		n.checkError = "未找到 lark-cli 命令，请先安装：npm install -g @larksuite/cli"
		return
	}

	// 2. 检查登录状态
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "lark-cli", "auth", "status")
	out, err := cmd.Output()
	if err != nil {
		n.available = false
		n.checkError = fmt.Sprintf("lark-cli auth status 失败：%v", err)
		return
	}

	var status larkAuthStatus
	if err := json.Unmarshal(out, &status); err != nil {
		n.available = false
		n.checkError = "解析 lark-cli auth status 输出失败"
		return
	}

	// 优先用 bot 身份（通常总是可用），bot 不可用再尝试 user
	if status.Identities.Bot.Available {
		n.available = true
		n.checkError = ""
		// bot 身份发消息需要知道用户 open_id，从 user 身份里拿（即便是 missing 也有 openId 字段）
		if status.Identities.User.OpenID != "" {
			n.userOpenID = status.Identities.User.OpenID
			n.userName = status.Identities.User.UserName
		}
		return
	}

	n.available = false
	n.checkError = "lark-cli 未登录，请运行：lark-cli auth login"
}
