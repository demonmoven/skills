import { describe, it, expect } from 'vitest';
import { mkdtemp, cp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { execa } from 'execa';
import { runAnalyze } from '../../scripts/orchestrator.js';

const __dirname = dirname(fileURLToPath(import.meta.url));

describe('clean fixtures — low FP', () => {
  it('clean-python produces no critical findings', async () => {
    const dir = await mkdtemp(join(tmpdir(), 'cp-'));
    await cp(join(__dirname, '..', 'fixtures', 'clean-python'), dir, { recursive: true });
    await execa('git', ['init', '-q'], { cwd: dir });
    await execa('git', ['config', 'user.email', 't@x.com'], { cwd: dir });
    await execa('git', ['config', 'user.name', 't'], { cwd: dir });
    await execa('git', ['add', '.'], { cwd: dir });
    await execa('git', ['commit', '-q', '-m', 'init'], { cwd: dir });
    process.env.ARCHREVIEW_NO_LLM = '1';
    const r = await runAnalyze({
      rootPath: dir, output: join(dir, 'report.md'),
      noBaseline: true, mode: 'quick', resume: false,
      disabledToolIds: new Set(), parallelism: 4,
    });
    // Clean fixture should have minimal findings; check that there are no critical ones
    const criticals = r.reportedFindings.filter((f) => f.severity === 'critical');
    expect(criticals).toHaveLength(0);
    // Verify pipeline completed
    expect(Array.isArray(r.reportedFindings)).toBe(true);
    await rm(dir, { recursive: true, force: true });
  }, 600_000);
});
