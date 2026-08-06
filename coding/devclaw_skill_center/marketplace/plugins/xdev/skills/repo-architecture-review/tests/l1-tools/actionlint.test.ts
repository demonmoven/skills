import { describe, it, expect } from 'vitest';
import { actionlintAdapter, computeWorkflowComplexity } from '../../scripts/l1-tools/adapters/actionlint.js';
import type { RepoProfile } from '../../scripts/types.js';

const profile: RepoProfile = {
  rootPath: '/tmp/x',
  languages: [],
  hasGoMod: false,
  hasPyProject: false,
  hasSetupCfg: false,
  hasRequirementsTxt: false,
  dockerfiles: [],
  ghaWorkflows: ['.github/workflows/ci.yml'],
  topLevelPackages: {},
  workspaces: [],
  totalFiles: 1,
  totalLocMillions: 0,
  gitLogLineEstimate: 0,
  detectedConfigs: {},
};

describe('actionlint adapter', () => {
  it('applies when GHA workflows exist', () => {
    expect(actionlintAdapter.detectApplies(profile)).toBe(true);
    expect(actionlintAdapter.detectApplies({ ...profile, ghaWorkflows: [] })).toBe(false);
  });

  it('computes workflow complexity metrics', () => {
    const yaml = `
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        node: [18, 20, 22]
        os: [ubuntu-latest, macos-latest]
    steps:
      - uses: actions/checkout@v4
      - run: npm ci
      - run: npm test
      - uses: actions/upload-artifact@v3
  deploy:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - run: ./deploy.sh
`;
    const m = computeWorkflowComplexity('.github/workflows/ci.yml', yaml);
    expect(m.jobCount).toBe(2);
    expect(m.matrixCombinations).toBe(6);
    expect(m.stepCount).toBe(5);
    expect(m.totalActionsUses).toBe(2);
    expect(m.unpinnedActions).toBe(0);
    expect(m.shaPinnedActions).toBe(0);
  });
});
