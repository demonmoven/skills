import { spawn } from 'node:child_process';
import { platform } from 'node:os';

/**
 * Open a local file in the user's default browser.
 *
 * Cross-platform: darwin → `open`, linux → `xdg-open`, win32 → `cmd /c start`.
 */
export async function openInBrowser(filePath: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const p = platform();
    let cmd: string;
    let args: string[];

    if (p === 'darwin') {
      cmd = 'open';
      args = [filePath];
    } else if (p === 'win32') {
      cmd = 'cmd';
      args = ['/c', 'start', '""', filePath];
    } else {
      cmd = 'xdg-open';
      args = [filePath];
    }

    const child = spawn(cmd, args, { detached: true, stdio: 'ignore' });
    child.on('error', reject);
    child.unref();
    // We don't wait for the browser to actually open — spawn is enough.
    resolve();
  });
}
