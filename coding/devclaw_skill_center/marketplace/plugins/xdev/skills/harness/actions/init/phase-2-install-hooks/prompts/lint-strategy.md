# Lint 策略指南

本文档定义了与语言无关的 lint 关注领域，以及每种主流语言的推荐工具和规则映射。harness-bootstrap 的 Agent 根据本文档为目标仓库选择合适的 lint 配置。

## 核心理念

lint 配置不是"越多越好"，而是"精准拦截 AI 常犯错误"。AI Agent 的错误模式与人类不同——它不会拼错变量名，但会忽略错误返回值、生成超长函数、复制粘贴重复代码。lint 规则应针对这些特定的失败模式。

## 关注领域矩阵

下表列出每个关注领域的 **目的** 和 **核心逻辑**，以及各语言的推荐实现。Agent 根据目标仓库的语言，在此表中查找对应工具和规则。

### 领域 1：错误处理不规范

**目的**：AI 经常忽略函数返回的错误值，或用 panic/raise 代替正常错误处理。

**核心逻辑**：确保每个可能失败的调用都有错误检查。

| 语言 | 工具 | 规则/配置 | 示例 |
|---|---|---|---|
| Go | golangci-lint | `errcheck`, `errorlint`, `err113` | 捕获未检查的 `err` 返回值 |
| Python | ruff | `B`(flake8-bugbear), `E722`(bare-except) | 禁止空 `except:` |
| Rust | clippy | `clippy::unwrap_used`, `clippy::expect_used` | 鼓励用 `?` 代替 unwrap |
| TypeScript | eslint | `@typescript-eslint/no-floating-promises` | 捕获未 await 的 Promise |
| Java | checkstyle / error-prone | `MissingSwitchDefault`, `EmptyCatchBlock` | 空 catch 块检查 |

### 领域 2：函数/文件过长

**目的**：AI 倾向于生成超长函数和文件。过长的代码难以理解和维护。

**核心逻辑**：限制单个函数的行数和单个文件的行数。限制值应基于仓库的实际代码分布，而非一刀切。

| 语言 | 工具 | 规则/配置 | 建议默认值 |
|---|---|---|---|
| Go | golangci-lint | `funlen` (函数行数) | 120 行（测试文件豁免） |
| Python | ruff | `PLR0915`(too-many-statements) | 50 语句 |
| Rust | clippy | `clippy::too_many_lines` | 100 行 |
| TypeScript | eslint | `max-lines-per-function` | 100 行 |
| 通用 | pre-commit script | `check-file-length.sh` | 600 行（可通过环境变量调整） |

**适配注意**：如果仓库中大量文件在 800 行且结构合理，上限应适配为 800-1000。对数据管道、测试文件等天然较长的文件可豁免。

### 领域 3：圈复杂度过高

**目的**：AI 生成的函数可能有过多嵌套分支，难以理解和测试。

**核心逻辑**：限制函数的圈复杂度（cyclomatic complexity）或认知复杂度（cognitive complexity）。

| 语言 | 工具 | 规则/配置 | 建议默认值 |
|---|---|---|---|
| Go | golangci-lint | `cyclop`(max-complexity), `gocognit` | cyclop=15, gocognit=20 |
| Python | ruff | `C901`(mccabe complexity) | max-complexity=15 |
| Rust | clippy | `clippy::cognitive_complexity` | 25 |
| TypeScript | eslint | `complexity` | 15 |

### 领域 4：未使用代码

**目的**：AI 经常生成未使用的变量、导入和函数，增加代码噪音。

**核心逻辑**：禁止未使用的导入、变量和参数。

| 语言 | 工具 | 规则/配置 |
|---|---|---|
| Go | golangci-lint | `unused`, `ineffassign`, `unparam` |
| Python | ruff | `F401`(unused-import), `F841`(unused-variable) |
| Rust | rustc | 内置 `dead_code`, `unused_variables` warning |
| TypeScript | eslint/biome | `noUnusedVariables`, `noUnusedImports` |

### 领域 5：重复代码

**目的**：AI 的 copy-paste 倾向特别强，容易在多处复制相似逻辑。

**核心逻辑**：检测代码块级别的重复，超过阈值时报告。

| 语言 | 工具 | 规则/配置 | 建议阈值 |
|---|---|---|---|
| Go | golangci-lint | `dupl` | 100 token |
| Python | ruff / pylint | `PLR0801`(duplicate-code) | 默认 |
| Rust | - | 无内置工具，依赖 code review | - |
| TypeScript | eslint | `no-duplicate-case` + jscpd | 默认 |

### 领域 6：魔数和硬编码

**目的**：AI 经常硬编码数字和字符串，不提取为常量。

**核心逻辑**：检测频繁出现的字面量，建议提取为命名常量。

| 语言 | 工具 | 规则/配置 |
|---|---|---|
| Go | golangci-lint | `goconst` (min-len=3, min-occurrences=3) |
| Python | ruff | `PLR2004`(magic-value-comparison) |
| TypeScript | eslint | `no-magic-numbers` |

### 领域 7：拼写错误

**目的**：AI 偶尔在标识符和注释中引入拼写错误。

| 语言 | 工具 | 规则/配置 |
|---|---|---|
| Go | golangci-lint | `misspell` (locale: US) |
| Python | codespell (pre-commit hook) | 独立工具 |
| 通用 | typos (pre-commit hook) | `crate-ci/typos` |

### 领域 8：类型安全

**目的**：确保类型系统被正确利用，不被 `any`/`interface{}` 绕过。

| 语言 | 工具 | 规则/配置 |
|---|---|---|
| Go | golangci-lint | `govet` (shadow) |
| Python | mypy / pyright | strict mode |
| TypeScript | tsc + eslint | `strict: true`, `noExplicitAny` |
| Rust | rustc | 内置类型系统足够强 |

## 推荐的 lint 工具总览

| 语言 | 主推工具 | 替代工具 | 配置文件 |
|---|---|---|---|
| Go | golangci-lint v2 | - | `.golangci.yml` |
| Python | ruff | flake8 + pylint | `pyproject.toml` 或 `ruff.toml` |
| Rust | clippy | - | `clippy.toml` 或 `Cargo.toml` |
| TypeScript/JavaScript | biome | eslint + prettier | `biome.json` 或 `.eslintrc.*` |
| Java | checkstyle + error-prone | spotbugs | `checkstyle.xml` |

## Agent 的决策流程

1. 从 Phase 1 的检测结果确定主语言
2. 在上方矩阵中查找该语言对应的工具和规则
3. 检查仓库是否已有该工具的配置文件
4. 如果已有：检查缺失的领域，建议补充（展示 diff，不覆盖）
5. 如果没有：以 `<skill_dir>/actions/init/phase-2-install-hooks/assets/lint-examples/` 中的示例为基线，根据仓库代码特征调整参数后生成
6. 特别注意测试文件的豁免——测试中的长函数、高复杂度通常是合理的

## 从 lint-examples/ 中选择

`<skill_dir>/actions/init/phase-2-install-hooks/assets/lint-examples/` 提供了各语言的参考配置，**不是直接复制的模板**，而是带注释的示例，说明每条规则的用途。Agent 应该：

1. 读取对应语言的示例文件，理解每条规则
2. 对照仓库实际代码特征，决定启用哪些、调整哪些参数
3. 生成适配后的配置文件
