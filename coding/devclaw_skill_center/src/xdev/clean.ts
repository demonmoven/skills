import { Command } from 'commander';
import { existsSync, readFileSync, writeFileSync, type Dirent } from 'node:fs';
import { readdir, rm } from 'node:fs/promises';
import { join } from 'node:path';
import {
  XDEV_HOME,
  CLAUDE_PLUGIN_CACHE,
  CODEX_PLUGIN_CACHE,
  CODEX_CONFIG_FILE,
  MARKETPLACE_NAME,
  PLUGIN_NAME,
} from './paths.js';
import { LARK_MCP_DIR_NAME } from '../lark-mcp/paths.js';

/**
 * Subdirectories under XDEV_HOME that `clean` and `uninstall` must NOT delete.
 *
 * Currently only `lark-mcp-server/` is preserved — it holds the deployed
 * MCP Server file referenced by Claude Desktop's claude_desktop_config.json.
 * Wiping it would silently break Chat/Cowork飞书 capability without any visible
 * trigger, so we treat it as user data, not cache.
 *
 * To remove the MCP Server, use `xdev lark-mcp mac stop` (config-only) or
 * delete `~/.xdev/lark-mcp-server/` manually.
 */
const XDEV_HOME_PRESERVE: ReadonlySet<string> = new Set([LARK_MCP_DIR_NAME]);

/**
 * Selectively wipe XDEV_HOME, leaving entries listed in XDEV_HOME_PRESERVE
 * intact. If XDEV_HOME doesn't exist, no-op.
 */
async function cleanXdevHome(): Promise<{ removed: string[]; preserved: string[] }> {
  const removed: string[] = [];
  const preserved: string[] = [];
  let entries: Dirent[];
  try {
    entries = (await readdir(XDEV_HOME, { withFileTypes: true })) as Dirent[];
  } catch {
    return { removed, preserved };
  }
  for (const entry of entries) {
    const name = String(entry.name);
    const target = join(XDEV_HOME, name);
    if (XDEV_HOME_PRESERVE.has(name)) {
      preserved.push(target);
      continue;
    }
    await rm(target, { recursive: true, force: true });
    removed.push(target);
  }
  return { removed, preserved };
}

export async function runClean(): Promise<void> {
  console.log('[xdev] Cleaning all xdev caches...');
  const xdevResult = await cleanXdevHome();
  for (const path of xdevResult.removed) {
    console.log(`  - ${path}`);
  }
  for (const path of xdevResult.preserved) {
    console.log(`  - kept: ${path} (lark-cli MCP Server, run 'xdev lark-mcp mac stop' to disable)`);
  }
  await rm(CLAUDE_PLUGIN_CACHE, { recursive: true, force: true });
  console.log(`  - ${CLAUDE_PLUGIN_CACHE}`);
  // Codex legacy user-level residue: the previous installToCodex hack wrote
  // plugins into ~/.codex/plugins/cache/xdev-official/ and added a
  // [plugins."xdev@xdev-official"] section to ~/.codex/config.toml. The new
  // project-scoped flow no longer touches either, but we clean both here so
  // users upgrading from the old version don't end up with two parallel
  // mechanisms loading skills.
  await rm(CODEX_PLUGIN_CACHE, { recursive: true, force: true });
  console.log(`  - ${CODEX_PLUGIN_CACHE}`);
  removeLegacyCodexConfigSection();

  // Coco / Codex are project-scoped: the persistent state lives in
  // <project-root>/.coco/coco.yaml and <project-root>/.codex/skills/xdev/
  // respectively. Both are project source files (sometimes checked into VCS).
  // Touching them from a global `clean` command would risk damaging the
  // user's project. To fully remove xdev from a project, manually:
  //   - delete xdev-official marketplace + xdev plugin entries from .coco/coco.yaml
  //   - rm -rf <repo>/.codex/skills/xdev
  console.log('[xdev] Done.');
}

/**
 * Remove the legacy [plugins."xdev@xdev-official"] section from
 * ~/.codex/config.toml left by the pre-A2 installToCodex implementation.
 *
 * Uses a line-anchored regex matching the section header line, then deletes
 * everything until the next [section] header (or end of file). This is
 * deliberately surgical: it does not parse the whole TOML file, so it will
 * not reorder or normalize any other section the user has manually added.
 *
 * No-op if the file doesn't exist or doesn't contain the section.
 */
function removeLegacyCodexConfigSection(): void {
  if (!existsSync(CODEX_CONFIG_FILE)) return;
  const content = readFileSync(CODEX_CONFIG_FILE, 'utf-8');
  const escaped = `${PLUGIN_NAME}@${MARKETPLACE_NAME}`.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const sectionRe = new RegExp(
    String.raw`\n?\[plugins\."` + escaped + String.raw`"\][^\[]*`,
    'm',
  );
  if (!sectionRe.test(content)) return;
  const newContent = content.replace(sectionRe, '').trimEnd() + '\n';
  writeFileSync(CODEX_CONFIG_FILE, newContent);
  console.log(`  - removed [plugins."${PLUGIN_NAME}@${MARKETPLACE_NAME}"] from ${CODEX_CONFIG_FILE}`);
}

export function registerCleanCommand(parent: Command): void {
  parent
    .command('clean')
    .description('清理 xdev 与 plugin 本地缓存')
    .action(async () => {
      await runClean();
    });
}
