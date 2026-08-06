import type { UnifiedGraph } from '../schema.js';
import type { Violation } from '../../types.js';

export interface OrphanOpts {
  entryPatterns?: RegExp[];
}

export function detectOrphans(g: UnifiedGraph, opts?: OrphanOpts): Violation[] {
  const entryPatterns = opts?.entryPatterns ?? [
    /main\.py$/,
    /__init__\.py$/,
    /main\.go$/,
    /cmd\/[^/]+\/main\.go$/,
    /cmd\/.*\.go$/,           // cmd/ dir often holds CLI entry scripts (e.g. cmd/generate.go)
    /go_script\//,            // standalone Go scripts (each main package is an entry)
    /handler\.go$/,           // framework-generated router/handler entry
    /conftest\.py$/,
    /.*_test\.go$/,
    /test_.*\.go$/,
    /test_.*\.py$/,
  ];

  // Max-coverage mode: no exclusion list. Emit every file that is not reachable
  // from an entry point, including generated code / mocks / pure-config packages.
  const files = g.files();
  const isEntry = (file: string): boolean => entryPatterns.some((p) => p.test(file));
  const isExcluded = (_file: string): boolean => false;

  // BFS to find all reachable files from entry points
  const reachable = new Set<string>();
  const queue: string[] = [];

  for (const file of files) {
    if (isEntry(file)) {
      queue.push(file);
      reachable.add(file);
    }
  }

  while (queue.length > 0) {
    const file = queue.shift()!;
    for (const target of g.importsOf(file)) {
      if (!reachable.has(target)) {
        reachable.add(target);
        queue.push(target);
      }
    }
  }

  // Detect orphans (files not reachable, not entry points, not excluded).
  // Collect corroborating evidence so downstream confidence scoring can
  // distinguish strong dead-code (tiny/empty/generated) from a weak "happens
  // to be unreachable" signal that needs a human check.
  const violations: Violation[] = [];
  for (const file of files) {
    if (reachable.has(file) || isEntry(file) || isExcluded(file)) continue;
    const loc = (g.raw().getNodeAttribute(file, 'loc') as number | undefined) ?? 0;
    const signals: string[] = ['orphan:unreachable'];
    if (loc <= 5) signals.push(`orphan:near-empty(loc=${loc})`);
    else if (loc <= 30) signals.push(`orphan:tiny(loc=${loc})`);
    if (/\.(gen|pb)\.(go|py|ts)$/.test(file) || /\/(query_gen|generated|gen)\//.test(file)) {
      signals.push('orphan:generated-code');
    }
    // Upgrade to medium severity when we have strong corroborating evidence:
    // tiny generated file or near-empty file — these are almost always dead.
    const strong = signals.length >= 3 || signals.some((s) => s.startsWith('orphan:near-empty'));
    violations.push({
      id: `orphan-${file}`,
      source: 'l2-graph:orphan',
      category: 'dead-code',
      ruleId: 'orphan-file',
      ruleSource: 'skill-builtin',
      severity: strong ? 'medium' : 'low',
      locations: [{ file }],
      evidence: { metric: { name: 'loc', value: loc, threshold: 30 }, signals },
      message: `Orphan file (loc=${loc}): ${signals.join(', ')}`,
    });
  }

  return violations;
}
