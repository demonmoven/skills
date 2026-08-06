import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtemp, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { indexFiles } from '../../scripts/l2-graph/tree-sitter-index.js';

describe('tree-sitter index', () => {
  let dir: string;
  beforeEach(async () => { dir = await mkdtemp(join(tmpdir(), 'tsi-')); });
  afterEach(async () => { await rm(dir, { recursive: true, force: true }); });

  it('extracts symbols from Python files', async () => {
    const file = join(dir, 'm.py');
    await writeFile(file, 'def foo():\n    return 1\n\nclass Bar:\n    pass\n');
    const index = await indexFiles([{ path: file, lang: 'python' }]);
    const names = index.symbols.map((s) => s.name).sort();
    expect(names).toContain('foo');
    expect(names).toContain('Bar');
  });

  it('extracts symbols from Go files', async () => {
    const file = join(dir, 'm.go');
    await writeFile(file, 'package main\n\nfunc Foo() int { return 1 }\n\ntype Bar struct{}\n');
    const index = await indexFiles([{ path: file, lang: 'go' }]);
    const names = index.symbols.map((s) => s.name).sort();
    expect(names).toContain('Foo');
    expect(names).toContain('Bar');
  });
});
