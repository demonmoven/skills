import type { Finding } from '../types.js';

// Passthrough / noise findings are already folded out of the main report; they
// should not drag the health score either, otherwise a repo with zero real
// issues but many orphan candidates would look artificially unhealthy.
function isNoise(f: Finding): boolean {
  return f.title.startsWith('[LOW-SIGNAL]') || (f.severity === 'low' && f.confidence === 'low');
}

export function computeHealthScore(findings: Finding[]): number {
  let score = 100;
  for (const f of findings) {
    if (isNoise(f)) continue;
    const sev = { low: 2, medium: 6, high: 12, critical: 25 }[f.severity];
    score -= sev * f.confidenceScore;
  }
  return Math.max(0, Math.round(score));
}
