/**
 * Multi-select prompt at xdev startup, plus plugin-name conflict detection.
 *
 * Called from `runLaunch()` (src/xdev/launch.ts) AFTER agent selection but
 * BEFORE the install step. The flow is:
 *
 *   1. scanExternalMarketplaces() → list
 *   2. if list non-empty:
 *        a. promptExternalMarketplaceSelection(list) → user's selection
 *        b. detectPluginNameConflicts(selection) → throw if any
 *   3. caller installs the chosen plugins to the target agent
 *
 * The TTY check ensures CI runs (process.stdin.isTTY === false) auto-skip
 * the prompt and install nothing extra. Pass --no-external from the launch
 * options to force-skip even on a TTY.
 */

import { createInterface } from 'node:readline/promises';
import type { ExternalMarketplaceInfo, PluginNameConflict } from './types.js';

/**
 * Show a numbered list of marketplaces and ask the user which to load.
 *
 * Returns the chosen subset (possibly empty for "n" / quit).
 *
 * Recognized inputs:
 *   "1,3"   — pick by index, comma-separated
 *   "a"     — all
 *   "n"     — none (skip external)
 *   "q"     — quit (treat as none)
 *   ""      — none (default on empty input)
 *
 * In non-TTY mode, returns [] without prompting.
 */
export async function promptExternalMarketplaceSelection(
  available: ExternalMarketplaceInfo[],
): Promise<ExternalMarketplaceInfo[]> {
  if (available.length === 0) return [];
  if (!process.stdin.isTTY) return [];

  console.log('');
  console.log(`[xdev] Detected ${available.length} third-party marketplace(s) under ~/.xdev/marketplaces/:`);
  available.forEach((m, i) => {
    console.log(`  ${i + 1}. ${m.name}  (${m.pluginCount} plugin / ${m.skillCount} skill)`);
  });
  console.log('');
  console.log('Select which to load (alongside the built-in xdev plugin):');
  console.log('  comma-separated indices (e.g. "1,3")');
  console.log('  a → all');
  console.log('  n → none (load only built-in xdev)');
  console.log('');

  const rl = createInterface({ input: process.stdin, output: process.stdout });
  let raw: string;
  try {
    raw = (await rl.question('> ')).trim().toLowerCase();
  } finally {
    rl.close();
  }

  if (raw === '' || raw === 'n' || raw === 'q') return [];
  if (raw === 'a') return [...available];

  const indices = parseIndexList(raw, available.length);
  return indices.map((i) => available[i]);
}

/**
 * Parse "1,3,5" into [0, 2, 4]. Silently ignores invalid / out-of-range
 * tokens to be lenient on user input.
 */
export function parseIndexList(raw: string, max: number): number[] {
  const result: number[] = [];
  for (const token of raw.split(',')) {
    const trimmed = token.trim();
    if (!trimmed) continue;
    const n = Number.parseInt(trimmed, 10);
    if (Number.isFinite(n) && n >= 1 && n <= max) {
      const zeroBased = n - 1;
      if (!result.includes(zeroBased)) {
        result.push(zeroBased);
      }
    }
  }
  return result;
}

/**
 * Detect plugin-name collisions across the user-selected marketplaces.
 *
 * Returns one PluginNameConflict per duplicate plugin name. Empty array
 * means no conflicts (safe to proceed with installation).
 *
 * Note: this only checks WITHIN the user's selected set. The built-in
 * `xdev` plugin name is not checked here — if a third-party marketplace
 * also has a plugin called `xdev`, that conflict is detected by the
 * launcher's combined-plugin-list check (caller is responsible).
 */
export function detectPluginNameConflicts(
  selected: ExternalMarketplaceInfo[],
): PluginNameConflict[] {
  // pluginName → list of (marketplaceName, pluginPath)
  const seen = new Map<string, Array<{ marketplaceName: string; pluginPath: string }>>();

  for (const mkp of selected) {
    for (const plugin of mkp.plugins) {
      const arr = seen.get(plugin.name) ?? [];
      arr.push({ marketplaceName: mkp.name, pluginPath: plugin.path });
      seen.set(plugin.name, arr);
    }
  }

  const conflicts: PluginNameConflict[] = [];
  for (const [name, occurrences] of seen) {
    if (occurrences.length > 1) {
      conflicts.push({ pluginName: name, occurrences });
    }
  }

  return conflicts;
}

/**
 * Format a list of conflicts as a human-readable error message.
 *
 * Used by the launcher to construct the final error before exit(1).
 */
export function formatConflictError(conflicts: PluginNameConflict[]): string {
  const lines: string[] = ['❌ Plugin name conflict(s) detected across selected marketplaces:', ''];
  for (const c of conflicts) {
    lines.push(`  '${c.pluginName}' exists in:`);
    for (const o of c.occurrences) {
      lines.push(`    - ${o.marketplaceName}  (${o.pluginPath})`);
    }
    lines.push('');
  }
  lines.push(
    'Please rename the conflicting plugin(s) in their source repo, then re-run:',
    '  xdev marketplace add-update <git-url> --force',
  );
  return lines.join('\n');
}
