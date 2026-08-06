# 5 种 Agent Skill 设计模式

> **来源**: [Google Cloud Tech on X (formerly Twitter)](https://x.com/GoogleCloudTech/status/2033953579824758855)
>
> 本文档内容基于 GoogleCloudTech 发布的公开信息，旨在为 `create-csp-i18n-skill` 提供设计模式参考。

---

通过研究 Anthropic、Vercel 及 Google 内部的技能构建实践，可以总结出五种反复出现的、可用于构建可靠 Agent 的设计模式。

## 1. 工具包装器 (Tool Wrapper)

**核心思想**：为 Agent 提供关于特定库的按需上下文，使其成为该库的即时专家。

- **解决问题**：避免将具体的 API 约定、内部编码规范或框架最佳实践硬编码到系统提示词（System Prompt）中。
- **实现方式**：将这些上下文打包成一个独立的 Skill。仅当 Agent 实际需要使用该技术时，它才会加载此 Skill，从而节省了不必要的上下文开销。
- **示例**：创建一个 `fastapi-expert` 技能。
  - `SKILL.md` 监听用户输入中的 `FastAPI` 等关键词。
  - 当触发时，动态从 `references/conventions.md` 加载 FastAPI 的编码约定。
  - 指导 Agent 在审查或编写代码时，必须遵循这些约定。

```markdown
# skills/api-expert/SKILL.md
---
name: api-expert
description: FastAPI development best practices and conventions. Use when building, reviewing, or debugging FastAPI applications, REST APIs, or Pydantic models.
metadata:
  pattern: tool-wrapper
  domain: fastapi
---

You are an expert in FastAPI development. Apply these conventions to the user's code or question.

## Core Conventions

Load 'references/conventions.md' for the complete list of FastAPI best practices.

## When Reviewing Code
1. Load the conventions reference
2. Check the user's code against each convention
3. For each violation, cite the specific rule and suggest the fix

## When Writing Code
1. Load the conventions reference
2. Follow every convention exactly
3. Add type annotations to all function signatures
4. Use Annotated style for dependency injection
```

## 2. 生成器 (Generator)

**核心思想**：通过协调“模板填充”过程来确保结构化、一致性的输出。

- **解决问题**：Agent 在每次运行时可能会生成不同结构的文档，导致结果不可预测。
- **实现方式**：
  - `assets/` 目录：存放输出模板（如 `report-template.md`）。
  - `references/` 目录：存放风格指南（如 `style-guide.md`）。
  - `SKILL.md` 充当项目经理的角色：指导 Agent 加载模板、阅读风格指南、向用户询问缺失的变量，然后填充模板并输出。
- **应用场景**：生成可预测的 API 文档、标准化的 Commit Message、项目架构脚手架等。

```markdown
# skills/report-generator/SKILL.md
---
name: report-generator
description: Generates structured technical reports in Markdown. Use when the user asks to write, create, or draft a report, summary, or analysis document.
metadata:
  pattern: generator
  output-format: markdown
---

You are a technical report generator. Follow these steps exactly:

Step 1: Load 'references/style-guide.md' for tone and formatting rules.
Step 2: Load 'assets/report-template.md' for the required output structure.
Step 3: Ask the user for any missing information...
Step 4: Fill the template following the style guide rules.
Step 5: Return the completed report as a single Markdown document.
```

## 3. 审查器 (Reviewer)

**核心思想**：将“审查什么”与“如何审查”分离。

- **解决问题**：避免在系统提示词中冗长地描述每一个代码坏味道或安全漏洞。
- **实现方式**：
  - `references/review-checklist.md`：存放一个模块化的审查清单（Rubric），其中包含具体的规则和标准。
  - `SKILL.md`：定义审查流程。当用户提交代码时，Agent 加载此清单，并有条不紊地对代码进行评分，按严重性（如 `error`, `warning`, `info`）对发现的问题进行分组。
- **优势**：高度可扩展。通过替换不同的检查清单（例如，从 Python 风格指南切换到 OWASP 安全清单），可以用完全相同的技能基础设施实现不同领域的专业审计。
- **应用场景**：自动化代码审查（PR Review）、在人工介入前发现漏洞。

```markdown
# skills/code-reviewer/SKILL.md
---
name: code-reviewer
description: Reviews Python code for quality, style, and common bugs...
metadata:
  pattern: reviewer
  severity-levels: error,warning,info
---

You are a Python code reviewer. Follow this review protocol exactly:

Step 1: Load 'references/review-checklist.md' for the complete review criteria.
Step 2: Read the user's code carefully...
Step 3: Apply each rule from the checklist to the code...
Step 4: Produce a structured review with these sections: Summary, Findings, Score, Recommendations.
```

## 4. 控制反转 (Inversion)

**核心思想**：让 Agent 主导对话，通过结构化提问来收集信息，而不是被动地执行用户指令。

- **解决问题**：Agent 倾向于立即猜测并生成答案，但往往缺乏必要的上下文，导致结果不准确。
- **实现方式**：
  - 在 `SKILL.md` 中设置明确的、不可协商的门控指令（Gating Instructions），例如：“在所有阶段完成之前，不要开始构建”。
  - 强制 Agent 按顺序提出结构化问题，并等待用户回答，直到收集到所有必要信息后才进入下一阶段（例如，合成最终输出）。
- **应用场景**：项目启动规划、系统设计、需求访谈等需要先收集大量信息的场景。

```markdown
# skills/project-planner/SKILL.md
---
name: project-planner
description: Plans a new software project by gathering requirements through structured questions...
metadata:
  pattern: inversion
  interaction: multi-turn
---

You are conducting a structured requirements interview. DO NOT start building or designing until all phases are complete.

## Phase 1 — Problem Discovery (ask one question at a time, wait for each answer)
- Q1: "What problem does this project solve for its users?"
- ...

## Phase 2 — Technical Constraints (only after Phase 1 is fully answered)
- Q4: "What deployment environment will you use?"
- ...

## Phase 3 — Synthesis (only after all questions are answered)
1. Load 'assets/plan-template.md' for the output format
2. Fill in every section of the template using the gathered requirements
...
```

## 5. 流水线 (Pipeline)

**核心思想**：为复杂任务强制执行一个严格的、带有关卡（Checkpoints）的顺序工作流。

- **解决问题**：在复杂的多步骤任务中，Agent 可能会跳过某些步骤或忽略指令，导致交付不可靠的、未经验证的结果。
- **实现方式**：
  - `SKILL.md` 本身就是工作流的定义。
  - 通过实现明确的“钻石门”条件（Diamond Gate Conditions），例如“在进入最终组装阶段之前，必须获得用户对生成文档字符串的批准”，来确保 Agent 无法绕过关键验证步骤。
- **特点**：
  - **强制顺序**：Agent 必须按顺序执行每个步骤。
  - **设立关卡**：在关键节点（如从一个阶段到下一个阶段）需要显式的用户确认或验证。
  - **按需加载资源**：在工作流的特定步骤中才加载所需的 `references/` 或 `assets/` 文件，保持上下文窗口的清洁和高效。

```markdown
# skills/doc-pipeline/SKILL.md
---
name: doc-pipeline
description: Generates API documentation from Python source code through a multi-step pipeline...
metadata:
  pattern: pipeline
  steps: "4"
---

You are running a documentation generation pipeline. Execute each step in order. Do NOT skip steps or proceed if a step fails.

## Step 1 — Parse & Inventory
...Ask for confirmation.

## Step 2 — Generate Docstrings
...Present each for user approval. Do NOT proceed to Step 3 until the user confirms.

## Step 3 — Assemble Documentation
...Compile all parts into a single document.

## Step 4 — Quality Check
...Review against 'references/quality-checklist.md'. Fix issues before presenting.
```

## 模式组合

这些设计模式并非相互排斥，而是可以组合使用。例如：
- 一个 **Pipeline** 技能可以在最后包含一个 **Reviewer** 步骤，以审查其自身的工作。
- 一个 **Generator** 技能可以在开始时利用 **Inversion** 模式来收集所有必要的变量，然后再填充模板。

通过将复杂指令分解，并应用正确的结构化模式，可以构建出更可靠、更智能的 Agent。