import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { mkdtemp, cp, rm, readFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { execa } from 'execa';
import { runAnalyze } from '../../scripts/orchestrator.js';

const __dirname = dirname(fileURLToPath(import.meta.url));

async function prepFixture(name: string): Promise<string> {
  const dir = await mkdtemp(join(tmpdir(), `archr-${name}-`));
  await cp(join(__dirname, '..', 'fixtures', name), dir, { recursive: true });
  await execa('git', ['init', '-q'], { cwd: dir });
  await execa('git', ['config', 'user.email', 't@x.com'], { cwd: dir });
  await execa('git', ['config', 'user.name', 't'], { cwd: dir });
  await execa('git', ['add', '.'], { cwd: dir });
  await execa('git', ['commit', '-q', '-m', 'init'], { cwd: dir });
  for (let i = 0; i < 5; i++) {
    // Create a unique file each time to ensure git sees a change
    await execa('sh', ['-c', `echo "churn ${i}" > churn_${i}`], { cwd: dir });
    await execa('git', ['add', '.'], { cwd: dir });
    await execa('git', ['commit', '-q', '-m', `churn ${i}`], { cwd: dir });
  }
  return dir;
}

describe('rotten-python recall', () => {
  let dir: string;
  beforeAll(async () => { dir = await prepFixture('rotten-python'); }, 60_000);
  afterAll(async () => { await rm(dir, { recursive: true, force: true }); });

  it('detects findings and renders report sections', async () => {
    process.env.ARCHREVIEW_NO_LLM = '1';
    const r = await runAnalyze({
      rootPath: dir, output: join(dir, 'report.md'),
      noBaseline: true, mode: 'quick', resume: false,
      disabledToolIds: new Set(), parallelism: 4,
    });
    // Pipeline completes without exception
    expect(Array.isArray(r.reportedFindings)).toBe(true);
    const md = await readFile(r.reportPath, 'utf-8');
    expect(md).toContain('## 关键发现');
    expect(md).toContain('## 执行审计');
  }, 600_000);
});

describe('rotten-go recall', () => {
  let dir: string;
  beforeAll(async () => { dir = await prepFixture('rotten-go'); }, 60_000);
  afterAll(async () => { await rm(dir, { recursive: true, force: true }); });

  it('runs end-to-end on Go fixture', async () => {
    process.env.ARCHREVIEW_NO_LLM = '1';
    const r = await runAnalyze({
      rootPath: dir, output: join(dir, 'report.md'),
      noBaseline: true, mode: 'quick', resume: false,
      disabledToolIds: new Set(), parallelism: 4,
    });
    expect(Array.isArray(r.reportedFindings)).toBe(true);
  }, 600_000);
});
