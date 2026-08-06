import { describe, it, expect } from 'vitest';
import { UnifiedGraph } from '../../scripts/l2-graph/schema.js';

describe('UnifiedGraph', () => {
  it('adds nodes and edges and queries correctly', () => {
    const g = new UnifiedGraph();
    g.addFile('a.py', { lang: 'python', loc: 50 });
    g.addFile('b.py', { lang: 'python', loc: 80 });
    g.addImport('a.py', 'b.py');
    expect(g.files()).toEqual(['a.py', 'b.py']);
    expect(g.importsOf('a.py')).toEqual(['b.py']);
    expect(g.importersOf('b.py')).toEqual(['a.py']);
  });

  it('supports symbols attached to files', () => {
    const g = new UnifiedGraph();
    g.addFile('a.py', { lang: 'python', loc: 10 });
    g.addSymbol('a.py:load', { file: 'a.py', line: 2, kind: 'function', name: 'load' });
    expect(g.symbolsOf('a.py').map((s) => s.name)).toEqual(['load']);
  });

  it('supports co-change edges with weights', () => {
    const g = new UnifiedGraph();
    g.addFile('a.py', { lang: 'python', loc: 10 });
    g.addFile('b.py', { lang: 'python', loc: 10 });
    g.addCoChange('a.py', 'b.py', 42);
    expect(g.coChangeWeight('a.py', 'b.py')).toBe(42);
    expect(g.coChangeWeight('b.py', 'a.py')).toBe(42);
  });
});
