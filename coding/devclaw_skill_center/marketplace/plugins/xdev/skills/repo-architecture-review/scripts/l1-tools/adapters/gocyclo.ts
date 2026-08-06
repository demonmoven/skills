import { writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { runCmd } from '../../util/exec.js';
import { ensureBinary } from '../auto-install.js';
import { hashRuleSource, type ToolAdapter, type AdapterContext, type PreflightResult, type RawRunOutput } from '../adapter.js';
import type { Violation } from '../../types.js';

// gocyclo report threshold (complexity below this is silently dropped).
const REPORT_THRESHOLD = 15;
// Severity tiers — chosen so that ordinary glue code never makes the report,
// but real "switch-on-everything" or "if-else mountain" functions show up.
const HIGH_THRESHOLD = 30;
const MED_THRESHOLD = 20;

interface ParsedFn { complexity: number; pkg: string; func: string; file: string; line: number }

// gocyclo line format: "<complexity> <package> <function> <file>:<line>:<col>"
function parseLine(line: string): ParsedFn | null {
  const m = line.match(/^(\d+)\s+(\S+)\s+(\S+)\s+(\S+):(\d+):\d+/);
  if (!m) return null;
  return { complexity: Number(m[1]), pkg: m[2], func: m[3], file: m[4], line: Number(m[5]) };
}

function severityFor(c: number): Violation['severity'] {
  if (c >= HIGH_THRESHOLD) return 'high';
  if (c >= MED_THRESHOLD) return 'medium';
  return 'low';
}

export const gocycloAdapter: ToolAdapter & {
  normalizeFromParsed(rows: ParsedFn[], ctx: AdapterContext): Violation[];
} = {
  id: 'gocyclo',

  detectApplies(profile) {
    return profile.hasGoMod;
  },

  async preflight(): Promise<PreflightResult> {
    const ensured = await ensureBinary({
      binary: 'gocyclo',
      installPlan: [{ via: 'go-install', pkg: 'github.com/fzipp/gocyclo/cmd/gocyclo@latest' }],
    });
    return ensured.ready ? { ready: true, installedVia: ensured.installedVia } : { ready: false, reason: 'install-failed' };
  },

  async run(ctx): Promise<RawRunOutput> {
    const rawOutputPath = join(ctx.tmpDir, 'gocyclo.txt');
    // -ignore filters generated/test files which would otherwise dominate noise.
    const r = await runCmd(
      'gocyclo',
      ['-over', String(REPORT_THRESHOLD - 1), '-ignore', '\\.gen\\.go|_test\\.go|mocks_autogen', '.'],
      { cwd: ctx.profile.rootPath, timeoutMs: ctx.timeoutMs },
    );
    await writeFile(rawOutputPath, r.stdout || '');
    // gocyclo exits non-zero when it finds anything above threshold — treat as ok.
    return { rawOutputPath, exitCode: 0, stdout: r.stdout, stderr: r.stderr, durationMs: r.durationMs };
  },

  normalize(raw, ctx): Violation[] {
    const rows: ParsedFn[] = [];
    for (const line of raw.stdout.split('\n')) {
      const p = parseLine(line);
      if (p) rows.push(p);
    }
    return this.normalizeFromParsed(rows, ctx);
  },

  normalizeFromParsed(rows, _ctx): Violation[] {
    const out: Violation[] = [];
    for (const r of rows) {
      out.push({
        id: hashRuleSource('gocyclo', 'high-cyclomatic-complexity', r.file, r.func),
        source: 'l1-tool:gocyclo',
        category: 'structure',
        ruleId: 'high-cyclomatic-complexity',
        ruleSource: 'skill-builtin',
        severity: severityFor(r.complexity),
        locations: [{ file: r.file, line: r.line, symbol: `${r.pkg}.${r.func}` }],
        evidence: { metric: { name: 'cyclomatic-complexity', value: r.complexity, threshold: REPORT_THRESHOLD } },
        message: `${r.pkg}.${r.func} has cyclomatic complexity ${r.complexity} (threshold ${REPORT_THRESHOLD})`,
      });
    }
    return out;
  },
};
