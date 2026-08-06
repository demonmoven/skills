import { describe, it, expect } from 'vitest';
import { hashRuleSource, runAdapterSafely, type ToolAdapter, type AdapterContext, type PreflightResult, type RawRunOutput } from '../../scripts/l1-tools/adapter.js';
import type { RepoProfile } from '../../scripts/types.js';

const profile: RepoProfile = {
  rootPath: '/tmp/x',
  languages: [{ lang: 'python', fileCount: 10, locTotal: 100 }],
  hasGoMod: false, hasPyProject: true, hasSetupCfg: false, hasRequirementsTxt: false,
  dockerfiles: [], ghaWorkflows: [], topLevelPackages: {}, workspaces: [],
  totalFiles: 10, totalLocMillions: 0.0001, gitLogLineEstimate: 0,
  detectedConfigs: {},
};

const ctx: AdapterContext = { profile, tmpDir: '/tmp/tmp-x', rulesFromBuiltin: true, timeoutMs: 5000, disabledToolIds: new Set() };

describe('hashRuleSource', () => {
  it('is stable', () => {
    expect(hashRuleSource('a', 'r', 'f.py')).toBe(hashRuleSource('a', 'r', 'f.py'));
    expect(hashRuleSource('a', 'r', 'f.py')).not.toBe(hashRuleSource('a', 'r', 'g.py'));
  });
});

describe('runAdapterSafely', () => {
  it('skips when detectApplies returns false', async () => {
    const adapter: ToolAdapter = {
      id: 'fake',
      detectApplies: () => false,
      preflight: async () => ({ ready: false }),
      run: async () => { throw new Error('should not run'); },
      normalize: () => [],
    };
    const r = await runAdapterSafely(adapter, ctx);
    expect(r.status).toBe('skipped');
    expect(r.reason).toBe('not-applicable');
  });

  it('skips when adapter is disabled via ctx.disabledToolIds', async () => {
    const adapter: ToolAdapter = {
      id: 'fake',
      detectApplies: () => true,
      preflight: async () => { throw new Error('should not run'); },
      run: async () => { throw new Error('should not run'); },
      normalize: () => [],
    };
    const disabledCtx: AdapterContext = { ...ctx, disabledToolIds: new Set(['fake']) };
    const r = await runAdapterSafely(adapter, disabledCtx);
    expect(r.status).toBe('skipped');
    expect(r.reason).toBe('no-binary');
  });

  it('skips when preflight returns not ready', async () => {
    const adapter: ToolAdapter = {
      id: 'fake',
      detectApplies: () => true,
      preflight: async (): Promise<PreflightResult> => ({ ready: false, reason: 'no-binary' }),
      run: async () => { throw new Error('should not run'); },
      normalize: () => [],
    };
    const r = await runAdapterSafely(adapter, ctx);
    expect(r.status).toBe('skipped');
    expect(r.reason).toBe('no-binary');
  });

  it('reports failed when run throws', async () => {
    const adapter: ToolAdapter = {
      id: 'fake',
      detectApplies: () => true,
      preflight: async () => ({ ready: true, installedVia: 'pre-existing' }),
      run: async () => { throw new Error('boom'); },
      normalize: () => [],
    };
    const r = await runAdapterSafely(adapter, ctx);
    expect(r.status).toBe('failed');
    expect(r.error).toContain('boom');
  });

  it('returns ok when run + normalize succeed', async () => {
    const adapter: ToolAdapter = {
      id: 'fake',
      detectApplies: () => true,
      preflight: async () => ({ ready: true, installedVia: 'pre-existing' }),
      run: async (): Promise<RawRunOutput> => ({ rawOutputPath: '/tmp/raw', exitCode: 0, stdout: '[]', stderr: '', durationMs: 1 }),
      normalize: () => [{ id: 'x', source: 'l1-tool:fake', category: 'dependency', ruleId: 'r', ruleSource: 'skill-builtin', severity: 'low', locations: [{ file: 'a.py' }], evidence: {}, message: 'm' }],
    };
    const r = await runAdapterSafely(adapter, ctx);
    expect(r.status).toBe('ok');
    expect(r.violations).toHaveLength(1);
  });
});
