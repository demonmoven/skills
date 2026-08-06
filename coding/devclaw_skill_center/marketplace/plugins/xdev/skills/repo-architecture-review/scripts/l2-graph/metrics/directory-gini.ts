export function computeDirectoryGini(locByDir: Map<string, number>): number {
  if (locByDir.size === 0) {
    return 0;
  }

  // Sort by value ascending
  const values = Array.from(locByDir.values()).sort((a, b) => a - b);
  const n = values.length;

  // Compute total
  const total = values.reduce((sum, v) => sum + v, 0);
  if (total === 0) {
    return 0;
  }

  // Gini formula: (2 * Σ(i+1)*v[i]) / (n * total) - (n+1)/n
  let sum = 0;
  for (let i = 0; i < n; i++) {
    sum += (i + 1) * values[i];
  }

  const gini = (2 * sum) / (n * total) - (n + 1) / n;
  return Math.max(0, gini); // Clamp to [0, 1]
}
