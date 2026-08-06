import { describe, it, expect, afterEach } from 'vitest';
import { writeFile, readFile, rm } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { createGunzip } from 'node:zlib';
import { createReadStream } from 'node:fs';
import { gzipFile, gzipString, cleanupTempFile } from './compress.js';

const tempFiles: string[] = [];

afterEach(async () => {
  for (const f of tempFiles) {
    await rm(f, { force: true });
  }
  tempFiles.length = 0;
});

async function gunzipToString(gzPath: string): Promise<string> {
  return new Promise((resolve, reject) => {
    const chunks: Buffer[] = [];
    createReadStream(gzPath)
      .pipe(createGunzip())
      .on('data', (chunk: Buffer) => chunks.push(chunk))
      .on('end', () => resolve(Buffer.concat(chunks).toString('utf-8')))
      .on('error', reject);
  });
}

describe('gzipFile', () => {
  it('compresses a file and produces valid gzip', async () => {
    const inputPath = join(tmpdir(), `trace-test-input-${Date.now()}.txt`);
    await writeFile(inputPath, 'hello world\nline 2\n');
    tempFiles.push(inputPath);

    const gzPath = await gzipFile(inputPath);
    tempFiles.push(gzPath);

    expect(gzPath.endsWith('.gz')).toBe(true);
    const decompressed = await gunzipToString(gzPath);
    expect(decompressed).toBe('hello world\nline 2\n');
  });
});

describe('gzipString', () => {
  it('compresses a string to a gzip file', async () => {
    const content = 'diff --git a/file.ts b/file.ts\n+new line\n';
    const gzPath = await gzipString(content);
    tempFiles.push(gzPath);

    const decompressed = await gunzipToString(gzPath);
    expect(decompressed).toBe(content);
  });
});

describe('cleanupTempFile', () => {
  it('removes existing file', async () => {
    const filePath = join(tmpdir(), `trace-cleanup-test-${Date.now()}.tmp`);
    await writeFile(filePath, 'temp');

    await cleanupTempFile(filePath);

    await expect(readFile(filePath)).rejects.toThrow();
  });

  it('does not throw for non-existent file', async () => {
    await expect(cleanupTempFile('/tmp/nonexistent-file-xyz')).resolves.toBeUndefined();
  });
});
