import type { UnifiedGraph } from '../schema.js';
import type { Violation } from '../../types.js';

export function extractMermaidEdges(md: string): Array<[string, string]> {
  const edges: Array<[string, string]> = [];

  // Find all mermaid fenced blocks
  const mermaidRegex = /```mermaid\n([\s\S]*?)```/g;
  let match;

  while ((match = mermaidRegex.exec(md)) !== null) {
    const mermaidContent = match[1]!;

    // Extract arrows: A --> B
    const arrowRegex = /(\S+)\s*-->\s*(\S+)/g;
    let arrowMatch;

    while ((arrowMatch = arrowRegex.exec(mermaidContent)) !== null) {
      const [, from, to] = arrowMatch;
      if (from && to) {
        edges.push([from, to]);
      }
    }
  }

  return edges;
}

export function computeDocDrift(g: UnifiedGraph, docEdges: Array<[string, string]>): Violation[] {
  const violations: Violation[] = [];
  const files = new Set(g.files());

  for (const [from, to] of docEdges) {
    // Check if both nodes exist in graph
    const fromExists = files.has(from);
    const toExists = files.has(to);

    if (!fromExists || !toExists) {
      // Doc mentions non-existent node
      const ghostNode = fromExists ? to : from;
      violations.push({
        id: `doc-ghost-${ghostNode}`,
        source: 'l2-graph:doc-drift',
        category: 'doc-drift',
        ruleId: 'doc-ghost-node',
        ruleSource: 'skill-builtin',
        severity: 'medium',
        locations: [{ file: '<docs>' }],
        evidence: {
          signals: [`Non-existent node referenced in docs: ${ghostNode}`],
        },
        message: `Documentation references non-existent file or module: ${ghostNode}`,
      });
    } else {
      // Both exist: check if code import matches docs
      const codeImports = g.importsOf(from);
      if (!codeImports.includes(to)) {
        violations.push({
          id: `doc-missing-${from}-${to}`,
          source: 'l2-graph:doc-drift',
          category: 'doc-drift',
          ruleId: 'doc-missing-in-code',
          ruleSource: 'skill-builtin',
          severity: 'medium',
          locations: [{ file: from }],
          evidence: {
            signals: [`Documentation claims ${from} imports ${to}, but code doesn't show this import`],
          },
          message: `Code doesn't show documented import: ${from} → ${to}`,
        });
      }
    }
  }

  return violations;
}
