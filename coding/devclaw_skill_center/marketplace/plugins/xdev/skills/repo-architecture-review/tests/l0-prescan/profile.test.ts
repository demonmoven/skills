// tests/l0-prescan/profile.test.ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtemp, writeFile, mkdir, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { buildRepoProfile } from '../../scripts/l0-prescan/index.js';

describe('buildRepoProfile', () => {
  let dir: string;
  beforeEach(async () => { dir = await mkdtemp(join(tmpdir(), 'rp-')); });
  afterEach(async () => { await rm(dir, { recursive: true, force: true }); });

  it('builds profile for Go + Python + Docker + GHA monorepo', async () => {
    await writeFile(join(dir, 'go.mod'), 'module x\ngo 1.22\n');
    await mkdir(join(dir, 'internal', 'domain'), { recursive: true });
    await writeFile(join(dir, 'internal', 'domain', 'a.go'), 'package domain\n');

    await writeFile(join(dir, 'pyproject.toml'), '[project]\nname="x"\n');
    await mkdir(join(dir, 'mypkg'), { recursive: true });
    await writeFile(join(dir, 'mypkg', '__init__.py'), '');
    await writeFile(join(dir, 'mypkg', 'm.py'), 'x=1\n');

    await writeFile(join(dir, 'Dockerfile'), 'FROM alpine\n');
    await mkdir(join(dir, '.github', 'workflows'), { recursive: true });
    await writeFile(join(dir, '.github', 'workflows', 'ci.yml'), 'on: push\n');

    const p = await buildRepoProfile(dir);
    expect(p.hasGoMod).toBe(true);
    expect(p.hasPyProject).toBe(true);
    expect(p.dockerfiles).toEqual(['Dockerfile']);
    expect(p.ghaWorkflows).toEqual(['.github/workflows/ci.yml']);
    const langs = p.languages.map((l) => l.lang).sort();
    expect(langs).toEqual(['go', 'python']);
    expect(p.totalFiles).toBeGreaterThan(0);
  });
});
