import { describe, it, expect } from 'vitest';
import {
  computeAlvesThresholds,
  type WeightedSample,
  type AlvesThresholds,
} from '../../scripts/l2-graph/metrics/alves-percentile.js';

describe('alves-percentile', () => {
  it('computes LOC-weighted 70/80/90 percentiles correctly', () => {
    const samples: WeightedSample[] = [
      { value: 10, weight: 100 },
      { value: 20, weight: 200 },
      { value: 30, weight: 300 },
      { value: 40, weight: 400 },
      { value: 50, weight: 500 },
    ];

    const thresholds = computeAlvesThresholds(samples);

    expect(thresholds.p70).toBeLessThanOrEqual(thresholds.p80);
    expect(thresholds.p80).toBeLessThanOrEqual(thresholds.p90);
    expect(thresholds.p70).toBeGreaterThan(0);
    expect(thresholds.n).toBe(5);
  });

  it('respects big-weight samples dominance', () => {
    const samples: WeightedSample[] = [
      { value: 100, weight: 1 }, // small weight
      { value: 10, weight: 1000 }, // large weight dominates
    ];

    const thresholds = computeAlvesThresholds(samples);

    // Most of the cumulative weight is from the second sample (1000 out of 1001)
    expect(thresholds.p70).toBeGreaterThanOrEqual(10);
    expect(thresholds.p70).toBeLessThan(100);
  });

  it('handles empty samples gracefully', () => {
    const thresholds = computeAlvesThresholds([]);
    expect(thresholds.p70).toBe(0);
    expect(thresholds.p80).toBe(0);
    expect(thresholds.p90).toBe(0);
    expect(thresholds.n).toBe(0);
  });
});
