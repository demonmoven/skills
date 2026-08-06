import { exec, spawn } from 'node:child_process';
import { platform } from 'node:os';
import { logger } from './logger.js';

export function openInBrowser(url: string): void {
  const cmd =
    platform() === 'darwin'
      ? `open "${url.replace(/"/g, '\\"')}"`
      : platform() === 'win32'
        ? `start "" "${url}"`
        : `xdg-open "${url.replace(/"/g, '\\"')}"`;
  exec(cmd, (err) => {
    if (err) {
      logger.debug(`Failed to open browser: ${err.message}`);
      logger.info(`Open manually: ${url}`);
    }
  });
}

export function copyToClipboard(text: string): Promise<void> {
  const tool =
    platform() === 'darwin' ? 'pbcopy' : platform() === 'win32' ? 'clip' : 'xclip';
  const args = platform() === 'linux' ? ['-selection', 'clipboard'] : [];
  return new Promise((resolve) => {
    try {
      const child = spawn(tool, args, { stdio: ['pipe', 'ignore', 'ignore'] });
      child.on('error', (err) => {
        logger.debug(`clipboard tool '${tool}' not available: ${err.message}`);
        resolve();
      });
      child.on('exit', () => resolve());
      child.stdin.write(text);
      child.stdin.end();
    } catch (err) {
      logger.debug(`copyToClipboard failed: ${(err as Error).message}`);
      resolve();
    }
  });
}
