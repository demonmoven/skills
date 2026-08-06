---
name: harness-hook-init
description: "交互式为任意仓库建立 pre-commit hook 质量护栏。检测技术栈和已有 hook 基建，逐层向用户确认要启用的检查项，生成配置文件和自定义脚本，执行安装并验证生效。当用户提到安装 hooks、hook init、建立提交护栏、pre-commit 配置、harness hook 时激活。"
license: Apache-2.0
metadata:
  author: guoshuai.030
  version: "1.4"
---

# Harness Hook Init — 交互式提交护栏建立

## 核心理念

> **更多的 AI 自治 = 更严格的运行时约束。**
> 约束不靠 Agent 自觉，靠工具链机械化执行。

本技能通过交互式向导，为仓库建立分层的 pre-commit 质量护栏。

**产出**：
- hook 框架配置文件（`.pre-commit-config.yaml` / `lefthook.yml` / `.husky/`）
- 自定义检查脚本（文件长度、占位符检测等，放入 `.hooks/` 或项目约定目录）
- 安装完成后的验证确认

**原则**：
- 已有配置不覆盖，只增量追加
- 每个决策点都征求用户确认
- 安装后 dry-run 验证生效
- 报错信息对 Agent 友好（报错即指导）

## 激活后参考

- [references/FRAMEWORK-GUIDE.md](references/FRAMEWORK-GUIDE.md) — hook 框架选型对比
- [references/HOOK-RECIPES.md](references/HOOK-RECIPES.md) — 按技术栈的现成配置片段

## 完整流程

### Step 1: 检测现状

**1.1 检测技术栈**

扫描仓库根目录，记录所有检测到的语言/框架：

```
go.mod / go.work          → Go
package.json              → Node/TypeScript（进一步检查 biome.json / .eslintrc / tsconfig）
pyproject.toml / setup.py → Python（进一步检查 ruff.toml / mypy.ini）
Cargo.toml                → Rust
pom.xml / build.gradle    → Java/Kotlin
```

多种共存时全部记录——hook 配置需要覆盖所有技术栈。

**1.2 检测已有 hook 基建**

| 文件/目录 | 含义 |
|-----------|------|
| `.pre-commit-config.yaml` | 已使用 pre-commit 框架 |
| `.husky/` | 已使用 husky（Node 生态） |
| `lefthook.yml` | 已使用 lefthook |
| `.git/hooks/pre-commit`（非符号链接） | 手写 hook 脚本 |
| `lint-staged` 配置（package.json 或独立文件） | 已有 staged 文件过滤 |
| `.hooks/` / `git-hooks/` | 已有自定义脚本目录 |

**如果已有框架**：后续步骤在现有框架上增量追加，不迁移框架。
**如果无框架**：进入 Step 2 选择框架。

**1.3 检测已有检查项**

如果已有配置，解析出当前已启用的 hook ID 列表，避免重复建议。

### Step 2: 框架选择（仅无已有框架时）

向用户展示对比，征求选择：

| 框架 | 适用场景 | 优点 | 缺点 |
|------|---------|------|------|
| **pre-commit** | 多语言/Go/Python 项目 | 语言无关、社区 hook 丰富、隔离环境执行 | 依赖 Python 运行时 |
| **lefthook** | 多语言项目、不想装 Python | Go 单二进制、快、配置简洁 | 社区 hook 少于 pre-commit |
| **husky + lint-staged** | 纯 Node/TS 项目 | 与 npm 生态深度集成 | 仅适合 Node 项目 |

**推荐逻辑**：
- 已有 `package.json` 且无 Go/Python/Rust → 推荐 husky
- 已有 Go 或 Python → 推荐 pre-commit
- 用户不想安装 Python → 推荐 lefthook

询问用户选择后继续。

### Step 3: 逐层确认检查项

分 4 层逐层展示，每层由用户确认启用哪些：

#### Layer 0: 通用卫生（强烈推荐全部启用）

向用户展示：

```
Layer 0 — 通用卫生（所有语言、零配置）：
  ✓ trailing-whitespace    删除行尾空格
  ✓ end-of-file-fixer      确保文件末尾换行
  ✓ check-yaml             YAML 格式校验
  ✓ check-json             JSON 格式校验
  ✓ check-merge-conflict   冲突标记检测
  ✓ check-added-large-files 大文件拦截（>500KB）
  ✓ mixed-line-ending      统一为 LF
  ✓ gitleaks               密钥/凭据泄露检测

全部启用？[Y/n] 或输入要排除的项
```

#### Layer 1: 语言质量（按检测到的技术栈展示）

仅展示与仓库技术栈匹配的检查项。例如对 Go 项目：

```
Layer 1 — Go 语言质量：
  ✓ go fmt                 格式化
  ✓ go build ./...         编译检查
  ✓ go mod tidy            依赖一致性
  ✓ golangci-lint run      静态分析

启用哪些？[全部/选择/跳过]
```

如果是混合技术栈，每种语言分别展示和确认。

#### Layer 2: AI 专项守护

```
Layer 2 — AI 专项守护（推荐 Agent-first 仓库启用）：
  ○ 文件行数上限          默认 600 行，超限报错提示拆分
  ○ 占位符/幻觉标记检测   拦截 TODO implement、<PLACEHOLDER> 等
  ○ 禁止调试语句          拦截非测试文件的 console.log / fmt.Println

启用哪些？[全部/选择/跳过]
```

如果用户选择启用，追问：
- 文件行数上限：默认 600，是否调整？
- 排除目录：默认排除 `vendor/`、`node_modules/`、`dist/`，还需排除其他？
- 自定义脚本放置目录：默认 `.hooks/`，是否修改？

#### Layer 3: 架构护栏（可选，适合成熟仓库）

```
Layer 3 — 架构护栏（可选）：
  ○ 依赖方向检查            自定义脚本检查 import 路径合规
  ○ 活动计划完成度守护      push 前检查 docs/plans/ 未完成项
  ○ 代码生成一致性检查      确认生成文件未被手改

启用哪些？[选择/跳过]
```

Layer 3 的每一项都需要用户提供额外信息（模块边界定义、计划目录路径等），逐项收集。

### Step 4: 生成配置

基于用户确认的选项，生成：

1. **框架配置文件** — 完整的 `.pre-commit-config.yaml` / `lefthook.yml` / `.husky/pre-commit`
2. **自定义脚本** — Layer 2/3 中需要的 shell 脚本，带执行权限
3. **辅助文件** — 如 `.golangci.yml`（如果选了 golangci-lint 但不存在）

**生成前向用户展示完整产物列表**，确认后写入。

如果已有配置文件：
- 展示 diff（将追加的内容 vs 已有内容）
- 明确标注"新增"部分
- 征求确认后追加

### Step 5: 安装

执行对应框架的安装命令：

pre-commit：

```bash
pre-commit install
```

lefthook：

```bash
lefthook install
```

husky（通常 `npx husky init` 已在 Step 4 处理）。

### Step 6: 验证

安装后执行 dry-run 验证：

pre-commit — 对所有文件运行一次：

```bash
pre-commit run --all-files
```

lefthook — 运行 pre-commit hooks：

```bash
lefthook run pre-commit
```

husky — 手动触发：

```bash
.husky/pre-commit
```

**向用户报告结果**：
- 通过：✓ 已安装并验证，N 条检查全部生效
- 部分失败：列出失败项，询问是否需要修正（可能是已有代码不合规）
- 安装失败：诊断原因，提供修复建议

## Gotchas

- **已有配置绝不覆盖**。如果用户已有 `.pre-commit-config.yaml`，只能追加 repo/hook，不能重写整个文件
- **gitleaks 可能首次就报大量 false positive**。建议用户先跑一次 `gitleaks detect`，确认 baseline 后再加入 hook；或提供 `.gitleaksignore` 基线
- **golangci-lint 首次运行很慢**。提示用户首次可能需要等待缓存构建
- **husky 需要 `prepare` script**。确保 `package.json` 中有 `"prepare": "husky"` 或等价配置
- **文件行数检查的排除规则很重要**。生成代码、vendor、minified 文件必须排除，否则噪音太大
- **check-added-large-files 的阈值按仓库调整**。有些仓库需要提交二进制资源，阈值可能需要放宽到 1MB 或更高
- **mixed-line-ending 在 Windows 混合团队中可能有争议**。如果仓库有 Windows 开发者，需确认 `.gitattributes` 配置
- **Layer 3 脚本高度仓库特定**。不要用通用模板硬套，必须根据用户提供的模块边界信息定制
