/**
 * Type definitions for the third-party marketplace subsystem.
 *
 * The third-party marketplace flow has two halves:
 *
 *  1. Add-update side (`xdev marketplace add-update <git-url>`):
 *     clones an external git repo, walks `marketplaces/` (plural) at the
 *     repo root, validates each `marketplaces/<name>/` subdirectory, and
 *     copies them to `~/.xdev/marketplaces/<name>/`.
 *
 *  2. Launch side (`xdev --cc` etc.):
 *     scans `~/.xdev/marketplaces/`, presents a multi-select prompt to the
 *     user, then installs the chosen marketplaces' plugins to the target
 *     coding agent (CC / Codex / Coco / Trae) using the same install
 *     strategies as the built-in xdev plugin.
 *
 * Both halves share the same plugin/marketplace info shape, defined here.
 */

/**
 * One plugin inside an external marketplace.
 */
export interface ExternalPluginInfo {
  /** plugin name (lowercase + dash, must match the directory name under plugins/) */
  name: string;
  /** absolute path: ~/.xdev/marketplaces/<mkp>/plugins/<plugin>/ */
  path: string;
  /** parent marketplace name, for diagnostics and conflict detection */
  marketplaceName: string;
  /** how many SKILL.md files this plugin contains under skills/ */
  skillCount: number;
}

/**
 * One external marketplace, as discovered by scanExternalMarketplaces().
 */
export interface ExternalMarketplaceInfo {
  /** marketplace name (must match marketplace.json's `name` field) */
  name: string;
  /** absolute path: ~/.xdev/marketplaces/<name>/ */
  path: string;
  /** discovered plugin entries under <path>/plugins/ */
  plugins: ExternalPluginInfo[];
  /** convenience: plugins.length */
  pluginCount: number;
  /** convenience: sum of every plugin's skillCount */
  skillCount: number;
}

/**
 * Result of `xdev marketplace add-update` discovery — represents one valid
 * `marketplaces/<name>/` directory found inside the cloned external repo
 * before it is copied to `~/.xdev/marketplaces/`.
 */
export interface DiscoveredMarketplace {
  /** marketplace name from marketplace.json */
  name: string;
  /** absolute path inside the temporary clone, e.g. /tmp/xdev-mkp-xxx/marketplaces/<name>/ */
  sourcePath: string;
  /** how many top-level plugin directories under <sourcePath>/plugins/ */
  pluginCount: number;
  /** total skill count across all plugins (sum of SKILL.md files) */
  skillCount: number;
}

/**
 * One plugin name conflict between two external marketplaces, detected at
 * launch time before installing to the target coding agent.
 *
 * If the user selects two marketplaces that contain a plugin with the same
 * name, the launcher refuses to install anything and reports the conflicts
 * via this shape.
 */
export interface PluginNameConflict {
  pluginName: string;
  occurrences: Array<{ marketplaceName: string; pluginPath: string }>;
}
