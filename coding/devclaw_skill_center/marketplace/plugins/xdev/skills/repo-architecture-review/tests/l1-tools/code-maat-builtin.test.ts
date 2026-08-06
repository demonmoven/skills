import { describe, it, expect } from 'vitest';
import { computeCoChangeFromLog, type Commit } from '../../scripts/l1-tools/adapters/code-maat-builtin.js';

const commits: Commit[] = [
  { sha: 'c1', files: ['src/domain/user.py', 'src/infra/db.py'] },
  { sha: 'c2', files: ['src/domain/user.py', 'src/infra/db.py'] },
  { sha: 'c3', files: ['src/domain/user.py', 'src/infra/db.py'] },
  { sha: 'c4', files: ['src/domain/user.py'] },
  { sha: 'c5', files: ['src/unrelated.py'] },
];

describe('computeCoChangeFromLog', () => {
  it('counts co-change pairs and ignores single-file commits', () => {
    const pairs = computeCoChangeFromLog(commits, { minSharedRevs: 2, maxChangesetSize: 30, minRevs: 1, minCouplingPct: 0 });
    const hit = pairs.find((p) => p.a === 'src/domain/user.py' && p.b === 'src/infra/db.py');
    expect(hit).toBeDefined();
    expect(hit!.shared).toBe(3);
    expect(hit!.revsA).toBe(4);
    expect(hit!.revsB).toBe(3);
    expect(hit!.couplingPct).toBeCloseTo(3 / 4 * 100, 0);
  });

  it('filters out large changesets (refactor noise)', () => {
    const bigCommit: Commit = { sha: 'big', files: new Array(50).fill(0).map((_, i) => `f${i}.py`) };
    const pairs = computeCoChangeFromLog([...commits, bigCommit], { minSharedRevs: 1, maxChangesetSize: 30, minRevs: 1, minCouplingPct: 0 });
    expect(pairs.every((p) => !p.a.startsWith('f') && !p.b.startsWith('f'))).toBe(true);
  });
});
