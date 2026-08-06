# trace-report 用户使用手册

本文档面向 `xdev trace` 系列命令的使用者，由 trace-report skill 对应。内容涵盖自动上报、手动排查、可视化查看、日志诊断等所有日常场景。

---

## 一句话理解

`xdev trace` 自动采集你在 Claude Code / Codex / Coco / OpenCode / Trae 里的会话，上传到 TOS，并提供 Web Dashboard 和离线 HTML 报告两种可视化方式。TOS 凭证与 hook 在 `xdev --cc/--cdx/--coco` 启动时自动预置，**大多数用户无需手动配置**。

---

## 快速开始

### 最佳路径

```bash
# 1) 任选一个 agent 启动（自动完成 TOS 凭证预置 + hook 注入）
xdev --cc         # 或 --cdx / --coco

# 2) SSO 登录（首次）
xdev trace auth login

# 3) 正常使用 agent，会话结束后自动上报
#    想立刻确认上报成功:
xdev trace tail -n 5
```

### 什么会被自动配置

| 文件 | 作用 |
|---|---|
| `~/.trace/config.yaml` | TOS bucket / AK / SK（字段缺失才写入，不覆盖用户值） |
| `<cwd>/.claude/settings.json` | CC 项目级 hook：SessionStart / Stop / PostToolUse |
| `<cwd>/.codex/hooks.json` + `<cwd>/.codex/config.toml` | Codex hooks + `[features] codex_hooks = true` |
| `<cwd>/.coco/coco.yaml` | Coco 项目级 `hooks:` 字段（仅 trace 上报，Coco 自带代码统计） |

---

## 命令完整参考

### auth — 身份认证

```bash
xdev trace auth login       # SSO 登录（Device Code 流程，输出 URL 和 code）
xdev trace auth status      # 查看登录态（VALID / EXPIRED / Not logged in）
xdev trace auth logout      # 登出并删除 ~/.trace/credentials.json
```

### config — 配置管理

```bash
xdev trace config view                       # 查看当前完整配置（yaml）
xdev trace config set <key> <value>          # 设置配置项（支持 dotted key）

# 常用 key:
xdev trace config set tos.bucket my-bucket
xdev trace config set tos.accessKey MY_AK
xdev trace config set tos.secretKey MY_SK
xdev trace config set tos.region cn-shanghai
```

> 配置优先级（高到低）：环境变量（`TRACE_TOS_*`） > 项目级 `.trace/config.yaml` > 全局 `~/.trace/config.yaml` > 默认值。

### session-start — 记录 Git 基线（hook 用）

```bash
xdev trace session-start --session-id <id> [--cwd <dir>]
```

在会话开始时由 SessionStart hook 调用。记录会话当时的 git commit / branch / cwd 到 `~/.trace/sessions/<sessionId>.json`，用于后续 diff 计算。

### forward — 上报到 TOS

```bash
# 真实上传
xdev trace forward --file <path/to/session.jsonl>

# 可选参数
xdev trace forward --file <path> --source claude-code   # 显式指定工具类型
xdev trace forward --file <path> --cwd <dir>            # 指定 git 上下文目录
xdev trace forward --file <path> --dry-run              # 不上传，只生成本地产物
xdev trace forward --file <path> --dry-run --dry-run-output /tmp/debug
```

输出 JSON 含 `jsonlObjectKey`、`diffObjectKey`、`bucket`、`etag` 等字段。

### tail — 查看 hook 执行日志 ⭐ **新增**

查看由 hook 触发的 `xdev trace session-start` / `xdev trace forward` 最近的执行结果。适合「关掉 session 后立刻确认是否成功上报」场景。

```bash
xdev trace tail                       # 默认最近 10 行
xdev trace tail -n 50                 # 最近 50 行
xdev trace tail -f                    # 实时跟踪当天文件
xdev trace tail -F                    # 跟踪 + 跨日自动切换
xdev trace tail --date 2026-04-17     # 查看指定日期历史
xdev trace tail -n 20 --date 2026-04-15   # 组合用法
```

**日志位置**：`~/.trace/logs/hook.YYYY-MM-DD.log`（每日一个文件，权限 0600）

**日志格式**（行式，首字段 ISO timestamp + 事件类型 + JSON payload）：

```
2026-04-17T16:45:23.123+0800 session_start {"result":"ok","sessionId":"abc-123","gitCommit":"deadbeef","gitBranch":"main"}
2026-04-17T16:48:10.456+0800 forward {"result":"ok","sessionId":"abc-123","tool":"claude-code","jsonlObjectKey":"xtrace/bingwang/2026-04-17/claude-code/abc-123.session.tar.gz","diffObjectKey":"xtrace/...","bucket":"stone-costudio-boe","etag":"abc123"}
2026-04-17T16:50:01.789+0800 forward {"result":"error","sessionId":"xyz-789","errorCode":"NotAuthenticated","errorMessage":"Token expired, run xdev trace auth login"}
2026-04-17T16:52:30.000+0800 forward {"result":"dry-run","sessionId":"abc-999","tool":"claude-code","jsonlObjectKey":"xtrace/...","bucket":"...","dryRunOutput":"/tmp/trace-debug"}
```

**用 jq 做进阶查询**：

```bash
# 只看 forward 成功的 TOS URL
xdev trace tail -n 100 | awk '{print $2, $3}' | grep '^forward ' \
  | sed 's/^forward //' | jq -r 'select(.result=="ok") | .jsonlObjectKey'

# 统计今日成功/失败数
xdev trace tail -n 500 | awk '{print $3}' | sed 's/^/\{"line":/;s/$/\}/' > /dev/null
xdev trace tail -n 500 | awk '{print $2,$3}' | grep '^forward' \
  | sed 's/^forward //' | jq -s 'group_by(.result) | map({result:.[0].result, count:length})'
```

### analyze — 生成单文件 HTML 报告

针对单个 LeadAgent JSONL 生成可离线查看的 HTML，含 Trajectory 折叠树 + subagent drill-down。适合发给别人看或归档。

```bash
xdev trace analyze --file <path/to/lead-agent.jsonl>         # 默认 /tmp/xdev-analyze-<id>-<ts>.html
xdev trace analyze --file <path> --output ./report.html      # 自定义输出
xdev trace analyze --file <path> --no-open                   # 不自动打开浏览器
xdev trace analyze --file <path> --agent cc                  # 指定 agent 类型（v1 只支持 cc）
```

### serve — Web Dashboard

启动 express + React SPA 的本地 Dashboard，从 TOS 拉取会话做多会话分析、LLM-Judge 评估、Trae 导入等。

```bash
xdev trace serve                 # 默认 :3210，自动打开浏览器
xdev trace serve --port 8080     # 自定义端口
xdev trace serve --no-open       # 不开浏览器
```

### url — 单 key → 签名 URL（可分享）

```bash
xdev trace url <key>                     # 默认 24h 过期
xdev trace url <key> --expires 1h        # 1h / 24h / 7d / 3600(秒)
xdev trace url <key> --open              # 顺便用默认浏览器打开
xdev trace url <key> --copy              # 复制到剪贴板
```

输出形如：

```
http://stone-costudio-boe.tos-cn-north-boe.byted.org/xtrace%2F.../abc.jsonl.gz?tos-algorithm=TOS-HMAC-SHA256&tos-expiration=...&tos-signature=...
```

可贴给任何人在浏览器打开、curl 下载，对方不用装 xdev。

### get — 单 key → 本地文件

```bash
xdev trace get <key>                     # 默认保存到 ./<basename>
xdev trace get <key> -o ./session.gz     # 自定义输出路径
xdev trace get <key> --gunzip            # 下载+解压,输出 ./<basename-without-.gz>
```

objectKey 前导 `/` 可直接粘贴（CLI 自动 trim）。

### download — 批量下载最近 N 个 session

```bash
xdev trace download                       # 默认: 下载最近 5 个到当前目录
xdev trace download -n 3 --save /tmp/x    # 指定目录
xdev trace download -n 10 --open          # 逐个浏览器打开(>10 会提示防爆)
xdev trace download -n 5 --url            # 只打印 5 条签名 URL
xdev trace download --with-diff           # 同时下载 diff.gz
xdev trace download --tool claude-code --date 2026-04-17 -n 20
xdev trace download --session abc         # 过滤 sessionId 前缀
```

**数据源**：hook 日志 `~/.trace/logs/hook.*.log` 里的 `forward:ok` 记录，不扫 TOS bucket。同 sessionId 自动去重保留最新。

**三个动作 flag 互斥**：`--save <dir>` / `--open` / `--url`，不传走默认 `--save .`。

### open — 傻瓜模式 ⭐

```bash
xdev trace open                           # 最近 5 个 session, 浏览器批量打开(自动下载)
xdev trace open -n 3
xdev trace open --tool claude-code
```

等价于 `xdev trace download --open`，语义更清晰，参数更精简（只保留核心 filter）。

---

## 常见场景

### 「我刚退出 Claude，想确认上报成功」

```bash
xdev trace tail -n 3     # 看最近 3 条执行记录
# 预期看到:
#   ...forward {"result":"ok","jsonlObjectKey":"xtrace/..."}
```

如果 `result=error`，根据 `errorCode` 处理：
- `NotAuthenticated` → `xdev trace auth login`
- `FileNotFoundError` → 确认 transcript 路径
- `AdapterDetectionError` → 加 `--source claude-code` 手动指定

### 「我想看今天所有 session 上报了哪些 TOS 对象」

```bash
xdev trace tail -n 500 | grep -o '"jsonlObjectKey":"[^"]*"' | sort -u
```

### 「我想把刚才的 session 发给同事看」

```bash
# 方式 1: 拿最新一条 key 生成签名 URL,复制到剪贴板,贴到 IM
xdev trace tail -n 1 | awk '{print $3}' | sed 's/.*"jsonlObjectKey":"\([^"]*\)".*/\1/' \
  | xargs -I{} xdev trace url {} --copy

# 方式 2: 直接下载到本地,发附件
xdev trace download -n 1 --save ~/Desktop/
```

### 「我想批量下载昨天调试的一堆 session」

```bash
xdev trace download --date 2026-04-16 -n 50 --save ~/debug-sessions/ --with-diff
```

### 「我要分享最近几个 session 给别人在浏览器里点开看」

```bash
xdev trace open -n 3
# 会一次开 3 个浏览器标签页,各自触发 .jsonl.gz 下载
```

### 「我想实时监控 hook 触发情况」

```bash
xdev trace tail -F       # 在另一个终端开着，准备下次操作时能实时看到
```

### 「我想调试 hook 会上报什么内容（不实际上传）」

```bash
# 找到最近的 transcript
LATEST=$(ls -t ~/.claude/projects/*/*.jsonl | head -1)
# 跑 dry-run
xdev trace forward --file "$LATEST" --dry-run --dry-run-output /tmp/debug
# 看产物
ls -la /tmp/debug
cat /tmp/debug/*.metadata.json | jq .
```

### 「我想用自己的 TOS 账号（不用 xdev 预置的）」

```bash
xdev trace config set tos.accessKey MY_AK
xdev trace config set tos.secretKey MY_SK
# 之后 xdev 启动不会再覆盖（字段缺失才预置）
```

### 「我想在不同项目间隔离 TOS bucket」

在项目根目录创建 `.trace/config.yaml`：

```yaml
tos:
  bucket: project-specific-bucket
```

项目级会覆盖全局设置。

---

## 诊断清单（对应 skill 的 §B 健康检查）

```bash
xdev trace tail -n 10         # 1) 最近 hook 执行日志（最快速）
xdev trace auth status        # 2) 登录态
xdev trace config view        # 3) TOS 配置
ls -lt ~/.trace/sessions/     # 4) SessionStart 是否在跑
ls ~/.trace/logs/             # 5) 每日 hook 执行日志
```

---

## 重要提示

- AK/SK 是敏感信息，不要在对话中明文粘贴。xdev 已预置默认凭证（最小权限 put-only），需要自定义时用 `xdev trace config set` 自己设。
- `async: true` 的 hook（CC）由 agent 丢弃 stdout/stderr，因此**唯一可靠的执行证据是 `~/.trace/logs/hook.*.log`**（由 `xdev trace` 子命令内部写）。
- 老的 `xtrace` 顶层命令已废弃；统一用 `xdev trace ...`。
- TOS object key 保留 `xtrace/` 前缀（数据契约）。
