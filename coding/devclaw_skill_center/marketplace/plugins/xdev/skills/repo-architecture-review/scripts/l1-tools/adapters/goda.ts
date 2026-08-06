import { writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { runCmd } from '../../util/exec.js';
import { ensureBinary } from '../auto-install.js';
import { hashRuleSource, type ToolAdapter, type AdapterContext, type PreflightResult, type RawRunOutput } from '../adapter.js';
import type { Violation } from '../../types.js';

const DEFAULT_FAN_IN_THRESHOLD = 20;

function parseDotFanIn(dot: string): Map<string, number> {
  const fanIn = new Map<string, number>();
  for (const line of dot.split('\n')) {
    const m = line.match(/"([^"]+)"\s*->\s*"([^"]+)"/);
    if (m) {
      const target = m[2];
      fanIn.set(target, (fanIn.get(target) ?? 0) + 1);
    }
  }
  return fanIn;
}

export const godaAdapter: ToolAdapter & {
  normalizeFromFanIn(fanIn: Map<string, number>, ctx: AdapterContext): Violation[];
} = {
  id: 'goda',

  detectApplies(profile) {
    return profile.hasGoMod;
  },

  async preflight(): Promise<PreflightResult> {
    const ensured = await ensureBinary({
      binary: 'goda',
      installPlan: [{ via: 'go-install', pkg: 'github.com/loov/goda@latest' }],
    });
    return ensured.ready ? { ready: true, installedVia: ensured.installedVia } : { ready: false, reason: 'install-failed' };
  },

  async run(ctx): Promise<RawRunOutput> {
    const rawOutputPath = join(ctx.tmpDir, 'goda.dot');
    const r = await runCmd('goda', ['graph', '-cluster', '-short', './...'], { cwd: ctx.profile.rootPath, timeoutMs: ctx.timeoutMs });
    await writeFile(rawOutputPath, r.stdout || '');
    return { rawOutputPath, exitCode: r.exitCode, stdout: r.stdout, stderr: r.stderr, durationMs: r.durationMs };
  },

  normalize(raw, ctx): Violation[] {
    const fanIn = parseDotFanIn(raw.stdout);
    return this.normalizeFromFanIn(fanIn, ctx);
  },

  normalizeFromFanIn(fanIn, ctx): Violation[] {
    void ctx;
    const out: Violation[] = [];
    for (const [pkg, count] of fanIn) {
      if (count < DEFAULT_FAN_IN_THRESHOLD) continue;
      out.push({
        id: hashRuleSource('goda', 'high-fan-in', pkg),
        source: 'l1-tool:goda',
        category: 'coupling',
        ruleId: 'high-fan-in',
        ruleSource: 'skill-builtin',
        severity: 'medium',
        locations: [{ file: pkg.split('/').slice(-2).join('/') + '/', module: pkg }],
        evidence: { metric: { name: 'fan-in', value: count, threshold: DEFAULT_FAN_IN_THRESHOLD } },
        message: `Package ${pkg} has fan-in ${count} (threshold ${DEFAULT_FAN_IN_THRESHOLD})`,
      });
    }
    return out;
  },
};
