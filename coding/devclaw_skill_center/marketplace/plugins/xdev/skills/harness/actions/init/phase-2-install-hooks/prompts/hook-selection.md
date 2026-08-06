# Hook 目录与选择指南

本文档列出 harness-bootstrap 可安装的所有 pre-commit hook 类别，说明每个 hook 的 **目的**、**适用条件** 和 **不适用条件**，帮助 Agent 为目标仓库做出正确选择。

## Hook 分类

### 类别 1：文件卫生（base.yaml）

**目的**：确保提交的文件格式一致，避免合并冲突和格式噪音。

**适用条件**：几乎所有项目。

**包含的 hook**：

| Hook | 作用 | 为什么需要 |
|---|---|---|
| trailing-whitespace | 删除行尾空格 | 避免无意义的 diff 噪音 |
| end-of-file-fixer | 确保文件以换行结尾 | POSIX 标准，避免 git diff 警告 |
| check-yaml | 校验 YAML 语法 | CI 配置、K8s manifest 等 |
| check-json | 校验 JSON 语法 | package.json、tsconfig 等 |
| check-toml | 校验 TOML 语法 | Cargo.toml、pyproject.toml 等 |
| check-added-large-files | 拦截大文件（默认 500KB） | 防止误提交 binary、model 文件 |
| check-merge-conflict | 检查未解决的冲突标记 | 防止 `<<<<<<<` 进入代码库 |
| mixed-line-ending | 统一换行符为 LF | 跨平台一致性 |
| check-symlinks | 检查断裂的符号链接 | 防止引用不存在的路径 |

**适配项**：
- `check-added-large-files` 的阈值可调整（ML 项目可能需要放大到 5MB）
- 如果仓库含大量 YAML/JSON/TOML 以外的格式，对应 check 可省略

---

### 类别 2：密钥泄露检测（security.yaml）

**目的**：防止 API Key、密码、Token 等敏感信息进入代码库。AI Agent 尤其容易在代码中内嵌示例 API Key。

**适用条件**：有敏感信息风险的项目（几乎所有项目都适用）。

**不适用条件**：纯内部 library 且确认无任何凭据的项目（极少见）。

**工具**：gitleaks

**适配项**：
- 如果仓库有合法的 mock token 用于测试，需配置 `.gitleaksignore` 排除

---

### 类别 3：Go 语言质量（go.yaml）

**目的**：对 Go 代码执行格式化、编译检查、依赖一致性和 lint。

**适用条件**：检测到 `go.mod`。

**包含的 hook**：

| Hook | 作用 | 脚本 |
|---|---|---|
| no-commit-to-main | 禁止在 main/master 分支直接 commit | `.hooks/no-commit-to-main.sh` |
| go-fmt | `gofmt -s` 格式化 + `interface{} -> any` 替换 | `.hooks/go-fmt.sh` |
| go-build | `go build ./...` 编译检查 | 直接命令 |
| go-mod-tidy | `go mod tidy` 依赖一致性 | `.hooks/go-mod-tidy.sh` |
| golangci-lint | 综合 lint 检查 | 直接命令 |

**适配项**：
- monorepo 有多个 `go.mod` 时，脚本需遍历所有 module 目录
- golangci-lint 规则参见 `lint-strategy-guide.md`

---

### 类别 4：Python 语言质量（python.yaml）

**目的**：对 Python 代码执行格式化、类型检查和 lint。

**适用条件**：检测到 `pyproject.toml`、`setup.py`、`requirements.txt` 或 `Pipfile`。

**包含的 hook**：

| Hook | 作用 | 说明 |
|---|---|---|
| no-commit-to-main | 禁止在 main/master 分支直接 commit | 通用脚本 |
| ruff-check | `ruff check --fix` lint + 自动修复 | 替代 flake8+isort+pyupgrade |
| ruff-format | `ruff format` 格式化 | 替代 black |
| mypy | `mypy` 类型检查 | 可选，需项目已配置 mypy |

**适配项**：
- 如果项目使用 black 而非 ruff format，应保留 black
- mypy 仅在项目已有 type annotations 时安装，不强制引入
- 如果项目使用 poetry/conda/uv，需确认 hook 运行环境

---

### 类别 5：Rust 语言质量（rust.yaml）

**目的**：对 Rust 代码执行格式化和 lint。

**适用条件**：检测到 `Cargo.toml`。

**包含的 hook**：

| Hook | 作用 | 说明 |
|---|---|---|
| no-commit-to-main | 禁止在 main/master 分支直接 commit | 通用脚本 |
| cargo-fmt | `cargo fmt --check` 格式化检查 | 直接命令 |
| cargo-clippy | `cargo clippy -- -D warnings` lint | 所有 warning 视为 error |

**适配项**：
- workspace 有多个 crate 时，命令需要 `--workspace` 参数
- nightly-only 的 clippy lint 不强制启用

---

### 类别 6：前端质量 — Biome（frontend-biome.yaml）

**目的**：对前端代码执行 lint + 格式化 + 类型检查。

**适用条件**：检测到 `biome.json` 或 `biome.jsonc`。

**包含的 hook**：

| Hook | 作用 |
|---|---|
| biome-check | `biome check --fix` lint + format 一体化 |
| tsc-check | `tsc --noEmit` TypeScript 类型检查 |

**不适用条件**：项目使用 ESLint + Prettier 组合时，应使用 `frontend-eslint.yaml`。

---

### 类别 7：前端质量 — ESLint（frontend-eslint.yaml）

**目的**：对前端代码执行 ESLint lint + 类型检查。

**适用条件**：检测到 `.eslintrc.*` 或 `eslint.config.*`。

**包含的 hook**：

| Hook | 作用 |
|---|---|
| eslint | `eslint --fix` lint + 自动修复 |
| tsc-check | `tsc --noEmit` TypeScript 类型检查 |

---

### 类别 8：AI Guard（ai-guard.yaml）

**目的**：拦截 AI Agent 的典型质量问题——占位符代码、幻觉标记、超长文件、未完成的执行计划。

**适用条件**：团队使用 AI 编码工具（Claude Code、Codex、Copilot、Cursor 等）。

**不适用条件**：团队完全不使用 AI 编码工具时不安装（需向用户确认）。

**包含的 hook**：

| Hook | 作用 | 类型 |
|---|---|---|
| ai-no-placeholder-code | 禁止 `TODO implement` / `PLACEHOLDER` / `STUB` 等 | pygrep |
| ai-no-hallucination-markers | 禁止 `<PLACEHOLDER>` / `<INSERT_HERE>` 等 | pygrep |
| ai-file-length-limit | 文件行数上限检查 | script |
| ai-active-plans-clean | 活跃 ExecPlan 完成度检查 | script (pre-push) |

**适配项**：
- `ai-file-length-limit` 的上限值通过 `HARNESS_MAX_FILE_LINES` 环境变量或脚本默认值设定，基于 Phase 1 的文件长度分布分析
- `ai-no-placeholder-code` 的正则可根据语言注释风格调整（如 `#` vs `//`）
- 如果项目不使用 ExecPlan，`ai-active-plans-clean` 可省略

---

## Agent 的 Hook 选择决策树

    检测到 git 仓库？
    ├── 否 → 提示先 git init
    └── 是
        ├── base.yaml → 几乎总是安装
        ├── security.yaml → 有凭据风险？是则安装
        ├── 检测语言：
        │   ├── Go → go.yaml
        │   ├── Python → python.yaml
        │   ├── Rust → rust.yaml
        │   ├── 前端 + Biome → frontend-biome.yaml
        │   ├── 前端 + ESLint → frontend-eslint.yaml
        │   └── Java/其他 → 不提供预设，引导用户自行配置
        └── 使用 AI 编码工具？
            ├── 是 → ai-guard.yaml
            └── 否 → 跳过

## 已有 Hook 管理工具的处理

### 已有 .pre-commit-config.yaml（路径 A）

1. 读取现有配置，理解已安装的 hook
2. 将要安装的 hook 与已有的做 diff
3. 展示差异让用户决定合并方式
4. **绝不默认覆盖**

### 已有 husky / lefthook / 等 Node.js hook 管理工具（路径 B）

**核心原则：利用已有机制，不引入新的 hook 管理工具。**

当检测到仓库已使用 husky 等 Node.js 生态的 git hook 管理工具时：

1. **不安装 pre-commit (Python)**——避免两套 hook 管理工具共存的复杂性
2. **将 hook 检查逻辑以 shell 脚本形式添加到 `.hooks/` 目录**
3. **在已有的 git hooks 入口脚本中调用 `.hooks/` 下的脚本**
4. **对于支持 lint-staged 的检查**（如文件长度限制），优先集成到 lint-staged 配置中，利用其增量检查能力

Agent 需要理解每个 hook 的 **目的** 而非 **实现方式**，然后选择最适合已有工具链的实现：

| Hook 目的 | pre-commit 方式 | husky/lint-staged 方式 |
|---|---|---|
| 文件卫生 | pre-commit-hooks repo | lint-staged 中已有的 prettier/eslint 通常已覆盖 |
| 密钥泄露 | gitleaks hook | 在 pre-commit 脚本中调用 gitleaks（需系统安装） |
| 语言 lint | 对应语言 hook | 通常 lint-staged 已覆盖 |
| AI guard | pygrep + script hook | 在 pre-commit 脚本中用 grep + `.hooks/` 脚本实现 |
| 活跃计划检查 | pre-push stage hook | 添加 husky pre-push hook 脚本 |

**注意**：husky 方式的一些检查（如 trailing-whitespace、end-of-file-fixer）可能已被 prettier 覆盖。Agent 需检查已有的 lint-staged 配置，避免重复。
