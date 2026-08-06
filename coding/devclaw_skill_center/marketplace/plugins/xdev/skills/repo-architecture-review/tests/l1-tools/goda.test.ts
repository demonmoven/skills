import { describe, it, expect } from 'vitest';
import { godaAdapter } from '../../scripts/l1-tools/adapters/goda.js';
import type { RepoProfile } from '../../scripts/types.js';

const profile: RepoProfile = {
  rootPath: '/tmp/x',
  languages: [{ lang: 'go', fileCount: 20, locTotal: 400 }],
  hasGoMod: true,
  hasPyProject: false,
  hasSetupCfg: false,
  hasRequirementsTxt: false,
  dockerfiles: [],
  ghaWorkflows: [],
  topLevelPackages: { go: ['internal/domain'] },
  workspaces: [],
  totalFiles: 20,
  totalLocMillions: 0,
  gitLogLineEstimate: 0,
  detectedConfigs: {},
};

describe('goda adapter', () => {
  it('applies to Go projects', () => {
    expect(godaAdapter.detectApplies(profile)).toBe(true);
    expect(godaAdapter.detectApplies({ ...profile, hasGoMod: false })).toBe(false);
  });

  it('normalizes DOT output into build-time fan-in records (not violations by itself)', () => {
    const fanIn = new Map<string, number>();
    fanIn.set('example.com/x/internal/shared', 25);
    fanIn.set('example.com/x/internal/util', 40);
    const ctx = {
      profile,
      tmpDir: '/tmp',
      rulesFromBuiltin: true,
      timeoutMs: 1000,
      disabledToolIds: new Set<string>(),
    };
    const v = godaAdapter.normalizeFromFanIn(fanIn, ctx);
    expect(v).toHaveLength(2);
    expect(v[0].ruleId).toBe('high-fan-in');
    expect(v[0].category).toBe('coupling');
    expect(v[0].severity).toBe('medium');
    expect(v[0].evidence.metric?.value).toBe(25);
    expect(v[1].evidence.metric?.value).toBe(40);
  });

  it('filters packages below fan-in threshold', () => {
    const fanIn = new Map<string, number>();
    fanIn.set('example.com/x/internal/low', 5);
    fanIn.set('example.com/x/internal/high', 25);
    const ctx = {
      profile,
      tmpDir: '/tmp',
      rulesFromBuiltin: true,
      timeoutMs: 1000,
      disabledToolIds: new Set<string>(),
    };
    const v = godaAdapter.normalizeFromFanIn(fanIn, ctx);
    expect(v).toHaveLength(1);
    expect(v[0].locations[0].module).toBe('example.com/x/internal/high');
  });
});
