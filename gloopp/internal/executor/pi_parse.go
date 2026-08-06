package executor

import (
	"encoding/json"
	"strings"
)

type piParseResult struct {
	Content    string
	StopReason string
	UsageIn    int64
	UsageOut   int64
	IsError    bool
}

func parsePiJSON(raw string) piParseResult {
	var out piParseResult
	scanner := newJSONLineScanner(raw)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}
		typ, _ := obj["type"].(string)
		switch typ {
		case "session":
			continue
		case "message_end", "turn_end":
			if msg, ok := obj["message"].(map[string]any); ok {
				if text := piMessageText(msg); text != "" {
					out.Content = text
				}
				if sr, ok := msg["stopReason"].(string); ok && sr != "" {
					out.StopReason = sr
				}
				if usage, ok := msg["usage"].(map[string]any); ok {
					out.UsageIn += int64FromAny(usage["input"])
					out.UsageOut += int64FromAny(usage["output"])
				}
			}
		case "agent_end":
			if messages, ok := obj["messages"].([]any); ok {
				for _, item := range messages {
					if msg, ok := item.(map[string]any); ok {
						if text := piMessageText(msg); text != "" {
							out.Content = text
						}
					}
				}
			}
		case "error", "extension_error":
			out.IsError = true
			if msg, ok := obj["message"].(string); ok && out.Content == "" {
				out.Content = msg
			}
		}
	}
	if out.Content == "" && out.IsError {
		out.Content = strings.TrimSpace(raw)
	}
	return out
}

func piMessageText(msg map[string]any) string {
	role, _ := msg["role"].(string)
	if role != "" && role != "assistant" {
		return ""
	}
	content := msg["content"]
	switch c := content.(type) {
	case string:
		return c
	case []any:
		var sb strings.Builder
		for _, item := range c {
			part, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if typ, _ := part["type"].(string); typ != "" && typ != "text" {
				continue
			}
			if text, ok := part["text"].(string); ok {
				sb.WriteString(text)
			}
		}
		return sb.String()
	default:
		return ""
	}
}
