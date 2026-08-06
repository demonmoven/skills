import { spawn } from 'node:child_process';

export interface GitResult {
  cmd: string[];
  cwd: string;
  exitCode: number;
  stdout: string;
  stderr: string;
}

/**
 * Spawn `git <args>` in `cwd`, capturing stdout/stderr. Resolves to a
 * GitResult regardless of exit code; the caller decides how to react.
 *
 * If `inheritStdout` is true, git's stdout/stderr stream straight to the
 * parent process (used by interactive `clone` to show progress bars).
 */
export function gitSpawn(
  args: string[],
  cwd: string,
  opts: { inheritStdout?: boolean } = {},
): Promise<GitResult> {
  return new Promise((resolve) => {
    const stdoutChunks: Buffer[] = [];
    const stderrChunks: Buffer[] = [];
    const stdio = opts.inheritStdout
      ? (['ignore', 'inherit', 'inherit'] as const)
      : (['ignore', 'pipe', 'pipe'] as const);
    const child = spawn('git', args, { cwd, stdio });
    if (!opts.inheritStdout) {
      child.stdout?.on('data', (c: Buffer) => stdoutChunks.push(c));
      child.stderr?.on('data', (c: Buffer) => stderrChunks.push(c));
    }
    child.on('close', (code: number | null) => {
      resolve({
        cmd: ['git', ...args],
        cwd,
        exitCode: code ?? -1,
        stdout: Buffer.concat(stdoutChunks).toString('utf-8'),
        stderr: Buffer.concat(stderrChunks).toString('utf-8'),
      });
    });
  });
}

export class GitError extends Error {
  constructor(public readonly result: GitResult) {
    super(
      `git ${result.cmd.slice(1).join(' ')} (cwd=${result.cwd}) 失败 [exit ${result.exitCode}]\n` +
        (result.stderr || result.stdout),
    );
    this.name = 'GitError';
  }
}

/** Throws GitError on non-zero exit. Returns the result on success. */
export async function gitOrThrow(args: string[], cwd: string): Promise<GitResult> {
  const r = await gitSpawn(args, cwd);
  if (r.exitCode !== 0) throw new GitError(r);
  return r;
}

/**
 * Run `tasks` with a fixed concurrency cap. Each task is a thunk to defer
 * starting work until a slot opens. Returns PromiseSettledResult[] in input
 * order, so caller can map back to a parallel "names" list for reporting.
 */
export async function runConcurrent<T>(
  tasks: Array<() => Promise<T>>,
  parallelism: number,
): Promise<PromiseSettledResult<T>[]> {
  const results: PromiseSettledResult<T>[] = new Array(tasks.length);
  let next = 0;
  const workers: Promise<void>[] = [];
  const limit = Math.max(1, Math.min(parallelism, tasks.length));
  for (let w = 0; w < limit; w++) {
    workers.push(
      (async () => {
        while (true) {
          const i = next++;
          if (i >= tasks.length) return;
          try {
            const value = await tasks[i]();
            results[i] = { status: 'fulfilled', value };
          } catch (reason) {
            results[i] = { status: 'rejected', reason };
          }
        }
      })(),
    );
  }
  await Promise.all(workers);
  return results;
}

/**
 * Pretty-print a row-per-task summary of runConcurrent results, returning
 * how many tasks failed so the caller can set process.exitCode.
 */
export function summarizeResults<T>(
  results: PromiseSettledResult<T>[],
  names: string[],
  label: string,
): number {
  let failed = 0;
  console.log(`[xdev] ${label} 汇总：`);
  results.forEach((r, i) => {
    const name = names[i] ?? `<#${i}>`;
    if (r.status === 'fulfilled') {
      console.log(`  ✓ ${name}`);
    } else {
      failed++;
      const msg = r.reason instanceof Error ? r.reason.message : String(r.reason);
      const firstLine = msg.split('\n')[0];
      console.log(`  ✗ ${name} — ${firstLine}`);
    }
  });
  console.log(`[xdev] 成功 ${results.length - failed} / 失败 ${failed} / 总计 ${results.length}`);
  return failed;
}
