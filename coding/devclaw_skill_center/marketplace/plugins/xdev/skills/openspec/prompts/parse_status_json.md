# Change 状态扫描指令

> 通过扫描 `docs/xdev/openspec/changes/<name>/` 目录,判断 spec-driven schema 下各 artifact 的 ready/blocked/done 状态。
>
> **历史注**:本文件早期名为 `parse_status_json.md`,负责解析 `openspec status --change <name> --json` 的输出。当前版本已重构为纯目录扫描,不再依赖外部 CLI。文件名暂保留以兼容现有引用。

## 输入参数

- `PRIMARY_REPO`:主仓库根目录
- `CHANGE_NAME`:要查询的 change 名称

## 输出

- `SCHEMA_NAME`:使用的 schema(默认 `spec-driven`)
- `ARTIFACTS`:完整的 artifact 列表,每项含 `id` / `status` / `dependencies` / `outputPath`
- `READY_ARTIFACTS`:当前所有 `status: ready` 的 artifact ID 列表
- `BLOCKED_ARTIFACTS`:当前所有 `status: blocked` 的 artifact ID 列表
- `DONE_ARTIFACTS`:当前所有 `status: done` 的 artifact ID 列表
- `APPLY_REQUIRES`:进入 apply 阶段需要 done 的 artifact ID 列表(spec-driven 默认 = `["tasks"]`)
- `IS_APPLY_READY`:布尔值,apply 是否就绪
- `IS_ALL_DONE`:布尔值,所有 artifact 是否全部 done

---

## Schema(spec-driven 默认)

```yaml
artifacts:
  - id: proposal
    outputPath: proposal.md
    requires: []
  - id: specs
    outputPath: specs/**/*.md         # 至少一个文件存在即视为 done
    requires: [proposal]
  - id: design
    outputPath: design.md
    requires: [proposal]              # design 是 optional,not-existing 时视为 "done"(skipped)
  - id: tasks
    outputPath: tasks.md
    requires: [specs, design]

apply:
  requires: [tasks]
  tracks: tasks.md                    # apply 阶段通过勾选 tasks.md 跟踪进度
```

---

## 步骤 1:确认 change 目录存在

```bash
ls -d "{PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}/" 2>/dev/null
```

不存在 → 报错:"Change `{CHANGE_NAME}` 不存在于 `{PRIMARY_REPO}/docs/xdev/openspec/changes/` 下"

## 步骤 2:扫描每个 artifact 的状态

对 schema 中的每个 artifact,检查 `outputPath` 文件是否存在:

```bash
CHANGE_DIR="{PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}"

# proposal
[ -f "$CHANGE_DIR/proposal.md" ] && proposal_done=true || proposal_done=false

# specs (至少一个 specs/**/*.md 存在)
specs_files=$(ls -1 "$CHANGE_DIR"/specs/*/spec.md 2>/dev/null | wc -l)
[ "$specs_files" -gt 0 ] && specs_done=true || specs_done=false

# design (optional)
[ -f "$CHANGE_DIR/design.md" ] && design_done=true || design_done=false

# tasks
[ -f "$CHANGE_DIR/tasks.md" ] && tasks_done=true || tasks_done=false
```

## 步骤 3:派生 status

对每个 artifact:

- **done** = `outputPath` 文件存在
- **ready** = 不 done 且所有 `requires` 都 done(可以现在生成)
- **blocked** = 不 done 且至少一个 `requires` 还没 done(无法生成)

特殊情况:
- `design` 是 **optional**,即使没生成也视为"可跳过"。在 spec-driven schema 中:
  - 如果用户明确要 design → 必须 done 才能进 tasks
  - 如果用户判断不需要 design(简单 change)→ 可以视为"已 skip",此时 tasks 可以 ready
  - 由 Claude Agent 判断:如果 proposal.md 中标注"不需要 design",则 `design = "skipped"` 并按 done 计入 dependencies

## 步骤 4:派生汇总变量

```text
SCHEMA_NAME       = "spec-driven"
ARTIFACTS         = [{id, status, dependencies, outputPath}, ...]
READY_ARTIFACTS   = [a.id for a in ARTIFACTS if a.status == "ready"]
BLOCKED_ARTIFACTS = [a.id for a in ARTIFACTS if a.status == "blocked"]
DONE_ARTIFACTS    = [a.id for a in ARTIFACTS if a.status == "done"]

APPLY_REQUIRES    = ["tasks"]
IS_APPLY_READY    = "tasks" in DONE_ARTIFACTS
IS_ALL_DONE       = all(a.status == "done" for a in ARTIFACTS)
```

## 步骤 5:返回

返回上述 8 个变量给调用方。

---

## 与早期 CLI 版本的差异

- **早期版本**:调用 `openspec status --change <name> --json` 拿到结构化 JSON
- **当前版本**:Claude Agent 直接 `ls` 文件并判断状态。优势:
  - 零外部依赖
  - 状态判断逻辑透明可追溯
  - 支持任意 schema(在对话里告诉 Claude 新的 artifact 编排,无需改 CLI)
