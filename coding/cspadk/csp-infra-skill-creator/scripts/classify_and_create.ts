#!/usr/bin/env node

import * as fs from 'fs';
import * as path from 'path';

const PATTERN_KEYS = ['tool-wrapper', 'generator', 'reviewer', 'inversion', 'pipeline'] as const;

export type SkillPattern = (typeof PATTERN_KEYS)[number];

export interface ClassificationResult {
  patterns: SkillPattern[];
  reasons: string[];
}

interface CliArgs {
  skillName: string;
  inputPath?: string;
}

function parseArgs(argv: string[]): CliArgs {
  const args: CliArgs = { skillName: '' };

  for (let i = 2; i < argv.length; i += 1) {
    const arg = argv[i];
    const next = argv[i + 1];

    if ((arg === '--name' || arg === '-n') && next) {
      args.skillName = next;
      i += 1;
    } else if ((arg === '--input' || arg === '-i') && next) {
      args.inputPath = next;
      i += 1;
    }
  }

  if (!args.skillName) {
    throw new Error('Missing required argument: --name <skill-name>');
  }

  return args;
}

function readRequirementText(inputPath?: string): string {
  if (inputPath) {
    const abs = path.resolve(process.cwd(), inputPath);
    return fs.readFileSync(abs, 'utf8');
  }

  if (process.stdin.isTTY) {
    // No stdin piped in.
    return '';
  }

  return fs.readFileSync(0, 'utf8');
}

function classifyPatterns(text: string): ClassificationResult {
  const lower = text.toLowerCase();
  const patterns = new Set<SkillPattern>();
  const reasons: string[] = [];

  const add = (pattern: SkillPattern, reason: string) => {
    patterns.add(pattern);
    reasons.push(reason);
  };

  if (/checklist|severity|review|审查|评审|打分/.test(lower)) {
    add('reviewer', '检测到与审查、评分或检查清单相关的需求，适合 Reviewer 模式。');
  }

  if (/template|模板|生成文档|report|报告|规范输出/.test(lower)) {
    add('generator', '需求强调结构化输出或基于模板生成内容，适合 Generator 模式。');
  }

  if (/库|sdk|api|client|wrapper|fastapi|http|rest/.test(lower)) {
    add('tool-wrapper', '涉及特定库或 API 约定，适合用 Tool Wrapper 将约定封装为可复用上下文。');
  }

  if (/多步|阶段|step|phase|pipeline|流程|工作流/.test(lower)) {
    add('pipeline', '需求包含清晰的多步骤流程或阶段，适合 Pipeline 模式。');
  }

  if (/问卷|问题列表|interview|访谈|收集需求|gather requirements/.test(lower)) {
    add('inversion', '需要先通过问答收集信息再行动，适合 Inversion 模式。');
  }

  // 如果没有命中任何模式，默认使用 Generator + Tool Wrapper 的组合，便于后续补充。
  if (patterns.size === 0) {
    add('generator', '未识别到明确模式，默认使用 Generator 以保证输出结构化。');
    add('tool-wrapper', '未识别到明确模式，默认允许按需加载参考资料作为 Tool Wrapper。');
  }

  return {
    patterns: Array.from(patterns),
    reasons,
  };
}

function ensureDir(dir: string): void {
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }
}

function buildSkillRoot(skillName: string): string {
  const safeName = skillName.trim().replace(/\s+/g, '-').toLowerCase();
  const repoRoot = process.cwd();
  return path.join(repoRoot, 'skills', safeName);
}

function createSkillSkeleton(skillName: string, classification: ClassificationResult, requirementSnippet: string): string {
  const skillRoot = buildSkillRoot(skillName);
  const scriptsDir = path.join(skillRoot, 'scripts');
  const referencesDir = path.join(skillRoot, 'references');

  ensureDir(skillRoot);
  ensureDir(scriptsDir);
  ensureDir(referencesDir);

  const patternsYaml = classification.patterns.map((p) => `  - ${p}`).join('\n');
  const snippet = requirementSnippet.trim().slice(0, 400);

  const skillMd = `---
name: ${skillName}
description: 待补充：根据国际电商客服场景，描述该 Skill 的职责与目标。
metadata:
  patterns:
${patternsYaml}
---

# 技能概览

本 Skill 由 \`create-csp-i18n-skill\` 自动生成，用于支撑国际电商客服团队的 AI 场景。请在落地前补充具体业务背景与边界。

## 触发场景

- 当用户提出与本 Skill 服务领域相关的需求时，由上游路由 Agent 或 SubAgent 触发。

## 输入

- 自然语言需求：来自用户或上游 Agent 的结构化/非结构化描述。
- 可选上下文：相关代码片段、配置、文档链接等。

## 输出

- 结构化的执行计划或最终产物（文档/代码/配置等），具体形式由业务 Owner 补充。

## 执行步骤（模式组合）

1. **解析输入**：
   - 识别需求类型、目标对象以及约束条件。
2. **应用设计模式**：
   - 根据下述模式组合组织执行流程：
${classification.patterns.map((p) => `     - ${p}`).join('\n')}
3. **生成中间产物**：
   - 按照 SOP 多阶段结构拆分执行，必要时在 Stage 间引导 SubAgent 协作。
4. **返回结果**：
   - 将关键决策、产物路径与下一步建议以结构化方式返回给调用方。

## 与 create-csp-i18n-skill / aime-skill-creator 的关系

- 本 Skill 目录由 \`skills/create-csp-i18n-skill/scripts/classify_and_create.ts\` 自动生成。
- 上述脚本遵循 \`aime-skill-creator\` 的 SOP 约定，将技能拆分为 \`SKILL.md + scripts/ + references/\` 的多阶段结构。
- 后续如需接入完整的 Stage1/Stage2/Stage3 流程，可在 \`references/\` 中补充设计文档，并在 \`scripts/\` 中固化关键命令。

## 初始需求摘录

> ${snippet || '（生成时未提供需求摘要，请在此处补充一段 2–3 句的需求描述。）'}
`;

  fs.writeFileSync(path.join(skillRoot, 'SKILL.md'), skillMd, 'utf8');

  const referencesReadme = `# references

该目录用于存放本 Skill 依赖的详细资料，例如：

- 业务背景说明
- API 或协议规范
- 检查清单 / 模板 等

请将体量较大的内容放在此处，并在 SKILL.md 中以相对路径引用。
`;

  if (!fs.existsSync(path.join(referencesDir, 'README.md'))) {
    fs.writeFileSync(path.join(referencesDir, 'README.md'), referencesReadme, 'utf8');
  }

  return skillRoot;
}

function main() {
  try {
    const args = parseArgs(process.argv);
    const requirementText = readRequirementText(args.inputPath);
    const classification = classifyPatterns(requirementText);
    const skillRoot = createSkillSkeleton(args.skillName, classification, requirementText);

    // 输出简要信息，方便上游 Agent 或脚本解析。
    const summary = {
      skillName: args.skillName,
      patterns: classification.patterns,
      skillRoot,
    };

    // eslint-disable-next-line no-console
    console.log(JSON.stringify(summary, null, 2));
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    // eslint-disable-next-line no-console
    console.error(`[create-csp-i18n-skill] ${message}`);
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}
