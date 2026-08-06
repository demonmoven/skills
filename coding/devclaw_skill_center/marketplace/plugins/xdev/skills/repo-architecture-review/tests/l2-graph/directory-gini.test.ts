import { describe, it, expect } from 'vitest';
import { computeDirectoryGini } from '../../scripts/l2-graph/metrics/directory-gini.js';

describe('directory-gini', () => {
  it('returns near 0 for evenly distributed LOC', () => {
    const locByDir = new Map([
      ['dir1', 1000],
      ['dir2', 1000],
      ['dir3', 1000],
      ['dir4', 1000],
    ]);

    const gini = computeDirectoryGini(locByDir);
    expect(gini).toBeLessThan(0.05);
  });

  it('returns > 0.7 for skewed distribution', () => {
    const locByDir = new Map([
      ['bigdir', 9000],
      ['small1', 100],
      ['small2', 100],
      ['small3', 100],
      ['small4', 100],
    ]);

    const gini = computeDirectoryGini(locByDir);
    expect(gini).toBeGreaterThan(0.7);
  });

  it('handles empty map gracefully', () => {
    const locByDir = new Map<string, number>();
    const gini = computeDirectoryGini(locByDir);
    expect(gini).toBe(0);
  });
});
