import { relative, join, dirname } from 'node:path';
import { readFile } from 'node:fs/promises';
import type { RepoProfile } from '../types.js';
import { walkRepo } from '../util/fs.js';
import { UnifiedGraph } from './schema.js';
import { indexFiles } from './tree-sitter-index.js';
import { loadCommitsFromGit, computeCoChangeFromLog, DEFAULT_CC_OPTS } from '../l1-tools/adapters/code-maat-builtin.js';
import { shouldChunk, type Chunk } from '../mapreduce/partition.js';
import { chunkedIndex } from '../mapreduce/index.js';

async function readGoModulePath(rootPath: string): Promise<string | undefined> {
  try {
    const txt = await readFile(join(rootPath, 'go.mod'), 'utf-8');
    const m = txt.match(/^\s*module\s+(\S+)/m);
    return m ? m[1] : undefined;
  } catch { return undefined; }
}

export async function buildUnifiedGraph(profile: RepoProfile): Promise<UnifiedGraph> {
  const g = new UnifiedGraph();
  const allFiles = await walkRepo(profile.rootPath);
  const indexed: Array<{ path: string; lang: 'go' | 'python' }> = [];

  for (const f of allFiles) {
    if (f.endsWith('.go')) indexed.push({ path: f, lang: 'go' });
    else if (f.endsWith('.py')) indexed.push({ path: f, lang: 'python' });
  }

  for (const f of indexed) {
    const rel = relative(profile.rootPath, f.path);
    g.addFile(rel, { lang: f.lang, loc: 0 });
  }

  const idx = shouldChunk(profile) ? await chunkedIndex(profile, indexed, 4) : await indexFiles(indexed);

  for (const s of idx.symbols) {
    if (s.isReference) continue;
    const rel = relative(profile.rootPath, s.file);
    let kind = s.kind;
    if (kind === 'method') kind = 'function';
    g.addSymbol(`${rel}:${s.name}`, { file: rel, line: s.line, kind, name: s.name });
  }

  // Resolve imports to files.
  // - Python: convert dotted module path to a slash path + .py suffix, match any file
  //   whose relative path ends in that suffix.
  // - Go: Go imports a whole *package* (directory), not a single file. Strip the go.mod
  //   module prefix, then emit an edge from the importing file to every non-test .go
  //   file in that directory. Without this, the naive `last-segment + .go` heuristic
  //   only matches one file per directory, marking the rest of the package as orphans.
  const goModulePath = await readGoModulePath(profile.rootPath);
  const relByPath = new Map<string, string>();        // absPath -> relPath
  const filesByDir = new Map<string, string[]>();     // relDir  -> [relPath, ...]
  for (const f of indexed) {
    const rel = relative(profile.rootPath, f.path);
    relByPath.set(f.path, rel);
    const d = dirname(rel);
    const arr = filesByDir.get(d) ?? [];
    arr.push(rel);
    filesByDir.set(d, arr);
  }

  for (const i of idx.imports) {
    const fromRel = relative(profile.rootPath, i.fromFile);
    const mod = i.toModule;

    // Python: dotted module path → slash path + .py, or a package __init__.py.
    if (mod.includes('.') && !mod.startsWith(goModulePath ?? '__never__')) {
      const candidate = mod.replace(/\./g, '/') + '.py';
      const matchFile = indexed.find((f) => {
        const rel = relByPath.get(f.path)!;
        return rel.endsWith(candidate) || rel.endsWith(candidate.replace(/\.py$/, '/__init__.py'));
      });
      if (matchFile) g.addImport(fromRel, relByPath.get(matchFile.path)!);
      continue;
    }

    // Go: resolve to a directory under the module root, then every non-test .go file in it.
    let dir: string | undefined;
    if (goModulePath && mod === goModulePath) dir = '.';
    else if (goModulePath && mod.startsWith(goModulePath + '/')) dir = mod.slice(goModulePath.length + 1);

    if (dir !== undefined) {
      const pkgFiles = filesByDir.get(dir) ?? [];
      for (const rel of pkgFiles) {
        if (rel.endsWith('.go') && !rel.endsWith('_test.go')) {
          g.addImport(fromRel, rel);
        }
      }
      continue;
    }

    // Third-party / stdlib Go packages: no in-repo resolution.
  }

  try {
    const commits = await loadCommitsFromGit(profile.rootPath, DEFAULT_CC_OPTS.sinceMonths);
    const pairs = computeCoChangeFromLog(commits, DEFAULT_CC_OPTS);
    for (const p of pairs) {
      if (g.raw().hasNode(p.a) && g.raw().hasNode(p.b)) {
        g.addCoChange(p.a, p.b, p.shared);
      }
    }
  } catch { /* git not available */ }

  return g;
}
