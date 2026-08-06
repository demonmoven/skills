package orchestrator

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
)

// ==================== 通用工具函数 ====================

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}

// randSuffix 生成 n 位随机字母数字后缀，用于 post_id 等唯一标识。
func randSuffix(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func toJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%+v", v)
	}
	return string(b)
}

func marshalMap(m map[string]any) string {
	if len(m) == 0 {
		return ""
	}
	return toJSON(m)
}

// extractJSON 从文本中提取第一个 JSON 对象（简单实现）
func extractJSON(text string) (string, bool) {
	start := strings.Index(text, "{")
	if start == -1 {
		return "", false
	}
	depth := 0
	for i := start; i < len(text); i++ {
		switch text[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[start : i+1], true
			}
		}
	}
	return "", false
}

// planTaskSummary 把 phase plan 转成摘要列表（事件 payload 用）
func planTaskSummary(plan []PhaseTaskSummary) []map[string]any {
	out := make([]map[string]any, 0, len(plan))
	for _, t := range plan {
		out = append(out, map[string]any{
			"phase": t.PhaseIdx,
			"class": t.Class,
			"role":  t.Role,
			"goal":  truncate(t.Goal, 80),
		})
	}
	return out
}

// PhaseTaskSummary 用于事件摘要的简化结构
type PhaseTaskSummary struct {
	PhaseIdx int
	Class    string
	Role     string
	Goal     string
}
