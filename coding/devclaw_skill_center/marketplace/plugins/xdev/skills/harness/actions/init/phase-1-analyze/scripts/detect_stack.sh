#!/usr/bin/env bash
# detect_stack.sh — 检测目标仓库的技术栈
# 输出 JSON 格式的检测结果，供 harness-bootstrap SKILL.md 使用
# 用法: bash detect_stack.sh [repo_root]

set -euo pipefail

REPO_ROOT="${1:-.}"
cd "$REPO_ROOT"

# --- 语言检测 ---
HAS_GO=false
HAS_NODE=false
HAS_PYTHON=false
HAS_RUST=false
HAS_JAVA=false
GO_MOD_COUNT=0
PACKAGE_JSON_COUNT=0
CARGO_TOML_COUNT=0

# Go
if find . -name "go.mod" -not -path "./.git/*" -maxdepth 5 2>/dev/null | grep -q .; then
  HAS_GO=true
  GO_MOD_COUNT=$(find . -name "go.mod" -not -path "./.git/*" -maxdepth 5 2>/dev/null | wc -l | tr -d ' ')
fi

# Node
if find . -name "package.json" -not -path "./.git/*" -not -path "*/node_modules/*" -maxdepth 5 2>/dev/null | grep -q .; then
  HAS_NODE=true
  PACKAGE_JSON_COUNT=$(find . -name "package.json" -not -path "./.git/*" -not -path "*/node_modules/*" -maxdepth 5 2>/dev/null | wc -l | tr -d ' ')
fi

# Python
if [ -f "pyproject.toml" ] || [ -f "setup.py" ] || [ -f "requirements.txt" ] || [ -f "Pipfile" ] || [ -f "setup.cfg" ]; then
  HAS_PYTHON=true
fi

# Rust
if find . -name "Cargo.toml" -not -path "./.git/*" -maxdepth 5 2>/dev/null | grep -q .; then
  HAS_RUST=true
  CARGO_TOML_COUNT=$(find . -name "Cargo.toml" -not -path "./.git/*" -maxdepth 5 2>/dev/null | wc -l | tr -d ' ')
fi

# Java
if [ -f "pom.xml" ] || [ -f "build.gradle" ] || [ -f "build.gradle.kts" ]; then
  HAS_JAVA=true
fi

# --- 前端框架检测 ---
HAS_FRONTEND=false
HAS_BIOME=false
HAS_ESLINT=false
HAS_PRETTIER=false

for pj in $(find . -maxdepth 4 -name "package.json" -not -path "*/node_modules/*" -not -path "./.git/*" 2>/dev/null); do
  if grep -qE '"react"|"vue"|"angular"|"svelte"|"next"|"nuxt"|"solid-js"' "$pj" 2>/dev/null; then
    HAS_FRONTEND=true
    break
  fi
done

# Lint 工具检测（递归搜索，支持 monorepo）
find . -maxdepth 4 -name "biome.json" -o -name "biome.jsonc" 2>/dev/null | grep -q . && HAS_BIOME=true || true
find . -maxdepth 4 \( -name ".eslintrc" -o -name ".eslintrc.js" -o -name ".eslintrc.json" -o -name ".eslintrc.yml" -o -name "eslint.config.js" -o -name "eslint.config.mjs" \) 2>/dev/null | grep -q . && HAS_ESLINT=true || true
find . -maxdepth 4 \( -name ".prettierrc" -o -name ".prettierrc.js" -o -name ".prettierrc.json" -o -name ".prettierrc.yml" -o -name "prettier.config.js" \) 2>/dev/null | grep -q . && HAS_PRETTIER=true || true

# --- Python 工具检测 ---
HAS_RUFF=false
HAS_BLACK=false
HAS_MYPY=false

[ -f "ruff.toml" ] || [ -f ".ruff.toml" ] && HAS_RUFF=true
if [ -f "pyproject.toml" ]; then
  grep -q '\[tool\.ruff\]' pyproject.toml 2>/dev/null && HAS_RUFF=true
  grep -q '\[tool\.black\]' pyproject.toml 2>/dev/null && HAS_BLACK=true
  grep -q '\[tool\.mypy\]' pyproject.toml 2>/dev/null && HAS_MYPY=true
fi

# --- Go 工具检测 ---
HAS_GOLANGCI=false
find . -maxdepth 4 -name ".golangci.yml" -o -name ".golangci.yaml" 2>/dev/null | grep -q . && HAS_GOLANGCI=true || true

# --- 已有工具链 ---
HAS_PRECOMMIT=false
HAS_MAKEFILE=false
HAS_DOCKER=false
HAS_CI=false
HAS_HUSKY=false
HAS_LINT_STAGED=false
HOOK_MANAGER="none"

[ -f ".pre-commit-config.yaml" ] && HAS_PRECOMMIT=true
[ -f "Makefile" ] || [ -f "Taskfile.yml" ] && HAS_MAKEFILE=true
[ -f "Dockerfile" ] || [ -f "docker-compose.yml" ] || [ -f "docker-compose.yaml" ] && HAS_DOCKER=true
[ -d ".github/workflows" ] || [ -f ".gitlab-ci.yml" ] || [ -f "Jenkinsfile" ] || [ -f ".circleci/config.yml" ] && HAS_CI=true

# Git hook manager detection: husky, lefthook, simple-git-hooks, etc.
# Check husky via package.json "prepare" script or .husky/ directory or git-hooks/ directory
if [ -f "package.json" ]; then
  grep -qE '"husky"' package.json 2>/dev/null && HAS_HUSKY=true
  grep -qE '"lint-staged"' package.json 2>/dev/null && HAS_LINT_STAGED=true
fi
# Also check for husky hooks directory
[ -d ".husky" ] || [ -d "git-hooks" ] && HAS_HUSKY=true
# Check for lint-staged config files
[ -f ".lintstagedrc" ] || [ -f ".lintstagedrc.js" ] || [ -f ".lintstagedrc.json" ] || [ -f ".lintstagedrc.yml" ] || [ -f "lint-staged.config.js" ] || [ -f "lint-staged.config.mjs" ] && HAS_LINT_STAGED=true

# Determine primary hook manager
if [ "$HAS_PRECOMMIT" = true ]; then
  HOOK_MANAGER="pre-commit"
elif [ "$HAS_HUSKY" = true ]; then
  HOOK_MANAGER="husky"
fi

# --- 已有文档 ---
HAS_AGENTS_MD=false
HAS_ARCHITECTURE_MD=false
HAS_DOCS_DIR=false
HAS_PLANS_DIR=false
HAS_SKILLS_DIR=false

[ -f "AGENTS.md" ] || [ -f "CLAUDE.md" ] && HAS_AGENTS_MD=true
[ -f "ARCHITECTURE.md" ] && HAS_ARCHITECTURE_MD=true
[ -d "docs" ] && HAS_DOCS_DIR=true
[ -d "docs/plans" ] && HAS_PLANS_DIR=true
[ -d ".skills" ] || [ -d ".claude/skills" ] || [ -d ".opencode/skills" ] && HAS_SKILLS_DIR=true

# --- 输出 JSON ---
cat <<EOF
{
  "languages": {
    "go": $HAS_GO,
    "node": $HAS_NODE,
    "python": $HAS_PYTHON,
    "rust": $HAS_RUST,
    "java": $HAS_JAVA
  },
  "frontend": {
    "detected": $HAS_FRONTEND,
    "biome": $HAS_BIOME,
    "eslint": $HAS_ESLINT,
    "prettier": $HAS_PRETTIER
  },
  "python_tools": {
    "ruff": $HAS_RUFF,
    "black": $HAS_BLACK,
    "mypy": $HAS_MYPY
  },
  "go_tools": {
    "golangci_lint": $HAS_GOLANGCI
  },
  "toolchain": {
    "pre_commit": $HAS_PRECOMMIT,
    "husky": $HAS_HUSKY,
    "lint_staged": $HAS_LINT_STAGED,
    "hook_manager": "$HOOK_MANAGER",
    "makefile": $HAS_MAKEFILE,
    "docker": $HAS_DOCKER,
    "ci": $HAS_CI
  },
  "docs": {
    "agents_md": $HAS_AGENTS_MD,
    "architecture_md": $HAS_ARCHITECTURE_MD,
    "docs_dir": $HAS_DOCS_DIR,
    "plans_dir": $HAS_PLANS_DIR,
    "skills_dir": $HAS_SKILLS_DIR
  },
  "counts": {
    "go_mod": $GO_MOD_COUNT,
    "package_json": $PACKAGE_JSON_COUNT,
    "cargo_toml": $CARGO_TOML_COUNT
  }
}
EOF
