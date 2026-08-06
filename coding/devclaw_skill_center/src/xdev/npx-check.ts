import { execFileSync } from 'node:child_process';
import { platform } from 'node:os';

export function ensureNpxOrWarn(): boolean {
  try {
    execFileSync('npx', ['--version'], { stdio: ['ignore', 'pipe', 'pipe'] });
    return true;
  } catch (err) {
    const e = err as NodeJS.ErrnoException;
    if (e.code !== 'ENOENT') {
      console.error(`[xdev] Warn: 'npx --version' failed: ${e.message}`);
      return false;
    }
    printInstallGuide();
    return false;
  }
}

function printInstallGuide(): void {
  const isMac = platform() === 'darwin';
  const isLinux = platform() === 'linux';
  console.error('');
  console.error('[xdev] ⚠ 未检测到 npx 命令。');
  console.error('[xdev]   xdev 注入的 hook（trace 上报、AI 代码统计）需要 npx 运行，');
  console.error('[xdev]   未安装时这些 hook 会在会话结束时报错但不影响主流程。');
  console.error('[xdev]');
  console.error('[xdev] 一键安装建议：');
  if (isMac) {
    console.error('[xdev]   brew install node          # macOS 推荐');
  } else if (isLinux) {
    console.error('[xdev]   curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash');
    console.error('[xdev]   nvm install --lts');
  } else {
    console.error('[xdev]   访问 https://nodejs.org 下载并安装 Node.js (含 npx)');
  }
  console.error('[xdev]');
}
