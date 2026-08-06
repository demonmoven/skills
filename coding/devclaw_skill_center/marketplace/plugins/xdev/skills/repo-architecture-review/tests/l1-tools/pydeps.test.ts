import { describe, it, expect } from 'vitest';
import { pydepsAdapter } from '../../scripts/l1-tools/adapters/pydeps.js';
import type { RepoProfile } from '../../scripts/types.js';

const baseProfile = (overrides: Partial<RepoProfile>): RepoProfile => ({
  rootPath: '/tmp/x',
  languages: [{ lang: 'python', fileCount: 5, locTotal: 50 }],
  hasGoMod: false,
  hasPyProject: true,
  hasSetupCfg: false,
  hasRequirementsTxt: false,
  dockerfiles: [],
  ghaWorkflows: [],
  topLevelPackages: { python: ['mypkg'] },
  workspaces: [],
  totalFiles: 5,
  totalLocMillions: 0,
  gitLogLineEstimate: 0,
  detectedConfigs: {},
  ...overrides,
});

describe('pydeps adapter', () => {
  it('applies when a python top package exists', () => {
    expect(pydepsAdapter.detectApplies(baseProfile({}))).toBe(true);
    expect(pydepsAdapter.detectApplies(baseProfile({ topLevelPackages: {} }))).toBe(false);
  });

  it('normalizes JSON cycle output into dependency violations', () => {
    const parsed = { bacon: [['mypkg.a', 'mypkg.b', 'mypkg.a']] };
    const ctx = {
      profile: baseProfile({}),
      tmpDir: '/tmp',
      rulesFromBuiltin: true,
      timeoutMs: 1000,
      disabledToolIds: new Set<string>(),
    };
    const v = pydepsAdapter.normalizeFromParsed(parsed, ctx);
    expect(v).toHaveLength(1);
    expect(v[0].ruleId).toBe('circular');
    expect(v[0].category).toBe('dependency');
    expect(v[0].severity).toBe('high');
  });

  it('skips cycles of size < 2', () => {
    const parsed = { bacon: [[], ['solo']] };
    const ctx = {
      profile: baseProfile({}),
      tmpDir: '/tmp',
      rulesFromBuiltin: true,
      timeoutMs: 1000,
      disabledToolIds: new Set<string>(),
    };
    const v = pydepsAdapter.normalizeFromParsed(parsed, ctx);
    expect(v).toHaveLength(0);
  });

  it('handles multiple cycles', () => {
    const parsed = {
      bacon: [
        ['mypkg.a', 'mypkg.b', 'mypkg.a'],
        ['mypkg.x', 'mypkg.y', 'mypkg.z', 'mypkg.x'],
      ],
    };
    const ctx = {
      profile: baseProfile({}),
      tmpDir: '/tmp',
      rulesFromBuiltin: true,
      timeoutMs: 1000,
      disabledToolIds: new Set<string>(),
    };
    const v = pydepsAdapter.normalizeFromParsed(parsed, ctx);
    expect(v).toHaveLength(2);
    expect(v[0].locations).toHaveLength(3);
    expect(v[1].locations).toHaveLength(4);
  });

  it('converts module names to file paths', () => {
    const parsed = { bacon: [['mypkg.a.b', 'mypkg.c', 'mypkg.a.b']] };
    const ctx = {
      profile: baseProfile({}),
      tmpDir: '/tmp',
      rulesFromBuiltin: true,
      timeoutMs: 1000,
      disabledToolIds: new Set<string>(),
    };
    const v = pydepsAdapter.normalizeFromParsed(parsed, ctx);
    expect(v).toHaveLength(1);
    expect(v[0].locations[0].file).toBe('mypkg/a/b.py');
    expect(v[0].locations[1].file).toBe('mypkg/c.py');
  });

  it('handles empty bacon array', () => {
    const parsed = { bacon: [] };
    const ctx = {
      profile: baseProfile({}),
      tmpDir: '/tmp',
      rulesFromBuiltin: true,
      timeoutMs: 1000,
      disabledToolIds: new Set<string>(),
    };
    const v = pydepsAdapter.normalizeFromParsed(parsed, ctx);
    expect(v).toHaveLength(0);
  });

  it('handles undefined bacon', () => {
    const parsed = {};
    const ctx = {
      profile: baseProfile({}),
      tmpDir: '/tmp',
      rulesFromBuiltin: true,
      timeoutMs: 1000,
      disabledToolIds: new Set<string>(),
    };
    const v = pydepsAdapter.normalizeFromParsed(parsed, ctx);
    expect(v).toHaveLength(0);
  });
});
