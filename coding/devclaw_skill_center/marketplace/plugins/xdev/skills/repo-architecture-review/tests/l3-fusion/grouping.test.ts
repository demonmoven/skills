import { describe, it, expect } from 'vitest';
import { groupViolations } from '../../scripts/l3-fusion/grouping.js';
import type { Violation } from '../../scripts/types.js';

const mk = (id: string, category: Violation['category'], file: string, source: string): Violation => ({
  id, source, category, ruleId: id, ruleSource: 'skill-builtin', severity: 'medium',
  locations: [{ file }], evidence: {}, message: id,
});

describe('groupViolations', () => {
  it('groups by (category, nearby location)', () => {
    const vs: Violation[] = [
      mk('v1', 'layering', 'src/domain/user.py', 'l1-tool:import-linter'),
      mk('v2', 'layering', 'src/domain/user.py', 'l2-graph:co-change'),
      mk('v3', 'layering', 'src/unrelated.py', 'l1-tool:import-linter'),
      mk('v4', 'dead-code', 'src/domain/user.py', 'l2-graph:orphan'),
    ];
    const groups = groupViolations(vs);
    expect(groups).toHaveLength(3);
    const first = groups.find((g) => g.violations.map((v) => v.id).sort().join(',') === 'v1,v2');
    expect(first).toBeDefined();
  });
});
