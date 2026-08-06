# Phase 2: Pre-commit Hooks 安装

> **本 phase 在 `/harness init` 流程中的位置**：第 2 步，可委托 subagent 执行。依赖 Phase 1 产出的 `harness-init-analysis.md`。

**目标**：安装提交前自动检查，建立代码质量的第一道防线。

Hook 的选择逻辑参见 `<skill_dir>/actions/init/phase-2-install-hooks/prompts/hook-selection.md`。

**重要：Agent 必须先根据 Phase 1 的检测结果确定 hook 管理路径。** 仓库可能已有 git hook 管理机制（husky、lefthook、simple-git-hooks 等），此时应利用已有机制而非强制引入新工具。

## 2.0 确定 Hook 管理路径

根据 Phase 1 的 `detect_stack.sh` 输出中 `toolchain.hook_manager` 字段选择路径：

| 检测结果 | 路径 | 说明 |
|---|---|---|
| `hook_manager: "husky"` | **路径 B: 集成到已有 husky** | 在已有的 git hooks 脚本中添加检查逻辑 |
| `hook_manager: "pre-commit"` | **路径 A: 使用 pre-commit** | 标准 `.pre-commit-config.yaml` 方式 |
| `hook_manager: "none"` | **由用户选择** | 推荐 pre-commit (Python)，但也可选 husky 或其他方案 |

---

## 路径 A: 使用 pre-commit (Python) 管理

> 当仓库**未使用**其他 hook 管理工具，或已有 `.pre-commit-config.yaml` 时使用此路径。

### 2a-A. 组装 .pre-commit-config.yaml

根据 Phase 1 的适配方案，从 `<skill_dir>/actions/init/phase-2-install-hooks/assets/pre-commit-config/` 中选择配置片段。每个片段是一个独立的 YAML 数组元素，Agent 将选中的片段合并为完整的 `.pre-commit-config.yaml`。

| 配置片段 | 选择条件 | 说明 |
|---|---|---|
| `base.yaml` | 几乎所有项目 | 文件卫生（trailing-whitespace、end-of-file-fixer 等） |
| `security.yaml` | 有敏感信息风险时 | gitleaks 密钥泄露检测 |
| `go.yaml` | 检测到 Go | go-fmt、go-build、golangci-lint 等 |
| `python.yaml` | 检测到 Python | ruff check、mypy type-check 等 |
| `rust.yaml` | 检测到 Rust | cargo fmt、cargo clippy 等 |
| `frontend-biome.yaml` | 检测到前端 + Biome | biome check + tsc |
| `frontend-eslint.yaml` | 检测到前端 + ESLint | eslint + tsc |
| `ai-guard.yaml` | 团队使用 AI 编码工具时 | 占位符/幻觉标记检测、文件长度、活跃计划 |

### 2b-A. 安装 hooks

运行 `pre-commit install --install-hooks`（或 `prek install --install-hooks`）。

---

## 路径 B: 集成到已有 Hook 管理工具（husky / lefthook / 等）

> 当仓库**已使用** husky、lefthook、simple-git-hooks 等 Node.js 生态的 hook 管理工具时使用此路径。**不引入 pre-commit (Python)**，避免多套 hook 管理工具共存的复杂性。

核心思路：将需要的检查逻辑以 shell 脚本形式添加到仓库的 `.hooks/` 目录，然后在已有的 git hooks 脚本（如 `git-hooks/pre-commit`、`.husky/pre-commit`）中调用这些脚本。

### 2a-B. 识别已有 hook 入口

Agent 需要找到已有 git hooks 的入口文件：

| Hook 管理工具 | 典型入口位置 |
|---|---|
| husky v8+ | `.husky/pre-commit`、`.husky/commit-msg` |
| husky (自定义目录) | `git-hooks/pre-commit`（通过 `husky install git-hooks` 指定） |
| lefthook | `lefthook.yml` 配置文件 |
| simple-git-hooks | `package.json` 中的 `simple-git-hooks` 字段 |

### 2b-B. 添加 harness hook 脚本

将 `<skill_dir>/actions/init/phase-2-install-hooks/assets/hook-scripts/` 中需要的脚本复制到目标仓库的 `.hooks/` 目录。根据需求选择：

| 脚本 | 选择条件 | 说明 |
|---|---|---|
| `check-file-length.sh` | 团队使用 AI 编码工具 | 文件行数上限检查，上限基于 Phase 1 分析 |
| `check-active-plans.sh` | 使用 ExecPlan 工作流 | 活跃计划完成度检查（pre-push） |
| `no-commit-to-main.sh` | 所有项目 | 禁止在 main/master 分支直接 commit |

### 2c-B. 集成到已有 git hooks

在已有的 pre-commit hook 脚本中，**追加** harness 检查逻辑。Agent 根据已有脚本的风格来添加，确保：

1. **不破坏已有逻辑**：在已有命令（如 lint-staged）之前或之后添加
2. **保持风格一致**：如果已有脚本用 `set -e`，新增部分也用
3. **可跳过**：通过环境变量（如 `SKIP_HARNESS_HOOKS=1`）支持临时绕过

典型的集成方式（以 husky 为例）：

```bash
# === harness hooks (由 harness 添加) ===
# AI guard: 占位符检测（对暂存文件）
git diff --cached --name-only --diff-filter=ACM | grep -E '\.(ts|tsx|js|jsx)$' | while read f; do
  if grep -nE '(//|#)\s*(TODO:?\s*implement|PLACEHOLDER|STUB|HACK|XXX|FIXME)' "$f" 2>/dev/null; then
    echo "ERROR: $f contains placeholder code"
    exit 1
  fi
done

# AI guard: 文件长度检查（对暂存文件）
if [ "${SKIP_HARNESS_HOOKS:-0}" != "1" ]; then
  .hooks/check-file-length.sh $(git diff --cached --name-only --diff-filter=ACM | grep -E '\.(ts|tsx|js|jsx)$')
fi
# === end harness hooks ===
```

对于 AI 幻觉标记检测，可以同样方式添加 pygrep 等效的 `grep -nE` 检查。

### 2d-B. 配置 lint-staged（可选）

如果仓库已有 lint-staged 配置，可以将部分检查逻辑集成到 lint-staged 中，利用其增量检查能力（只检查暂存文件）：

```json
{
  "**/*.{ts,tsx,js,jsx}": [
    ".hooks/check-file-length.sh"
  ]
}
```

这比在 pre-commit 脚本中手动过滤暂存文件更优雅。

---

## 通用步骤（两个路径共用）

### 2e. Lint 配置

**核心思路**：`<skill_dir>/actions/init/phase-2-install-hooks/prompts/lint-strategy.md` 定义了与语言无关的 lint 关注领域（错误处理、复杂度、未使用代码等），以及每种语言的推荐工具和规则映射。Agent 参考该指南，为目标仓库选择合适的 lint 配置。

- **已有 lint 配置**：不覆盖。检查是否缺少关键的 AI 友好规则，建议补充
- **无 lint 配置**：参考 `<skill_dir>/actions/init/phase-2-install-hooks/assets/lint-examples/` 下的示例为基线生成，但必须根据仓库特征调整

**适配要求**：

- **文件长度上限**：不能直接用默认 600 行，基于 Phase 1 的文件长度分布分析选择合理值
- **语言 hook 互斥**：只安装目标仓库实际使用的语言的 hook，不同语言之间不混装
- **已有 CI 中的 lint**：向用户说明 hooks（本地快速反馈）与 CI（最终门禁）的关系

### 2f. Makefile / Taskfile

如果仓库已有 `Makefile`，将 harness 相关 target（`install-hooks`、`lint`）作为建议追加项展示给用户。如果没有，以 `<skill_dir>/actions/init/phase-2-install-hooks/assets/makefile/Makefile.base` 为基线生成，但需根据仓库实际的构建、测试命令调整。

**注意**：如果仓库使用 husky 路径且 `package.json` 中已有 `"prepare": "husky install ..."` 脚本，Makefile 可能不是必需的。此时考虑在 `package.json` 的 `scripts` 中添加 `lint:harness` 命令代替。

### 2g. .gitignore

将 `<skill_dir>/actions/init/phase-2-install-hooks/assets/gitignore/agent-rules.gitignore` 的内容追加到目标仓库 `.gitignore`（检查避免重复）。

**汇报**：安装了哪些 hooks 和配置（使用了哪个路径），跳过了哪些，做了哪些适配调整及原因。
