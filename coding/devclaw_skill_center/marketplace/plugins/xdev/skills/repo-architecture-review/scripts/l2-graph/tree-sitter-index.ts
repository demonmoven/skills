import { readFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import Parser from 'tree-sitter';
import Go from 'tree-sitter-go';
import Python from 'tree-sitter-python';
import type { Language } from '../types.js';

const __dirname = dirname(fileURLToPath(import.meta.url));

export interface IndexFileInput { path: string; lang: Language; }

export interface IndexedSymbol {
  file: string;
  name: string;
  kind: 'function' | 'class' | 'method' | 'type' | 'module';
  line: number;
  isReference?: boolean;
  referenceTarget?: string;
}

export interface Index {
  symbols: IndexedSymbol[];
  imports: Array<{ fromFile: string; toModule: string; line: number }>;
}

const LANG_MAP: Record<string, { parser: unknown; queryFile: string }> = {
  python: { parser: Python, queryFile: 'tree-sitter-queries/python.scm' },
  go: { parser: Go, queryFile: 'tree-sitter-queries/go.scm' },
};

async function loadQueryText(relPath: string): Promise<string> {
  const fromHere = join(__dirname, relPath);
  try { return await readFile(fromHere, 'utf-8'); } catch { /* try source */ }
  // Fallback: look in source tree (for compiled dist/ paths).
  const fromSource = join(__dirname, '..', '..', 'scripts', 'l2-graph', relPath);
  return readFile(fromSource, 'utf-8');
}

export async function indexFiles(files: IndexFileInput[]): Promise<Index> {
  const symbols: IndexedSymbol[] = [];
  const imports: Index['imports'] = [];
  const parserCache = new Map<string, Parser>();
  const queryCache = new Map<string, Parser.Query>();

  for (const f of files) {
    const meta = LANG_MAP[f.lang];
    if (!meta) continue;
    let parser = parserCache.get(f.lang);
    if (!parser) {
      parser = new Parser();
      (parser as unknown as { setLanguage: (l: unknown) => void }).setLanguage(meta.parser);
      parserCache.set(f.lang, parser);
    }
    let query = queryCache.get(f.lang);
    if (!query) {
      const queryText = await loadQueryText(meta.queryFile);
      query = new Parser.Query(meta.parser as unknown, queryText);
      queryCache.set(f.lang, query);
    }
    let source: string;
    try { source = await readFile(f.path, 'utf-8'); } catch { continue; }
    let tree;
    try {
      // tree-sitter-node default bufferSize is 32KB; bump to 10MB to fit large source files.
      tree = (parser as unknown as { parse: (s: string, t?: unknown, o?: { bufferSize?: number }) => unknown }).parse(source, undefined, { bufferSize: 10 * 1024 * 1024 });
    } catch (e) {
      console.warn(`[tree-sitter-index] skip ${f.path}: ${(e as Error)?.message ?? e}`);
      continue;
    }
    const matches = (query as unknown as { matches: (n: unknown) => Array<{ captures: Array<{ name: string; node: { startPosition: { row: number }; text: string } }> }> }).matches((tree as { rootNode: unknown }).rootNode);
    for (const m of matches) {
      for (const cap of m.captures) {
        const tagName = cap.name;
        const line = cap.node.startPosition.row + 1;
        const text = cap.node.text;
        if (tagName.startsWith('name.definition.')) {
          const kind = tagName.split('.')[2] as IndexedSymbol['kind'];
          symbols.push({ file: f.path, name: text, kind, line });
        } else if (tagName === 'name.reference.import') {
          const mod = text.replace(/^["']|["']$/g, '');
          imports.push({ fromFile: f.path, toModule: mod, line });
        } else if (tagName.startsWith('name.reference.')) {
          symbols.push({ file: f.path, name: text, kind: 'function', line, isReference: true, referenceTarget: text });
        }
      }
    }
  }
  return { symbols, imports };
}
