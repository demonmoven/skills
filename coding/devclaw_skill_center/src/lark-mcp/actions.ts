import { spawnSync } from 'node:child_process';
import { copyFileSync, existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname } from 'node:path';
import {
  LARK_MCP_DIR,
  LARK_MCP_FILE,
  MCP_SERVER_BUNDLE,
  MCP_SERVER_KEY,
  Platform,
  detectPlatform,
  getClaudeDesktopConfigPath,
} from './paths.js';

// ─── Setup options ────────────────────────────────────────────────────

export interface SetupOptions {
  skipInstall?: boolean;
  skipAuth?: boolean;
}

// ─── Utility: pretty status icons ──────────────────────────────────────

const ok = '\x1b[32m✅\x1b[0m';
const warn = '\x1b[33m⚠️ \x1b[0m';
const fail = '\x1b[31m❌\x1b[0m';
const info = '\x1b[36mℹ️ \x1b[0m';
const point = '\x1b[35m👉\x1b[0m';

// ─── Utility: run a shell command and capture result ────────────────────

interface CommandResult {
  ok: boolean;
  stdout: string;
  stderr: string;
  code: number | null;
}

function run(cmd: string, args: string[], opts: { inheritStdio?: boolean } = {}): CommandResult {
  const result = spawnSync(cmd, args, {
    stdio: opts.inheritStdio ? 'inherit' : ['ignore', 'pipe', 'pipe'],
    encoding: 'utf-8',
  });
  return {
    ok: result.status === 0,
    stdout: result.stdout?.toString() ?? '',
    stderr: result.stderr?.toString() ?? '',
    code: result.status,
  };
}

// ─── lark-cli installation / config / auth checks ──────────────────────

interface LarkAuthStatus {
  appId?: string;
  identity?: string;
  userName?: string;
  userOpenId?: string;
  tokenStatus?: string;
  brand?: string;
}

/**
 * Cache the parsed `lark-cli auth status` JSON for the lifetime of one
 * action invocation, to avoid spawning lark-cli three times during `status`.
 */
let cachedAuthStatus: LarkAuthStatus | null | undefined;

function readAuthStatus(): LarkAuthStatus | null {
  if (cachedAuthStatus !== undefined) return cachedAuthStatus;
  const result = run('lark-cli', ['auth', 'status']);
  if (!result.ok || !result.stdout.trim()) {
    cachedAuthStatus = null;
    return null;
  }
  try {
    cachedAuthStatus = JSON.parse(result.stdout) as LarkAuthStatus;
  } catch {
    cachedAuthStatus = null;
  }
  return cachedAuthStatus;
}

export function resetAuthStatusCache(): void {
  cachedAuthStatus = undefined;
}

export function checkLarkCli(): { installed: boolean; version?: string; path?: string } {
  const versionResult = run('lark-cli', ['--version']);
  if (!versionResult.ok) {
    return { installed: false };
  }
  // `lark-cli --version` prints e.g. "lark-cli version 1.0.6"
  const match = versionResult.stdout.match(/version\s+(\S+)/);
  const which = run('which', ['lark-cli']);
  return {
    installed: true,
    version: match ? match[1] : versionResult.stdout.trim(),
    path: which.ok ? which.stdout.trim() : undefined,
  };
}

export function checkLarkCliConfig(): { configured: boolean; appId?: string } {
  const status = readAuthStatus();
  if (!status?.appId) return { configured: false };
  return { configured: true, appId: status.appId };
}

export function checkLarkCliAuth(): { authed: boolean; user?: string } {
  const status = readAuthStatus();
  if (!status) return { authed: false };
  // identity must be "user" (not "bot") and tokenStatus must be valid
  if (status.identity !== 'user' || status.tokenStatus !== 'valid') {
    return { authed: false };
  }
  return { authed: true, user: status.userName ?? status.userOpenId };
}

export function installLarkCli(): boolean {
  console.log(`${info} 正在安装 lark-cli (npm install -g @larksuite/cli)...`);
  const result = run('npm', ['install', '-g', '@larksuite/cli'], { inheritStdio: true });
  if (!result.ok) {
    console.log(`${fail} 安装失败 (exit ${result.code})`);
    return false;
  }
  console.log(`${ok} lark-cli 安装完成`);
  return true;
}

// ─── MCP Server file deployment ────────────────────────────────────────

export function deployMcpServer(): { ok: boolean; reason?: string } {
  if (!existsSync(MCP_SERVER_BUNDLE)) {
    return {
      ok: false,
      reason: `MCP Server bundle 不存在: ${MCP_SERVER_BUNDLE}\n  请先运行 npm run build 重新编译 xdev`,
    };
  }
  mkdirSync(LARK_MCP_DIR, { recursive: true });
  copyFileSync(MCP_SERVER_BUNDLE, LARK_MCP_FILE);
  return { ok: true };
}

export function isMcpServerDeployed(): boolean {
  return existsSync(LARK_MCP_FILE);
}

// ─── Claude Desktop config manipulation ────────────────────────────────

interface DesktopConfig {
  mcpServers?: Record<string, { command: string; args?: string[]; env?: Record<string, string> }>;
  [key: string]: unknown;
}

function readDesktopConfig(configPath: string): DesktopConfig {
  if (!existsSync(configPath)) return {};
  const raw = readFileSync(configPath, 'utf-8').trim();
  if (!raw) return {};
  try {
    return JSON.parse(raw) as DesktopConfig;
  } catch (err) {
    throw new Error(
      `解析 Claude Desktop 配置文件失败: ${configPath}\n  ` +
        (err instanceof Error ? err.message : String(err)),
    );
  }
}

function writeDesktopConfig(configPath: string, config: DesktopConfig): void {
  mkdirSync(dirname(configPath), { recursive: true });
  writeFileSync(configPath, JSON.stringify(config, null, 2) + '\n');
}

export function isRegisteredInDesktopConfig(plat: Platform): boolean {
  const configPath = getClaudeDesktopConfigPath(plat);
  if (!existsSync(configPath)) return false;
  try {
    const config = readDesktopConfig(configPath);
    const entry = config.mcpServers?.[MCP_SERVER_KEY];
    if (!entry) return false;
    // Verify it points to our deployed file
    return Array.isArray(entry.args) && entry.args.includes(LARK_MCP_FILE);
  } catch {
    return false;
  }
}

export function registerInDesktopConfig(plat: Platform): { ok: boolean; reason?: string; alreadyRegistered?: boolean } {
  const configPath = getClaudeDesktopConfigPath(plat);
  let config: DesktopConfig;
  try {
    config = readDesktopConfig(configPath);
  } catch (err) {
    return { ok: false, reason: err instanceof Error ? err.message : String(err) };
  }

  config.mcpServers ??= {};

  const existing = config.mcpServers[MCP_SERVER_KEY];
  const desired = {
    command: 'node',
    args: [LARK_MCP_FILE],
  };
  if (
    existing &&
    existing.command === desired.command &&
    Array.isArray(existing.args) &&
    existing.args.length === 1 &&
    existing.args[0] === LARK_MCP_FILE
  ) {
    return { ok: true, alreadyRegistered: true };
  }

  config.mcpServers[MCP_SERVER_KEY] = desired;
  writeDesktopConfig(configPath, config);
  return { ok: true };
}

export function unregisterFromDesktopConfig(plat: Platform): { ok: boolean; removed: boolean; reason?: string } {
  const configPath = getClaudeDesktopConfigPath(plat);
  if (!existsSync(configPath)) {
    return { ok: true, removed: false };
  }
  let config: DesktopConfig;
  try {
    config = readDesktopConfig(configPath);
  } catch (err) {
    return { ok: false, removed: false, reason: err instanceof Error ? err.message : String(err) };
  }
  if (!config.mcpServers || !(MCP_SERVER_KEY in config.mcpServers)) {
    return { ok: true, removed: false };
  }
  delete config.mcpServers[MCP_SERVER_KEY];
  writeDesktopConfig(configPath, config);
  return { ok: true, removed: true };
}

// ─── Desktop process check ───────────────────────────────────────────────

export function isClaudeDesktopRunning(): { running: boolean; pid?: number } {
  const result = run('pgrep', ['-x', 'Claude']);
  if (!result.ok) return { running: false };
  const pid = parseInt(result.stdout.trim().split('\n')[0] ?? '', 10);
  return { running: true, pid: Number.isFinite(pid) ? pid : undefined };
}

// ─── Action: setup ───────────────────────────────────────────────────────

export async function runSetup(plat: Platform, opts: SetupOptions = {}): Promise<void> {
  console.log(`\n${info} xdev lark-mcp ${plat} setup\n`);

  // Step 1/5: lark-cli installed
  console.log('Step 1/5  检查 lark-cli');
  let larkCli = checkLarkCli();
  if (larkCli.installed) {
    console.log(`  ${ok} 已安装 v${larkCli.version} (${larkCli.path ?? ''})`);
  } else if (opts.skipInstall) {
    console.log(`  ${fail} 未安装，且指定了 --skip-install。请手动安装：npm install -g @larksuite/cli`);
    throw new Error('lark-cli 未安装');
  } else {
    if (!installLarkCli()) {
      throw new Error('lark-cli 安装失败');
    }
    larkCli = checkLarkCli();
    if (!larkCli.installed) throw new Error('lark-cli 安装后仍不可用');
    console.log(`  ${ok} 安装完成 v${larkCli.version}`);
  }

  // Step 2/5: lark-cli config (app)
  console.log('\nStep 2/5  飞书应用配置');
  const cfg = checkLarkCliConfig();
  if (cfg.configured) {
    console.log(`  ${ok} 已配置${cfg.appId ? ` (app_id: ${cfg.appId})` : ''}`);
  } else {
    console.log(`  ${warn} 未配置应用`);
    console.log(`  ${point} 请运行：lark-cli config init --new`);
    console.log(`     该命令会输出授权链接，在飞书中打开链接完成应用创建`);
    console.log(`     完成后重新运行 xdev lark-mcp ${plat} setup`);
    throw new Error('需要先配置飞书应用');
  }

  // Step 3/5: user auth (optional)
  console.log('\nStep 3/5  用户授权');
  const auth = checkLarkCliAuth();
  if (auth.authed) {
    console.log(`  ${ok} 已授权 (${auth.user ?? 'user'})`);
  } else if (opts.skipAuth) {
    console.log(`  ${warn} 未授权，已跳过 (--skip-auth)`);
    console.log(`  ${info} 不授权也能用：bot 身份的能力（发消息、创建文档等）仍可用`);
    console.log(`  ${info} 之后可运行：lark-cli auth login 完成授权`);
  } else {
    console.log(`  ${warn} 未授权`);
    console.log(`  ${point} 请运行：lark-cli auth login`);
    console.log(`     该命令会输出 OAuth 链接，在飞书中确认即可以个人身份操作飞书`);
    console.log(`     完成后重新运行 xdev lark-mcp ${plat} setup`);
    console.log(`  ${info} 如不需要个人身份，可加 --skip-auth 跳过此步`);
    throw new Error('需要先完成用户授权（或加 --skip-auth）');
  }

  // Step 4/5: deploy MCP server file
  console.log('\nStep 4/5  部署 MCP Server');
  const deploy = deployMcpServer();
  if (!deploy.ok) {
    console.log(`  ${fail} ${deploy.reason}`);
    throw new Error('MCP Server 部署失败');
  }
  console.log(`  ${ok} ${LARK_MCP_FILE}`);

  // Step 5/5: register in Claude Desktop config
  console.log('\nStep 5/5  注册到 Claude Desktop 配置');
  const reg = registerInDesktopConfig(plat);
  if (!reg.ok) {
    console.log(`  ${fail} ${reg.reason}`);
    throw new Error('注册到 Claude Desktop 失败');
  }
  const configPath = getClaudeDesktopConfigPath(plat);
  if (reg.alreadyRegistered) {
    console.log(`  ${ok} 已注册（${configPath}）`);
  } else {
    console.log(`  ${ok} 写入 ${configPath}`);
  }

  // Done message
  console.log('');
  console.log(`${ok} lark-cli MCP Server 配置完成！\n`);
  console.log(`${info} MCP Server 进程由 Claude Desktop 自动 spawn，无需手动启动`);
  console.log(`${point} 请完全退出 Claude Desktop App（Cmd+Q），然后重新打开`);
  console.log(`${point} 重启后，在 Chat 或 Cowork 中试试：「帮我看一下今天的飞书日程」\n`);
}

// ─── Action: start ────────────────────────────────────────────────────────

export async function runStart(plat: Platform): Promise<void> {
  console.log(`\n${info} xdev lark-mcp ${plat} start\n`);

  if (!isMcpServerDeployed()) {
    console.log(`${fail} MCP Server 未部署：${LARK_MCP_FILE}`);
    console.log(`${point} 请先运行：xdev lark-mcp ${plat} setup`);
    throw new Error('MCP Server 未部署');
  }

  const reg = registerInDesktopConfig(plat);
  if (!reg.ok) {
    console.log(`${fail} ${reg.reason}`);
    throw new Error('注册到 Claude Desktop 失败');
  }

  const configPath = getClaudeDesktopConfigPath(plat);
  if (reg.alreadyRegistered) {
    console.log(`${ok} 已注册（无变更）：${configPath}`);
  } else {
    console.log(`${ok} 已注册到：${configPath}`);
  }
  console.log('');
  console.log(`${info} MCP Server 进程由 Claude Desktop 自动 spawn，无需手动启动`);
  console.log(`${point} 请重启 Claude Desktop App 后生效\n`);
}

// ─── Action: stop ────────────────────────────────────────────────────────

export async function runStop(plat: Platform): Promise<void> {
  console.log(`\n${info} xdev lark-mcp ${plat} stop\n`);

  const result = unregisterFromDesktopConfig(plat);
  if (!result.ok) {
    console.log(`${fail} ${result.reason}`);
    throw new Error('从 Claude Desktop 配置移除失败');
  }
  if (!result.removed) {
    console.log(`${info} 当前未注册到 Claude Desktop 配置，无需移除`);
    return;
  }
  const configPath = getClaudeDesktopConfigPath(plat);
  console.log(`${ok} 已从 Claude Desktop 配置移除：${configPath}`);
  console.log('');
  console.log(`${info} 正在运行的 MCP Server 子进程会在下次 Desktop 重启时不再 spawn`);
  console.log(`${info} MCP Server 文件本身保留在 ${LARK_MCP_FILE}`);
  console.log(`${point} 请重启 Claude Desktop App 让变更生效\n`);
}

// ─── Action: status ──────────────────────────────────────────────────────

export async function runStatus(plat: Platform): Promise<void> {
  console.log('\n  lark-cli MCP Server Status');
  console.log('  ──────────────────────────────────\n');

  // lark-cli
  const cli = checkLarkCli();
  if (cli.installed) {
    console.log(`  lark-cli        ${ok} v${cli.version}${cli.path ? ` (${cli.path})` : ''}`);
  } else {
    console.log(`  lark-cli        ${fail} 未安装（运行 npm install -g @larksuite/cli）`);
  }

  // 应用配置
  const cfg = cli.installed ? checkLarkCliConfig() : { configured: false };
  if (cfg.configured) {
    console.log(`  应用配置        ${ok} ${('appId' in cfg && cfg.appId) || '已配置'}`);
  } else if (cli.installed) {
    console.log(`  应用配置        ${warn} 未配置（运行 lark-cli config init --new）`);
  } else {
    console.log(`  应用配置        ${fail} -`);
  }

  // 用户授权
  const auth = cli.installed ? checkLarkCliAuth() : { authed: false };
  if (auth.authed) {
    console.log(`  用户授权        ${ok} ${('user' in auth && auth.user) || '已授权'}`);
  } else if (cli.installed) {
    console.log(`  用户授权        ${warn} 未授权（运行 lark-cli auth login，可选）`);
  } else {
    console.log(`  用户授权        ${fail} -`);
  }

  // MCP Server file
  if (isMcpServerDeployed()) {
    console.log(`  MCP Server      ${ok} ${LARK_MCP_FILE}`);
  } else {
    console.log(`  MCP Server      ${fail} 未部署（运行 xdev lark-mcp ${plat} setup）`);
  }

  // Desktop config
  const configPath = getClaudeDesktopConfigPath(plat);
  if (isRegisteredInDesktopConfig(plat)) {
    console.log(`  Desktop 配置    ${ok} mcpServers.${MCP_SERVER_KEY} 已注册`);
    console.log(`                     ${configPath}`);
  } else if (existsSync(configPath)) {
    console.log(`  Desktop 配置    ${fail} mcpServers.${MCP_SERVER_KEY} 未注册（运行 xdev lark-mcp ${plat} start）`);
  } else {
    console.log(`  Desktop 配置    ${fail} 配置文件不存在：${configPath}`);
  }

  // Desktop 进程
  const desktop = isClaudeDesktopRunning();
  if (desktop.running) {
    console.log(`  Desktop 进程    ${ok} 运行中${desktop.pid ? ` (pid ${desktop.pid})` : ''}`);
  } else {
    console.log(`  Desktop 进程    ${warn} 未运行`);
  }

  console.log('');
  const allOk =
    cli.installed && cfg.configured && isMcpServerDeployed() && isRegisteredInDesktopConfig(plat);
  if (allOk) {
    console.log(`  ${ok} 全部就绪。如有问题请重启 Claude Desktop App。\n`);
  } else {
    console.log(`  ${warn} 部分组件未就绪。运行 xdev lark-mcp ${plat} setup 一键修复。\n`);
  }
}

// ─── Helper used by uninstall path ────────────────────────────────────────

/**
 * Best-effort cleanup of the Claude Desktop config entry. Used by
 * `xdev uninstall` to avoid leaving a stale mcpServers.lark-cli pointing at
 * a deleted ~/.xdev/lark-mcp-server/index.mjs after the user uninstalls xdev.
 */
export function cleanupDesktopConfigOnUninstall(): void {
  try {
    const plat = detectPlatform();
    const result = unregisterFromDesktopConfig(plat);
    if (result.removed) {
      console.log(`  - removed mcpServers.${MCP_SERVER_KEY} from ${getClaudeDesktopConfigPath(plat)}`);
    }
  } catch {
    // Best-effort: don't fail uninstall if Desktop config can't be cleaned
  }
}
