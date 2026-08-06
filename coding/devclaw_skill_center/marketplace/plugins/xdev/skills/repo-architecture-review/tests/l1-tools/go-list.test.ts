import { describe, it, expect } from 'vitest';
import { goListAdapter } from '../../scripts/l1-tools/adapters/go-list.js';
import type { RepoProfile } from '../../scripts/types.js';

const profile: RepoProfile = {
  rootPath: '/tmp/x',
  languages: [{ lang: 'go', fileCount: 5, locTotal: 40 }],
  hasGoMod: true,
  hasPyProject: false,
  hasSetupCfg: false,
  hasRequirementsTxt: false,
  dockerfiles: [],
  ghaWorkflows: [],
  topLevelPackages: { go: ['internal/domain', 'internal/infra'] },
  workspaces: [],
  totalFiles: 5,
  totalLocMillions: 0,
  gitLogLineEstimate: 0,
  detectedConfigs: {},
};

describe('go-list adapter', () => {
  it('applies to Go repos', () => {
    expect(goListAdapter.detectApplies(profile)).toBe(true);
  });

  it('does not apply to non-Go repos', () => {
    const nonGoProfile: RepoProfile = { ...profile, hasGoMod: false };
    expect(goListAdapter.detectApplies(nonGoProfile)).toBe(false);
  });

  it('detects package-level circular dependencies', () => {
    const packages = [
      { ImportPath: 'example.com/x/a', Imports: ['example.com/x/b'] },
      { ImportPath: 'example.com/x/b', Imports: ['example.com/x/a'] },
    ];
    const ctx = { profile, tmpDir: '/tmp', rulesFromBuiltin: true, timeoutMs: 1000, disabledToolIds: new Set<string>() };
    const v = goListAdapter.normalizeFromPackages(packages, ctx);
    expect(v).toHaveLength(1);
    expect(v[0].ruleId).toBe('circular');
    expect(v[0].severity).toBe('high');
    expect(v[0].category).toBe('dependency');
  });

  it('detects larger cycles', () => {
    const packages = [
      { ImportPath: 'example.com/x/a', Imports: ['example.com/x/b'] },
      { ImportPath: 'example.com/x/b', Imports: ['example.com/x/c'] },
      { ImportPath: 'example.com/x/c', Imports: ['example.com/x/a'] },
    ];
    const ctx = { profile, tmpDir: '/tmp', rulesFromBuiltin: true, timeoutMs: 1000, disabledToolIds: new Set<string>() };
    const v = goListAdapter.normalizeFromPackages(packages, ctx);
    expect(v).toHaveLength(1);
    expect(v[0].locations).toHaveLength(3);
    expect(v[0].evidence.signals).toContain('scc-size=3');
  });

  it('ignores acyclic graphs', () => {
    const packages = [
      { ImportPath: 'example.com/x/a', Imports: ['example.com/x/b'] },
      { ImportPath: 'example.com/x/b', Imports: ['example.com/x/c'] },
      { ImportPath: 'example.com/x/c', Imports: [] },
    ];
    const ctx = { profile, tmpDir: '/tmp', rulesFromBuiltin: true, timeoutMs: 1000, disabledToolIds: new Set<string>() };
    const v = goListAdapter.normalizeFromPackages(packages, ctx);
    expect(v).toHaveLength(0);
  });

  it('handles missing imports field', () => {
    const packages = [
      { ImportPath: 'example.com/x/a' },
      { ImportPath: 'example.com/x/b', Imports: [] },
    ];
    const ctx = { profile, tmpDir: '/tmp', rulesFromBuiltin: true, timeoutMs: 1000, disabledToolIds: new Set<string>() };
    const v = goListAdapter.normalizeFromPackages(packages, ctx);
    expect(v).toHaveLength(0);
  });
});
