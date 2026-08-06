// scripts/l1-tools/adapters/go-arch.ts
//
// Pure-TypeScript Go architecture adapter. Does NOT depend on arch-go (which
// crashes on v1.7.0 when globbing `**/` patterns). Uses `go list` for ground-
// truth import graph and derives 5 independent signal families:
//
//   1. layering violations (inner layer importing outer layer)
//   2. package-level cycles (Tarjan SCC > 1)
//   3. horizontal coupling across business modules (siblings under service/)
//   4. God packages (high fan-in *and* high instability)
//   5. cross-package code clones (same-name functions, large identical runs)
//
// All five produce concrete metric-based violations; the metrics themselves
// are the evidence, so L3 fusion does not need LLM judgement for them — they
// bypass LLM folding via the `metricHardEvidence` flag on Violation.evidence.
//
// KEY DESIGN CHANGE (v2): Layering rules and thresholds are no longer
// hardcoded. The layering convention is discovered by LLM during L0 pre-scan
// (see layer-discovery.ts) and stored in RepoProfile.layering. Thresholds
// for fan-in and code clones are computed from the repo's own statistical
// distribution (Alves percentile method) so the tool adapts to repos of
// different sizes and styles.

import { readFile } from 'node:fs/promises';
import { dirname, relative, join } from 'node:path';
import { runCmd } from '../../util/exec.js';
import { walkRepo } from '../../util/fs.js';
import { hashRuleSource, type ToolAdapter, type AdapterContext, type PreflightResult, type RawRunOutput } from '../adapter.js';
import type { Violation, LayeringConvention } from '../../types.js';
import { FALLBACK_CONVENTION } from '../../l0-prescan/layer-discovery.js';

// ── Threshold defaults (used as floors; actual thresholds are computed from
// the repo's distribution when possible) ─────────────────────────────────────
const GOD_INSTABILITY = 0.5;            // fan_out / (fan_in + fan_out)
const HORIZONTAL_MIN_CROSSINGS = 1;     // any sibling-to-sibling import is a smell
const CLONE_MIN_IDENTICAL_LINES = 50;   // absolute floor for clone detection
const CLONE_MIN_RATIO = 0.5;            // overlap as fraction of smaller file

// Adaptive threshold floors — we never go below these even if the repo's
// distribution is very low, to avoid trivial noise.
const FAN_IN_FLOOR = 5;
const CLONE_LINES_FLOOR = 50;

interface GoListEntry {
  ImportPath: string;
  Dir?: string;
  Imports?: string[];
  Module?: { Main?: boolean };
}

interface PackageGraph {
  modulePath: string;
  packages: string[];                  // internal packages only
  imports: Map<string, Set<string>>;   // pkg -> deps (internal only)
  reverse: Map<string, Set<string>>;   // pkg -> dependents
  pkgDir: Map<string, string>;         // pkg -> absolute dir
}

async function buildGraph(rootPath: string, modulePath: string, timeoutMs: number): Promise<PackageGraph | null> {
  const r = await runCmd('go', ['list', '-deps', '-json', './...'], { cwd: rootPath, timeoutMs });
  if (!r.stdout.trim()) return null;
  // `go list -json` emits concatenated JSON objects; split on `}\n{`.
  const entries: GoListEntry[] = [];
  let buf = '';
  let depth = 0;
  let inStr = false;
  let esc = false;
  for (const ch of r.stdout) {
    buf += ch;
    if (esc) { esc = false; continue; }
    if (ch === '\\') { esc = true; continue; }
    if (ch === '"') inStr = !inStr;
    if (inStr) continue;
    if (ch === '{') depth++;
    else if (ch === '}') {
      depth--;
      if (depth === 0) {
        try { entries.push(JSON.parse(buf)); } catch { /* skip malformed */ }
        buf = '';
      }
    }
  }

  const modPrefix = modulePath;
  const internal = new Set<string>();
  const pkgDir = new Map<string, string>();
  for (const e of entries) {
    if (!e.ImportPath) continue;
    if (!e.ImportPath.startsWith(modPrefix)) continue;
    internal.add(e.ImportPath);
    if (e.Dir) pkgDir.set(e.ImportPath, e.Dir);
  }
  const imports = new Map<string, Set<string>>();
  const reverse = new Map<string, Set<string>>();
  for (const p of internal) { imports.set(p, new Set()); reverse.set(p, new Set()); }
  for (const e of entries) {
    if (!e.ImportPath || !internal.has(e.ImportPath)) continue;
    for (const dep of e.Imports ?? []) {
      if (!internal.has(dep) || dep === e.ImportPath) continue;
      imports.get(e.ImportPath)!.add(dep);
      reverse.get(dep)!.add(e.ImportPath);
    }
  }
  return { modulePath: modPrefix, packages: [...internal].sort(), imports, reverse, pkgDir };
}

function detectLayer(pkg: string, modulePath: string, layerOrder: string[]): string | null {
  const rel = pkg.startsWith(modulePath + '/') ? pkg.slice(modulePath.length + 1) : pkg;
  const segs = rel.split('/');
  for (const seg of segs) {
    if (layerOrder.includes(seg)) return seg;
  }
  return null;
}

function layeringViolations(graph: PackageGraph, convention: LayeringConvention): Violation[] {
  if (!convention.hasLayering || convention.layers.length < 2) return [];
  const layerOrder = convention.layers;
  const out: Violation[] = [];
  const { modulePath, imports } = graph;
  // The deepest layers (last in list) get high severity when they violate upward.
  const deepLayers = new Set(layerOrder.slice(Math.ceil(layerOrder.length * 0.7)));
  const seen = new Set<string>();
  for (const [src, deps] of imports) {
    const srcLayer = detectLayer(src, modulePath, layerOrder);
    if (!srcLayer) continue;
    const srcIdx = layerOrder.indexOf(srcLayer);
    for (const dep of deps) {
      const depLayer = detectLayer(dep, modulePath, layerOrder);
      if (!depLayer) continue;
      const depIdx = layerOrder.indexOf(depLayer);
      // inner layer (higher idx, lower in list) should NOT import outer layer
      if (depIdx < srcIdx) {
        const key = `${srcLayer}->${depLayer}|${src}->${dep}`;
        if (seen.has(key)) continue;
        seen.add(key);
        const srcShort = src.slice(modulePath.length + 1);
        const depShort = dep.slice(modulePath.length + 1);
        out.push({
          id: hashRuleSource('go-arch', `layering-${srcLayer}-imports-${depLayer}`, srcShort, depShort),
          source: 'l1-tool:go-arch',
          category: 'layering',
          ruleId: `go-arch:layering:${srcLayer}->${depLayer}`,
          ruleSource: 'skill-builtin',
          severity: deepLayers.has(srcLayer) ? 'high' : 'medium',
          locations: [{ file: srcShort + '/', module: src }],
          evidence: {
            metric: { name: 'layering-violation', value: 1, threshold: 0 },
            signals: [
              `src-layer=${srcLayer}`, `dep-layer=${depLayer}`,
              `src-pkg=${srcShort}`, `dep-pkg=${depShort}`,
              `convention=${convention.conventionSummary}`,
              'metric-hard-evidence',
            ],
          },
          message: `Layering violation: ${srcLayer} package \`${srcShort}\` imports ${depLayer} package \`${depShort}\` (inner should not depend on outer per convention: ${convention.conventionSummary})`,
        });
      }
    }
  }
  return out;
}

function tarjanSCC(nodes: string[], edges: Map<string, Set<string>>): string[][] {
  let index = 0;
  const stack: string[] = [];
  const idx = new Map<string, number>();
  const low = new Map<string, number>();
  const onStack = new Map<string, boolean>();
  const result: string[][] = [];
  function strongconnect(v: string): void {
    idx.set(v, index); low.set(v, index); index++;
    stack.push(v); onStack.set(v, true);
    for (const w of edges.get(v) ?? []) {
      if (!idx.has(w)) {
        strongconnect(w);
        low.set(v, Math.min(low.get(v)!, low.get(w)!));
      } else if (onStack.get(w)) {
        low.set(v, Math.min(low.get(v)!, idx.get(w)!));
      }
    }
    if (low.get(v) === idx.get(v)) {
      const comp: string[] = [];
      while (stack.length) {
        const w = stack.pop()!;
        onStack.set(w, false);
        comp.push(w);
        if (w === v) break;
      }
      if (comp.length > 1) result.push(comp);
    }
  }
  for (const n of nodes) if (!idx.has(n)) strongconnect(n);
  return result;
}

function cycleViolations(graph: PackageGraph): Violation[] {
  const comps = tarjanSCC(graph.packages, graph.imports);
  const out: Violation[] = [];
  const mod = graph.modulePath;
  for (const comp of comps) {
    const short = comp.map((p) => p.slice(mod.length + 1)).sort();
    const id = hashRuleSource('go-arch', 'package-cycle', short[0], short[short.length - 1]);
    out.push({
      id,
      source: 'l1-tool:go-arch',
      category: 'dependency',
      ruleId: 'go-arch:cycle',
      ruleSource: 'skill-builtin',
      severity: 'high',
      locations: short.map((s) => ({ file: s + '/', module: mod + '/' + s })),
      evidence: {
        metric: { name: 'cycle-size', value: comp.length, threshold: 1 },
        signals: ['metric-hard-evidence', `cycle-members=${short.join(',')}`],
      },
      message: `Package cycle of size ${comp.length}: ${short.join(' → ')} → (back)`,
    });
  }
  return out;
}

function horizontalCouplingViolations(graph: PackageGraph, convention: LayeringConvention): Violation[] {
  // Only business-oriented layers have a notion of "modules" that are supposed
  // to be isolated. Infrastructure layers (dal/dao) and auto-generated code
  // directories intentionally share helpers — flagging them produces noise.
  const HORIZONTAL_BUSINESS_LAYERS = new Set(
    convention.hasLayering && convention.businessLayers.length > 0
      ? convention.businessLayers
      : ['handler', 'consumer', 'facade', 'service']
  );
  // Sibling coupling: two direct children of the same business-layered
  // directory importing each other. Flags handler-to-handler, service-to-
  // service, consumer-to-consumer imports which break module isolation.
  // Infrastructure layers (dal, dao, repository) are excluded — tight
  // coupling between infra siblings is usually legitimate.
  const out: Violation[] = [];
  const mod = graph.modulePath;
  const crossings = new Map<string, Set<string>>(); // sourceSibling -> targetSiblings
  const parentMap = new Map<string, { parent: string; sibling: string; pkg: string }>();
  for (const pkg of graph.packages) {
    const rel = pkg.slice(mod.length + 1);
    const segs = rel.split('/');
    // Only flag sibling crossings under business-oriented parent layers.
    for (let i = 0; i < segs.length - 1; i++) {
      if (HORIZONTAL_BUSINESS_LAYERS.has(segs[i])) {
        const parent = segs.slice(0, i + 1).join('/');
        const sibling = segs[i + 1];
        parentMap.set(pkg, { parent, sibling, pkg });
        break;
      }
    }
  }
  for (const [src, deps] of graph.imports) {
    const sm = parentMap.get(src);
    if (!sm) continue;
    for (const dep of deps) {
      const dm = parentMap.get(dep);
      if (!dm) continue;
      if (sm.parent !== dm.parent) continue;
      if (sm.sibling === dm.sibling) continue; // within same sibling is fine
      const key = `${sm.parent}|${sm.sibling}`;
      if (!crossings.has(key)) crossings.set(key, new Set());
      crossings.get(key)!.add(dm.sibling);
    }
  }
  for (const [key, targets] of crossings) {
    if (targets.size < HORIZONTAL_MIN_CROSSINGS) continue;
    const [parent, sibling] = key.split('|');
    out.push({
      id: hashRuleSource('go-arch', 'horizontal-coupling', parent + '/' + sibling, [...targets].join(',')),
      source: 'l1-tool:go-arch',
      category: 'boundary',
      ruleId: 'go-arch:horizontal-coupling',
      ruleSource: 'skill-builtin',
      severity: targets.size >= 3 ? 'high' : 'medium',
      locations: [{ file: parent + '/' + sibling + '/', module: mod + '/' + parent + '/' + sibling }],
      evidence: {
        metric: { name: 'sibling-crossings', value: targets.size, threshold: HORIZONTAL_MIN_CROSSINGS },
        signals: [
          `parent=${parent}`, `sibling=${sibling}`,
          `depends-on-siblings=${[...targets].sort().join(',')}`,
          'metric-hard-evidence',
        ],
      },
      message: `Business module \`${parent}/${sibling}\` crosses its sibling boundary — directly imports sibling module(s) ${[...targets].sort().map((t) => '`' + t + '`').join(', ')} under the same ${parent}/ layer`,
    });
  }
  return out;
}

/** Compute percentile from a sorted numeric array. */
function percentile(sorted: number[], p: number): number {
  if (sorted.length === 0) return 0;
  const idx = Math.ceil(sorted.length * p / 100) - 1;
  return sorted[Math.max(0, Math.min(idx, sorted.length - 1))];
}

/** Compute the adaptive fan-in threshold for God Package detection.
 *  Uses the repo's own fan-in distribution (p90) with an absolute floor. */
function computeAdaptiveFanInThreshold(graph: PackageGraph): number {
  const fanIns = graph.packages.map((p) => graph.reverse.get(p)?.size ?? 0).sort((a, b) => a - b);
  const p90 = percentile(fanIns, 90);
  // The threshold should be meaningful: at least FAN_IN_FLOOR (5) to avoid
  // trivial noise in small repos, but no more than what p90 suggests.
  return Math.max(FAN_IN_FLOOR, p90);
}

function godPackageViolations(graph: PackageGraph): Violation[] {
  // I = fan_out / (fan_in + fan_out); >0.5 with fan_in above the repo's p90
  // is a classic "god bag": many dependents but itself also depends on many
  // things. The threshold adapts to the repo's size — a 20-package repo with
  // max fan-in of 4 won't fire, but a 200-package repo where p90 is 8 will.
  const fanInThreshold = computeAdaptiveFanInThreshold(graph);
  const highSevThreshold = Math.ceil(fanInThreshold * 1.5);
  const out: Violation[] = [];
  const mod = graph.modulePath;
  for (const pkg of graph.packages) {
    const fi = graph.reverse.get(pkg)?.size ?? 0;
    const fo = graph.imports.get(pkg)?.size ?? 0;
    if (fi < fanInThreshold) continue;
    const I = fi + fo === 0 ? 0 : fo / (fi + fo);
    if (I <= GOD_INSTABILITY) continue;
    const short = pkg.slice(mod.length + 1);
    out.push({
      id: hashRuleSource('go-arch', 'god-package', short),
      source: 'l1-tool:go-arch',
      category: 'coupling',
      ruleId: 'go-arch:god-package',
      ruleSource: 'skill-builtin',
      severity: fi >= highSevThreshold ? 'high' : 'medium',
      locations: [{ file: short + '/', module: pkg }],
      evidence: {
        metric: { name: 'instability-I', value: Number(I.toFixed(2)), threshold: GOD_INSTABILITY },
        signals: [
          `fan-in=${fi}`, `fan-out=${fo}`, `instability=${I.toFixed(2)}`,
          `fan-in-threshold=${fanInThreshold} (repo p90, floor=${FAN_IN_FLOOR})`,
          'metric-hard-evidence',
        ],
      },
      message: `God package \`${short}\`: fan-in=${fi}, fan-out=${fo}, instability I=${I.toFixed(2)}. Threshold fan-in≥${fanInThreshold} (repo p90). High fan-in means many dependents; simultaneous high fan-out means it is also pulling in many things — a core refactor blocker.`,
    });
  }
  return out;
}

// --- code-clone (cross-file line overlap) -----------------------------------

/** Compute adaptive clone detection threshold based on repo's file size distribution.
 *  For repos with many large files, we raise the bar to avoid noise from boilerplate.
 *  For repos with small files, we lower it to catch meaningful duplication. */
function computeAdaptiveCloneThreshold(fileSizes: number[]): number {
  if (fileSizes.length === 0) return CLONE_MIN_IDENTICAL_LINES;
  const sorted = [...fileSizes].sort((a, b) => a - b);
  const median = sorted[Math.floor(sorted.length / 2)];
  // Use 40% of median file size as the threshold, but never below the floor.
  // Rationale: if the median file is 500 lines, then 200 identical lines (40%)
  // is a real problem. If the median is 100 lines, then 50 lines (40%) is
  // already significant.
  return Math.max(CLONE_LINES_FLOOR, Math.round(median * 0.4));
}

async function findCodeClones(rootPath: string, graph: PackageGraph): Promise<Violation[]> {
  const all = await walkRepo(rootPath);
  const files = all.filter((f) =>
    f.endsWith('.go') &&
    !f.endsWith('_test.go') &&
    !f.endsWith('.gen.go') &&
    !f.includes('/mocks_autogen/') &&
    !f.includes('/vendor/'));

  // Load & normalize files (drop blank/comment lines to reduce noise).
  const lines = new Map<string, string[]>();
  const allFileSizes: number[] = [];
  for (const f of files) {
    try {
      const content = await readFile(f, 'utf-8');
      const filtered = content.split('\n').map((l) => l.trim()).filter((l) => l && !l.startsWith('//'));
      allFileSizes.push(filtered.length);
      if (filtered.length < CLONE_LINES_FLOOR) continue;
      lines.set(f, filtered);
    } catch { /* ignore */ }
  }

  const cloneThreshold = computeAdaptiveCloneThreshold(allFileSizes);
  const highSevThreshold = Math.max(cloneThreshold * 2.5, 500);
  const entries = [...lines.entries()];
  const out: Violation[] = [];
  const seenPairs = new Set<string>();

  // Heuristic: only compare files whose basenames are related (same last part
  // OR both named common.go / service.go / helper.go). O(n^2) but only on
  // medium-size files; typical repos yield a few hundred files.
  for (let i = 0; i < entries.length; i++) {
    for (let j = i + 1; j < entries.length; j++) {
      const [fa, la] = entries[i];
      const [fb, lb] = entries[j];
      const baseA = fa.split('/').pop()!;
      const baseB = fb.split('/').pop()!;
      if (baseA !== baseB && !(baseA.startsWith('common') && baseB.startsWith('common'))) continue;
      // Must live in different package dirs.
      if (dirname(fa) === dirname(fb)) continue;
      const setA = new Set(la);
      let overlap = 0;
      for (const line of lb) if (setA.has(line)) overlap++;
      const smaller = Math.min(la.length, lb.length);
      const ratio = overlap / smaller;
      if (overlap < cloneThreshold || ratio < CLONE_MIN_RATIO) continue;
      const key = [fa, fb].sort().join('||');
      if (seenPairs.has(key)) continue;
      seenPairs.add(key);
      const relA = relative(rootPath, fa);
      const relB = relative(rootPath, fb);
      out.push({
        id: hashRuleSource('go-arch', 'code-clone', relA, relB),
        source: 'l1-tool:go-arch',
        category: 'structure',
        ruleId: 'go-arch:code-clone',
        ruleSource: 'skill-builtin',
        severity: overlap >= highSevThreshold ? 'high' : 'medium',
        locations: [
          { file: relA },
          { file: relB },
        ],
        evidence: {
          metric: { name: 'identical-lines', value: overlap, threshold: cloneThreshold },
          signals: [
            `overlap-lines=${overlap}`,
            `overlap-ratio=${ratio.toFixed(2)}`,
            `file-a-loc=${la.length}`, `file-b-loc=${lb.length}`,
            `clone-threshold=${cloneThreshold} (adaptive, median-file=${allFileSizes.length > 0 ? Math.round(allFileSizes.sort((a, b) => a - b)[Math.floor(allFileSizes.length / 2)]) : '?'} LOC)`,
            'metric-hard-evidence',
          ],
        },
        message: `Cross-package code clone: \`${relA}\` and \`${relB}\` share ${overlap} identical non-comment lines (${Math.round(ratio * 100)}% of the smaller file, threshold=${cloneThreshold}). Consider extracting the common code into a shared package.`,
      });
    }
  }
  return out;
}

void join; // keep import for potential future use without breaking ts build

export const goArchAdapter: ToolAdapter = {
  id: 'go-arch',

  detectApplies(profile) {
    return profile.hasGoMod && !!profile.goModulePath;
  },

  async preflight(): Promise<PreflightResult> {
    // Only needs `go` in PATH, which is a prerequisite for a Go repo anyway.
    const r = await runCmd('go', ['version'], { timeoutMs: 10_000 });
    if (r.status !== 'ok') return { ready: false, reason: 'no-binary' };
    return { ready: true, installedVia: 'pre-existing' };
  },

  async run(ctx): Promise<RawRunOutput> {
    const mod = ctx.profile.goModulePath!;
    const graph = await buildGraph(ctx.profile.rootPath, mod, ctx.timeoutMs);
    if (!graph) {
      throw new Error('go list failed to produce any output');
    }
    // We stash the graph itself as raw output so debuggers can inspect it.
    const raw = {
      modulePath: mod,
      packageCount: graph.packages.length,
      fanIn: Object.fromEntries([...graph.reverse].map(([k, v]) => [k, v.size])),
      fanOut: Object.fromEntries([...graph.imports].map(([k, v]) => [k, v.size])),
    };
    const rawOutputPath = join(ctx.tmpDir, 'go-arch.json');
    const { writeFile } = await import('node:fs/promises');
    await writeFile(rawOutputPath, JSON.stringify(raw, null, 2));
    // Attach graph to ctx for the normalize step (cheap hack; we compute
    // violations inline here and cache them on the ctx object).
    (ctx as unknown as { _goArchGraph: PackageGraph })._goArchGraph = graph;
    return { rawOutputPath, exitCode: 0, stdout: '', stderr: '', durationMs: 0 };
  },

  normalize(raw, ctx): Violation[] {
    const graph = (ctx as unknown as { _goArchGraph?: PackageGraph })._goArchGraph;
    if (!graph) return [];
    const convention = ctx.profile.layering ?? FALLBACK_CONVENTION;
    const vios: Violation[] = [];
    vios.push(...layeringViolations(graph, convention));
    vios.push(...cycleViolations(graph));
    vios.push(...horizontalCouplingViolations(graph, convention));
    vios.push(...godPackageViolations(graph));
    // code-clone is async; run it synchronously via a thenable hack is not
    // clean. Instead compute it at run-time and cache, similar to graph.
    const clones = (ctx as unknown as { _goArchClones?: Violation[] })._goArchClones ?? [];
    vios.push(...clones);
    void raw;
    return vios;
  },
};

// The adapter's `run` is async, but `normalize` is sync per the existing
// ToolAdapter contract. We expose a small helper so the orchestrator (or a
// wrapper adapter) can pre-seed code-clone results before `normalize` runs.
// In practice we fold the clone scan into `run` via a side-effect on ctx.
const originalRun = goArchAdapter.run.bind(goArchAdapter);
goArchAdapter.run = async function (ctx: AdapterContext): Promise<RawRunOutput> {
  const r = await originalRun(ctx);
  const graph = (ctx as unknown as { _goArchGraph?: PackageGraph })._goArchGraph;
  if (graph) {
    try {
      const clones = await findCodeClones(ctx.profile.rootPath, graph);
      (ctx as unknown as { _goArchClones: Violation[] })._goArchClones = clones;
    } catch {
      (ctx as unknown as { _goArchClones: Violation[] })._goArchClones = [];
    }
  }
  return r;
};
