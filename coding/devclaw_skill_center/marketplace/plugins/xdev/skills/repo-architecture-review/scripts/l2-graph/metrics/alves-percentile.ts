export interface WeightedSample {
  value: number;
  weight: number;
}

export interface AlvesThresholds {
  p70: number;
  p80: number;
  p90: number;
  p95: number;
  p99: number;
  n: number;
}

export function computeAlvesThresholds(samples: WeightedSample[]): AlvesThresholds {
  if (samples.length === 0) {
    return { p70: 0, p80: 0, p90: 0, p95: 0, p99: 0, n: 0 };
  }

  // Sort by value ascending
  const sorted = [...samples].sort((a, b) => a.value - b.value);

  // Compute total weight
  const totalWeight = sorted.reduce((sum, s) => sum + s.weight, 0);

  // Walk the weighted CDF once and record the first sample whose cumulative
  // fraction crosses each target. `captured` acts as the "once only" flag so a
  // legitimately-zero threshold (happens when most samples share the lowest
  // value, e.g. PageRank on sparse graphs) is not overwritten by later samples.
  const targets = [0.7, 0.8, 0.9, 0.95, 0.99];
  const values = new Array<number>(targets.length).fill(0);
  const captured = new Array<boolean>(targets.length).fill(false);

  let cumulative = 0;
  for (const sample of sorted) {
    cumulative += sample.weight;
    const fraction = cumulative / totalWeight;
    for (let i = 0; i < targets.length; i++) {
      if (!captured[i] && fraction >= targets[i]) {
        values[i] = sample.value;
        captured[i] = true;
      }
    }
  }

  return {
    p70: values[0],
    p80: values[1],
    p90: values[2],
    p95: values[3],
    p99: values[4],
    n: samples.length,
  };
}
