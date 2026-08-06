import { describe, expect, it } from 'vitest';
import { runConcurrent } from './git-spawn.js';

describe('runConcurrent', () => {
  it('preserves input order in results', async () => {
    const tasks = [10, 30, 5, 20].map((delay, i) => async () => {
      await new Promise((r) => setTimeout(r, delay));
      return i;
    });
    const results = await runConcurrent(tasks, 4);
    expect(results.map((r) => (r.status === 'fulfilled' ? r.value : null))).toEqual([0, 1, 2, 3]);
  });

  it('caps concurrency to the parallelism arg', async () => {
    let active = 0;
    let peak = 0;
    const tasks = Array.from({ length: 20 }, () => async () => {
      active++;
      peak = Math.max(peak, active);
      await new Promise((r) => setTimeout(r, 10));
      active--;
      return 1;
    });
    await runConcurrent(tasks, 4);
    expect(peak).toBeLessThanOrEqual(4);
    expect(peak).toBeGreaterThan(0);
  });

  it('collects fulfilled and rejected results separately', async () => {
    const tasks = [
      async () => 'ok-1',
      async () => { throw new Error('boom-2'); },
      async () => 'ok-3',
    ];
    const results = await runConcurrent(tasks, 2);
    expect(results[0]).toEqual({ status: 'fulfilled', value: 'ok-1' });
    expect(results[1].status).toBe('rejected');
    if (results[1].status === 'rejected') {
      expect((results[1].reason as Error).message).toBe('boom-2');
    }
    expect(results[2]).toEqual({ status: 'fulfilled', value: 'ok-3' });
  });

  it('handles empty task list', async () => {
    const results = await runConcurrent([], 4);
    expect(results).toEqual([]);
  });
});
