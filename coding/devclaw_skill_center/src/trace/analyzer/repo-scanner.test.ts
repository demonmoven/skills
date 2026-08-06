import { describe, it, expect } from 'vitest';
import { classifyPath, normalizePath } from './harness-metrics.js';

describe('classifyPath (exported for repo-scanner)', () => {
  it('classifies root-level docs', () => {
    expect(classifyPath('AGENTS.md')).toBe('doc-root');
    expect(classifyPath('CLAUDE.md')).toBe('doc-root');
    expect(classifyPath('ARCHITECTURE.md')).toBe('doc-root');
    expect(classifyPath('README.md')).toBe('doc-root');
  });

  it('classifies docs/ subdirectories', () => {
    expect(classifyPath('docs/rules/invariants.md')).toBe('doc-rules');
    expect(classifyPath('docs/guidance/local-dev-setup.md')).toBe('doc-guidance');
    expect(classifyPath('docs/reference/code-patterns.md')).toBe('doc-reference');
    expect(classifyPath('docs/quality/debt-log.md')).toBe('doc-quality');
    expect(classifyPath('docs/AGENTS.md')).toBe('doc-root');
  });

  it('classifies other docs/ files as doc-other', () => {
    expect(classifyPath('docs/some-random-doc.md')).toBe('doc-other');
  });

  it('excludes docs/plans/', () => {
    expect(classifyPath('docs/plans/active/2026-04-09/my-plan.md')).toBeNull();
    expect(classifyPath('docs/plans/proposal/plan.md')).toBeNull();
  });

  it('classifies skill files', () => {
    expect(classifyPath('.skills/exec-plan/SKILL.md')).toBe('skill');
    expect(classifyPath('.claude/skills/my-skill/SKILL.md')).toBe('skill');
    expect(classifyPath('.opencode/skills/my-skill/SKILL.md')).toBe('skill');
  });

  it('returns null for non-harness files', () => {
    expect(classifyPath('src/index.ts')).toBeNull();
    expect(classifyPath('package.json')).toBeNull();
    expect(classifyPath('web/src/App.tsx')).toBeNull();
  });
});

describe('normalizePath (exported for repo-scanner)', () => {
  it('strips absolute path prefixes for docs/', () => {
    expect(normalizePath('/Users/foo/project/docs/rules/invariants.md')).toBe('docs/rules/invariants.md');
  });

  it('strips absolute path prefixes for root docs', () => {
    expect(normalizePath('/Users/foo/project/AGENTS.md')).toBe('AGENTS.md');
  });

  it('passes through already-relative paths', () => {
    expect(normalizePath('docs/guidance/setup.md')).toBe('docs/guidance/setup.md');
  });

  it('strips leading ./', () => {
    expect(normalizePath('./ARCHITECTURE.md')).toBe('ARCHITECTURE.md');
  });
});
