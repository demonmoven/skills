/**
 * Implementation of `xdev marketplace add-update <git-url>`.
 *
 * Flow:
 *   1. mkdtemp() → /tmp/xdev-mkp-<random>
 *   2. git clone <git-url> [--branch <name>] /tmp/xdev-mkp-<random>
 *   3. validate /tmp/xdev-mkp-<random>/marketplaces/ exists (plural!)
 *   4. for each subdir <name> under marketplaces/:
 *        - parse <name>/.claude-plugin/marketplace.json
 *        - validate name == directory name
 *        - validate plugins/ exists
 *        - count plugins + skills
 *   5. detect name conflicts vs ~/.xdev/marketplaces/<name>/
 *   6. if conflicts && !--force: ask Y/N (TTY) or refuse (non-TTY)
 *   7. backup conflicting marketplaces to ~/.xdev/marketplaces.bak/<timestamp>/<name>/
 *   8. cp -R each <subdir> → ~/.xdev/marketplaces/<name>/
 *   9. rm -rf /tmp/xdev-mkp-<random>
 *  10. report what was installed/updated
 */

import { execFileSync } from 'node:child_process';
import {
  cpSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  readdirSync,
  rmSync,
  statSync,
} from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { createInterface } from 'node:readline/promises';
import { EXTERNAL_MARKETPLACES_BAK_DIR, EXTERNAL_MARKETPLACES_DIR } from '../paths.js';
import type { DiscoveredMarketplace } from './types.js';

export interface AddUpdateOptions {
  branch?: string;
  force?: boolean;
}

/**
 * Entry point for `xdev marketplace add-update <git-url>`.
 */
export async function addUpdateMarketplaces(
  gitUrl: string,
  options: AddUpdateOptions = {},
): Promise<void> {
  // Step 1: temp directory
  const tmpRoot = mkdtempSync(join(tmpdir(), 'xdev-mkp-'));
  console.log(`[xdev] Cloning ${gitUrl} into ${tmpRoot}...`);

  let discovered: DiscoveredMarketplace[];
  try {
    // Step 2: git clone
    cloneRepo(gitUrl, tmpRoot, options.branch);

    // Step 3-4: discover marketplaces/
    discovered = discoverMarketplaces(tmpRoot);
    if (discovered.length === 0) {
      throw new Error(
        `[xdev] No valid marketplaces found under ${tmpRoot}/marketplaces/. ` +
          `Make sure the repo root contains a 'marketplaces/' directory (plural).`,
      );
    }

    console.log(`[xdev] Discovered ${discovered.length} valid marketplace(s):`);
    for (const m of discovered) {
      console.log(`  - ${m.name} (${m.pluginCount} plugin / ${m.skillCount} skill)`);
    }

    // Step 5: name conflict check
    const conflicts = discovered.filter((m) => existsSync(join(EXTERNAL_MARKETPLACES_DIR, m.name)));

    if (conflicts.length > 0 && !options.force) {
      // Step 6: ask Y/N (TTY only)
      const proceed = await confirmOverwrite(conflicts);
      if (!proceed) {
        console.log('[xdev] Aborted. No changes made.');
        return;
      }
    }

    // Step 7: backup conflicting
    let backupDir: string | null = null;
    if (conflicts.length > 0) {
      backupDir = backupConflicts(conflicts);
    }

    // Step 8: cp each marketplace
    mkdirSync(EXTERNAL_MARKETPLACES_DIR, { recursive: true });
    for (const m of discovered) {
      const targetPath = join(EXTERNAL_MARKETPLACES_DIR, m.name);
      // Wipe the existing target (we already backed it up if it was a conflict).
      rmSync(targetPath, { recursive: true, force: true });
      cpSync(m.sourcePath, targetPath, { recursive: true, dereference: true });
    }

    // Step 10: report
    console.log('');
    console.log(`✅ Installed/updated ${discovered.length} marketplace(s):`);
    for (const m of discovered) {
      console.log(`  - ${m.name} (${m.pluginCount} plugin / ${m.skillCount} skill)`);
    }
    if (backupDir) {
      console.log(`Backup of overwritten marketplaces: ${backupDir}`);
    }
    console.log('');
    console.log('Next time you run `xdev --cc` (or any agent), you will be prompted to');
    console.log('select which third-party marketplaces to load alongside the built-in xdev plugin.');
  } finally {
    // Step 9: always cleanup the temp clone
    rmSync(tmpRoot, { recursive: true, force: true });
  }
}

/**
 * Run `git clone` synchronously into the given target directory.
 */
function cloneRepo(gitUrl: string, target: string, branch?: string): void {
  const args = ['clone', '--depth', '1'];
  if (branch) {
    args.push('--branch', branch);
  }
  args.push(gitUrl, target);
  try {
    execFileSync('git', args, { stdio: 'inherit' });
  } catch (err) {
    throw new Error(`[xdev] git clone failed: ${(err as Error).message}`);
  }
}

/**
 * Walk <tmpRoot>/marketplaces/<name>/ and validate each subdirectory.
 *
 * Throws if `marketplaces/` itself doesn't exist. Returns an empty array if
 * `marketplaces/` exists but contains no valid subdirectories (caller will
 * decide what to do).
 */
function discoverMarketplaces(tmpRoot: string): DiscoveredMarketplace[] {
  const marketplacesRoot = join(tmpRoot, 'marketplaces');
  if (!existsSync(marketplacesRoot)) {
    throw new Error(
      `[xdev] No 'marketplaces/' directory found at the repo root (${tmpRoot}). ` +
        `Note: it must be plural ('marketplaces/'), not singular ('marketplace/').`,
    );
  }

  const result: DiscoveredMarketplace[] = [];

  for (const entry of readdirSync(marketplacesRoot, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    const subPath = join(marketplacesRoot, entry.name);
    const validated = validateOne(subPath, entry.name);
    if (validated) {
      result.push(validated);
    }
  }

  return result;
}

/**
 * Parse and validate one `marketplaces/<name>/` subdirectory.
 *
 * Returns null (with stderr warning) on any validation failure.
 */
function validateOne(subPath: string, dirName: string): DiscoveredMarketplace | null {
  const marketplaceJsonPath = join(subPath, '.claude-plugin', 'marketplace.json');
  if (!existsSync(marketplaceJsonPath)) {
    console.warn(`[xdev] Skipping ${subPath}: missing .claude-plugin/marketplace.json`);
    return null;
  }

  let parsed: { name?: string };
  try {
    parsed = JSON.parse(readFileSync(marketplaceJsonPath, 'utf-8'));
  } catch (err) {
    console.warn(`[xdev] Skipping ${subPath}: invalid JSON: ${(err as Error).message}`);
    return null;
  }

  if (!parsed.name || parsed.name !== dirName) {
    console.warn(
      `[xdev] Skipping ${subPath}: directory '${dirName}' does not match marketplace.json name '${parsed.name ?? '(missing)'}'`,
    );
    return null;
  }

  const pluginsDir = join(subPath, 'plugins');
  if (!existsSync(pluginsDir)) {
    console.warn(`[xdev] Skipping ${subPath}: missing plugins/ subdirectory`);
    return null;
  }

  let pluginCount = 0;
  let skillCount = 0;
  for (const entry of readdirSync(pluginsDir, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    pluginCount++;
    const skillsRoot = join(pluginsDir, entry.name, 'skills');
    if (existsSync(skillsRoot)) {
      for (const skillEntry of readdirSync(skillsRoot, { withFileTypes: true })) {
        if (!skillEntry.isDirectory()) continue;
        const skillMd = join(skillsRoot, skillEntry.name, 'SKILL.md');
        try {
          if (statSync(skillMd).isFile()) skillCount++;
        } catch {
          // skip
        }
      }
    }
  }

  return {
    name: dirName,
    sourcePath: subPath,
    pluginCount,
    skillCount,
  };
}

/**
 * Interactive Y/N prompt to confirm overwriting existing marketplaces.
 *
 * Refuses (returns false) on non-TTY stdin so CI runs don't hang.
 * Pass --force to skip this check entirely (handled by the caller).
 */
async function confirmOverwrite(conflicts: DiscoveredMarketplace[]): Promise<boolean> {
  if (!process.stdin.isTTY) {
    console.error('[xdev] Refusing to overwrite without --force (non-interactive stdin)');
    console.error('[xdev] Conflicting marketplaces:');
    for (const c of conflicts) {
      console.error(`  - ${c.name}`);
    }
    return false;
  }

  console.log('');
  console.log(`Detected ${conflicts.length} marketplace(s) already installed locally:`);
  for (const c of conflicts) {
    console.log(`  - ${c.name}  (will be overwritten)`);
  }
  console.log('');
  console.log(`Existing copies will be backed up to ${EXTERNAL_MARKETPLACES_BAK_DIR}/<timestamp>/`);
  console.log('');

  const rl = createInterface({ input: process.stdin, output: process.stdout });
  try {
    const answer = (await rl.question('Continue? [Y/N]: ')).trim().toLowerCase();
    return answer === 'y' || answer === 'yes';
  } finally {
    rl.close();
  }
}

/**
 * Backup the conflicting marketplaces to ~/.xdev/marketplaces.bak/<timestamp>/<name>/.
 *
 * Returns the timestamped backup root directory.
 */
function backupConflicts(conflicts: DiscoveredMarketplace[]): string {
  const stamp = formatTimestamp(new Date());
  const backupRoot = join(EXTERNAL_MARKETPLACES_BAK_DIR, stamp);
  mkdirSync(backupRoot, { recursive: true });

  for (const c of conflicts) {
    const src = join(EXTERNAL_MARKETPLACES_DIR, c.name);
    const dst = join(backupRoot, c.name);
    cpSync(src, dst, { recursive: true, dereference: true });
  }

  console.log(`[xdev] Backed up ${conflicts.length} marketplace(s) to ${backupRoot}`);
  return backupRoot;
}

/**
 * Format a Date as YYYYMMDD-HHMMSS for backup directory naming.
 */
function formatTimestamp(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0');
  return (
    `${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}` +
    `-${pad(d.getHours())}${pad(d.getMinutes())}${pad(d.getSeconds())}`
  );
}
