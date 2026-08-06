import pLimit from 'p-limit';
import { relative } from 'node:path';
import type { RepoProfile } from '../types.js';
import { indexFiles, type IndexFileInput } from '../l2-graph/tree-sitter-index.js';
import { partition } from './partition.js';

export async function chunkedIndex(profile: RepoProfile, files: IndexFileInput[], parallelism: number): Promise<ReturnType<typeof indexFiles> extends Promise<infer R> ? R : never> {
  const byChunk = new Map<string, IndexFileInput[]>();
  for (const f of files) {
    const rel = relative(profile.rootPath, f.path);
    const key = rel.includes('/') ? rel.split('/')[0] : '<root>';
    if (!byChunk.has(key)) byChunk.set(key, []);
    byChunk.get(key)!.push(f);
  }
  const limit = pLimit(parallelism);
  const results = await Promise.all([...byChunk.values()].map((group) => limit(() => indexFiles(group))));
  return {
    symbols: results.flatMap((r) => r.symbols),
    imports: results.flatMap((r) => r.imports),
  };
}
