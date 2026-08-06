import { writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { runCmd } from '../../util/exec.js';
import { ensureBinary } from '../auto-install.js';
import { type ToolAdapter, type AdapterContext, type PreflightResult, type RawRunOutput } from '../adapter.js';
import type { Violation } from '../../types.js';

interface SccEntry {
  Language: string;
  Files: number;
  Lines: number;
  Code: number;
  Comments: number;
  Blanks: number;
  Complexity: number;
}

export const sccAdapter: ToolAdapter & {
  normalizeFromParsed(parsed: SccEntry[], ctx: AdapterContext): Violation[];
} = {
  id: 'scc',

  detectApplies(profile) {
    return profile.totalFiles > 0;
  },

  async preflight(): Promise<PreflightResult> {
    const ensured = await ensureBinary({
      binary: 'scc',
      installPlan: [{ via: 'go-install', pkg: 'github.com/boyter/scc/v3@latest' }],
    });
    return ensured.ready ? { ready: true, installedVia: ensured.installedVia } : { ready: false, reason: 'install-failed' };
  },

  async run(ctx): Promise<RawRunOutput> {
    const rawOutputPath = join(ctx.tmpDir, 'scc.json');
    const r = await runCmd('scc', ['--format', 'json', '.'], { cwd: ctx.profile.rootPath, timeoutMs: ctx.timeoutMs });
    await writeFile(rawOutputPath, r.stdout || '[]');
    return { rawOutputPath, exitCode: r.exitCode, stdout: r.stdout, stderr: r.stderr, durationMs: r.durationMs };
  },

  normalize(raw, ctx): Violation[] {
    let parsed: SccEntry[];
    try {
      parsed = JSON.parse(raw.stdout || '[]');
    } catch {
      parsed = [];
    }
    return this.normalizeFromParsed(parsed, ctx);
  },

  normalizeFromParsed(parsed, ctx): Violation[] {
    void ctx;
    void parsed;
    return [];
  },
};
