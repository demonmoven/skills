// scripts/l0-prescan/detectors/gha.ts
import { relative } from 'node:path';
import { walkRepo } from '../../util/fs.js';

export async function detectGha(root: string): Promise<string[]> {
  const all = await walkRepo(root);
  return all
    .filter((f) => /\.github\/workflows\/[^/]+\.(ya?ml)$/.test(f))
    .map((f) => relative(root, f))
    .sort();
}
