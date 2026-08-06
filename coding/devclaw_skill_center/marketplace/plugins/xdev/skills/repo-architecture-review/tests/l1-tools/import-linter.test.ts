import { describe, it, expect } from 'vitest';
import { importLinterAdapter } from '../../scripts/l1-tools/adapters/import-linter.js';
import type { RepoProfile } from '../../scripts/types.js';

const baseProfile = (overrides: Partial<RepoProfile>): RepoProfile => ({
  rootPath: '/tmp/x',
  languages: [{ lang: 'python', fileCount: 10, locTotal: 100 }],
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
  ...overrides,
});

describe('import-linter adapter', () => {
  it('applies to Python projects', () => {
    expect(importLinterAdapter.detectApplies(baseProfile({ hasPyProject: true }))).toBe(true);
    expect(importLinterAdapter.detectApplies(baseProfile({ hasSetupCfg: true }))).toBe(true);
    expect(importLinterAdapter.detectApplies(baseProfile({}))).toBe(false);
  });

  it('normalizes Contract report lines into violations', () => {
    const ctx = {
      profile: baseProfile({
        hasPyProject: true,
        topLevelPackages: { python: ['mypkg', 'mypkg.domain', 'mypkg.infra'] },
      }),
      tmpDir: '/tmp/archreview',
      rulesFromBuiltin: true,
      timeoutMs: 1000,
      disabledToolIds: new Set<string>(),
    };
    const parsed = {
      contracts: [
        {
          name: 'domain must not import infra',
          kind: 'Forbidden' as const,
          broken: true,
          violations: [
            {
              importer: 'mypkg.domain.user',
              imported: 'mypkg.infra.db',
              line_number: 10,
              line_contents: 'from mypkg.infra.db import foo',
            },
          ],
        },
      ],
    };
    const v = importLinterAdapter.normalizeFromParsed(parsed, ctx);
    expect(v).toHaveLength(1);
    expect(v[0].category).toBe('layering');
    expect(v[0].locations[0].file).toBe('mypkg/domain/user.py');
    expect(v[0].evidence.signals).toContain('imported=mypkg.infra.db');
  });
});
