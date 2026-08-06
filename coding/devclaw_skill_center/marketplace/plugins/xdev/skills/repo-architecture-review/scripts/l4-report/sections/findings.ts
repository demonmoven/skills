import type { Finding } from '../../types.js';

const PASSTHROUGH_PREFIX = '[LOW-SIGNAL]';

const SEV_EMOJI: Record<Finding['severity'], string> = {
  critical: '🔴',
  high: '🟠',
  medium: '🟡',
  low: '⚪️',
};

const SEV_ZH: Record<Finding['severity'], string> = {
  critical: '严重',
  high: '高',
  medium: '中',
  low: '低',
};

const CONF_ZH: Record<Finding['confidence'], string> = {
  high: '高',
  medium: '中',
  low: '低',
};

const CAT_ZH: Record<string, string> = {
  coupling: '耦合',
  layering: '分层',
  boundary: '边界',
  structure: '结构',
  dependency: '依赖',
  'dead-code': '死代码',
  'doc-drift': '文档/代码漂移',
  'test-org': '测试组织',
  'data-layer': '数据层',
  'security-boundary': '安全边界',
  config: '配置',
  'build-deploy': '构建/部署',
};

function isLowSignal(f: Finding): boolean {
  return f.title.startsWith(PASSTHROUGH_PREFIX);
}

function renderEvidence(f: Finding): string {
  const lines: string[] = [];
  lines.push('');
  lines.push('<details>');
  lines.push('<summary>📎 证据与涉及位置</summary>');
  lines.push('');

  // Dedup signals
  const seen = new Set<string>();
  const sigs: string[] = [];
  for (const s of f.signals) {
    const key = `${s.source}|${s.description.slice(0, 80)}`;
    if (seen.has(key)) continue;
    seen.add(key);
    sigs.push(`- \`${s.source}\` — ${s.description}`);
  }
  const SIG_CAP = 8;
  if (sigs.length > 0) {
    lines.push('**信号来源**');
    lines.push('');
    if (sigs.length <= SIG_CAP) lines.push(...sigs);
    else { lines.push(...sigs.slice(0, SIG_CAP), `- _…还有 ${sigs.length - SIG_CAP} 条同类信号_`); }
    lines.push('');
  }

  if (f.locations.length > 0) {
    lines.push('**涉及位置**');
    lines.push('');
    const LOC_CAP = 12;
    const locStrs = f.locations.map((l) => {
      const sym = l.symbol ? ` · \`${l.symbol}\`` : '';
      const ln = l.line ? `:${l.line}` : '';
      return `- \`${l.file}${ln}\`${sym}`;
    });
    if (locStrs.length <= LOC_CAP) lines.push(...locStrs);
    else { lines.push(...locStrs.slice(0, LOC_CAP), `- _…还有 ${locStrs.length - LOC_CAP} 个位置_`); }
    lines.push('');
  }

  lines.push('</details>');
  return lines.join('\n');
}

function renderOneFinding(f: Finding, i: number): string {
  const cat = CAT_ZH[f.category] ?? f.category;
  const sevTag = `${SEV_EMOJI[f.severity]} ${SEV_ZH[f.severity]}`;
  const confTag = `置信度 ${CONF_ZH[f.confidence]}`;
  const parts: string[] = [];

  // Card header: number + title + tags on the same visual block
  parts.push(`### ${i}. ${f.title}`);
  parts.push('');
  parts.push(`\`${sevTag}\` \`${cat}\` \`${confTag} (${f.confidenceScore.toFixed(2)})\``);
  parts.push('');

  // Root cause — the main narrative
  parts.push(f.rootCause.trim());

  // Impact — visually distinct
  if (f.impact && f.impact.trim()) {
    parts.push('');
    parts.push(`> **影响**：${f.impact.trim()}`);
  }

  // Actions — numbered list with clear header
  if (f.actions.length > 0) {
    parts.push('');
    parts.push('**建议动作**');
    parts.push('');
    f.actions.forEach((a, idx) => parts.push(`${idx + 1}. ${a}`));
  }

  // Evidence — collapsed by default to reduce visual noise
  parts.push(renderEvidence(f));
  parts.push('');
  return parts.join('\n');
}

export function renderFindings(findings: Finding[]): string {
  const real = findings.filter((f) => !isLowSignal(f));
  const passthrough = findings.filter(isLowSignal);

  const out: string[] = [];
  out.push('## 关键发现', '');

  if (real.length === 0) {
    out.push('_本次未发现高/中置信度的架构问题。下方"低信号参考"区域有原始信号供人工核查。_', '');
  } else {
    // Group by severity for visual scanning
    const highSev = real.filter((f) => f.severity === 'critical' || f.severity === 'high');
    const medSev = real.filter((f) => f.severity === 'medium');
    const lowSev = real.filter((f) => f.severity === 'low');

    let idx = 1;
    if (highSev.length > 0) {
      out.push(`### 🟠 高严重度（${highSev.length} 条）`, '');
      for (const f of highSev) {
        out.push(renderOneFinding(f, idx++));
      }
    }
    if (medSev.length > 0) {
      out.push(`### 🟡 中严重度（${medSev.length} 条）`, '');
      for (const f of medSev) {
        out.push(renderOneFinding(f, idx++));
      }
    }
    if (lowSev.length > 0) {
      out.push(`### ⚪️ 低严重度（${lowSev.length} 条）`, '');
      for (const f of lowSev) {
        out.push(renderOneFinding(f, idx++));
      }
    }
  }

  // Low-signal section: collapsed by category for minimal visual weight
  if (passthrough.length > 0) {
    out.push('---', '');
    out.push('## 低信号参考', '');
    out.push(`以下 ${passthrough.length} 条由工具触发但置信度较低，保留供人工判定。`);
    out.push('');

    // Group by category
    const byCat = new Map<string, Finding[]>();
    for (const f of passthrough) {
      const cat = CAT_ZH[f.category] ?? f.category;
      if (!byCat.has(cat)) byCat.set(cat, []);
      byCat.get(cat)!.push(f);
    }

    for (const [cat, items] of byCat) {
      out.push('<details>');
      out.push(`<summary>${cat}（${items.length} 条）</summary>`);
      out.push('');
      for (const f of items) {
        const loc = f.locations[0]?.file ?? '<repo>';
        const stripped = f.title.replace(/^\[LOW-SIGNAL\]\s*/, '');
        out.push(`- \`${loc}\` — ${stripped}`);
      }
      out.push('');
      out.push('</details>');
      out.push('');
    }
  }

  return out.join('\n');
}
