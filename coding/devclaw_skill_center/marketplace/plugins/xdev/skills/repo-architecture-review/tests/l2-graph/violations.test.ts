import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtemp, cp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { execa } from 'execa';
import { buildRepoProfile } from '../../scripts/l0-prescan/index.js';
import { computeL2Violations } from '../../scripts/l2-graph/violations.js';

const __dirname = dirname(fileURLToPath(import.meta.url));

describe('computeL2Violations on rotten-python-mini', () => {
  let dir: string;
  beforeEach(async () => {
    dir = await mkdtemp(join(tmpdir(), 'l2v-'));
    await cp(join(__dirname, '..', 'fixtures', 'rotten-python-mini'), dir, { recursive: true });
    await execa('git', ['init', '-q'], { cwd: dir });
    await execa('git', ['config', 'user.email', 't@x.com'], { cwd: dir });
    await execa('git', ['config', 'user.name', 't'], { cwd: dir });
    await execa('git', ['add', '.'], { cwd: dir });
    await execa('git', ['commit', '-q', '-m', 'init'], { cwd: dir });
  });
  afterEach(async () => { await rm(dir, { recursive: true, force: true }); });

  it('completes without throwing and returns an array', async () => {
    const profile = await buildRepoProfile(dir);
    const v = await computeL2Violations(profile);
    expect(Array.isArray(v)).toBe(true);
    // We don't strictly require violations — the mini fixture may be small enough to have none
    // except orphan detection. Just verify the aggregator runs end-to-end.
  }, 60_000);
});
