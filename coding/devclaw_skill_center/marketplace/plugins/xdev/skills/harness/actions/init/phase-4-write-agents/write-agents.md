# Phase 4: 文档生成（AGENTS.md 体系）

> **本 phase 在 `/harness init` 流程中的位置**：第 4 步，可委托 subagent 执行。依赖 Phase 3 完成（需要 `docs/` 目录结构存在）。

**目标**：分析目标仓库，生成分层的 AGENTS.md 文档体系。

## 操作

调用 prompt 文件 `<skill_dir>/actions/init/phase-4-write-agents/prompts/write-agents-guide.md`，按其中的流程为当前仓库生成分层的 AGENTS.md 文档体系。

### 委托方式（推荐：subagent）

```
Agent(
  subagent_type="general-purpose",
  prompt="
请阅读 <skill_dir>/actions/init/phase-4-write-agents/prompts/write-agents-guide.md 并按照其中的流程，
为当前仓库生成分层的 AGENTS.md 文档体系。

注意：
- 文档层数由项目复杂度决定，不强制固定层数
- 核心约束必须从代码分析中提取，不能是空泛的
- 如果已有 AGENTS.md，使用更新模式（增量更新，保留自定义内容）
- 根 AGENTS.md 的『文档体系』或『知识导航』表中必须体现 docs/ 的分层结构，
  包括 reference/、guidance/、plans/ 等子目录的定位和导航
- 生成完成后进行自检
"
)
```

如果环境不支持 Task tool，主 Agent 直接读取 prompt 文件并按其中流程顺序执行。

## 汇报

完成后向用户汇报：
- 生成了几层文档（根级 / 模块级 / docs 级）
- 每层覆盖了什么模块/职责
- 提取的核心约束清单
- 自检结果
