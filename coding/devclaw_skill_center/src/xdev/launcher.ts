import { spawn, spawnSync, execFileSync } from 'node:child_process';
import { installToClaudeCode, installToCodex, installToTrae, installToCoco } from './plugin-injector.js';
import { installExternalMarketplacesToAgent } from './marketplace/install.js';
import type { ExternalMarketplaceInfo } from './marketplace/types.js';
import { injectHooksToClaudeCode, injectHooksToCodex } from './hook-injector.js';
import { ensureNpxOrWarn } from './npx-check.js';
import { ensureAgentCliOrExit } from './agent-cli-check.js';
import { applyBundledCredsIfMissing } from '../trace/bundled-creds.js';

/**
 * Ensure ab-agent-collect-event is globally installed so hooks can call it
 * directly instead of going through npx (which re-downloads the package every time).
 */
function ensureAbAgentCollectEvent(): void {
  if (spawnSync('which', ['ab-agent-collect-event'], { stdio: 'ignore' }).status === 0) {
    return; // already installed
  }
  try {
    console.log('[xdev] Installing ab-agent-collect-event globally...');
    execFileSync(
      'npm',
      ['install', '-g', '@dp/ab-agent-collect-event', '--registry', 'https://bnpm.byted.org'],
      { stdio: ['ignore', 'pipe', 'pipe'], timeout: 30_000 },
    );
    console.log('[xdev] ab-agent-collect-event installed.');
  } catch (err) {
    console.warn(`[xdev] Warning: failed to install ab-agent-collect-event globally: ${(err as Error).message}`);
    console.warn('[xdev]   Code stat hooks will silently skip. This does not affect Claude Code.');
  }
}

function spawnAndExit(cmd: string, args: string[], options: { cwd?: string } = {}): void {
  const child = spawn(cmd, args, { stdio: 'inherit', cwd: options.cwd });
  child.on('exit', (code) => process.exit(code ?? 0));
  child.on('error', (err: NodeJS.ErrnoException) => {
    if (err.code === 'ENOENT') {
      console.error(`[xdev] Command not found: ${cmd}. Make sure it is installed and on PATH.`);
    } else {
      console.error(`[xdev] Failed to spawn ${cmd}: ${err.message}`);
    }
    process.exit(127);
  });
}

export function launchClaudeCode(
  extraArgs: string[],
  externalMarketplaces: ExternalMarketplaceInfo[] = [],
): void {
  ensureAgentCliOrExit('cc');
  ensureNpxOrWarn();
  ensureAbAgentCollectEvent();
  applyBundledCredsIfMissing();
  installToClaudeCode();
  installExternalMarketplacesToAgent(externalMarketplaces, 'cc');
  injectHooksToClaudeCode();
  console.log('[xdev] Launching Claude Code...');
  spawnAndExit('claude', ['--dangerously-skip-permissions', ...extraArgs]);
}

export function launchCodex(
  extraArgs: string[],
  externalMarketplaces: ExternalMarketplaceInfo[] = [],
): void {
  // installToCodex now writes to <project-root>/.codex/skills/xdev/ and returns
  // the resolved project root for logging. Unlike launchCoco, we do NOT pass
  // cwd: projectRoot — codex auto-walks parent directories during skill
  // discovery, so the user's original cwd works fine.
  ensureAgentCliOrExit('cdx');
  ensureNpxOrWarn();
  ensureAbAgentCollectEvent();
  applyBundledCredsIfMissing();
  const projectRoot = installToCodex();
  installExternalMarketplacesToAgent(externalMarketplaces, 'cdx');
  injectHooksToCodex();
  console.log(`[xdev] Launching Codex (skills loaded from ${projectRoot}/.codex/skills/xdev/)...`);
  spawnAndExit('codex', ['--dangerously-bypass-approvals-and-sandbox', ...extraArgs]);
}

export function launchTrae(
  extraArgs: string[],
  externalMarketplaces: ExternalMarketplaceInfo[] = [],
  selectedSkills?: string[],
): void {
  installToTrae(selectedSkills);
  installExternalMarketplacesToAgent(externalMarketplaces, 'trae');
  console.log('[xdev] Launching Trae...');
  spawnAndExit('trae', ['.', ...extraArgs]);
}

export function launchTraeCn(
  extraArgs: string[],
  externalMarketplaces: ExternalMarketplaceInfo[] = [],
  selectedSkills?: string[],
): void {
  installToTrae(selectedSkills);
  installExternalMarketplacesToAgent(externalMarketplaces, 'trae-cn');
  console.log('[xdev] Launching Trae CN...');
  spawnAndExit('trae-cn', ['.', ...extraArgs]);
}

export function launchCoco(
  extraArgs: string[],
  externalMarketplaces: ExternalMarketplaceInfo[] = [],
): void {
  // installToCoco resolves the project root, writes .coco/coco.yaml there, and
  // returns the root. We must spawn coco with cwd=root because coco 0.120.16
  // only loads .coco/coco.yaml from the exact cwd (no parent walk-up). Without
  // this, running `xdev --coco` from any subdirectory would silently fail to
  // load the xdev plugin.
  ensureAgentCliOrExit('coco');
  ensureNpxOrWarn();
  ensureAbAgentCollectEvent();
  applyBundledCredsIfMissing();
  const projectRoot = installToCoco();
  installExternalMarketplacesToAgent(externalMarketplaces, 'coco');
  console.log(`[xdev] Launching Coco (TRAE CLI) from project root: ${projectRoot}`);
  spawnAndExit('coco', ['--yolo', ...extraArgs], { cwd: projectRoot });
}
