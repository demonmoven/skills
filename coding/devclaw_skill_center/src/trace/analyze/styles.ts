/**
 * Inline CSS for the analyze report. Dark-theme only for v1.
 *
 * Color palette modeled on the GitHub dark theme + the screenshot reference.
 */
export const STYLES_CSS = `
:root {
  --bg: #0d1117;
  --bg-elev: #161b22;
  --bg-hover: #1f2933;
  --border: #30363d;
  --border-strong: #3d4450;
  --text: #e6edf3;
  --text-dim: #8b949e;
  --text-faint: #6e7681;
  --accent: #58a6ff;
  --accent-dim: #1f6feb;
  --green: #3fb950;
  --red: #f85149;
  --yellow: #d29922;
  --purple: #bc8cff;
  --cyan: #39c5cf;
  --mono: 'SF Mono', Menlo, Monaco, 'Courier New', monospace;
}

* { box-sizing: border-box; }

html, body {
  margin: 0;
  padding: 0;
  background: var(--bg);
  color: var(--text);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;
  font-size: 14px;
  line-height: 1.5;
}

body {
  padding: 24px 32px 48px;
  max-width: 1400px;
  margin: 0 auto;
}

.report-header {
  border-bottom: 1px solid var(--border);
  padding-bottom: 20px;
  margin-bottom: 24px;
}
.report-header h1 {
  margin: 0 0 6px;
  font-size: 24px;
  font-weight: 600;
  letter-spacing: -0.02em;
}
.report-meta {
  color: var(--text-dim);
  font-family: var(--mono);
  font-size: 12px;
}

.tabs {
  display: flex;
  gap: 24px;
  border-bottom: 1px solid var(--border);
  margin-bottom: 24px;
}
.tab-button {
  background: none;
  border: none;
  padding: 12px 2px;
  font-size: 14px;
  color: var(--text-dim);
  cursor: pointer;
  border-bottom: 2px solid transparent;
  transition: color 0.15s, border-color 0.15s;
  font-family: inherit;
}
.tab-button:hover { color: var(--text); }
.tab-button.active {
  color: var(--accent);
  border-bottom-color: var(--accent);
}

.tab-pane { display: none; }
.tab-pane.active { display: block; }

/* ===== Summary ===== */
.metric-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 32px;
}
.card {
  background: var(--bg-elev);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 20px;
}
.card h3 {
  margin: 0 0 12px;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-dim);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}
.card .value {
  font-size: 28px;
  font-weight: 600;
  line-height: 1.1;
}
.card .value.accent { color: var(--accent); }
.card .value.green { color: var(--green); }
.card .value.red { color: var(--red); }
.card .value.yellow { color: var(--yellow); }
.card .value.purple { color: var(--purple); }
.card .subtext {
  font-size: 12px;
  color: var(--text-dim);
  margin-top: 6px;
}

.panels {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-top: 16px;
}
@media (max-width: 900px) { .panels { grid-template-columns: 1fr; } }

.panel {
  background: var(--bg-elev);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 20px;
}
.panel h3 {
  margin: 0 0 16px;
  font-size: 14px;
  font-weight: 600;
}

.bar-row {
  display: grid;
  grid-template-columns: 140px 1fr 60px;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
  font-size: 12px;
}
.bar-row .label {
  font-family: var(--mono);
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.bar-row .bar {
  height: 8px;
  background: var(--bg-hover);
  border-radius: 4px;
  overflow: hidden;
}
.bar-row .bar-fill {
  height: 100%;
  background: var(--accent);
  border-radius: 4px;
}
.bar-row .count {
  font-family: var(--mono);
  color: var(--text-dim);
  text-align: right;
}

.kv {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 8px 16px;
  font-family: var(--mono);
  font-size: 13px;
}
.kv .k { color: var(--text-dim); }
.kv .v { color: var(--text); text-align: right; }

/* ===== Trajectory ===== */
.toolbar {
  display: flex;
  gap: 12px;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border);
}
.filter-input {
  flex: 1;
  background: var(--bg-elev);
  border: 1px solid var(--border);
  color: var(--text);
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 13px;
  font-family: inherit;
}
.filter-input:focus {
  outline: none;
  border-color: var(--accent);
}
.toolbar button {
  background: var(--bg-elev);
  border: 1px solid var(--border);
  color: var(--text);
  padding: 8px 14px;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  font-family: inherit;
}
.toolbar button:hover { background: var(--bg-hover); }

.trajectory {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.node {
  border-left: 2px solid var(--border);
  padding-left: 0;
  margin-left: 0;
}
.node.depth-1 { margin-left: 24px; border-left-color: var(--accent-dim); }
.node.depth-2 { margin-left: 48px; border-left-color: var(--purple); }
.node.depth-3 { margin-left: 72px; }

.node-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  cursor: pointer;
  user-select: none;
  border-radius: 4px;
  transition: background 0.1s;
}
.node-header:hover { background: var(--bg-hover); }

.caret {
  font-size: 10px;
  color: var(--text-faint);
  width: 12px;
  transition: transform 0.1s;
  flex-shrink: 0;
  display: inline-block;
}
.node.expanded > .node-header .caret,
.tool-call.expanded > .tool-header .caret {
  transform: rotate(90deg);
}

.type-badge {
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 3px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  flex-shrink: 0;
}
.badge-user-text { background: rgba(88, 166, 255, 0.15); color: var(--accent); }
.badge-user-tool-result { background: rgba(188, 140, 255, 0.15); color: var(--purple); }
.badge-assistant-text { background: rgba(63, 185, 80, 0.15); color: var(--green); }
.badge-assistant-thinking { background: rgba(210, 153, 34, 0.15); color: var(--yellow); }
.badge-system { background: rgba(139, 148, 158, 0.15); color: var(--text-dim); }
.badge-other { background: rgba(139, 148, 158, 0.15); color: var(--text-faint); }

.timestamp {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--text-faint);
  flex-shrink: 0;
}
.summary-text {
  flex: 1;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}
.tokens-badge {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--text-faint);
  flex-shrink: 0;
}

.node-body {
  padding: 8px 12px 12px 32px;
  display: none;
  border-left: 1px dashed var(--border);
  margin-left: 6px;
}
.node.expanded > .node-body { display: block; }

.node-body .full-text {
  white-space: pre-wrap;
  word-wrap: break-word;
  background: var(--bg-elev);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 10px 12px;
  font-family: var(--mono);
  font-size: 12px;
  color: var(--text);
  max-height: 400px;
  overflow: auto;
}

/* ===== Tool calls ===== */
.tool-call {
  background: var(--bg-elev);
  border: 1px solid var(--border);
  border-radius: 6px;
  margin-top: 8px;
  overflow: hidden;
}
.tool-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  user-select: none;
}
.tool-header:hover { background: var(--bg-hover); }
.tool-icon { font-size: 12px; flex-shrink: 0; opacity: 0.7; }

/* Tool name is rendered as a colored pill. Each known tool has its own
 * palette entry; unknown tools fall back to .tool-pill default styling. */
.tool-pill {
  display: inline-flex;
  align-items: center;
  padding: 2px 10px;
  border-radius: 11px;
  font-family: var(--mono);
  font-size: 11px;
  font-weight: 600;
  background: rgba(139, 148, 158, 0.15);
  color: var(--text-dim);
  border: 1px solid rgba(139, 148, 158, 0.3);
  flex-shrink: 0;
  letter-spacing: 0.02em;
}
.tool-pill[data-tool="Agent"]     { background: rgba(188,140,255,0.15); color: #d2a8ff; border-color: rgba(188,140,255,0.35); }
.tool-pill[data-tool="Skill"]     { background: rgba(57,197,207,0.15);  color: #56d4dd; border-color: rgba(57,197,207,0.35); }
.tool-pill[data-tool="Bash"]      { background: rgba(210,153,34,0.15);  color: #e3b341; border-color: rgba(210,153,34,0.35); }
.tool-pill[data-tool="Read"]      { background: rgba(63,185,80,0.15);   color: #7ee787; border-color: rgba(63,185,80,0.35); }
.tool-pill[data-tool="Write"]     { background: rgba(88,166,255,0.15);  color: #79c0ff; border-color: rgba(88,166,255,0.35); }
.tool-pill[data-tool="Edit"],
.tool-pill[data-tool="MultiEdit"] { background: rgba(240,136,62,0.15);  color: #ffa657; border-color: rgba(240,136,62,0.35); }
.tool-pill[data-tool="Grep"],
.tool-pill[data-tool="Glob"]      { background: rgba(57,197,207,0.12);  color: #56d4dd; border-color: rgba(57,197,207,0.3); }
.tool-pill[data-tool="WebFetch"],
.tool-pill[data-tool="WebSearch"] { background: rgba(188,140,255,0.12); color: #d2a8ff; border-color: rgba(188,140,255,0.3); }
.tool-pill[data-tool="TaskCreate"],
.tool-pill[data-tool="TaskUpdate"],
.tool-pill[data-tool="TaskList"],
.tool-pill[data-tool="TaskGet"]   { background: rgba(255,123,114,0.15); color: #ff9b8b; border-color: rgba(255,123,114,0.35); }
.tool-pill[data-tool="NotebookEdit"] { background: rgba(255,200,130,0.15); color: #ffcf6b; border-color: rgba(255,200,130,0.35); }
.tool-pill[data-tool="ExitPlanMode"],
.tool-pill[data-tool="EnterPlanMode"] { background: rgba(140,200,255,0.15); color: #8cc8ff; border-color: rgba(140,200,255,0.35); }
.tool-summary {
  color: var(--text-dim);
  font-size: 12px;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tool-badge {
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 3px;
  font-family: var(--mono);
  flex-shrink: 0;
}
.tool-badge.error { background: rgba(248, 81, 73, 0.2); color: var(--red); }
.tool-badge.subagent { background: rgba(188, 140, 255, 0.15); color: var(--purple); }

.tool-body {
  display: none;
  padding: 12px;
  border-top: 1px solid var(--border);
}
.tool-call.expanded > .tool-body { display: block; }

.tool-body h4 {
  margin: 12px 0 6px;
  font-size: 11px;
  color: var(--text-dim);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  font-weight: 600;
}
.tool-body h4:first-child { margin-top: 0; }

.tool-body pre {
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 10px 12px;
  margin: 0;
  font-family: var(--mono);
  font-size: 11px;
  color: var(--text);
  max-height: 360px;
  overflow: auto;
  white-space: pre-wrap;
  word-wrap: break-word;
}

.subagent-stats {
  display: inline-block;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 6px 10px;
  font-family: var(--mono);
  font-size: 11px;
  color: var(--text-dim);
  margin-bottom: 8px;
}

.subagent-trajectory {
  margin-top: 12px;
  border-top: 1px solid var(--border);
  padding-top: 12px;
}

/* Filtered-out nodes */
.node.filtered-out { display: none; }
.node.tool-filter-hidden { display: none; }
.tool-call.tool-filter-hidden { display: none; }

/* ===== Tool filter chips ===== */
.tool-chip-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 10px;
}
.tool-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 12px;
  background: var(--bg-elev);
  border: 1px solid var(--border);
  color: var(--text-dim);
  font-family: var(--mono);
  font-size: 11px;
  cursor: pointer;
  user-select: none;
  transition: background 0.1s, color 0.1s;
}
.tool-chip:hover { background: var(--bg-hover); color: var(--text); }
.tool-chip.active {
  background: var(--accent-dim);
  color: var(--text);
  border-color: var(--accent);
}
.tool-chip .chip-count {
  opacity: 0.7;
  font-size: 10px;
}
.tool-chip.active .chip-count { opacity: 1; }

/* ===== Work Directory tab ===== */
.workdir-container {
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: 16px;
  min-height: 500px;
}
@media (max-width: 900px) {
  .workdir-container { grid-template-columns: 1fr; }
}
.workdir-tree {
  background: var(--bg-elev);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 12px;
  overflow: auto;
  max-height: 80vh;
}
.workdir-tree ul {
  list-style: none;
  padding-left: 16px;
  margin: 0;
}
.workdir-tree > ul {
  padding-left: 0;
}
.workdir-tree li {
  line-height: 1.8;
}
.workdir-tree .node-dir,
.workdir-tree .node-file {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 2px 6px;
  border-radius: 4px;
  cursor: pointer;
  user-select: none;
  font-family: var(--mono);
  font-size: 12px;
  color: var(--text);
}
.workdir-tree .node-dir:hover,
.workdir-tree .node-file:hover { background: var(--bg-hover); }
.workdir-tree .node-file.active {
  background: var(--accent-dim);
  color: var(--text);
}
.workdir-tree .file-icon,
.workdir-tree .dir-icon { font-size: 12px; width: 14px; flex-shrink: 0; }
.workdir-tree .file-size {
  margin-left: auto;
  color: var(--text-faint);
  font-size: 10px;
}
.workdir-tree li.collapsed > ul { display: none; }

.workdir-content {
  background: var(--bg-elev);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 20px;
  overflow: auto;
  max-height: 80vh;
}
.workdir-content .empty-state {
  color: var(--text-dim);
  text-align: center;
  padding: 80px 0;
}
.workdir-content h2 {
  margin: 0 0 6px;
  font-size: 14px;
  font-family: var(--mono);
  color: var(--text);
}
.workdir-content .file-meta {
  font-size: 11px;
  color: var(--text-faint);
  margin-bottom: 16px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--border);
}
.workdir-content pre {
  margin: 0;
  padding: 0;
  font-family: var(--mono);
  font-size: 12px;
  color: var(--text);
  white-space: pre-wrap;
  word-wrap: break-word;
  line-height: 1.5;
}

/* ===== Skill invocations list (Summary tab) ===== */
.count-pill {
  display: inline-block;
  background: var(--accent-dim);
  color: var(--text);
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 10px;
  margin-left: 8px;
  vertical-align: middle;
}
.skill-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.skill-invocation {
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 10px 12px;
}
.skill-row-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 4px;
}
.skill-name {
  font-family: var(--mono);
  font-weight: 600;
  color: var(--accent);
  font-size: 13px;
}
.skill-args {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--text-dim);
  background: var(--bg-elev);
  border-radius: 4px;
  padding: 6px 10px;
  white-space: pre-wrap;
  word-wrap: break-word;
  max-height: 140px;
  overflow: auto;
}
`;
