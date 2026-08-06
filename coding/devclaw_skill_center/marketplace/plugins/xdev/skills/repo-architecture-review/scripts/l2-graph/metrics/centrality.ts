import { createRequire } from 'node:module';
import type { UnifiedGraph } from '../schema.js';

const require = createRequire(import.meta.url);
const PageRankFunc = require('graphology-metrics/centrality/pagerank');
const BetweennessFunc = require('graphology-metrics/centrality/betweenness');

export interface CentralityResult {
  pagerank: Map<string, number>;
  betweenness: Map<string, number>;
}

export function computeCentrality(g: UnifiedGraph): CentralityResult {
  const rawGraph = g.raw();

  // Build file + imports subgraph (undirected)
  const subgraph = new (rawGraph.constructor)({ type: 'undirected' });
  const files = g.files();

  // Add file nodes
  files.forEach((file) => {
    const attrs = rawGraph.getNodeAttributes(file);
    subgraph.addNode(file, attrs);
  });

  // Add import edges (undirected)
  files.forEach((file) => {
    g.importsOf(file).forEach((target: string) => {
      if (!subgraph.hasEdge(file, target)) {
        subgraph.addUndirectedEdge(file, target);
      }
    });
  });

  // Compute PageRank — bump iteration cap and loosen tolerance so graphs with
  // disconnected components or self-loop-heavy nodes still converge. On repeat
  // failure, fall back to normalized degree centrality (deterministic, always
  // defined, still a meaningful "how central is this file" signal). Returning
  // an empty Map would silently drop the god-module rule.
  let pagerankResult: Record<string, number>;
  try {
    pagerankResult = PageRankFunc(subgraph, { maxIterations: 1000, tolerance: 1e-3 });
  } catch {
    try {
      pagerankResult = PageRankFunc(subgraph, { maxIterations: 2000, tolerance: 1e-2 });
    } catch (e) {
      console.warn(`[centrality] pagerank failed to converge; falling back to normalized degree centrality: ${(e as Error)?.message ?? e}`);
      pagerankResult = {};
      const nodes: string[] = subgraph.nodes();
      let totalDeg = 0;
      const degs: Record<string, number> = {};
      for (const n of nodes) {
        const d = subgraph.degree(n);
        degs[n] = d;
        totalDeg += d;
      }
      if (totalDeg > 0) {
        for (const n of nodes) pagerankResult[n] = degs[n] / totalDeg;
      } else {
        for (const n of nodes) pagerankResult[n] = 0;
      }
    }
  }
  const pagerankMap = new Map(Object.entries(pagerankResult));

  // Compute Betweenness
  const betweennessResult: Record<string, number> = BetweennessFunc(subgraph);
  const betweennessMap = new Map(Object.entries(betweennessResult));

  return {
    pagerank: pagerankMap,
    betweenness: betweennessMap,
  };
}
