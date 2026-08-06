package notifications

import (
	"context"
	"os"
)

// Notifier 是通知发送器的通用接口。
// 所有通知方法都是 best-effort：失败只返回 ok=false + 原因，不返回 error。
type Notifier interface {
	// Available 通知是否可用。
	Available() bool

	// StatusError 不可用时返回原因描述，可用时返回空。
	StatusError() string

	// SetBaseURL 设置 dashboard 基础 URL（用于拼接跳转链接）。
	SetBaseURL(url string)

	// BaseURL 返回当前 dashboard 基础 URL。
	BaseURL() string

	// Recipient 返回接收人信息（open_id + 姓名），不可用时返回空。
	Recipient() (openID, name string)

	// Send 发送简单 Markdown 消息。
	Send(ctx context.Context, title, body string) (ok bool, message string)

	// SendCard 发送交互式卡片消息。
	SendCard(ctx context.Context, card map[string]any) (ok bool, message string)
}

// DisabledByEnv 返回是否通过环境变量禁用了通知。
// GLOOP_NOTIFICATIONS_DISABLED 或 GLOOP_NO_NOTIFICATIONS 设为非空时禁用。
func DisabledByEnv() bool {
	if os.Getenv("GLOOP_NOTIFICATIONS_DISABLED") != "" {
		return true
	}
	if os.Getenv("GLOOP_NO_NOTIFICATIONS") != "" {
		return true
	}
	return false
}

// NoopNotifier 是一个空实现的通知器，永远返回不可用。
// 用于测试或不想发送通知的场景。
type NoopNotifier struct {
	baseURL string
}

// NewNoopNotifier 创建一个空通知器。
func NewNoopNotifier() *NoopNotifier {
	return &NoopNotifier{}
}

func (n *NoopNotifier) Available() bool             { return false }
func (n *NoopNotifier) StatusError() string         { return "通知已禁用（noop）" }
func (n *NoopNotifier) SetBaseURL(url string)       { n.baseURL = url }
func (n *NoopNotifier) BaseURL() string             { return n.baseURL }
func (n *NoopNotifier) Recipient() (string, string) { return "", "" }
func (n *NoopNotifier) Send(ctx context.Context, title, body string) (bool, string) {
	return false, "通知已禁用"
}
func (n *NoopNotifier) SendCard(ctx context.Context, card map[string]any) (bool, string) {
	return false, "通知已禁用"
}
