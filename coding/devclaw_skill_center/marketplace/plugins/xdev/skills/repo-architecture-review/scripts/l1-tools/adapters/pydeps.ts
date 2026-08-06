import { writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { runCmd } from '../../util/exec.js';
import { ensureBinary } from '../auto-install.js';
import { hashRuleSource, type ToolAdapter, type AdapterContext, type PreflightResult, type RawRunOutput } from '../adapter.js';
import type { Violation } from '../../types.js';

interface PydepsOutput {
  bacon?: string[][];
}

function modToFile(mod: string): string {
  return mod.replace(/\./g, '/') + '.py';
}

export const pydepsAdapter: ToolAdapter & {
  normalizeFromParsed(parsed: PydepsOutput, ctx: AdapterContext): Violation[];
} = {
  id: 'pydeps',

  detectApplies(profile) {
    return !!profile.topLevelPackages.python && profile.topLevelPackages.python.length > 0;
  },

  async preflight(): Promise<PreflightResult> {
    const ensured = await ensureBinary({
      binary: 'pydeps',
      installPlan: [{ via: 'pipx', pkg: 'pydeps' }],
    });
    if (!ensured.ready) return { ready: false, reason: 'install-failed' };
    return { ready: true, installedVia: ensured.installedVia };
  },

  async run(ctx): Promise<RawRunOutput> {
    const root = (ctx.profile.topLevelPackages.python ?? [])[0];
    const rawOutputPath = join(ctx.tmpDir, 'pydeps.json');
    if (!root) {
      await writeFile(rawOutputPath, '{}');
      return { rawOutputPath, exitCode: 0, stdout: '{}', stderr: '', durationMs: 0 };
    }
    const r = await runCmd('pydeps', [root, '--show-cycles', '--no-show', '--json'], {
      cwd: ctx.profile.rootPath,
      timeoutMs: ctx.timeoutMs,
    });
    await writeFile(rawOutputPath, r.stdout || '{}');
    return { rawOutputPath, exitCode: r.exitCode, stdout: r.stdout, stderr: r.stderr, durationMs: r.durationMs };
  },

  normalize(raw, ctx): Violation[] {
    let parsed: PydepsOutput;
    try {
      parsed = JSON.parse(raw.stdout);
    } catch {
      parsed = {};
    }
    return this.normalizeFromParsed(parsed, ctx);
  },

  normalizeFromParsed(parsed, ctx): Violation[] {
    void ctx;
    const out: Violation[] = [];
    const cycles = parsed.bacon ?? [];
    for (const cycle of cycles) {
      if (cycle.length < 2) continue;
      const files = cycle.map(modToFile);
      out.push({
        id: hashRuleSource('pydeps', 'circular', files.join(','), cycle.join('|')),
        source: 'l1-tool:pydeps',
        category: 'dependency',
        ruleId: 'circular',
        ruleSource: 'skill-builtin',
        severity: 'high',
        locations: files.map((file, i) => ({ file, module: cycle[i] })),
        evidence: { signals: [`cycle=${cycle.join(' -> ')}`] },
        message: `Circular import chain: ${cycle.join(' -> ')}`,
      });
    }
    return out;
  },
};
