// tests/l0-prescan/docker-gha.test.ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtemp, writeFile, mkdir, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { detectDocker } from '../../scripts/l0-prescan/detectors/docker.js';
import { detectGha } from '../../scripts/l0-prescan/detectors/gha.js';

describe('docker/gha detectors', () => {
  let dir: string;
  beforeEach(async () => { dir = await mkdtemp(join(tmpdir(), 'dg-')); });
  afterEach(async () => { await rm(dir, { recursive: true, force: true }); });

  it('detectDocker finds Dockerfiles', async () => {
    await writeFile(join(dir, 'Dockerfile'), 'FROM alpine\n');
    await mkdir(join(dir, 'images', 'api'), { recursive: true });
    await writeFile(join(dir, 'images', 'api', 'Dockerfile.api'), 'FROM alpine\n');
    const files = await detectDocker(dir);
    expect(files.sort()).toEqual(['Dockerfile', 'images/api/Dockerfile.api']);
  });

  it('detectGha finds workflow yaml files', async () => {
    await mkdir(join(dir, '.github', 'workflows'), { recursive: true });
    await writeFile(join(dir, '.github', 'workflows', 'ci.yml'), 'on: push\n');
    await writeFile(join(dir, '.github', 'workflows', 'deploy.yaml'), 'on: push\n');
    const files = await detectGha(dir);
    expect(files.sort()).toEqual(['.github/workflows/ci.yml', '.github/workflows/deploy.yaml']);
  });
});
