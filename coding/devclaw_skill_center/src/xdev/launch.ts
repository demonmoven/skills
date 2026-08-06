import { launchClaudeCode, launchCodex, launchTrae, launchTraeCn, launchCoco } from './launcher.js';
import { promptAgent, type Agent } from './prompt.js';
import { scanExternalMarketplaces } from './marketplace/scan.js';
import {
  promptExternalMarketplaceSelection,
  detectPluginNameConflicts,
  formatConflictError,
} from './marketplace/prompt-selection.js';
import { installExternalMarketplacesToAgent } from './marketplace/install.js';
import { listAvailableSkills, promptSkillSelection } from './skill-selector.js';
import { ensureBytedcliDeps } from './bytedcli-deps.js';
import { ensureTraceAuth } from '../trace/commands/auth.js';
import type { ExternalMarketplaceInfo } from './marketplace/types.js';

export interface LaunchOptions {
  cc?: boolean;
  cdx?: boolean;
  trae?: boolean;
  traeCn?: boolean;
  coco?: boolean;
  /** Skip the third-party marketplace multi-select (CI / scripts) */
  noExternal?: boolean;
  /** Skip skill selection for Trae/Trae CN, install all skills */
  allSkills?: boolean;
  /**
   * Controls the bytedcli fixloop-prerequisite dependency check performed
   * right before the xdev plugin is installed into the selected agent.
   * Commander turns `--no-bytedcli-deps` into `bytedcliDeps: false`; the
   * default is `true` (check enabled). Set `XDEV_SKIP_BYTEDCLI_DEPS=1` to
   * disable via environment.
   */
  bytedcliDeps?: boolean;
  /**
   * Controls the Byte SSO trace authentication check performed before
   * launching an agent. Commander turns `--no-trace-auth` into
   * `traceAuth: false`; the default is `true` (check enabled).
   * Set `XDEV_SKIP_TRACE_AUTH=1` to disable via environment.
   */
  traceAuth?: boolean;
}

/**
 * Default xdev action: resolve agent → install plugin → spawn agent.
 *
 * Wired up as the top-level program action in src/index.ts, so users invoke
 * it via bare `xdev`, `xdev --trae-cn`, `xdev --trae`, `xdev --coco`,
 * `xdev --cc`, `xdev --cdx`. No `xdev launch` subcommand exists — keeps the
 * surface identical to the original bash xdev.
 *
 * The marketplace content (skills/plugins) is read directly from the npm
 * package on disk via paths.ts:PLUGIN_SRC_DIR — no git sync, no
 * `~/.xdev/marketplace/` cache. The dev-mode case (working
 * on xdev itself) is automatic: `make install` runs `npm link` which points
 * the global `xdev` bin at the source repo's dist/, so PLUGIN_SRC_DIR
 * resolves to the working tree's marketplace/ directory.
 *
 * Update check (synchronous, may exit if upgrade succeeds) runs in
 * src/index.ts top level BEFORE commander parseAsync, so all xdev commands
 * including --version / --help are covered.
 */
export async function runLaunch(options: LaunchOptions, extraArgs: string[]): Promise<void> {
  // Resolve agent — at most one of --trae-cn/--trae/--coco/--cc/--cdx may be set
  const selected: Agent[] = [];
  if (options.traeCn) selected.push('trae-cn');
  if (options.trae) selected.push('trae');
  if (options.coco) selected.push('coco');
  if (options.cc) selected.push('cc');
  if (options.cdx) selected.push('cdx');

  if (selected.length > 1) {
    console.error('[xdev] Error: --trae-cn, --trae, --coco, --cc, --cdx are mutually exclusive');
    process.exit(1);
  }

  const agent: Agent = selected.length === 1 ? selected[0] : await promptAgent();

  // ─── Third-party marketplace multi-select ───
  // Scans ~/.xdev/marketplaces/ and prompts the user to pick which to load
  // alongside the built-in xdev plugin. Skipped on --no-external or non-TTY.
  let externalSelection: ExternalMarketplaceInfo[] = [];
  if (!options.noExternal) {
    const available = scanExternalMarketplaces();
    if (available.length > 0) {
      externalSelection = await promptExternalMarketplaceSelection(available);

      // Plugin-name conflict detection across the selected marketplaces.
      // If two selected marketplaces both contain a plugin with the same
      // name, we refuse to install ANY of them and exit early.
      const conflicts = detectPluginNameConflicts(externalSelection);
      if (conflicts.length > 0) {
        console.error(formatConflictError(conflicts));
        process.exit(1);
      }
    }
  }

  // ─── Trae / Trae CN skill multi-select ───
  // Prompts the user to pick which built-in xdev skills to install.
  // Skipped on --all-skills, non-TTY, or non-Trae agents.
  let selectedSkills: string[] | undefined;
  if ((agent === 'trae' || agent === 'trae-cn') && !options.allSkills) {
    const skills = listAvailableSkills();
    selectedSkills = await promptSkillSelection(skills);
  }

  // ─── Trace SSO authentication ───
  // Ensures the user is logged in via Byte SSO so that trace uploads carry a
  // real userId instead of "anonymous". Opt out via `--no-trace-auth` or env
  // `XDEV_SKIP_TRACE_AUTH=1` (CI / offline).
  if (options.traceAuth !== false && !process.env['XDEV_SKIP_TRACE_AUTH']) {
    await ensureTraceAuth();
  }

  // ─── bytedcli dependency check (fixloop prerequisite) ───
  // Runs before any xdev plugin installation so that fixloop's auto-discovery
  // (`for dir in .*/ "$HOME"/.*/` looking for `skills/bytedance-tce/SKILL.md`)
  // always resolves. Opt out via `--no-bytedcli-deps` or env
  // `XDEV_SKIP_BYTEDCLI_DEPS=1` (CI / offline).
  if (options.bytedcliDeps !== false) {
    await ensureBytedcliDeps(agent);
  }

  switch (agent) {
    case 'trae-cn':
      launchTraeCn(extraArgs, externalSelection, selectedSkills);
      break;
    case 'trae':
      launchTrae(extraArgs, externalSelection, selectedSkills);
      break;
    case 'coco':
      launchCoco(extraArgs, externalSelection);
      break;
    case 'cc':
      launchClaudeCode(extraArgs, externalSelection);
      break;
    case 'cdx':
      launchCodex(extraArgs, externalSelection);
      break;
  }
}
