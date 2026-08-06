# harness-init.progress 规范

本文档定义 `harness-init.progress` 文件的格式和使用规则。该文件用于追踪 harness-bootstrap 在目标仓库中的执行进度，支持中断恢复。

## 文件位置

`harness-init.progress` 存放在目标仓库的根目录下。该文件应被 `.gitignore` 忽略（harness-bootstrap 会自动添加忽略规则）。

## 文件格式

采用 Markdown checkbox 列表格式：

    # harness-init progress
    # 创建时间: 2026-03-31 10:00:00+08:00
    # 目标仓库: /path/to/repo
    # 检测到的技术栈: python, node, frontend

    - [x] (2026-03-31 10:01:00+08:00) Phase 1: 仓库理解与环境检测
    - [x] (2026-03-31 10:05:00+08:00) Phase 2: Pre-commit Hooks 安装
    - [x] (2026-03-31 10:08:00+08:00) Phase 3: 文档骨架与规则模板
    - [ ] Phase 4: 文档生成（AGENTS.md 体系）
    - [ ] Phase 5: 架构文档（ARCHITECTURE.md）
    - [ ] Phase 6: Skill 安装
    - [ ] Phase 7: 多平台兼容
    - [ ] Phase 8: 验证

    ## 适配方案摘要

    - 主语言: Python 3.12
    - Lint 工具: ruff (已有配置，需补充 AI 友好规则)
    - 文件长度上限: 500 行（P95=420）
    - Hook 选择: base + security + python + ai-guard
    - Skill 选择: exec-plan + harness-debt-fix

    ## 详细日志

    ### Phase 1: 仓库理解与环境检测
    - 检测到 Python 项目（pyproject.toml 存在，使用 poetry）
    - 检测到 React 前端（apps/web/package.json）
    - 已有 ruff 配置，缺少 PLR0915 和 C901
    - 已有 GitHub Actions CI，包含 pytest + ruff check
    - 未检测到已有 AGENTS.md

    ### Phase 2: Pre-commit Hooks 安装
    - 安装了 pre-commit hooks（base + security + python + ai-guard）
    - 跳过 frontend-eslint（前端 CI 中已有 eslint 步骤，避免重复）
    - 向 ruff 配置补充了 PLR0915 和 C901 规则
    - 文件长度上限设为 500 行

## 使用规则

1. **创建时机**：Phase 1 完成后立即创建
2. **更新时机**：每个 Phase 完成后立即更新，标记 `[x]` 并写入时间戳
3. **恢复逻辑**：Agent 读取该文件后，跳过已完成的 Phase，从第一个未完成的 Phase 继续
4. **幂等性**：每个 Phase 的操作都必须是幂等的
5. **清理**：所有 Phase 完成后，提示用户可以删除该文件
