import { spawnSync } from 'node:child_process';
import { rm } from 'node:fs/promises';
import { Command } from 'commander';
import { runClean } from './clean.js';
import { LARK_MCP_DIR } from '../lark-mcp/paths.js';
import { cleanupDesktopConfigOnUninstall } from '../lark-mcp/actions.js';

/**
 * Self-uninstall command — `xdev uninstall`.
 *
 * Two-phase cleanup:
 *
 *   1. Run `runClean()` first (while the xdev binary is still on disk and
 *      our own dist/index.js is still loaded into Node memory). This wipes
 *      ~/.xdev/, ~/.claude/plugins/cache/xdev-official/, and the
 *      ~/.codex/plugins/cache/xdev-official/ + ~/.codex/config.toml legacy
 *      residue from the pre-project-scope era.
 *
 *   2. Then `spawnSync('npm', ['uninstall', '-g', '@byted/xdex'])` to remove
 *      the global npm package itself. Wrapped in a soft-fail check because:
 *
 *      - In npx mode (NPM_CONFIG_REGISTRY=... npx @byted/xdex uninstall),
 *        the package isn't installed globally — it's just running from npx
 *        cache. The npm uninstall -g will exit non-zero with "not installed".
 *        That's expected and not an error: npx-mode users intentionally never
 *        installed globally, they just want the cleanup-only behavior.
 *
 *      - The user may have already run `npm uninstall -g` manually before
 *        invoking `xdev uninstall`. The double-uninstall should still succeed
 *        with a no-op.
 *
 * Project-level files (`.codex/skills/xdev/`, `.coco/coco.yaml` xdev sections,
 * `.trae/skills/xdev-*`, `.claude/settings.json` enabledPlugins) are NEVER
 * touched — those are project source files that may be checked into the
 * user's version control. The user must clean those manually if desired.
 *
 * Self-removal safety: `npm uninstall -g` will delete the dist/index.js file
 * we're currently running from. Node has already mmap'd it into memory, so
 * the running process keeps working until exit. This is the same self-
 * removal pattern used by brew uninstall, gh extension uninstall, etc.
 */

const PACKAGE_NAME = '@byted/xdex';
const NPM_TIMEOUT_MS = 30_000;

export async function runUninstall(): Promise<void> {
  console.log('[xdev] Uninstalling xdev...');
  console.log('');

  // Phase 1: clean all caches and legacy residue
  await runClean();
  console.log('');

  // Phase 1b: explicitly remove the lark-cli MCP Server (which `clean`
  // intentionally preserves). On full uninstall the user clearly wants
  // everything gone, so we drop both the deployed file and the
  // Claude Desktop config entry that references it.
  console.log('[xdev] Removing lark-cli MCP Server (deployed file + Desktop config entry)');
  await rm(LARK_MCP_DIR, { recursive: true, force: true });
  console.log(`  - ${LARK_MCP_DIR}`);
  cleanupDesktopConfigOnUninstall();
  console.log('');

  // Phase 2: try to remove the global npm package
  console.log(`[xdev] Removing global npm package: ${PACKAGE_NAME}`);
  try {
    const result = spawnSync('npm', ['uninstall', '-g', PACKAGE_NAME], {
      stdio: ['ignore', 'inherit', 'inherit'],
      timeout: NPM_TIMEOUT_MS,
    });
    if (result.error) {
      console.log(
        `  (skipped: ${result.error.message} — likely npm not on PATH)`,
      );
    } else if (result.status !== 0) {
      console.log(
        `  (skipped: ${PACKAGE_NAME} was not a global install. ` +
          `In npx mode this is expected; if you ran 'npm uninstall -g' before this, also expected.)`,
      );
    } else {
      console.log(`  ✓ ${PACKAGE_NAME} removed from global npm`);
    }
  } catch (err) {
    console.log(
      `  (skipped: ${err instanceof Error ? err.message : String(err)})`,
    );
  }

  console.log('');
  console.log('[xdev] ✓ uninstall complete');
  console.log('');
  console.log('The following project-level files were NOT touched (they may be in your VCS):');
  console.log('  - <project-root>/.codex/skills/xdev/        (codex plugin install)');
  console.log('  - <project-root>/.coco/coco.yaml            (coco marketplaces / plugins entries for xdev)');
  console.log('  - <project-root>/.trae/skills/xdev-*        (trae IDE skill install)');
  console.log('  - <project-root>/.claude/settings.json      (cc enabledPlugins entry for xdev@xdev-official)');
  console.log('');
  console.log('To clean these too, manually remove them from each project.');
}

export function registerUninstallCommand(parent: Command): void {
  parent
    .command('uninstall')
    .description('彻底卸载 xdev (清理所有 cache + npm 全局包,但不动项目级文件)')
    .action(async () => {
      await runUninstall();
    });
}
