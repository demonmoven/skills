import { createRequire } from 'node:module';
import type { UnifiedGraph } from '../schema.js';
import type { Violation } from '../../types.js';

const require = createRequire(import.meta.url);
const louvainModule = require('graphology-communities-louvain');

export function computeModularity(g: UnifiedGraph): number {
  const rawGraph = g.raw();

  // Build file-only import undirected subgraph
  const fileOnlyGraph = new (rawGraph.constructor)({ type: 'undirected' });
  const files = g.files();

  // Add all file nodes
  files.forEach((file) => {
    const attrs = rawGraph.getNodeAttributes(file);
    fileOnlyGraph.addNode(file, attrs);
  });

  // Add import edges (undirected)
  files.forEach((file) => {
    g.importsOf(file).forEach((target: string) => {
      if (!fileOnlyGraph.hasEdge(file, target)) {
        fileOnlyGraph.addUndirectedEdge(file, target);
      }
    });
  });

  // Run Louvain and get modularity
  const result = louvainModule.detailed(fileOnlyGraph);
  return result.modularity;
}

export function detectBallOfMud(g: UnifiedGraph, threshold = 0.3): Violation[] {
  const q = computeModularity(g);

  if (q >= threshold) {
    return [];
  }

  const severity = q < 0.15 ? 'high' : 'medium';
  return [{
    id: `ball-of-mud-${q.toFixed(3)}`,
    source: 'l2-graph:modularity',
    category: 'structure',
    ruleId: 'modularity-low',
    ruleSource: 'skill-builtin',
    severity,
    locations: [{ file: '<repo>' }],
    evidence: {
      metric: {
        name: 'modularity-Q',
        value: Number(q.toFixed(3)),
        threshold,
      },
    },
    message: `Low modularity Q=${q.toFixed(3)} indicates poor separation of concerns (ball-of-mud pattern)`,
  }];
}
