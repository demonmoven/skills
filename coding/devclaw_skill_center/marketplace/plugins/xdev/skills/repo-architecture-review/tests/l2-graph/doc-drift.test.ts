import { describe, it, expect } from 'vitest';
import { UnifiedGraph } from '../../scripts/l2-graph/schema.js';
import { extractMermaidEdges, computeDocDrift } from '../../scripts/l2-graph/metrics/doc-drift.js';

describe('doc-drift', () => {
  it('extracts edges from mermaid fenced blocks', () => {
    const md = `
# Architecture

\`\`\`mermaid
graph TD
  A --> B
  B --> C
  C --> A
\`\`\`

Some text

\`\`\`mermaid
X --> Y
\`\`\`
`;

    const edges = extractMermaidEdges(md);
    expect(edges).toContainEqual(['A', 'B']);
    expect(edges).toContainEqual(['B', 'C']);
    expect(edges).toContainEqual(['C', 'A']);
    expect(edges).toContainEqual(['X', 'Y']);
    expect(edges).toHaveLength(4);
  });

  it('detects doc-ghost-node violations', () => {
    const g = new UnifiedGraph();
    g.addFile('a.py', { lang: 'python', loc: 100 });
    g.addFile('b.py', { lang: 'python', loc: 100 });

    // Documentation mentions a non-existent file
    const docEdges: [string, string][] = [['a.py', 'nonexistent.py']];

    const violations = computeDocDrift(g, docEdges);

    const ghostViolations = violations.filter((v) => v.ruleId === 'doc-ghost-node');
    expect(ghostViolations.length).toBeGreaterThan(0);
    expect(ghostViolations[0]?.category).toBe('doc-drift');
  });

  it('detects doc-missing-in-code violations', () => {
    const g = new UnifiedGraph();
    g.addFile('a.py', { lang: 'python', loc: 100 });
    g.addFile('b.py', { lang: 'python', loc: 100 });
    g.addFile('c.py', { lang: 'python', loc: 100 });

    // a imports b, but documentation says a imports c
    g.addImport('a.py', 'b.py');

    const docEdges: [string, string][] = [['a.py', 'c.py']];

    const violations = computeDocDrift(g, docEdges);

    const missingViolations = violations.filter((v) => v.ruleId === 'doc-missing-in-code');
    expect(missingViolations.length).toBeGreaterThan(0);
    expect(missingViolations[0]?.category).toBe('doc-drift');
  });
});
