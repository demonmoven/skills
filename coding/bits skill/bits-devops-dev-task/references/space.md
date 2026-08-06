# 空间管理能力

> **入口判定**：用户说"最近访问的空间"、"搜索空间"、"切换空间"、"查空间"、"空间下的需求/缺陷"、"搜索 Meego 工作项"等。

用于在开发任务操作前查找目标空间 ID，或搜索空间绑定的 Meego 工作项（为创建开发任务提供 meego 参数）。

## 目录

- [列出最近访问的空间](#列出最近访问的空间)
- [搜索空间](#搜索空间)
- [搜索空间绑定的 Meego 工作项](#搜索空间绑定的-meego-工作项)
  - [Step 1：查询空间详情，提取 Meego 空间 projectKey](#step-1查询空间详情提取-meego-空间-projectkey)
  - [Step 2：搜索 Meego 工作项](#step-2搜索-meego-工作项)
  
---

## 列出最近访问的空间

当用户说"最近访问的空间"、"我的空间列表"、"切换空间"等时使用。

```bash
bitscli devops space recently-accessed
```

返回 JSON，`spaces` 为空间数组。

**展示规则**：遍历 `spaces[]`，每条展示如下格式：

```
- 空间名：<name>
  类型：<type 的中文含义>
  空间 ID：<id>
  标识：<identification>
```

**`type` 枚举对照**：

| 值 | 向用户展示 |
|----|-----------|
| 1 | 服务端/前端 |
| 其它 | 直接展示数字 |

---

## 搜索空间

当用户说"搜索空间 xxx"、"找一下叫 xxx 的空间"等时使用。

```bash
bitscli devops space search --keyword <用户提供的关键词>
```

按关键词对空间名称进行模糊匹配，返回最多 20 条结果。

**展示规则**：与上文「列出最近访问的空间」相同，遍历 `spaces[]`，每条展示空间名 / 类型 / 空间 ID / 标识。若 `total` 大于结果数量，末尾提示"共匹配到 <total> 条，当前展示前 <N> 条"。

---

## 搜索空间绑定的 Meego 工作项

当用户说"搜索空间下的需求"、"空间绑定的缺陷"、"空间里的工作项"等时使用。

分两步：先查空间详情获取关联 Meego 空间的 `projectKey`，再用 `projectKey` 搜索工作项。

### Step 1：查询空间详情，提取 Meego 空间 projectKey

```bash
bitscli devops space detail --id <space_id>
```

返回 JSON，关注 `relatedStorySpaces[]` 数组。`space detail` 已自动为每条关联记录补充了 `meegoSpace` 字段（含 `id` / `name` / `simpleName`）。

**提取规则**：

- 遍历 `relatedStorySpaces[]`，取每条的 `meegoSpace.id` 作为 Meego 空间的 `projectKey`。
- 若 `meegoSpace` 为空或不存在，跳过。
- 若存在多个关联 Meego 空间，向用户列出所有可用项，让用户选择一个 `projectKey`。
- 若只有一个，直接使用。

**向用户展示 Meego 空间选择列表**：

```
该空间关联了以下 Meego 空间：
1. <meegoSpace.name>（projectKey: <meegoSpace.id>）
2. <meegoSpace.name>（projectKey: <meegoSpace.id>）
```

### Step 2：搜索 Meego 工作项

```bash
bitscli devops feature work-item search \
  --project-key <projectKey> \
  --query <搜索关键词> \
  --mine <true|false>
```

参数说明：

| 参数 | 含义 |
|------|------|
| `--project-key` | Step 1 中提取的 Meego 空间 `projectKey` |
| `--query` | 用户提供的搜索关键词 |
| `--mine` | `true` = 仅搜索我的工作项；`false` = 搜索所有工作项 |

返回 JSON：`total` 为匹配总数，`data.workItems` 为工作项数组。

**展示规则**：遍历 `data.workItems[]`，每条展示如下格式：

```
- [<category 中文>] <name>
  ID：<id>
  状态：<isCompleted 的中文含义>
  链接：<link>
```

**`category` 对照**：

| 值 | 向用户展示 |
|----|-----------|
| `story` | 需求 |
| `issue` | 缺陷 |
| `sub_task` | 子任务 |
| 其它 | 直接展示原值 |

**`isCompleted` 对照**：

| 值 | 向用户展示 |
|----|-----------|
| `true` | 已完成 |
| `false` | 进行中 |

若 `total` 大于当前返回数量，末尾提示"共匹配到 <total> 条，当前展示前 <N> 条"。

---