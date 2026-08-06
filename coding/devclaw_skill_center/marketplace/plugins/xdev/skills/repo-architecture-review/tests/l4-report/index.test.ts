import { describe, it, expect } from 'vitest';
import { renderReport } from '../../scripts/l4-report/index.js';
import type { AnalyzeResult } from '../../scripts/orchestrator.js';

const result: AnalyzeResult = {
  profile: {
    rootPath: '/tmp/x', languages: [{ lang: 'python', fileCount: 10, locTotal: 100 }],
    hasGoMod: false, hasPyProject: true, hasSetupCfg: false, hasRequirementsTxt: false,
    dockerfiles: [], ghaWorkflows: [], topLevelPackages: { python: ['mypkg'] }, workspaces: [],
    totalFiles: 10, totalLocMillions: 0, gitLogLineEstimate: 0, detectedConfigs: {},
  },
  toolRuns: [
    { toolId: 'import-linter', status: 'ok', durationMs: 500, violations: [], installedVia: 'pipx', generatedConfigPath: 'pyproject.toml' },
    { toolId: 'hadolint', status: 'skipped', reason: 'not-applicable', durationMs: 0, violations: [] },
  ],
  graphViolations: [],
  findings: [],
  reportedFindings: [{
    id: 'g1', category: 'layering', title: 'domain -> infra', rootCause: 'File [src/domain/user.py:10] imports infra',
    impact: 'hurts testability', actions: ['extract interface'],
    confidence: 'high', confidenceScore: 0.9, severity: 'high',
    signals: [{ source: 'l1-tool:import-linter', description: '14 forbidden imports' }],
    locations: [{ file: 'src/domain/user.py', line: 10 }], violationIds: ['v1'],
  }],
  knownUnchanged: [],
  runMeta: {
    startedAt: '2026-04-22T00:00:00.000Z', finishedAt: '2026-04-22T00:05:00.000Z', wallClockMs: 300_000,
    toolRuns: [], generatedFiles: ['pyproject.toml'], coverageDowngrades: ['hadolint: not-applicable'],
    stackDetection: ['python'], mapReduceUsed: false,
  },
  reportPath: '/tmp/report.md',
};

describe('renderReport', () => {
  it('emits Summary, Findings, Action Roadmap, and Audit sections', () => {
    const md = renderReport(result);
    expect(md).toContain('架构体检报告');
    expect(md).toContain('健康总评');
    expect(md).toContain('## 关键发现');
    expect(md).toMatch(/1\.\s+domain -> infra/);
    expect(md).toContain('🟠 高');
    expect(md).toContain('## 执行审计');
    expect(md).toContain('import-linter');
    expect(md).toContain('## 行动路线图');
  });
});
