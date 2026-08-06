import { describe, it, expect } from 'vitest';
import { extractJsonFromLlmText } from '../../scripts/l3-fusion/llm-cli.js';

describe('extractJsonFromLlmText', () => {
  it('extracts JSON from fenced code block', () => {
    const text = 'here is my output:\n```json\n{"verdict":"high","reasoning":"because"}\n```\n';
    expect(extractJsonFromLlmText(text)).toEqual({ verdict: 'high', reasoning: 'because' });
  });
  it('extracts bare JSON', () => {
    expect(extractJsonFromLlmText('   {"a":1}   ')).toEqual({ a: 1 });
  });
  it('repairs trailing truncation', () => {
    const text = '```json\n{"a":1,"b":[1,2';
    const r = extractJsonFromLlmText(text);
    expect(r).toEqual({ a: 1, b: [1, 2] });
  });
});
