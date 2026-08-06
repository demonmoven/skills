import { spawnSync } from 'node:child_process';
import { existsSync, readFileSync, writeFileSync } from 'node:fs';
import { homedir, platform as osPlatform } from 'node:os';
import { join } from 'node:path';
import {
  END_MARKER,
  START_MARKER,
  XDEV_TMUX_CONF_BLOCK,
} from './conf-template.js';

const ok = '\x1b[32m✅\x1b[0m';
const fail = '\x1b[31m❌\x1b[0m';
const point = '\x1b[35m👉\x1b[0m';

function commandExists(cmd: string): boolean {
  return spawnSync('which', [cmd], { stdio: 'ignore' }).status === 0;
}

function captureVersion(cmd: string, arg = '-V'): string {
  const r = spawnSync(cmd, [arg], { encoding: 'utf-8' });
  return (r.stdout ?? '').trim() || (r.stderr ?? '').trim();
}

export async function runTmuxSetup(): Promise<void> {
  console.log('[xdev] tmux mac setup — 4 steps');

  if (osPlatform() !== 'darwin') {
    console.error(`${fail} 仅支持 macOS（当前 platform: ${osPlatform()}）`);
    process.exit(1);
  }

  // Step 1/4: brew
  console.log('  [1/4] 检查 Homebrew...');
  if (!commandExists('brew')) {
    console.error(`${fail} 未检测到 brew`);
    console.error(`       ${point} 请先安装 Homebrew：https://brew.sh`);
    process.exit(1);
  }
  console.log(`        ${ok} brew 已安装`);

  // Step 2/4: tmux
  console.log('  [2/4] 检查 tmux...');
  if (commandExists('tmux')) {
    console.log(`        ${ok} ${captureVersion('tmux')}`);
  } else {
    console.log('        → brew install tmux');
    const r = spawnSync('brew', ['install', 'tmux'], { stdio: 'inherit' });
    if (r.status !== 0) {
      console.error(`${fail} brew install tmux 失败（exit ${r.status}）`);
      process.exit(1);
    }
    console.log(`        ${ok} ${captureVersion('tmux')}`);
  }

  // Step 3/4: ~/.tmux.conf
  console.log('  [3/4] 写入 ~/.tmux.conf 的 XDEV-TMUX 段（幂等）...');
  const confPath = join(homedir(), '.tmux.conf');
  const replaced = mergeTmuxConf(confPath);
  console.log(`        ${ok} ${confPath}（${replaced ? '已替换' : '已新增'} XDEV-TMUX 段）`);

  // Step 4/4: 提示热加载
  console.log('  [4/4] 提示');
  console.log(`        ${point} 已运行的 tmux session 请在其内执行 \`tmux source ~/.tmux.conf\` 让新配置生效`);
  console.log(`        ${point} 新开的 tmux session 自动生效`);

  console.log('[xdev] tmux mac setup done.');
}

function escapeRegex(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/**
 * Idempotently merge XDEV_TMUX_CONF_BLOCK into ~/.tmux.conf, fenced by
 * START_MARKER / END_MARKER. Returns true if an existing block was replaced,
 * false if the block was newly appended (or the file was newly created).
 *
 * Anything outside the marker pair is preserved verbatim.
 */
export function mergeTmuxConf(confPath: string): boolean {
  const existing = existsSync(confPath) ? readFileSync(confPath, 'utf-8') : '';

  const re = new RegExp(
    `\\n?${escapeRegex(START_MARKER)}[\\s\\S]*?${escapeRegex(END_MARKER)}\\n?`,
    'g',
  );
  const hadBlock = re.test(existing);
  const stripped = existing.replace(re, '').trimEnd();
  const merged = (stripped ? stripped + '\n\n' : '') + XDEV_TMUX_CONF_BLOCK;

  writeFileSync(confPath, merged);
  return hadBlock;
}
