---
description: 扩展 OpenSpec proposal 工作流，将 proposal 从单 repo 提升到 workspace-scope，可在父目录一次性生成涉及多个 repo 的 proposal。
argument-hint: feature_desc.md  branch_name
---
<!-- OPENSPEC:START -->

# Prompt: Workspace 级 OpenSpec Proposal

你是一名后端架构师/工程师，精通 OpenSpec 与微服务/多仓库协作。你的任务是在 **workspace 父目录**（例如 `ad/`）一次性为多个 repo 生成Proposal，并输出可机器解析的执行报告。

## 输入

- `feature_desc.md`：用户给定的功能描述文档。
- `branch_name`：用户给定的本次仓库开发分支名

### 推荐（可选）结构化字段
如果 `feature_desc.md` 中包含如下 YAML/JSON 片段（任意一种），优先使用它来确定 repo 与依赖关系：

```yaml
workspace:
  change_id: feature-refund-api-20260306
repos:
  - name: star_task
    path: star_task
    depends_on: []
  - name: star_aggregator
    path: star_aggregator
    depends_on: [star_task]
```
/
若未提供，则使用启发式解析（见下文）。

## 目标

- 自动识别涉及的一个或多个 repo。
- 在每个 repo 下对应的branch_name分支生成 Proposal，且所有 repo 使用同一个 workspace-level `change-id`。
- 在执行前做状态校验；按依赖排序；同一依赖层级并行执行；产出执行日志与最终 JSON 报告。

## 约束

- 仅对“状态校验通过”的 repo 进行写入。
- 不要修改业务代码；本命令只负责生成 Proposal
- 若遇到不确定（无法解析 repo 或依赖），先收敛出候选列表并提示用户确认（优先用 `AskUserQuestion`）。

## Steps（请用 TODO 逐步推进）

### 1）解析 workspace 与 repo 列表

1. 读取 `feature_desc.md`（见“输入”规则），解析：
   - `repos`：repo 名称（优先 `name`/`path`，否则用目录名）。
   - `branch_name`：repo 开发分支名。
   - `depends_on`：repo 依赖（可选）。
2. 若没有结构化字段，启发式解析：
   - 从文档中提取形如 `` `star_xxx` `` 的 repo 名；或从 `In Scope/范围/仓库` 等段落中提取 `- xxx：...` 的前缀。
   - 将候选 repo 与当前目录下的一级子目录做交叉校验：目录存在且包含 `go.mod` 时才算有效候选。
3. 若候选结果为空/歧义（例如命中文档中的简称但目录不存在），用 `AskUserQuestion` 让用户确认：
   - 最终 repo 列表
   -（可选）依赖关系

### 2）生成 workspace-level change-id

生成唯一且可复用的 `change-id`：

- 若结构化字段中提供 `workspace.change_id`（或文本中出现 `change-id:`），使用之。
- 否则：
  - 从文档第一行标题（`# ...`）提取语义并做 `kebab-case`（仅保留 `a-z0-9-`）。
  - 拼接日期：`YYYYMMDD`。
  - 示例：`feature-refund-api-20260306`。

### 3）逐 repo 状态检查（记录可读日志）

对每个 repo（用绝对路径/工作区相对路径均可）：

1. 目录存在性：目录不存在 -> `skipped`。
2. OpenSpec 初始化：`openspec/project.md` 不存在 -> `skipped`。
3. 工作区干净：
   - 若是 git 仓库（`git rev-parse --is-inside-work-tree` 为 true），且 `git status --porcelain` 非空 -> `skipped`。
   - 非 git 仓库则仅记录 warning，不强制跳过。

### 4）依赖排序与并行分层

- 根据 `depends_on` 进行拓扑排序。
- 若依赖缺失或存在环：记录 warning，并退化为文档中出现顺序/目录顺序。
- 以“层级”方式执行：同一层级（其依赖均已完成）可并行。

### 5）Proposal 生成（对通过校验的 repo）

对每个可执行 repo：
1. 先切换到master分支：`git checkout master`
2. 拉取最新代码：`git pull`
3. 切换到对应的branch_name分支：`git checkout branch_name`
4. 执行 `/openspec:proposal feature_desc.md` 生成提案
5. 可选）验证：若该 repo 已存在 `openspec` 且 `openspec validate <change-id> --strict` 可运行，则执行一次并记录结果；验证失败不阻断整体流程。

### 6）输出 JSON 报告（最后一步，必须输出）

严格输出一段 JSON（不要包裹额外解释文字），包含：

```json
{
  "change-id": "feature-refund-api-20260306",
  "success_repos": ["star_task", "star_aggregator"],
  "skipped_repos": ["star_gomarketing"],
  "proposal_paths": {
    "star_task": "star_task/openspec/changes/feature-refund-api-20260306/proposal.md",
    "star_aggregator": "star_aggregator/openspec/changes/feature-refund-api-20260306/proposal.md"
  },
  "logs": [
    "star_gomarketing: 未初始化 openspec（缺少 openspec/project.md），跳过"
  ]
}
```

$ARGUMENTS

<!-- OPENSPEC:END -->
