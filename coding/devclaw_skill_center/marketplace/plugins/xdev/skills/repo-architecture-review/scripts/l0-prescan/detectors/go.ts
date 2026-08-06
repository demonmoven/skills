// scripts/l0-prescan/detectors/go.ts
import { readFile } from 'node:fs/promises';
import { join, relative, dirname } from 'node:path';
import { fileExists, walkRepo } from '../../util/fs.js';

export interface GoDetection {
  hasGoMod: boolean;
  modulePath?: string;
  topLevelPackages: string[];
  fileCount: number;
  locTotal: number;
}

export async function detectGo(root: string): Promise<GoDetection> {
  const goMod = join(root, 'go.mod');
  if (!(await fileExists(goMod))) {
    return { hasGoMod: false, topLevelPackages: [], fileCount: 0, locTotal: 0 };
  }
  const modContent = await readFile(goMod, 'utf-8');
  const modulePath = modContent.match(/^module\s+(\S+)/m)?.[1];

  const allFiles = await walkRepo(root);
  const goFiles = allFiles.filter((f) => f.endsWith('.go'));

  const packageDirs = new Set<string>();
  let locTotal = 0;
  for (const f of goFiles) {
    const rel = relative(root, f);
    const parts = rel.split('/');
    if (parts.length >= 2) packageDirs.add(parts.slice(0, 2).join('/'));
    else packageDirs.add(dirname(rel));
    try {
      const content = await readFile(f, 'utf-8');
      locTotal += content.split('\n').length;
    } catch { /* ignore */ }
  }

  return {
    hasGoMod: true,
    modulePath,
    topLevelPackages: [...packageDirs].sort(),
    fileCount: goFiles.length,
    locTotal,
  };
}
