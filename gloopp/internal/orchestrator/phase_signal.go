package orchestrator

import (
	"encoding/json"
	"fmt"
)

// PhaseSignal 是 agent→gloop CLI 阶段信号的载体。
//
// v0.3 起 agent 通过 `gloop phase done` / `gloop review` 等 CLI 命令提交阶段结论，
// CLI 写信号文件或调 server API，orchestrator 消费后得到 PhaseSignal。
// 它是 maker/checker 模型里 checker 侧的判定结果：决定阶段是否结束、verdict 是什么。
//
// （历史：这个类型原叫 platformtools.ToolResult，是 v0.2 native tool 分发层的返回值。
// v0.3.4 删掉了 native tool 分发层，类型随语义改名迁入 orchestrator。）
type PhaseSignal struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	// 标记本阶段结束
	PhaseEnded bool `json:"phase_ended,omitempty"`
	// 阶段结论（由 macro_loop 解析，驱动 quest 状态流转）
	PhaseVerdict string `json:"phase_verdict,omitempty"`
	PhaseComment string `json:"phase_comment,omitempty"`
	PhaseHints   string `json:"phase_hints,omitempty"`
	PhaseScore   int    `json:"phase_score,omitempty"` // 评审评分（1-10），verdict=pass 时使用
	// StructuredReview mage 提交的 typed 评审协议（Phase 1.5+，见 mage-review-spec §4.1）。
	// 原始 JSON 字节，由 macro_loop 解析为 domain.StructuredReview 并做两层校验。
	StructuredReview json.RawMessage `json:"structured_review,omitempty"`
}

// PhaseSignalToJSON 把信号序列化成 JSON 字符串（写 session row / 事件 payload 用）。
func PhaseSignalToJSON(s PhaseSignal) string {
	b, err := json.Marshal(s)
	if err != nil {
		return fmt.Sprintf(`{"ok":false,"message":"序列化失败: %s"}`, err.Error())
	}
	return string(b)
}
