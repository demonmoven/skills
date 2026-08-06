# Phase 5: 架构文档（ARCHITECTURE.md）

> **本 phase 在 `/harness init` 流程中的位置**：第 5 步，可委托 subagent 执行。依赖 Phase 3 完成（需要 `docs/` 目录结构存在）。

**目标**：为目标仓库生成 ARCHITECTURE.md，遵循 matklad 方法论。

## 操作

调用 prompt 文件 `<skill_dir>/actions/init/phase-5-write-architecture/prompts/write-architecture-guide.md`，按其中的流程为当前仓库生成 ARCHITECTURE.md。

### 委托方式（推荐：subagent）

```
Agent(
  subagent_type="general-purpose",
  prompt="
请阅读 <skill_dir>/actions/init/phase-5-write-architecture/prompts/write-architecture-guide.md 并按照其中的流程，
为当前仓库生成 ARCHITECTURE.md。

注意：
- 遵循 matklad 方法论：只写不太会变的东西，保持简短，命名但不链接
- 每个模块至少一个 Architecture Invariant
- 总行数控制在 150-350 行
- 不变量提取标准参见 <skill_dir>/actions/init/phase-3-scaffold-docs/prompts/invariants-extraction.md
"
)
```

如果环境不支持 Task tool，主 Agent 直接读取 prompt 文件并按其中流程顺序执行。

## 汇报

完成后向用户汇报：
- ARCHITECTURE.md 的主要章节
- 覆盖的模块清单
- 总行数（应在 150-350 之间）
- 每个模块声明的 invariants 数量
