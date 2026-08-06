// tests/util/fs.test.ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtemp, writeFile, mkdir, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { walkRepo } from '../../scripts/util/fs.js';

describe('walkRepo', () => {
  let dir: string;

  beforeEach(async () => {
    dir = await mkdtemp(join(tmpdir(), 'archreview-'));
    await mkdir(join(dir, 'src'), { recursive: true });
    await writeFile(join(dir, 'src', 'a.py'), 'x = 1\n');
    await writeFile(join(dir, 'src', 'b.go'), 'package main\n');
    await mkdir(join(dir, 'node_modules', 'x'), { recursive: true });
    await writeFile(join(dir, 'node_modules', 'x', 'skip.js'), 'nope');
    await writeFile(join(dir, '.gitignore'), 'node_modules/\n*.log\n');
    await writeFile(join(dir, 'run.log'), 'skip');
  });

  afterEach(async () => {
    await rm(dir, { recursive: true, force: true });
  });

  it('returns source files respecting .gitignore', async () => {
    const files = await walkRepo(dir);
    const rel = files.map((f) => f.replace(dir + '/', '')).sort();
    expect(rel).toEqual(['.gitignore', 'src/a.py', 'src/b.go']);
  });
});
