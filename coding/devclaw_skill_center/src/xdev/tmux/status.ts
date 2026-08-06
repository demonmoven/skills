import { spawnSync } from 'node:child_process';
import { existsSync, readFileSync } from 'node:fs';
import { homedir, platform as osPlatform } from 'node:os';
import { join } from 'node:path';
import { END_MARKER, START_MARKER } from './conf-template.js';

const ok = '\x1b[32m✅\x1b[0m';
const warn = '\x1b[33m⚠️ \x1b[0m';
const fail = '\x1b[31m❌\x1b[0m';

export async function runTmuxStatus(): Promise<void> {
  console.log('[xdev] tmux mac status');

  // Platform
  const plat = osPlatform();
  if (plat !== 'darwin') {
    console.log(`  ${warn} 当前 platform: ${plat}（本命令仅适用于 macOS）`);
  } else {
    console.log(`  ${ok} platform: darwin`);
  }

  // brew
  const brew = spawnSync('which', ['brew'], { stdio: 'ignore' }).status === 0;
  console.log(`  ${brew ? ok : fail} brew ${brew ? '已安装' : '未安装'}`);

  // tmux
  const tmux = spawnSync('tmux', ['-V'], { encoding: 'utf-8' });
  if (tmux.status === 0) {
    console.log(`  ${ok} ${tmux.stdout.trim()}`);
  } else {
    console.log(`  ${fail} tmux 未安装`);
  }

  // ~/.tmux.conf XDEV-TMUX 段
  const confPath = join(homedir(), '.tmux.conf');
  if (!existsSync(confPath)) {
    console.log(`  ${fail} ${confPath} 不存在`);
  } else {
    const content = readFileSync(confPath, 'utf-8');
    const hasStart = content.includes(START_MARKER);
    const hasEnd = content.includes(END_MARKER);
    if (hasStart && hasEnd) {
      console.log(`  ${ok} ${confPath}（含 XDEV-TMUX 段）`);
    } else if (hasStart || hasEnd) {
      console.log(`  ${warn} ${confPath} 含残缺的 XDEV-TMUX marker，建议重跑 setup`);
    } else {
      console.log(`  ${warn} ${confPath} 不含 XDEV-TMUX 段，可运行 \`xdev tmux mac setup\` 写入`);
    }
  }
}
