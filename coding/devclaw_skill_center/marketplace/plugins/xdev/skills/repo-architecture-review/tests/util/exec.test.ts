// tests/util/exec.test.ts
import { describe, it, expect } from 'vitest';
import { runCmd } from '../../scripts/util/exec.js';

describe('runCmd', () => {
  it('captures stdout on success', async () => {
    const r = await runCmd('echo', ['hello'], { timeoutMs: 5000 });
    expect(r.status).toBe('ok');
    expect(r.stdout.trim()).toBe('hello');
  });

  it('reports non-zero exit as failed', async () => {
    const r = await runCmd('sh', ['-c', 'exit 7'], { timeoutMs: 5000 });
    expect(r.status).toBe('failed');
    expect(r.exitCode).toBe(7);
  });

  it('reports timeout', async () => {
    const r = await runCmd('sh', ['-c', 'sleep 5'], { timeoutMs: 200 });
    expect(r.status).toBe('timeout');
  });
});
