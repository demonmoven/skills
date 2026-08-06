import { describe, it, expect } from 'vitest';
import { partition, shouldChunk } from '../../scripts/mapreduce/partition.js';
import type { RepoProfile } from '../../scripts/types.js';

const big = (overrides: Partial<RepoProfile>): RepoProfile => ({
  rootPath: '/tmp/x', languages: [], hasGoMod: false, hasPyProject: false, hasSetupCfg: false, hasRequirementsTxt: false,
  dockerfiles: [], ghaWorkflows: [], topLevelPackages: {}, workspaces: [],
  totalFiles: 6000, totalLocMillions: 1, gitLogLineEstimate: 100_000, detectedConfigs: {},
  ...overrides,
});

describe('shouldChunk', () => {
  it('true when > 5000 files', () => {
    expect(shouldChunk(big({}))).toBe(true);
    expect(shouldChunk(big({ totalFiles: 500 }))).toBe(false);
  });
  it('true when > 500k git log lines', () => {
    expect(shouldChunk(big({ totalFiles: 500, gitLogLineEstimate: 600_000 }))).toBe(true);
  });
  it('true when > 10 workspaces', () => {
    expect(shouldChunk(big({ totalFiles: 500, workspaces: Array(12).fill('w') }))).toBe(true);
  });
});

describe('partition', () => {
  it('splits by top-level directories', () => {
    const files = ['a/x.py', 'a/y.py', 'b/z.py', 'c/q.go', 'c/r.go'];
    const chunks = partition(files);
    const keys = chunks.map((c) => c.key).sort();
    expect(keys).toEqual(['a', 'b', 'c']);
  });
});
