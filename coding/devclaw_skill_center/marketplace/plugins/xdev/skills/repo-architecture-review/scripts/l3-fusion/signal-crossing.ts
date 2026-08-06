import type { Group } from './grouping.js';

export interface Crossed {
  group: Group;
  independentSources: string[];       // e.g. ['l1-tool', 'l2-graph']
  baseConfidence: 'high' | 'medium' | 'low';
}

// Count corroborating evidence signals carried on individual violations.
// A single source family can still be high-confidence when it attaches
// multiple independent signals (e.g. orphan + near-empty + generated-code).
function evidenceSignalCount(g: Group): number {
  const all = new Set<string>();
  for (const v of g.violations) {
    for (const s of v.evidence.signals ?? []) all.add(s);
  }
  return all.size;
}

export function crossSignals(g: Group): Crossed {
  const families = new Set<string>();
  for (const v of g.violations) families.add(v.source.split(':')[0]);
  const list = [...families].sort();
  const signals = evidenceSignalCount(g);
  const hasToolFamily = list.includes('l1-tool') || list.includes('l2-graph');

  let bc: Crossed['baseConfidence'];
  if (list.length >= 2 && hasToolFamily) {
    // Multi-family with at least one deterministic tool source: strongest case.
    bc = 'high';
  } else if (hasToolFamily && signals >= 3) {
    // Single deterministic family but rich evidence (e.g. orphan + tiny +
    // generated-code) — still worth surfacing as high-confidence because the
    // evidence is independent within the family.
    bc = 'high';
  } else if (hasToolFamily) {
    // Single tool family, thin evidence — medium. Dead-code orphans land here.
    bc = 'medium';
  } else if (list.length === 1 && list[0] !== 'l3-llm') {
    bc = 'medium';
  } else {
    bc = 'low';
  }
  return { group: g, independentSources: list, baseConfidence: bc };
}
