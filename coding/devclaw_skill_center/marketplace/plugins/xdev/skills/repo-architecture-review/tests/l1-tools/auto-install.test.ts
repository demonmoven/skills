// tests/l1-tools/auto-install.test.ts
import { describe, it, expect, vi } from 'vitest';
import { ensureBinary } from '../../scripts/l1-tools/auto-install.js';
import * as execMod from '../../scripts/util/exec.js';

describe('ensureBinary', () => {
  it('returns pre-existing when which succeeds', async () => {
    vi.spyOn(execMod, 'runCmd').mockResolvedValueOnce({
      status: 'ok', stdout: '/usr/local/bin/foo\n', stderr: '', exitCode: 0, durationMs: 1,
    });
    const r = await ensureBinary({ binary: 'foo', installPlan: [{ via: 'pipx', pkg: 'foo' }] });
    expect(r.ready).toBe(true);
    expect(r.installedVia).toBe('pre-existing');
    vi.restoreAllMocks();
  });

  it('attempts install when binary missing and reports pipx on success', async () => {
    const spy = vi.spyOn(execMod, 'runCmd')
      .mockResolvedValueOnce({ status: 'failed', stdout: '', stderr: '', exitCode: 1, durationMs: 1 })
      .mockResolvedValueOnce({ status: 'ok', stdout: 'installed', stderr: '', exitCode: 0, durationMs: 10 })
      .mockResolvedValueOnce({ status: 'ok', stdout: '/usr/local/bin/foo\n', stderr: '', exitCode: 0, durationMs: 1 });
    const r = await ensureBinary({ binary: 'foo', installPlan: [{ via: 'pipx', pkg: 'foo' }] });
    expect(r.ready).toBe(true);
    expect(r.installedVia).toBe('pipx');
    expect(spy).toHaveBeenCalledTimes(3);
    vi.restoreAllMocks();
  });

  it('returns not ready if all install plans fail', async () => {
    vi.spyOn(execMod, 'runCmd')
      .mockResolvedValueOnce({ status: 'failed', stdout: '', stderr: '', exitCode: 1, durationMs: 1 })
      .mockResolvedValueOnce({ status: 'failed', stdout: '', stderr: 'pipx not found', exitCode: 1, durationMs: 1 });
    const r = await ensureBinary({ binary: 'foo', installPlan: [{ via: 'pipx', pkg: 'foo' }] });
    expect(r.ready).toBe(false);
    expect(r.reason).toBe('install-failed');
    vi.restoreAllMocks();
  });
});
