---
name: gloop-quest-fanout
version: 1.0.0
description: "委托扇出：剑士将大任务拆分为多个独立并行委托。典型触发：任务包含多个无强依赖的子目标、需要对同一代码库的不同模块分别执行独立变更、调研类任务需要并行探索多个方向。不适用：子任务间有串行依赖需要前序结果、单一任务不可拆分、拆分后子 quest 无法独立验收。"
metadata:
  class: warrior
  category: orchestration
  kind: orchestration
  related_skills:
    - name: gloop-quest-execution
      type: related
      description: 扇出是剑士执行的扩展模式，每个子 quest 独立走 execution 流程
    - name: gloop-user-context
      type: related
      description: 子 quest 之间通过共享 context dim 传递中间结论
  requires:
    bins: ["gloop"]
    cliHelp: "gloop --help"
---

# gloop-quest-fanout

剑士侧「委托扇出」方法论：判断何时拆分、如何拆分、扇出后如何协调。

## Trigger Examples

| 类型 | 场景 | 说明 |
|---|---|---|
| 适用 | 任务包含 3+ 个独立子目标 | 子目标间无数据流依赖 |
| 适用 | 同一仓库不同模块的独立变更 | 每个模块可独立测试验收 |
| 适用 | 调研任务需要并行探索多个技术方案 | 各方案探索互不阻塞 |
| 适用 | 批量处理（N 个文件/服务/配置的同类操作） | 每项可独立完成 |
| 不适用 | 子任务必须串行执行（B 依赖 A 的产出） | 用笔记记录中间结果，顺序执行 |
| 不适用 | 单个原子操作不可拆分 | 直接执行即可 |
| 不适用 | 拆出的子 quest 无法独立定义验收标准 | 说明拆分粒度不对 |

## CLI Contract

通过 `gloop quest spawn` 命令扇出独立委托，用于将大任务拆分为多个并行子 quest。

### 命令一览

| 命令 | 用途 |
|---|---|
| `gloop quest spawn` | 扇出一个新的独立委托（子 quest） |

### 详细说明

#### gloop quest spawn

创建一个全新的独立委托，有自己的剑士和法师，立即启动执行，并通过 GroupID 与当前 quest 关联。

**用法：**
```bash
gloop quest spawn \
  --query "新委托的任务描述" \
  --group-id "分组标签" \
  --leaf-id "api-contract" \
  --ownership-scopes "internal/server,web/src/api" \
  --merge-strategy single_leaf \
  --merge-owner-leaf-id "api-contract"
# 或简写
gloop quest spawn "新委托的任务描述"
```

**参数：**

| 参数 | 必填 | 类型 | 说明 |
|---|---|---|---|
| `--query` / 位置参数 | 是 | string | 新委托的任务描述 |
| `--group-id` | 否 | string | 分组标签；不填则继承当前 quest 的 group_id 或自动生成 |
| `--leaf-id` | 否 | string | 当前 leaf 的稳定标识；声明 ownership 或 merge 时必填 |
| `--ownership-scopes` | 否 | csv | 当前 leaf 负责的文件/模块/问题域，逗号分隔 |
| `--merge-strategy` | 否 | enum | `single_leaf` / `sequential` / `no_merge` |
| `--merge-owner-leaf-id` | 否 | string | `single_leaf` / `sequential` 的合并责任 leaf |
| `--work-dir` | 否 | string | 工作目录；默认与当前 quest 一致 |
| `--json` | 否 | flag | 以 JSON 格式输出结果 |

**返回：**

加 `--json` 时以 JSON 格式返回新委托信息；默认输出人类可读文本。

**注意：**
- 调用后平台会创建一个全新的独立委托（有自己的剑士和法师），并立即启动执行。
- 新委托通过 GroupID 与当前 quest 关联；如果声明了 ownership，平台会记录 leaf contract。
- 同一 group 内两个 leaf 声明了相同 `ownership_scopes` 时，必须声明共同的 `merge_owner_leaf_id`，否则平台拒绝创建，避免并行写同一 ownership scope 后无人负责合并。

### 通用约定

- CLI 是本地 agent 调用 gloop 平台能力的主路径
- 平台注入 `GLOOP_*` 环境变量，通常不需要手填 qid/sid

## Coordination Patterns

扇出后的 quest 之间**没有直接通信通道**。协调方式：

1. **共享 context dims** — 任何 quest 的剑士都可以写 context dim，其他 quest 可读。
   用于传递中间结论、共享发现。
2. **GroupID + leaf ownership** — 同一次扇出的 quest 共享 GroupID；每个 leaf 用 `leaf_id` 和 `ownership_scopes` 声明负责范围。
3. **独立验收** — 每个子 quest 由各自的法师独立评审，无跨 quest 联合判定。
4. **显式合并责任** — 有重叠 ownership 时，必须用 `merge_strategy` + `merge_owner_leaf_id` 点名合并责任 leaf；不要让平台猜主从。

## Discipline

- 扇出不是逃避复杂度的手段。每个子 quest 的 query 必须足够清晰，让一个全新的剑士无额外上下文也能执行。
- 不要把状态管理推给子 quest 之间的隐式依赖。如果子任务需要读前序结果，要么不扇出，要么先把前序结果写入 context dim 再扇出。
- 不要并行写同一 ownership scope，除非你已经声明共同的 merge owner。没有合并责任声明就拆开写，是平台会拒绝的错误用法。
- 扇出数量应合理（通常 2-5 个）。超过 5 个考虑是否应该分批或用 automation。
- 扇出后当前 quest 可以继续执行自己的工作，也可以通过 `gloop phase done` 结束（总结扇出了哪些子任务）。
- GroupID 仍不做生命周期耦合。子 quest 失败不会自动影响父 quest 或其他兄弟 quest；ownership/merge contract 只约束扇出创建和后续展示/审计。
