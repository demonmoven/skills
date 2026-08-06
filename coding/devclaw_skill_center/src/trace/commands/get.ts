import { writeFile } from 'node:fs/promises';
import { basename, resolve } from 'node:path';
import { gunzipSync } from 'node:zlib';
import { loadConfig } from '../config/loader.js';
import { downloadRawBuffer } from '../server/tos-reader.js';

export interface GetOptions {
  output?: string;
  gunzip?: boolean;
}

export async function runGet(key: string, options: GetOptions): Promise<void> {
  const config = await loadConfig(process.cwd());
  const normalized = key.replace(/^\//, '');
  const result = await downloadRawBuffer(config.tos, normalized);
  if (!result) {
    throw new Error(`Object not found or unreadable: ${key}`);
  }

  const finalBuf = options.gunzip ? gunzipSync(result.buffer) : result.buffer;
  const outPath = resolveOutPath(normalized, options.output, options.gunzip === true);
  await writeFile(outPath, finalBuf);
  process.stdout.write(`Saved ${finalBuf.length} bytes to ${outPath}\n`);
}

function resolveOutPath(normalizedKey: string, output: string | undefined, gunzip: boolean): string {
  if (output) return resolve(output);
  let name = basename(normalizedKey);
  if (gunzip && name.endsWith('.gz')) {
    name = name.slice(0, -3);
  }
  return resolve(process.cwd(), name);
}
