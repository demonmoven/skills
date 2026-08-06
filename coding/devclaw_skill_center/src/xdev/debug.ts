import { spawnSync, spawn } from 'node:child_process';
import { join } from 'node:path';

export function isDebugMode(): boolean {
  return process.argv.includes('--debug');
}

export function handleDebugMode(): void {
  const repoPath = process.env['XDEV_DEV_REPO'];
  if (!repoPath) {
    console.error('[xdev] --debug requires XDEV_DEV_REPO environment variable');
    console.error('[xdev] Add to your shell profile (~/.zshrc or ~/.bashrc):');
    console.error(
      '[xdev]   export XDEV_DEV_REPO=~/workspace/src/codebase/stone/devclaw_skills_center',
    );
    process.exit(1);
  }

  const originalCwd = process.cwd();
  const args = process.argv.slice(2).filter((a) => a !== '--debug');

  console.log(`[xdev] debug mode — building from ${repoPath}...`);

  const buildResult = spawnSync('npm', ['run', 'build'], {
    cwd: repoPath,
    stdio: 'inherit',
  });
  if (buildResult.status !== 0) {
    console.error('[xdev] build failed');
    process.exit(buildResult.status ?? 1);
  }

  console.log(`[xdev] debug mode — launching with local build...`);

  const child = spawn('node', [join(repoPath, 'dist', 'index.js'), ...args], {
    cwd: originalCwd,
    stdio: 'inherit',
    env: { ...process.env, XDEV_NO_UPDATE: '1' },
  });
  child.on('exit', (code) => process.exit(code ?? 0));
  child.on('error', (err: NodeJS.ErrnoException) => {
    console.error(`[xdev] failed to spawn local build: ${err.message}`);
    process.exit(1);
  });
}
