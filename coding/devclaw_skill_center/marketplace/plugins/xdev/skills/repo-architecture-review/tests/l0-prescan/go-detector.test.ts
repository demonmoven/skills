// tests/l0-prescan/go-detector.test.ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtemp, writeFile, mkdir, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { detectGo } from '../../scripts/l0-prescan/detectors/go.js';

describe('detectGo', () => {
  let dir: string;
  beforeEach(async () => { dir = await mkdtemp(join(tmpdir(), 'gd-')); });
  afterEach(async () => { await rm(dir, { recursive: true, force: true }); });

  it('returns empty when no go.mod', async () => {
    const r = await detectGo(dir);
    expect(r.hasGoMod).toBe(false);
    expect(r.topLevelPackages).toEqual([]);
  });

  it('detects go.mod and enumerates top packages', async () => {
    await writeFile(join(dir, 'go.mod'), 'module example.com/myapp\n\ngo 1.22\n');
    for (const p of ['internal/domain', 'internal/infra', 'cmd/api', 'pkg/shared']) {
      await mkdir(join(dir, p), { recursive: true });
      await writeFile(join(dir, p, 'x.go'), 'package x\n');
    }
    const r = await detectGo(dir);
    expect(r.hasGoMod).toBe(true);
    expect(r.modulePath).toBe('example.com/myapp');
    expect(r.topLevelPackages.sort()).toEqual(['cmd/api', 'internal/domain', 'internal/infra', 'pkg/shared']);
  });
});
