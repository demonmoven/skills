import type { Finding } from '../../types.js';

const SEV_EMOJI: Record<string, string> = {
  critical: '🔴',
  high: '🟠',
  medium: '🟡',
  low: '⚪️',
};

/**
 * Renders an action roadmap: a prioritized table of the top findings with
 * their first suggested action. This gives the reader a quick "what should
 * I do first?" view without having to read every finding in detail.
 */
export function renderActionRoadmap(findings: Finding[]): string {
  // Only include non-low-signal findings.
  const real = findings.filter((f) => !f.title.startsWith('[LOW-SIGNAL]'));
  if (real.length === 0) return '';

  // Assign priority: P0 for critical/high-severity + high-confidence,
  // P1 for high-severity or high-confidence, P2 for the rest.
  const rows: Array<{ priority: string; sev: string; title: string; action: string }> = [];
  for (const f of real) {
    let priority: string;
    if ((f.severity === 'critical' || f.severity === 'high') && f.confidence === 'high') {
      priority = 'P0';
    } else if (f.severity === 'critical' || f.severity === 'high' || f.confidence === 'high') {
      priority = 'P1';
    } else {
      priority = 'P2';
    }
    const firstAction = f.actions[0] ?? '审阅并评估';
    const shortTitle = f.title.length > 60 ? f.title.slice(0, 57) + '...' : f.title;
    rows.push({ priority, sev: SEV_EMOJI[f.severity] ?? '⚪️', title: shortTitle, action: firstAction });
  }

  // Sort by priority
  const order = { P0: 0, P1: 1, P2: 2 };
  rows.sort((a, b) => (order[a.priority as keyof typeof order] ?? 9) - (order[b.priority as keyof typeof order] ?? 9));

  const lines: string[] = [];
  lines.push('## 行动路线图');
  lines.push('');
  lines.push('| 优先级 | 严重度 | 问题 | 建议首要动作 |');
  lines.push('|:---:|:---:|---|---|');
  for (const r of rows.slice(0, 15)) {
    lines.push(`| **${r.priority}** | ${r.sev} | ${r.title} | ${r.action} |`);
  }
  if (rows.length > 15) {
    lines.push(`| | | _…还有 ${rows.length - 15} 条_ | |`);
  }
  lines.push('');
  return lines.join('\n');
}
