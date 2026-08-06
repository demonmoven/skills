---
name: csp-infra-skill-creator
description: 根据用户需求，结合 5 种标准设计模式，为国际电商客服（i18n）场景快速生成一个结构化的新 Aime 技能骨架。
version: 1.0.3
metadata:
  pattern:
    - generator
    - inversion
  domain: csp-infra
  i18n_level: 0
  prompt_version: "1.0.0"
  agent_support:
    - claude-code
    - cursor
  language:
    - zh-CN
    - en
---

# 技能：创建客服 i18n 场景新技能

本技能旨在固化“从用户需求到标准技能骨架”的创建流程，确保新技能遵循 `aime-skill-creator` 的 SOP 约束和业界公认的设计模式。

## 1. 技能目标与触发场景

- **目标**：接收一段关于国际电商客服场景的 AI 技能需求，自动将其分类到一种或多种设计模式，并生成一个符合 `aime-skill-creator` 规范的、包含 `SKILL.md` 和 `scripts/` 等基础目录的技能骨架。
- **触发场景**：
  - 当用户明确表示“创建一个新技能”、“我想开发一个xx技能”时。
  - 当需要将一个模糊的想法或一个具体的工作流转化为一个可复用的 Aime 技能时。
  - 作为 `aime-skill-creator` SOP 的上游，用于快速启动技能的“创建”阶段。

## 2. 输入理解与分类

本技能的核心是首先理解用户输入的需求，并将其映射到 `references/patterns.md` 中定义的五种标准设计模式之一或其组合。

1.  **读取用户需求**：
    - 从标准输入（stdin）或指定的输入文件（`--input` 参数）读取完整的需求描述文本。
2.  **输入分类 (Pattern Classification)**：
    - 调用内部脚本 `scripts/classify_and_create.ts` 的分类逻辑。
    - 该脚本会根据需求文本中的关键词（如 "模板"、"审查"、"工作流"、"API"、"提问" 等）和语义，将需求归类到以下一种或多种模式：
      - **Tool Wrapper**：当需求侧重于封装特定库/API 的使用方法时。
      - **Generator**：当需求侧重于生成结构化、模板化的输出时。
      - **Reviewer**：当需求侧重于根据清单或规则对某些内容进行审查/打分时。
      - **Inversion**：当需求需要通过一系列问答来主动收集信息时。
      - **Pipeline**：当需求包含一个明确、严格的多步骤工作流时。
    - 如果无法明确匹配，默认会采用 `Generator` + `Tool Wrapper` 的组合，以保证输出的结构化和可扩展性。

## 3. 执行流程

本技能的执行流程由 `scripts/classify_and_create.ts` 固化，并优先由 SubAgent 执行，以确保流程的稳定性和原子性。

1.  **选择模式 -> 调用脚本**：
    - Aime Agent 分析用户需求后，调用 `bash` 工具执行以下命令：
      ```bash
      # 需求文本通过管道传入
      cat /path/to/user_requirement.txt | npx tsx skills/create-csp-i18n-skill/scripts/classify_and_create.ts --name "your-new-skill-name"

      # 或直接从文件读取
      npx tsx skills/create-csp-i18n-skill/scripts/classify_and_create.ts --name "your-new-skill-name" --input /path/to/user_requirement.txt
      ```
    - **注意**：脚本使用 `tsx` 运行，不依赖平台私有命令，保证了可移植性。

2.  **生成目标 Skill**：
    - `classify_and_create.ts` 脚本执行以下操作：
      - **创建目录**：在顶层 `skills/` 目录下创建一个新的技能目录，如 `skills/your-new-skill-name/`。
      - **创建子目录**：在其中创建 `scripts/` 和 `references/` 子目录。
      - **生成 SKILL.md**：在新技能目录中生成 `SKILL.md` 文件。该文件已包含：
        - 基于分类结果的 `metadata.patterns`。
        - 明确的技能目标、输入、输出和执行步骤说明。
        - 对 `aime-skill-creator` SOP 的引用，指导后续开发。
        - 对所应用的具体设计模式的说明。
        - 一小段原始需求文本的摘录，作为上下文参考。
      - **生成辅助文件**：在 `references/` 目录下创建 `README.md`，说明其用途。

3.  **返回结果**：
    - 脚本执行成功后，会向标准输出（stdout）打印一个 JSON 对象，包含新技能的名称和根目录路径。
    - Aime Agent 捕获此输出，并向用户报告新技能已创建成功，同时提供新技能的路径和其他简要信息。

## 4. 与 `aime-skill-creator` 的关系

本技能是 `aime-skill-creator` 的一个具体实现和“快捷方式”，专注于其 SOP 中的“Stage 2: 创建 Skill”。

- **遵循 SOP**：本技能生成的目录结构（`SKILL.md` + `scripts/` + `references/`）严格遵循 `aime-skill-creator` 定义的多阶段结构约定，确保了后续可以无缝接入 `Stage 3: Review Skill` 等后续流程。
- **固化流程**：它将“需求分类 -> 模式选择 -> 创建骨架”这一最佳实践固化为一段可执行、可重复调用的 Node.js 脚本，降低了手动创建技能的心智负担。
- **SubAgent 优先**：`SKILL.md` 中明确指出，本技能生成的脚本优先由 SubAgent 执行，这与 `aime-skill-creator` 推荐的”调度者-执行者”协作模式保持一致。

## Output Language

Detect the user's input language and respond in the same language.
- If the user writes in Chinese, produce all output in Chinese.
- If the user writes in English, produce all output in English.
- If unclear, default to English (en).
