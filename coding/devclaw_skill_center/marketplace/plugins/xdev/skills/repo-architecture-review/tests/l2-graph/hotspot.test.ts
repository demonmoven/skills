import { describe, it, expect } from 'vitest';
import {
  computeHotspots,
  computeChurnFromCommits,
  type HotspotInput,
} from '../../scripts/l2-graph/metrics/hotspot.js';

describe('hotspot', () => {
  it('sorts hotspots by churn × LOC descending', () => {
    const inputs: HotspotInput[] = [
      { file: 'a.py', churn: 5000, loc: 200 },  // score = 5000 * 200 = 1,000,000
      { file: 'b.py', churn: 4000, loc: 1000 }, // score = 4000 * 1000 = 4,000,000
      { file: 'c.py', churn: 1000, loc: 1000 }, // score = 1000 * 1000 = 1,000,000
    ];

    const hotspots = computeHotspots(inputs, 10);

    expect(hotspots).toHaveLength(3);
    expect(hotspots[0]!.file).toBe('b.py');
    expect(hotspots[0]!.score).toBe(4_000_000);
    expect(hotspots[1]!.file).toBe('a.py');
    expect(hotspots[1]!.file).not.toBe('c.py'); // c ties with a but order is preserved after sort
  });

  it('respects top-N limit', () => {
    const inputs: HotspotInput[] = Array.from({ length: 100 }, (_, i) => ({
      file: `f${i}.py`,
      churn: 100 - i,
      loc: 100,
    }));

    const hotspots = computeHotspots(inputs, 50);
    expect(hotspots).toHaveLength(50);
  });

  it('computeChurnFromCommits counts commits per file', async () => {
    const commits = [
      { files: ['a.py', 'b.py'] },
      { files: ['a.py', 'c.py'] },
      { files: ['a.py'] },
    ];

    const churn = await computeChurnFromCommits(commits as any);
    expect(churn.get('a.py')).toBe(3);
    expect(churn.get('b.py')).toBe(1);
    expect(churn.get('c.py')).toBe(1);
  });
});
