import { describe, it, expect } from 'vitest';
import { crossSignals } from '../../scripts/l3-fusion/signal-crossing.js';
import type { Group } from '../../scripts/l3-fusion/grouping.js';

const mkGroup = (sources: string[]): Group => ({
  id: 'g',
  category: 'layering',
  violations: sources.map((s, i) => ({
    id: `v${i}`, source: s, category: 'layering', ruleId: 'r', ruleSource: 'skill-builtin' as const,
    severity: 'medium' as const, locations: [{ file: 'a.py' }], evidence: {}, message: '',
  })),
});

describe('crossSignals', () => {
  it('high when ≥2 independent sources', () => {
    const c = crossSignals(mkGroup(['l1-tool:import-linter', 'l2-graph:co-change']));
    expect(c.baseConfidence).toBe('high');
    expect(c.independentSources).toEqual(['l1-tool', 'l2-graph']);
  });
  it('medium when 1 tool source only', () => {
    expect(crossSignals(mkGroup(['l1-tool:import-linter'])).baseConfidence).toBe('medium');
  });
  it('medium when only graph sources', () => {
    expect(crossSignals(mkGroup(['l2-graph:modularity'])).baseConfidence).toBe('medium');
  });
  it('low when only llm (placeholder — this case arises later)', () => {
    expect(crossSignals(mkGroup(['l3-llm:semantic'])).baseConfidence).toBe('low');
  });
});
