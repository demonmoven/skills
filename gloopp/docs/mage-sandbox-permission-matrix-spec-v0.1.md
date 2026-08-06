# Mage 验证沙箱 + 行为矩阵权限 v0.1

> Status: DRAFT
> Owner: gloop architecture
> Related: mage 阶段保留 Bash + gloop CLI syscall；替代 mage 阶段"一刀切关 Bash"的临时方案

## 1. 背景与问题陈述

### 1.1 现状

Gloop 支持的 agent 都具备 Bash 和 skill 能力，因此阶段协议应统一为：

- agent 通过 skill 学习协议
- agent 通过 Bash 调用 `gloop ...` CLI
- CLI 子命令自己根据 `.gloop/context.json` / `GLOOP_*` 判断职业和阶段权限

Gloop 不再暴露 `Gloop` platform/native tool，也不保留 fenced `gloop` 旁路作为主协议。

### 1.2 过度一刀切的副作用

当前实现如果把 mage 的 Bash、Write、Edit 全部过滤掉（`effectiveAllowedTools` read-only 分支只保留 Read/Glob/Grep），会导致 mage：

1. **无法独立复现剑士声称的"已通过验证"**——`go build` / `go test` / `pytest` / `npm test` / `eslint` 等都需要在临时目录写编译产物、日志、coverage 文件；这些命令被一刀切禁止。
2. **法师退化为"静态扫描器"**——只能用肉眼读代码给评审意见，和"剑士自检 + 换个模型再读一遍"没有本质差异，双职业模型的独立验证价值归零。
3. **平台 P0-3 自动证据注入**变成了法师验证能力的唯一可信来源，但平台只覆盖非常窄的命令集合（`git status` / `diff` / 白名单 doctor）。
4. **gloop command run** 虽在 mage CLI 里，但只允许预设命令，扩展成本高，且不允许 mage 自由组合验证流水线。

### 1.3 从第一性原理看问题

gloop 双职业模型只依赖**一条公理**：

> **公理 S（审计独立性）**：审计者与生产者不能对"业务态"享有同一组修改权限，否则审计结论独立度为零。

由此衍生**三条约束**：
1. **权限独立**：法师不得写业务态（用户源码、quest 除评审字段外的元数据、phase 信号、diff stat）
2. **视角独立**：法师应该被允许从多个维度**独立验证**剑士交付——读 diff、跑构建、跑测试、跑 linter、打快照
3. **激励独立**：法师评分/经验应与"发现真问题的密度"挂钩，不与"和剑士达成一致"挂钩

→ **公理里没有任何一条禁止法师执行命令或写文件。** 被禁止的只是"写业务态"。当前实现把"临时验证态写入 + 执行验证命令"和"业务态写入"一起砍掉了，这才是真正的过度设计。

## 2. 设计目标

### 2.1 功能目标

- **F1：** mage 阶段可以执行任意验证命令（build / test / lint / type-check / 自定义），命令的 stdout/stderr / exit code 完整暴露给 mage。
- **F2：** 任何写入业务态的行为被强制隔离——**执行完命令后用户工作区和 quest 元数据与执行前 byte-for-byte 相同**。
- **F3：** mage 可以用 gloop CLI 写 note / 提交评审；不引入额外 platform/native tool。
- **F4：** mage 对工作区的任何写尝试（有意或无意）都被**捕获 + 审计 + 自动丢弃**，同时作为"评审者越权"信号进入僵局检测。
- **F5：** 所有 mage 行为（执行的命令、exit code、被丢弃的写入）可在 Quest Detail 页作为 evidence 展示。

### 2.2 非功能目标

- **NF1：** 沙箱 overhead ≤ 5%（相对在非沙箱里跑同样命令）
- **NF2：** 不依赖内核特权 / LSM / root；普通用户容器环境可运行
- **NF3：** warrior 阶段完全不受影响；改动仅在 `effectiveAllowedTools` 分支和新沙箱包内
- **NF4：** 失败降级——环境不支持沙箱时，保留普通 Bash CLI 路径，但继续依赖 `gloop command run` allowlist 和 CLI 权限校验

## 2.3 当前落地切片：copy sandbox

当前实现先落地最小可验证闭环，而不是一次性引入 overlayfs / bwrap：

- mage/read-only phase 启动时复制真实 quest workspace 到 `quests/<qid>/sandboxes/<sid>`。
- mage agent 进程的 `WorkingDir`、`GLOOP_WORKSPACE_PATH`、`GLOOP_CONTEXT` 都指向 sandbox copy，不暴露真实 workspace 绝对路径。
- 这对 Claude Code / Codex / TraeX 这类 SOTA agent 的内部 `Edit` / `Write` / `Task` / Skill 等能力也成立：子工具继承的是 sandbox 根，而不是只靠 Bash cwd 约束。
- `gloop command run` 在 `GLOOP_COMMAND_WORKDIR` 指向的 sandbox copy 中执行验证命令。
- `gloop review` / `gloop phase` / `gloop note` 通过 `GLOOP_DATA_DIR + GLOOP_QUEST_ID + GLOOP_SESSION_ID` 加载真实 quest 元数据，fallback signal 写回真实 quest workspace，而不是写进 sandbox。
- phase 结束时比较真实 workspace 与 sandbox，记录 `mage_sandbox_writes_captured` session row，然后删除 sandbox。真实 workspace 不应用 sandbox 里的写入。

这个切片不是强隔离安全容器：它不阻止 agent 主动访问系统其他绝对路径，也不约束网络和 L2 外部副作用。它解决的是当前最关键的工程边界：**法师内部工具对工作区的普通写入不会持久化污染被审计对象**。后续 overlayfs / bwrap 是增强隔离层，不改变协议模型。

## 3. 行为矩阵权限（替代"agent 类型一刀切"）

权限判定从 `if warrior: allow X; if mage: deny Y` 变为三维查表：**行为类型 × 写入对象归属 × 执行阶段**。

| 行为类型 | 业务态对象 | 临时验证态 | 审计态（.gloop/、评审字段、note） | 外部副作用态（L2） |
|---|---|---|---|---|
| **读（Read/Glob/Grep/git log）** | ✓ 全阶段 | ✓ 全阶段 | ✓ 全阶段 | N/A |
| **执行验证命令**（build/test/lint） | 禁止写入；执行时以 overlay/COW 挂到验证态 | ✓ mage / warrior 都允许 | N/A | 仅 allowlist 的域名/IP |
| **写验证产物**（build cache、log、cov、临时文件） | ✗ 违反权限独立 | ✓ mage 专属目录；exit 丢弃 | N/A | N/A |
| **写审计态**（评审、note、评审报告） | ✗ | N/A | ✓ mage 仅通过 `gloop ...` CLI 写 | N/A |
| **推进业务流转**（phase done、quest status） | ✗ mage / 仅 warrior 且受控 | N/A | N/A | N/A |
| **外部副作用**（HTTP、消息发送、部署） | N/A | N/A | N/A | 默认 ✗；必须 automation policy 或用户显式授权 |

### 3.1 对象归属判定

"写入属于哪个归属域"不是用路径前缀匹配，而是**用 mount namespace / overlay 的上层挂载点精确判定**：

```
业务态下层（lower）= 用户工作区 / quest 根
  ↓ overlayfs 挂载
验证态上层（upper）= /tmp/gloop-mage-sandbox/$qid-$sid/upper
工作目录（merged）= /tmp/gloop-mage-sandbox/$qid-$sid/merged
```

- 所有写入自动落到 upper 层 → 属于临时验证态，exit 时 `rm -rf upper` 即可丢弃
- lower 层永远只读，`chattr +i` 二次兜底（如果内核支持）
- 审计态（`.gloop/` 下的评审信号 / note）**走受控 `gloop ...` CLI**，不在沙箱内暴露 writable 路径

## 4. 架构：三层实现

```
 ┌──────────────────────────────────────────────────────────────────┐
 │                        orchestrator                              │
 │  micro_loop ─▶ phase 切换 ─▶ 判定当前是否为 mage_readonly_phase │
 └──────────────────────────────────────────────────────────────────┘
         │
         ▼
 ┌──────────────────────────────────────────────────────────────────┐
 │                  permission: behavior matrix resolver            │
 │  Phase × 行为类型 → 走"沙箱 Bash" / "gloop CLI" / "原生 Read"   │
 └──────────────────────────────────────────────────────────────────┘
         │                 │                │
         ▼                 ▼                ▼
  ┌─────────────┐  ┌────────────────┐  ┌───────────────────┐
  │Sandbox Bash │  │ gloop CLI      │  │ Read / Glob / Grep│
  │ (overlayfs) │  │  (cli_base.go) │  │ (unchanged)       │
  └──────┬──────┘  └────────────────┘  └───────────────────┘
         │
         ▼
  ┌─────────────────────────────────────────────────────┐
  │ gloop-sandboxd（轻量 helper，100 LOC，可选）         │
  │  · 建/拆 overlay mount                              │
  │  · 写审计：upper 层 diff、白名单外命令尝试、退出码   │
  │  · 降级：overlay 不支持 → chroot + rbind mount ro    │
  │  · 再降级：v0.1.21 一刀切关 Bash（baseline）         │
  └─────────────────────────────────────────────────────┘
```

### 4.1 SandboxBash：mage 阶段的新工具

mage 保留 Bash，但在后续沙箱版本中 Bash 的执行根被隔离到验证态：

- `gloop review ...` / `gloop note ...` / `gloop command run ...` 仍是普通 CLI 调用
- 直接写源码的命令只会写入 overlay upper 层，退出后丢弃
- `gloop command run` 继续由 `mage_command_allowlist` 控制验证命令集合

实现位置：`internal/sandbox/` 新增包。执行流程：

1. `SandboxBuilder.Build(qid, sid, readonlyRoot)` → 返回 `Sandbox` handle：
   - 探测 `unshare -Urm --mount` 是否可用（user namespace，不需 root）
   - 不可用 → 探测 plain `mount -t overlay`（fuse-overlayfs 路径）
   - 不可用 → 探测 `bwrap --ro-bind`（bubblewrap，如果系统有）
   - 都不可用 → 返回 `Sandbox{Mode: Degraded}`，降级到 v0.1.21
2. `Sandbox.Exec(ctx, argv, cwd)`：在新 mount namespace 里 `exec.CommandContext`
   - lowerdir = `workspace_root:quest_root`（按序，quest_root 覆盖业务态 lower 中的冲突字段不重要，都是只读）
   - upperdir = `$TMPDIR/gloop-mage-sandbox/$qid-$sid-$seq/upper`
   - workdir = `$TMPDIR/gloop-mage-sandbox/$qid-$sid-$seq/work`
   - merged 作为 bash `$PWD`
3. `Sandbox.Teardown()`：
   - `umount merged`；`rm -rf upper work`（写入整体丢弃）
   - **审计捕获**：teardown 前用 `diff -r lowerdir merged`（或直接走 upper 层 listing）得到"mage 试图修改的文件清单、大小、类型"，写入 `PhaseReviewArtifact.PlatformMechanisms` 字段，并 publish `mage_sandbox_writes_captured` 事件

### 4.2 effectiveAllowedTools 的 mage 分支重构

当前的实现：

```go
if opts.ReadOnly {
    for _, tool := range tools {
        if IsReadOnlySafeTool(tool) { out = append(out, tool) }
    }
    return out
}
```

新实现：

```go
if opts.ReadOnly {
    for _, tool := range tools {
        // Bash 用于调用 gloop CLI；Read 家族用于审计读取。
        if IsReadOnlySafeTool(tool) { out = append(out, tool); continue }
    }
    return out
}
```

### 4.3 micro_loop 拦截点的调整

- warrior 阶段：完全不变（`Bash` 原样）
- mage 阶段：
  - `Bash` 原生 tool 调用进入 `Sandbox.Exec`
  - `gloop review ...` / `gloop command run ...` 是 Bash 中执行的普通 CLI
  - 不再维护 `ParseGloopToolCalls` / fenced argv / native `Gloop` platform tool 旁路
  - 沙箱执行统一 publish 事件 + 审计落盘

## 5. 审计、僵局检测与展示

### 5.1 新增的事件与信号

| Event | 触发 | 用途 |
|---|---|---|
| `mage_sandbox_created` | 每轮 mage turn 开始 | 跟踪沙箱模式（NativeOverlay/Fuse/Degraded） |
| `mage_sandbox_exec` | 每条 mage Bash 命令完成 | 记录 cmd/cwd/exit/runtime |
| `mage_sandbox_writes_captured` | teardown 后 upper 非空 | 进入僵局检测：如果写入路径里出现 `*.go`/`*.ts` 等源码且 >10 次，认为 mage 在尝试"越权自修复"，发 policy 告警 → 自动降级为 request_changes 并要求剑士处理 |
| `mage_sandbox_degraded` | 沙箱不支持，退回一刀切 | 平台级可观测：统计部署环境里多少 quest 退化为只读静态扫描 |

### 5.2 僵局检测用新特征

`policy / deadlock_detector.go` 追加两个输入特征：

- `mage_sandbox_cmd_coverage = unique_basename_count(executed_commands)`，低覆盖率 + 高 pass 率 → 可能是法师放水（没验证）
- `mage_sandbox_write_volume = total_written_bytes_upper_layer`，高写入 + 结果 pass → 可能是越权修复

### 5.3 Quest Detail 页展示（Frontend）

在现有 `platform_auto_evidence` 下新增 **Mage Sandbox Evidence** 折叠面板：
- 沙箱模式（native/fuse-overlay/bwrap/degraded）
- 命令列表：`命令 / exit code / 用时 / 写了多少字节 / 是否在白名单`
- 捕获的写入清单：`路径 / 大小 / 修改时间（upper 层）`

## 6. 权限矩阵的统一 Resolver（替代 CLIExecutorBase 内硬编码）

把散落的"if warrior / if mage / if read-only"硬编码抽成**单一 resolver**，供 executor / orchestrator / sandbox 三处共用，后续扩展新职业（rogue 侦察、healer 修复等）只改矩阵不改分支。

```go
// internal/permissions/resolver.go
type ActorClass string
type BehaviorKind string
type TargetDomain string // BusinessTemporaryAuditExternal

type Resolver struct {
    Matrix map[ActorClass]map[BehaviorKind]map[TargetDomain]Decision
}

func (r *Resolver) Decide(class ActorClass, behavior BehaviorKind, target TargetDomain) Decision
```

v0.1 只实现 mage/warrior × 本 spec 定义的 6 行 × 4 列。后续阶段管道扩展（design / execute / qa 三期）时，新职业直接加一行即可。

## 7. 降级路径与回滚开关

v0.1.21 的"一刀切关 Bash"就是天然的安全降级。保留两个 env：

| Env | 默认 | 作用 |
|---|---|---|
| `GLOOP_MAGE_SANDBOX_ENABLED=1` | 1 | 0 = 退回 v0.1.21 |
| `GLOOP_MAGE_SANDBOX_MODE` | auto | `native` / `fuse` / `bwrap` / `degraded`，强制指定调试 |

**回滚保障**：把整个 spec 的改动收敛在 `internal/sandbox/` + `permissions/` + cli_base.go 1 个分支判断中，删除这两个包 + 恢复分支就是回滚。

## 8. 迁移 / 兼容

- **warrior 零影响**：不读 mage 分支
- **ACP executor**：ACP 原生 Bash 调用由 micro_loop 拦截判断当前 phase；mage 阶段就走 Sandbox.Exec，warrior 走原生
- **存量 quest**：在中途升级到带沙箱版本的 quest（mage 阶段已开始）——本轮 turn 开始前建沙箱即可，不依赖之前的 turn 状态
- **doctor e2e**：新增 `doctor mage-sandbox` 子命令，探测当前环境沙箱支持度 + 输出 mode + 跑一条写文件验证（`touch /workspace/x` + teardown 后确认 x 不存在）

## 9. 工作量分解（WBS）

| ID | 任务 | 估算（人日） | 风险 |
|---|---|---|---|
| 1 | `internal/sandbox/`：overlay + fuse-overlayfs + bwrap 三级 builder + Exec/Teardown | 2 | 环境兼容，需要 CI + BOE + 本机三种环境测 |
| 2 | `internal/permissions/resolver.go`：行为矩阵 + Decide，单元测试覆盖 6×4 = 24 格 | 0.5 | 低 |
| 3 | cli_base.go：effectiveAllowedTools 接入 resolver、`Sandbox.Mode` 透传、AppendSandboxBashToolIfEnabled | 0.5 | 低 |
| 4 | micro_loop.go：mage Bash 调用分支走 Sandbox.Exec，结果回灌 + 审计 | 1 | 中（拦截点改动需要重跑 orchestrator 所有实验测试） |
| 5 | 事件 / 僵局检测：`mage_sandbox_*` 事件 + deadlock_detector 两个特征 | 1 | 中（要更新僵局检测 baseline） |
| 6 | doctor mage-sandbox + 现有 doctor e2e 增加 mage 沙箱场景 | 0.5 | 低 |
| 7 | Frontend：Mage Sandbox Evidence 面板 | 2 | 中 |
| 8 | 文档 / skill 更新：`gloop-quest-review` skill 追加"可在沙箱中自由跑验证"说明 | 0.5 | 低 |
| 9 | AB 实验：`mage-review-sandbox-2026Q3`，对比开启前后 reviewer_score_density / true_positive_rate | 1 | 中（等待数据） |
| | 合计 | 9 | |

## 10. 验证与验收标准

- **UT 覆盖率**：sandbox 包 ≥ 80%，permissions resolver 100%（24 格全测）
- **e2e 新增场景**：
  - mage 在沙箱中跑 `go build ./...`，stdout 可见"编译错误"，用户工作区不出现任何 `.o` / `.a` / 二进制
  - mage 在沙箱中 `echo hacked > ./main.go`，teardown 后 diff 确认 main.go 内容未变，`mage_sandbox_writes_captured` 事件 fired
  - 禁用 user ns 的环境（`/proc/sys/kernel/unprivileged_userns_clone=0`）下，自动降级到 degraded，和 v0.1.21 行为 byte 一致
  - warrior 的 Bash 行为**零变化**（用 AB 对比脚本跑存量 doctor）
- **性能**：sandbox build/teardown 单次 < 50 ms；`go test ./...` 同命令在沙箱内外耗时差 ≤ 5%
- **业务指标**：上线 4 周后，mage true_positive_rate（经返工被剑士确认的问题 / mage 所有 request_changes 中的问题）相较基线提升 ≥ 15%；放水率（score ≥ 阈值且后续被用户驳回的 pass / 所有 pass）下降 ≥ 10%

## 11. 风险与对策

| 风险 | 影响 | 对策 |
|---|---|---|
| overlayfs / user ns 在 BOE/PPE 某些内核版本上不可用 | 自动降 degraded，大量 quest 变静态扫描 | 新增 `doctor mage-sandbox` 作部署预检 + `mage_sandbox_degraded` 事件可观测，SOP 文档写清楚内核参数开关 |
| mage 通过 `--bind` / `/proc` 逃逸沙箱（极罕见但严重） | 写业务态 | 双兜底：lower 层 `chattr +i`（若可用）+ teardown 后强制 `git diff HEAD --exit-code`，非零则 quest 立刻升级为用户终审 + 告警 |
| 沙箱内命令卡死 mage turn | 超时 | Sandbox.Exec 沿用 turnCtx 超时 + 已有的 CLIExecutorBase Interrupt 路径 |
| micro_loop 拦截点分支改多了影响 warrior | 回归 | CI 上所有 orchestrator 实验（ReworkContextExperiment 三条）必须 100% 通过 + 加一条 warrior Bash 行为对照测试 |

## 12. 里程碑

| Milestone | 交付 |
|---|---|
| M1：沙箱原型 + UT | internal/sandbox/ 可在本机三种 mode 下跑 doctor |
| M2：权限矩阵 + micro_loop 接入 | 全量 UT / 现有 orchestrator 测试通过 |
| M3：事件 + 僵局检测 + 前端面板 | 可观测端到端 |
| M4：AB 实验 → 全量 | 业务指标达成验收标准 |
