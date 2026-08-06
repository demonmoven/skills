#!/usr/bin/env node

import { Command } from 'commander';
import { runLaunch, type LaunchOptions } from './xdev/launch.js';
import { registerCleanCommand } from './xdev/clean.js';
import { registerUninstallCommand } from './xdev/uninstall.js';
import { registerMarketplaceCommand } from './xdev/marketplace/cli.js';
import { registerTraceCommand } from './trace/cli.js';
import { registerLarkMcpCommand } from './lark-mcp/index.js';
import { registerTmuxCommand } from './xdev/tmux/index.js';
import { registerWorkspaceCommand } from './xdev/workspace/index.js';
import { registerClaudeSettingsCommand } from './xdev/claude-settings/index.js';
import { TraceError } from './trace/utils/errors.js';
import { runSelfUpdateCheck } from './xdev/version-check.js';
import { isDebugMode, handleDebugMode } from './xdev/debug.js';

// Injected by esbuild at build time from package.json's `version` field.
// See build.js define block. Avoids drift between .version() string and
// the actual package version on every release.
declare const XDEV_VERSION: string;

const program = new Command();

// Strict positional option parsing: parent options stay on the parent and
// subcommand options stay on the subcommand. Without this, `xdev --cc` (launch
// Claude Code) shares the `--cc` name with `xdev trace analyze --cc` and
// commander silently routes the latter back to the parent, dropping it from
// the subcommand's opts. See commander.js docs on `enablePositionalOptions`.
program.enablePositionalOptions();

program
  .name('xdev')
  .description('DevClaw 统一 CLI：启动 Trae CN / Trae / Coco / Claude Code / Codex（含 trace 会话上报子命令）')
  .version(XDEV_VERSION)
  .option('--trae-cn', '启动 Trae CN (IDE)')
  .option('--trae', '启动 Trae (IDE)')
  .option('--coco', '启动 Coco / TRAE CLI')
  .option('--cc', '启动 Claude Code')
  .option('--cdx', '启动 Codex CLI')
  .option('--no-external', '跳过第三方 marketplace 多选（CI / 脚本场景）')
  .option('--all-skills', '跳过 skill 选择，安装全部（Trae / Trae CN）')
  .option('--no-bytedcli-deps', '跳过 bytedcli 依赖检查与自动安装（CI / 离线场景）')
  .option('--no-trace-auth', '跳过 Byte SSO trace 认证检查（CI / 离线场景）')
  .argument('[extra...]', '透传给 agent 的额外参数（用 -- 分隔）')
  .allowExcessArguments(true)
  .action(async (extra: string[], options: LaunchOptions) => {
    await runLaunch(options, extra);
  });

registerCleanCommand(program);
registerUninstallCommand(program);
registerMarketplaceCommand(program);
registerTraceCommand(program);
registerLarkMcpCommand(program);
registerTmuxCommand(program);
registerWorkspaceCommand(program);
registerClaudeSettingsCommand(program);

process.on('unhandledRejection', (err) => {
  if (err instanceof TraceError) {
    console.error(`Error [${err.code}]: ${err.message}`);
    process.exit(1);
  }
  console.error('Unexpected error:', err instanceof Error ? err.message : String(err));
  process.exit(1);
});

// --debug: build from local XDEV_DEV_REPO and re-exec — must intercept
// before version check and commander parse to avoid auto-upgrade interference.
// When active, the parent process does nothing else — it just waits for the
// child to exit.
if (!isDebugMode()) {
  runSelfUpdateCheck();

  program.parseAsync(process.argv).catch((err) => {
    if (err instanceof TraceError) {
      console.error(`Error [${err.code}]: ${err.message}`);
      process.exit(1);
    }
    console.error('Unexpected error:', err instanceof Error ? err.message : String(err));
    process.exit(1);
  });
} else {
  handleDebugMode();
}
