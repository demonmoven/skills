import { readFile } from 'node:fs/promises';
import { join, relative } from 'node:path';
import type { RepoProfile, Violation } from '../types.js';
import { buildUnifiedGraph } from './index.js';
import { detectBallOfMud } from './metrics/modularity.js';
import { computeCentrality } from './metrics/centrality.js';
import { computeAlvesThresholds } from './metrics/alves-percentile.js';
import { detectOrphans } from './metrics/orphan.js';
import { computeDirectoryGini } from './metrics/directory-gini.js';
import { extractMermaidEdges, computeDocDrift } from './metrics/doc-drift.js';
import { hashRuleSource } from '../l1-tools/adapter.js';
import { walkRepo } from '../util/fs.js';

// God-file thresholds — absolute floors. The actual thresholds are computed
// from the repo's file-size distribution (p95/p99) and never go below these.
const GOD_FILE_FLOOR_MED = 300;   // minimum for medium severity
const GOD_FILE_FLOOR_HIGH = 800;  // minimum for high severity

function isLikelyGenerated(rel: string): boolean {
  return /\.gen\.(go|py)$/.test(rel)
    || /(^|\/)mocks?_autogen\//.test(rel)
    || /(^|\/)kitex_gen\//.test(rel)
    || /(^|\/)hertz_gen\//.test(rel)
    || /_test\.(go|py)$/.test(rel);
}

function countMeaningfulLines(src: string, lang: 'go' | 'python'): number {
  const lineCommentRe = lang === 'python' ? /^\s*#/ : /^\s*\/\//;
  let n = 0;
  for (const line of src.split('\n')) {
    if (!line.trim()) continue;
    if (lineCommentRe.test(line)) continue;
    n++;
  }
  return n;
}

async function detectGodFiles(profile: RepoProfile): Promise<Violation[]> {
  const out: Violation[] = [];
  const all = await walkRepo(profile.rootPath);

  // First pass: collect all file sizes to compute adaptive thresholds.
  const fileLocs: Array<{ rel: string; abs: string; lang: 'go' | 'python'; loc: number }> = [];
  for (const abs of all) {
    const rel = relative(profile.rootPath, abs);
    let lang: 'go' | 'python' | null = null;
    if (rel.endsWith('.go')) lang = 'go';
    else if (rel.endsWith('.py')) lang = 'python';
    if (!lang) continue;
    if (isLikelyGenerated(rel)) continue;
    let src: string;
    try { src = await readFile(abs, 'utf-8'); } catch { continue; }
    const loc = countMeaningfulLines(src, lang);
    if (loc > 0) fileLocs.push({ rel, abs, lang, loc });
  }

  // Adaptive thresholds: use p90 for medium, p97 for high, with absolute floors.
  // This means in a repo where files are typically 50–200 LOC, a 400-line file
  // is already suspicious. In a repo where files are typically 200–800 LOC,
  // only 1000+ line files get flagged.
  const sorted = fileLocs.map((f) => f.loc).sort((a, b) => a - b);
  const p90 = sorted.length > 0 ? sorted[Math.ceil(sorted.length * 0.9) - 1] : GOD_FILE_FLOOR_MED;
  const p97 = sorted.length > 0 ? sorted[Math.ceil(sorted.length * 0.97) - 1] : GOD_FILE_FLOOR_HIGH;
  const godFileMed = Math.max(GOD_FILE_FLOOR_MED, p90);
  const godFileHigh = Math.max(GOD_FILE_FLOOR_HIGH, p97);

  for (const { rel, loc } of fileLocs) {
    if (loc < godFileMed) continue;
    const severity: Violation['severity'] = loc >= godFileHigh ? 'high' : 'medium';
    out.push({
      id: hashRuleSource('l2-graph', 'god-file', rel),
      source: 'l2-graph:loc',
      category: 'structure',
      ruleId: 'god-file',
      ruleSource: 'skill-builtin',
      severity,
      locations: [{ file: rel }],
      evidence: {
        metric: { name: 'meaningful-loc', value: loc, threshold: godFileMed, percentile: 90 },
        signals: [
          `threshold-medium=${godFileMed} (repo p90, floor=${GOD_FILE_FLOOR_MED})`,
          `threshold-high=${godFileHigh} (repo p97, floor=${GOD_FILE_FLOOR_HIGH})`,
        ],
      },
      message: `${rel} has ${loc} meaningful lines of code (adaptive thresholds: ≥${godFileMed} medium [p90], ≥${godFileHigh} high [p97])`,
    });
  }
  // Sort biggest first so the report leads with the worst offenders.
  out.sort((a, b) => (b.evidence.metric?.value ?? 0) - (a.evidence.metric?.value ?? 0));
  return out;
}

export async function computeL2Violations(profile: RepoProfile): Promise<Violation[]> {
  const g = await buildUnifiedGraph(profile);
  const out: Violation[] = [];

  out.push(...detectBallOfMud(g, 0.3));

  const centrality = computeCentrality(g);
  const prSamples = [...centrality.pagerank.entries()].map(([file, pr]) => ({
    value: pr,
    weight: (g.raw().getNodeAttribute(file, 'loc') as number | undefined) ?? 1,
  }));
  const prThresholds = computeAlvesThresholds(prSamples);
  // "God module" = top centrality *and* absolute PageRank above a floor. Using
  // p95 (instead of p90) trims the flat-distribution tail, and the absolute
  // floor (PR_FLOOR) prevents degenerate distributions where p95 itself is
  // near-zero from firing on every file. We also cap the number of emissions
  // per run to keep the report readable on large repos.
  const PR_FLOOR = 0.01;
  const MAX_GOD_MODULES = 15;
  const prThreshold = Math.max(prThresholds.p95, PR_FLOOR);
  const godCandidates = [...centrality.pagerank.entries()]
    .filter(([, pr]) => pr > prThreshold)
    .sort((a, b) => b[1] - a[1])
    .slice(0, MAX_GOD_MODULES);
  for (const [file, pr] of godCandidates) {
    out.push({
      id: hashRuleSource('l2-graph', 'god-module', file),
      source: 'l2-graph:centrality',
      category: 'coupling',
      ruleId: 'god-module',
      ruleSource: 'skill-builtin',
      severity: 'high',
      locations: [{ file }],
      evidence: { metric: { name: 'pagerank', value: pr, threshold: prThreshold, percentile: 95 } },
      message: `${file} has PageRank ${pr.toFixed(3)} above repo 95th percentile (${prThreshold.toFixed(3)})`,
    });
  }

  out.push(...detectOrphans(g));

  out.push(...await detectGodFiles(profile));

  const locByDir = new Map<string, number>();
  for (const f of g.files()) {
    const dir = f.includes('/') ? f.split('/').slice(0, 2).join('/') : '.';
    const loc = (g.raw().getNodeAttribute(f, 'loc') as number | undefined) ?? 0;
    locByDir.set(dir, (locByDir.get(dir) ?? 0) + loc);
  }
  const gini = computeDirectoryGini(locByDir);
  if (gini > 0.7) {
    out.push({
      id: hashRuleSource('l2-graph', 'directory-gini-high', '<repo>'),
      source: 'l2-graph:directory-gini',
      category: 'structure',
      ruleId: 'directory-gini-high',
      ruleSource: 'skill-builtin',
      severity: 'medium',
      locations: [{ file: '<repo>' }],
      evidence: { metric: { name: 'gini', value: Number(gini.toFixed(3)), threshold: 0.7 } },
      message: `Directory LOC Gini ${gini.toFixed(3)} > 0.7 — one or a few dirs dominate`,
    });
  }

  try {
    const files = await walkRepo(profile.rootPath);
    const docs = files.filter((f) => /\bdocs\/.*\.(md|mmd)$/.test(relative(profile.rootPath, f)));
    const docEdges: Array<[string, string]> = [];
    for (const d of docs) {
      const content = await readFile(d, 'utf-8');
      docEdges.push(...extractMermaidEdges(content));
    }
    if (docEdges.length > 0) {
      out.push(...computeDocDrift(g, docEdges));
    }
  } catch { /* ignore */ }

  return out;
}
