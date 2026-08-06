# Hook 框架选型指南

## Table of Contents

- [三大主流框架对比](#三大主流框架对比)
- [选择决策树](#选择决策树)
- [各框架配置示例](#各框架配置示例)
- [混合技术栈策略](#混合技术栈策略)

---

## 三大主流框架对比

| 维度 | pre-commit | lefthook | husky + lint-staged |
|------|-----------|----------|-------------------|
| **语言** | Python（框架本身） | Go（单二进制） | Node（npm 包） |
| **安装** | `pip install pre-commit` | `brew install lefthook` 或下载二进制 | `npm install husky lint-staged` |
| **配置文件** | `.pre-commit-config.yaml` | `lefthook.yml` | `.husky/pre-commit` + `lint-staged` config |
| **社区 hook** | 极其丰富（数千个预制 repo） | 较少，多数需要自写 | 依赖 npm scripts |
| **隔离执行** | 是（每个 hook 在独立 venv/env 中运行） | 否（直接调系统命令） | 否（直接调 npm scripts） |
| **多语言支持** | 优秀（官方支持 20+ 语言） | 良好（通过 `run:` 调任意命令） | 仅 Node 生态 |
| **速度** | 中等（首次慢，有缓存） | 快（并行执行） | 快 |
| **staged 文件过滤** | 内置 | 内置（`{staged_files}`） | 通过 lint-staged |
| **CI 复用** | `pre-commit run --all-files` | `lefthook run pre-commit` | 需单独写 |

## 选择决策树

```
仓库是纯 Node/TS 项目？
├── 是 → 已有 package.json scripts 生态完善？
│       ├── 是 → husky + lint-staged
│       └── 否 → pre-commit（更全面）
└── 否 → 有 Python 运行时或不介意安装？
        ├── 是 → pre-commit（社区最丰富）
        └── 否 → lefthook（零依赖单二进制）
```

## 各框架配置示例

### pre-commit

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

  - repo: local
    hooks:
      - id: check-file-length
        name: Check file length (max 600 lines)
        entry: .hooks/check-file-length.sh
        language: script
        types: [text]
        exclude: '(vendor|node_modules|dist|\.min\.|go\.sum)'
```

### lefthook

```yaml
# lefthook.yml
pre-commit:
  parallel: true
  commands:
    trailing-whitespace:
      run: "sed -i '' 's/[[:space:]]*$//' {staged_files}"
      glob: "*.{go,ts,tsx,js,py,rs,md}"
    check-yaml:
      run: "python -c 'import yaml,sys; yaml.safe_load(open(sys.argv[1]))' {staged_files}"
      glob: "*.{yml,yaml}"
    gitleaks:
      run: "gitleaks protect --staged --no-banner"
    go-fmt:
      run: "gofmt -w {staged_files}"
      glob: "*.go"
    go-build:
      run: "go build ./..."
    golangci-lint:
      run: "golangci-lint run --new-from-rev=HEAD~1"
    file-length:
      run: ".hooks/check-file-length.sh {staged_files}"
      glob: "*.{go,ts,tsx,js,py,rs}"
```

### husky + lint-staged

```json
// package.json
{
  "scripts": {
    "prepare": "husky"
  },
  "lint-staged": {
    "*.{ts,tsx,js,jsx}": [
      "biome check --fix",
      "tsc-files --noEmit"
    ],
    "*.{md,json,yaml,yml}": [
      "prettier --write"
    ]
  }
}
```

```bash
# .husky/pre-commit
npx lint-staged
.hooks/check-file-length.sh $(git diff --cached --name-only --diff-filter=ACM)
```

## 混合技术栈策略

如果仓库同时包含多种语言（如 Go 后端 + TypeScript 前端）：

1. **推荐 pre-commit**：天然支持多语言，每个 hook 声明自己的 `types` 或 `files` 过滤
2. **备选 lefthook**：通过 `glob` 字段按文件扩展名路由到不同命令
3. **不推荐 husky**：仅适合 Node 部分，Go/Python/Rust 部分需要额外机制

如果已有 husky 但仓库扩展了非 Node 语言：
- 方案 A：在 `.husky/pre-commit` 中混入 shell 命令调用 Go/Python 工具
- 方案 B：迁移到 pre-commit 或 lefthook（破坏性更大，需用户确认）
