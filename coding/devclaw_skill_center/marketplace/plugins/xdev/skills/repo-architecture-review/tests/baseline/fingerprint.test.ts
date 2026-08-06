import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtemp, writeFile, mkdir, rm, rename } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { execa } from 'execa';
import { resolveRenames } from '../../scripts/baseline/fingerprint.js';

describe('resolveRenames', () => {
  let dir: string;
  beforeEach(async () => {
    dir = await mkdtemp(join(tmpdir(), 'fp-'));
    await execa('git', ['init', '-q'], { cwd: dir });
    await execa('git', ['config', 'user.email', 't@x.com'], { cwd: dir });
    await execa('git', ['config', 'user.name', 't'], { cwd: dir });
    await mkdir(join(dir, 'src'), { recursive: true });
    await writeFile(join(dir, 'src', 'old.py'), 'x = 1\n');
    await execa('git', ['add', '.'], { cwd: dir });
    await execa('git', ['commit', '-q', '-m', 'init'], { cwd: dir });
    await execa('git', ['mv', 'src/old.py', 'src/new.py'], { cwd: dir });
    await execa('git', ['commit', '-q', '-m', 'rename'], { cwd: dir });
  });
  afterEach(async () => { await rm(dir, { recursive: true, force: true }); });

  it('maps the old path to the new path', async () => {
    const map = await resolveRenames(dir, ['src/old.py']);
    expect(map.get('src/old.py')).toBe('src/new.py');
  });
});
