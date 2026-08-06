# Pre-commit Hook 目录

本文件提供按技术栈分类的 pre-commit hook 建议，供 harness-init Phase 5 使用。

## Table of Contents

- [分层策略](#分层策略)
- [L0 通用卫生（所有仓库）](#l0-通用卫生所有仓库)
- [L1 语言质量](#l1-语言质量)
- [L2 AI 专项守护](#l2-ai-专项守护)
- [L3 架构护栏（成熟仓库）](#l3-架构护栏成熟仓库)
- [配置生成指南](#配置生成指南)

---

## 分层策略

| 层级 | 目标 | 安装优先级 |
|------|------|----------|
| **L0 通用卫生** | 所有仓库都应有的基础检查 | 必装 |
| **L1 语言质量** | 按技术栈的编译/lint/format | 按需装 |
| **L2 AI 专项守护** | 针对 AI 生成代码的常见问题 | 推荐装 |
| **L3 架构护栏** | 模块边界、依赖方向、计划完成度 | 成熟仓库选装 |

---

## L0 通用卫生（所有仓库）

适用工具：`pre-commit` / `lefthook` / `husky + lint-staged`

| 检查项 | 说明 | pre-commit hook ID |
|--------|------|-------------------|
| trailing-whitespace | 删除行尾空格 | `trailing-whitespace` |
| end-of-file-fixer | 确保文件末尾换行 | `end-of-file-fixer` |
| check-yaml | YAML 格式校验 | `check-yaml` |
| check-json | JSON 格式校验 | `check-json` |
| check-toml | TOML 格式校验 | `check-toml` |
| check-merge-conflict | 冲突标记检测 | `check-merge-conflict` |
| check-added-large-files | 大文件拦截（默认 500KB） | `check-added-large-files` |
| mixed-line-ending | 统一换行符为 LF | `mixed-line-ending` |
| fix-byte-order-marker | 清除 BOM | `fix-byte-order-marker` |
| gitleaks / detect-secrets | 密钥/凭据泄露检测 | `gitleaks` |

---

## L1 语言质量

### Go

| 检查项 | 说明 | 工具 |
|--------|------|------|
| go fmt | 格式化 | `gofmt` |
| go build | 编译检查 | `go build ./...` |
| go mod tidy | 依赖一致性 | `go mod tidy` |
| golangci-lint | 静态分析（含 funlen、errcheck 等） | `golangci-lint run` |
| go test | 单元测试 | `go test ./...`（可选，较慢） |

建议 `.golangci.yml` 启用的 linter：
- `errcheck`、`govet`、`staticcheck` — 正确性
- `funlen`（建议上限 120 行）— 函数复杂度
- `gocyclo` / `cyclop` — 圈复杂度
- `dupl` — 重复代码检测

### Node / TypeScript

| 检查项 | 说明 | 工具 |
|--------|------|------|
| biome check | lint + format 一体化 | `npx biome check --fix` |
| 或 eslint + prettier | 传统 lint + format | `npx eslint --fix` + `npx prettier --write` |
| tsc --noEmit | TypeScript 类型检查 | `npx tsc --noEmit` |
| npm run build | 构建检查 | `npm run build`（可选） |

### Python

| 检查项 | 说明 | 工具 |
|--------|------|------|
| ruff check | lint（替代 flake8/pylint/isort） | `ruff check --fix` |
| ruff format | 格式化（替代 black） | `ruff format` |
| mypy | 类型检查 | `mypy .` |
| pytest | 单元测试 | `pytest`（可选，较慢） |

### Rust

| 检查项 | 说明 | 工具 |
|--------|------|------|
| cargo fmt | 格式化 | `cargo fmt --check` |
| cargo clippy | 静态分析 | `cargo clippy -- -D warnings` |
| cargo build | 编译检查 | `cargo build` |
| cargo test | 单元测试 | `cargo test`（可选） |

### Java / Kotlin

| 检查项 | 说明 | 工具 |
|--------|------|------|
| spotless / google-java-format | 格式化 | Gradle/Maven plugin |
| checkstyle / ktlint | lint | Gradle/Maven plugin |
| compile | 编译检查 | `./gradlew compileJava` |

---

## L2 AI 专项守护

这些检查专门针对 AI Agent 生成代码的常见问题。

| 检查项 | 说明 | 实现方式 |
|--------|------|---------|
| 禁止占位符代码 | 拦截 `TODO implement`、`STUB`、`HACK`、`FIXME`、`XXX` | 正则 grep 脚本 |
| 禁止幻觉标记 | 拦截 `<PLACEHOLDER>`、`<INSERT_HERE>`、`<YOUR_...>` | 正则 grep 脚本 |
| 文件行数上限 | 建议 600 行，AI 倾向生成超长文件 | `wc -l` 脚本 |
| 禁止硬编码凭据 | 拦截 `password=`、`secret=`、`api_key=` 的字面量赋值 | gitleaks 覆盖 |
| 禁止调试语句 | 拦截 `console.log(`、`fmt.Println(`、`print(` 在非测试文件 | 正则 grep 脚本 |

### 示例：文件行数检查脚本

```bash
#!/bin/bash
# check-file-length.sh — 检查代码文件行数不超过上限
MAX_LINES=${MAX_LINES:-600}
EXIT_CODE=0

for file in "$@"; do
  lines=$(wc -l < "$file")
  if [ "$lines" -gt "$MAX_LINES" ]; then
    echo "❌ $file: $lines 行 (上限 $MAX_LINES 行, 请拆分文件)"
    EXIT_CODE=1
  fi
done

exit $EXIT_CODE
```

### 示例：占位符检测脚本

```bash
#!/bin/bash
# check-no-placeholders.sh — 拦截 AI 常见的占位符模式
PATTERNS=(
  'TODO.*implement'
  'FIXME'
  'STUB'
  'HACK'
  '<PLACEHOLDER>'
  '<INSERT_HERE>'
  '<YOUR_'
  '// \.\.\.'
)

EXIT_CODE=0
REGEX=$(IFS='|'; echo "${PATTERNS[*]}")

for file in "$@"; do
  if grep -nEi "$REGEX" "$file" 2>/dev/null; then
    echo "❌ $file 包含占位符或幻觉标记"
    EXIT_CODE=1
  fi
done

exit $EXIT_CODE
```

---

## L3 架构护栏（成熟仓库）

| 检查项 | 说明 | 实现方式 |
|--------|------|---------|
| 依赖方向检查 | 确保模块间依赖方向正确 | 自定义脚本或 ArchUnit 风格测试 |
| 活动计划完成度 | push 前检查 docs/plans/active/ 是否有未完成项 | shell 脚本扫描 `[ ]` |
| 代码生成一致性 | 确认生成代码与 IDL/proto 源一致 | `diff` 检查 |
| 导入路径合规 | 禁止跨模块的非法导入 | lint 规则或自定义脚本 |

---

## 配置生成指南

### 使用 pre-commit 框架

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.6.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-yaml
      - id: check-json
      - id: check-added-large-files
        args: ['--maxkb=500']
      - id: check-merge-conflict
      - id: mixed-line-ending
        args: ['--fix=lf']

  - repo: https://github.com/gitleaks/gitleaks
    rev: v8.18.0
    hooks:
      - id: gitleaks

  # 按技术栈添加语言特定 hooks...

  - repo: local
    hooks:
      - id: check-file-length
        name: Check file length
        entry: .hooks/check-file-length.sh
        language: script
        types: [text]
        exclude: '(vendor|node_modules|dist|\.min\.)'
```

### 安装命令

```bash
# 使用 pre-commit（Python 生态，支持所有语言）
pip install pre-commit  # 或 brew install pre-commit
pre-commit install

# 使用 husky（Node 项目）
npx husky init

# 使用 lefthook（Go 实现，跨语言）
lefthook install
```
