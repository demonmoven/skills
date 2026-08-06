import { writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { hashRuleSource, type ToolAdapter, type AdapterContext, type PreflightResult, type RawRunOutput } from '../adapter.js';
import type { Violation } from '../../types.js';
import { computeCoChangeFromLog, DEFAULT_CC_OPTS, loadCommitsFromGit, type CoChangePair, type Commit } from './code-maat-builtin.js';

const HIGH_COUPLING_WARN = 74;

// Pairs that co-change for structural reasons (lockfiles, generated-alongside,
// source+its own test) are not signals of architectural coupling. Dropping them
// at source avoids polluting L3 grouping and the final report.
const TRIVIAL_PAIRS: ReadonlyArray<readonly [RegExp, RegExp]> = [
  [/(^|\/)go\.mod$/, /(^|\/)go\.sum$/],
  [/(^|\/)package\.json$/, /(^|\/)(package-lock\.json|pnpm-lock\.yaml|yarn\.lock)$/],
  [/(^|\/)Cargo\.toml$/, /(^|\/)Cargo\.lock$/],
  [/(^|\/)(requirements|constraints)\.txt$/, /(^|\/)(requirements|constraints)\.lock$/],
  [/(^|\/)Pipfile$/, /(^|\/)Pipfile\.lock$/],
  [/(^|\/)poetry\.lock$/, /(^|\/)pyproject\.toml$/],
];

function sameBasenameWithTestSuffix(a: string, b: string): boolean {
  // foo.go ↔ foo_test.go, foo.ts ↔ foo.test.ts, foo.py ↔ test_foo.py etc.
  const variants = (f: string): string[] => {
    const m = f.match(/^(.*)[\/]([^/]+)$/);
    const dir = m ? m[1] : '';
    const name = m ? m[2] : f;
    const base = name.replace(/\.(go|ts|tsx|js|jsx|py|rb|java|kt|rs)$/, '');
    const ext = name.slice(base.length);
    return [
      `${dir}/${base}_test${ext}`.replace(/^\//, ''),
      `${dir}/${base}.test${ext}`.replace(/^\//, ''),
      `${dir}/${base}.spec${ext}`.replace(/^\//, ''),
      `${dir}/test_${base}${ext}`.replace(/^\//, ''),
      `${dir}/${base}_spec${ext}`.replace(/^\//, ''),
    ];
  };
  return variants(a).includes(b) || variants(b).includes(a);
}

function isTrivialCoChange(a: string, b: string): boolean {
  for (const [pa, pb] of TRIVIAL_PAIRS) {
    if ((pa.test(a) && pb.test(b)) || (pa.test(b) && pb.test(a))) return true;
  }
  return sameBasenameWithTestSuffix(a, b);
}

export const codeMaatAdapter: ToolAdapter & {
  normalizeFromPairs(pairs: CoChangePair[], ctx: AdapterContext): Violation[];
} = {
  id: 'code-maat',

  detectApplies(profile) {
    return profile.totalFiles > 0;
  },

  async preflight(ctx): Promise<PreflightResult> {
    void ctx;
    return { ready: true, installedVia: 'pre-existing' };
  },

  async run(ctx): Promise<RawRunOutput> {
    const rawOutputPath = join(ctx.tmpDir, 'code-maat.json');
    let commits: Commit[] = [];
    try {
      commits = await loadCommitsFromGit(ctx.profile.rootPath, DEFAULT_CC_OPTS.sinceMonths);
    } catch {
      // No git history — emit empty result.
    }
    const pairs = computeCoChangeFromLog(commits, DEFAULT_CC_OPTS);
    await writeFile(rawOutputPath, JSON.stringify({ pairs, commitCount: commits.length }, null, 2));
    return { rawOutputPath, exitCode: 0, stdout: JSON.stringify(pairs), stderr: '', durationMs: 0 };
  },

  normalize(raw, ctx): Violation[] {
    let pairs: CoChangePair[];
    try {
      pairs = JSON.parse(raw.stdout);
    } catch {
      pairs = [];
    }
    return this.normalizeFromPairs(pairs, ctx);
  },

  normalizeFromPairs(pairs, ctx): Violation[] {
    void ctx;
    return pairs
      .filter((p) => p.couplingPct >= HIGH_COUPLING_WARN)
      .filter((p) => !isTrivialCoChange(p.a, p.b))
      .map((p) => ({
        id: hashRuleSource('code-maat', 'high-temporal-coupling', p.a, p.b),
        source: 'l1-tool:code-maat',
        category: 'coupling' as const,
        ruleId: 'high-temporal-coupling',
        ruleSource: 'skill-builtin' as const,
        severity: (p.couplingPct >= 85 ? 'high' : 'medium') as Violation['severity'],
        locations: [{ file: p.a }, { file: p.b }],
        evidence: {
          metric: { name: 'co-change-pct', value: p.couplingPct, threshold: HIGH_COUPLING_WARN },
          signals: [`shared=${p.shared}`, `revsA=${p.revsA}`, `revsB=${p.revsB}`],
        },
        message: `High temporal coupling: ${p.a} co-changes with ${p.b} ${p.couplingPct.toFixed(1)}% of the time`,
      }));
  },
};
