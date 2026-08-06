import { describe, it, expect } from 'vitest';
import { detectCli, parseEvalResult } from './evaluator.js';

describe('detectCli', () => {
  it('returns a detected CLI with type and path', async () => {
    const result = await detectCli();
    // In CI or environments without CLIs, this may return null.
    // On developer machines, at least one CLI should be available.
    if (result) {
      expect(result.type).toMatch(/^(claude-code|opencode)$/);
      expect(result.path).toBeTruthy();
    }
  });

  it('respects preferred parameter', async () => {
    const result = await detectCli('opencode');
    if (result) {
      // If opencode is available, it should be preferred
      // If not, falls back to claude-code
      expect(result.type).toMatch(/^(claude-code|opencode)$/);
    }
  });
});

describe('parseEvalResult', () => {
  it('parses a clean JSON output', () => {
    const stdout = `{
      "score": 0.82,
      "task_completion": 0.9,
      "efficiency": 0.75,
      "error_handling": 0.8,
      "communication": 0.85,
      "code_quality": 0.78,
      "harness_usage": 0.7,
      "comment": "助手表现良好，完成了任务需求。"
    }`;

    const result = parseEvalResult(stdout, 'claude-code');
    expect(result.score).toBe(0.82);
    expect(result.taskCompletion).toBe(0.9);
    expect(result.efficiency).toBe(0.75);
    expect(result.errorHandling).toBe(0.8);
    expect(result.communication).toBe(0.85);
    expect(result.codeQuality).toBe(0.78);
    expect(result.harnessUsage).toBe(0.7);
    expect(result.comment).toBe('助手表现良好，完成了任务需求。');
    expect(result.evaluatedBy).toBe('claude-code');
    expect(result.evaluatedAt).toBeTruthy();
  });

  it('parses JSON embedded in extra text', () => {
    const stdout = `
Here is my evaluation of the session:

\`\`\`json
{
  "score": 0.65,
  "task_completion": 0.7,
  "efficiency": 0.6,
  "error_handling": 0.55,
  "communication": 0.7,
  "code_quality": 0.65,
  "harness_usage": 0.5,
  "comment": "整体表现一般。"
}
\`\`\`

That's my assessment.
`;

    const result = parseEvalResult(stdout, 'opencode');
    expect(result.score).toBe(0.65);
    expect(result.taskCompletion).toBe(0.7);
    expect(result.evaluatedBy).toBe('opencode');
  });

  it('clamps out-of-range scores to 0-1', () => {
    const stdout = `{
      "score": 1.5,
      "task_completion": -0.3,
      "efficiency": 0.8,
      "error_handling": 0.8,
      "communication": 0.8,
      "code_quality": 0.8,
      "harness_usage": 0.8,
      "comment": "test"
    }`;

    const result = parseEvalResult(stdout, 'claude-code');
    expect(result.score).toBe(1);
    expect(result.taskCompletion).toBe(0);
  });

  it('falls back to score when dimension fields are missing', () => {
    const stdout = `{
      "score": 0.75,
      "comment": "缺少维度评分。"
    }`;

    const result = parseEvalResult(stdout, 'claude-code');
    expect(result.score).toBe(0.75);
    expect(result.taskCompletion).toBe(0.75);
    expect(result.efficiency).toBe(0.75);
    expect(result.errorHandling).toBe(0.75);
    expect(result.communication).toBe(0.75);
    expect(result.codeQuality).toBe(0.75);
    expect(result.harnessUsage).toBe(0.75);
  });

  it('provides default comment when missing', () => {
    const stdout = `{"score": 0.5}`;

    const result = parseEvalResult(stdout, 'claude-code');
    expect(result.comment).toBe('无评语');
  });

  it('throws on missing JSON', () => {
    expect(() => parseEvalResult('No JSON here', 'claude-code')).toThrow('no JSON found');
  });

  it('throws on invalid JSON with score keyword', () => {
    expect(() => parseEvalResult('{"score": invalid}', 'claude-code')).toThrow('Failed to parse');
  });

  it('throws on missing score field', () => {
    const stdout = `{"task_completion": 0.8, "comment": "no score"}`;
    expect(() => parseEvalResult(stdout, 'claude-code')).toThrow('missing or invalid "score"');
  });
});
