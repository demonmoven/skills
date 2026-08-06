import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { buildFlattenedTraeJsonl, splitTraeByBoundary } from './routes.js';
import type { FornaxSpan } from '../fornax/types.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const fixturesDir = join(__dirname, '..', 'test', 'fixtures');

function loadMultiTraceFixture(): FornaxSpan[] {
  const raw = readFileSync(join(fixturesDir, 'trae-multi-trace-flatten.json'), 'utf-8');
  return JSON.parse(raw) as FornaxSpan[];
}

describe('buildFlattenedTraeJsonl', () => {
  it('produces a single unified JSONL without boundary markers', () => {
    const spans = loadMultiTraceFixture();
    // Split spans into two "traces" by trace_id, preserving the per-trace grouping
    // that fetchTraeThread would return.
    const traceA = spans.filter(s => s.trace_id === 'TRACE_A');
    const traceB = spans.filter(s => s.trace_id === 'TRACE_B');

    const { jsonl, analysis } = buildFlattenedTraeJsonl(
      [
        { trace_id: 'TRACE_A', spans: traceA },
        { trace_id: 'TRACE_B', spans: traceB },
      ],
      'THREAD_X',
    );

    // No boundary marker lines in the output
    const lines = jsonl.split('\n').filter(l => l.trim());
    for (const line of lines) {
      const obj = JSON.parse(line);
      expect(obj._trae_trace_boundary).toBeUndefined();
      expect(obj._trae_format).toBe(true);
    }

    // First-occurrence reconstruction walks ALL model spans in order, seeding the
    // leading user message from trace A's input and detecting the new user turn
    // ("read a.txt") from the trailing-user position in trace B's input — so both
    // user turns appear correctly even though they come from different traces.
    const userTurns = analysis.conversation.filter(m => m.role === 'user' && m.content);
    expect(userTurns.map(m => m.content)).toEqual(['hi', 'read a.txt']);

    // Final assistant turn comes from trace B's output
    const lastAssistant = [...analysis.conversation].reverse().find(m => m.role === 'assistant');
    expect(lastAssistant?.content).toContain('hello world');
  });

  it('attributes tokens to each assistant turn from its originating model span', () => {
    const spans = loadMultiTraceFixture();
    const traceA = spans.filter(s => s.trace_id === 'TRACE_A');
    const traceB = spans.filter(s => s.trace_id === 'TRACE_B');

    const { analysis } = buildFlattenedTraeJsonl(
      [
        { trace_id: 'TRACE_A', spans: traceA },
        { trace_id: 'TRACE_B', spans: traceB },
      ],
      'THREAD_X',
    );

    const assistantTurns = analysis.conversation.filter(m => m.role === 'assistant');
    expect(assistantTurns).toHaveLength(2);

    // First assistant turn (from trace A model span): 100 in, 50 out
    expect(assistantTurns[0]!.usage?.inputTokens).toBe(100);
    expect(assistantTurns[0]!.usage?.outputTokens).toBe(50);

    // Second assistant turn (from trace B model span — the final one): 300 in, 80 out
    expect(assistantTurns[1]!.usage?.inputTokens).toBe(300);
    expect(assistantTurns[1]!.usage?.outputTokens).toBe(80);
  });
});

describe('splitTraeByBoundary (legacy compat)', () => {
  it('splits a boundary-marked JSONL into per-trace slices', () => {
    const jsonl = [
      JSON.stringify({ _trae_format: true, _trae_trace_boundary: true, traceId: 'T1', traceIndex: 0, traceCount: 2 }),
      JSON.stringify({ _trae_format: true, role: 'user', timestamp: '', content: 'hi' }),
      JSON.stringify({ _trae_format: true, role: 'assistant', timestamp: '', content: 'hello' }),
      JSON.stringify({ _trae_format: true, _trae_trace_boundary: true, traceId: 'T2', traceIndex: 1, traceCount: 2 }),
      JSON.stringify({ _trae_format: true, role: 'user', timestamp: '', content: 'thanks' }),
      JSON.stringify({ _trae_format: true, role: 'assistant', timestamp: '', content: 'youre welcome' }),
    ].join('\n') + '\n';

    const slices = splitTraeByBoundary(jsonl, 'session-id');
    expect(slices).toHaveLength(2);
    expect(slices[0]!.traceId).toBe('T1');
    expect(slices[1]!.traceId).toBe('T2');
    expect(slices[0]!.analysis.conversation.length).toBeGreaterThan(0);
  });

  it('returns empty array when no boundary markers present (new format)', () => {
    const jsonl = [
      JSON.stringify({ _trae_format: true, role: 'user', timestamp: '', content: 'hi' }),
      JSON.stringify({ _trae_format: true, role: 'assistant', timestamp: '', content: 'hello' }),
    ].join('\n') + '\n';

    const slices = splitTraeByBoundary(jsonl, 'session-id');
    expect(slices).toEqual([]);
  });
});
