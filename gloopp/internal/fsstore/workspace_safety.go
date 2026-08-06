package fsstore

import (
	"context"
	"fmt"
	"strings"
)

// ==================== 安全检查 ====================

type SafetySeverity string

const (
	SeverityHigh   SafetySeverity = "high"
	SeverityMedium SafetySeverity = "medium"
	SeverityLow    SafetySeverity = "low"
)

type SafetyWarning struct {
	Severity SafetySeverity `json:"severity"`
	Category string         `json:"category"` // dangerous_file / mass_delete / path_traversal
	Message  string         `json:"message"`
	Path     string         `json:"path,omitempty"`
}

// 危险文件模式（apply 前提醒）
var dangerousFilePatterns = []struct {
	pattern  string
	severity SafetySeverity
	message  string
}{
	{".env", SeverityHigh, "环境变量文件可能包含敏感信息"},
	{".env.local", SeverityHigh, "本地环境变量文件可能包含敏感信息"},
	{".env.production", SeverityHigh, "生产环境变量文件可能包含密钥"},
	{"id_rsa", SeverityHigh, "SSH 私钥文件"},
	{"id_ed25519", SeverityHigh, "SSH 私钥文件"},
	{".pem", SeverityHigh, "PEM 证书/私钥文件"},
	{".key", SeverityHigh, "密钥文件"},
	{".git/config", SeverityMedium, "Git 配置文件可能包含 token"},
	{".git-credentials", SeverityHigh, "Git 凭证文件"},
	{"secret", SeverityMedium, "疑似密钥/凭证文件"},
	{"password", SeverityMedium, "疑似密码文件"},
	{"token", SeverityMedium, "疑似 token 文件"},
	{"credentials", SeverityHigh, "凭证文件"},
}

// SafetyCheck 检查工作区改动的安全性，返回警告列表。
// 主要检查：
//   - 危险文件（密钥、凭证、环境变量等）
//   - 批量删除（删除文件过多）
//   - 路径穿越（.. 之类）
func (wm *WorkspaceManager) SafetyCheck(ctx context.Context, qid string, q *QuestMeta) ([]SafetyWarning, error) {
	diff, err := wm.ComputeDiff(ctx, qid, q)
	if err != nil {
		return nil, err
	}

	var warnings []SafetyWarning
	deletedCount := 0

	for _, f := range diff.Files {
		// 路径穿越检查
		if strings.Contains(f.Path, "..") {
			warnings = append(warnings, SafetyWarning{
				Severity: SeverityHigh,
				Category: "path_traversal",
				Message:  "文件路径包含 ..，可能存在路径穿越风险",
				Path:     f.Path,
			})
		}

		// 删除文件统计
		if f.Status == "deleted" {
			deletedCount++
		}

		// 危险文件检查（只针对新增或修改的文件，删除的不算危险）
		if f.Status == "added" || f.Status == "modified" {
			for _, pattern := range dangerousFilePatterns {
				if strings.Contains(strings.ToLower(f.Path), strings.ToLower(pattern.pattern)) {
					warnings = append(warnings, SafetyWarning{
						Severity: pattern.severity,
						Category: "dangerous_file",
						Message:  pattern.message,
						Path:     f.Path,
					})
					break // 一个文件只报一次最严重的
				}
			}
		}
	}

	// 批量删除检查
	totalFiles := diff.ChangedFiles
	if totalFiles > 0 && deletedCount > 0 {
		deleteRatio := float64(deletedCount) / float64(totalFiles)
		if deletedCount >= 20 || deleteRatio >= 0.5 {
			sev := SeverityMedium
			if deletedCount >= 50 || deleteRatio >= 0.8 {
				sev = SeverityHigh
			}
			warnings = append(warnings, SafetyWarning{
				Severity: sev,
				Category: "mass_delete",
				Message:  fmt.Sprintf("批量删除 %d 个文件（占变更文件的 %.0f%%）", deletedCount, deleteRatio*100),
			})
		}
	}

	return warnings, nil
}

// HighSeverity 返回是否存在高严重级别的警告。
func HasHighSeverity(warnings []SafetyWarning) bool {
	for _, w := range warnings {
		if w.Severity == SeverityHigh {
			return true
		}
	}
	return false
}
