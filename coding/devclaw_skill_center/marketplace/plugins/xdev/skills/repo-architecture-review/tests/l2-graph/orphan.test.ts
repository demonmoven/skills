import { describe, it, expect } from 'vitest';
import { UnifiedGraph } from '../../scripts/l2-graph/schema.js';
import { detectOrphans } from '../../scripts/l2-graph/metrics/orphan.js';

describe('orphan detection', () => {
  it('detects unreachable files as orphans', () => {
    const g = new UnifiedGraph();
    // Entry points
    g.addFile('main.py', { lang: 'python', loc: 100 });
    g.addFile('__init__.py', { lang: 'python', loc: 10 });

    // Reachable from main
    g.addFile('util.py', { lang: 'python', loc: 50 });
    g.addImport('main.py', 'util.py');

    // Orphan: not reachable, not an entry point
    g.addFile('dead.py', { lang: 'python', loc: 30 });

    const violations = detectOrphans(g);

    expect(violations.some((v) => v.locations.some((loc) => loc.file.endsWith('dead.py')))).toBe(true);
    const deadViolation = violations.find((v) => v.locations.some((loc) => loc.file.endsWith('dead.py')));
    expect(deadViolation?.category).toBe('dead-code');
    expect(deadViolation?.ruleId).toBe('orphan-file');
    expect(deadViolation?.severity).toBe('low');
  });

  it('does not flag entry points as orphans', () => {
    const g = new UnifiedGraph();
    g.addFile('main.py', { lang: 'python', loc: 100 });
    g.addFile('conftest.py', { lang: 'python', loc: 50 });

    const violations = detectOrphans(g);

    expect(violations.some((v) => v.locations.some((loc) => loc.file.endsWith('main.py')))).toBe(false);
    expect(violations.some((v) => v.locations.some((loc) => loc.file.endsWith('conftest.py')))).toBe(false);
  });
});
