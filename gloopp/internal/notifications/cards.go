package notifications

import (
	"fmt"
	"strings"

	"code.byted.org/lihuanyu.0w0/gloop/internal/model"
)

// 卡片模板色：对应飞书卡片 header.template
// green=成功, red=失败/错误, yellow=警告/待审核, orange=阻塞, blue=信息
const (
	cardTplGreen  = "green"
	cardTplRed    = "red"
	cardTplYellow = "yellow"
	cardTplOrange = "orange"
	cardTplBlue   = "blue"
)

// QuestInfo 是组装通知卡片需要的 quest 基本信息。
// 由调用方（事件订阅器）从存储层读取后传入。
type QuestInfo struct {
	ID          string // quest id
	ShortID     string // 短 id
	Title       string // quest 标题（query）
	Status      string // 状态文本
	ReworkCount int    // 返工次数
	DurationMs  int64  // 耗时毫秒（0 表示未知）
	Comment     string // 补充说明（失败原因、阻塞原因、评审摘要等）
	Impact      model.ImpactSummary // HOTL: agent 主动声明的影响（波及什么）。非空时影响通知优先用它
}

// CardAction 是卡片底部的操作按钮。
type CardAction struct {
	Label string // 按钮文字
	URL   string // 跳转链接
	Type  string // primary / default / danger
}

// buildQuestCard 构造一张 quest 相关的通用通知卡片。
// headerTemplate: 飞书卡片 header.template 颜色
// headerTitle: 标题（带 emoji 前缀，如 "✅ Quest 完成"）
// summary: 卡片正文（lark_md 格式）
// info: quest 基本信息
// baseURL: dashboard base URL（用于拼接详情链接）
// actions: 底部操作按钮（可选）
func buildQuestCard(
	headerTemplate, headerTitle string,
	summary string,
	info QuestInfo,
	baseURL string,
	actions []CardAction,
) map[string]any {
	// 卡片元素列表
	var elements []map[string]any

	// 元信息区：两列网格布局
	// 第一行：状态 + 耗时
	// 第二行（有返工/短ID时）：短ID + 返工次数
	var firstRow []map[string]any
	var secondRow []map[string]any

	firstRow = append(firstRow, metaCell("状态", info.Status))
	if info.DurationMs > 0 {
		firstRow = append(firstRow, metaCell("耗时", formatDuration(info.DurationMs)))
	}

	hasShortID := info.ShortID != ""
	hasRework := info.ReworkCount > 0
	if hasShortID || hasRework {
		if hasShortID {
			secondRow = append(secondRow, metaCell("编号", "#"+info.ShortID))
		}
		if hasRework {
			secondRow = append(secondRow, metaCell("返工", fmt.Sprintf("%d 次", info.ReworkCount)))
		}
	}

	// 把两行拼成一个 column_set
	var columns []map[string]any
	if len(firstRow) > 0 {
		for _, cell := range firstRow {
			columns = append(columns, cell)
		}
	}
	if len(secondRow) > 0 {
		for _, cell := range secondRow {
			columns = append(columns, cell)
		}
	}
	if len(columns) > 0 {
		// 两列布局：每 2 个 cell 组成一行
		// 飞书卡片用 column_set，flex_mode = "none" 时是等分列
		// 这里简化处理：两个一行，依次排列
		for i := 0; i < len(columns); i += 2 {
			rowCols := columns[i:min(i+2, len(columns))]
			elements = append(elements, map[string]any{
				"tag":       "column_set",
				"flex_mode": "none",
				"columns":   rowCols,
			})
		}
	}

	// 摘要区（如果有）
	if strings.TrimSpace(summary) != "" {
		displaySummary := summary
		truncated := false
		const maxSummaryRunes = 400
		runes := []rune(summary)
		if len(runes) > maxSummaryRunes {
			displaySummary = string(runes[:maxSummaryRunes]) + "..."
			truncated = true
		}

		elements = append(elements, map[string]any{"tag": "hr"})
		elements = append(elements, map[string]any{
			"tag": "div",
			"text": map[string]any{
				"tag":     "lark_md",
				"content": displaySummary,
			},
		})

		// 截断了就加一个"查看详情"的补充提示
		if truncated && baseURL != "" {
			elements = append(elements, map[string]any{
				"tag": "div",
				"text": map[string]any{
					"tag":     "lark_md",
					"content": fmt.Sprintf("[查看完整摘要 →](%s)", questDetailURL(baseURL, info.ID)),
				},
			})
		}
	}

	// 操作按钮区（如果有）
	if len(actions) > 0 {
		elements = append(elements, map[string]any{"tag": "hr"})

		actionBtns := []map[string]any{}
		for _, a := range actions {
			btnType := a.Type
			if btnType == "" {
				btnType = "default"
			}
			actionBtns = append(actionBtns, map[string]any{
				"tag":  "button",
				"text": map[string]any{"tag": "plain_text", "content": a.Label},
				"type": btnType,
				"url":  a.URL,
			})
		}
		elements = append(elements, map[string]any{
			"tag":     "action",
			"actions": actionBtns,
		})
	}

	// 底部签名：Gloop 标识
	signature := "*来自 Gloop 自动化助手*"
	if baseURL != "" {
		signature += " · [打开控制台](" + baseURL + "/dashboard)"
	}
	elements = append(elements, map[string]any{"tag": "hr"})
	elements = append(elements, map[string]any{
		"tag": "div",
		"text": map[string]any{
			"tag":     "lark_md",
			"content": signature,
		},
	})

	// 完整卡片
	card := map[string]any{
		"config": map[string]any{
			"wide_screen_mode": true,
		},
		"header": map[string]any{
			"title": map[string]any{
				"tag":     "plain_text",
				"content": "Gloop · " + headerTitle,
			},
			"template": headerTemplate,
		},
		"elements": elements,
	}

	return card
}

// BuildQuestSuccessCard 构造 quest 成功通知卡片。
func BuildQuestSuccessCard(info QuestInfo, comment string, baseURL string) map[string]any {
	title := fmt.Sprintf("✅ 委托完成：%s", truncate(info.Title, 30))
	status := "已完成"
	summary := ""
	if comment != "" {
		summary = "**评审摘要：**\n" + formatSummaryBody(comment)
	}

	actions := []CardAction{
		{Label: "查看详情", URL: questDetailURL(baseURL, info.ID), Type: "primary"},
	}

	return buildQuestCard(cardTplGreen, title, summary,
		QuestInfo{
			ID:          info.ID,
			ShortID:     info.ShortID,
			Title:       info.Title,
			Status:      status,
			ReworkCount: info.ReworkCount,
			DurationMs:  info.DurationMs,
		},
		baseURL, actions)
}

// BuildQuestFailedCard 构造 quest 失败通知卡片。
func BuildQuestFailedCard(info QuestInfo, reason string, baseURL string) map[string]any {
	title := fmt.Sprintf("❌ 委托失败：%s", truncate(info.Title, 30))
	status := "已失败"
	summary := ""
	if reason != "" {
		summary = "**失败原因：**\n" + formatSummaryBody(reason)
	}

	actions := []CardAction{
		{Label: "查看详情", URL: questDetailURL(baseURL, info.ID), Type: "danger"},
	}

	return buildQuestCard(cardTplRed, title, summary,
		QuestInfo{
			ID:          info.ID,
			ShortID:     info.ShortID,
			Title:       info.Title,
			Status:      status,
			ReworkCount: info.ReworkCount,
			DurationMs:  info.DurationMs,
		},
		baseURL, actions)
}

// BuildQuestBlockedCard 构造 quest 阻塞通知卡片。
func BuildQuestBlockedCard(info QuestInfo, reason string, baseURL string) map[string]any {
	title := fmt.Sprintf("⚠️ Quest 阻塞：%s", truncate(info.Title, 30))
	status := "已阻塞"
	summary := ""
	if reason != "" {
		summary = "**阻塞原因：**\n" + reason
	}

	actions := []CardAction{
		{Label: "查看详情", URL: questDetailURL(baseURL, info.ID), Type: "primary"},
		{Label: "恢复执行", URL: questResumeURL(baseURL, info.ID), Type: "default"},
	}

	return buildQuestCard(cardTplOrange, title, summary,
		QuestInfo{
			ID:          info.ID,
			ShortID:     info.ShortID,
			Title:       info.Title,
			Status:      status,
			ReworkCount: info.ReworkCount,
			DurationMs:  info.DurationMs,
		},
		baseURL, actions)
}

// BuildQuestReviewCard 构造 quest 待审核通知卡片。
func BuildQuestReviewCard(info QuestInfo, comment string, baseURL string) map[string]any {
	title := fmt.Sprintf("🔍 委托待审核：%s", truncate(info.Title, 30))
	status := "待用户审核"
	summary := ""
	if comment != "" {
		summary = "**评审摘要：**\n" + formatSummaryBody(comment)
	}

	actions := []CardAction{
		{Label: "✅ 批准", URL: questApproveURL(baseURL, info.ID), Type: "primary"},
		{Label: "🔄 返工", URL: questReworkURL(baseURL, info.ID), Type: "default"},
		{Label: "❌ 拒绝", URL: questRejectURL(baseURL, info.ID), Type: "danger"},
	}

	return buildQuestCard(cardTplYellow, title, summary,
		QuestInfo{
			ID:          info.ID,
			ShortID:     info.ShortID,
			Title:       info.Title,
			Status:      status,
			ReworkCount: info.ReworkCount,
			DurationMs:  info.DurationMs,
		},
		baseURL, actions)
}

// BuildQuestImpactCard 构造"影响通知"卡片（HOTL v0.2 环 4）。
//
// 与 BuildQuestSuccessCard 的区别：
//   - 自主闭环的 quest 不需要人审批，所以底部只有"查看详情"，没有 approve/rework/reject
//   - 标题用"影响"而非"完成"，强调"感知变化"而非"等你处理"
//   - summary 显示 quest 做了什么 + 影响范围（文件改动等）
func BuildQuestImpactCard(info QuestInfo, summary string, baseURL string) map[string]any {
	title := fmt.Sprintf("✅ 已自主闭环：%s", truncate(info.Title, 30))
	status := "已自动完成"
	body := ""
	if strings.TrimSpace(summary) != "" {
		body = "**影响摘要：**\n" + formatSummaryBody(summary)
	}

	actions := []CardAction{
		{Label: "查看影响", URL: questDetailURL(baseURL, info.ID), Type: "default"},
	}

	return buildQuestCard(cardTplGreen, title, body,
		QuestInfo{
			ID:          info.ID,
			ShortID:     info.ShortID,
			Title:       info.Title,
			Status:      status,
			ReworkCount: info.ReworkCount,
			DurationMs:  info.DurationMs,
		},
		baseURL, actions)
}

// questDetailURL 拼接 quest 详情页链接。
func questDetailURL(baseURL, qid string) string {
	if baseURL == "" {
		return ""
	}
	return baseURL + "/quests/" + qid
}

// 下面几个 URL 先用占位路径，前端实际有了再改。

func questResumeURL(baseURL, qid string) string {
	if baseURL == "" {
		return ""
	}
	return baseURL + "/quests/" + qid + "?action=resume"
}

func questApproveURL(baseURL, qid string) string {
	if baseURL == "" {
		return ""
	}
	return baseURL + "/quests/" + qid + "?action=approve"
}

func questReworkURL(baseURL, qid string) string {
	if baseURL == "" {
		return ""
	}
	return baseURL + "/quests/" + qid + "?action=rework"
}

func questRejectURL(baseURL, qid string) string {
	if baseURL == "" {
		return ""
	}
	return baseURL + "/quests/" + qid + "?action=reject"
}

// metaCell 构造一个元信息单元格（用于 column_set 的一列）。
func metaCell(label, value string) map[string]any {
	return map[string]any{
		"tag":            "column",
		"width":          "weighted",
		"weight":         1,
		"vertical_align": "center",
		"elements": []map[string]any{
			{
				"tag": "div",
				"text": map[string]any{
					"tag":     "lark_md",
					"content": fmt.Sprintf("**%s：** %s", label, value),
				},
			},
		},
	}
}

func formatDuration(ms int64) string {
	if ms <= 0 {
		return "未知"
	}
	seconds := ms / 1000
	minutes := seconds / 60
	hours := minutes / 60

	if hours > 0 {
		return fmt.Sprintf("%d小时%d分", hours, minutes%60)
	}
	if minutes > 0 {
		return fmt.Sprintf("%d分%d秒", minutes, seconds%60)
	}
	if seconds > 0 {
		return fmt.Sprintf("%d秒", seconds)
	}
	return "不足1秒"
}

// formatSummaryBody 格式化摘要正文，提升可读性。
// - 识别"### 标题"形式的 Markdown 三级标题，转为飞书粗体标题
// - 兼容"xxx:"形式的旧格式小节标题并加粗（如 evidence:、confidence:）
// - 保留原有换行和段落结构
func formatSummaryBody(text string) string {
	lines := strings.Split(text, "\n")
	var out []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 1. Markdown 三级标题：### 标题内容
		if strings.HasPrefix(trimmed, "### ") {
			title := strings.TrimSpace(strings.TrimPrefix(trimmed, "### "))
			if title != "" {
				// 前后加空行 + 粗体，在飞书卡片里呈现小节标题效果
				if len(out) > 0 && out[len(out)-1] != "" {
					out = append(out, "")
				}
				out = append(out, "**"+title+"**")
				continue
			}
		}

		// 2. 兼容旧格式：小写字母/下划线组成的标签: 内容
		if idx := strings.Index(trimmed, ":"); idx > 0 && idx < 40 {
			label := trimmed[:idx]
			isSectionLabel := len(label) >= 2
			for _, r := range label {
				if !((r >= 'a' && r <= 'z') || r == '_' || (r >= '0' && r <= '9')) {
					isSectionLabel = false
					break
				}
			}
			if isSectionLabel {
				rest := strings.TrimSpace(trimmed[idx+1:])
				if rest != "" {
					out = append(out, fmt.Sprintf("**%s：** %s", label, rest))
				} else {
					out = append(out, fmt.Sprintf("**%s：**", label))
				}
				continue
			}
		}

		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// truncate 按 rune 截断字符串，超出部分用 ... 代替。
// 安全处理中文、emoji 等多字节字符。
func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}
