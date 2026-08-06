// scripts/l0-prescan/detectors/docker.ts
import { relative } from 'node:path';
import { walkRepo } from '../../util/fs.js';

export async function detectDocker(root: string): Promise<string[]> {
  const all = await walkRepo(root);
  return all
    .filter((f) => {
      const name = f.split('/').pop() ?? '';
      return name === 'Dockerfile' || name.startsWith('Dockerfile.');
    })
    .map((f) => relative(root, f))
    .sort();
}
