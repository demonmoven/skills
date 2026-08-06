# Hook 配置片段

本文件提供按技术栈的即用配置片段，供 Step 4 组装配置文件时使用。

## Table of Contents

- [Go](#go)
- [Node / TypeScript](#node--typescript)
- [Python](#python)
- [Rust](#rust)
- [AI 专项守护脚本](#ai-专项守护脚本)
- [组装策略](#组装策略)

---

## Go

### pre-commit 格式

```yaml
- repo: local
  hooks:
    - id: go-fmt
      name: Go format
      entry: gofmt -w
      language: system
      types: [go]
    - id: go-build
      name: Go build
      entry: bash -c 'go build ./...'
      language: system
      pass_filenames: false
    - id: go-mod-tidy
      name: Go mod tidy
      entry: bash -c 'go mod tidy && git diff --exit-code go.sum'
      language: system
      pass_filenames: false
    - id: golangci-lint
      name: golangci-lint
      entry: golangci-lint run --fix
      language: system
      types: [go]
      pass_filenames: false
```

### golangci-lint 推荐配置

如果仓库没有 `.golangci.yml`，建议生成：

```yaml
# .golangci.yml
run:
  timeout: 5m

linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - funlen
    - gocyclo
    - dupl
    - gosimple
    - ineffassign
    - unused

linters-settings:
  funlen:
    lines: 120
    statements: 80
  gocyclo:
    min-complexity: 20
  dupl:
    threshold: 150

issues:
  exclude-dirs:
    - vendor
    - testdata
```

---

## Node / TypeScript

### pre-commit 格式

```yaml
- repo: local
  hooks:
    - id: biome-check
      name: Biome lint + format
      entry: npx biome check --fix --staged
      language: system
      types_or: [ts, tsx, javascript, jsx, json]
    - id: tsc
      name: TypeScript type check
      entry: npx tsc --noEmit
      language: system
      pass_filenames: false
      types: [ts, tsx]
```

### 纯 eslint + prettier（如果不使用 biome）

```yaml
- repo: local
  hooks:
    - id: eslint
      name: ESLint
      entry: npx eslint --fix
      language: system
      types_or: [ts, tsx, javascript, jsx]
    - id: prettier
      name: Prettier
      entry: npx prettier --write
      language: system
      types_or: [ts, tsx, javascript, jsx, json, yaml, markdown, css]
```

### lint-staged 配置（配合 husky）

```json
{
  "lint-staged": {
    "*.{ts,tsx,js,jsx}": ["biome check --fix"],
    "*.{ts,tsx}": ["tsc-files --noEmit"],
    "*.{json,md,yml,yaml}": ["prettier --write"]
  }
}
```

---

## Python

### pre-commit 格式

```yaml
- repo: https://github.com/astral-sh/ruff-pre-commit
  rev: v0.8.0
  hooks:
    - id: ruff
      args: [--fix]
    - id: ruff-format

- repo: local
  hooks:
    - id: mypy
      name: mypy type check
      entry: mypy
      language: system
      types: [python]
      pass_filenames: false
```

---

## Rust

### pre-commit 格式

```yaml
- repo: local
  hooks:
    - id: cargo-fmt
      name: Cargo format
      entry: cargo fmt --check
      language: system
      pass_filenames: false
      types: [rust]
    - id: cargo-clippy
      name: Cargo clippy
      entry: cargo clippy -- -D warnings
      language: system
      pass_filenames: false
      types: [rust]
```

---

## AI 专项守护脚本

### check-file-length.sh

```bash
#!/bin/bash
# 检查代码文件行数不超过上限
# 用法：check-file-length.sh [file1] [file2] ...
# 环境变量 MAX_LINES 可覆盖默认值
MAX_LINES=${MAX_LINES:-600}
EXIT_CODE=0

for file in "$@"; do
  # 跳过不存在的文件（可能已被删除）
  [ -f "$file" ] || continue
  lines=$(wc -l < "$file")
  if [ "$lines" -gt "$MAX_LINES" ]; then
    echo "❌ $file: ${lines} 行 (上限 ${MAX_LINES} 行, 请拆分文件)"
    EXIT_CODE=1
  fi
done

exit $EXIT_CODE
```

### check-no-placeholders.sh

```bash
#!/bin/bash
# 拦截 AI 常见的占位符和幻觉标记
# 用法：check-no-placeholders.sh [file1] [file2] ...
EXIT_CODE=0

PATTERNS='TODO.*[Ii]mplement|FIXME|STUB|HACK[^E]|<PLACEHOLDER>|<INSERT_HERE>|<YOUR_'

for file in "$@"; do
  [ -f "$file" ] || continue
  matches=$(grep -nEi "$PATTERNS" "$file" 2>/dev/null)
  if [ -n "$matches" ]; then
    echo "❌ $file 包含占位符或幻觉标记："
    echo "$matches" | head -5
    EXIT_CODE=1
  fi
done

exit $EXIT_CODE
```

### check-no-debug.sh

```bash
#!/bin/bash
# 拦截非测试文件中的调试语句
# 用法：check-no-debug.sh [file1] [file2] ...
EXIT_CODE=0

for file in "$@"; do
  [ -f "$file" ] || continue
  # 跳过测试文件
  case "$file" in
    *_test.go|*.test.ts|*.test.tsx|*.spec.ts|*.spec.tsx|*_test.py|test_*) continue ;;
  esac

  matches=$(grep -nE '(console\.(log|debug|warn)\(|fmt\.Print(ln|f)?\(|print\()' "$file" 2>/dev/null)
  if [ -n "$matches" ]; then
    echo "❌ $file 包含调试语句（非测试文件）："
    echo "$matches" | head -3
    EXIT_CODE=1
  fi
done

exit $EXIT_CODE
```

### pre-commit 配置片段（AI 专项）

```yaml
- repo: local
  hooks:
    - id: check-file-length
      name: File length check (max 600 lines)
      entry: .hooks/check-file-length.sh
      language: script
      types: [text]
      exclude: '(vendor|node_modules|dist|\.min\.|go\.sum|\.lock$|generated)'
    - id: check-no-placeholders
      name: No AI placeholders
      entry: .hooks/check-no-placeholders.sh
      language: script
      types: [text]
      exclude: '(CHANGELOG|\.md$)'
    - id: check-no-debug
      name: No debug statements
      entry: .hooks/check-no-debug.sh
      language: script
      types_or: [go, ts, tsx, javascript, python]
      exclude: '(vendor|node_modules|dist)'
```

---

## 组装策略

生成最终配置时，按以下顺序拼接：

1. **L0 通用卫生**（pre-commit-hooks repo + gitleaks）
2. **L1 语言质量**（按检测到的每种技术栈依次追加）
3. **L2 AI 专项守护**（local repo 下的自定义脚本）
4. **L3 架构护栏**（如有，追加到 local repo）

如果已有配置文件，只追加新 repo/hook，保留已有内容不动。
