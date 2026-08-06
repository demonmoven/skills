package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
)

// ==================== 自动证据注入 ====================
//
// 机制层改进：法师评审阶段开始前，平台自动运行轻量验证命令，
// 把结果作为 platform_tool_evidence 注入，法师直接可见可信证据。
//
// 设计原则：
// - 只跑 L0 纯读命令，无副作用
// - 只跑快速命令（总耗时 < 30s）
// - 失败不影响流程（打日志，跳过）
// - 项目类型自动检测，挑相关命令跑
// - 结果标注 auto_collected，与 agent 主动调用区分

// autoEvidenceMaxTotalMs 自动证据采集总预算
const autoEvidenceMaxTotalMs = 30 * 1000

// autoEvidenceCommands 返回要自动运行的命令 ID 列表。
// 根据工作区文件自动检测项目类型，挑相关的命令。
//
// 项目类型按优先级互斥判定：go.mod 存在即视为 Go 项目，不再跑 npm
// 命令——即便同目录有 package.json（Go 仓库常带 eslint/prettier 之类
// 的 package.json，或前端子目录）。原因：
//   - npm run build 在 30s L0 预算内经常超时/失败（无 node_modules、
//     缺 build script），产出噪音淹没真正相关的 Go 证据；
//   - 法师仍可通过 Bash + 完整 allowlist 手动跑 npm 命令审前端改动。
//
// 纯 JS 项目（无 go.mod、有 package.json）才跑 npm 证据。
func (e *Engine) autoEvidenceCommands(workspacePath string) []string {
	if workspacePath == "" {
		return nil
	}

	var cmds []string

	// Git 相关：总是尝试（失败也没关系）
	cmds = append(cmds, "git-diff-stat", "git-diff-name-only")

	isGo := fileExists(filepath.Join(workspacePath, "go.mod"))
	isJS := fileExists(filepath.Join(workspacePath, "package.json"))

	// Go 项目优先：只跑 Go 证据，跳过 npm（见函数注释）。
	if isGo {
		cmds = append(cmds, "go-build", "go-vet", "go-mod-verify")
		return cmds
	}

	// 纯 JS/Node 项目
	if isJS {
		cmds = append(cmds, "npm-build-check", "npm-audit")
	}

	return cmds
}

// collectAutoEvidence 自动运行验证命令，返回格式化的证据字符串。
// 证据格式与 platform_tool_evidence 一致，可直接拼接到 artifact 中。
func (e *Engine) collectAutoEvidence(ctx context.Context, qid, sid string, q *fsstore.QuestMeta) string {
	text, _ := e.collectAutoEvidencePack(ctx, qid, sid, q)
	return text
}

func (e *Engine) collectAutoEvidencePack(ctx context.Context, qid, sid string, q *fsstore.QuestMeta) (string, []fsstore.EvidenceRef) {
	if q == nil || q.WorkspacePath == "" {
		return "", nil
	}

	cmds := e.autoEvidenceCommands(q.WorkspacePath)
	if len(cmds) == 0 {
		return "", nil
	}

	// 总超时预算
	totalCtx, cancel := context.WithTimeout(ctx, time.Duration(autoEvidenceMaxTotalMs)*time.Millisecond)
	defer cancel()

	var evidence []string
	var refs []fsstore.EvidenceRef
	evidence = append(evidence, "=== auto_collected_platform_evidence ===")
	evidence = append(evidence, fmt.Sprintf("quest_id: %s", qid))
	evidence = append(evidence, fmt.Sprintf("collected_at_ms: %d", fsstore.NowMs()))
	evidence = append(evidence, "source: gloop_platform_auto_evidence_injection")
	evidence = append(evidence, "")

	ran := 0
	for _, cmdID := range cmds {
		if totalCtx.Err() != nil {
			evidence = append(evidence, fmt.Sprintf("- %s: skipped (auto_evidence_budget_exhausted)", cmdID))
			continue
		}

		result, err := e.RunAllowedCommand(totalCtx, qid, sid, cmdID, nil)
		if err != nil {
			evidence = append(evidence, fmt.Sprintf("- %s: error (%s)", cmdID, truncate(err.Error(), 120)))
			continue
		}

		ran++
		evidence = append(evidence, fmt.Sprintf("- command: %s", cmdID))
		evidence = append(evidence, fmt.Sprintf("  exit_code: %d", result.ExitCode))
		evidence = append(evidence, fmt.Sprintf("  duration_ms: %d", result.DurationMs))
		if result.TimedOut {
			evidence = append(evidence, "  timed_out: true")
		}
		// 只放 stdout 摘要，太长的截断
		stdout := strings.TrimSpace(result.Stdout)
		if stdout == "" {
			evidence = append(evidence, "  stdout: (empty)")
		} else if lines := strings.Split(stdout, "\n"); len(lines) <= 10 {
			evidence = append(evidence, "  stdout:")
			for _, line := range lines {
				evidence = append(evidence, "    "+line)
			}
		} else {
			// 超过 10 行：前 5 行 + 省略 + 后 3 行
			evidence = append(evidence, "  stdout (truncated):")
			for i := 0; i < 5; i++ {
				evidence = append(evidence, "    "+lines[i])
			}
			evidence = append(evidence, "    ...")
			for i := len(lines) - 3; i < len(lines); i++ {
				evidence = append(evidence, "    "+lines[i])
			}
		}
		if stderr := strings.TrimSpace(result.Stderr); stderr != "" {
			if lines := strings.Split(stderr, "\n"); len(lines) <= 5 {
				evidence = append(evidence, "  stderr:")
				for _, line := range lines {
					evidence = append(evidence, "    "+line)
				}
			} else {
				evidence = append(evidence, fmt.Sprintf("  stderr: (%d lines, truncated)", len(lines)))
			}
		}
		refs = append(refs, fsstore.EvidenceRef{
			ID:          fmt.Sprintf("ev_%s_%d", cmdID, ran),
			Kind:        "l0_command",
			TrustTier:   "T0",
			Source:      "gloop_platform_auto_evidence_injection",
			CommandID:   cmdID,
			ExitCode:    result.ExitCode,
			DurationMs:  result.DurationMs,
			TimedOut:    result.TimedOut,
			Snippet:     autoEvidenceSnippet(result.Stdout, result.Stderr),
			CreatedAtMs: fsstore.NowMs(),
		})
		evidence = append(evidence, "")
	}

	evidence = append(evidence, fmt.Sprintf("=== end_auto_evidence (%d commands run) ===", ran))

	return strings.Join(evidence, "\n"), refs
}

// injectAutoEvidence 把自动采集的证据合并到 artifact 的 PlatformToolEvidence 中。
// 自动证据放在前面，方便法师先看到。
func injectAutoEvidence(artifact *fsstore.PhaseReviewArtifact, autoEvidence string) {
	if autoEvidence == "" {
		return
	}
	artifact.Evidence.AutoCollected = autoEvidence
	if artifact.PlatformToolEvidence == "" {
		artifact.PlatformToolEvidence = autoEvidence
	} else {
		artifact.PlatformToolEvidence = autoEvidence + "\n\n" + artifact.PlatformToolEvidence
	}
}

func autoEvidenceSnippet(stdout, stderr string) string {
	text := strings.TrimSpace(stdout)
	if text == "" {
		text = strings.TrimSpace(stderr)
	}
	return truncate(text, 500)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
