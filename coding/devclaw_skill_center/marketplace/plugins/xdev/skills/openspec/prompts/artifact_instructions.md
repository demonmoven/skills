# Artifact 生成指令

> 把单个 OpenSpec artifact(`proposal` / `specs` / `design` / `tasks`)的「模板路径 + 输出路径 + 依赖项 + 生成 instruction」集中在一处,被 `actions/propose.md` 在循环生成 artifact 时调用。

## 输入参数

- `ARTIFACT_ID`:要生成的 artifact 类型,枚举值 `proposal` / `specs` / `design` / `tasks`
- `CHANGE_DIR`:目标 change 目录绝对路径(`{PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}`)
- `PRIMARY_REPO`:主仓库根目录
- `PROJECT_CONTEXT`:由调用方(propose Step 5)收集的 agentic project context(README/AGENTS/CLAUDE.md/package.json 摘要 + 用户对话补充),作为生成时的隐式背景

## 输出

- 在 `outputPath` 处写入 artifact 文件
- 输出 `Created {ARTIFACT_ID} at {outputPath}`

---

## 分支:根据 ARTIFACT_ID 选择执行

### A. ARTIFACT_ID = "proposal"

- **模板**:Read `<skill_dir>/templates/proposal.md`
- **输出路径**:`{CHANGE_DIR}/proposal.md`
- **dependencies**:无(proposal 是入口)
- **instruction**(嵌入,来自 spec-driven schema):

> Create the proposal document that establishes WHY this change is needed.
>
> Sections:
> - **Why**: 1-2 sentences on the problem or opportunity. What problem does this solve? Why now?
> - **What Changes**: Bullet list of changes. Be specific about new capabilities, modifications, or removals. Mark breaking changes with **BREAKING**.
> - **Capabilities**: Identify which specs will be created or modified:
>   - **New Capabilities**: List capabilities being introduced. Each becomes a new `specs/<name>/spec.md`. Use kebab-case names (e.g., `user-auth`, `data-export`).
>   - **Modified Capabilities**: List existing capabilities whose REQUIREMENTS are changing. Only include if spec-level behavior changes (not just implementation details). Each needs a delta spec file. Check `{PRIMARY_REPO}/docs/xdev/openspec/specs/` for existing spec names. Leave empty if no requirement changes.
> - **Impact**: Affected code, APIs, dependencies, or systems.
>
> The Capabilities section is critical. It creates the contract between proposal and specs phases. Research existing specs before filling this in. Each capability listed here will need a corresponding spec file.
>
> Keep it concise (1-2 pages). Focus on the "why" not the "how" — implementation details belong in design.md.

### B. ARTIFACT_ID = "specs"

- **模板**:Read `<skill_dir>/templates/spec.md`
- **输出路径**:对每个 capability(从 proposal 的 Capabilities section 提取),创建 `{CHANGE_DIR}/specs/<capability>/spec.md`
- **dependencies**:
  - Read `{CHANGE_DIR}/proposal.md`(必读)
  - 对每个 Modified Capability,Read `{PRIMARY_REPO}/docs/xdev/openspec/specs/<capability>/spec.md`(必读,用于复制 ENTIRE 原始 requirement 块)
- **instruction**:

> Create specification files that define WHAT the system should do.
>
> Create one spec file per capability listed in the proposal's Capabilities section.
> - New capabilities: use the exact kebab-case name from the proposal (`specs/<capability>/spec.md`).
> - Modified capabilities: use the existing spec folder name from `{PRIMARY_REPO}/docs/xdev/openspec/specs/<capability>/` when creating the delta spec at `specs/<capability>/spec.md`.
>
> Delta operations (use ## headers):
> - **ADDED Requirements**: New capabilities
> - **MODIFIED Requirements**: Changed behavior - MUST include full updated content
> - **REMOVED Requirements**: Deprecated features - MUST include **Reason** and **Migration**
> - **RENAMED Requirements**: Name changes only - use FROM:/TO: format
>
> Format requirements:
> - Each requirement: `### Requirement: <name>` followed by description
> - Use SHALL/MUST for normative requirements (avoid should/may)
> - Each scenario: `#### Scenario: <name>` with WHEN/THEN format
> - **CRITICAL**: Scenarios MUST use exactly 4 hashtags (`####`). Using 3 hashtags or bullets will fail silently.
> - Every requirement MUST have at least one scenario.
>
> MODIFIED requirements workflow:
> 1. Locate the existing requirement in `{PRIMARY_REPO}/docs/xdev/openspec/specs/<capability>/spec.md`
> 2. Copy the ENTIRE requirement block (from `### Requirement:` through all scenarios)
> 3. Paste under `## MODIFIED Requirements` and edit to reflect new behavior
> 4. Ensure header text matches exactly (whitespace-insensitive)
>
> Common pitfall: Using MODIFIED with partial content loses detail at archive time. If adding new concerns without changing existing behavior, use ADDED instead.
>
> Specs should be testable - each scenario is a potential test case.

### C. ARTIFACT_ID = "design"

- **模板**:Read `<skill_dir>/templates/design.md`
- **输出路径**:`{CHANGE_DIR}/design.md`
- **dependencies**:Read `{CHANGE_DIR}/proposal.md`
- **instruction**:

> Create the design document that explains HOW to implement the change.
>
> When to include design.md (create only if any apply):
> - Cross-cutting change (multiple services/modules) or new architectural pattern
> - New external dependency or significant data model changes
> - Security, performance, or migration complexity
> - Ambiguity that benefits from technical decisions before coding
>
> If none apply, **skip design.md**(把 design 视为 done,直接进入 tasks)。在 proposal.md 中标注"设计简单,不需要 design.md"以保留判断痕迹。
>
> Sections:
> - **Context**: Background, current state, constraints, stakeholders
> - **Goals / Non-Goals**: What this design achieves and explicitly excludes
> - **Decisions**: Key technical choices with rationale (why X over Y?). Include alternatives considered for each decision.
> - **Risks / Trade-offs**: Known limitations, things that could go wrong. Format: [Risk] → Mitigation
> - **Migration Plan**: Steps to deploy, rollback strategy (if applicable)
> - **Open Questions**: Outstanding decisions or unknowns to resolve
>
> Focus on architecture and approach, not line-by-line implementation. Reference the proposal for motivation and specs for requirements.

### D. ARTIFACT_ID = "tasks"

- **模板**:Read `<skill_dir>/templates/tasks.md`
- **输出路径**:`{CHANGE_DIR}/tasks.md`
- **dependencies**:
  - Read `{CHANGE_DIR}/proposal.md`
  - Read `{CHANGE_DIR}/specs/**/*.md`
  - Read `{CHANGE_DIR}/design.md`(如有)
- **instruction**:

> Create the task list that breaks down the implementation work.
>
> **IMPORTANT**: Follow the template format exactly. The apply phase parses checkbox format to track progress. Tasks not using `- [ ]` won't be tracked.
>
> Guidelines:
> - Group related tasks under `## N. <Group Name>` headings
> - Each task MUST be a checkbox: `- [ ] X.Y Task description`
> - Tasks should be small enough to complete in one session
> - Order tasks by dependency (what must be done first?)
>
> Reference specs for what needs to be built, design for how to build it. Each task should be verifiable - you know when it's done.

---

## 通用生成步骤(适用于上述任一 ARTIFACT_ID)

1. Read 上面分支指定的**模板文件**(`<skill_dir>/templates/<artifact>.md`)作为骨架
2. Read 所有 **dependencies** 文件作为生成上下文
3. 把 `PROJECT_CONTEXT`(propose Step 5 收集的隐式背景)作为生成时的额外约束/参考
4. 按 instruction 的要求填充内容,替换模板中的 `<!-- ... -->` 占位注释为真实内容
5. **不要**把 `<context>` / `<rules>` / `<project_context>` 等 wrapper 文本写入 artifact 文件,只把它们当作生成时的约束
6. Write 到 `outputPath`
7. 验证文件存在:`ls -l {outputPath}` → 输出 `Created {ARTIFACT_ID} at {outputPath}`

如果遇到关键歧义(比如 capabilities 不清楚),用 **AskUserQuestion 工具**反问,但默认偏向"做出合理判断继续"。

---

## Guardrails

- 一次只生成一个 artifact;调用方负责循环
- 不复制 wrapper 块到 artifact 文件
- 生成前必须 Read 所有 dependencies
- 生成后立即 verify 文件存在
- 如果用户判断 design 不需要,在 proposal.md 中明确标注,以便后续 status 扫描时把 design 视为 skipped
