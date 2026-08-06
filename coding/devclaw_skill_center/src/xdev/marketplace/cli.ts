/**
 * Commander.js subcommand group for `xdev marketplace ...`.
 *
 * Currently registers:
 *   - `xdev marketplace add-update <git-url> [--branch X] [--force]`
 *
 * Future commands (planned, not yet implemented):
 *   - `xdev marketplace list`
 *   - `xdev marketplace remove <name>`
 *   - `xdev marketplace info <name>`
 */

import type { Command } from 'commander';
import { addUpdateMarketplaces } from './add-update.js';

interface AddUpdateCliOptions {
  branch?: string;
  force?: boolean;
}

export function registerMarketplaceCommand(program: Command): void {
  const marketplace = program
    .command('marketplace')
    .description('管理第三方 marketplace（add-update / list / remove）');

  marketplace
    .command('add-update')
    .description('从 git 仓库添加或更新一个 marketplace（同名直接覆盖，覆盖前自动 backup）')
    .argument('<git-url>', '外部 marketplaces 仓库的 git 地址（git@... 或 https://...）')
    .option('--branch <name>', '指定分支（默认走 git 仓库的默认分支）')
    .option('--force', '跳过同名确认 prompt 直接覆盖')
    .action(async (gitUrl: string, options: AddUpdateCliOptions) => {
      try {
        await addUpdateMarketplaces(gitUrl, {
          branch: options.branch,
          force: options.force,
        });
      } catch (err) {
        console.error((err as Error).message);
        process.exit(1);
      }
    });
}
