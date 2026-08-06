import { describe, it, expect, vi } from 'vitest';
import { judgeGroup, selectConsistent } from '../../scripts/l3-fusion/llm-judge.js';
import * as llm from '../../scripts/l3-fusion/llm-cli.js';
import type { Group } from '../../scripts/l3-fusion/grouping.js';

const group: Group = {
  id: 'g1',
  category: 'layering',
  violations: [{
    id: 'v1', source: 'l1-tool:import-linter', category: 'layering', ruleId: 'forbidden',
    ruleSource: 'repo-config', severity: 'high', locations: [{ file: 'src/domain/user.py', line: 10 }],
    evidence: { signals: ['imported=src/infra/db.py'] }, message: 'domain -> infra',
  }],
};

describe('LLM judge', () => {
  it('does majority voting across 5 calls', () => {
    const verdicts = ['real', 'real', 'insufficient', 'real', 'noise'] as const;
    const picked = selectConsistent(verdicts.map((v) => ({ verdict: v, reasoning: 'x', confidence: 'high' })));
    expect(picked.verdict).toBe('real');
    expect(picked.agreement).toBeCloseTo(0.6);
  });

  it('routes real-valued calls through llm-cli and enforces evidence anchors (round 1 final)', async () => {
    vi.spyOn(llm, 'callLlm').mockResolvedValue({
      status: 'ok',
      rawText: '```json\n{"kind":"final","verdict":"real","reasoning":"Domain [src/domain/user.py:10] imports infra","confidence":"high"}\n```',
      durationMs: 50,
    });
    const r = await judgeGroup(group, '/tmp/fake-repo');
    expect(r.verdict).toBe('real');
    expect(r.agreement).toBeGreaterThan(0.5);
    vi.restoreAllMocks();
  });

  it('handles a round-1 lookup request followed by a round-2 final', async () => {
    const spy = vi.spyOn(llm, 'callLlm')
      .mockResolvedValueOnce({                // round 1
        status: 'ok',
        rawText: '```json\n{"kind":"lookup","symbols":[{"file":"src/domain/user.py","line":10,"reason":"need to see load()"}]}\n```',
        durationMs: 10,
      })
      .mockResolvedValueOnce({                // round 2
        status: 'ok',
        rawText: '```json\n{"kind":"final","verdict":"real","reasoning":"load() at [src/domain/user.py:10] calls infra","confidence":"high"}\n```',
        durationMs: 10,
      });
    // The layering group IS tier-2 eligible; the mock will be called twice per round per loop.
    // With rounds=1 we get exactly 2 callLlm invocations.
    const r = await judgeGroup(group, '/tmp/fake-repo', 1);
    expect(r.verdict).toBe('real');
    expect(spy).toHaveBeenCalledTimes(2);
    vi.restoreAllMocks();
  });
});
