---
name: gloop-iteration-workflow
version: 1.0.0
description: "Gloop 自身迭代工作流：agent（卡车呆呆）在飞书群与用户 HOTL 协作迭代 gloop 时的工作方式。典型触发：用户提出 gloop 的 bug/优化/架构问题、需要发版部署、需要评估某项设计是否全局最优。不负责：普通 quest 执行（走 gloop-quest-execution）、纯上下文查询（走 gloop-self-awareness）。"
metadata:
  class: warrior
  category: execution
  kind: orchestration
  related_skills:
    - name: gloop-self-awareness
      type: related
      description: 迭代前先查当前版本、配置、quest 状态
    - name: gloop-quest-execution
      type: related
      description: gloop 迭代本身就是 quest 执行的特例
  requires:
    bins: ["gloop", "git", "node", "go"]
    cliHelp: "gloop --help"
---

# gloop-iteration-workflow

agent（卡车呆呆）在飞书群与用户 HOTL 协作迭代 gloop 自身的工作方式。核心原则：**自主端到端执行，用户只感知影响不逐步 review**。

## Trigger Examples

| 类型 | 场景 | 说明 |
|---|---|---|
| 适用 | 用户报 bug | "首页显示 7 个需介入"、"iOS 点输入框放大" |
| 适用 | 用户要优化 | "并发数合理吗"、"存储使用合理吗" |
| 适用 | 用户要架构评估 | "换 Rust 会更优吗" |
| 适用 | 用户要发版 | "直接改代码，全局最优" |
| 不适用 | 普通开发任务 | 走 gloop-quest-execution |

## 协作模型

### HOTL（Human-on-the-loop）

用户偏好自主端到端执行，不逐步 review。agent 的职责：
1. **接到指令直接干**，不问"要我怎么做"，先从第一性原理分析根因
2. **自主完成全链路**：分析 → 改代码 → build → 部署 → 测试 → commit → tag → bnpm 发布 → changelog → 飞书汇报
3. **只在必要时汇报**：发版完成、遇到分歧、需要用户决策时才 botmux send
4. **感知影响**：用户通过飞书和 dashboard 感知结果，不通过 diff review

### 飞书汇报规则

- `botmux send --mention-back`：有实质结论、要用户确认/决策时
- `botmux send --no-mention`：纯进度更新、低优先级
- 多行必须用 heredoc，禁止 `\n` 字面量
- 汇报内容：根因 + 做了什么 + 验证 + 部署状态 + dashboard 链接

## 迭代工作流（单次发版全链路）

### 1. 分析根因（第一性原理）

- 先看实际数据，不猜：quest 状态分布、配置值、文件体积、日志
- 找到决定性因素：瓶颈在哪、根因是什么、什么是全局最优
- 评估"不做"是否更优：有些优化是伪优化（如 P2/P3 评估）

### 2. 改代码

- 改代码默认值而非用户 config（让所有用户受益）
- 加配置版本迁移（旧 config 自动升级，参考 `applyDefaults` + `semverLess`）
- LSP 误报常见（"use of internal package"、"undefined: Engine"），以 `go build` 为准
- 改完跑测试：`go test ./internal/fsstore/ -run "TestXXX" -v`

### 3. Build + 部署

```bash
# 前端
cd web && npm run build

# 后端 + 版本 bump
# 1. internal/version/version.go: var Version = "0.2.X"
# 2. package.json: "version": "0.2.X"
mkdir -p ~/.gloop/bin/versions/v0.2.X
go build -o ~/.gloop/bin/versions/v0.2.X/gloop ./cmd/gloop
~/.gloop/bin/versions/v0.2.X/gloop version  # 验证

# 更新 wrapper
# ~/.gloop/bin/gloop: exec "$DIR/versions/v0.2.X/gloop" "$@"

# 重启
~/.gloop/bin/gloop stop; sleep 1; ~/.gloop/bin/gloop start
```

### 4. 验证

- 看启动日志：`grep -i "storage\|error\|warn" ~/.gloop/server.log`
- 跑临时测试验证运行时行为（internal 包不能外部引用，写 `*_test.go` 放包内跑完删）
- 实际数据验证：`du -sh`、`ls`、quest 状态统计

### 5. Commit + Tag + 发布

```bash
git add -A
git commit -m "fix(...): 描述 v0.2.X

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
git tag v0.2.X
node scripts/release-bnpm.js  # 注意：必须在 repo 根目录跑，cd web 后会漂
git add -A && git commit -m "chore: refresh v0.2.X checksums ..."
```

### 6. Changelog + 汇报

- `docs/changelog-v0.2.X.md`：现象 + 根因 + 修复 + 评估为不做的
- `botmux send --mention-back`：根因 + 做了什么 + 验证 + 部署状态

## CLI Contract

本技能是 orchestration 类型，不绑定单一 CLI syscall。迭代过程中用到的 CLI 命令散布在各步骤里：

- `gloop` quest 管理、状态查询（见 gloop-self-awareness / gloop-quest-execution）
- `git`、`go`、`node` 用于发版全链路
- `botmux send` 飞书汇报

具体命令示例见下方「迭代工作流」各步骤。

## Discipline

- **cwd 漂移**：`cd web && npm run build` 后 cwd 变成 web/，`node scripts/release-bnpm.js` 会找不到。用绝对路径或 cd 回根目录
- **applyDefaults 不覆盖非零值**：改代码默认值只影响新安装，已有 config 需要 migration（`semverLess` 判断 + 向上提升）
- **bnpm 不支持同版本覆盖**：发了 0.2.12 发现漏了，只能 bump 到 0.2.13
- **竞态根因误判**：readonly answer 不恢复，最初以为是 WorkspacePath 为空，实际是 reserveQuestRuntime 与 releaseQuestRuntime 的 running map 残留竞态
- **flushCommentCursor edit 字符串不匹配**：文件有注释行，Read 确认后再 edit
- **LSP 误报**：`go build` 通过就别管 LSP 报的 "undefined: Engine"

## 版本节奏参考

v0.2.2 → v0.2.14 在一天内连发，每版一个小修复 + bnpm 发布。HOTL 模式下迭代速度是优势，保持高频小步发版。
