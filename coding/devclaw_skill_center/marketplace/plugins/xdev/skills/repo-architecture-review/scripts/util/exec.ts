// scripts/util/exec.ts
import { execa, ExecaError } from 'execa';

export interface CmdResult {
  status: 'ok' | 'failed' | 'timeout';
  stdout: string;
  stderr: string;
  exitCode: number;
  durationMs: number;
}

export interface CmdOptions {
  cwd?: string;
  env?: Record<string, string>;
  timeoutMs?: number;
  input?: string;
}

export async function runCmd(cmd: string, args: string[], opts: CmdOptions = {}): Promise<CmdResult> {
  const start = Date.now();
  try {
    const res = await execa(cmd, args, {
      cwd: opts.cwd,
      env: opts.env ? { ...process.env, ...opts.env } : process.env,
      timeout: opts.timeoutMs ?? 300_000,
      input: opts.input,
      reject: false,
    });
    const status: CmdResult['status'] = res.timedOut ? 'timeout' : (res.exitCode === 0 ? 'ok' : 'failed');
    return {
      status,
      stdout: res.stdout ?? '',
      stderr: res.stderr ?? '',
      exitCode: res.exitCode ?? -1,
      durationMs: Date.now() - start,
    };
  } catch (e) {
    const err = e as ExecaError;
    return {
      status: err.timedOut ? 'timeout' : 'failed',
      stdout: err.stdout?.toString() ?? '',
      stderr: err.stderr?.toString() ?? err.message,
      exitCode: err.exitCode ?? -1,
      durationMs: Date.now() - start,
    };
  }
}
