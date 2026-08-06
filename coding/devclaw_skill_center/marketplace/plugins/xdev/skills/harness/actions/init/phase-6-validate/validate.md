# Phase 6: 验证

> **本 phase 在 `/harness init` 流程中的位置**：第 6 步（最终步骤），必须主 Agent 执行（统一检查所有前序产出）。

**目标**：确认所有 harness 组件已正确安装。

> **注**：原 harness-bootstrap 的 18 点检查清单中，与 `.skills/` 目录和多平台符号链接相关的 6 项已删除——演进能力（debt-fix/doc-fix/evolve/lint-promote）现作为 harness skill 自身的 action 提供，无需安装到目标仓库 `.skills/`；多平台兼容由 xdev CLI 统一处理。

**操作**：

逐项检查并输出验证报告：

1. Hook 配置就绪：`.pre-commit-config.yaml` 存在（路径 A），或已有 git hook 入口包含 harness 检查（路径 B）
2. `.hooks/` 目录下的脚本都有执行权限
3. `.git/hooks/pre-commit` 存在（hooks 已安装）
4. lint 配置存在（如果是对应语言项目）
5. `Makefile` 包含 `install-hooks` target（如有 Makefile），或 `package.json` scripts 中有等效命令
6. `docs/plans/` 目录结构存在
7. `AGENTS.md` 存在
8. `ARCHITECTURE.md` 存在
9. `harness-init.progress` 记录了所有 Phase 为已完成
10. `docs/quality/knowledge-gaps.md` 存在且包含至少一个条目
11. `docs/reference/code-patterns.md` 存在
12. `docs/guidance/debugging-playbook.md` 存在

**输出格式**：

    ## Harness 化验证报告

    | 检查项 | 状态 | 备注 |
    |--------|------|------|
    | hook 配置 | PASS | .pre-commit-config.yaml 存在 / husky 已集成 harness 检查 |
    | hook 脚本权限 | PASS | 3 个脚本均有 +x |
    | hooks 已安装 | PASS | .git/hooks/pre-commit 存在 |
    | ... | ... | ... |

    总计: N/N 通过

**完成后**：
- 更新 `harness-init.progress` 标记所有 Phase 完成
- 提示用户可以删除 `harness-init.progress`（不应入库）
- 建议用户运行 `git add -A && git status` 查看所有变更
