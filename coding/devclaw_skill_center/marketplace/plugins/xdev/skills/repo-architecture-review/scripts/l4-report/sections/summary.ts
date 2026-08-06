import type { AnalyzeResult } from '../../orchestrator.js';
import { computeHealthScore } from '../health-score.js';

function isLowSignal(title: string, severity: string, confidence: string): boolean {
  return title.startsWith('[LOW-SIGNAL]') || (severity === 'low' && confidence === 'low');
}

function healthEmoji(score: number): string {
  if (score >= 80) return '🟢';
  if (score >= 60) return '🟡';
  if (score >= 40) return '🟠';
  return '🔴';
}

export function renderSummary(r: AnalyzeResult): string {
  const main = r.reportedFindings.filter((f) => !isLowSignal(f.title, f.severity, f.confidence));
  const lowSignalCount = r.reportedFindings.length - main.length;
  const sevCounts = { critical: 0, high: 0, medium: 0, low: 0 } as Record<string, number>;
  for (const f of main) sevCounts[f.severity] = (sevCounts[f.severity] ?? 0) + 1;
  const confCounts = { high: 0, medium: 0, low: 0 } as Record<string, number>;
  for (const f of main) confCounts[f.confidence] = (confCounts[f.confidence] ?? 0) + 1;
  const health = computeHealthScore(r.reportedFindings);
  const date = r.runMeta.startedAt.slice(0, 10);
  const rootName = r.profile.rootPath.split('/').pop() ?? 'repo';
  const minutes = (r.runMeta.wallClockMs / 60_000).toFixed(1);
  const okCount = r.toolRuns.filter((t) => t.status === 'ok').length;
  const failedCount = r.toolRuns.filter((t) => t.status === 'failed').length;
  const totalLoc = r.profile.languages.reduce((sum, l) => sum + l.locTotal, 0);
  const locDisplay = totalLoc > 1_000_000
    ? `${(totalLoc / 1_000_000).toFixed(1)}M`
    : totalLoc > 1_000
      ? `${(totalLoc / 1_000).toFixed(1)}K`
      : String(totalLoc);
  const stack = r.runMeta.stackDetection.join(' + ');
  const convention = r.profile.layering?.conventionSummary ?? '未检测';

  const lines: string[] = [];

  // ── Title ──
  lines.push(`# ${rootName} 架构体检报告`);
  lines.push(`_${date} · ${stack} · ${r.profile.totalFiles} 文件 · ${locDisplay} LOC · 耗时 ${minutes} 分钟_`);
  lines.push('');

  // ── Dashboard ──
  lines.push('---');
  lines.push('');
  lines.push(`## ${healthEmoji(health)} 健康总评：${health}/100`);
  lines.push('');

  // Severity bar
  const sevBar = [
    sevCounts.critical > 0 ? `🔴 ${sevCounts.critical} 严重` : '',
    sevCounts.high > 0 ? `🟠 ${sevCounts.high} 高` : '',
    sevCounts.medium > 0 ? `🟡 ${sevCounts.medium} 中` : '',
    sevCounts.low > 0 ? `⚪️ ${sevCounts.low} 低` : '',
  ].filter(Boolean).join('　');
  lines.push(`**发现分布**：${sevBar || '无发现'}` + (lowSignalCount > 0 ? `　|　📋 ${lowSignalCount} 条低信号参考` : ''));
  lines.push('');

  // Key metrics row
  lines.push(`| 指标 | 值 |`);
  lines.push(`|---|---|`);
  lines.push(`| 置信度分布 | 高 ${confCounts.high ?? 0} · 中 ${confCounts.medium ?? 0} · 低 ${confCounts.low ?? 0} |`);
  lines.push(`| 工具覆盖 | ${okCount}/${r.toolRuns.length} 成功${failedCount > 0 ? ` (${failedCount} 失败 ⚠️)` : ''} |`);
  lines.push(`| 分层约定 | ${convention} |`);
  lines.push('');

  if (failedCount > 0) {
    const failedNames = r.toolRuns.filter((t) => t.status === 'failed').map((t) => t.toolId).join(', ');
    lines.push(`> ⚠️ **工具失败**：\`${failedNames}\`。这些维度的检测结果可能不完整，详见"执行审计"。`);
    lines.push('');
  }

  // One-line summary of the top problems (if any)
  if (main.length > 0) {
    const topFindings = main
      .filter((f) => f.severity === 'critical' || f.severity === 'high')
      .slice(0, 3);
    if (topFindings.length > 0) {
      lines.push('**最值得关注的问题**：');
      for (const f of topFindings) {
        const sev = f.severity === 'critical' ? '🔴' : '🟠';
        lines.push(`- ${sev} ${f.title}`);
      }
      lines.push('');
    }
  }

  lines.push('---');
  lines.push('');

  return lines.join('\n');
}
