import { exec } from 'node:child_process';
import { loadConfig } from '../config/loader.js';
import { createServer } from '../server/index.js';
import { logger } from '../utils/logger.js';

export interface ServeOptions {
  port: string;
  open: boolean;
}

/**
 * Start the Web Dashboard server.
 */
export async function runServe(options: ServeOptions): Promise<void> {
  const port = parseInt(options.port, 10);
  if (isNaN(port) || port < 1 || port > 65535) {
    throw new Error(`Invalid port: ${options.port}`);
  }

  const config = await loadConfig(process.cwd());

  const { start } = createServer({ config, port });
  await start();

  // Auto-open browser unless --no-open
  if (options.open) {
    const url = `http://localhost:${port}`;
    openBrowser(url);
  }
}

/**
 * Open a URL in the default browser using platform-specific commands.
 */
function openBrowser(url: string): void {
  const platform = process.platform;
  let command: string;

  switch (platform) {
    case 'darwin':
      command = `open "${url}"`;
      break;
    case 'win32':
      command = `start "" "${url}"`;
      break;
    default:
      // Linux and others
      command = `xdg-open "${url}"`;
      break;
  }

  exec(command, (err) => {
    if (err) {
      logger.debug(`Failed to open browser: ${err.message}`);
      logger.info(`Open ${url} in your browser manually.`);
    }
  });
}
