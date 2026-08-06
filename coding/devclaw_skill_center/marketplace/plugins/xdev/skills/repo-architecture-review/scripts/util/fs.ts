// scripts/util/fs.ts
import { readFile, readdir, stat } from 'node:fs/promises';
import { join, relative, sep } from 'node:path';

const DEFAULT_IGNORES = new Set(['.git', 'node_modules', '__pycache__', 'dist', 'build', '.next', '.venv', 'venv', 'vendor']);

interface IgnoreRule { pattern: string; isDir: boolean; }

async function loadGitignore(root: string): Promise<IgnoreRule[]> {
  try {
    const content = await readFile(join(root, '.gitignore'), 'utf-8');
    return content
      .split('\n')
      .map((l) => l.trim())
      .filter((l) => l && !l.startsWith('#'))
      .map((l) => ({ pattern: l.replace(/\/$/, ''), isDir: l.endsWith('/') }));
  } catch {
    return [];
  }
}

function matches(relPath: string, rules: IgnoreRule[]): boolean {
  for (const r of rules) {
    // Simple glob: exact-match, prefix-match, or suffix-match via `*.ext`
    if (r.pattern.startsWith('*.')) {
      if (relPath.endsWith(r.pattern.slice(1))) return true;
    } else if (r.pattern.includes('/')) {
      if (relPath === r.pattern || relPath.startsWith(r.pattern + '/')) return true;
    } else {
      const parts = relPath.split(sep);
      if (parts.includes(r.pattern)) return true;
    }
  }
  return false;
}

export async function walkRepo(root: string): Promise<string[]> {
  const rules = await loadGitignore(root);
  const out: string[] = [];

  async function walk(dir: string): Promise<void> {
    let entries;
    try { entries = await readdir(dir, { withFileTypes: true }); } catch { return; }
    for (const e of entries) {
      const full = join(dir, e.name);
      const rel = relative(root, full);
      if (DEFAULT_IGNORES.has(e.name)) continue;
      if (matches(rel, rules)) continue;
      if (e.isDirectory()) {
        await walk(full);
      } else if (e.isFile()) {
        out.push(full);
      }
    }
  }

  await walk(root);
  return out;
}

export async function fileExists(path: string): Promise<boolean> {
  try { await stat(path); return true; } catch { return false; }
}

export async function globFiles(root: string, patterns: RegExp[]): Promise<string[]> {
  const all = await walkRepo(root);
  return all.filter((f) => {
    const rel = relative(root, f);
    return patterns.some((p) => p.test(rel));
  });
}
