import { loadConfig } from '../config/loader.js';
import { generatePresignedUrl } from '../upload/tos.js';
import { parseDuration } from '../utils/duration.js';
import { openInBrowser, copyToClipboard } from '../utils/open.js';

export interface UrlOptions {
  expires?: string;
  open?: boolean;
  copy?: boolean;
}

export async function runUrl(key: string, options: UrlOptions): Promise<void> {
  const config = await loadConfig(process.cwd());
  if (!config.tos.endpoint) {
    throw new Error('tos.endpoint not configured. Set via: xdev trace config set tos.endpoint tos-cn-north-boe.byted.org');
  }
  const expiresSec = parseDuration(options.expires ?? '24h');
  const url = generatePresignedUrl(config.tos, key, expiresSec);
  process.stdout.write(url + '\n');
  if (options.copy) {
    await copyToClipboard(url);
    process.stderr.write('[trace] Copied to clipboard\n');
  }
  if (options.open) {
    openInBrowser(url);
    process.stderr.write('[trace] Opening in browser...\n');
  }
}
