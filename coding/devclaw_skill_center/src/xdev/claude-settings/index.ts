import { Command } from 'commander';
import { runClaudeSettingsSetup } from './setup.js';

interface SetupCliOptions {
  preferTemplate?: boolean;
  dryRun?: boolean;
}

export function registerClaudeSettingsCommand(parent: Command): void {
  const cs = parent
    .command('claude-settings')
    .description('~/.claude/settings.json 模板分发');

  const mac = cs.command('mac').description('macOS 平台');

  mac
    .command('setup')
    .description('增量合并 xdev 内置模板到 ~/.claude/settings.json（幂等）')
    .option('--prefer-template', '冲突时偏好模板值（默认偏好用户已有值）')
    .option('--dry-run', '只预览 diff，不真写')
    .action(async (opts: SetupCliOptions) => {
      await runClaudeSettingsSetup({
        preferTemplate: opts.preferTemplate,
        dryRun: opts.dryRun,
      });
    });
}
