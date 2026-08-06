import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { mkdtemp, cp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { execa } from 'execa';
import { runAnalyze } from '../../scripts/orchestrator.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const fixturesRoot = join(__dirname, '..', 'fixtures');

async function copyFixture(name: string): Promise<string> {
  const target = await mkdtemp(join(tmpdir(), `archr-${name}-`));
  await cp(join(fixturesRoot, name), target, { recursive: true });
  await execa('git', ['init', '-q'], { cwd: target });
  await execa('git', ['config', 'user.email', 'test@example.com'], { cwd: target });
  await execa('git', ['config', 'user.name', 'Test'], { cwd: target });
  await execa('git', ['add', '.'], { cwd: target });
  await execa('git', ['commit', '-q', '-m', 'initial'], { cwd: target });
  return target;
}

describe('L1 smoke — rotten Python mini', () => {
  let dir: string;
  beforeAll(async () => { dir = await copyFixture('rotten-python-mini'); }, 30_000);
  afterAll(async () => { await rm(dir, { recursive: true, force: true }); });

  it('runs the full analyze pipeline on a Python repo', async () => {
    const result = await runAnalyze({
      rootPath: dir,
      output: join(dir, 'report.md'),
      noBaseline: true,
      mode: 'deep',
      resume: false,
      disabledToolIds: new Set(),
      parallelism: 4,
    });
    expect(result.toolRuns.length).toBeGreaterThan(0);
    // The pipeline must complete without exception; individual adapters may skip if tools aren't installable.
  }, 600_000);
});

describe('L1 smoke — rotten Go mini', () => {
  let dir: string;
  beforeAll(async () => { dir = await copyFixture('rotten-go-mini'); }, 30_000);
  afterAll(async () => { await rm(dir, { recursive: true, force: true }); });

  it('runs the full analyze pipeline on a Go repo', async () => {
    const result = await runAnalyze({
      rootPath: dir,
      output: join(dir, 'report.md'),
      noBaseline: true,
      mode: 'deep',
      resume: false,
      disabledToolIds: new Set(),
      parallelism: 4,
    });
    expect(result.toolRuns.length).toBeGreaterThan(0);
  }, 600_000);
});
