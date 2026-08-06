import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtemp, writeFile, mkdir, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { execa } from 'execa';
import { buildUnifiedGraph } from '../../scripts/l2-graph/index.js';
import { buildRepoProfile } from '../../scripts/l0-prescan/index.js';

describe('buildUnifiedGraph', () => {
  let dir: string;
  beforeEach(async () => { dir = await mkdtemp(join(tmpdir(), 'ug-')); });
  afterEach(async () => { await rm(dir, { recursive: true, force: true }); });

  it('builds graph for Python mini', async () => {
    await writeFile(join(dir, 'pyproject.toml'), '[project]\nname="x"\n');
    await mkdir(join(dir, 'mypkg'), { recursive: true });
    await writeFile(join(dir, 'mypkg', '__init__.py'), '');
    await writeFile(join(dir, 'mypkg', 'a.py'), 'def foo(): pass\n');
    await writeFile(join(dir, 'mypkg', 'b.py'), 'from mypkg.a import foo\nfoo()\n');
    await execa('git', ['init', '-q'], { cwd: dir });
    await execa('git', ['config', 'user.email', 't@x.com'], { cwd: dir });
    await execa('git', ['config', 'user.name', 't'], { cwd: dir });
    await execa('git', ['add', '.'], { cwd: dir });
    await execa('git', ['commit', '-q', '-m', 'init'], { cwd: dir });

    const profile = await buildRepoProfile(dir);
    const g = await buildUnifiedGraph(profile);
    const files = g.files();
    expect(files.some((f) => f.endsWith('mypkg/a.py'))).toBe(true);
    expect(files.some((f) => f.endsWith('mypkg/b.py'))).toBe(true);
  }, 60_000);
});
