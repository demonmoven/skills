import { runDownload } from './download.js';

export interface OpenOptions {
  lines?: number;
  expires?: string;
  tool?: string;
  session?: string;
  date?: string;
  withDiff?: boolean;
}

export async function runOpen(options: OpenOptions): Promise<void> {
  const n = options.lines ?? 5;
  if (isNaN(n) || n <= 0) throw new Error(`Invalid --lines value: ${options.lines}`);
  await runDownload({
    lines: n,
    open: true,
    expires: options.expires,
    tool: options.tool,
    session: options.session,
    date: options.date,
    withDiff: options.withDiff,
  });
}
