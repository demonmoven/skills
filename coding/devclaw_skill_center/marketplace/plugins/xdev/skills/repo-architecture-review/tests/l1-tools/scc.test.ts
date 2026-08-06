import { describe, it, expect } from 'vitest';
import { sccAdapter } from '../../scripts/l1-tools/adapters/scc.js';
import type { RepoProfile } from '../../scripts/types.js';

const profile: RepoProfile = {
  rootPath: '/tmp/x',
  languages: [{ lang: 'go', fileCount: 10, locTotal: 100 }],
  hasGoMod: false,
  hasPyProject: false,
  hasSetupCfg: false,
  hasRequirementsTxt: false,
  dockerfiles: [],
  ghaWorkflows: [],
  topLevelPackages: {},
  workspaces: [],
  totalFiles: 10,
  totalLocMillions: 0,
  gitLogLineEstimate: 0,
  detectedConfigs: {},
};

describe('scc adapter', () => {
  it('applies always', () => {
    expect(sccAdapter.detectApplies(profile)).toBe(true);
  });

  it('returns empty violations (pure info source)', () => {
    const ctx = {
      profile,
      tmpDir: '/tmp',
      rulesFromBuiltin: true,
      timeoutMs: 1000,
      disabledToolIds: new Set<string>(),
    };
    const v = sccAdapter.normalizeFromParsed(
      [{ Language: 'Go', Files: 10, Lines: 100, Code: 80, Comments: 10, Blanks: 10, Complexity: 0 }],
      ctx
    );
    expect(v).toEqual([]);
  });
});
