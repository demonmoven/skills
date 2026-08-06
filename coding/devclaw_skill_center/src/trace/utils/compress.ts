import { createReadStream, createWriteStream } from 'node:fs';
import { createGzip } from 'node:zlib';
import { pipeline } from 'node:stream/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { randomUUID } from 'node:crypto';
import { unlink } from 'node:fs/promises';

/**
 * Compress a file using gzip. Returns the path to the compressed temp file.
 * Caller is responsible for cleaning up the temp file after use.
 */
export async function gzipFile(inputPath: string): Promise<string> {
  const outputPath = join(tmpdir(), `trace-${randomUUID()}.gz`);

  await pipeline(
    createReadStream(inputPath),
    createGzip({ level: 6 }),
    createWriteStream(outputPath),
  );

  return outputPath;
}

/**
 * Compress a string to a gzip temp file. Returns the path.
 */
export async function gzipString(content: string): Promise<string> {
  const outputPath = join(tmpdir(), `trace-${randomUUID()}.gz`);

  const { Readable } = await import('node:stream');
  const readable = Readable.from([content]);

  await pipeline(
    readable,
    createGzip({ level: 6 }),
    createWriteStream(outputPath),
  );

  return outputPath;
}

/**
 * Clean up a temporary file (best-effort, no error on missing).
 */
export async function cleanupTempFile(filePath: string): Promise<void> {
  try {
    await unlink(filePath);
  } catch {
    // Ignore cleanup errors
  }
}
