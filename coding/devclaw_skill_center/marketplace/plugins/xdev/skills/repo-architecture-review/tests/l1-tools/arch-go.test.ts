import { describe, it, expect } from 'vitest';
import { archGoAdapter } from '../../scripts/l1-tools/adapters/arch-go.js';
import type { RepoProfile } from '../../scripts/types.js';

const profile: RepoProfile = {
  rootPath: '/tmp/x',
  languages: [{ lang: 'go', fileCount: 20, locTotal: 400 }],
  hasGoMod: true, hasPyProject: false, hasSetupCfg: false, hasRequirementsTxt: false,
  dockerfiles: [], ghaWorkflows: [], topLevelPackages: { go: ['internal/domain', 'internal/infra', 'cmd/api'] }, workspaces: [],
  totalFiles: 20, totalLocMillions: 0, gitLogLineEstimate: 0, detectedConfigs: {},
};

describe('arch-go adapter', () => {
  it('applies to Go projects', () => {
    expect(archGoAdapter.detectApplies(profile)).toBe(true);
    expect(archGoAdapter.detectApplies({ ...profile, hasGoMod: false })).toBe(false);
  });

  it('normalizes arch-go JSON violations into layering violations', () => {
    const parsed = {
      pass: false,
      details: {
        dependenciesRuleResults: [
          {
            rule: 'domain must not depend on infra',
            passes: false,
            verifications: [{
              package: 'example.com/x/internal/domain',
              passes: false,
              details: ['disallowed import example.com/x/internal/infra'],
            }],
          },
        ],
        functionsRuleResults: [],
        contentsRuleResults: [],
        namingRuleResults: [],
      },
      compliance: 80,
      coverage: 100,
    };
    const ctx = { profile, tmpDir: '/tmp', rulesFromBuiltin: true, timeoutMs: 1000, disabledToolIds: new Set<string>() };
    const v = archGoAdapter.normalizeFromParsed(parsed, ctx);
    expect(v).toHaveLength(1);
    expect(v[0].category).toBe('layering');
    expect(v[0].evidence.signals).toContain('rule=domain must not depend on infra');
  });
});
