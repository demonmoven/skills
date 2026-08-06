import type { AnalyzeResult, ProcessedEntry, Summary, ToolCall, WorkDirNode } from './types.js';
import { STYLES_CSS } from './styles.js';
import { INTERACTIVITY_JS } from './interactivity.js';

/**
 * Render the final single-file HTML report.
 *
 * Approach: pure server-side string concatenation. No client-side templating,
 * no data injection — the DOM is fully baked at build time, JS only handles
 * click-to-expand/collapse and tab switching.
 */
export function renderHtml(result: AnalyzeResult): string {
  const { summary } = result;
  const hasWorkDir = result.workDir !== undefined;

  const html = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>xdev trace analyze · ${escapeHtml(summary.sessionId)}</title>
<style>${STYLES_CSS}</style>
</head>
<body>
${renderHeader(summary, result.cwd)}
<nav class="tabs">
  <button class="tab-button active" data-tab="summary">Summary</button>
${hasWorkDir ? '  <button class="tab-button" data-tab="workdir">Work Directory</button>\n' : ''}  <button class="tab-button" data-tab="trace">Trace</button>
</nav>
<main>
<section class="tab-pane active" data-tab="summary">
${renderSummaryTab(summary)}
</section>
${hasWorkDir ? `<section class="tab-pane" data-tab="workdir">
${renderWorkDirTab(result.workDir!, result.cwd)}
</section>
` : ''}<section class="tab-pane" data-tab="trace">
${renderTraceTab(result.mainEntries, summary)}
</section>
</main>
${hasWorkDir ? renderWorkDirDataScript(result.workDir!) : ''}
<script>${INTERACTIVITY_JS}</script>
</body>
</html>`;

  return html;
}

// ============ Header ============

function renderHeader(s: Summary, cwd?: string): string {
  const modelPart = s.model ? ` · ${escapeHtml(s.model)}` : '';
  const cwdPart = cwd ? ` · ${escapeHtml(cwd)}` : '';
  const durationStr = formatDuration(s.durationMs);
  return `
<header class="report-header">
  <h1>Session Analysis Report</h1>
  <div class="report-meta">${escapeHtml(s.sessionId)} · ${durationStr}${modelPart}${cwdPart}</div>
</header>`;
}

// ============ Summary tab ============

function renderSummaryTab(s: Summary): string {
  const totalTokens = s.tokenTotals.input + s.tokenTotals.output;
  const skillCount = s.skillInvocations.length;
  const cards = [
    card('DURATION', formatDuration(s.durationMs), fmtDate(s.startedAt)),
    card('TURNS', fmtNumber(s.totalTurns), 'user prompts'),
    card('TOOL CALLS', fmtNumber(s.totalToolCalls), `${Object.keys(s.toolDistribution).length} tool types`),
    card('TOKENS', fmtNumber(totalTokens), `${fmtNumber(s.tokenTotals.input)} in / ${fmtNumber(s.tokenTotals.output)} out`, 'accent'),
    card('CACHE READ', fmtNumber(s.tokenTotals.cacheRead), 'cached tokens', 'purple'),
    card('ERRORS', fmtNumber(s.errorCount), `${s.totalToolCalls > 0 ? ((s.errorCount / s.totalToolCalls) * 100).toFixed(1) : '0'}% of tool calls`, s.errorCount > 0 ? 'red' : 'green'),
    card('SUBAGENTS', fmtNumber(s.subagentCount), Object.keys(s.subagentTypes).join(', ') || 'none', 'purple'),
    card('SKILLS', fmtNumber(skillCount), Object.keys(s.skillDistribution).join(', ') || 'none', skillCount > 0 ? 'accent' : undefined),
  ];

  return `
<div class="metric-cards">
${cards.join('\n')}
</div>

<div class="panels">
  <div class="panel">
    <h3>Tool Usage Distribution</h3>
    ${renderBarChart(s.toolDistribution)}
  </div>
  <div class="panel">
    <h3>Token Breakdown</h3>
    <div class="kv">
      <div class="k">Input tokens</div><div class="v">${fmtNumber(s.tokenTotals.input)}</div>
      <div class="k">Output tokens</div><div class="v">${fmtNumber(s.tokenTotals.output)}</div>
      <div class="k">Cache read</div><div class="v">${fmtNumber(s.tokenTotals.cacheRead)}</div>
      <div class="k">Cache creation</div><div class="v">${fmtNumber(s.tokenTotals.cacheCreation)}</div>
    </div>
    ${Object.keys(s.subagentTypes).length > 0 ? `
    <h3 style="margin-top:24px">Subagent Types</h3>
    ${renderBarChart(s.subagentTypes)}
    ` : ''}
  </div>
</div>

${renderSkillsPanel(s)}
`;
}

function renderSkillsPanel(s: Summary): string {
  const hasSkills = s.skillInvocations.length > 0;
  const distChart = hasSkills ? renderBarChart(s.skillDistribution) : '<div class="subtext">No skill invocations in this session.</div>';

  const invocationList = hasSkills
    ? s.skillInvocations
        .map((inv) => {
          const argsPreview = inv.args ? truncate(inv.args, 180) : '';
          const depthBadge = inv.depth > 0 ? `<span class="tool-badge subagent">in subagent</span>` : '';
          const errBadge = inv.isError ? `<span class="tool-badge error">ERROR</span>` : '';
          return `<div class="skill-invocation">
<div class="skill-row-header">
<span class="skill-name">${escapeHtml(inv.skillName)}</span>
<span class="timestamp">${escapeHtml(formatTimestamp(inv.timestamp))}</span>
${depthBadge}
${errBadge}
</div>
${argsPreview ? `<div class="skill-args">${escapeHtml(argsPreview)}</div>` : ''}
</div>`;
        })
        .join('\n')
    : '';

  return `
<div class="panel" style="margin-top:16px">
  <h3>Skills Invoked ${hasSkills ? `<span class="count-pill">${s.skillInvocations.length}</span>` : ''}</h3>
  ${distChart}
  ${hasSkills ? `<h3 style="margin-top:24px">Invocation Timeline</h3>
  <div class="skill-list">
  ${invocationList}
  </div>` : ''}
</div>
`;
}

function card(
  title: string,
  value: string,
  subtext: string,
  valueClass?: string,
): string {
  const clsAttr = valueClass ? ` ${valueClass}` : '';
  return `<div class="card">
<h3>${escapeHtml(title)}</h3>
<div class="value${clsAttr}">${escapeHtml(value)}</div>
<div class="subtext">${escapeHtml(subtext)}</div>
</div>`;
}

function renderBarChart(dist: Record<string, number>): string {
  const entries = Object.entries(dist).sort((a, b) => b[1] - a[1]);
  if (entries.length === 0) {
    return '<div class="subtext">No data</div>';
  }
  const max = Math.max(...entries.map((e) => e[1]));
  return entries
    .map(([label, count]) => {
      const pct = max > 0 ? (count / max) * 100 : 0;
      return `<div class="bar-row">
  <div class="label">${escapeHtml(label)}</div>
  <div class="bar"><div class="bar-fill" style="width:${pct.toFixed(1)}%"></div></div>
  <div class="count">${fmtNumber(count)}</div>
</div>`;
    })
    .join('\n');
}

// ============ Trace tab ============

function renderTraceTab(entries: ProcessedEntry[], summary: Summary): string {
  return `
<div class="toolbar">
  <input class="filter-input" type="search" placeholder="Filter text (/ to focus, Esc to clear)">
  <button class="expand-all">Expand all</button>
  <button class="collapse-all">Collapse all</button>
</div>
${renderToolChipBar(summary.toolDistribution)}
<div class="trajectory">
${entries.map((e) => renderNode(e)).join('\n')}
</div>
`;
}

function renderToolChipBar(dist: Record<string, number>): string {
  const entries = Object.entries(dist).sort((a, b) => b[1] - a[1]);
  if (entries.length === 0) return '';
  const chips = entries
    .map(
      ([tool, count]) =>
        `<span class="tool-chip" data-tool="${escapeHtml(tool)}"><span class="tool-pill" data-tool="${escapeHtml(tool)}">${escapeHtml(tool)}</span><span class="chip-count">${count}</span></span>`,
    )
    .join('\n');
  return `<div class="tool-chip-bar" title="Click to filter by tool (multi-select)">
<span class="tool-chip chip-all active" data-tool="__all__">All</span>
${chips}
</div>`;
}

// ============ Work Directory tab ============

function renderWorkDirTab(root: WorkDirNode, cwd?: string): string {
  return `
<div class="toolbar">
  <span style="color:var(--text-dim);font-family:var(--mono);font-size:12px">Browsing <code>${escapeHtml(cwd ?? '')}/docs/</code></span>
</div>
<div class="workdir-container">
  <div class="workdir-tree">
${renderWorkDirTreeNode(root, true)}
  </div>
  <div class="workdir-content" id="workdir-content">
    <div class="empty-state">Select a file on the left to view its content</div>
  </div>
</div>
`;
}

function renderWorkDirTreeNode(node: WorkDirNode, isRoot = false): string {
  if (node.isDir) {
    const children = node.children ?? [];
    const childrenHtml = children.map((c) => renderWorkDirTreeNode(c)).join('\n');
    if (isRoot) {
      return `<ul>
<li>
  <div class="node-dir" data-dir>
    <span class="dir-icon">▼</span>
    <span>${escapeHtml(node.name)}/</span>
  </div>
  <ul>
${childrenHtml}
  </ul>
</li>
</ul>`;
    }
    return `<li class="collapsed">
  <div class="node-dir" data-dir>
    <span class="dir-icon">▶</span>
    <span>${escapeHtml(node.name)}/</span>
  </div>
  <ul>
${childrenHtml}
  </ul>
</li>`;
  }
  // File node
  const sizeStr = node.size !== undefined ? `${formatFileSize(node.size)}` : '';
  const skippedAttr = node.contentSkipped ? ` data-skipped="${escapeHtml(node.contentSkipped)}"` : '';
  return `<li>
  <div class="node-file" data-file data-path="${escapeHtml(node.relativePath)}"${skippedAttr}>
    <span class="file-icon">📄</span>
    <span>${escapeHtml(node.name)}</span>
    <span class="file-size">${sizeStr}</span>
  </div>
</li>`;
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes}B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}K`;
  return `${(bytes / 1024 / 1024).toFixed(1)}M`;
}

/**
 * Emit a <script type="application/json"> block containing the files map
 * (relativePath → content). Interactivity JS reads this on file click.
 */
function renderWorkDirDataScript(root: WorkDirNode): string {
  const files: Record<string, string> = {};
  collectFiles(root, files);
  // Use textContent-safe JSON embedding
  const json = JSON.stringify(files).replace(/</g, '\\u003c').replace(/>/g, '\\u003e');
  return `<script id="workdir-files" type="application/json">${json}</script>`;
}

function collectFiles(node: WorkDirNode, out: Record<string, string>): void {
  if (!node.isDir && node.content !== undefined) {
    out[node.relativePath] = node.content;
  }
  for (const child of node.children ?? []) collectFiles(child, out);
}

function renderNode(entry: ProcessedEntry): string {
  const depthCls = entry.depth > 0 ? ` depth-${Math.min(entry.depth, 3)}` : '';
  const typeCls = `badge-${entry.type}`;
  const summary = nodeSummaryText(entry);
  const timestamp = formatTimestamp(entry.timestamp);
  const tokens = entry.tokens
    ? `${fmtNumber(entry.tokens.inputTokens + entry.tokens.outputTokens)} tok`
    : '';

  const hasText = entry.text && entry.text.trim().length > 0;
  const hasToolCalls = entry.toolCalls.length > 0;
  const hasBody = hasText || hasToolCalls;

  // If there's nothing to show in body, don't render a caret (still render header)
  const caret = hasBody ? '<span class="caret">▶</span>' : '<span class="caret" style="visibility:hidden"></span>';

  const body = hasBody
    ? `<div class="node-body">
${hasText ? `<div class="full-text">${escapeHtml(entry.text as string)}</div>` : ''}
${hasToolCalls ? entry.toolCalls.map((tc) => renderToolCall(tc)).join('\n') : ''}
</div>`
    : '';

  return `<div class="node${depthCls}" data-type="${escapeHtml(entry.type)}" data-depth="${entry.depth}">
<div class="node-header">
${caret}
<span class="type-badge ${typeCls}">${typeLabel(entry.type)}</span>
<span class="timestamp">${escapeHtml(timestamp)}</span>
<span class="summary-text">${escapeHtml(summary)}</span>
${tokens ? `<span class="tokens-badge">${escapeHtml(tokens)}</span>` : ''}
</div>
${body}
</div>`;
}

function renderToolCall(tc: ToolCall): string {
  const isError = tc.result?.isError === true;
  const isAgent = tc.name === 'Agent' && tc.subagent;
  const badgeHtml = isError
    ? '<span class="tool-badge error">ERROR</span>'
    : isAgent
    ? `<span class="tool-badge subagent">${escapeHtml(tc.subagent!.agentType)}</span>`
    : '';

  const summary = toolCallSummary(tc);

  const inputPretty = safeJsonStringify(tc.input);
  const resultPretty = tc.result ? formatToolResult(tc.result.content) : '';

  const subagentBlock = tc.subagent
    ? `
<h4>Subagent: ${escapeHtml(tc.subagent.agentType)}${tc.subagent.description ? ' — ' + escapeHtml(tc.subagent.description) : ''}</h4>
<div class="subagent-stats">
${fmtNumber(tc.subagent.aggregates.totalToolUseCount ?? 0)} tools · ${fmtNumber(tc.subagent.aggregates.totalTokens ?? 0)} tokens · ${formatDuration(tc.subagent.aggregates.totalDurationMs ?? 0)}
</div>
<div class="subagent-trajectory">
${tc.subagent.processed.map((e) => renderNode(e)).join('\n')}
</div>
`
    : '';

  return `<div class="tool-call" data-tool="${escapeHtml(tc.name)}">
<div class="tool-header">
<span class="caret">▶</span>
<span class="tool-pill" data-tool="${escapeHtml(tc.name)}">${escapeHtml(tc.name)}</span>
<span class="tool-summary">${escapeHtml(summary)}</span>
${badgeHtml}
</div>
<div class="tool-body">
<h4>Input</h4>
<pre>${escapeHtml(inputPretty)}</pre>
${tc.result ? `<h4>Result${isError ? ' (error)' : ''}</h4>
<pre>${escapeHtml(resultPretty)}</pre>` : '<p class="subtext">No result recorded</p>'}
${subagentBlock}
</div>
</div>`;
}

// ============ Display helpers ============

function nodeSummaryText(e: ProcessedEntry): string {
  if (e.type === 'user-text') return truncate(e.text ?? '(empty)', 120);
  if (e.type === 'assistant-text') {
    const hasTools = e.toolCalls.length > 0;
    const toolHint = hasTools
      ? ` [${e.toolCalls.length} tool${e.toolCalls.length > 1 ? 's' : ''}: ${e.toolCalls.map((t) => t.name).join(', ')}]`
      : '';
    const text = e.text ? truncate(e.text, 120 - toolHint.length) : '';
    return text + toolHint;
  }
  if (e.type === 'assistant-thinking') return `💭 ${truncate(e.text ?? '', 100)}`;
  if (e.type === 'user-tool-result') {
    const firstTool = e.toolCalls[0];
    return firstTool ? `← ${firstTool.name} result` : '← tool result';
  }
  if (e.type === 'system') return truncate(e.text ?? '(system event)', 120);
  return '(event)';
}

function typeLabel(t: string): string {
  switch (t) {
    case 'user-text': return 'USER';
    case 'user-tool-result': return 'RESULT';
    case 'assistant-text': return 'ASSISTANT';
    case 'assistant-thinking': return 'THINKING';
    case 'system': return 'SYSTEM';
    default: return t.toUpperCase();
  }
}

function toolCallSummary(tc: ToolCall): string {
  const input = tc.input as Record<string, unknown> | undefined;
  if (!input || typeof input !== 'object') return '';

  // Heuristic summaries per tool
  switch (tc.name) {
    case 'Bash':
      return truncate(String(input['command'] ?? ''), 120);
    case 'Read':
      return truncate(String(input['file_path'] ?? ''), 120);
    case 'Write':
      return truncate(String(input['file_path'] ?? ''), 120);
    case 'Edit':
    case 'MultiEdit':
      return truncate(String(input['file_path'] ?? ''), 120);
    case 'Grep':
      return truncate(String(input['pattern'] ?? ''), 120);
    case 'Glob':
      return truncate(String(input['pattern'] ?? ''), 120);
    case 'Agent':
      return truncate(String(input['description'] ?? ''), 120);
    case 'Skill':
      return truncate(String(input['skill'] ?? input['name'] ?? ''), 120);
    default: {
      // Fallback: first string field
      for (const [, v] of Object.entries(input)) {
        if (typeof v === 'string') return truncate(v, 120);
      }
      return '';
    }
  }
}

function formatToolResult(content: unknown): string {
  if (content == null) return '';
  if (typeof content === 'string') return content;
  if (Array.isArray(content)) {
    // content array: [{ type: 'text', text: '...' }, ...]
    return content
      .map((c: unknown) => {
        if (typeof c === 'string') return c;
        if (c && typeof c === 'object' && 'text' in c) {
          return String((c as { text: unknown }).text ?? '');
        }
        return safeJsonStringify(c);
      })
      .join('\n\n');
  }
  return safeJsonStringify(content);
}

function safeJsonStringify(v: unknown): string {
  try {
    return JSON.stringify(v, null, 2);
  } catch {
    return String(v);
  }
}

// ============ Formatters ============

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function truncate(s: string, n: number): string {
  if (!s) return '';
  const clean = s.replace(/\s+/g, ' ').trim();
  if (clean.length <= n) return clean;
  return clean.slice(0, n - 1) + '…';
}

function fmtNumber(n: number): string {
  if (n < 1000) return String(n);
  if (n < 1000000) return (n / 1000).toFixed(1).replace(/\.0$/, '') + 'k';
  return (n / 1000000).toFixed(2).replace(/\.00$/, '') + 'M';
}

function formatDuration(ms: number): string {
  if (!ms || ms < 0) return '-';
  const s = Math.floor(ms / 1000);
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  if (h > 0) return `${h}h ${m}m ${sec}s`;
  if (m > 0) return `${m}m ${sec}s`;
  return `${sec}s`;
}

function formatTimestamp(iso?: string): string {
  if (!iso) return '';
  try {
    const d = new Date(iso);
    const h = String(d.getHours()).padStart(2, '0');
    const m = String(d.getMinutes()).padStart(2, '0');
    const s = String(d.getSeconds()).padStart(2, '0');
    return `${h}:${m}:${s}`;
  } catch {
    return '';
  }
}

function fmtDate(iso?: string): string {
  if (!iso) return '';
  try {
    const d = new Date(iso);
    return d.toISOString().slice(0, 10);
  } catch {
    return '';
  }
}
