import type { AnalyzeResult } from '../../orchestrator.js';

export function renderAudit(r: AnalyzeResult): string {
  const lines: string[] = [];
  lines.push('## 执行审计');
  lines.push('');

  const failed = r.toolRuns.filter((t) => t.status === 'failed');
  if (failed.length > 0) {
    lines.push(`> ⚠️ **${failed.length} 个工具执行失败**：其违规为零并不代表仓库没问题，而是该维度未被检测。`);
    lines.push('');
  }

  // Tool status table
  lines.push('| 工具 | 状态 | 耗时 | 说明 |');
  lines.push('|---|:---:|---:|---|');
  for (const t of r.toolRuns) {
    const seconds = (t.durationMs / 1000).toFixed(1) + 's';
    let status: string;
    let note: string;
    if (t.status === 'ok') {
      status = '✅';
      note = `${t.violations.length} 条违规`;
      if (t.generatedConfigPath) note += ` · 生成 ${t.generatedConfigPath}`;
    } else if (t.status === 'skipped') {
      status = '⏭️';
      note = t.reason ?? 'skipped';
    } else {
      status = '❌';
      const errSummary = (t.error ?? '').split('\n').find((l) => l.trim()) ?? 'unknown';
      note = errSummary.slice(0, 100);
    }
    lines.push(`| ${t.toolId} | ${status} | ${seconds} | ${note} |`);
  }
  lines.push('');

  // Meta
  if (r.runMeta.coverageDowngrades.length > 0) {
    lines.push(`**覆盖降级**：${r.runMeta.coverageDowngrades.join('；')}`);
    lines.push('');
  }

  lines.push(`_总耗时 ${(r.runMeta.wallClockMs / 60_000).toFixed(1)} 分钟 · map-reduce: ${r.runMeta.mapReduceUsed ? '是' : '否'}_`);
  lines.push('');

  return lines.join('\n');
}
