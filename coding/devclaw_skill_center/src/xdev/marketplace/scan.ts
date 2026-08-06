/**
 * Scan `~/.xdev/marketplaces/` and report all valid third-party marketplaces
 * + their plugins.
 *
 * Called from `runLaunch()` (src/xdev/launch.ts) at xdev startup, before the
 * multi-select prompt. Also called from `xdev marketplace list` (future).
 *
 * "Valid" means:
 *  - subdirectory under EXTERNAL_MARKETPLACES_DIR
 *  - contains `.claude-plugin/marketplace.json` whose `name` field equals the
 *    subdirectory name
 *  - contains a `plugins/` subdirectory
 *
 * Invalid subdirectories are silently skipped (with a warning to stderr).
 */

import { existsSync, readFileSync, readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { EXTERNAL_MARKETPLACES_DIR } from '../paths.js';
import type { ExternalMarketplaceInfo, ExternalPluginInfo } from './types.js';

/**
 * Scan ~/.xdev/marketplaces/ and return all valid marketplaces.
 *
 * Returns an empty array if the directory doesn't exist (first-time user
 * who never ran `xdev marketplace add-update`).
 */
export function scanExternalMarketplaces(): ExternalMarketplaceInfo[] {
  if (!existsSync(EXTERNAL_MARKETPLACES_DIR)) {
    return [];
  }

  const result: ExternalMarketplaceInfo[] = [];

  for (const entry of readdirSync(EXTERNAL_MARKETPLACES_DIR, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    const mkpPath = join(EXTERNAL_MARKETPLACES_DIR, entry.name);
    const info = parseMarketplace(mkpPath, entry.name);
    if (info) {
      result.push(info);
    }
  }

  // Sort by name for stable display order in the multi-select prompt.
  result.sort((a, b) => a.name.localeCompare(b.name));
  return result;
}

/**
 * Parse one `<root>/<name>/` directory and validate its structure.
 *
 * Returns null (with stderr warning) if any of:
 *  - missing `.claude-plugin/marketplace.json`
 *  - marketplace.json has no/wrong `name` field
 *  - missing `plugins/` directory
 */
function parseMarketplace(mkpPath: string, dirName: string): ExternalMarketplaceInfo | null {
  const marketplaceJsonPath = join(mkpPath, '.claude-plugin', 'marketplace.json');
  if (!existsSync(marketplaceJsonPath)) {
    console.warn(
      `[xdev] Skipping marketplace at ${mkpPath}: missing .claude-plugin/marketplace.json`,
    );
    return null;
  }

  let parsed: { name?: string };
  try {
    parsed = JSON.parse(readFileSync(marketplaceJsonPath, 'utf-8'));
  } catch (err) {
    console.warn(`[xdev] Skipping marketplace at ${mkpPath}: invalid JSON: ${(err as Error).message}`);
    return null;
  }

  if (!parsed.name || parsed.name !== dirName) {
    console.warn(
      `[xdev] Skipping marketplace at ${mkpPath}: directory name '${dirName}' does not match marketplace.json name '${parsed.name ?? '(missing)'}'`,
    );
    return null;
  }

  const pluginsDir = join(mkpPath, 'plugins');
  if (!existsSync(pluginsDir)) {
    console.warn(`[xdev] Skipping marketplace at ${mkpPath}: missing plugins/ directory`);
    return null;
  }

  const plugins = parsePlugins(pluginsDir, dirName);

  return {
    name: dirName,
    path: mkpPath,
    plugins,
    pluginCount: plugins.length,
    skillCount: plugins.reduce((sum, p) => sum + p.skillCount, 0),
  };
}

/**
 * Walk `<mkp>/plugins/` and collect each plugin subdirectory.
 *
 * Counts `<plugin>/skills/<skill>/SKILL.md` files for the display label.
 */
function parsePlugins(pluginsDir: string, marketplaceName: string): ExternalPluginInfo[] {
  const result: ExternalPluginInfo[] = [];
  for (const entry of readdirSync(pluginsDir, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    const pluginPath = join(pluginsDir, entry.name);
    const skillCount = countSkills(join(pluginPath, 'skills'));
    result.push({
      name: entry.name,
      path: pluginPath,
      marketplaceName,
      skillCount,
    });
  }
  return result;
}

/**
 * Count SKILL.md files directly under <pluginDir>/skills/<sub>/SKILL.md.
 *
 * Does not recurse — skill nesting deeper than one level is not standard.
 */
function countSkills(skillsDir: string): number {
  if (!existsSync(skillsDir)) return 0;
  let count = 0;
  for (const entry of readdirSync(skillsDir, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    const skillMdPath = join(skillsDir, entry.name, 'SKILL.md');
    try {
      if (statSync(skillMdPath).isFile()) count++;
    } catch {
      // missing or unreadable, skip
    }
  }
  return count;
}
