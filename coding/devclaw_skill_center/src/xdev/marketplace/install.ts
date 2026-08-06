/**
 * Install third-party marketplace plugins into the target Coding Agent.
 *
 * Generalized version of plugin-injector.ts: takes (marketplaceDir,
 * marketplaceName, pluginName) instead of using the hardcoded built-in
 * xdev marketplace constants. Used for the user-selected external
 * marketplaces from `~/.xdev/marketplaces/<name>/`.
 *
 * Mirrors the four agent-specific install strategies of plugin-injector.ts:
 *
 *   - CC:    register marketplace + install plugin (project scope)
 *   - Codex: write <project-root>/.codex/skills/<plugin>/{.codex-plugin/, skills/}
 *   - Coco:  upsert into <project-root>/.coco/coco.yaml
 *   - Trae:  cp into <cwd>/.trae/skills/<plugin>-<skill>/  (no plugin namespace)
 *
 * IMPORTANT: this file does NOT replace plugin-injector.ts. The built-in
 * xdev plugin still uses the (parameter-less) functions in plugin-injector.ts
 * to keep the hot-path identical and zero-regression. This file is only
 * called for third-party marketplaces selected at startup.
 */

import { execFileSync } from 'node:child_process';
import {
  cpSync,
  existsSync,
  mkdirSync,
  readFileSync,
  readdirSync,
  rmSync,
  writeFileSync,
} from 'node:fs';
import { homedir } from 'node:os';
import { join } from 'node:path';
import YAML from 'yaml';
import { findProjectRoot } from '../paths.js';
import { rewriteFrontmatterName } from '../plugin-injector.js';
import type { ExternalMarketplaceInfo, ExternalPluginInfo } from './types.js';

/**
 * Top-level entry: install all selected external marketplaces' plugins
 * into the given agent.
 *
 * Plugin-name conflict detection is the caller's responsibility (see
 * detectPluginNameConflicts in prompt-selection.ts) — by the time we
 * reach here, the input is conflict-free.
 */
export function installExternalMarketplacesToAgent(
  selected: ExternalMarketplaceInfo[],
  agent: 'cc' | 'cdx' | 'coco' | 'trae' | 'trae-cn',
): void {
  if (selected.length === 0) return;

  for (const mkp of selected) {
    for (const plugin of mkp.plugins) {
      switch (agent) {
        case 'cc':
          installExternalPluginToClaudeCode(mkp, plugin);
          break;
        case 'cdx':
          installExternalPluginToCodex(mkp, plugin);
          break;
        case 'coco':
          installExternalPluginToCoco(mkp, plugin);
          break;
        case 'trae':
        case 'trae-cn':
          installExternalPluginToTrae(mkp, plugin);
          break;
      }
    }
  }
}

/* ─────────────────── Claude Code ─────────────────── */

function installExternalPluginToClaudeCode(
  mkp: ExternalMarketplaceInfo,
  plugin: ExternalPluginInfo,
): void {
  console.log(`[xdev] [external] Registering CC marketplace '${mkp.name}'...`);
  try {
    execFileSync('claude', ['plugin', 'marketplace', 'add', mkp.path], {
      stdio: ['ignore', 'inherit', 'pipe'],
    });
  } catch {
    // already added; non-fatal
  }

  console.log(`[xdev] [external] Installing plugin '${plugin.name}@${mkp.name}' to project scope...`);
  execFileSync(
    'claude',
    ['plugin', 'install', `${plugin.name}@${mkp.name}`, '--scope', 'project'],
    { stdio: 'inherit' },
  );
}

/* ─────────────────── Codex ─────────────────── */

function installExternalPluginToCodex(
  mkp: ExternalMarketplaceInfo,
  plugin: ExternalPluginInfo,
): void {
  const projectRoot = findProjectRoot(process.cwd());
  if (projectRoot === homedir()) {
    throw new Error(
      `[xdev] Refusing to write external plugin to .codex/skills in $HOME (${projectRoot}).`,
    );
  }

  const pluginSkillsSource = join(plugin.path, 'skills');
  const codexManifestSource = join(plugin.path, '.codex-plugin', 'plugin.json');

  // Form D: <root>/.codex/skills/<plugin-name>/{.codex-plugin/, skills/}
  // Note: per-plugin namespace (uses plugin.name, not mkp.name).
  const targetPluginDir = join(projectRoot, '.codex', 'skills', plugin.name);
  const targetManifestDir = join(targetPluginDir, '.codex-plugin');
  const targetManifestPath = join(targetManifestDir, 'plugin.json');
  const targetSkillsDir = join(targetPluginDir, 'skills');

  console.log(`[xdev] [external] Installing '${plugin.name}' to Codex (project scope)...`);
  rmSync(targetPluginDir, { recursive: true, force: true });
  mkdirSync(targetManifestDir, { recursive: true });
  mkdirSync(targetSkillsDir, { recursive: true });

  if (existsSync(codexManifestSource)) {
    cpSync(codexManifestSource, targetManifestPath);
  } else {
    // Fallback: synthesize a minimal Codex manifest from the plugin name.
    // This lets third-party plugins that only ship .claude-plugin/plugin.json
    // still load on Codex (with plugin-name namespace).
    const fallback = {
      name: plugin.name,
      version: '0.0.1',
      description: `Auto-generated Codex manifest for ${plugin.name} (from ${mkp.name})`,
      skills: './skills/',
      interface: { displayName: plugin.name, category: 'External' },
    };
    writeFileSync(targetManifestPath, JSON.stringify(fallback, null, 2));
  }

  if (!existsSync(pluginSkillsSource)) {
    console.warn(`[xdev] [external] Plugin '${plugin.name}' has no skills/ directory; nothing to copy`);
    return;
  }

  let installed = 0;
  for (const entry of readdirSync(pluginSkillsSource, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    cpSync(
      join(pluginSkillsSource, entry.name),
      join(targetSkillsDir, entry.name),
      { recursive: true, dereference: true },
    );
    installed++;
  }
  console.log(`[xdev] [external] Installed ${installed} skill(s) for '${plugin.name}' to ${targetSkillsDir}`);
}

/* ─────────────────── Coco / TRAE CLI ─────────────────── */

function installExternalPluginToCoco(
  mkp: ExternalMarketplaceInfo,
  plugin: ExternalPluginInfo,
): void {
  const projectRoot = findProjectRoot(process.cwd());
  if (projectRoot === homedir()) {
    throw new Error(
      `[xdev] Refusing to write external plugin to .coco/coco.yaml in $HOME (${projectRoot}).`,
    );
  }

  const cocoDir = join(projectRoot, '.coco');
  const cocoYamlPath = join(cocoDir, 'coco.yaml');
  mkdirSync(cocoDir, { recursive: true });

  let doc: YAML.Document;
  if (existsSync(cocoYamlPath)) {
    const src = readFileSync(cocoYamlPath, 'utf-8');
    doc = YAML.parseDocument(src);
    if (doc.errors.length > 0) {
      throw new Error(
        `[xdev] Failed to parse existing ${cocoYamlPath}: ${doc.errors[0].message}`,
      );
    }
  } else {
    doc = new YAML.Document({});
  }

  upsertNamedItem(doc, 'marketplaces', {
    enabled: true,
    name: mkp.name,
    source: mkp.path,
    type: 'local',
  });
  upsertNamedItem(doc, 'plugins', {
    enabled: true,
    marketplace: mkp.name,
    name: plugin.name,
    source: plugin.name,
    type: 'marketplace',
  });

  writeFileSync(cocoYamlPath, doc.toString());
  console.log(`[xdev] [external] Wrote external plugin '${plugin.name}@${mkp.name}' to ${cocoYamlPath}`);
}

/**
 * Local copy of plugin-injector.ts:upsertNamedItem to avoid an import cycle.
 * If you change one, change the other.
 */
function upsertNamedItem(
  doc: YAML.Document,
  key: string,
  item: Record<string, unknown>,
): void {
  const existing = doc.get(key);
  if (!YAML.isSeq(existing)) {
    doc.set(key, [item]);
    return;
  }
  const seq = existing;
  const idx = seq.items.findIndex((node) => {
    return YAML.isMap(node) && (node as YAML.YAMLMap).get('name') === item.name;
  });
  if (idx >= 0) {
    seq.set(idx, item);
  } else {
    seq.add(item);
  }
}

/* ─────────────────── Trae IDE / Trae CN IDE ─────────────────── */

function installExternalPluginToTrae(
  mkp: ExternalMarketplaceInfo,
  plugin: ExternalPluginInfo,
): void {
  // Trae has no plugin mechanism — we cp each skill directly under
  // <cwd>/.trae/skills/<plugin>-<skill>/ to avoid colliding with built-in
  // xdev's `xdev-` prefix or the user's own skills. The frontmatter `name:`
  // is rewritten to match the new directory name.
  const pluginSkillsSource = join(plugin.path, 'skills');
  if (!existsSync(pluginSkillsSource)) {
    console.warn(`[xdev] [external] Plugin '${plugin.name}' has no skills/ directory; nothing to copy`);
    return;
  }

  const targetSkillsDir = join(process.cwd(), '.trae', 'skills');
  mkdirSync(targetSkillsDir, { recursive: true });

  // Per-plugin prefix so external plugins don't collide with built-in
  // `xdev-*` skills nor with each other across marketplaces.
  const prefix = `${plugin.name}-`;

  // Wipe any prior copies of THIS plugin (idempotent re-run).
  for (const entry of readdirSync(targetSkillsDir, { withFileTypes: true })) {
    if (entry.isDirectory() && entry.name.startsWith(prefix)) {
      rmSync(join(targetSkillsDir, entry.name), { recursive: true, force: true });
    }
  }

  let installed = 0;
  for (const entry of readdirSync(pluginSkillsSource, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    const srcName = entry.name;
    const dstName = `${prefix}${srcName}`;
    const srcDir = join(pluginSkillsSource, srcName);
    const dstDir = join(targetSkillsDir, dstName);

    cpSync(srcDir, dstDir, { recursive: true, dereference: true });

    const skillMdPath = join(dstDir, 'SKILL.md');
    if (existsSync(skillMdPath)) {
      const original = readFileSync(skillMdPath, 'utf-8');
      const rewritten = rewriteFrontmatterName(original, srcName, dstName);
      if (rewritten !== original) {
        writeFileSync(skillMdPath, rewritten);
      }
    }
    installed++;
  }

  console.log(`[xdev] [external] Installed ${installed} skill(s) for '${plugin.name}' to ${targetSkillsDir} (prefix: ${prefix})`);
}
