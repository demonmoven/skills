import { describe, it, expect } from 'vitest';
import { hadolintAdapter } from '../../scripts/l1-tools/adapters/hadolint.js';
import type { RepoProfile } from '../../scripts/types.js';

const baseProfile = (overrides: Partial<RepoProfile>): RepoProfile => ({
  rootPath: '/tmp/x',
  languages: [],
  hasGoMod: false,
  hasPyProject: false,
  hasSetupCfg: false,
  hasRequirementsTxt: false,
  dockerfiles: ['Dockerfile', 'images/api/Dockerfile.api'],
  ghaWorkflows: [],
  topLevelPackages: {},
  workspaces: [],
  totalFiles: 2,
  totalLocMillions: 0,
  gitLogLineEstimate: 0,
  detectedConfigs: {},
  ...overrides,
});

describe('hadolint adapter', () => {
  it('applies when Dockerfiles exist', () => {
    expect(hadolintAdapter.detectApplies(baseProfile({}))).toBe(true);
    expect(hadolintAdapter.detectApplies(baseProfile({ dockerfiles: [] }))).toBe(false);
  });

  it('normalizes hadolint JSON array into build-deploy violations', () => {
    const parsed = [
      {
        file: 'Dockerfile',
        line: 1,
        column: 1,
        level: 'warning' as const,
        code: 'DL3006',
        message: 'Always tag the version of an image explicitly',
      },
      {
        file: 'Dockerfile',
        line: 5,
        column: 1,
        level: 'error' as const,
        code: 'DL3002',
        message: 'Last USER should not be root',
      },
    ];
    const ctx = {
      profile: baseProfile({}),
      tmpDir: '/tmp',
      rulesFromBuiltin: true,
      timeoutMs: 1000,
      disabledToolIds: new Set<string>(),
    };
    const v = hadolintAdapter.normalizeFromParsed(parsed, ctx);
    expect(v).toHaveLength(2);
    expect(v[0].category).toBe('build-deploy');
    expect(v[0].ruleId).toBe('DL3006');
    expect(v[0].severity).toBe('medium');
    expect(v[1].severity).toBe('high');
  });
});
