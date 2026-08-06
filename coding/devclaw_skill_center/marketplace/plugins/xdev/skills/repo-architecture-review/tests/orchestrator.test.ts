import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtemp, writeFile, mkdir, rm, readFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { runAnalyze } from '../scripts/orchestrator.js';

describe('runAnalyze (skeleton — no adapters registered yet)', () => {
  let dir: string;
  beforeEach(async () => {
    dir = await mkdtemp(join(tmpdir(), 'orch-'));
    await writeFile(join(dir, 'go.mod'), 'module x\n');
    await mkdir(join(dir, 'cmd'), { recursive: true });
    await writeFile(join(dir, 'cmd', 'main.go'), 'package main\n');
  });
  afterEach(async () => { await rm(dir, { recursive: true, force: true }); });

  it('produces a report and persists profile + tool-runs artifacts', async () => {
    const result = await runAnalyze({
      rootPath: dir,
      output: join(dir, 'report.md'),
      noBaseline: true,
      mode: 'deep',
      resume: false,
      disabledToolIds: new Set(),
      parallelism: 2,
    });
    const report = await readFile(result.reportPath, 'utf-8');
    expect(report).toContain('架构体检报告');
    const profile = JSON.parse(await readFile(join(dir, '.architecture-review', 'tmp', 'repo-profile.json'), 'utf-8'));
    expect(profile.hasGoMod).toBe(true);
    const runs = JSON.parse(await readFile(join(dir, '.architecture-review', 'tmp', 'tool-runs.json'), 'utf-8'));
    expect(Array.isArray(runs)).toBe(true);
    expect(runs.every((r: {status: string}) => ['ok','skipped','failed'].includes(r.status))).toBe(true);
  });
});
