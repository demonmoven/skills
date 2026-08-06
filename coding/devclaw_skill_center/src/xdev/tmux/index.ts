import { Command } from 'commander';
import { runTmuxSetup } from './setup.js';
import { runTmuxStatus } from './status.js';

export function registerTmuxCommand(parent: Command): void {
  const tmux = parent
    .command('tmux')
    .description('tmux 安装与配置（macOS）');

  const mac = tmux.command('mac').description('macOS 平台');

  mac
    .command('setup')
    .description('一键安装 tmux + 写入 ~/.tmux.conf XDEV-TMUX 段（幂等）')
    .action(async () => {
      await runTmuxSetup();
    });

  mac
    .command('status')
    .description('查看 tmux 安装与 ~/.tmux.conf 段状态')
    .action(async () => {
      await runTmuxStatus();
    });
}
