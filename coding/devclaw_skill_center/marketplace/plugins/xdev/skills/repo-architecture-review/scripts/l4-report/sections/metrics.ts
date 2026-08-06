import type { AnalyzeResult } from '../../orchestrator.js';

export function renderMetrics(r: AnalyzeResult): string {
  const lines: string[] = [];
  lines.push('## 全景指标');
  lines.push('');

  // Language breakdown
  if (r.profile.languages.length > 0) {
    lines.push('| 语言 | 文件数 | LOC |');
    lines.push('|---|---:|---:|');
    for (const l of r.profile.languages) {
      lines.push(`| ${l.lang} | ${l.fileCount.toLocaleString()} | ${l.locTotal.toLocaleString()} |`);
    }
    lines.push('');
  }

  // Infrastructure
  const infra: string[] = [];
  if (r.profile.dockerfiles.length > 0) infra.push(`Dockerfile × ${r.profile.dockerfiles.length}`);
  if (r.profile.ghaWorkflows.length > 0) infra.push(`GHA workflow × ${r.profile.ghaWorkflows.length}`);
  if (infra.length > 0) {
    lines.push(`**基础设施**：${infra.join('、')}`);
    lines.push('');
  }

  // Layering convention
  if (r.profile.layering) {
    const lc = r.profile.layering;
    lines.push(`**分层约定**：${lc.hasLayering ? lc.conventionSummary : '未检测到明确分层'}`);
    if (lc.reasoning && lc.reasoning.length > 0 && lc.reasoning.length < 200) {
      lines.push(`_${lc.reasoning}_`);
    }
    lines.push('');
  }

  // Adaptive thresholds used (if available from findings)
  const thresholdSignals: string[] = [];
  for (const f of r.reportedFindings) {
    for (const s of f.signals) {
      if (s.description.includes('threshold=') || s.description.includes('fan-in-threshold=')) {
        const match = s.description.match(/(?:threshold|fan-in-threshold)=(\d+)/);
        if (match) {
          const key = s.description.split('(')[1]?.replace(')', '') ?? '';
          thresholdSignals.push(`${s.source}: ${match[0]} ${key}`);
        }
      }
    }
  }
  if (thresholdSignals.length > 0) {
    lines.push('<details>');
    lines.push('<summary>本次使用的自适应阈值</summary>');
    lines.push('');
    const seen = new Set<string>();
    for (const t of thresholdSignals) {
      if (!seen.has(t)) { lines.push(`- ${t}`); seen.add(t); }
    }
    lines.push('');
    lines.push('</details>');
    lines.push('');
  }

  return lines.join('\n');
}
