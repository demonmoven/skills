import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { mkdtempSync, rmSync, readFileSync, existsSync, readdirSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { appendHookLog, resolveLogPath } from './hook-log.js';

let baseDir: string;

beforeEach(() => {
  baseDir = mkdtempSync(join(tmpdir(), 'xdev-hooklog-'));
});

afterEach(() => {
  rmSync(baseDir, { recursive: true, force: true });
  vi.useRealTimers();
});

describe('resolveLogPath', () => {
  it('formats date as YYYY-MM-DD with zero-padding', () => {
    const path = resolveLogPath(new Date(2026, 0, 3), baseDir);
    expect(path).toBe(join(baseDir, 'hook.2026-01-03.log'));
  });

  it('handles December correctly', () => {
    const path = resolveLogPath(new Date(2026, 11, 25), baseDir);
    expect(path).toBe(join(baseDir, 'hook.2026-12-25.log'));
  });
});

describe('appendHookLog', () => {
  it('writes a single line with ISO timestamp + event + JSON payload', () => {
    appendHookLog({ event: 'session_start', result: 'ok', sessionId: 'abc' }, baseDir);
    const files = readdirSync(baseDir);
    expect(files).toHaveLength(1);
    const content = readFileSync(join(baseDir, files[0] ?? ''), 'utf-8');
    expect(content.endsWith('\n')).toBe(true);
    const [ts, event, ...rest] = content.trim().split(' ');
    expect(ts).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}[+-]\d{4}$/);
    expect(event).toBe('session_start');
    const payload = JSON.parse(rest.join(' '));
    expect(payload).toEqual({ result: 'ok', sessionId: 'abc' });
  });

  it('appends multiple lines to the same file for same day', () => {
    appendHookLog({ event: 'session_start', result: 'ok', sessionId: '1' }, baseDir);
    appendHookLog({ event: 'forward', result: 'ok', sessionId: '1', jsonlObjectKey: 'foo/bar.gz' }, baseDir);
    const files = readdirSync(baseDir);
    expect(files).toHaveLength(1);
    const content = readFileSync(join(baseDir, files[0] ?? ''), 'utf-8');
    const lines = content.trim().split('\n');
    expect(lines).toHaveLength(2);
  });

  it('writes to different files for different days', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(2026, 3, 17, 10, 0, 0));
    appendHookLog({ event: 'session_start', result: 'ok', sessionId: 'a' }, baseDir);
    vi.setSystemTime(new Date(2026, 3, 18, 10, 0, 0));
    appendHookLog({ event: 'session_start', result: 'ok', sessionId: 'b' }, baseDir);
    const files = readdirSync(baseDir).sort();
    expect(files).toEqual(['hook.2026-04-17.log', 'hook.2026-04-18.log']);
  });

  it('creates parent directory if missing', () => {
    const nested = join(baseDir, 'nested', 'logs');
    expect(existsSync(nested)).toBe(false);
    appendHookLog({ event: 'session_start', result: 'ok' }, nested);
    expect(existsSync(nested)).toBe(true);
  });

  it('includes forward TOS fields in payload', () => {
    appendHookLog(
      {
        event: 'forward',
        result: 'ok',
        sessionId: 'sid',
        tool: 'claude-code',
        jsonlObjectKey: 'xtrace/u/2026-04-17/claude-code/sid.session.tar.gz',
        bucket: 'stone-costudio-boe',
        etag: 'abc123',
      },
      baseDir,
    );
    const files = readdirSync(baseDir);
    const content = readFileSync(join(baseDir, files[0] ?? ''), 'utf-8').trim();
    const jsonStart = content.indexOf('{');
    const payload = JSON.parse(content.slice(jsonStart));
    expect(payload.tool).toBe('claude-code');
    expect(payload.jsonlObjectKey).toMatch(/sid\.session\.tar\.gz$/);
    expect(payload.bucket).toBe('stone-costudio-boe');
    expect(payload.etag).toBe('abc123');
  });

  it('does not throw when the log directory cannot be created', () => {
    // Pass a path that starts with a file (so mkdir will fail)
    const fakeFile = join(baseDir, 'blocker');
    require('node:fs').writeFileSync(fakeFile, 'occupied');
    const underFile = join(fakeFile, 'impossible');
    expect(() => {
      appendHookLog({ event: 'session_start', result: 'ok' }, underFile);
    }).not.toThrow();
  });
});
