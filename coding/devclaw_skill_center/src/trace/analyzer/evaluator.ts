import { execFile as execFileCb } from 'node:child_process';
import { promisify } from 'node:util';
import { writeFileSync, mkdirSync, unlinkSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { randomUUID } from 'node:crypto';
import type { EvalResult, HarnessMetrics, HarnessAnalysis, UserCorrection, Inefficiency, SubagentStats, NormalizedMessage } from './types.js';
import { logger } from '../utils/logger.js';

const execFile = promisify(execFileCb);

// ─── CLI Detection ─────────────────────────────────────────────────

export type CliType = 'claude-code' | 'opencode';

export interface DetectedCli {
  type: CliType;
  path: string;
}

/**
 * Map from logical CLI type to the actual binary name on PATH.
 * claude-code's binary is installed as `claude`, not `claude-code`.
 */
const CLI_BINARY_NAME: Record<CliType, string> = {
  'claude-code': 'claude',
  'opencode': 'opencode',
};

/**
 * Detect if a Coding CLI is available on the system.
 *
 * @param preferred - Which CLI to try first (default: 'claude-code')
 * @returns Detected CLI info, or null if neither is available
 */
export async function detectCli(
  preferred: CliType = 'claude-code',
): Promise<DetectedCli | null> {
  const candidates: CliType[] =
    preferred === 'claude-code' ? ['claude-code', 'opencode'] : ['opencode', 'claude-code'];

  for (const cli of candidates) {
    const binaryName = CLI_BINARY_NAME[cli];
    const path = await whichCli(binaryName);
    if (path) {
      return { type: cli, path };
    }
  }
  return null;
}

async function whichCli(name: string): Promise<string | null> {
  try {
    const { stdout } = await execFile('which', [name], { timeout: 5000 });
    const path = stdout.trim();
    return path || null;
  } catch {
    return null;
  }
}

// ─── Evaluation Prompt ─────────────────────────────────────────────

/**
 * Evaluation prompt template.
 * __JSONL_PATH__ is replaced with the actual temp file path at call time.
 * The CLI reads the file itself — we don't embed the content in the prompt.
 */
export const EVAL_PROMPT = `你是一位 AI Coding Assistant 的轨迹评测专家。你的任务是阅读一份 AI 编程助手的完整会话记录（JSONL 格式），并从多个维度进行评测。

## 输入

请读取以下文件的完整内容：
__JSONL_PATH__

这是一个 JSONL 文件，每行是一条 JSON 记录，记录了用户与 AI 编程助手之间的完整交互过程，包括用户消息、助手回复、工具调用（如读写文件、执行命令）及工具返回结果。

## 评测维度

请从以下 8 个维度对这次交互进行评分（每个维度 0-1 分，精确到小数点后两位）：

1. **task_completion（任务完成度，权重 20%）**：助手是否完整地完成了用户的请求？是否遗漏了子任务？最终结果是否符合用户预期？

2. **user_alignment（用户对齐度，权重 20%）**：助手是否正确理解并执行了用户的真实意图？这是最能体现助手质量的维度，重点关注：
   - **显式纠正**：用户是否需要明确纠正助手（如"不要这样做"、"我是说X不是Y"、拒绝工具调用、重新表述同一请求）
   - **方向偏移**：助手是否偏离了用户的真实意图去做用户没要求的事情
   - **手动接管**：用户是否因为助手无法完成而自己动手操作（自己执行命令、自己编写代码）
   - **隐含期望**：助手的输出风格、详细程度、语言是否符合用户的隐含预期
   - 如果没有发现任何用户纠正/不满信号，给予较高分数；每发现一次严重纠正扣 0.15-0.25 分

3. **efficiency（效率，权重 15%）**：工具调用和工作流程是否高效？具体关注以下低效模式：
   - **不必要的工具调用**：读取了与任务无关的文件，搜索了已知的信息
   - **冗余循环**：反复读取同一文件，重试相同的失败命令却不改变策略
   - **过度探索**：花费大量轮次探索代码库而实际只需少量操作即可完成
   - **遗漏并行机会**：多个独立操作被不必要地串行执行
   - **过度工程化**：创建了超出需求的辅助函数、抽象层或额外文件
   - **过度确认**：在显而易见的下一步操作前反复请求用户确认
   - **子代理委派**：是否恰当地使用了子代理？委派的任务描述是否清晰？是否避免了不必要的子代理调用？子代理之间是否有效并行？

4. **error_handling（错误处理，权重 10%）**：遇到错误（工具调用失败、编译错误、测试失败等）时的恢复能力。按以下标准评判：
   - **优秀（0.8-1.0）**：准确诊断错误根因 → 针对性修复 → 一次成功
   - **一般（0.4-0.7）**：尝试 2-3 种方案 → 最终解决问题
   - **糟糕（0-0.3）**：盲目重试相同操作不改变策略、直接放弃、或要求用户自行解决
   - 如果会话中没有发生错误，基于助手的防御性编程意识给分（默认 0.7）

5. **decision_quality（决策质量，权重 10%）**：助手在关键决策点的判断是否合理？包括：
   - **工具选择**：是否选择了最合适的工具完成任务（如用 Grep 而非手动翻找文件）
   - **方案选择**：第一次选择的方案是否合理，还是频繁推倒重来
   - **范围判断**：工作范围是否恰当——有无过度扩展做了用户没要求的事，或遗漏了关键部分

6. **communication（沟通质量，权重 10%）**：助手的沟通是否清晰、简洁、有条理？是否准确理解了用户意图？是否在关键节点主动汇报进展而非冗余地汇报每一步？

7. **code_quality（代码质量，权重 10%）**：代码修改是否恰当？是否遵循了项目已有的编码风格和模式？修改范围是否精准，有无不必要的改动？

8. **harness_usage（工程实践，权重 5%）**：助手是否遵循了良好的工程实践？具体判断标准：
   - 修改代码前是否先阅读了相关文档和上下文
   - 面对复杂任务时是否有规划意识（而非直接上手试错）
   - 是否保持修改范围聚焦，避免在同一次操作中混入不相关改动
   - 修改后是否运行了测试和验证
   - 是否更新了受影响的文档
__HARNESS_DATA_SECTION__
__SUBAGENT_DATA_SECTION__
## 诊断数据

除了评分外，请提取以下结构化诊断数据：

### user_corrections：用户纠正事件
逐一检查会话中用户是否对助手进行了纠正、拒绝或方向调整。每个事件记录为：
- turn：大约在对话中的位置
- behavior：助手做了什么
- expectation：用户实际期望什么
- gap：归纳为通用模式（不要包含具体文件名或变量名）
- severity：critical（显式拒绝/否定）| high（方向调整/手动接管）| medium（隐含不满/忽略输出）

如果没有发现用户纠正事件，返回空数组。

### inefficiencies：低效模式
检测会话中出现的低效模式。每个模式记录为：
- pattern：模式名称（如"冗余文件读取"、"盲目重试"、"过度探索"）
- description：具体发生了什么
- wasted_turns：估计浪费的轮次数
- suggestion：应该怎么做更高效

如果没有发现低效模式，返回空数组。

## 输出格式

请严格按照以下 JSON 格式输出评测结果，不要输出任何其他内容：

\`\`\`json
{
  "score": <0-1 综合评分>,
  "task_completion": <0-1>,
  "user_alignment": <0-1>,
  "efficiency": <0-1>,
  "error_handling": <0-1>,
  "decision_quality": <0-1>,
  "communication": <0-1>,
  "code_quality": <0-1>,
  "harness_usage": <0-1>,
  "comment": "<200-500 字的中文评语，分析助手在本次交互中的表现亮点和改进空间>",
  "user_corrections": [
    {"turn": "<位置>", "behavior": "<助手行为>", "expectation": "<用户期望>", "gap": "<通用模式>", "severity": "critical|high|medium"}
  ],
  "inefficiencies": [
    {"pattern": "<模式名称>", "description": "<描述>", "wasted_turns": <数字>, "suggestion": "<建议>"}
  ],
  "harness_analysis": {
    "summary": "<500-1000 字的中文分析总结，说明 harness 产物的整体使用效果、哪些机制有效哪些无效以及改进方向>",
    "used_artifacts": [
      {"path": "<文件路径>", "read_count": <数字>, "mechanism": "<作用方式描述>", "evidence": "<具体证据>"}
    ],
    "unused_artifacts": [
      {"path": "<文件路径>", "intended_purpose": "<如果使用了这个文件，能对本次任务带来什么帮助>"}
    ],
    "mechanism_effectiveness": {
      "auto_injected": "<自动注入机制（如系统上下文）的效果评价>",
      "hard_constraints": "<硬约束机制（如 pre-commit hook）的效果评价>",
      "explicit_invocation": "<显式调用机制（如 skill 调用）的效果评价>",
      "passive_documentation": "<期望 AI 主动阅读的文档的效果评价>"
    }
  }
}
\`\`\`

综合评分 score 应该是 8 个维度的加权平均：任务完成度 20%、用户对齐度 20%、效率 15%、错误处理 10%、决策质量 10%、沟通质量 10%、代码质量 10%、工程实践 5%。

请确保：
- 所有评分在 0 到 1 之间
- 评语使用中文，200-500 字
- user_corrections 和 inefficiencies 必须基于轨迹中的具体证据，不能凭空编造；没有发现则返回空数组
- harness_analysis 的分析必须基于上方提供的 Harness 产物统计数据（如果有的话）和 JSONL 轨迹记录
- used_artifacts：列出实际起了作用的 harness 产物，附上作用方式和轨迹证据
- unused_artifacts：**不是列出所有未使用的文件**，而是列出你认为**对本次任务有帮助、AI 应该使用但实际没有使用的产物**。只列出真正有遗漏价值的，不要把所有未引用的文件都列上
- 输出是合法的 JSON，可以被 JSON.parse 解析
- 不要在 JSON 之外输出任何内容`;

/**
 * Build the harness data section to inject into the evaluation prompt.
 * If no harness metrics are provided, returns empty string.
 */
function buildHarnessDataSection(metrics: HarnessMetrics | undefined): string {
  if (!metrics || metrics.totalArtifacts === 0) return '';

  const lines: string[] = [];
  lines.push('');
  lines.push('## Harness 产物统计数据（程序化预分析结果）');
  lines.push('');
  if (metrics.repoScanned) {
    lines.push('以下数据基于**仓库完整扫描**（git ls-tree）+ 会话轨迹交叉比对。未使用产物是仓库中确实存在但整个会话中从未被 Read、提及或注入 prompt 的文件。');
  } else {
    lines.push('以下数据**仅基于会话轨迹记录**（未能获取仓库快照），可能遗漏仓库中存在但从未被引用的产物。未使用产物数量可能被低估。');
  }
  lines.push('');
  lines.push(`- 检测到的 Harness 产物总数：${metrics.totalArtifacts}`);
  lines.push(`- 被使用（Read 或提及）的产物：${metrics.usedArtifacts}`);
  lines.push(`- 未被使用的产物：${metrics.unusedArtifacts}`);
  lines.push(`- 是否使用 ExecPlan：${metrics.execPlanUsed ? '是' : '否'}`);
  lines.push(`- 是否在编辑代码前先读了文档：${metrics.docReadBeforeEdit ? '是' : '否'}`);

  if (metrics.skillInvocations.length > 0) {
    lines.push('');
    lines.push('Skill 调用统计：');
    for (const s of metrics.skillInvocations) {
      lines.push(`  - ${s.name}: ${s.count} 次`);
    }
  }

  lines.push('');
  lines.push('各产物详细引用情况：');
  lines.push('');
  lines.push('| 文件路径 | 分类 | Read 次数 | 提及次数 | Prompt 注入次数 | 首次读取轮次 |');
  lines.push('|---------|------|----------|---------|---------------|------------|');
  for (const a of metrics.artifacts) {
    lines.push(`| ${a.path} | ${a.category} | ${a.readCount} | ${a.mentionCount} | ${a.injectedCount} | ${a.firstReadTurn ?? '-'} |`);
  }
  lines.push('');

  return lines.join('\n');
}

/**
 * Build the subagent data section to inject into the evaluation prompt.
 * Includes subagent statistics and optionally full conversation records.
 */
function buildSubagentDataSection(
  subagentStats: SubagentStats | undefined,
  subagentMessages?: Map<string, NormalizedMessage[]>,
): string {
  if (!subagentStats || subagentStats.totalSubagents === 0) return '';

  const lines: string[] = [];
  lines.push('');
  lines.push('## Subagent 使用统计');
  lines.push('');
  lines.push(`本次会话共调度了 ${subagentStats.totalSubagents} 个子代理，以下是统计数据：`);
  lines.push('');
  lines.push(`- 子代理总 Token：${subagentStats.totalSubagentTokens.toLocaleString()}`);
  lines.push(`- 子代理总成本：$${subagentStats.totalSubagentCost.toFixed(4)}`);
  lines.push(`- 子代理工具错误率：${subagentStats.subagentErrorRate}%`);
  lines.push('');
  lines.push('各子代理概览：');
  lines.push('');
  lines.push('| 名称 | 类型 | 描述 | 消息数 | Token | 成本 |');
  lines.push('|------|------|------|--------|-------|------|');
  for (const sub of subagentStats.subagents) {
    lines.push(`| ${sub.slug} | ${sub.agentType} | ${sub.description.slice(0, 80)} | ${sub.messageCount} | ${sub.totalTokens.toLocaleString()} | $${sub.totalCost.toFixed(4)} |`);
  }

  if (Object.keys(subagentStats.subagentToolCounter).length > 0) {
    lines.push('');
    lines.push('子代理工具调用分布：');
    const sorted = Object.entries(subagentStats.subagentToolCounter).sort(([, a], [, b]) => b - a);
    for (const [tool, count] of sorted.slice(0, 15)) {
      lines.push(`  - ${tool}: ${count}`);
    }
  }

  // Include full subagent conversations if total size is manageable
  if (subagentMessages && subagentMessages.size > 0) {
    let totalChars = 0;
    for (const msgs of subagentMessages.values()) {
      for (const m of msgs) totalChars += m.content.length;
    }

    if (totalChars < 500_000) {
      lines.push('');
      lines.push('### 子代理完整对话记录');
      lines.push('');
      for (const [agentId, msgs] of subagentMessages) {
        const meta = subagentStats.subagents.find(s => s.agentId === agentId);
        lines.push(`#### ${meta?.slug ?? agentId} (${meta?.agentType ?? 'unknown'}): ${meta?.description ?? ''}`);
        lines.push('');
        for (const msg of msgs) {
          if (msg.role === 'system') continue;
          const prefix = msg.role === 'user' ? '[User]' : '[Assistant]';
          lines.push(`${prefix} ${msg.content.slice(0, 2000)}`);
          if (msg.toolCalls) {
            for (const tc of msg.toolCalls) {
              lines.push(`  → Tool: ${tc.name} | ${tc.input.slice(0, 200)}`);
            }
          }
        }
        lines.push('');
      }
    } else {
      lines.push('');
      lines.push('（子代理对话数据量过大，仅提供上方统计摘要）');
    }
  }

  lines.push('');
  return lines.join('\n');
}

// ─── Evaluation ────────────────────────────────────────────────────

// No timeout — LLM evaluation of large JSONL files can take a long time
// and we don't want to kill the process prematurely.

/**
 * Run LLM-as-Judge evaluation on a JSONL session.
 *
 * 1. Writes JSONL content to a temp file
 * 2. Builds prompt with the temp file path (content is NOT embedded)
 * 3. Calls the Coding CLI in non-interactive mode with permission bypass
 * 4. Parses the JSON result from CLI stdout
 * 5. Cleans up the temp file
 *
 * @param jsonlContent - Raw JSONL text content
 * @param cli - Detected CLI to use
 * @returns EvalResult on success
 * @throws Error on CLI failure, timeout, or parse error
 */
export async function evaluate(
  jsonlContent: string,
  cli: DetectedCli,
  harnessMetrics?: HarnessMetrics,
  subagentStats?: SubagentStats,
  subagentMessages?: Map<string, NormalizedMessage[]>,
): Promise<EvalResult> {
  // 1. Write JSONL to temp file
  const evalDir = join(tmpdir(), 'xtrace-eval');
  mkdirSync(evalDir, { recursive: true });
  const jsonlPath = join(evalDir, `session-${randomUUID()}.jsonl`);
  writeFileSync(jsonlPath, jsonlContent, 'utf-8');
  logger.info(`[evaluator] Wrote JSONL to ${jsonlPath} (${(jsonlContent.length / 1024).toFixed(1)} KB)`);

  // 2. Build prompt — inject harness data and subagent data if available
  const harnessSection = buildHarnessDataSection(harnessMetrics);
  const subagentSection = buildSubagentDataSection(subagentStats, subagentMessages);
  const prompt = EVAL_PROMPT
    .replace('__JSONL_PATH__', jsonlPath)
    .replace('__HARNESS_DATA_SECTION__', harnessSection)
    .replace('__SUBAGENT_DATA_SECTION__', subagentSection);

  // 3. Call CLI
  const args = buildCliArgs(cli.type, prompt);
  logger.info(`[evaluator] Calling ${cli.type} CLI (JSONL path: ${jsonlPath})`);

  let stdout: string;
  try {
    const result = await execFile(cli.path, args, {
      timeout: 0, // no timeout
      maxBuffer: 10 * 1024 * 1024, // 10 MB
      env: { ...process.env },
    });
    stdout = result.stdout;
  } catch (err: unknown) {
    // Some CLIs write to stdout even on non-zero exit
    const execErr = err as { stdout?: string; stderr?: string; message?: string };
    if (execErr.stdout && (execErr.stdout.includes('"score"') || execErr.stdout.includes('"result"'))) {
      stdout = execErr.stdout;
    } else {
      throw new Error(
        `CLI evaluation failed: ${execErr.message ?? String(err)}` +
        (execErr.stderr ? `\nstderr: ${execErr.stderr.slice(0, 500)}` : '')
      );
    }
  }

  // 4. Cleanup temp file (best-effort)
  try { unlinkSync(jsonlPath); } catch { /* ignore */ }

  // 5. Extract text from CLI output envelope if needed
  //    claude-code --output-format json wraps the response in {"type":"result","result":"..."}
  const evalText = extractCliOutput(stdout, cli.type);

  // 6. Parse result
  return parseEvalResult(evalText, cli.type);
}

/**
 * Extract the text content from CLI stdout.
 *
 * - claude-code with `--output-format json` returns a JSON envelope:
 *   `{"type":"result","result":"<the actual text>", ...}`
 *   We extract the `result` field.
 * - opencode and other CLIs return plain text directly.
 */
function extractCliOutput(stdout: string, cliType: string): string {
  if (cliType === 'claude-code') {
    try {
      const envelope = JSON.parse(stdout.trim()) as Record<string, unknown>;
      if (typeof envelope['result'] === 'string') {
        logger.info(`[evaluator] Extracted result from claude-code JSON envelope (${(envelope['result'] as string).length} chars)`);
        return envelope['result'] as string;
      }
    } catch {
      // Not a JSON envelope — fall through to return raw stdout
      // (e.g. if claude-code was called with --output-format text)
    }
  }
  return stdout;
}

function buildCliArgs(cliType: CliType, prompt: string): string[] {
  switch (cliType) {
    case 'claude-code':
      // --output-format json: returns a single JSON envelope with `result` field
      //   after the session fully completes. This avoids stdout truncation issues
      //   that occur with --output-format text when tool_use causes streaming output
      //   to be flushed incompletely before the process exits.
      // --permission-mode bypassPermissions: allow CLI to read the temp JSONL file
      //   outside the project directory without prompting for permission
      return ['-p', prompt, '--output-format', 'json', '--permission-mode', 'bypassPermissions'];
    case 'opencode':
      return ['-p', prompt];
  }
}

/**
 * Parse the JSON eval result from CLI stdout.
 * The output may contain extra text before/after the JSON block;
 * we try multiple strategies to extract the JSON object.
 */
export function parseEvalResult(stdout: string, cliType: string): EvalResult {
  let parsed: Record<string, unknown> | null = null;

  // Strategy 1: Try ```json ... ``` code block first (most reliable)
  const codeBlockMatch = stdout.match(/```json\s*\n?([\s\S]*?)\n?\s*```/);
  if (codeBlockMatch?.[1]) {
    try {
      parsed = JSON.parse(codeBlockMatch[1]) as Record<string, unknown>;
    } catch { /* try next strategy */ }
  }

  // Strategy 2: Find a JSON object using balanced brace matching
  if (!parsed) {
    const startIdx = stdout.indexOf('{');
    if (startIdx !== -1) {
      // Find the matching closing brace
      let depth = 0;
      let endIdx = -1;
      for (let i = startIdx; i < stdout.length; i++) {
        if (stdout[i] === '{') depth++;
        else if (stdout[i] === '}') {
          depth--;
          if (depth === 0) { endIdx = i; break; }
        }
      }
      if (endIdx !== -1) {
        try {
          parsed = JSON.parse(stdout.slice(startIdx, endIdx + 1)) as Record<string, unknown>;
        } catch { /* try next strategy */ }
      }
    }
  }

  // Strategy 3: Truncated JSON recovery — LLM output may be cut off mid-response
  // (e.g. token limit reached). Try to salvage by closing open strings and braces.
  if (!parsed) {
    parsed = tryRepairTruncatedJson(stdout);
  }

  if (!parsed) {
    throw new Error(
      `Failed to parse evaluation result: no JSON found in CLI output.\n` +
      `Output preview: ${stdout.slice(0, 500)}`
    );
  }

  // Validate and normalize fields
  const score = toNumber(parsed['score']);
  const taskCompletion = toNumber(parsed['task_completion']);
  const userAlignment = toNumber(parsed['user_alignment']);
  const efficiency = toNumber(parsed['efficiency']);
  const errorHandling = toNumber(parsed['error_handling']);
  const decisionQuality = toNumber(parsed['decision_quality']);
  const communication = toNumber(parsed['communication']);
  const codeQuality = toNumber(parsed['code_quality']);
  const harnessUsage = toNumber(parsed['harness_usage']);
  const comment = typeof parsed['comment'] === 'string' ? parsed['comment'] : '';

  if (score === null) {
    throw new Error(`Invalid evaluation result: missing or invalid "score" field. Parsed: ${JSON.stringify(parsed).slice(0, 300)}`);
  }

  const result: EvalResult = {
    score: clamp01(score),
    taskCompletion: clamp01(taskCompletion ?? score),
    userAlignment: clamp01(userAlignment ?? score),
    efficiency: clamp01(efficiency ?? score),
    errorHandling: clamp01(errorHandling ?? score),
    decisionQuality: clamp01(decisionQuality ?? score),
    communication: clamp01(communication ?? score),
    codeQuality: clamp01(codeQuality ?? score),
    harnessUsage: clamp01(harnessUsage ?? score),
    comment: comment || '无评语',
    evaluatedAt: new Date().toISOString(),
    evaluatedBy: cliType,
  };

  // Parse user corrections if present
  const correctionsRaw = parsed['user_corrections'];
  if (Array.isArray(correctionsRaw) && correctionsRaw.length > 0) {
    result.userCorrections = parseUserCorrections(correctionsRaw as Array<Record<string, unknown>>);
  }

  // Parse inefficiencies if present
  const inefficienciesRaw = parsed['inefficiencies'];
  if (Array.isArray(inefficienciesRaw) && inefficienciesRaw.length > 0) {
    result.inefficiencies = parseInefficiencies(inefficienciesRaw as Array<Record<string, unknown>>);
  }

  // Parse harness analysis if present
  const harnessRaw = parsed['harness_analysis'] as Record<string, unknown> | undefined;
  if (harnessRaw) {
    result.harnessAnalysis = parseHarnessAnalysis(harnessRaw);
  }

  return result;
}

function parseHarnessAnalysis(raw: Record<string, unknown>): HarnessAnalysis {
  const summary = typeof raw['summary'] === 'string' ? raw['summary'] : '';

  const usedRaw = Array.isArray(raw['used_artifacts']) ? raw['used_artifacts'] as Array<Record<string, unknown>> : [];
  const usedArtifacts = usedRaw.map(a => ({
    path: String(a['path'] ?? ''),
    readCount: typeof a['read_count'] === 'number' ? a['read_count'] : 0,
    mechanism: String(a['mechanism'] ?? ''),
    evidence: String(a['evidence'] ?? ''),
  }));

  const unusedRaw = Array.isArray(raw['unused_artifacts']) ? raw['unused_artifacts'] as Array<Record<string, unknown>> : [];
  const unusedArtifacts = unusedRaw.map(a => ({
    path: String(a['path'] ?? ''),
    intendedPurpose: String(a['intended_purpose'] ?? ''),
  }));

  const mechRaw = (raw['mechanism_effectiveness'] ?? {}) as Record<string, unknown>;
  const mechanismEffectiveness = {
    autoInjected: String(mechRaw['auto_injected'] ?? ''),
    hardConstraints: String(mechRaw['hard_constraints'] ?? ''),
    explicitInvocation: String(mechRaw['explicit_invocation'] ?? ''),
    passiveDocumentation: String(mechRaw['passive_documentation'] ?? ''),
  };

  return { summary, usedArtifacts, unusedArtifacts, mechanismEffectiveness };
}

function parseUserCorrections(raw: Array<Record<string, unknown>>): UserCorrection[] {
  return raw.map(item => ({
    turn: String(item['turn'] ?? ''),
    behavior: String(item['behavior'] ?? ''),
    expectation: String(item['expectation'] ?? ''),
    gap: String(item['gap'] ?? ''),
    severity: (['critical', 'high', 'medium'].includes(String(item['severity']))
      ? String(item['severity']) as UserCorrection['severity']
      : 'medium'),
  }));
}

function parseInefficiencies(raw: Array<Record<string, unknown>>): Inefficiency[] {
  return raw.map(item => ({
    pattern: String(item['pattern'] ?? ''),
    description: String(item['description'] ?? ''),
    wastedTurns: typeof item['wasted_turns'] === 'number' ? item['wasted_turns'] : 0,
    suggestion: String(item['suggestion'] ?? ''),
  }));
}

function toNumber(val: unknown): number | null {
  if (typeof val === 'number' && !isNaN(val)) return val;
  if (typeof val === 'string') {
    const n = parseFloat(val);
    if (!isNaN(n)) return n;
  }
  return null;
}

function clamp01(val: number): number {
  return Math.max(0, Math.min(1, val));
}

/**
 * Attempt to repair truncated JSON from LLM output.
 *
 * LLM responses may be cut off mid-token due to output length limits.
 * This results in valid JSON that is missing closing quotes, braces, or brackets.
 *
 * Strategy:
 * 1. Extract the JSON candidate (from first `{` or from ```json block)
 * 2. Close any open string literal (append `"`)
 * 3. Close open arrays `]` and objects `}`
 * 4. Try `JSON.parse`
 *
 * Only returns a result if the repaired JSON contains the essential `score` field.
 */
function tryRepairTruncatedJson(stdout: string): Record<string, unknown> | null {
  // Extract the JSON portion
  let jsonStr: string | null = null;

  // Try from ```json block first
  const blockStart = stdout.indexOf('```json');
  if (blockStart !== -1) {
    const contentStart = stdout.indexOf('\n', blockStart);
    if (contentStart !== -1) {
      const blockEnd = stdout.indexOf('```', contentStart);
      jsonStr = blockEnd !== -1
        ? stdout.slice(contentStart + 1, blockEnd)
        : stdout.slice(contentStart + 1); // block never closed — take everything after
    }
  }

  // Fallback: from first `{`
  if (!jsonStr) {
    const firstBrace = stdout.indexOf('{');
    if (firstBrace !== -1) {
      jsonStr = stdout.slice(firstBrace);
    }
  }

  if (!jsonStr || !jsonStr.includes('"score"')) return null;

  // Try parsing as-is first (maybe it's valid after trimming)
  try {
    return JSON.parse(jsonStr.trim()) as Record<string, unknown>;
  } catch { /* needs repair */ }

  // Repair: close open string, then close open brackets/braces
  let repaired = jsonStr.trimEnd();

  // If we're inside a string value (odd number of unescaped quotes), close it
  let inString = false;
  for (let i = 0; i < repaired.length; i++) {
    if (repaired[i] === '\\' && inString) { i++; continue; } // skip escaped char
    if (repaired[i] === '"') inString = !inString;
  }
  if (inString) {
    repaired += '"';
  }

  // Count open braces and brackets, append closers
  let openBraces = 0;
  let openBrackets = 0;
  let inStr = false;
  for (let i = 0; i < repaired.length; i++) {
    if (repaired[i] === '\\' && inStr) { i++; continue; }
    if (repaired[i] === '"') { inStr = !inStr; continue; }
    if (inStr) continue;
    if (repaired[i] === '{') openBraces++;
    else if (repaired[i] === '}') openBraces--;
    else if (repaired[i] === '[') openBrackets++;
    else if (repaired[i] === ']') openBrackets--;
  }

  // Remove trailing comma before closing (common in truncated output)
  repaired = repaired.replace(/,\s*$/, '');

  repaired += ']'.repeat(Math.max(0, openBrackets));
  repaired += '}'.repeat(Math.max(0, openBraces));

  try {
    const result = JSON.parse(repaired) as Record<string, unknown>;
    // Sanity check: must have a score
    if (typeof result['score'] === 'number' || typeof result['score'] === 'string') {
      logger.warn(`[evaluator] Repaired truncated JSON (added ${Math.max(0, openBraces)} braces, ${Math.max(0, openBrackets)} brackets)`);
      return result;
    }
  } catch { /* repair failed */ }

  return null;
}
