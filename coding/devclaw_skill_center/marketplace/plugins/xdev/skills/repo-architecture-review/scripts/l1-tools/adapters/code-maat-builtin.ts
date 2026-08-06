import { simpleGit, type SimpleGit } from 'simple-git';

export interface Commit {
  sha: string;
  files: string[];
}

export interface CoChangePair {
  a: string;
  b: string;
  shared: number;
  revsA: number;
  revsB: number;
  couplingPct: number;
}

export interface CoChangeOptions {
  sinceMonths: number;
  minRevs: number;
  minSharedRevs: number;
  maxChangesetSize: number;
  minCouplingPct: number;
}

export const DEFAULT_CC_OPTS: CoChangeOptions = {
  sinceMonths: 24,
  minRevs: 5,
  minSharedRevs: 5,
  maxChangesetSize: 30,
  minCouplingPct: 30,
};

export async function loadCommitsFromGit(repoPath: string, sinceMonths: number): Promise<Commit[]> {
  const git: SimpleGit = simpleGit(repoPath);
  const sinceDate = new Date();
  sinceDate.setMonth(sinceDate.getMonth() - sinceMonths);
  const logArgs = ['--name-only', '--pretty=format:__COMMIT__%H', `--since=${sinceDate.toISOString()}`];
  const raw = await git.raw(['log', ...logArgs]);
  const commits: Commit[] = [];
  let current: Commit | null = null;
  for (const line of raw.split('\n')) {
    if (line.startsWith('__COMMIT__')) {
      if (current) commits.push(current);
      current = { sha: line.slice(10), files: [] };
    } else if (current && line.trim()) {
      current.files.push(line.trim());
    }
  }
  if (current) commits.push(current);
  return commits;
}

export function computeCoChangeFromLog(commits: Commit[], opts: Partial<CoChangeOptions> = {}): CoChangePair[] {
  const cfg = { ...DEFAULT_CC_OPTS, ...opts };
  const revCounts = new Map<string, number>();
  const pairCounts = new Map<string, number>();

  for (const c of commits) {
    if (c.files.length > cfg.maxChangesetSize) continue;
    if (c.files.length < 2) {
      for (const f of c.files) revCounts.set(f, (revCounts.get(f) ?? 0) + 1);
      continue;
    }
    for (const f of c.files) revCounts.set(f, (revCounts.get(f) ?? 0) + 1);
    for (let i = 0; i < c.files.length; i++) {
      for (let j = i + 1; j < c.files.length; j++) {
        const [a, b] = c.files[i] < c.files[j] ? [c.files[i], c.files[j]] : [c.files[j], c.files[i]];
        const key = a + '\x00' + b;
        pairCounts.set(key, (pairCounts.get(key) ?? 0) + 1);
      }
    }
  }

  const out: CoChangePair[] = [];
  for (const [key, shared] of pairCounts) {
    if (shared < cfg.minSharedRevs) continue;
    const [a, b] = key.split('\x00');
    const revsA = revCounts.get(a) ?? 0;
    const revsB = revCounts.get(b) ?? 0;
    if (revsA < cfg.minRevs || revsB < cfg.minRevs) continue;
    const couplingPct = (shared / Math.max(revsA, revsB)) * 100;
    if (couplingPct < cfg.minCouplingPct) continue;
    out.push({ a, b, shared, revsA, revsB, couplingPct });
  }
  out.sort((x, y) => y.couplingPct - x.couplingPct);
  return out;
}
