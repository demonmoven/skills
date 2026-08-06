import { describe, it, expect } from 'vitest';
import type { NormalizedMessage } from './types.js';
import { extractHarnessMetrics } from './harness-metrics.js';

function msg(overrides: Partial<NormalizedMessage>): NormalizedMessage {
  return {
    role: 'assistant',
    timestamp: '2026-04-09T10:00:00Z',
    content: '',
    ...overrides,
  };
}

describe('extractHarnessMetrics', () => {
  it('counts Read calls to harness files', () => {
    const messages: NormalizedMessage[] = [
      msg({ toolCalls: [{ name: 'Read', input: '/project/AGENTS.md' }] }),
      msg({ toolCalls: [{ name: 'Read', input: '/project/AGENTS.md' }] }),
      msg({ toolCalls: [{ name: 'Read', input: '/project/docs/rules/invariants.md' }] }),
    ];

    const result = extractHarnessMetrics(messages);
    expect(result.totalArtifacts).toBe(2);
    expect(result.usedArtifacts).toBe(2);

    const agents = result.artifacts.find(a => a.path === 'AGENTS.md');
    expect(agents).toBeDefined();
    expect(agents!.readCount).toBe(2);
    expect(agents!.category).toBe('doc-root');

    const invariants = result.artifacts.find(a => a.path.includes('invariants'));
    expect(invariants).toBeDefined();
    expect(invariants!.readCount).toBe(1);
    expect(invariants!.category).toBe('doc-rules');
  });

  it('counts mentions in assistant content', () => {
    const messages: NormalizedMessage[] = [
      msg({ content: 'I will check AGENTS.md and ARCHITECTURE.md for guidance.' }),
    ];

    const result = extractHarnessMetrics(messages);
    // Both are mentioned but not Read — they should appear with mentionCount > 0
    const agents = result.artifacts.find(a => a.path === 'AGENTS.md');
    expect(agents).toBeDefined();
    expect(agents!.mentionCount).toBeGreaterThan(0);
    expect(agents!.readCount).toBe(0);
  });

  it('returns zero stats for messages with no harness files', () => {
    const messages: NormalizedMessage[] = [
      msg({ toolCalls: [{ name: 'Read', input: 'src/index.ts' }] }),
      msg({ toolCalls: [{ name: 'Edit', input: 'src/server/routes.ts' }] }),
    ];

    const result = extractHarnessMetrics(messages);
    expect(result.totalArtifacts).toBe(0);
    expect(result.usedArtifacts).toBe(0);
    expect(result.unusedArtifacts).toBe(0);
  });

  it('tracks skill invocations', () => {
    const messages: NormalizedMessage[] = [
      msg({ toolCalls: [{ name: 'Skill', input: '{"name":"exec-plan"}' }] }),
      msg({ toolCalls: [{ name: 'Skill', input: '{"name":"exec-plan"}' }] }),
      msg({ toolCalls: [{ name: 'skill', input: 'brainstorming' }] }),
    ];

    const result = extractHarnessMetrics(messages);
    expect(result.skillInvocations).toHaveLength(2);
    const execPlan = result.skillInvocations.find(s => s.name === 'exec-plan');
    expect(execPlan).toBeDefined();
    expect(execPlan!.count).toBe(2);
    const brainstorm = result.skillInvocations.find(s => s.name === 'brainstorming');
    expect(brainstorm).toBeDefined();
    expect(brainstorm!.count).toBe(1);
  });

  it('detects ExecPlan usage via docs/plans/ operations', () => {
    const messages: NormalizedMessage[] = [
      msg({ toolCalls: [{ name: 'Write', input: 'docs/plans/active/2026-04-09/my-plan.md' }] }),
    ];

    const result = extractHarnessMetrics(messages);
    expect(result.execPlanUsed).toBe(true);
    // docs/plans/ should NOT appear as a harness artifact
    expect(result.artifacts.find(a => a.path.includes('docs/plans/'))).toBeUndefined();
  });

  it('detects docReadBeforeEdit pattern', () => {
    const messages: NormalizedMessage[] = [
      msg({ toolCalls: [{ name: 'Read', input: 'ARCHITECTURE.md' }] }),
      msg({ toolCalls: [{ name: 'Edit', input: 'src/server/routes.ts' }] }),
    ];

    const result = extractHarnessMetrics(messages);
    expect(result.docReadBeforeEdit).toBe(true);
  });

  it('docReadBeforeEdit is false when edit comes first', () => {
    const messages: NormalizedMessage[] = [
      msg({ toolCalls: [{ name: 'Edit', input: 'src/server/routes.ts' }] }),
      msg({ toolCalls: [{ name: 'Read', input: 'ARCHITECTURE.md' }] }),
    ];

    const result = extractHarnessMetrics(messages);
    expect(result.docReadBeforeEdit).toBe(false);
  });

  it('classifies docs/ subdirectories correctly (excluding docs/plans/)', () => {
    const messages: NormalizedMessage[] = [
      msg({ toolCalls: [{ name: 'Read', input: 'docs/guidance/local-dev-setup.md' }] }),
      msg({ toolCalls: [{ name: 'Read', input: 'docs/reference/code-patterns.md' }] }),
      msg({ toolCalls: [{ name: 'Read', input: 'docs/quality/debt-log.md' }] }),
      msg({ toolCalls: [{ name: 'Read', input: 'docs/AGENTS.md' }] }),
    ];

    const result = extractHarnessMetrics(messages);
    expect(result.totalArtifacts).toBe(4);

    const guidance = result.artifacts.find(a => a.path.includes('guidance'));
    expect(guidance!.category).toBe('doc-guidance');

    const reference = result.artifacts.find(a => a.path.includes('reference'));
    expect(reference!.category).toBe('doc-reference');

    const quality = result.artifacts.find(a => a.path.includes('quality'));
    expect(quality!.category).toBe('doc-quality');

    const docsAgents = result.artifacts.find(a => a.path === 'docs/AGENTS.md');
    expect(docsAgents!.category).toBe('doc-root');
  });

  it('does not count user message toolCalls as reads', () => {
    const messages: NormalizedMessage[] = [
      { role: 'user', timestamp: '', content: '', toolCalls: [{ name: 'Read', input: 'AGENTS.md' }] },
    ];

    const result = extractHarnessMetrics(messages);
    // user toolCalls are not scanned — only assistant toolCalls are
    expect(result.artifacts.filter(a => a.readCount > 0)).toHaveLength(0);
  });

  it('detects "Instructions from:" in system messages as injected', () => {
    const messages: NormalizedMessage[] = [
      {
        role: 'system',
        timestamp: '',
        content: 'Instructions from: /Users/foo/project/docs/reference/feishu-channel-and-binding-routing.md\nSome content here...',
      },
      msg({ content: 'I will work on the task.' }),
    ];

    const result = extractHarnessMetrics(messages);
    const artifact = result.artifacts.find(a => a.path.includes('feishu-channel'));
    expect(artifact).toBeDefined();
    expect(artifact!.injectedCount).toBe(1);
    expect(artifact!.category).toBe('doc-reference');
    expect(result.usedArtifacts).toBe(1);
  });

  it('detects harness paths embedded in user message content', () => {
    const messages: NormalizedMessage[] = [
      {
        role: 'user',
        timestamp: '',
        content: 'Please follow the rules in docs/rules/invariants.md and docs/guidance/local-dev-setup.md',
      },
      msg({ content: 'OK' }),
    ];

    const result = extractHarnessMetrics(messages);
    expect(result.artifacts.find(a => a.path.includes('invariants'))).toBeDefined();
    expect(result.artifacts.find(a => a.path.includes('local-dev-setup'))).toBeDefined();
    expect(result.usedArtifacts).toBe(2);
  });

  it('merges repo-scanned files not found in trace as unused', () => {
    const messages: NormalizedMessage[] = [
      msg({ toolCalls: [{ name: 'Read', input: 'AGENTS.md' }] }),
    ];

    const repoFiles = [
      'AGENTS.md',
      'docs/rules/invariants.md',
      'docs/guidance/local-dev-setup.md',
    ];

    const result = extractHarnessMetrics(messages, repoFiles);
    expect(result.repoScanned).toBe(true);
    expect(result.totalArtifacts).toBe(3);
    expect(result.usedArtifacts).toBe(1); // only AGENTS.md was Read
    expect(result.unusedArtifacts).toBe(2); // invariants + local-dev-setup

    const invariants = result.artifacts.find(a => a.path.includes('invariants'));
    expect(invariants).toBeDefined();
    expect(invariants!.readCount).toBe(0);
    expect(invariants!.mentionCount).toBe(0);
    expect(invariants!.injectedCount).toBe(0);
  });

  it('repoScanned is false when no repoHarnessFiles provided', () => {
    const result = extractHarnessMetrics([]);
    expect(result.repoScanned).toBe(false);
  });

  it('repoScanned is false when repoHarnessFiles is empty array', () => {
    const result = extractHarnessMetrics([], []);
    expect(result.repoScanned).toBe(false);
  });

  it('does not duplicate artifacts already found in trace', () => {
    const messages: NormalizedMessage[] = [
      msg({ toolCalls: [{ name: 'Read', input: 'docs/rules/invariants.md' }] }),
    ];

    const repoFiles = ['docs/rules/invariants.md', 'docs/guidance/setup.md'];

    const result = extractHarnessMetrics(messages, repoFiles);
    // invariants.md should appear once with readCount=1, not duplicated
    const invariantsEntries = result.artifacts.filter(a => a.path.includes('invariants'));
    expect(invariantsEntries).toHaveLength(1);
    expect(invariantsEntries[0]!.readCount).toBe(1);
    expect(result.totalArtifacts).toBe(2);
    expect(result.usedArtifacts).toBe(1);
  });
});
