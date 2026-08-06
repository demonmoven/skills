---
name: trace-report
description: "AI Coding 会话 trace 上报与诊断（融合自 xtrace）。当用户提到「上报 trace」「上传会话」「配置 trace」「trace login」「trace 状态」「检查 trace」「dry run trace」「调试 trace」「trace dashboard」「trace serve」「trace analyze」时使用。底层调用 xdev trace 子命令。"
---

# Trace Report Skill

通过 `xdev trace` 完成 AI Coding 会话的 trace 采集、上报、诊断与可视化。所有实际操作都通过 Bash 工具调用 CLI 子命令完成，本 skill 只负责理解意图、编排步骤、解释结果。

> **本轮（2026-04-17）合并**：xdev 启动 `--cc` / `--cdx` / `--coco` 时会自动完成两件事：
> 1. 将 TOS 凭证预置到 `~/.trace/config.yaml`（字段缺失才写入，不覆盖用户自定义）
> 2. 将 trace 上报 hook（以及 AI 代码统计 hook，Coco 除外）增量写入项目级配置（`.claude/settings.json` / `.codex/hooks.json` + `.codex/config.toml` / `.coco/coco.yaml`）
>
> 因此绝大多数用户**不再需要手动配置**，只需启动 xdev 即可开箱即用。本 skill 更多用于**健康检查、故障排查、手动补传、可视化查看**。

## 适用分支

按用户意图判断进入哪一个分支。意图不明确时用 AskUserQuestion 询问。

| 分支 | 触发关键词 |
|---|---|
| §A 首次接入向导 | "配置 trace"、"初始化 trace"、"trace login"、"第一次用 trace"、"接入 trace" |
| §B 健康检查 | "trace 状态"、"trace 在工作吗"、"检查 trace"、"trace 配置对吗" |
| §C 手动上报 | "上报 trace"、"上传当前会话"、"forward 一下" |
| §D dry-run 调试 | "dry run trace"、"调试 trace 上报"、"看下 trace 会上报什么" |
| §E Web Dashboard | "trace dashboard"、"trace serve"、"看 trace 可视化"、"多会话分析" |
| §F 本地 HTML 报告 | "trace analyze"、"本地 trace 报告"、"把这个 session 生成 html"、"trajectory 折叠视图" |
| §G 分享与下载 | "分享 trace"、"给别人看 session"、"下载 session 原文件"、"trace 签名 URL"、"批量下载 session" |

---

## §A 首次接入向导

新机器首次使用 trace 的流程。xdev 启动已自动预置 TOS 凭证与 hook，**通常只差 SSO 登录这一步**。

### Step A1：SSO 登录

```bash
xdev trace auth login
```

CLI 输出 URL + user code，引导用户在浏览器打开链接、输入 code 完成授权。授权完成后 CLI 打印 "Login successful! User: ..."。

### Step A2：验证自动预置

```bash
xdev trace config view
```

确认 `tos.bucket` / `tos.accessKey` / `tos.secretKey` 都已有值（可能 mask 显示）。若仍然为空，说明 xdev 尚未启动过任一 agent，让用户先跑一次 `xdev --cc` / `xdev --cdx` / `xdev --coco`（任选其一）即可自动预置。

### Step A3：验证 hook 已就位

选一个 agent 启动过，对应配置文件应出现 trace hook：

| Agent | 检查文件 | 关键字 |
|---|---|---|
| `xdev --cc` | `<cwd>/.claude/settings.json` | `xdev trace session-start` |
| `xdev --cdx` | `<cwd>/.codex/hooks.json` + `<cwd>/.codex/config.toml`（含 `[features] codex_hooks = true`） | 同上 |
| `xdev --coco` | `<cwd>/.coco/coco.yaml` 顶层 `hooks:` 列表 | 同上 |

```bash
grep -l "xdev trace session-start" .claude/settings.json .codex/hooks.json .coco/coco.yaml 2>/dev/null
```

### Step A4：完成

输出报告：

```
trace 首次接入完成 ✅
- SSO 登录：OK（用户：<userId>）
- TOS 凭证：已预置
- Hook 写入：<发现的配置文件列表>

下次开 AI coding 会话时会自动上报 trace 到 TOS。
```

> 如果用户**不想要 xdev 预置的凭证**（比如想用自己的 TOS 账号），可以手动覆盖：
> ```bash
> xdev trace config set tos.accessKey <MY_AK>
> xdev trace config set tos.secretKey <MY_SK>
> ```
> 由于 `applyBundledCredsIfMissing` 采用「字段缺失才写」策略，用户手动 set 过后，后续 xdev 启动不会覆盖。

---

## §B 健康检查

诊断 trace 当前是否在工作。按顺序跑下面几条并解读结果。

### Step B0：看最近 hook 执行日志（最快速）

```bash
xdev trace tail -n 10            # 最近 10 条执行记录
xdev trace tail -f               # 实时跟踪当天
xdev trace tail -F               # 跟踪 + 跨日自动切换
xdev trace tail --date 2026-04-17 # 看指定日期
```

日志位于 `~/.trace/logs/hook.YYYY-MM-DD.log`，每行格式 `ISO-timestamp event JSON-payload`。看到 `event=forward result=ok` 且 `jsonlObjectKey=xtrace/...` 说明上报成功。

### Step B1：检查登录态

```bash
xdev trace auth status
```

| 输出 | 解读 |
|---|---|
| `User: ... / Status: VALID` | 登录态有效 ✅ |
| `Status: EXPIRED` | Token 过期 → 让用户跑 `xdev trace auth login` |
| `Not logged in` | 未登录 → 让用户跑 `xdev trace auth login` |

### Step B2：检查配置

```bash
xdev trace config view
```

检查输出 yaml：`tos.bucket` / `tos.accessKey` / `tos.secretKey` 不为空。全空说明 xdev 启动预置没跑过（或 bundle 凭证被清空），让用户跑一次任意 `xdev --cc/--cdx/--coco`。

### Step B3：检查最近的 session 状态

```bash
ls -lt ~/.trace/sessions/ 2>/dev/null | head -5
```

| 输出 | 解读 |
|---|---|
| 列出近期 session 文件 | session-start hook 在工作 ✅ |
| 目录不存在 / 空 | session-start hook 没在跑 → 执行 §B4 |

### Step B4：检查 Hook 是否写入项目级配置

根据用户在用哪个 agent，检查对应文件：

```bash
# Claude Code
jq '.hooks.SessionStart, .hooks.Stop, .hooks.PostToolUse' .claude/settings.json 2>/dev/null

# Codex
jq '.hooks' .codex/hooks.json 2>/dev/null
grep -A1 "\[features\]" .codex/config.toml 2>/dev/null  # 应含 codex_hooks = true

# Coco
grep -A1 "^hooks:" .coco/coco.yaml 2>/dev/null
```

如果对应 agent 的文件中没有 `xdev trace session-start` / `xdev trace forward`：
- 引导用户退出当前 agent，再用 xdev 重新启动（会自动注入 hook）
- 或让用户跑 `xdev --cc/--cdx/--coco` 一次即可（幂等，不会重复写）

### Step B5：诊断报告

汇总输出：

```
Trace 健康检查报告

✅ / ❌ 登录态：...
✅ / ❌ TOS 配置：bucket=<x>, ak=<masked>, sk=<masked>
✅ / ❌ Session 文件：最近一次 <时间>
✅ / ❌ Hook 配置：
    - .claude/settings.json:  ✅ / ❌ / N/A
    - .codex/hooks.json:      ✅ / ❌ / N/A（含 codex_hooks feature 检查）
    - .coco/coco.yaml:         ✅ / ❌ / N/A

诊断结论：<整体是否健康，最可能的修复动作>
```

---

## §C 手动上报

测试、补传或离线 hook 模式下手动触发一次 trace 上报。

### Step C1：定位 transcript 文件

```bash
ls -lt ~/.claude/projects/*/*.jsonl 2>/dev/null | head -10   # Claude Code
# OpenCode / Codex / Coco 的 transcript 位置不同，向用户询问
```

### Step C2：上报

```bash
xdev trace forward --file "{TRANSCRIPT}"
# 可选：--source claude-code / opencode 显式指定类型
```

CLI 输出 JSON 含 `jsonlObjectKey` / `diffObjectKey` / `etag`。

### Step C3：解读结果

```
✅ Trace 上报成功

JSONL: <bucket>/<objectKey>
Diff:  <bucket>/<objectKey>  (如有)
```

命令报错场景：
- `Not authenticated` → `xdev trace auth login`
- `File not found` → 检查 `{TRANSCRIPT}` 路径
- `Cannot detect tool type` → 加 `--source claude-code` 或 `--source opencode`

---

## §D dry-run 调试

不实际上传到 TOS，只生成本地产物供检查。

### Step D1：跑 dry-run

```bash
xdev trace forward --file "{TRANSCRIPT}" --dry-run --dry-run-output /tmp/trace-debug
```

### Step D2：检查产物

```bash
ls -la /tmp/trace-debug
# <sessionId>.jsonl.gz / .diff.gz / .metadata.json
```

### Step D3：解读 metadata

```bash
cat /tmp/trace-debug/*.metadata.json | jq .
```

重点：`uploadMeta.userId` / `gitUrl` / `jsonlObjectKey`（格式 `xtrace/<userId>/<date>/<tool>/<sessionId>.jsonl.gz`）/ `objectMetadata.x-tos-meta-*`。

---

## §E Web Dashboard（多会话可视化分析）

基于 express + React SPA 的本地 Dashboard，从 TOS 列举会话并分析。适合查看多个会话、成本趋势、LLM-Judge 评估、Trae thread 导入等。

### Step E1：启动服务

```bash
xdev trace serve               # 默认 :3210，自动打开浏览器
xdev trace serve --port 8080   # 自定义端口
xdev trace serve --no-open     # 不自动开浏览器
```

### Step E2：Dashboard 功能

- **会话列表**：从 TOS 列举用户上报的会话，支持按日期、工具类型筛选
- **Summary Tab**：时长、成本、turn 数、工具调用分布饼图
- **Conversation Tab**：消息时间线
- **Cost Tab**：成本折线、token 堆叠柱
- **Trajectory Tab**：效率图、工具调用频率、延迟分布
- **LLM-as-Judge 评估**（按需触发）：8 维度评分 + Harness 文件使用分析
- **Trae thread 导入**：输入 Fornax thread id + 时间范围，自动拉取并上传到 TOS

### Step E3：常见问题

- 页面空白 → 确认 `xdev trace auth status` 登录有效
- "No sessions found" → 确认 `~/.trace/config.yaml` 的 bucket 正确，以及最近是否上报过
- Trae 导入报 "FORNAX_AK missing" → 前端导入对话框直接输入 AK/SK（一次性，不持久化）

---

## §F 本地 HTML 报告（Trajectory 折叠树）

针对单个会话生成**可离线查看的单文件 HTML**，包含 Summary/Conversation/Cost/**Trajectory** 4 个 Tab。Trajectory 视图支持 subagent drill-down、节点折叠展开、skill 过滤等高级交互，适合发给别人看或归档。

### Step F1：定位主 JSONL

同 §C1，找到 LeadAgent 的主 JSONL 文件（注意**不是**子 agent 的 trace）。

### Step F2：生成报告

```bash
xdev trace analyze --file <path/to/lead-agent.jsonl> --agent cc
# 默认输出 /tmp/xdev-analyze-<session>-<ts>.html 并自动打开
# 自定义输出：--output <path.html>
# 不自动打开：--no-open
```

### Step F3：对比 §E 选哪个

| 维度 | §E serve (Web Dashboard) | §F analyze (HTML 报告) |
|---|---|---|
| 数据源 | TOS 上已上报的会话 | 本地 JSONL 文件 |
| 形态 | 起 express 服务 + 浏览器 | 单文件 HTML，离线可用 |
| 多会话 | ✅ 会话列表 | ❌ 一次一个 |
| Trajectory 折叠树 | ✅ 基础版 | ✅ 专精版（subagent drill-down、skill 过滤） |
| LLM-Judge 评估 | ✅ | ❌ |
| Trae 导入 | ✅ | ❌ |
| 分享归档 | 需要对方能访问 TOS | 单文件发送即可 |

---

## §G 分享与下载

拿到 hook 日志里的 `jsonlObjectKey`（或从 `xdev trace tail` 看到的），可用以下命令分享/下载对应 TOS 对象。

### 最傻瓜：一键打开最近几个会话

```bash
xdev trace open              # 最近 5 个 session 各生成签名 URL,自动开浏览器(触发下载)
xdev trace open -n 3         # 取最近 3 个
xdev trace open --tool claude-code --date 2026-04-17
```

### 单个 key → 签名 URL（可分享）

```bash
xdev trace url <key>                     # 默认 24h 过期
xdev trace url <key> --expires 1h        # 1 小时
xdev trace url <key> --expires 7d        # 7 天
xdev trace url <key> --open              # 同时用默认浏览器打开
xdev trace url <key> --copy              # 复制到剪贴板
```

URL 例子（`http://stone-costudio-boe.tos-cn-north-boe.byted.org/...?tos-signature=...`）可贴给任何人在浏览器打开，对方不用装 xdev。

### 单个 key → 下载到本地

```bash
xdev trace get <key>                     # 默认保存到 ./<basename>.jsonl.gz
xdev trace get <key> -o ./session.gz     # 自定义输出
xdev trace get <key> --gunzip            # 下载+解压,输出 ./<basename>.jsonl
```

### 批量：最近 N 个 session

```bash
xdev trace download                       # 默认:下载最近 5 个到当前目录
xdev trace download -n 3 --save /tmp/x    # 下 3 个到指定目录
xdev trace download -n 10 --open          # 10 个全部浏览器打开(>10 会提示防爆)
xdev trace download -n 5 --url            # 只打 5 条 URL,一行一个
xdev trace download --tool claude-code --date 2026-04-17 --with-diff
```

### 关键事实

- **依赖 `tos.endpoint` 配置**（默认 xdev 已预置 `tos-cn-north-boe.byted.org`）；缺失会报错
- URL 里的签名与 `expiredAt` 绑定，过期后需重新生成
- `xdev trace download` 的数据来源是 **hook 日志**（`~/.trace/logs/hook.*.log`），不扫 TOS bucket — 想下载的 session 必须先通过 hook 上报过
- 同 sessionId 多次 forward 自动去重，保留最新一次
- objectKey 前导 `/` 可直接粘贴，CLI 内部 trim 处理

## 通用注意事项

- 所有命令通过 Bash 工具执行，**不要**让用户在对话中粘贴 AK/SK。
- 本轮合并后 TOS 凭证**默认已预置**（用户首次启动 xdev 即自动写入 `~/.trace/config.yaml`）；但凭证以 base64 形式存放在 xdev 源码中，只具备 TOS put 权限，不具备列举/删除/跨 bucket 权限。
- 上报失败不硬失败：CC/Coco 的 hook 带 `async: true`；Codex 的 hook 带 `timeout`，失败不影响主流程。
- TOS object key 仍以 `xtrace/` 前缀开头（数据契约保留）。
- `xtrace` 顶层命令已废弃，统一为 `xdev trace ...`。

## 相关命令速查

```
xdev trace auth login / status / logout         # SSO
xdev trace config set <key> <value> / view      # 配置
xdev trace session-start --session-id <id>      # 记录 Git 基线（hook 用）
xdev trace forward --file <path> [--dry-run]    # 上报 / dry-run
xdev trace analyze --file <path>                # 生成单文件 HTML 报告（§F）
xdev trace serve [--port 3210] [--no-open]      # 启动 Web Dashboard（§E）
xdev trace tail [-n N] [-f|-F] [--date X]       # 看 hook 执行日志（§B0 快速诊断）
xdev trace url <key> [--expires 24h] [--open] [--copy]    # 单 key 生成签名 URL（§G）
xdev trace get <key> [-o <path>] [--gunzip]               # 单 key 下载到本地（§G）
xdev trace download [-n N] [--save|--open|--url] ...      # 批量（§G）
xdev trace open [-n 5] [--tool X] [--date X]              # 傻瓜模式:最近 N 个自动浏览器打开（§G）
```
