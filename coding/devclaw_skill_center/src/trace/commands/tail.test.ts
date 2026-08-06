import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { mkdtempSync, rmSync, writeFileSync, mkdirSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { runTail } from './tail.js';

let baseDir: string;
let stdoutChunks: string[];
let stderrChunks: string[];
let stdoutSpy: ReturnType<typeof vi.spyOn>;
let stderrSpy: ReturnType<typeof vi.spyOn>;

function todayLogPath(): string {
  const now = new Date();
  const y = now.getFullYear();
  const m = String(now.getMonth() + 1).padStart(2, '0');
  const d = String(now.getDate()).padStart(2, '0');
  return join(baseDir, `hook.${y}-${m}-${d}.log`);
}

beforeEach(() => {
  baseDir = mkdtempSync(join(tmpdir(), 'xdev-tail-'));
  mkdirSync(baseDir, { recursive: true });
  stdoutChunks = [];
  stderrChunks = [];
  stdoutSpy = vi.spyOn(process.stdout, 'write').mockImplementation((chunk: unknown) => {
    stdoutChunks.push(typeof chunk === 'string' ? chunk : chunk?.toString() ?? '');
    return true;
  });
  stderrSpy = vi.spyOn(process.stderr, 'write').mockImplementation((chunk: unknown) => {
    stderrChunks.push(typeof chunk === 'string' ? chunk : chunk?.toString() ?? '');
    return true;
  });
});

afterEach(() => {
  stdoutSpy.mockRestore();
  stderrSpy.mockRestore();
  rmSync(baseDir, { recursive: true, force: true });
});

describe('runTail', () => {
  it('prints last N lines of today\'s log', async () => {
    const path = todayLogPath();
    const allLines = Array.from({ length: 20 }, (_, i) => `line-${i + 1}`);
    writeFileSync(path, allLines.join('\n') + '\n');

    await runTail({ lines: 5, baseDir });

    const out = stdoutChunks.join('');
    const printed = out.trim().split('\n');
    expect(printed).toEqual(['line-16', 'line-17', 'line-18', 'line-19', 'line-20']);
  });

  it('honors --date flag to show a specific date', async () => {
    writeFileSync(join(baseDir, 'hook.2025-12-31.log'), 'old-year-line\n');

    await runTail({ lines: 10, date: '2025-12-31', baseDir });

    expect(stdoutChunks.join('').trim()).toBe('old-year-line');
  });

  it('warns to stderr when the target log file does not exist', async () => {
    await runTail({ lines: 10, baseDir });

    expect(stdoutChunks.join('')).toBe('');
    expect(stderrChunks.join('')).toMatch(/no log file at/);
  });

  it('rejects invalid --date format', async () => {
    await expect(runTail({ lines: 10, date: '2025/12/31', baseDir })).rejects.toThrow(
      /Invalid --date format/,
    );
  });

  it('rejects invalid date values (e.g. Feb 30)', async () => {
    await expect(runTail({ lines: 10, date: '2025-02-30', baseDir })).rejects.toThrow(/Invalid date/);
  });

  it('prints nothing but does not crash when log file is empty', async () => {
    writeFileSync(todayLogPath(), '');
    await runTail({ lines: 10, baseDir });
    expect(stdoutChunks.join('')).toBe('');
  });

  it('handles file with no trailing newline', async () => {
    writeFileSync(todayLogPath(), 'a\nb\nc');
    await runTail({ lines: 10, baseDir });
    expect(stdoutChunks.join('').trim().split('\n')).toEqual(['a', 'b', 'c']);
  });
});
