import { writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { runCmd } from '../../util/exec.js';
import { hashRuleSource, type ToolAdapter, type AdapterContext, type PreflightResult, type RawRunOutput } from '../adapter.js';
import type { Violation } from '../../types.js';

interface GoPackage {
  ImportPath: string;
  Imports?: string[];
  Dir?: string;
}

function findCycles(packages: GoPackage[]): string[][] {
  const idxOf = new Map<string, number>();
  packages.forEach((p, i) => idxOf.set(p.ImportPath, i));
  const graph: number[][] = packages.map((p) =>
    (p.Imports ?? []).map((imp) => idxOf.get(imp) ?? -1).filter((i) => i >= 0)
  );

  const indices = new Array(packages.length).fill(-1);
  const lowlinks = new Array(packages.length).fill(-1);
  const onStack = new Array(packages.length).fill(false);
  const stack: number[] = [];
  let index = 0;
  const sccs: string[][] = [];

  function strongconnect(v: number): void {
    indices[v] = index;
    lowlinks[v] = index;
    index++;
    stack.push(v);
    onStack[v] = true;
    for (const w of graph[v]) {
      if (indices[w] < 0) {
        strongconnect(w);
        lowlinks[v] = Math.min(lowlinks[v], lowlinks[w]);
      } else if (onStack[w]) {
        lowlinks[v] = Math.min(lowlinks[v], indices[w]);
      }
    }
    if (lowlinks[v] === indices[v]) {
      const scc: string[] = [];
      let w = -1;
      do {
        w = stack.pop()!;
        onStack[w] = false;
        scc.push(packages[w].ImportPath);
      } while (w !== v);
      if (scc.length > 1) sccs.push(scc);
    }
  }

  for (let v = 0; v < packages.length; v++) {
    if (indices[v] < 0) strongconnect(v);
  }
  return sccs;
}

export const goListAdapter: ToolAdapter & {
  normalizeFromPackages(packages: GoPackage[], ctx: AdapterContext): Violation[];
} = {
  id: 'go-list',

  detectApplies(profile) {
    return profile.hasGoMod;
  },

  async preflight(): Promise<PreflightResult> {
    const r = await runCmd('go', ['version'], { timeoutMs: 10_000 });
    if (r.status !== 'ok') return { ready: false, reason: 'no-binary' };
    return { ready: true, installedVia: 'pre-existing' };
  },

  async run(ctx): Promise<RawRunOutput> {
    const rawOutputPath = join(ctx.tmpDir, 'go-list.json');
    const r = await runCmd('go', ['list', '-deps', '-json', './...'], {
      cwd: ctx.profile.rootPath,
      timeoutMs: ctx.timeoutMs,
    });
    await writeFile(rawOutputPath, r.stdout || '');
    return { rawOutputPath, exitCode: r.exitCode, stdout: r.stdout, stderr: r.stderr, durationMs: r.durationMs };
  },

  normalize(raw, ctx): Violation[] {
    const packages: GoPackage[] = [];
    const chunks = raw.stdout.split(/^}\n/m).map((c) => c.trim()).filter(Boolean);
    for (const chunk of chunks) {
      try {
        packages.push(JSON.parse(chunk.endsWith('}') ? chunk : chunk + '}'));
      } catch {
        /* skip malformed */
      }
    }
    return this.normalizeFromPackages(packages, ctx);
  },

  normalizeFromPackages(packages, ctx): Violation[] {
    void ctx;
    const out: Violation[] = [];
    const sccs = findCycles(packages);
    for (const scc of sccs) {
      const file = scc[0].split('/').slice(-2).join('/') + '/';
      out.push({
        id: hashRuleSource('go-list', 'circular', file, scc.join('|')),
        source: 'l1-tool:go-list',
        category: 'dependency',
        ruleId: 'circular',
        ruleSource: 'skill-builtin',
        severity: 'high',
        locations: scc.map((p) => ({ file: p.split('/').slice(-2).join('/') + '/', module: p })),
        evidence: { signals: [`scc-size=${scc.length}`, `cycle=${scc.join(' -> ')}`] },
        message: `Go package cycle: ${scc.join(' -> ')}`,
      });
    }
    return out;
  },
};
