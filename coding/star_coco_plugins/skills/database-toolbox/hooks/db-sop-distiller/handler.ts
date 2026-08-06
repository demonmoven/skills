import { readFileSync, writeFileSync, mkdirSync, existsSync } from "fs";
import { join, resolve } from "path";

// ─── 类型 ───────────────────────────────────────────────────

interface HookEvent {
  type: string;
  action: string;
  sessionKey: string;
  timestamp: Date;
  messages: string[];
  context: {
    sessionEntry?: {
      sessionId?: string;
      transcript?: Array<{ role: string; content: string }>;
      file?: string;
    };
    sessionId?: string;
    sessionFile?: string;
    workspaceDir?: string;
    cfg?: Record<string, unknown>;
    [key: string]: unknown;
  };
}

interface Correction {
  wrongDirection: string;
  userGuidance: string;
  correctedApproach: string;
  lesson: string;
  date: string;
}

interface RedundantStep {
  step: string;
  reason: string;
  suggestion: string;
}

interface SOPEntry {
  date: string;
  instanceId: string;
  userTurns: number;
  steps: string[];
  corrections: Correction[];
  redundant: RedundantStep[];
}

// ─── 常量 ─────────────────────────────────────────────────

const DB_TOOLBOX_MARKERS = [
  "from toolbox import", "create_client", "list_tables", "get_table_info",
  "execute_sql", "nl2sql", "describe_slow_logs", "describe_aggregate_slow_logs",
  "describe_deadlock", "list_transactions", "list_active_sessions",
  "redis_list_big_keys", "describe_health_summary", "describe_table_space",
  "list_instances", "search_cached_instances", "describe_full_sql_detail",
  "transaction_snapshots", "slow_query_trend",
  "slow_query_advice", "get_metric_data_predict", "describe_instance_nodes",
  "get_ticket_url", "table_write_analysis", "export_transactions",
];

// 收紧到明确的"纠正 agent 方向"类指令，排除描述数据本身的表述
const CORRECTION_SIGNALS = [
  "方向不对", "不是这样查", "不是这个问题", "查错了", "看错了",
  "应该看", "你应该", "换个思路", "别看这个", "不要查",
  "跑偏了", "应该先", "不用查这个", "别纠结",
  "wrong approach", "should look at", "instead try",
];

// 排除：用户在描述数据/结果问题，不是纠正 agent
const CORRECTION_EXCLUDES = [
  "数据不对", "值不对", "结果不对", "数字不对", "显示不对",
  "格式不对", "图表不对", "报告不对",
  "data is wrong", "value is wrong", "result is wrong",
];

const SCENARIO_KEYWORDS: Record<string, string[]> = {
  "慢查询诊断": ["慢查询", "slow query", "slow log", "慢日志", "查询慢", "响应慢", "sql慢"],
  "死锁排查":   ["死锁", "deadlock", "lock wait", "锁等待", "锁超时"],
  "事务问题":   ["事务", "transaction", "trx", "长事务", "未提交"],
  "连接数问题": ["连接数", "connection", "too many connections", "processlist", "连接满"],
  "空间问题":   ["磁盘", "空间", "table space", "空间不足", "数据量大"],
  "大Key分析":  ["大key", "big key", "热key", "hot key", "redis.*内存"],
  "性能诊断":   ["cpu高", "内存高", "性能", "qps", "tps", "负载高"],
  "索引优化":   ["索引", "index", "全表扫描", "explain", "优化建议"],
  "数据查询":   ["查数据", "查一下", "select", "统计", "分析数据"],
  "元数据探查": ["有哪些表", "表结构", "建表语句", "库有什么", "list_tables"],
};

const SCENARIO_PRACTICES: Record<string, string[]> = {
  "慢查询诊断": [
    "先 describe_aggregate_slow_logs 拿 Top SQL 模板，再 describe_slow_logs 看明细",
    "对 Top SQL 执行 EXPLAIN 分析执行计划",
    "结合 describe_health_summary 判断是资源瓶颈还是 SQL 问题",
  ],
  "死锁排查": [
    "describe_deadlock → list_transactions → list_active_sessions 三步定位",
  ],
  "连接数问题": [
    "list_active_sessions 看实时连接，过滤 Sleep 状态",
    "list_transactions 检查长事务是否占用连接",
  ],
  "空间问题": [
    "describe_table_space 定位大表，判断数据增长 vs 碎片",
    "大表走 get_ticket_url 提交工单",
  ],
  "大Key分析": [
    "redis_list_big_keys 获取列表 → 按 Key 类型分析内存模式 → 评估拆分方案",
  ],
};

const SLUG_MAP: Record<string, string> = {
  "慢查询诊断": "slow-query", "死锁排查": "deadlock", "事务问题": "transaction",
  "连接数问题": "connections", "空间问题": "tablespace", "大Key分析": "big-key",
  "性能诊断": "performance", "索引优化": "index", "数据查询": "data-query",
  "元数据探查": "metadata", "通用数据库排查": "general",
};

const DEFAULT_MIN_TURNS = 4;
const SOP_DIR = "memory/db-sop";
const PROMOTE_THRESHOLD = 3;

// ─── 分析工具 ───────────────────────────────────────────────

function extractTranscriptText(sessionFile: string): string[] {
  try {
    const raw = readFileSync(sessionFile, "utf-8");
    const session = JSON.parse(raw);
    if (Array.isArray(session?.transcript)) {
      return session.transcript.map(
        (e: { role: string; content: string }) =>
          `[${e.role}]: ${typeof e.content === "string" ? e.content : JSON.stringify(e.content)}`
      );
    }
    return raw.split("\n").filter((l: string) => l.trim());
  } catch { return []; }
}

function hasDbToolboxUsage(lines: string[]): boolean {
  const t = lines.join("\n").toLowerCase();
  return DB_TOOLBOX_MARKERS.some((m) => t.includes(m.toLowerCase()));
}

/** 支持多场景：返回所有得分 > 0 且 >= top/2 的场景 */
function identifyScenarios(lines: string[]): string[] {
  const t = lines.join("\n").toLowerCase();
  const scored: Array<{ scenario: string; score: number }> = [];
  for (const [s, kws] of Object.entries(SCENARIO_KEYWORDS)) {
    const score = kws.reduce((a, k) => a + (t.includes(k.toLowerCase()) ? 1 : 0), 0);
    if (score > 0) scored.push({ scenario: s, score });
  }
  if (!scored.length) return ["通用数据库排查"];
  scored.sort((a, b) => b.score - a.score);
  const top = scored[0].score;
  return scored.filter((m) => m.score >= top / 2).map((m) => m.scenario);
}

function identifyDbType(lines: string[]): string {
  const t = lines.join("\n");
  // 用具体标识匹配，避免"缓存"等通用词误判
  if (/ByteRedis|redis_list_big_keys/i.test(t)) return "ByteRedis";
  if (/ByteDoc|文档数据库|mongodb/i.test(t)) return "ByteDoc";
  return "ByteRDS";
}

function extractInstanceId(lines: string[]): string {
  return lines.join("\n").match(/database\s*=\s*['"]([^'"]+)['"]/)?.[1] || "unknown";
}

/** 保留完整调用序列（含重复），还原真实排查路径 */
function extractFullStepSequence(lines: string[]): string[] {
  const steps: string[] = [];
  for (const line of lines) {
    if (!line.startsWith("[assistant]:") && !line.startsWith("[ai]:")) continue;
    for (const m of DB_TOOLBOX_MARKERS) {
      if (line.toLowerCase().includes(m.toLowerCase())) { steps.push(m); break; }
    }
  }
  return steps;
}

function detectCorrections(lines: string[]): Correction[] {
  const out: Correction[] = [];
  const date = new Date().toISOString().split("T")[0];

  for (let i = 0; i < lines.length; i++) {
    if (!lines[i].startsWith("[user]:") && !lines[i].startsWith("[human]:")) continue;
    const content = lines[i].replace(/^\[(user|human)\]:\s*/i, "");
    const lower = content.toLowerCase();

    // 排除：描述数据/结果问题
    if (CORRECTION_EXCLUDES.some((e) => lower.includes(e.toLowerCase()))) continue;
    // 要求命中纠偏信号
    if (!CORRECTION_SIGNALS.some((s) => lower.includes(s.toLowerCase()))) continue;

    // 上文：agent 的错误方向
    let wrong = "";
    for (let j = i - 1; j >= Math.max(0, i - 5); j--) {
      if (lines[j].startsWith("[assistant]:") || lines[j].startsWith("[ai]:")) {
        wrong = lines[j].replace(/^\[(assistant|ai)\]:\s*/i, "").slice(0, 300);
        break;
      }
    }
    // 下文：修正后 agent 的做法（扩大搜索范围到 8 行）
    let corrected = "";
    for (let k = i + 1; k < Math.min(lines.length, i + 8); k++) {
      if (lines[k].startsWith("[assistant]:") || lines[k].startsWith("[ai]:")) {
        corrected = lines[k].replace(/^\[(assistant|ai)\]:\s*/i, "").slice(0, 300);
        break;
      }
    }
    // 验证：修正后的回复应与错误方向有差异
    if (wrong && corrected && wrong === corrected) continue;

    out.push({
      wrongDirection: wrong || "（无法回溯）",
      userGuidance: content.slice(0, 400),
      correctedApproach: corrected || "（未捕获）",
      lesson: `排查方向被修正："${content.slice(0, 120)}"`,
      date,
    });
  }
  return out;
}

/** 冗余检测：按 函数名+database 签名去重 */
function detectRedundantSteps(lines: string[]): RedundantStep[] {
  const out: RedundantStep[] = [];
  const map: Map<string, number> = new Map();
  for (const line of lines) {
    for (const m of DB_TOOLBOX_MARKERS) {
      if (line.toLowerCase().includes(m.toLowerCase())) {
        const db = line.match(/database\s*=\s*['"]([^'"]+)['"]/)?.[1] || "?";
        const psm = line.match(/psm\s*=\s*['"]([^'"]+)['"]/)?.[1];
        const sig = `${m}|${psm || db}`;
        map.set(sig, (map.get(sig) || 0) + 1);
      }
    }
  }
  for (const [sig, count] of map) {
    if (count > 2) {
      const fn = sig.split("|")[0];
      out.push({
        step: `${fn} ×${count}（同实例同库）`,
        reason: "重复执行相同查询，排查路径不够清晰",
        suggestion: `先明确目标，一次性收集 ${fn} 所需信息`,
      });
    }
  }
  return out;
}

// ─── SOP 合并与渲染 ──────────────────────────────────────────

function sopFilename(scenario: string, dbType: string): string {
  const slug = SLUG_MAP[scenario] || "general";
  return `${slug}-${dbType.toLowerCase().replace("byte", "")}.md`;
}

/** 从已有 SOP 文件解析历史条目（嵌入在 HTML 注释中） */
function parseExistingEntries(filepath: string): SOPEntry[] {
  if (!existsSync(filepath)) return [];
  try {
    const content = readFileSync(filepath, "utf-8");
    const match = content.match(/<!-- entries:([\s\S]+?) -->/);
    if (match) return JSON.parse(match[1]);
  } catch { /* ignore */ }
  return [];
}

function renderMergedSOP(scenario: string, dbType: string, entries: SOPEntry[]): string {
  // 聚合所有 entries
  const stepFreq = new Map<string, number>();
  const allCorrections: Correction[] = [];
  const allRedundant: RedundantStep[] = [];

  for (const e of entries) {
    for (const s of e.steps) {
      stepFreq.set(s, (stepFreq.get(s) || 0) + 1);
    }
    allCorrections.push(...e.corrections);
    allRedundant.push(...e.redundant);
  }

  const sortedSteps = [...stepFreq.entries()]
    .sort((a, b) => b[1] - a[1])
    .map(([step]) => step);

  const practices = SCENARIO_PRACTICES[scenario] || [];
  const total = entries.length;

  let md = `---\nname: ${scenario}-${dbType}\n`;
  md += `description: ${scenario}（${dbType}）排查 SOP，基于 ${total} 次实际会话蒸馏\n`;
  md += `type: reference\n---\n\n`;
  md += `# ${scenario}（${dbType}）\n\n`;
  md += `> 基于 ${total} 次排查会话自动蒸馏 · 最近更新 ${new Date().toISOString().split("T")[0]}\n\n`;

  // 推荐路径（按使用频率排序）
  md += `## 推荐排查路径\n\n`;
  sortedSteps.forEach((s, i) => {
    const freq = stepFreq.get(s)!;
    md += `${i + 1}. \`${s}\`${freq > 1 && total > 1 ? ` (${freq}/${total} 次)` : ""}\n`;
  });
  md += "\n";

  // 最佳实践
  if (practices.length) {
    md += `## 最佳实践\n\n`;
    practices.forEach((p) => { md += `- ${p}\n`; });
    md += "\n";
  }

  // 纠偏记录（最多保留最近 10 条）
  if (allCorrections.length) {
    md += `## 纠偏记录（${allCorrections.length} 次）\n\n`;
    const recent = allCorrections.slice(-10);
    recent.forEach((c, i) => {
      md += `### ${c.date} #${i + 1}\n\n`;
      md += `- **错误方向**: ${c.wrongDirection}\n`;
      md += `- **用户指导**: ${c.userGuidance}\n`;
      md += `- **修正后**: ${c.correctedApproach}\n`;
      md += `- **教训**: ${c.lesson}\n\n`;
    });
  }

  // 冗余步骤（最多保留最近 5 条）
  if (allRedundant.length) {
    md += `## 冗余步骤\n\n`;
    allRedundant.slice(-5).forEach((r) => {
      md += `- **${r.step}**: ${r.reason} → ${r.suggestion}\n`;
    });
    md += "\n";
  }

  // 反模式：聚合相似纠偏，按频率排序，标注 promote 候选
  const patternCounts = new Map<string, number>();
  for (const c of allCorrections) {
    patternCounts.set(c.lesson, (patternCounts.get(c.lesson) || 0) + 1);
  }
  if (patternCounts.size) {
    md += `## 反模式摘要\n\n`;
    const sorted = [...patternCounts.entries()].sort((a, b) => b[1] - a[1]);
    for (const [pattern, count] of sorted) {
      const flag = count >= PROMOTE_THRESHOLD ? " **[建议 promote 到 AGENTS.md]**" : "";
      md += `- [×${count}] ${pattern}${flag}\n`;
    }
    md += "\n";
  }

  // 嵌入结构化数据供下次合并
  md += `<!-- entries:${JSON.stringify(entries)} -->\n`;

  return md;
}

// ─── Handler ────────────────────────────────────────────────

const handler = async (event: HookEvent): Promise<void> => {
  if (event.type !== "command" || event.action !== "new") return;

  try {
    const workspaceDir = event.context.workspaceDir;
    if (!workspaceDir) return;
    const minTurns = parseInt(process.env.DB_SOP_MIN_TURNS || String(DEFAULT_MIN_TURNS), 10);

    // 读取上一次 session
    let lines: string[] = [];
    if (event.context.sessionEntry?.transcript) {
      lines = event.context.sessionEntry.transcript.map(
        (e: { role: string; content: string }) =>
          `[${e.role}]: ${typeof e.content === "string" ? e.content : JSON.stringify(e.content)}`
      );
    } else {
      const f = event.context.sessionFile || event.context.sessionEntry?.file;
      if (f) lines = extractTranscriptText(f);
    }
    if (!lines.length || !hasDbToolboxUsage(lines)) return;

    const userTurns = lines.filter((l) => /^\[(user|human)\]:/.test(l)).length;
    if (userTurns < minTurns) return;

    // 分析
    const scenarios   = identifyScenarios(lines);
    const dbType      = identifyDbType(lines);
    const instanceId  = extractInstanceId(lines);
    const fullSteps   = extractFullStepSequence(lines);
    const steps       = [...new Set(fullSteps)];
    const corrections = detectCorrections(lines);
    const redundant   = detectRedundantSteps(lines);

    // 跳过无价值会话
    if (corrections.length === 0 && redundant.length === 0 && steps.length < 2) return;

    const sopDir = resolve(workspaceDir, SOP_DIR);
    mkdirSync(sopDir, { recursive: true });

    const date = new Date().toISOString().split("T")[0];
    const newEntry: SOPEntry = { date, instanceId, userTurns, steps, corrections, redundant };

    // 对每个匹配场景合并写入 SOP
    const writtenFiles: string[] = [];
    for (const scenario of scenarios) {
      const filename = sopFilename(scenario, dbType);
      const filepath = join(sopDir, filename);

      const existing = parseExistingEntries(filepath);
      existing.push(newEntry);

      const sop = renderMergedSOP(scenario, dbType, existing);
      writeFileSync(filepath, sop, "utf-8");
      writtenFiles.push(filename);
      console.log(`[db-sop-distiller] wrote ${sopDir}/${filename} (${existing.length} sessions)`);
    }

    // 注入消息到新 session
    const parts = [];
    if (corrections.length) parts.push(`${corrections.length} 次纠偏`);
    if (redundant.length) parts.push(`${redundant.length} 处冗余`);
    if (steps.length >= 2) parts.push(`${steps.length} 步路径`);

    const files = writtenFiles.map((f) => `memory/db-sop/${f}`).join(", ");
    event.messages.push(
      `DB SOP 已更新: ${scenarios.join("/")} → ${files}（${parts.join("，")}）`
    );
  } catch (err) {
    console.error("[db-sop-distiller]", err instanceof Error ? err.message : String(err));
  }
};

export default handler;
