// tests/l0-prescan/python-detector.test.ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtemp, writeFile, mkdir, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { detectPython } from '../../scripts/l0-prescan/detectors/python.js';

describe('detectPython', () => {
  let dir: string;
  beforeEach(async () => { dir = await mkdtemp(join(tmpdir(), 'pyd-')); });
  afterEach(async () => { await rm(dir, { recursive: true, force: true }); });

  it('returns empty when no config', async () => {
    const r = await detectPython(dir);
    expect(r.hasPyProject).toBe(false);
    expect(r.hasSetupCfg).toBe(false);
    expect(r.topLevelPackages).toEqual([]);
  });

  it('detects pyproject.toml and enumerates top packages', async () => {
    await writeFile(join(dir, 'pyproject.toml'), '[project]\nname = "myapp"\n');
    for (const p of ['mypkg', 'mypkg/domain', 'mypkg/infra']) {
      await mkdir(join(dir, p), { recursive: true });
      await writeFile(join(dir, p, '__init__.py'), '');
      await writeFile(join(dir, p, 'mod.py'), 'x = 1\n');
    }
    const r = await detectPython(dir);
    expect(r.hasPyProject).toBe(true);
    expect(r.topLevelPackages).toContain('mypkg');
    expect(r.topLevelPackages).toContain('mypkg.domain');
  });

  it('detects importlinter config in pyproject.toml', async () => {
    await writeFile(join(dir, 'pyproject.toml'), '[tool.importlinter]\nroot_packages = ["mypkg"]\n');
    await mkdir(join(dir, 'mypkg'), { recursive: true });
    await writeFile(join(dir, 'mypkg', '__init__.py'), '');
    const r = await detectPython(dir);
    expect(r.hasImportLinterConfig).toBe(true);
  });
});
