// tests/types.test.ts
import { describe, it, expect } from 'vitest';
import type { Violation, Finding, RepoProfile, ToolRunResult } from '../scripts/types.js';

describe('types', () => {
  it('Violation shape is stable', () => {
    const v: Violation = {
      id: 'abc',
      source: 'l1-tool:import-linter',
      category: 'layering',
      ruleId: 'forbidden-import',
      ruleSource: 'repo-config',
      severity: 'high',
      locations: [{ file: 'src/domain/user.py', line: 10 }],
      evidence: { signals: ['test evidence'] },
      message: 'test violation',
    };
    expect(v.category).toBe('layering');
  });

  it('Finding shape is stable', () => {
    const f: Finding = {
      id: 'group-1',
      category: 'layering',
      title: 'domain depends on infra',
      rootCause: 'see file X:10',
      impact: 'hard to refactor',
      actions: ['extract interface'],
      confidence: 'high',
      confidenceScore: 0.95,
      severity: 'high',
      signals: [{ source: 'l1-tool:import-linter', description: '14 forbidden imports' }],
      locations: [{ file: 'src/domain/user.py' }],
      violationIds: ['v1', 'v2'],
    };
    expect(f.confidence).toBe('high');
  });
});
