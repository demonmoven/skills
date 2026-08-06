import { describe, it, expect } from 'vitest';
import { grimpAdapter } from '../../scripts/l1-tools/adapters/grimp.js';
import type { RepoProfile } from '../../scripts/types.js';

const baseProfile = (overrides: Partial<RepoProfile>): RepoProfile => ({
  rootPath: '/tmp/x',
  languages: [{ lang: 'python', fileCount: 10, locTotal: 100 }],
  hasGoMod: false,
  hasPyProject: true,
  hasSetupCfg: false,
  hasRequirementsTxt: false,
  dockerfiles: [],
  ghaWorkflows: [],
  topLevelPackages: { python: ['mypkg'] },
  workspaces: [],
  totalFiles: 10,
  totalLocMillions: 0,
  gitLogLineEstimate: 0,
  detectedConfigs: {},
  ...overrides,
});

describe('grimp adapter', () => {
  it('applies to Python projects with a root package', () => {
    expect(grimpAdapter.detectApplies(baseProfile({}))).toBe(true);
    expect(grimpAdapter.detectApplies(baseProfile({ topLevelPackages: {} }))).toBe(false);
  });

  it('normalizes orphan modules into dead-code violations', () => {
    const parsed = {
      cycles: [],
      orphans: ['mypkg.unused', 'mypkg.utils.deprecated'],
      stats: { modules: 42, imports: 100 },
    };
    const ctx = {
      profile: baseProfile({}),
      tmpDir: '/tmp',
      rulesFromBuiltin: true,
      timeoutMs: 1000,
      disabledToolIds: new Set<string>(),
    };
    const v = grimpAdapter.normalizeFromParsed(parsed, ctx);
    expect(v).toHaveLength(2);
    expect(v.map((x) => x.category)).toEqual(['dead-code', 'dead-code']);
    expect(v[0].locations[0].file).toBe('mypkg/unused.py');
    expect(v[1].locations[0].file).toBe('mypkg/utils/deprecated.py');
  });

  it('normalizes cycles into dependency violations', () => {
    const parsed = {
      cycles: [['mypkg.a', 'mypkg.b', 'mypkg.a']],
      orphans: [],
      stats: { modules: 3, imports: 3 },
    };
    const ctx = {
      profile: baseProfile({}),
      tmpDir: '/tmp',
      rulesFromBuiltin: true,
      timeoutMs: 1000,
      disabledToolIds: new Set<string>(),
    };
    const v = grimpAdapter.normalizeFromParsed(parsed, ctx);
    expect(v).toHaveLength(1);
    expect(v[0].category).toBe('dependency');
    expect(v[0].ruleId).toBe('circular');
    expect(v[0].locations).toHaveLength(3);
    expect(v[0].locations[0].file).toBe('mypkg/a.py');
    expect(v[0].locations[0].module).toBe('mypkg.a');
  });

  it('skips cycles of size 0 or 1', () => {
    const parsed = {
      cycles: [[], ['solo']],
      orphans: [],
      stats: { modules: 2, imports: 0 },
    };
    const ctx = {
      profile: baseProfile({}),
      tmpDir: '/tmp',
      rulesFromBuiltin: true,
      timeoutMs: 1000,
      disabledToolIds: new Set<string>(),
    };
    const v = grimpAdapter.normalizeFromParsed(parsed, ctx);
    expect(v).toHaveLength(0);
  });

  it('normalizes mixed orphans and cycles', () => {
    const parsed = {
      cycles: [['mypkg.x', 'mypkg.y', 'mypkg.x']],
      orphans: ['mypkg.dead'],
      stats: { modules: 4, imports: 2 },
    };
    const ctx = {
      profile: baseProfile({}),
      tmpDir: '/tmp',
      rulesFromBuiltin: true,
      timeoutMs: 1000,
      disabledToolIds: new Set<string>(),
    };
    const v = grimpAdapter.normalizeFromParsed(parsed, ctx);
    expect(v).toHaveLength(2);
    expect(v[0].category).toBe('dead-code');
    expect(v[1].category).toBe('dependency');
  });
});
