# Action: init —— 全流程初始化

为任意代码仓库一键落地 Harness Engineering 最佳实践。语言无关、框架无关。

## 核心原则：适配优先，绝不照搬

**Harness 化不是把一套标准模板贴到所有仓库上，而是理解仓库的实际情况后，选择性地、适配性地引入实践。**

本 action 流转过程中所有 `assets/` 下的文件都是 **参考基线和示例**，不是最终产物。Agent 必须根据仓库实际情况调整后再安装。

执行本 action 时，必须遵守以下原则：

1. **先理解，再行动**：Phase 1 不仅检测技术栈，还要深入理解仓库的工作方式——现有的 CI/CD 流程、代码风格惯例、团队协作模式、已有的质量门禁。这些理解决定后续每个 Phase 的具体操作。

2. **适配而非复制**：
   - lint 规则：参阅 prompts/lint-strategy.md 了解每条规则的目的，然后为目标语言选择对应的实现工具
   - 文件长度上限：基于仓库实际文件长度分布决定（如 P95 向上取整），而非强制 600
   - hook 选择：如果团队不使用 AI 编码工具，`ai-guard` 系列不应安装
   - Skill 选择：只安装对当前项目有实际价值的 Skill

3. **尊重已有实践**：如果仓库已有 ESLint/Ruff/Clippy 配置、已有 CI 门禁、已有代码规范文档，harness 应增强而非替换。新引入的实践必须与已有实践协调一致，不能产生冲突或重复。

4. **向用户解释 "为什么"**：每个 Phase 的汇报不仅说 "做了什么"，更要说 "为什么这样做"——"因为你的仓库有 X 特征，所以选择了 Y 方案而非 Z"。

### 反模式警示

以下行为是 **错误的**：

- 对 100 行代码的小工具安装 5 个 Skill 和 18 个 hook
- 对已有完善 ESLint 配置的项目强行安装 Biome
- 对不使用 AI 编码工具的团队安装 ai-guard hooks
- 对函数普遍 200+ 行的数据管道项目强制 `funlen=120`
- 忽略仓库已有的 CI 门禁而重复设置相同检查
- 对 Python 项目安装 golangci-lint 配置
- 对 Rust 项目安装 go-fmt hook

---

## 交互模型

本 action 采用 **checklist 式交互**：

1. 每个 Phase 开始前，向用户简要说明即将做什么，以及为什么这样做
2. Phase 完成后，汇报做了什么、跳过了什么、做了哪些适配调整
3. 展示下一个 Phase 的计划，征求用户确认
4. 用户可以随时要求跳过某个 Phase 或调整方案

全程维护 `harness-init.progress` 文件追踪进度（规范见 `<skill_dir>/references/progress-spec.md`）。

---

## 执行模型：主 Agent 协调 + 可选 Subagent 委托

### 为什么需要委托

`/harness init` 的 6 个 Phase 全部在一个 Agent session 中顺序执行时，对于大仓库（如 monorepo）容易导致上下文窗口耗尽——Phase 1 的深度分析会读取大量文件和 git 历史，这些中间上下文会一直占用到 Phase 6 结束。

### 调度策略

**Phase 1 必须在主 Agent 执行**——需要与用户一问一答交互（结构化访谈），且其产出是所有后续 Phase 的输入。

**Phase 1 完成后**，主 Agent 将分析结果写入 `harness-init-analysis.md`（结构化的适配方案文档），然后检测当前环境是否支持 subagent（是否有 Task tool 可用）：

- **支持 subagent**：Phase 2/3/5 可以并行委托给 subagent 执行，主 Agent 负责协调和验证
- **不支持 subagent**：主 Agent 继续顺序执行所有 Phase（退化为传统模式）

```
Phase 1 (主 agent 执行，包含用户交互)
  │
  │  产出 → harness-init-analysis.md（适配方案文档）
  │
  ├──→ Phase 2 (subagent 或主 agent)  ──────────────────────┐
  ├──→ Phase 3 (subagent 或主 agent) ──→ Phase 4 (subagent) │
  └──→ Phase 5 (subagent 或主 agent)  ──────────────────────┤
                                                              ↓
                                                    Phase 6 (主 agent 验证)
```

### Subagent 委托规则

**1. 上下文传递**：每个 subagent 接收 `harness-init-analysis.md` 的相关章节（不是全部），加上该 Phase 需要的 assets/templates 路径。不传递 Phase 1 的原始分析过程。

**2. 产物验证（强制）**：主 Agent 必须验证每个 subagent 的产出：
- 检查文件是否创建成功
- 检查内容是否符合 Phase 要求（非空、不含未填充的占位符模板）
- 检查是否遗漏了 Phase 1 适配方案中要求的内容
- 如果验证失败，主 Agent 自行修复或重新委托

**3. 可并行的组合**：
- Phase 2 + Phase 3 + Phase 5（都只依赖 Phase 1，互不依赖）
- Phase 4（依赖 Phase 3 完成）

**4. 不可委托的 Phase**：
- Phase 1（需要用户交互 + 全仓库分析）
- Phase 6（需要验证所有前序产出，适合主 Agent 统一检查）

### 退化模式

如果当前环境不支持 subagent（无 Task tool），或者目标仓库很小（<10 个主要文件），主 Agent 直接顺序执行所有 Phase。

---

## 前置条件

- 目标仓库已 `git init`
- Hook 管理工具满足以下**任一**条件即可：
  - 已安装 `pre-commit`（Python）或 `prek`。未安装则提示用户执行 `pip install pre-commit`
  - 仓库已使用 **husky**（或 lefthook、simple-git-hooks 等 Node.js 生态的 hook 管理工具）。此时 harness 应**利用已有机制**，将 hook 检查逻辑添加到现有的 git hooks 脚本中

---

## 流程编排

按以下顺序读取并执行各 phase 主 md。每个 phase 完成后向用户汇报，征求确认进入下一阶段。

| Step | Phase | 入口文件 |
|------|-------|---------|
| 1 | 仓库深度理解（Phase 1） | `<skill_dir>/actions/init/phase-1-analyze/analyze.md` |
| 2 | Pre-commit Hooks 安装（Phase 2） | `<skill_dir>/actions/init/phase-2-install-hooks/install-hooks.md` |
| 3 | 文档骨架与知识库（Phase 3） | `<skill_dir>/actions/init/phase-3-scaffold-docs/scaffold-docs.md` |
| 4 | AGENTS.md 生成（Phase 4） | `<skill_dir>/actions/init/phase-4-write-agents/write-agents.md` |
| 5 | ARCHITECTURE.md 生成（Phase 5） | `<skill_dir>/actions/init/phase-5-write-architecture/write-architecture.md` |
| 6 | 验证（Phase 6） | `<skill_dir>/actions/init/phase-6-validate/validate.md` |

---

## 幂等性

每个 Phase 都设计为幂等操作：

- 文件复制：检查目标是否已存在，相同则跳过
- 内容追加：检查是否已包含相同内容，避免重复
- 目录创建：使用 `mkdir -p`
- 符号链接：检查是否已正确指向目标

`/harness init` 可以安全地重复运行（例如中断后恢复），不会破坏已有内容。

## 已有内容的处理策略

当目标仓库已有部分 harness 组件时：

- **已有 `.pre-commit-config.yaml`**：展示 diff，让用户决定合并还是覆盖
- **已有 husky/lefthook 等 hook 管理工具**：使用路径 B，将检查逻辑集成到已有工具中
- **已有 `AGENTS.md`**：使用更新模式（增量更新，保留自定义内容）
- **已有 `ARCHITECTURE.md`**：提示用户是否要重新生成
- **已有 `Makefile`**：展示需要追加的 target，让用户决定
- **已有 lint 配置**：不覆盖，只建议补充缺失的 AI 友好规则

## 注意事项

- 本 action 不修改任何业务代码，只添加工程基础设施和文档
- 所有生成的文档内容基于对仓库的实际分析，不是空泛的模板填充
- Phase 4 和 Phase 5 可能耗时较长（需要深度分析仓库）
- 如果用户只想安装部分组件，可以在 Phase 间确认时选择跳过
- **`assets/` 中的所有文件都是参考示例**，禁止不加思考地原样复制
