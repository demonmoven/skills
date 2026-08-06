import { describe, it, expect } from 'vitest';
import { UnifiedGraph } from '../../scripts/l2-graph/schema.js';
import { computeModularity, detectBallOfMud } from '../../scripts/l2-graph/metrics/modularity.js';

describe('modularity', () => {
  it('computes modularity > 0.3 for distinct clusters', () => {
    const g = new UnifiedGraph();
    // Cluster 1: a, b, c
    g.addFile('a.py', { lang: 'python', loc: 100 });
    g.addFile('b.py', { lang: 'python', loc: 100 });
    g.addFile('c.py', { lang: 'python', loc: 100 });
    g.addImport('a.py', 'b.py');
    g.addImport('b.py', 'c.py');
    g.addImport('a.py', 'c.py');

    // Cluster 2: x, y, z
    g.addFile('x.py', { lang: 'python', loc: 100 });
    g.addFile('y.py', { lang: 'python', loc: 100 });
    g.addFile('z.py', { lang: 'python', loc: 100 });
    g.addImport('x.py', 'y.py');
    g.addImport('y.py', 'z.py');
    g.addImport('x.py', 'z.py');

    const q = computeModularity(g);
    expect(q).toBeGreaterThan(0.3);
  });

  it('detects ball-of-mud (q < 0.3) as single violation', () => {
    const g = new UnifiedGraph();
    // Fully connected graph (poor modularity)
    const files = ['a.py', 'b.py', 'c.py', 'd.py', 'e.py'];
    files.forEach((f) => g.addFile(f, { lang: 'python', loc: 100 }));
    files.forEach((f1) => {
      files.forEach((f2) => {
        if (f1 !== f2) g.addImport(f1, f2);
      });
    });

    const violations = detectBallOfMud(g, 0.3);
    expect(violations).toHaveLength(1);
    const v = violations[0]!;
    expect(v.category).toBe('structure');
    expect(v.ruleId).toBe('modularity-low');
    expect(v.severity).toBe('high');
    expect(v.evidence.metric?.name).toBe('modularity-Q');
    expect(v.ruleSource).toBe('skill-builtin');
    expect(v.source).toBe('l2-graph:modularity');
  });
});
