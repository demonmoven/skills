import type { RepoProfile } from '../types.js';

export interface Chunk { key: string; files: string[]; }

export function shouldChunk(profile: RepoProfile): boolean {
  return profile.totalFiles > 5000 || profile.workspaces.length > 10 || profile.gitLogLineEstimate > 500_000;
}

export function partition(files: string[]): Chunk[] {
  const byDir = new Map<string, string[]>();
  for (const f of files) {
    const key = f.includes('/') ? f.split('/')[0] : '<root>';
    if (!byDir.has(key)) byDir.set(key, []);
    byDir.get(key)!.push(f);
  }
  return [...byDir.entries()].map(([key, fs]) => ({ key, files: fs }));
}
