import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtempSync, rmSync, writeFileSync, mkdirSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { collectForwardEntries } from './download.js';

let baseDir: string;

function writeLog(date: string, lines: string[]): void {
  mkdirSync(baseDir, { recursive: true });
  writeFileSync(join(baseDir, `hook.${date}.log`), lines.join('\n') + '\n');
}

function line(date: string, payload: Record<string, unknown>, event = 'forward'): string {
  return `${date}T12:00:00.000+0800 ${event} ${JSON.stringify(payload)}`;
}

beforeEach(() => {
  baseDir = mkdtempSync(join(tmpdir(), 'xdev-download-'));
});

afterEach(() => {
  rmSync(baseDir, { recursive: true, force: true });
});

describe('collectForwardEntries', () => {
  it('returns empty when log directory is missing', () => {
    rmSync(baseDir, { recursive: true, force: true });
    expect(collectForwardEntries({ lines: 5, baseDir })).toEqual([]);
  });

  it('parses recent forward:ok entries in reverse chronological order', () => {
    writeLog('2026-04-17', [
      line('2026-04-17', { result: 'ok', sessionId: 's1', tool: 'claude-code', jsonlObjectKey: '/k/s1.gz', bucket: 'b' }),
      line('2026-04-17', { result: 'ok', sessionId: 's2', tool: 'claude-code', jsonlObjectKey: '/k/s2.gz', bucket: 'b' }),
      line('2026-04-17', { result: 'ok', sessionId: 's3', tool: 'claude-code', jsonlObjectKey: '/k/s3.gz', bucket: 'b' }),
    ]);
    const got = collectForwardEntries({ lines: 10, date: '2026-04-17', baseDir });
    expect(got.map((e) => e.sessionId)).toEqual(['s3', 's2', 's1']);
  });

  it('skips forward:error entries', () => {
    writeLog('2026-04-17', [
      line('2026-04-17', { result: 'ok', sessionId: 's1', tool: 'cc', jsonlObjectKey: '/k/s1.gz', bucket: 'b' }),
      line('2026-04-17', { result: 'error', errorCode: 'X', errorMessage: 'y' }),
      line('2026-04-17', { result: 'ok', sessionId: 's2', tool: 'cc', jsonlObjectKey: '/k/s2.gz', bucket: 'b' }),
    ]);
    const got = collectForwardEntries({ lines: 10, date: '2026-04-17', baseDir });
    expect(got.map((e) => e.sessionId)).toEqual(['s2', 's1']);
  });

  it('deduplicates by sessionId, keeping latest occurrence', () => {
    writeLog('2026-04-17', [
      line('2026-04-17', { result: 'ok', sessionId: 's1', tool: 'cc', jsonlObjectKey: '/old/s1.gz', bucket: 'b' }),
      line('2026-04-17', { result: 'ok', sessionId: 's1', tool: 'cc', jsonlObjectKey: '/new/s1.gz', bucket: 'b' }),
    ]);
    const got = collectForwardEntries({ lines: 10, date: '2026-04-17', baseDir });
    expect(got).toHaveLength(1);
    expect(got[0]?.jsonlObjectKey).toBe('/new/s1.gz');
  });

  it('caps at --lines', () => {
    const entries = Array.from({ length: 20 }, (_, i) =>
      line('2026-04-17', { result: 'ok', sessionId: `s${i}`, tool: 'cc', jsonlObjectKey: `/k/s${i}.gz`, bucket: 'b' }),
    );
    writeLog('2026-04-17', entries);
    const got = collectForwardEntries({ lines: 3, date: '2026-04-17', baseDir });
    expect(got).toHaveLength(3);
  });

  it('filters by --tool', () => {
    writeLog('2026-04-17', [
      line('2026-04-17', { result: 'ok', sessionId: 's1', tool: 'claude-code', jsonlObjectKey: '/k/s1.gz', bucket: 'b' }),
      line('2026-04-17', { result: 'ok', sessionId: 's2', tool: 'opencode', jsonlObjectKey: '/k/s2.gz', bucket: 'b' }),
      line('2026-04-17', { result: 'ok', sessionId: 's3', tool: 'claude-code', jsonlObjectKey: '/k/s3.gz', bucket: 'b' }),
    ]);
    const got = collectForwardEntries({ lines: 10, date: '2026-04-17', tool: 'claude-code', baseDir });
    expect(got.map((e) => e.sessionId)).toEqual(['s3', 's1']);
  });

  it('filters by --session with prefix match', () => {
    writeLog('2026-04-17', [
      line('2026-04-17', { result: 'ok', sessionId: 'abc-111', tool: 'cc', jsonlObjectKey: '/k/a.gz', bucket: 'b' }),
      line('2026-04-17', { result: 'ok', sessionId: 'xyz-222', tool: 'cc', jsonlObjectKey: '/k/x.gz', bucket: 'b' }),
      line('2026-04-17', { result: 'ok', sessionId: 'abc-333', tool: 'cc', jsonlObjectKey: '/k/c.gz', bucket: 'b' }),
    ]);
    const got = collectForwardEntries({ lines: 10, date: '2026-04-17', session: 'abc', baseDir });
    expect(got.map((e) => e.sessionId).sort()).toEqual(['abc-111', 'abc-333']);
  });

  it('scans multiple days when --date not given (newest first)', () => {
    writeLog('2026-04-15', [
      line('2026-04-15', { result: 'ok', sessionId: 'old-a', tool: 'cc', jsonlObjectKey: '/k/a.gz', bucket: 'b' }),
    ]);
    writeLog('2026-04-17', [
      line('2026-04-17', { result: 'ok', sessionId: 'new-b', tool: 'cc', jsonlObjectKey: '/k/b.gz', bucket: 'b' }),
    ]);
    const got = collectForwardEntries({ lines: 10, baseDir });
    expect(got.map((e) => e.sessionId)).toEqual(['new-b', 'old-a']);
  });

  it('ignores malformed JSON lines', () => {
    writeLog('2026-04-17', [
      'not-even-a-log-line',
      '2026-04-17T12:00:00.000+0800 forward {broken-json}',
      line('2026-04-17', { result: 'ok', sessionId: 's1', tool: 'cc', jsonlObjectKey: '/k/s1.gz', bucket: 'b' }),
    ]);
    const got = collectForwardEntries({ lines: 10, date: '2026-04-17', baseDir });
    expect(got).toHaveLength(1);
    expect(got[0]?.sessionId).toBe('s1');
  });

  it('captures diffObjectKey when present', () => {
    writeLog('2026-04-17', [
      line('2026-04-17', {
        result: 'ok',
        sessionId: 's1',
        tool: 'cc',
        jsonlObjectKey: '/k/s1.gz',
        diffObjectKey: '/k/s1.diff.gz',
        bucket: 'b',
      }),
    ]);
    const got = collectForwardEntries({ lines: 10, date: '2026-04-17', baseDir });
    expect(got[0]?.diffObjectKey).toBe('/k/s1.diff.gz');
  });

  it('uses snapshotDiffsKey as the downloadable diff artifact', () => {
    writeLog('2026-04-17', [
      line('2026-04-17', {
        result: 'ok',
        sessionId: 's1',
        tool: 'cc',
        jsonlObjectKey: '/k/s1.gz',
        snapshotDiffsKey: '/k/s1.diffs.json.gz',
        bucket: 'b',
      }),
    ]);
    const got = collectForwardEntries({ lines: 10, date: '2026-04-17', baseDir });
    expect(got[0]?.diffObjectKey).toBe('/k/s1.diffs.json.gz');
  });
});
