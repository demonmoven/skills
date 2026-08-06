import { describe, it, expect } from 'vitest';
import { UnifiedGraph } from '../../scripts/l2-graph/schema.js';
import { computeCentrality } from '../../scripts/l2-graph/metrics/centrality.js';

describe('centrality', () => {
  it('computes PageRank + betweenness centrality on import subgraph', () => {
    const g = new UnifiedGraph();
    // Hub-and-spoke: hub.py imports from others, others import hub.py
    g.addFile('hub.py', { lang: 'python', loc: 100 });
    g.addFile('a.py', { lang: 'python', loc: 50 });
    g.addFile('b.py', { lang: 'python', loc: 50 });
    g.addFile('c.py', { lang: 'python', loc: 50 });
    g.addFile('d.py', { lang: 'python', loc: 50 });

    g.addImport('a.py', 'hub.py');
    g.addImport('b.py', 'hub.py');
    g.addImport('c.py', 'hub.py');
    g.addImport('d.py', 'hub.py');
    g.addImport('hub.py', 'a.py');

    const { pagerank, betweenness } = computeCentrality(g);

    // Hub should have higher PageRank than peripheral nodes
    expect(pagerank.get('hub.py')).toBeGreaterThan(pagerank.get('a.py')!);
    expect(pagerank.get('hub.py')).toBeGreaterThan(pagerank.get('b.py')!);
    expect(pagerank.get('hub.py')).toBeGreaterThan(pagerank.get('c.py')!);

    // Hub should have high betweenness (mediates many paths)
    expect(betweenness.get('hub.py')).toBeGreaterThan(0);
    expect(pagerank.size).toBe(5);
    expect(betweenness.size).toBe(5);
  });
});
