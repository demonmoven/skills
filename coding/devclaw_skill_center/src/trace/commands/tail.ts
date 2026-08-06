import { existsSync, readFileSync, statSync, createReadStream } from 'node:fs';
import { resolveLogPath } from '../utils/hook-log.js';

export interface TailOptions {
  lines: number;
  follow?: boolean;
  followRetry?: boolean;
  date?: string;
  baseDir?: string;
}

export async function runTail(options: TailOptions): Promise<void> {
  const targetDate = parseDate(options.date);
  const initialPath = resolveLogPath(targetDate, options.baseDir);

  printLastLines(initialPath, options.lines);

  if (!options.follow && !options.followRetry) return;

  if (options.followRetry) {
    await followAcrossDates(initialPath, options);
  } else {
    await followSingleFile(initialPath);
  }
}

function parseDate(input: string | undefined): Date {
  if (!input) return new Date();
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(input);
  if (!m) {
    throw new Error(`Invalid --date format: "${input}" (expected YYYY-MM-DD)`);
  }
  const [y, mo, d] = [m[1], m[2], m[3]].map(Number) as [number, number, number];
  const parsed = new Date(y, mo - 1, d);
  if (parsed.getFullYear() !== y || parsed.getMonth() !== mo - 1 || parsed.getDate() !== d) {
    throw new Error(`Invalid date: "${input}"`);
  }
  return parsed;
}

function printLastLines(path: string, n: number): void {
  if (!existsSync(path)) {
    process.stderr.write(`[trace] no log file at ${path}\n`);
    return;
  }
  const content = readFileSync(path, 'utf-8');
  const lines = content.split('\n');
  const trailingEmpty = lines.length > 0 && lines[lines.length - 1] === '' ? 1 : 0;
  const effective = lines.slice(0, lines.length - trailingEmpty);
  const tail = effective.slice(Math.max(0, effective.length - n));
  if (tail.length > 0) {
    process.stdout.write(tail.join('\n') + '\n');
  }
}

async function followSingleFile(path: string): Promise<void> {
  let size = existsSync(path) ? statSync(path).size : 0;
  return new Promise<void>((resolve) => {
    const interval = setInterval(() => {
      if (!existsSync(path)) return;
      const newSize = statSync(path).size;
      if (newSize > size) {
        streamRange(path, size, newSize);
        size = newSize;
      } else if (newSize < size) {
        size = newSize;
      }
    }, 500);
    process.on('SIGINT', () => {
      clearInterval(interval);
      resolve();
      process.exit(0);
    });
  });
}

async function followAcrossDates(initialPath: string, options: TailOptions): Promise<void> {
  let currentPath = initialPath;
  let size = existsSync(currentPath) ? statSync(currentPath).size : 0;

  return new Promise<void>((resolve) => {
    const interval = setInterval(() => {
      const todayPath = resolveLogPath(new Date(), options.baseDir);
      if (todayPath !== currentPath) {
        if (existsSync(currentPath)) {
          const finalSize = statSync(currentPath).size;
          if (finalSize > size) streamRange(currentPath, size, finalSize);
        }
        currentPath = todayPath;
        size = existsSync(currentPath) ? statSync(currentPath).size : 0;
        process.stderr.write(`[trace] rolled over to ${currentPath}\n`);
        return;
      }
      if (!existsSync(currentPath)) return;
      const newSize = statSync(currentPath).size;
      if (newSize > size) {
        streamRange(currentPath, size, newSize);
        size = newSize;
      } else if (newSize < size) {
        size = newSize;
      }
    }, 1000);
    process.on('SIGINT', () => {
      clearInterval(interval);
      resolve();
      process.exit(0);
    });
  });
}

function streamRange(path: string, start: number, end: number): void {
  const stream = createReadStream(path, { start, end: end - 1, encoding: 'utf-8' });
  stream.on('data', (chunk) => process.stdout.write(chunk));
}
