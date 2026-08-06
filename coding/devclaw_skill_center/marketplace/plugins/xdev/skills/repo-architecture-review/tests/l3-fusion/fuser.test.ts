import { describe, it, expect, vi } from 'vitest';
import { fuseGroup } from '../../scripts/l3-fusion/fuser.js';
import * as llm from '../../scripts/l3-fusion/llm-cli.js';
import type { Group } from '../../scripts/l3-fusion/grouping.js';

const group: Group = {
  id: 'g1',
  category: 'layering',
  violations: [{
    id: 'v1', source: 'l1-tool:import-linter', category: 'layering', ruleId: 'forbidden',
    ruleSource: 'repo-config', severity: 'high', locations: [{ file: 'src/domain/user.py', line: 10 }],
    evidence: { signals: ['imported=src/infra/db.py'] }, message: '',
  }],
};

describe('fuser', () => {
  it('produces a Finding with title/rootCause/impact/actions (round-1 final)', async () => {
    vi.spyOn(llm, 'callLlm').mockResolvedValue({
      status: 'ok',
      rawText: '```json\n{"kind":"final","title":"domain -> infra dependency","rootCause":"File [src/domain/user.py:10] imports infra","impact":"hurts testability","actions":["extract infra interface"]}\n```',
      durationMs: 50,
    });
    const f = await fuseGroup(group, '/tmp/fake-repo', { baseConfidence: 'high', independentSources: ['l1-tool'] }, { verdict: 'real', reasoning: 'x', confidence: 'high', agreement: 0.8 });
    expect(f).not.toBeNull();
    expect(f!.title).toBeTruthy();
    expect(f!.actions).toHaveLength(1);
    vi.restoreAllMocks();
  });

  it('produces a low-signal passthrough finding when LLM marks skip (max-coverage mode)', async () => {
    vi.spyOn(llm, 'callLlm').mockResolvedValue({
      status: 'ok',
      rawText: '```json\n{"kind":"final","title":"<skip>","rootCause":"","impact":"","actions":[]}\n```',
      durationMs: 10,
    });
    const f = await fuseGroup(group, '/tmp/fake-repo', { baseConfidence: 'low', independentSources: [] }, { verdict: 'insufficient', reasoning: '', confidence: 'low', agreement: 0.4 });
    expect(f).not.toBeNull();
    expect(f!.title).toMatch(/^\[LOW-SIGNAL\]/);
    expect(f!.severity).toBe('low');
    expect(f!.confidence).toBe('low');
    vi.restoreAllMocks();
  });
});
