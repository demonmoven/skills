export interface HotspotInput {
  file: string;
  churn: number;
  loc: number;
}

export interface Hotspot extends HotspotInput {
  score: number;
}

// Max-coverage mode: `top` defaults to Infinity so every hotspot is returned.
// Callers that want a capped list can still pass an explicit top = N.
export function computeHotspots(inputs: HotspotInput[], top: number = Number.POSITIVE_INFINITY): Hotspot[] {
  return inputs
    .map((input) => ({
      ...input,
      score: input.churn * input.loc,
    }))
    .sort((a, b) => b.score - a.score)
    .slice(0, top);
}

export async function computeChurnFromCommits(
  commits: Array<{ files: string[] }>,
): Promise<Map<string, number>> {
  const churn = new Map<string, number>();

  for (const commit of commits) {
    for (const file of commit.files) {
      const current = churn.get(file) ?? 0;
      churn.set(file, current + 1);
    }
  }

  return churn;
}
