import { describe, it, expect } from 'vitest';
import {
  computeBusFactor,
  type Ownership,
} from '../../scripts/l2-graph/metrics/bus-factor.js';

describe('bus-factor', () => {
  it('computes truck factor via greedy removal', () => {
    const ownerships: Ownership[] = [
      { file: 'a.py', author: 'alice', addedLines: 100 },
      { file: 'b.py', author: 'alice', addedLines: 100 },
      { file: 'c.py', author: 'bob', addedLines: 100 },
      { file: 'd.py', author: 'bob', addedLines: 100 },
      { file: 'e.py', author: 'charlie', addedLines: 100 },
    ];

    const result = computeBusFactor(ownerships);

    // alice owns a, b (2 files); bob owns c, d (2 files); charlie owns e (1 file)
    // 5 total, target unowned = ceil(5/2) = 3
    // After removing alice: unowned = {a, b} = 2 (< 3, continue)
    // After removing bob: unowned = {a, b, c, d} = 4 (>= 3, stop)
    expect(result.truckFactor).toBe(2);
    expect(result.authorsRemoved).toHaveLength(2);
    expect(result.orphansIfRemoved).toHaveLength(4);
  });

  it('identifies owner when > 50% of file lines added', () => {
    const ownerships: Ownership[] = [
      { file: 'a.py', author: 'alice', addedLines: 30 },
      { file: 'a.py', author: 'bob', addedLines: 70 }, // bob > 50%
      { file: 'b.py', author: 'bob', addedLines: 100 },
      { file: 'c.py', author: 'charlie', addedLines: 100 },
    ];

    const result = computeBusFactor(ownerships);

    // bob owns a and b (2 files); charlie owns c (1 file)
    // Remove bob: a, b become orphaned (2 files unowned)
    // 3 total files, target is ceil(3/2) = 2 unowned → truckFactor = 1
    expect(result.truckFactor).toBe(1);
    expect(result.orphansIfRemoved).toContain('a.py');
    expect(result.orphansIfRemoved).toContain('b.py');
  });
});
