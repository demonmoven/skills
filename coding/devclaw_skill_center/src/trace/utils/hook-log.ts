import { appendFileSync, mkdirSync } from 'node:fs';
import { homedir } from 'node:os';
import { dirname, join } from 'node:path';

export type HookLogEvent = 'session_start' | 'forward' | 'forward_dry_run';
export type HookLogResult = 'ok' | 'error' | 'dry-run';

export interface HookLogEntry {
  event: HookLogEvent;
  result: HookLogResult;
  sessionId?: string;
  tool?: string;
  jsonlObjectKey?: string;
  diffObjectKey?: string;
  bucket?: string;
  etag?: string;
  dryRunOutput?: string;
  gitCommit?: string;
  gitBranch?: string;
  errorCode?: string;
  errorMessage?: string;
  [extra: string]: unknown;
}

export function resolveLogDir(baseDir?: string): string {
  return baseDir ?? join(homedir(), '.trace', 'logs');
}

export function resolveLogPath(date: Date = new Date(), baseDir?: string): string {
  const yyyy = date.getFullYear();
  const mm = String(date.getMonth() + 1).padStart(2, '0');
  const dd = String(date.getDate()).padStart(2, '0');
  return join(resolveLogDir(baseDir), `hook.${yyyy}-${mm}-${dd}.log`);
}

function isoWithTz(date: Date): string {
  const pad = (n: number): string => String(n).padStart(2, '0');
  const padMs = (n: number): string => String(n).padStart(3, '0');
  const tzMin = -date.getTimezoneOffset();
  const sign = tzMin >= 0 ? '+' : '-';
  const tzH = pad(Math.floor(Math.abs(tzMin) / 60));
  const tzM = pad(Math.abs(tzMin) % 60);
  return (
    `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}` +
    `T${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}` +
    `.${padMs(date.getMilliseconds())}${sign}${tzH}${tzM}`
  );
}

export function appendHookLog(entry: HookLogEntry, baseDir?: string): void {
  const now = new Date();
  const logPath = resolveLogPath(now, baseDir);
  const { event, ...rest } = entry;
  const line = `${isoWithTz(now)} ${event} ${JSON.stringify(rest)}\n`;

  try {
    mkdirSync(dirname(logPath), { recursive: true, mode: 0o700 });
    appendFileSync(logPath, line, { mode: 0o600 });
  } catch (err) {
    console.error(`[trace] WARN: failed to write hook log to ${logPath}: ${(err as Error).message}`);
  }
}
