import { readFile, writeFile, mkdir } from 'node:fs/promises';
import { join } from 'node:path';
import type { Finding, Severity } from '../types.js';
import { fileExists } from '../util/fs.js';

export interface Baseline {
  createdAt: string;
  findings: Finding[];
}

const severityOrder: Record<Severity, number> = { low: 1, medium: 2, high: 3, critical: 4 };

export async function loadBaseline(repoRoot: string): Promise<Baseline | null> {
  const path = join(repoRoot, '.architecture-review', 'baseline.json');
  if (!(await fileExists(path))) return null;
  return JSON.parse(await readFile(path, 'utf-8')) as Baseline;
}

export async function saveBaseline(repoRoot: string, b: Baseline): Promise<void> {
  const dir = join(repoRoot, '.architecture-review');
  await mkdir(dir, { recursive: true });
  await writeFile(join(dir, 'baseline.json'), JSON.stringify(b, null, 2));
}

export async function diffBaseline(repoRoot: string, current: Finding[]): Promise<{ newOrWorse: Finding[]; unchanged: Finding[] }> {
  const baseline = await loadBaseline(repoRoot);
  if (!baseline) return { newOrWorse: current, unchanged: [] };
  const baselineMap = new Map(baseline.findings.map((f) => [f.id, f]));
  const newOrWorse: Finding[] = [];
  const unchanged: Finding[] = [];
  for (const c of current) {
    const prev = baselineMap.get(c.id);
    if (!prev) newOrWorse.push(c);
    else if (severityOrder[c.severity] > severityOrder[prev.severity]) newOrWorse.push(c);
    else unchanged.push(c);
  }
  return { newOrWorse, unchanged };
}
