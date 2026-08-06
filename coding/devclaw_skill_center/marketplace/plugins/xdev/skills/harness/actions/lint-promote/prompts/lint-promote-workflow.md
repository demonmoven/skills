# 规范 lint 化

> **本文件是 prompt 指令**，由 `<skill_dir>/actions/lint-promote/lint-promote.md` 加载。原属于 `harness-bootstrap` 项目的 `harness-lint-promote` 演进 skill，已合并对应的 `references/lint-promote-workflow.md` 详细参考（见文件末尾"附录"段）。

扫描 harness 文档体系中的原则（principles）、不变量（invariants）、代码模式（patterns），评估哪些可以从文档约束升级为自动化 lint 检查，然后实际生成对应的 lint 规则。

**核心洞察：lint > docs**。文档可以被忽略，但 lint 会阻断提交。将规范从"写在文档里希望被遵守"升级为"写在 lint 里必须被遵守"，是工程实践成熟度的关键一步。

## 何时使用

1. **定期规范审查**：团队每隔一段时间（如每月、每季度）执行一次全量扫描，检查哪些文档级约束可以升级为自动化检查。
2. **Promotion Rule 触发**：harness-evolve 在更新 debt-log.md 时发现同一 invariant 被违反 ≥3 次，自动触发本技能评估是否可升级 enforcement 层级。
3. **用户手动触发**：用户明确要求"规范 lint 化"、"lint promote"或"把这条规范变成 lint 规则"。

## 混合策略

不同类型的规范需要不同的检测方式和实现形式。不是所有规范都能变成 lint 规则——设计级约束只能通过文档和 prompt 强制引用来保障。

| 规范类型 | 检测方式 | 实现形式 | 例子 |
|---|---|---|---|
| 文本级约束 | grep / 正则 | Shell pre-commit hook | 文件长度、命名规范、TODO 格式、禁止特定字符串 |
| 语义级约束 | AST | 原生 linter 规则 | 导入规则、错误处理、类型安全、跨包引用 |
| 设计级约束 | AI 理解 | 文档 + prompt 强制引用 | 架构风格、模块职责边界、API 设计原则 |

## Enforcement 升级路径

规范的 enforcement 从弱到强有一条清晰的升级路径：

```
文档 → Shell hook → 原生 linter rule → pre-commit hook → CI check
```

每一层级代表更强的执行力度：

| 当前层级 | 升级目标 | 建议操作 |
|---|---|---|
| 仅文档 (doc) | Shell hook | 编写 `.hooks/scripts/` 下的检查脚本，注册到 pre-commit |
| Shell hook | 原生 linter rule | 将正则检查替换为 AST 级别的 linter 规则（更精准、更少误报） |
| 原生 linter rule | pre-commit hook | 确保 linter 已注册到 pre-commit，本地提交时自动执行 |
| pre-commit hook | CI check | 将检查加入 CI pipeline，确保即使跳过 hook 也能拦截 |
| CI check | （已是最强） | 维持现状，关注规则的误报率和维护成本 |

## 工作流

### Phase 1：扫描规范

扫描 harness 文档体系中的规范源文件：

- `invariants.md` — 项目不变量清单
- `golden-principles.md` — 黄金原则
- `code-patterns.md` — 代码模式约定

从每个文件中提取具体规则，记录：
- 规则 ID 和描述
- 当前 enforcement 层级（doc / hook / linter / CI）
- 原文出处

### Phase 2：分类评估

对每条提取的规则进行分类：

1. **文本级**：可以用 grep/正则检测 → 适合 Shell hook
2. **语义级**：需要 AST 分析 → 适合原生 linter 规则
3. **设计级**：需要 AI 理解上下文 → 只能留在文档 + prompt 强制引用
4. **已覆盖**：已经有对应的 lint 规则或 hook → 跳过

评估自动化是否值得：
- 规则是否足够明确，可以用代码表达？
- 误报率是否可控？
- 维护成本是否低于手动 review 成本？

### Phase 3：生成 lint 规则

根据分类结果生成对应的 lint 实现。**每条规则生成前必须请求用户确认**。

- **Shell hook** → 生成脚本到 `.hooks/scripts/`，注册到 pre-commit 配置
- **原生 linter** → 更新对应语言的 linter 配置文件（参考 `<skill_dir>/actions/init/phase-2-install-hooks/prompts/lint-strategy.md`）
- **Prompt 强制引用** → 更新 `AGENTS.md` 或 `CLAUDE.md`，添加强制引用段落

生成的规则应包含：
- 清晰的错误信息，说明违反了什么规范
- 指向规范文档的引用链接
- 合理的豁免机制（如特定文件、目录的排除）

### Phase 4：更新记录

- 更新 `invariants.md` 中对应规则的 Enforcement 列
- 如果是 Promotion Rule 触发的升级，更新 `debt-log.md` 记录升级动作
- 生成变更摘要，列出本次升级的所有规则

## Promotion Rule

当同一 invariant 在 `debt-log.md` 中累计违反 ≥3 次时，触发 Promotion Rule。

检查当前 enforcement 层级，向用户输出结构化报告：

```
⚠ Promotion Rule 触发
Invariant: {id} - {描述}
当前 Enforcement: {当前层级}
累计违反次数: {次数}
建议升级到: {下一层级}
理由: 该不变量已被违反 {次数} 次，说明当前的文档约束不足以阻止违反。
      升级到 {下一层级} 后可在 {拦截时机} 自动拦截，减少人工 review 负担。
```

用户确认后，执行升级操作：
1. 按照 Phase 3 的流程生成对应层级的 lint 实现
2. 更新 `invariants.md` 的 Enforcement 列
3. 在 `debt-log.md` 中记录升级操作

## 参考

- 详细工作流和模板：本文件末尾"附录：lint-promote workflow 详细参考"段
- 各语言 lint 工具推荐：`<skill_dir>/actions/init/phase-2-install-hooks/prompts/lint-strategy.md`
- 现有 hook 脚本模式：`<skill_dir>/actions/init/phase-2-install-hooks/assets/hook-scripts/`


---

# 附录：lint-promote workflow 详细参考

# 规范 lint 化 — 详细工作流参考

本文档提供 harness-lint-promote 技能的详细模板、配置示例和完整工作流参考。

## Shell hook 脚本模板

基于 `check-file-length.sh` 的模式，所有自定义 Shell hook 脚本应遵循以下模板：

```bash
#!/usr/bin/env bash
# AI 代码守护: {检查名称}
# {一句话说明检查目的和触发原因}
# 可通过环境变量 {ENV_VAR} 配置阈值
set -euo pipefail

# ── 配置 ─────────────────────────────────
# 使用环境变量允许项目级覆盖，提供合理默认值
THRESHOLD=${HARNESS_CHECK_THRESHOLD:-默认值}

# ── 文件过滤 ─────────────────────────────
# 定义需要检查和排除的文件模式
INCLUDE_PATTERN='\.go$|\.py$|\.ts$|\.rs$'
EXCLUDE_PATTERN='_test\.go$|test_.*\.py$|\.test\.ts$|vendor/|node_modules/'

exit_code=0

# ── 检查逻辑 ─────────────────────────────
for f in "$@"; do
    # 跳过不匹配的文件
    if ! echo "$f" | grep -qE "$INCLUDE_PATTERN"; then
        continue
    fi
    # 跳过排除的文件
    if echo "$f" | grep -qE "$EXCLUDE_PATTERN"; then
        continue
    fi

    # 执行具体检查
    result=$(检查命令 "$f")
    if [ 检查条件 ]; then
        echo "ERROR: $f: ${result} (规范: {invariant_id}, 限制: ${THRESHOLD})"
        echo "  参考: harness/invariants.md#{invariant_id}"
        exit_code=1
    fi
done

exit $exit_code
```

### 模板要点

- **`set -euo pipefail`**：严格模式，任何未处理的错误立即退出
- **环境变量配置**：所有阈值通过 `HARNESS_*` 前缀的环境变量配置，便于项目级覆盖
- **文件过滤**：通过 `INCLUDE_PATTERN` 和 `EXCLUDE_PATTERN` 控制检查范围，测试文件默认排除
- **错误信息**：包含文件路径、实际值、限制值、invariant ID 和参考文档链接
- **退出码**：0 表示通过，1 表示有违反

### 实用示例：禁止特定字符串

```bash
#!/usr/bin/env bash
# AI 代码守护: 禁止直接使用 fmt.Println 进行日志输出
# 项目应统一使用结构化日志库
set -euo pipefail

FORBIDDEN_PATTERNS=(
    'fmt\.Println'
    'fmt\.Printf'
    'log\.Print'
)
EXCLUDE_PATTERN='_test\.go$|vendor/'

exit_code=0

for f in "$@"; do
    if ! echo "$f" | grep -qE '\.go$'; then
        continue
    fi
    if echo "$f" | grep -qE "$EXCLUDE_PATTERN"; then
        continue
    fi

    for pattern in "${FORBIDDEN_PATTERNS[@]}"; do
        matches=$(grep -nE "$pattern" "$f" 2>/dev/null || true)
        if [ -n "$matches" ]; then
            echo "ERROR: $f: 使用了禁止的日志方式 '$pattern'"
            echo "$matches" | while IFS= read -r line; do
                echo "  $line"
            done
            echo "  请使用 slog 或项目统一的日志库"
            exit_code=1
        fi
    done
done

exit $exit_code
```

### 实用示例：TODO 格式检查

```bash
#!/usr/bin/env bash
# AI 代码守护: TODO 必须包含负责人和日期
# 格式: TODO(author, YYYY-MM-DD): description
set -euo pipefail

TODO_PATTERN='TODO[^(]|TODO\(\s*\)'
VALID_PATTERN='TODO\([a-zA-Z0-9_-]+,\s*[0-9]{4}-[0-9]{2}-[0-9]{2}\)'

exit_code=0

for f in "$@"; do
    # 检查是否有不规范的 TODO
    bad_todos=$(grep -nE "$TODO_PATTERN" "$f" 2>/dev/null || true)
    if [ -n "$bad_todos" ]; then
        echo "ERROR: $f: TODO 格式不规范，需要 TODO(author, YYYY-MM-DD): description"
        echo "$bad_todos" | while IFS= read -r line; do
            echo "  $line"
        done
        exit_code=1
    fi
done

exit $exit_code
```

## 各语言 lint 配置更新示例

### Go — golangci-lint

在 `.golangci.yml` 中添加自定义规则：

```yaml
# .golangci.yml — 新增规则示例
linters:
  enable:
    - errcheck      # 领域 1: 错误处理
    - funlen        # 领域 2: 函数长度
    - cyclop        # 领域 3: 圈复杂度
    - goconst       # 领域 6: 魔数

linters-settings:
  funlen:
    lines: 120          # invariant INV-003 要求
    statements: 60
  cyclop:
    max-complexity: 15   # invariant INV-005 要求
  goconst:
    min-len: 3
    min-occurrences: 3
  # 新增: 禁止特定导入
  depguard:
    rules:
      main:
        deny:
          - pkg: "io/ioutil"
            desc: "已废弃，使用 io 和 os 包替代 (invariant INV-012)"
          - pkg: "github.com/pkg/errors"
            desc: "使用标准库 fmt.Errorf 的 %w (invariant INV-013)"

issues:
  exclude-rules:
    # 测试文件豁免函数长度
    - path: _test\.go
      linters:
        - funlen
        - cyclop
```

### Python — ruff

在 `pyproject.toml` 中添加规则：

```toml
# pyproject.toml — ruff 新增规则示例
[tool.ruff]
line-length = 120

[tool.ruff.lint]
select = [
    "E",      # pycodestyle errors
    "F",      # pyflakes
    "B",      # flake8-bugbear (错误处理)
    "C901",   # mccabe complexity
    "PLR",    # pylint refactor (重复代码、魔数等)
]

[tool.ruff.lint.mccabe]
max-complexity = 15   # invariant INV-005

[tool.ruff.lint.pylint]
max-statements = 50   # invariant INV-003

[tool.ruff.lint.per-file-ignores]
"tests/**" = ["PLR0915", "C901"]  # 测试文件豁免
```

### Rust — clippy

在 `clippy.toml` 或通过属性添加规则：

```toml
# clippy.toml — 新增规则示例
too-many-lines-threshold = 100       # invariant INV-003
cognitive-complexity-threshold = 25   # invariant INV-005
```

在 `Cargo.toml` 或 `lib.rs` 中启用额外 lint：

```rust
// lib.rs 顶部 — 项目级 clippy 配置
#![deny(clippy::unwrap_used)]      // invariant INV-001: 禁止 unwrap
#![deny(clippy::expect_used)]      // invariant INV-001: 禁止 expect
#![warn(clippy::cognitive_complexity)] // invariant INV-005
#![deny(clippy::dbg_macro)]        // invariant INV-014: 禁止 dbg! 残留
```

### TypeScript — biome / eslint

biome 配置示例（`biome.json`）：

```json
{
  "linter": {
    "rules": {
      "complexity": {
        "noExcessiveCognitiveComplexity": {
          "level": "error",
          "options": { "maxAllowedComplexity": 15 }
        }
      },
      "correctness": {
        "noUnusedVariables": "error",
        "noUnusedImports": "error"
      },
      "suspicious": {
        "noExplicitAny": "error"
      },
      "style": {
        "noDefaultExport": "warn"
      }
    }
  },
  "overrides": [
    {
      "include": ["**/*.test.ts", "**/*.spec.ts"],
      "linter": {
        "rules": {
          "complexity": {
            "noExcessiveCognitiveComplexity": "off"
          }
        }
      }
    }
  ]
}
```

eslint 配置示例（`.eslintrc.cjs`）：

```javascript
// .eslintrc.cjs — 新增规则示例
module.exports = {
  rules: {
    'max-lines-per-function': ['error', { max: 100, skipBlankLines: true, skipComments: true }],
    'complexity': ['error', 15],
    '@typescript-eslint/no-floating-promises': 'error',
    '@typescript-eslint/no-explicit-any': 'error',
    // 自定义禁止特定导入
    'no-restricted-imports': ['error', {
      patterns: [
        { group: ['lodash'], message: '使用原生方法或 lodash-es (invariant INV-015)' },
      ],
    }],
  },
  overrides: [
    {
      files: ['**/*.test.ts', '**/*.spec.ts'],
      rules: {
        'max-lines-per-function': 'off',
        'complexity': 'off',
      },
    },
  ],
};
```

## Prompt 强制引用格式

对于设计级约束，无法通过 lint 工具检测，需要在 AI Agent 的上下文中强制可见。在 `AGENTS.md` 或 `CLAUDE.md` 中添加：

```markdown
## 强制规范（Mandatory Constraints）

以下规范必须在所有代码变更中遵守。违反这些约束的代码不应被合并。

### CONSTRAINT-001: 服务间通信必须通过 API Gateway

所有微服务之间的调用必须经过 API Gateway，禁止直接 service-to-service 调用。
- 来源: `harness/invariants.md#INV-020`
- 原因: 统一鉴权、限流、可观测性
- 违反标志: 代码中出现直接的 HTTP/gRPC 调用到其他服务的内部地址

### CONSTRAINT-002: 数据库 schema 变更必须向后兼容

任何数据库 migration 必须支持回滚，不允许破坏性变更（删列、改类型）在单次部署中完成。
- 来源: `harness/invariants.md#INV-021`
- 原因: 支持蓝绿部署和快速回滚
- 违反标志: migration 文件中包含 DROP COLUMN、ALTER TYPE 等不可逆操作
```

### 格式要求

每条强制引用必须包含：
- **唯一 ID**（CONSTRAINT-NNN）
- **一句话规则描述**
- **来源**：指向 invariants.md 中的具体条目
- **原因**：为什么需要这条约束
- **违反标志**：帮助 AI Agent 识别可能违反此约束的代码模式

## 规范扫描报告格式

Phase 1 和 Phase 2 的输出应以下表格式呈现：

```markdown
# 规范扫描报告

扫描时间: YYYY-MM-DD
扫描范围: invariants.md, golden-principles.md, code-patterns.md

| # | 规范 ID | 描述 | 来源文件 | 当前 Enforcement | 分类 | 可自动化？ | 推荐实现形式 | 优先级 |
|---|---------|------|----------|------------------|------|------------|-------------|--------|
| 1 | INV-001 | 函数不返回未检查的错误 | invariants.md | doc | 语义级 | 是 | 原生 linter (errcheck) | 高 |
| 2 | INV-002 | 文件不超过 600 行 | invariants.md | hook | 文本级 | 已覆盖 | — | — |
| 3 | INV-003 | 函数不超过 120 行 | invariants.md | doc | 语义级 | 是 | 原生 linter (funlen) | 高 |
| 4 | GP-001 | 模块间通过接口通信 | golden-principles.md | doc | 设计级 | 否 | Prompt 强制引用 | 中 |
| 5 | CP-001 | 错误用 fmt.Errorf %w 包装 | code-patterns.md | doc | 语义级 | 是 | 原生 linter (errorlint) | 中 |

## 汇总

- 总规范数: 5
- 已覆盖: 1
- 可升级到 Shell hook: 0
- 可升级到原生 linter: 3
- 保持为文档 + prompt: 1
```

## 完整工作流示例

### 场景 A：定期扫描

用户执行"规范 lint 化"，Agent 对 harness 文档进行全量扫描。

**Step 1：扫描发现 8 条规范**

```
扫描 invariants.md... 发现 5 条规范
扫描 golden-principles.md... 发现 2 条规范
扫描 code-patterns.md... 发现 1 条规范
共计: 8 条规范
```

**Step 2：分类评估**

| # | 规范 ID | 描述 | 当前 | 分类 | 推荐 |
|---|---------|------|------|------|------|
| 1 | INV-001 | 禁止 fmt.Println 日志 | doc | 文本级 | Shell hook |
| 2 | INV-002 | 文件不超过 600 行 | hook | 文本级 | 已覆盖 |
| 3 | INV-003 | TODO 必须有负责人和日期 | doc | 文本级 | Shell hook |
| 4 | INV-004 | 禁止导入废弃包 | doc | 语义级 | 原生 linter (depguard) |
| 5 | INV-005 | 错误必须用 %w 包装 | doc | 语义级 | 原生 linter (errorlint) |
| 6 | GP-001 | 服务间通信走 API Gateway | doc | 设计级 | Prompt 强制引用 |
| 7 | GP-002 | DB migration 向后兼容 | doc | 设计级 | Prompt 强制引用 |
| 8 | CP-001 | 禁止硬编码连接字符串 | doc | 文本级 | Shell hook |

**Step 3：生成结果**

- 3 条升级为 Shell hook：
  - `.hooks/scripts/check-no-println.sh`（检查 fmt.Println）
  - `.hooks/scripts/check-todo-format.sh`（检查 TODO 格式）
  - `.hooks/scripts/check-no-hardcoded-conn.sh`（检查硬编码连接字符串）
- 2 条升级为原生 linter：
  - `.golangci.yml` 启用 `depguard` 和 `errorlint`
- 3 条保持文档级（2 条设计级 + 1 条已覆盖）：
  - 在 `AGENTS.md` 中添加 GP-001、GP-002 的强制引用段落

**Step 4：更新 invariants.md**

```diff
- | INV-001 | 禁止 fmt.Println 日志 | doc |
+ | INV-001 | 禁止 fmt.Println 日志 | hook |
  | INV-002 | 文件不超过 600 行 | hook |
- | INV-003 | TODO 必须有负责人和日期 | doc |
+ | INV-003 | TODO 必须有负责人和日期 | hook |
- | INV-004 | 禁止导入废弃包 | doc |
+ | INV-004 | 禁止导入废弃包 | linter |
- | INV-005 | 错误必须用 %w 包装 | doc |
+ | INV-005 | 错误必须用 %w 包装 | linter |
```

### 场景 B：Promotion Rule 触发

harness-evolve 在记录技术债时发现 INV-007（禁止跨模块直接引用内部包）已被违反 4 次。

**Step 1：Promotion Rule 报告**

```
⚠ Promotion Rule 触发
Invariant: INV-007 - 禁止跨模块直接引用内部包（internal/）
当前 Enforcement: doc
累计违反次数: 4
建议升级到: hook
理由: 该不变量已被违反 4 次，说明当前的文档约束不足以阻止违反。
      升级到 hook 后可在 pre-commit 阶段自动拦截，减少人工 review 负担。
```

**Step 2：用户确认后，生成 Shell hook**

生成 `.hooks/scripts/check-no-cross-internal.sh`：

```bash
#!/usr/bin/env bash
# AI 代码守护: 禁止跨模块引用 internal/ 包
# 每个模块的 internal/ 目录只能被其父模块引用
# invariant: INV-007
set -euo pipefail

exit_code=0

for f in "$@"; do
    if ! echo "$f" | grep -qE '\.go$'; then
        continue
    fi
    if echo "$f" | grep -qE '_test\.go$|vendor/'; then
        continue
    fi

    # 获取文件所在模块路径
    file_dir=$(dirname "$f")

    # 查找 import 中引用其他模块 internal 包的情况
    bad_imports=$(grep -nE '"[^"]+/internal/' "$f" 2>/dev/null || true)
    if [ -z "$bad_imports" ]; then
        continue
    fi

    while IFS= read -r line; do
        # 提取 import 路径
        import_path=$(echo "$line" | grep -oE '"[^"]+"' | tr -d '"')
        # 获取 internal/ 之前的模块前缀
        internal_prefix=$(echo "$import_path" | sed 's|/internal/.*||')

        # 检查当前文件是否在该模块下
        if ! echo "$file_dir" | grep -q "$internal_prefix"; then
            echo "ERROR: $f: 跨模块引用 internal 包"
            echo "  $line"
            echo "  规范: INV-007, 参考: harness/invariants.md#INV-007"
            exit_code=1
        fi
    done <<< "$bad_imports"
done

exit $exit_code
```

**Step 3：注册到 pre-commit**

```yaml
# .pre-commit-config.yaml — 新增条目
- repo: local
  hooks:
    - id: check-no-cross-internal
      name: 禁止跨模块引用 internal 包
      entry: .hooks/scripts/check-no-cross-internal.sh
      language: script
      files: '\.go$'
```

**Step 4：更新记录**

`invariants.md` 更新：

```diff
- | INV-007 | 禁止跨模块直接引用内部包 | doc |
+ | INV-007 | 禁止跨模块直接引用内部包 | hook |
```

`debt-log.md` 追加：

```markdown
### [YYYY-MM-DD] INV-007 Enforcement 升级

- **操作**: Promotion Rule 触发升级
- **从**: doc
- **到**: hook (`.hooks/scripts/check-no-cross-internal.sh`)
- **原因**: 累计违反 4 次，文档约束不足以阻止
- **关联违反记录**: DEBT-012, DEBT-017, DEBT-023, DEBT-031
```

## invariants.md Enforcement 列更新格式

### 更新前

```markdown
# 项目不变量

| ID | 描述 | Enforcement | 备注 |
|---|---|---|---|
| INV-001 | 禁止 fmt.Println 日志 | doc | 应使用 slog |
| INV-002 | 文件不超过 600 行 | hook | check-file-length.sh |
| INV-003 | TODO 必须有负责人和日期 | doc | 格式: TODO(author, date) |
| INV-004 | 禁止导入废弃包 | doc | io/ioutil 等 |
| INV-005 | 错误必须用 %w 包装 | doc | 保留错误链 |
```

### 更新后

```markdown
# 项目不变量

| ID | 描述 | Enforcement | 备注 |
|---|---|---|---|
| INV-001 | 禁止 fmt.Println 日志 | hook | check-no-println.sh |
| INV-002 | 文件不超过 600 行 | hook | check-file-length.sh |
| INV-003 | TODO 必须有负责人和日期 | hook | check-todo-format.sh |
| INV-004 | 禁止导入废弃包 | linter | golangci-lint depguard |
| INV-005 | 错误必须用 %w 包装 | linter | golangci-lint errorlint |
```

变更说明：
- `Enforcement` 列从 `doc` 升级到实际实现形式（`hook` 或 `linter`）
- `备注` 列更新为具体的实现引用（脚本名或 linter 规则名）
- 已有 enforcement 的条目不变
