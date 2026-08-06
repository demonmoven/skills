# 2.5.2 Hooks

> **本节目标**：理解 Hooks 在 Agent 工作流中的两层含义——Coding Agent Hooks（Agent 行为钩子）和 Git Hooks（代码质量门禁），以及各自的配置方式和最佳实践。

---

## 概念与历史

### 什么是 Hooks

Hooks 是在**特定事件发生时自动执行的 shell 脚本**。在 Agent 工作流中，Hooks 有两层含义：

1. **Coding Agent Hooks**：Agent 执行特定操作（如调用工具、写文件）前后触发的 shell 命令——由 Claude Code 首创，Codex CLI 和 Coco 随后跟进
2. **Git Hooks（pre-commit / pre-push）**：代码提交/推送时触发的质量检查——传统 Git 能力，但在 Agent 时代被赋予了新的使命（AI Guard）

两者的共同目标：**让约束通过脚本自动执行，不依赖 Agent 自觉**。

---

## Coding Agent Hooks

### 本质：事件 + 匹配器 + Shell 脚本

Coding Agent Hooks 的核心机制非常简单：**当 Agent 触发某个事件时，如果事件匹配 matcher 规则，就执行一段 shell 命令**。

```text
事件（Event）         匹配器（Matcher）        Shell 脚本（Command）
─────────────      ────────────────      ─────────────────────
Agent 要调用 Bash   → matcher: "Bash"     → 执行 check-dangerous.sh
Agent 写完了文件     → matcher: "Write"    → 执行 auto-format.sh
Agent 停止运行       → matcher: (任意)     → 执行 notify-slack.sh
```

### 钩子勾的是什么？


| 事件                 | 触发时机             | 钩子能做什么                    |
| ------------------ | ---------------- | ------------------------- |
| `PreToolUse`       | Agent 调用工具**之前** | 检查、拦截（exit code 2 = 阻止执行） |
| `PostToolUse`      | Agent 调用工具**之后** | 格式化、日志、通知                 |
| `Notification`     | Agent 发出通知时      | 转发到飞书/Slack               |
| `Stop`             | Agent 停止运行时      | 清理临时文件、发送完成通知             |
| `UserPromptSubmit` | 用户提交 prompt 时    | 敏感信息检测、prompt 日志          |


### Matcher 匹配的是什么？

`matcher` 是一个字符串，匹配**工具名称**：


| matcher 值        | 匹配的工具               |
| ---------------- | ------------------- |
| `"Bash"`         | Agent 调用 Bash 执行命令时 |
| `"Write"`        | Agent 写文件时          |
| `"Edit"`         | Agent 编辑文件时         |
| `"Read"`         | Agent 读文件时          |
| `""` 或 `"*"` 或省略 | 匹配所有工具调用            |


### Shell 脚本怎么工作

钩子本质就是一段 shell 脚本/命令，通过 **stdin 接收 JSON 输入**（包含工具调用的参数），通过 **exit code 控制行为**：


| Exit Code | 含义                      |
| --------- | ----------------------- |
| `0`       | 通过，继续执行                 |
| `2`       | 阻止（仅 PreToolUse，阻止工具执行） |
| 其他        | 报错但不阻止                  |


### 各 Coding Agent 的配置方式

#### Claude Code

配置在 `settings.json`（项目级 `.claude/settings.json` 或全局 `~/.claude/settings.json`）：

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "bash .hooks/check-dangerous-commands.sh"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Write",
        "hooks": [
          {
            "type": "command",
            "command": "bash .hooks/auto-format.sh"
          }
        ]
      }
    ]
  }
}
```

管理命令：在 Claude Code 中输入 `/hooks` 浏览所有已配置的钩子。

> 官方文档：[Automate workflows with hooks](https://code.claude.com/docs/en/hooks-guide)

#### Codex CLI

配置在 `hooks.json`（项目级 `.codex/hooks.json` 或全局 `~/.codex/hooks.json`）：

```json
{
  "PreToolUse": [
    {
      "matcher": "Bash",
      "hooks": [
        {
          "type": "command",
          "command": "python3 .codex/hooks/pre_tool_use_policy.py",
          "statusMessage": "Checking Bash command"
        }
      ]
    }
  ]
}
```

> 官方文档：[Codex CLI Hooks](https://developers.openai.com/codex/hooks)

#### Coco / TRAE CLI

配置在 `.coco/coco.yaml`（项目级）或 `~/.coco/coco.yaml`（全局）：

```yaml
hooks:
  PreToolUse:
    - matcher: "Bash"
      command: "bash .hooks/check-dangerous-commands.sh"
```

---

## Git Hooks（AI Guard）

### 概念

Git pre-commit / pre-push hooks 是 Agent 代码提交前的**最后一道防线**。在 Agent 时代尤为重要——AI 生成的代码更容易出现占位符、幻觉标记、超长文件等问题。

### 安装

使用 `pre-commit` 框架（Python）或 `husky`（Node.js）统一管理：

```bash
# pre-commit 框架
pip install pre-commit
pre-commit install

# 或使用 prek（pre-commit 的 alias）
prek install --install-hooks

# 或使用 husky（Node.js 项目）
npx husky install
```

### AI Guard：专为 Agent 设计的 Hook


| 检查项    | 拦截内容                                            | 说明            |
| ------ | ----------------------------------------------- | ------------- |
| 占位符检测  | `TODO implement` / `STUB` / `PLACEHOLDER`       | AI 常生成未完成的桩代码 |
| 幻觉标记检测 | `<PLACEHOLDER>` / `<INSERT_HERE>` / `<FILL_IN>` | AI 的特有标记      |
| 文件长度检查 | 超过 600 行的文件                                     | AI 倾向生成超长文件   |
| 活跃计划检查 | 未完成的 ExecPlan                                   | 防止"半完成"代码被推送  |


---

## 最佳实践

### 目录结构

```text
项目根/
├── .pre-commit-config.yaml        ← pre-commit 框架配置
└── .hooks/                        ← 自定义 hook 脚本
    ├── check-file-length.sh       ← AI Guard: 文件行数上限
    ├── check-active-plans.sh      ← AI Guard: 活跃计划完成度
    ├── go-fmt.sh                  ← Go 格式化
    ├── go-mod-tidy.sh             ← Go 依赖一致性
    └── no-commit-to-main.sh       ← 禁止直接提交到 main
```

### Hook 脚本怎么写

以 `check-file-length.sh` 为例——一个典型的 AI Guard hook：

```bash
#!/usr/bin/env bash
# AI 代码守护: 文件行数上限检查
set -euo pipefail

MAX_LINES=600
exit_code=0

for f in "$@"; do
    case "$f" in
        docs/*|*.md) continue ;;   # 文档不限制行数
    esac
    lines=$(wc -l < "$f")
    if [ "$lines" -gt "$MAX_LINES" ]; then
        echo "❌ $f: ${lines} 行 (上限 ${MAX_LINES} 行, 请拆分文件)"
        exit_code=1
    fi
done

exit $exit_code
```

要点：

- **报错信息要有指导性**——Agent 看到 "❌ file: 700 行 (上限 600 行, 请拆分文件)" 就知道怎么修
- **排除合理的例外**——`docs/*|*.md` 不限制行数
- 脚本通过 `$@` 接收文件列表（pre-commit 框架自动传入）

### DevClaw 的完整 Hook 套件（18 条）


| 类别           | 检查项                              | 触发时机     | 说明                                 |
| ------------ | -------------------------------- | -------- | ---------------------------------- |
| **文件卫生**     | trailing-whitespace              | commit   | 行尾空格                               |
|              | end-of-file-fixer                | commit   | 文件末尾换行                             |
|              | check-yaml / json / toml         | commit   | 格式校验                               |
|              | check-added-large-files (>500KB) | commit   | 大文件拦截                              |
|              | check-merge-conflict             | commit   | 冲突标记                               |
|              | mixed-line-ending → LF           | commit   | 统一换行符                              |
|              | fix-byte-order-marker            | commit   | 清除 BOM                             |
| **安全防护**     | gitleaks                         | commit   | 密钥/凭据泄露检测                          |
|              | no-commit-to-main                | commit   | 禁止直接提交到 main                       |
| **Go 质量**    | go fmt                           | commit   | 格式化                                |
|              | go build                         | commit   | 编译检查                               |
|              | go mod tidy                      | commit   | 依赖一致性                              |
|              | golangci-lint (funlen ≤120)      | commit   | 静态分析                               |
| **前端质量**     | biome check --fix                | commit   | lint + format                      |
|              | tsc --noEmit                     | commit   | TypeScript 类型检查                    |
| **AI Guard** | 禁止占位符代码                          | commit   | `TODO implement` / `STUB` / `HACK` |
|              | 禁止幻觉标记                           | commit   | `<PLACEHOLDER>` / `<INSERT_HERE>`  |
|              | 文件行数 ≤600 行                      | commit   | AI 倾向生成超长文件                        |
| **计划守护**     | active plans 无未完成项               | **push** | 未完成阻止推送，全完成自动归档                    |


### 钩子应该勾什么才最有效

从 DevClaw 的实践总结，**最有价值的 Hook 按优先级排序**：

1. **AI Guard 类**（投入产出比最高）
  - 占位符 / 幻觉标记检测——Agent 最常犯的错误，零成本拦截
  - 文件长度检查——防止 Agent 生成超长文件导致后续维护困难
2. **安全类**
  - gitleaks 密钥检测——Agent 可能在代码中内嵌示例 API Key
  - 禁止直接提交 main——防止 Agent 跳过分支流程
3. **编译/类型检查**
  - `go build` / `tsc --noEmit`——确保代码至少能编译通过
4. **格式化**
  - `go fmt` / `biome check`——Agent 提交时自动格式化，减少噪音 diff
5. **计划守护**（push 阶段）
  - active plans 检查——防止"半完成"代码被推送到远端

**核心原则**：Hook 是 CI/CD 质量门禁的**左移**——提前暴露问题并由 Agent 自动修复，比进了 CI 打回来再修更敏捷。

---

## Good Case vs Bad Case

### Good Case

- DevClaw 18 条 hook：文件卫生 + 安全 + Go/TS 质量 + AI Guard + 计划守护
- 报错信息用中文且有指导性："❌ file: 700 行 (上限 600 行, 请拆分文件)"
- pre-push hook 自动归档已完成的 ExecPlan

### Bad Case


| 错误                                 | 后果                                |
| ---------------------------------- | --------------------------------- |
| 没有任何 pre-commit hook               | Agent 提交带 `TODO implement` 的代码到仓库 |
| Hook 太多太慢（>30s）                    | Agent 和开发者都想绕过                    |
| 使用 `--no-verify` 跳过 hook           | 所有门禁形同虚设                          |
| Hook 报错信息不清晰（只报 "failed"）          | Agent 不知道怎么修复，反复重试                |
| 只有 Git Hooks 没有 Coding Agent Hooks | Agent 调用危险命令时无法实时拦截               |


---

[上一节：SubAgent（流程编排）](025a-subagent-flow.md) | [返回上级：流程控制](025-flow-control.md)