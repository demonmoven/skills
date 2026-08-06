import { homedir, platform as osPlatform } from 'node:os';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { XDEV_HOME } from '../xdev/paths.js';

/**
 * Resolve the path to the bundled MCP server file inside the npm package.
 *
 * At runtime, the xdev CLI runs from `<package-root>/dist/index.js`. esbuild
 * preserves `import.meta.url` in the ESM bundle, so this resolves to the
 * actual on-disk location of dist/ regardless of whether xdev is installed
 * via npm, npm link, or running from source.
 *
 * The MCP server bundle is built by build.js into the same dist/ directory,
 * so MCP_SERVER_BUNDLE points to a sibling file.
 */
const CURRENT_FILE = fileURLToPath(import.meta.url);
const DIST_DIR = dirname(CURRENT_FILE);
export const MCP_SERVER_BUNDLE = join(DIST_DIR, 'lark-mcp-server.mjs');

/**
 * Where the MCP server file gets deployed on the user's machine.
 *
 * Lives under XDEV_HOME (~/.xdev) but is protected from `xdev clean` /
 * `xdev uninstall` by an explicit allowlist in src/xdev/clean.ts. The
 * deployment is just a copy of dist/lark-mcp-server.mjs.
 */
export const LARK_MCP_DIR_NAME = 'lark-mcp-server';
export const LARK_MCP_DIR = join(XDEV_HOME, LARK_MCP_DIR_NAME);
export const LARK_MCP_FILE = join(LARK_MCP_DIR, 'index.mjs');

/**
 * Name of the MCP server entry inside Claude Desktop's
 * `claude_desktop_config.json` → `mcpServers.<name>`.
 */
export const MCP_SERVER_KEY = 'lark-cli';

export type Platform = 'mac' | 'linux' | 'windows';

export function detectPlatform(): Platform {
  const p = osPlatform();
  switch (p) {
    case 'darwin':
      return 'mac';
    case 'linux':
      return 'linux';
    case 'win32':
      return 'windows';
    default:
      throw new Error(`不支持的平台: ${p}`);
  }
}

/**
 * Resolve the Claude Desktop App's config file path for the given platform.
 *
 * Refs:
 *   mac:     ~/Library/Application Support/Claude/claude_desktop_config.json
 *   linux:   ~/.config/Claude/claude_desktop_config.json
 *   windows: %APPDATA%\Claude\claude_desktop_config.json
 */
export function getClaudeDesktopConfigPath(plat: Platform = detectPlatform()): string {
  switch (plat) {
    case 'mac':
      return join(
        homedir(),
        'Library',
        'Application Support',
        'Claude',
        'claude_desktop_config.json',
      );
    case 'linux':
      return join(homedir(), '.config', 'Claude', 'claude_desktop_config.json');
    case 'windows': {
      const appData = process.env.APPDATA ?? join(homedir(), 'AppData', 'Roaming');
      return join(appData, 'Claude', 'claude_desktop_config.json');
    }
  }
}
