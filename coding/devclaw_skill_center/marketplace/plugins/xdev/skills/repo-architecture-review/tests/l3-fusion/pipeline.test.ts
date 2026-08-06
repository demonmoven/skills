import { describe, it, expect } from 'vitest';
import { fuseFindings } from '../../scripts/l3-fusion/index.js';
import type { Violation } from '../../scripts/types.js';

const v1: Violation = {
  id: 'v1', source: 'l1-tool:import-linter', category: 'layering', ruleId: 'forbidden',
  ruleSource: 'repo-config', severity: 'high', locations: [{ file: 'a.py' }], evidence: { signals: ['x'] }, message: 'm1',
};
const v2: Violation = {
  id: 'v2', source: 'l2-graph:co-change', category: 'layering', ruleId: 'cc', ruleSource: 'skill-builtin',
  severity: 'medium', locations: [{ file: 'a.py' }], evidence: { signals: ['shared=5'] }, message: 'm2',
};

describe('fuseFindings (llmOptional)', () => {
  it('produces non-LLM findings when llmOptional=true', async () => {
    const findings = await fuseFindings([v1, v2], { repoRoot: '/tmp/fake-repo', concurrency: 2, selfConsistencyRounds: 1, llmOptional: true });
    expect(findings.length).toBeGreaterThan(0);
    expect(findings[0].confidence).toBe('high');       // two independent sources
  });
});
