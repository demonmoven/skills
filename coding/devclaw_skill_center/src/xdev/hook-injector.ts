import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { homedir } from 'node:os';
import { dirname, join } from 'node:path';
import YAML from 'yaml';
import { parse as parseToml, stringify as stringifyToml } from 'smol-toml';
import { findProjectRoot } from './paths.js';
import {
  getCCBundledHookGroups,
  getCocoBundledHooks,
  getCodexBundledHookGroups,
  type ClaudeStyleHookGroup,
  type ClaudeStyleHookMap,
} from './bundled-hooks.js';

function commandExists(cmd: string): boolean {
  return spawnSync('which', [cmd], { stdio: 'ignore' }).status === 0;
}

/**
 * Check that runtime dependencies required by hook commands are available.
 * Logs warnings for anything missing so the user can fix it before hooks fail silently.
 */
function checkHookDeps(): void {
  if (!commandExists('jq')) {
    console.warn(
      '[xdev] Warning: "jq" not found on PATH. Trace hooks require jq to parse JSON input.\n' +
        '       Install it: sudo apt-get install jq (Debian/Ubuntu) or brew install jq (macOS)',
    );
  }
}

export function injectHooksToClaudeCode(projectRoot?: string): number {
  checkHookDeps();
  const root = projectRoot ?? findProjectRoot(process.cwd());
  guardAgainstHome(root, '.claude/settings.json');
  const settingsPath = join(root, '.claude', 'settings.json');
  const added = mergeClaudeStyleHooks(settingsPath, getCCBundledHookGroups());
  if (added > 0) {
    console.log(`[xdev] Injected ${added} hook entr${added === 1 ? 'y' : 'ies'} to ${settingsPath}`);
  } else {
    console.log(`[xdev] Hooks already present in ${settingsPath} (no change)`);
  }
  return added;
}

export function injectHooksToCodex(projectRoot?: string): number {
  checkHookDeps();
  const root = projectRoot ?? findProjectRoot(process.cwd());
  guardAgainstHome(root, '.codex/hooks.json');
  const hooksPath = join(root, '.codex', 'hooks.json');
  const configPath = join(root, '.codex', 'config.toml');
  const added = mergeClaudeStyleHooks(hooksPath, getCodexBundledHookGroups());
  enableCodexHooksFeature(configPath);
  if (added > 0) {
    console.log(`[xdev] Injected ${added} hook entr${added === 1 ? 'y' : 'ies'} to ${hooksPath}`);
  } else {
    console.log(`[xdev] Hooks already present in ${hooksPath} (no change)`);
  }
  return added;
}

export function injectHooksToCoco(doc: YAML.Document): number {
  checkHookDeps();
  let hooksNode = doc.get('hooks');
  if (!YAML.isSeq(hooksNode)) {
    hooksNode = new YAML.YAMLSeq();
    doc.set('hooks', hooksNode);
  }
  const seq = hooksNode as YAML.YAMLSeq;

  // Remove outdated xdev-managed hooks before injecting current versions
  const current = (seq.toJSON() ?? []) as Array<{ type?: string; command?: string }>;
  for (let i = current.length - 1; i >= 0; i--) {
    if (current[i].command && isXdevManagedCommand(current[i].command!)) {
      seq.deleteIn([i]);
    }
  }

  const bundled = getCocoBundledHooks();
  let added = 0;
  for (const h of bundled) {
    const updated = (seq.toJSON() ?? []) as Array<{ type?: string; command?: string }>;
    const dup = updated.some((existing) => existing.type === h.type && existing.command === h.command);
    if (!dup) {
      seq.add(h);
      added++;
    }
  }
  return added;
}

function guardAgainstHome(projectRoot: string, label: string): void {
  if (projectRoot === homedir()) {
    throw new Error(
      `[xdev] Refusing to write ${label} in $HOME (${projectRoot}). ` +
        `Run xdev from inside a project directory.`,
    );
  }
}

/**
 * Signature patterns that identify xdev-managed hook commands.
 * Used to remove outdated versions before injecting current ones,
 * so command string changes (e.g. npx→binary, echo→printf) don't
 * leave duplicate entries.
 */
const XDEV_HOOK_SIGNATURES = [
  'xdev trace session-start',
  'xtrace session-start',
  'xdev trace forward',
  'xtrace forward',
  'ab-agent-collect-event',
];

function isXdevManagedCommand(command: string): boolean {
  return XDEV_HOOK_SIGNATURES.some((sig) => command.includes(sig));
}

/**
 * Remove all hook entries whose command matches an xdev-managed signature.
 * This ensures old versions of xdev-injected hooks are cleaned up before
 * the current versions are merged in, preventing duplicates when command
 * strings change across xdev versions.
 */
function purgeStaleXdevHooks(hooks: Record<string, ClaudeStyleHookGroup[]>): void {
  for (const groups of Object.values(hooks)) {
    for (const group of groups) {
      group.hooks = group.hooks.filter((h) => !isXdevManagedCommand(h.command));
    }
  }
  // Remove empty groups
  for (const [event, groups] of Object.entries(hooks)) {
    hooks[event] = groups.filter((g) => g.hooks.length > 0);
  }
}

function mergeClaudeStyleHooks(filePath: string, bundled: ClaudeStyleHookMap): number {
  let settings: Record<string, unknown> = {};
  if (existsSync(filePath)) {
    const raw = readFileSync(filePath, 'utf-8').trim();
    if (raw.length > 0) {
      try {
        const parsed: unknown = JSON.parse(raw);
        if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
          settings = parsed as Record<string, unknown>;
        }
      } catch (err) {
        throw new Error(
          `[xdev] Failed to parse ${filePath}: ${(err as Error).message}. ` +
            `Refusing to overwrite a malformed file.`,
        );
      }
    }
  }

  const currentHooks = settings.hooks;
  const hooks: Record<string, ClaudeStyleHookGroup[]> =
    currentHooks && typeof currentHooks === 'object' && !Array.isArray(currentHooks)
      ? (currentHooks as Record<string, ClaudeStyleHookGroup[]>)
      : {};

  // Remove outdated xdev-managed hooks before merging current versions
  purgeStaleXdevHooks(hooks);

  let added = 0;
  for (const [event, bundledGroups] of Object.entries(bundled)) {
    if (!Array.isArray(hooks[event])) {
      hooks[event] = [];
    }
    const existingGroups = hooks[event];
    for (const bundledGroup of bundledGroups) {
      for (const entry of bundledGroup.hooks) {
        const sameMatcher = existingGroups.find((g) => (g.matcher ?? '') === (bundledGroup.matcher ?? ''));
        if (sameMatcher) {
          const dup = sameMatcher.hooks.some((h) => h.command === entry.command);
          if (!dup) {
            sameMatcher.hooks.push(entry);
            added++;
          }
        } else {
          existingGroups.push({
            ...(bundledGroup.matcher !== undefined ? { matcher: bundledGroup.matcher } : {}),
            hooks: [entry],
          });
          added++;
        }
      }
    }
  }

  settings.hooks = hooks;
  mkdirSync(dirname(filePath), { recursive: true });
  writeFileSync(filePath, JSON.stringify(settings, null, 2) + '\n');
  return added;
}

function enableCodexHooksFeature(configPath: string): void {
  let config: Record<string, unknown> = {};
  if (existsSync(configPath)) {
    const raw = readFileSync(configPath, 'utf-8');
    try {
      const parsed = parseToml(raw);
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
        config = parsed as Record<string, unknown>;
      }
    } catch (err) {
      console.error(
        `[xdev] Warn: Failed to parse ${configPath}: ${(err as Error).message}. ` +
          `Please set [features] codex_hooks = true manually.`,
      );
      return;
    }
  }

  const currentFeatures = config.features;
  const features: Record<string, unknown> =
    currentFeatures && typeof currentFeatures === 'object' && !Array.isArray(currentFeatures)
      ? (currentFeatures as Record<string, unknown>)
      : {};

  if (features.codex_hooks === true) return;
  features.codex_hooks = true;
  config.features = features;

  mkdirSync(dirname(configPath), { recursive: true });
  writeFileSync(configPath, stringifyToml(config) + '\n');
  console.log(`[xdev] Enabled [features] codex_hooks = true in ${configPath}`);
}
