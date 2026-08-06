import { writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { runCmd } from '../../util/exec.js';
import { ensureBinary } from '../auto-install.js';
import { hashRuleSource, type ToolAdapter, type AdapterContext, type PreflightResult, type RawRunOutput } from '../adapter.js';
import type { Violation } from '../../types.js';

interface GrimpOutput {
  cycles: string[][];
  orphans: string[];
  stats: { modules: number; imports: number };
}

function modToFile(mod: string): string {
  return mod.replace(/\./g, '/') + '.py';
}

function grimpScript(rootPackages: string[]): string {
  return `
import json, grimp
graph = grimp.build_graph(${JSON.stringify(rootPackages)})
try:
    sccs = [list(c) for c in graph.find_strongly_connected_components() if len(c) > 1]
except AttributeError:
    sccs = []
orphans = []
for m in graph.modules:
    if graph.count_imports_of(m) == 0 and graph.count_imports_by(m) == 0:
        orphans.append(m)
print(json.dumps({"cycles": sccs, "orphans": orphans, "stats": {"modules": len(graph.modules), "imports": sum(graph.count_imports_by(m) for m in graph.modules)}}))
`;
}

export const grimpAdapter: ToolAdapter & {
  normalizeFromParsed(parsed: GrimpOutput, ctx: AdapterContext): Violation[];
} = {
  id: 'grimp',

  detectApplies(profile) {
    return !!profile.topLevelPackages.python && profile.topLevelPackages.python.length > 0;
  },

  async preflight(): Promise<PreflightResult> {
    const ensured = await ensureBinary({
      binary: 'python3',
      installPlan: [],
      checkCommand: { cmd: 'python3', args: ['-c', 'import grimp'] },
    });
    if (ensured.ready) return { ready: true, installedVia: ensured.installedVia };
    const ins = await runCmd('pipx', ['install', '--include-deps', 'grimp'], { timeoutMs: 120_000 });
    if (ins.status !== 'ok') return { ready: false, reason: 'install-failed' };
    return { ready: true, installedVia: 'pipx' };
  },

  async run(ctx): Promise<RawRunOutput> {
    const rawOutputPath = join(ctx.tmpDir, 'grimp.json');
    const roots = (ctx.profile.topLevelPackages.python ?? []).filter((p) => !p.includes('.'));
    const r = await runCmd('python3', ['-c', grimpScript(roots)], {
      cwd: ctx.profile.rootPath,
      timeoutMs: ctx.timeoutMs,
    });
    await writeFile(
      rawOutputPath,
      r.stdout || '{"cycles":[],"orphans":[],"stats":{"modules":0,"imports":0}}'
    );
    return { rawOutputPath, exitCode: r.exitCode, stdout: r.stdout, stderr: r.stderr, durationMs: r.durationMs };
  },

  normalize(raw, ctx): Violation[] {
    let parsed: GrimpOutput;
    try {
      parsed = JSON.parse(raw.stdout);
    } catch {
      parsed = { cycles: [], orphans: [], stats: { modules: 0, imports: 0 } };
    }
    return this.normalizeFromParsed(parsed, ctx);
  },

  normalizeFromParsed(parsed, ctx): Violation[] {
    void ctx;
    const out: Violation[] = [];

    // Normalize orphans into dead-code violations
    for (const mod of parsed.orphans) {
      const file = modToFile(mod);
      out.push({
        id: hashRuleSource('grimp', 'orphan', file),
        source: 'l1-tool:grimp',
        category: 'dead-code',
        ruleId: 'orphan',
        ruleSource: 'skill-builtin',
        severity: 'low',
        locations: [{ file, module: mod }],
        evidence: { signals: ['orphan-module', 'importers=0'] },
        message: `Orphan Python module: ${mod}`,
      });
    }

    // Normalize cycles into dependency violations
    for (const scc of parsed.cycles) {
      if (scc.length < 2) continue;
      const files = scc.map(modToFile);
      out.push({
        id: hashRuleSource('grimp', 'circular', files.join(','), scc.join('|')),
        source: 'l1-tool:grimp',
        category: 'dependency',
        ruleId: 'circular',
        ruleSource: 'skill-builtin',
        severity: 'high',
        locations: files.map((file, i) => ({ file, module: scc[i] })),
        evidence: { signals: [`cycle-size=${scc.length}`, `cycle=${scc.join(' -> ')}`] },
        message: `Circular imports: ${scc.join(' -> ')}`,
      });
    }

    return out;
  },
};
