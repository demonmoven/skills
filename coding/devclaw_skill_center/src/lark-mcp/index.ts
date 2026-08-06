import { Command } from 'commander';
import { Platform, detectPlatform } from './paths.js';
import { runSetup, runStart, runStatus, runStop, SetupOptions } from './actions.js';

// ─── Help text fragments ────────────────────────────────────────────────

const PLATFORM_HELP_AFTER = `
完整流程：
  1. xdev lark-mcp mac setup     # 一键安装 lark-cli + 应用配置 + 用户授权 + MCP Server 部署 + Claude Desktop 注册
  2. 完全退出 Claude Desktop App（Cmd+Q），然后重新打开
  3. 在 Chat / Cowork 中输入 "帮我看一下今天的飞书日程"，验证 Claude 调用 lark_cli tool

命令说明：
  setup    一键完成所有 5 个步骤（幂等，已完成的步骤会自动跳过）
  start    把 mcpServers.lark-cli 写入 Claude Desktop 配置（已 deploy 时使用）
  stop     从 Claude Desktop 配置中移除 mcpServers.lark-cli
  status   查看 lark-cli、授权、MCP Server、Desktop 配置、Desktop 进程的全部状态

重要：MCP Server 进程由 Claude Desktop 自动 spawn 和 kill，**无需手动启停**
  - start/stop 只是改 ~/Library/Application Support/Claude/claude_desktop_config.json
  - 改完必须重启 Claude Desktop App（Cmd+Q + 重新打开）才生效
  - Desktop 启动时会读 config 自动 spawn MCP Server 子进程，退出时一起 kill

调试：
  xdev lark-mcp mac status                                                       # 查看所有组件状态
  cat "$HOME/Library/Application Support/Claude/claude_desktop_config.json"      # 查看 Desktop 配置
  ls ~/.xdev/lark-mcp-server/                                                    # 查看部署的 MCP Server 文件
  node ~/.xdev/lark-mcp-server/index.mjs                                         # 直接跑 MCP Server (Ctrl+D 退出)
`;

const SETUP_HELP_AFTER = `
五个步骤（每步都幂等，已完成的步骤会自动跳过）：

  Step 1/5  检查 lark-cli (没装就 npm install -g @larksuite/cli)
  Step 2/5  飞书应用配置 (没配就引导你跑 lark-cli config init --new)
  Step 3/5  用户授权 (没授就引导你跑 lark-cli auth login，可加 --skip-auth 跳过)
  Step 4/5  部署 MCP Server 到 ~/.xdev/lark-mcp-server/index.mjs
  Step 5/5  注册到 Claude Desktop 配置 (mcpServers.lark-cli)

setup 完成后：
  👉 完全退出 Claude Desktop App（Cmd+Q），然后重新打开
  👉 在 Chat / Cowork 中试一句：「帮我看一下今天的飞书日程」

如果 Step 2 / Step 3 中断（应用未配 / 未授权）：
  按提示在终端运行 lark-cli config init --new 或 lark-cli auth login
  完成后重新跑 xdev lark-mcp mac setup（已完成的步骤会自动跳过）

跳过用户授权（--skip-auth）：
  bot 身份的能力（发消息、创建文档等）仍可用
  user 身份的能力（看个人日历、私信、收件箱等）不可用
  之后可单独跑 lark-cli auth login 补授权
`;

const START_HELP_AFTER = `
注意：start 不是真的启动进程！
  MCP Server 进程由 Claude Desktop 自动 spawn 和 kill。
  start 实际只是把 mcpServers.lark-cli 写入 claude_desktop_config.json。
  改完必须完全退出并重启 Claude Desktop App（Cmd+Q + 重新打开）才生效。

前提：
  MCP Server 文件已经部署到 ~/.xdev/lark-mcp-server/index.mjs
  如果没部署，请先运行 xdev lark-mcp mac setup
`;

const STOP_HELP_AFTER = `
注意：stop 不是真的关闭进程！
  MCP Server 进程由 Claude Desktop 自动 spawn 和 kill。
  stop 实际只是从 claude_desktop_config.json 中移除 mcpServers.lark-cli。
  正在运行的子进程会在下次 Desktop 重启时不再被 spawn。
  改完必须完全退出并重启 Claude Desktop App（Cmd+Q + 重新打开）才生效。

stop 不会删除 ~/.xdev/lark-mcp-server/index.mjs 文件本身，
方便后续 xdev lark-mcp mac start 一键恢复。
彻底删除请运行 xdev uninstall 或手动 rm -rf ~/.xdev/lark-mcp-server。
`;

const STATUS_HELP_AFTER = `
status 检查的 6 个组件：

  lark-cli        是否已安装、版本号、二进制路径
  应用配置        飞书应用 app_id 是否已配置 (lark-cli config init)
  用户授权        是否完成 user 身份授权 (lark-cli auth login)
  MCP Server      ~/.xdev/lark-mcp-server/index.mjs 是否存在
  Desktop 配置    claude_desktop_config.json 是否注册了 mcpServers.lark-cli
  Desktop 进程    Claude Desktop App 是否正在运行 (pgrep)

任何一项 ❌ / ⚠️ ：跑 xdev lark-mcp mac setup 一键修复
全部 ✅ ：如果 Chat/Cowork 仍调不到 lark_cli tool，请完全退出并重启 Desktop
`;

/**
 * Mount setup / start / stop / status as leaf commands under a platform group.
 *
 * Each Platform (mac / linux / windows) gets its own subcommand group, so
 * users invoke as:  xdev lark-mcp mac setup
 *
 * Why expose the platform explicitly instead of always auto-detecting?
 * - Forward-compat: future linux/windows support reuses the same shape
 * - Self-documenting: `xdev lark-mcp --help` makes the platform matrix visible
 * - Auto-detection still happens inside each handler when --auto is desired
 */
function mountPlatformActions(target: Command, plat: Platform): void {
  // Attach the platform-level help-after to the parent group itself
  target.addHelpText('after', PLATFORM_HELP_AFTER);

  target
    .command('setup')
    .description(
      '一键安装：lark-cli + 应用配置 + 用户授权 + MCP Server 部署 + Claude Desktop 注册',
    )
    .option('--skip-install', '跳过 lark-cli 安装（已安装时使用）')
    .option('--skip-auth', '跳过用户授权步骤（之后可手动 lark-cli auth login）')
    .addHelpText('after', SETUP_HELP_AFTER)
    .action(async (opts: SetupOptions) => {
      await runSetup(plat, opts);
    });

  target
    .command('start')
    .description('注册 MCP Server 到 Claude Desktop 配置（已部署时使用）')
    .addHelpText('after', START_HELP_AFTER)
    .action(async () => {
      await runStart(plat);
    });

  target
    .command('stop')
    .description('从 Claude Desktop 配置中移除 MCP Server')
    .addHelpText('after', STOP_HELP_AFTER)
    .action(async () => {
      await runStop(plat);
    });

  target
    .command('status')
    .description('查看 lark-cli、授权、MCP Server、Desktop 配置等所有组件状态')
    .addHelpText('after', STATUS_HELP_AFTER)
    .action(async () => {
      await runStatus(plat);
    });
}

/**
 * Mount lark-mcp as a top-level subcommand on the xdev CLI.
 *
 * Use case:
 *   xdev lark-mcp mac setup
 *   xdev lark-mcp mac status
 *   xdev lark-mcp mac stop
 */
export function registerLarkMcpCommand(program: Command): void {
  const larkMcp = program
    .command('lark-mcp')
    .description(
      '管理 lark-cli MCP Server，让 Claude Desktop 的 Chat 和 Cowork 模式可以操作飞书',
    )
    .addHelpText(
      'after',
      `
前置条件：
  - Node.js >= 18
  - npm（用于安装 lark-cli @larksuite/cli）
  - Claude Desktop App 已安装

平台：
  mac                macOS（自动检测时默认）
  linux              Linux（占位，待实现）
  windows            Windows（占位，待实现）

快速开始：
  xdev lark-mcp mac setup        # 一键完成所有步骤
  xdev lark-mcp mac status       # 查看当前状态

工作原理：
  MCP Server 进程由 Claude Desktop 自动 spawn 和 kill，无需手动启停。
  setup/start/stop 实际是修改 ~/Library/Application Support/Claude/
  claude_desktop_config.json 的 mcpServers 配置，改完后必须重启 Desktop。

参考：
  飞书 CLI 介绍 https://bytedance.larkoffice.com/docx/WnHkdJQM6oGpQFxm9i7ckVdenSh
`,
    );

  // mac platform group
  const mac = larkMcp.command('mac').description('macOS 平台');
  mountPlatformActions(mac, 'mac');

  // linux / windows are placeholders for forward compatibility. The actions
  // already work on those platforms (paths.ts handles the config locations),
  // but we have not tested them so we surface them but mark as experimental.
  const linux = larkMcp
    .command('linux')
    .description('Linux 平台（实验性，未测试）');
  mountPlatformActions(linux, 'linux');

  const windows = larkMcp
    .command('windows')
    .description('Windows 平台（实验性，未测试）');
  mountPlatformActions(windows, 'windows');

  // Convenience: `xdev lark-mcp setup` (no platform) auto-detects.
  larkMcp
    .command('setup')
    .description('（自动检测平台）一键安装')
    .option('--skip-install', '跳过 lark-cli 安装')
    .option('--skip-auth', '跳过用户授权步骤')
    .action(async (opts: SetupOptions) => {
      await runSetup(detectPlatform(), opts);
    });

  larkMcp
    .command('status')
    .description('（自动检测平台）查看状态')
    .action(async () => {
      await runStatus(detectPlatform());
    });
}
