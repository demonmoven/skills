import { execFileSync } from 'node:child_process';
import { existsSync } from 'node:fs';
import { homedir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

export const MARKETPLACE_NAME = 'xdev-official';
export const PLUGIN_NAME = 'xdev';

/**
 * Self-contained marketplace path resolution (replaces the old git rolling
 * sync model).
 *
 * Background: previously, xdev shipped with `~/.xdev/marketplace/` as a cache
 * dir that was populated on each startup by sparse-cloning the repo from
 * git@code.byted.org:stone/devclaw_skills_center.git. This was a relic from
 * when marketplace lived only in the git repo. Once we moved to publishing
 * @byted/xdex to bnpm and including `marketplace/` in the npm package
 * `files` field (commit 97fff2c), the npm install already places marketplace
 * on disk — but sync.ts still ran, deleted the user's cache, then tried to
 * re-fetch from git, then failed (main branch hadn't merged the marketplace
 * refactor yet) and left the user with no marketplace at all.
 *
 * The fix: stop using `~/.xdev/marketplace/` entirely. Read marketplace
 * directly from the npm package itself, which is on disk thanks to the
 * `files` field in package.json. The path resolves automatically:
 *
 *   production: /opt/homebrew/lib/node_modules/@byted/xdex/dist/index.js
 *               → /opt/homebrew/lib/node_modules/@byted/xdex/marketplace/
 *
 *   source link: <repo>/dist/index.js
 *               → <repo>/marketplace/
 *
 * The dev-mode case (working on xdev itself) is automatic: `make install`
 * runs `npm link` which points the global `xdev` bin at `<repo>/dist/index.js`,
 * so `import.meta.url` resolves to the source repo and PLUGIN_SRC_DIR is the
 * working tree. The --debug flag was re-introduced (see src/xdev/debug.ts)
 * as a meta-command that builds from $XDEV_DEV_REPO and re-execs the local
 * build — it does NOT change the path resolution here.
 *
 * `import.meta.url` works in both ESM Node and esbuild bundles — esbuild
 * preserves it as-is for ESM output (we set `format: 'esm'` in build.js).
 */
const PATHS_FILE = fileURLToPath(import.meta.url);
const PATHS_DIR = dirname(PATHS_FILE);

/**
 * Walk up from `start` until we find a directory containing `package.json`.
 * Resilient to both production (dist/index.js → ../package.json) and dev /
 * vitest (src/xdev/paths.ts → ../../package.json).
 */
function findPackageRoot(start: string): string {
  let dir = start;
  while (true) {
    if (existsSync(join(dir, 'package.json'))) return dir;
    const parent = dirname(dir);
    if (parent === dir) {
      // No package.json found anywhere — fall back to the original 1-up
      // behavior so we surface a downstream "marketplace not found" error
      // rather than crashing inside paths.ts.
      return resolve(start, '..');
    }
    dir = parent;
  }
}

const PACKAGE_ROOT = findPackageRoot(PATHS_DIR);

export const MARKETPLACE_DIR = join(PACKAGE_ROOT, 'marketplace');
export const PLUGIN_SRC_DIR = join(MARKETPLACE_DIR, 'plugins', PLUGIN_NAME);

/**
 * Workspace scene config locations.
 *
 * `xdev workspace create --scene <name>` looks here in order:
 *   1. SCENES_DIR_USER  — `~/.xdev/scenes/<name>.json` (user override, optional)
 *   2. SCENES_DIR_BUNDLED — `<PLUGIN_SRC_DIR>/scenes/<name>.json` (shipped in npm pkg)
 *
 * The bundled directory is published with the npm package because the parent
 * `marketplace/` is already in package.json's `files` field. Adding new
 * scenes is a matter of dropping a JSON file alongside.
 */
export const SCENES_DIR_BUNDLED = join(PLUGIN_SRC_DIR, 'scenes');

/**
 * `xdev claude-settings mac setup` template locations. User override wins
 * over the bundled copy. The bundled file ships with the npm package via
 * the `marketplace/` files entry in package.json.
 */
export const CLAUDE_SETTINGS_TEMPLATE_BUNDLED = join(
  PLUGIN_SRC_DIR,
  'claude-settings',
  'settings.template.json',
);

/**
 * ~/.xdev/ — kept around as the parent directory for the version-check
 * cache file. The old marketplace cache that lived under here is gone.
 */
export const XDEV_HOME = join(homedir(), '.xdev');

/**
 * Third-party marketplace directories.
 *
 * `~/.xdev/marketplaces/` (plural) holds marketplaces installed by
 * `xdev marketplace add-update <git-url>`. Each subdirectory is one
 * marketplace, named after its `marketplace.json` `name` field.
 *
 * `~/.xdev/marketplaces.bak/<timestamp>/` holds backups created when an
 * existing marketplace is overwritten by a new add-update. Backups are
 * never auto-cleaned; users may rm them manually.
 *
 * NOTE: Do NOT confuse with the legacy `~/.xdev/marketplace/` (singular)
 * which was a deprecated git rolling-sync cache. We use the plural form
 * to avoid colliding with any leftover state on user machines.
 */
export const EXTERNAL_MARKETPLACES_DIR = join(XDEV_HOME, 'marketplaces');
export const EXTERNAL_MARKETPLACES_BAK_DIR = join(XDEV_HOME, 'marketplaces.bak');

export const SCENES_DIR_USER = join(XDEV_HOME, 'scenes');

export const CLAUDE_SETTINGS_TEMPLATE_USER = join(
  XDEV_HOME,
  'claude-settings',
  'settings.template.json',
);

export const CLAUDE_PLUGIN_CACHE = join(homedir(), '.claude', 'plugins', 'cache', MARKETPLACE_NAME);
export const CODEX_PLUGIN_CACHE = join(homedir(), '.codex', 'plugins', 'cache', MARKETPLACE_NAME);
export const CODEX_CONFIG_FILE = join(homedir(), '.codex', 'config.toml');

/**
 * Cache file for the CLI startup version check.
 *
 * Stores the last known latest version (fetched from bnpm) and timestamp,
 * so that we can throttle the registry hit to once per 24h and let the
 * runtime version-check decide whether to spawn `npm install -g`.
 *
 * See src/xdev/version-check.ts.
 */
export const VERSION_CHECK_CACHE = join(XDEV_HOME, '.last-version-check.json');

/**
 * Cache file for the bytedcli CLI upgrade check.
 *
 * bytedcli is an independent tool whose lifecycle is unrelated to xdev's own
 * version. We throttle the `npm view @bytedance-dev/bytedcli version` call to
 * once per 12h so that `xdev` startup isn't slowed down by a second registry
 * round-trip on every launch.
 */
export const BYTEDCLI_CHECK_CACHE = join(XDEV_HOME, '.last-bytedcli-check.json');

/**
 * Locate the project root for the given working directory.
 *
 * Used by the Coco / TRAE CLI integration: coco's project-level config
 * (`.coco/coco.yaml`) is only loaded when the cwd matches the directory
 * containing it — coco 0.120.16 does NOT walk up the parent chain. To make
 * `xdev --coco` work from any subdirectory of a project, we resolve the
 * project root upfront and (1) write the config at the root, (2) spawn coco
 * with cwd set to the root.
 *
 * Resolution order:
 *   1. `git rev-parse --show-toplevel` from `cwd` — works for any git repo
 *   2. `cwd` itself — fallback for non-git directories
 *
 * Caller is responsible for refusing to write into $HOME (the project root
 * equaling homedir() means the user ran xdev from their home directory,
 * which would silently pollute their global config).
 */
export function findProjectRoot(cwd: string): string {
  try {
    const out = execFileSync('git', ['rev-parse', '--show-toplevel'], {
      cwd,
      stdio: ['ignore', 'pipe', 'ignore'],
      encoding: 'utf-8',
    });
    const top = out.trim();
    if (top) return top;
  } catch {
    // not a git repo, or git not installed; fall through
  }
  return cwd;
}
