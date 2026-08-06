import { describe, expect, it, beforeEach, afterEach } from 'vitest';
import { existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { appendGitignoreLines } from './gitignore-edit.js';

describe('appendGitignoreLines', () => {
  let dir: string;

  beforeEach(() => {
    dir = mkdtempSync(join(tmpdir(), 'xdev-ignore-test-'));
  });

  afterEach(() => {
    rmSync(dir, { recursive: true, force: true });
  });

  it('creates .gitignore when missing', () => {
    const added = appendGitignoreLines(dir, ['.claude/', 'repos/']);
    expect(added).toEqual(['.claude/', 'repos/']);
    const content = readFileSync(join(dir, '.gitignore'), 'utf-8');
    expect(content).toBe('.claude/\nrepos/\n');
  });

  it('skips lines already present (idempotent)', () => {
    writeFileSync(join(dir, '.gitignore'), '.claude/\n');
    const added = appendGitignoreLines(dir, ['.claude/', 'repos/']);
    expect(added).toEqual(['repos/']);
    const content = readFileSync(join(dir, '.gitignore'), 'utf-8');
    expect(content).toBe('.claude/\nrepos/\n');
  });

  it('returns [] when all lines already present', () => {
    writeFileSync(join(dir, '.gitignore'), '.claude/\nrepos/\n');
    const added = appendGitignoreLines(dir, ['.claude/', 'repos/']);
    expect(added).toEqual([]);
  });

  it('preserves existing content and adds trailing newline if missing', () => {
    writeFileSync(join(dir, '.gitignore'), 'node_modules\ndist');
    const added = appendGitignoreLines(dir, ['.claude/']);
    expect(added).toEqual(['.claude/']);
    const content = readFileSync(join(dir, '.gitignore'), 'utf-8');
    expect(content).toBe('node_modules\ndist\n.claude/\n');
  });

  it('compares by trim (whitespace tolerant)', () => {
    writeFileSync(join(dir, '.gitignore'), '  .claude/  \n');
    const added = appendGitignoreLines(dir, ['.claude/']);
    expect(added).toEqual([]);
  });
});
