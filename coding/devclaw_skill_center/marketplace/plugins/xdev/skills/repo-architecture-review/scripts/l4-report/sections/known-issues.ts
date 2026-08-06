import type { Finding } from '../../types.js';

export function renderKnownIssues(unchanged: Finding[]): string {
  if (unchanged.length === 0) return '';
  const lines = [`## 已知问题（未变化，${unchanged.length} 条，已折叠）`, '', '<details><summary>展开</summary>', ''];
  for (const f of unchanged) lines.push(`- [${f.severity}] ${f.title} (${f.id})`);
  lines.push('', '</details>', '');
  return lines.join('\n');
}
