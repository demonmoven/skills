import { execFileSync } from 'node:child_process';
import {
  mkdirSync,
  rmSync,
  cpSync,
  existsSync,
  readFileSync,
  writeFileSync,
  readdirSync,
} from 'node:fs';
import { homedir } from 'node:os';
import { join } from 'node:path';
import YAML from 'yaml';
import {
  MARKETPLACE_DIR,
  MARKETPLACE_NAME,
  PLUGIN_NAME,
  PLUGIN_SRC_DIR,
  CLAUDE_PLUGIN_CACHE,
  findProjectRoot,
} from './paths.js';
import { injectHooksToCoco } from './hook-injector.js';

const TRAE_SKILL_PREFIX = 'xdev-';

/**
 * Install xdev plugin into Claude Code at project scope.
 * Refreshes the cache directory and re-registers the marketplace each call.
 */
export function installToClaudeCode(): void {
  console.log('[xdev] Refreshing Claude Code plugin cache...');
  rmSync(CLAUDE_PLUGIN_CACHE, { recursive: true, force: true });

  console.log('[xdev] Registering marketplace...');
  try {
    execFileSync('claude', ['plugin', 'marketplace', 'add', MARKETPLACE_DIR], {
      stdio: ['ignore', 'inherit', 'pipe'],
    });
  } catch {
    // Marketplace already added; non-fatal.
  }

  console.log('[xdev] Installing plugin to project scope...');
  execFileSync(
    'claude',
    ['plugin', 'install', `${PLUGIN_NAME}@${MARKETPLACE_NAME}`, '--scope', 'project'],
    { stdio: 'inherit' },
  );
}

/**
 * Install xdev as a real Codex plugin at PROJECT scope.
 *
 * Codex namespace mechanism (CRITICAL — empirically verified, codex 0.118.0):
 *
 * The `<plugin>:<skill>` namespace prefix shown in codex's `list skills` is
 * NOT produced by directory nesting alone. It is produced ONLY when codex
 * sees a `.codex-plugin/plugin.json` manifest in the directory — codex then
 * recognizes the directory as a plugin and prefixes all skills under
 * `<plugin-dir>/skills/<sub>/SKILL.md` with the plugin name.
 *
 * 4-form comparison test (/tmp/codex-ns-final, 2026-04-09):
 *   A. <root>/.codex/skills/<dir>/<sub>/SKILL.md (no manifest)   -> bare name (broken)
 *   B. <root>/.agents/plugins/<name>/.codex-plugin + skills/<sub> -> NOT loaded by codex
 *   C. <root>/plugins/<name> via .agents/plugins/marketplace.json -> NOT loaded by codex
 *   D. <root>/.codex/skills/<name>/.codex-plugin + skills/<sub>   -> 'xdev:<sub>' ✅
 *
 * Form D is the only path that produces a project-scoped namespaced plugin.
 * Forms B/C — the .agents/plugins/marketplace.json paths recommended by
 * OpenAI's official docs and the plugin-creator system skill — DO NOT WORK
 * in codex 0.118.0. They appear to be either feature-request stage or only
 * implemented for the user-level ~/.agents/plugins/marketplace.json.
 *
 * See feat-dev/2026/04/08/001902-codex-namespace-not-working-investigation/research.md
 * for the full evidence chain. This replaces the (incorrect) approach from
 * commit 16c164a which was based on a misread experiment in /tmp/ns-test/.
 *
 * Output structure:
 *   <project-root>/.codex/skills/xdev/
 *   ├── .codex-plugin/
 *   │   └── plugin.json          # copied from marketplace, triggers namespace
 *   └── skills/                   # extra nesting layer required by plugin spec
 *       ├── exec-plan/SKILL.md   # → 'xdev:exec-plan' in codex
 *       ├── speckit/SKILL.md     # → 'xdev:speckit'
 *       └── ... (18 skills)
 *
 * Idempotency: each invocation rebuilds only the `xdev/` subdirectory of
 * .codex/skills/, leaving other plugins and the user's own non-xdev skills
 * under .codex/skills/ untouched.
 *
 * Refuses to write into $HOME for the same reason as installToCoco.
 * Returns the project root path so the launcher can log it; codex itself
 * does NOT need cwd to be set to the project root because codex walks parent
 * directories during skill/plugin discovery.
 *
 * Future extension points (not implemented this round; the .codex/ directory
 * layer supports them via separate files outside the xdev plugin namespace):
 *   - Hooks: <project-root>/.codex/hooks.json (codex loads & merges multiple)
 *   - MCP servers: <project-root>/.codex/config.toml [mcp_servers] section
 *   - Trust note: hooks and mcp_servers require the project to be trusted
 *     by the user on first codex entry; plugin skills do not require trust.
 */
export function installToCodex(): string {
  const projectRoot = findProjectRoot(process.cwd());
  if (projectRoot === homedir()) {
    throw new Error(
      `[xdev] Refusing to write .codex/skills in $HOME (${projectRoot}). ` +
        `Run xdev from inside a project directory.`,
    );
  }

  const sourceSkillsDir = join(PLUGIN_SRC_DIR, 'skills');
  const sourceManifest = join(PLUGIN_SRC_DIR, '.codex-plugin', 'plugin.json');

  // Form D layout: <root>/.codex/skills/xdev/{.codex-plugin/, skills/}
  const xdevPluginDir = join(projectRoot, '.codex', 'skills', 'xdev');
  const xdevManifestDir = join(xdevPluginDir, '.codex-plugin');
  const xdevManifestPath = join(xdevManifestDir, 'plugin.json');
  const xdevSkillsDir = join(xdevPluginDir, 'skills');

  console.log('[xdev] Installing xdev plugin to Codex (project scope)...');
  // Rebuild the entire xdev plugin dir for idempotency. This intentionally
  // does NOT touch other plugins or the user's own skills under .codex/skills/.
  rmSync(xdevPluginDir, { recursive: true, force: true });
  mkdirSync(xdevManifestDir, { recursive: true });
  mkdirSync(xdevSkillsDir, { recursive: true });

  // 1. Copy the plugin manifest — this is the trigger that makes codex
  //    recognize the directory as a plugin and apply the 'xdev:' namespace.
  //    Without this file, every skill under skills/ would load as a flat
  //    personal skill (forms A/B/C above) and the namespace would be lost.
  cpSync(sourceManifest, xdevManifestPath);

  // 2. Copy each skill into xdev/skills/<sub>/ — note the extra skills/
  //    subdirectory level (matches the `skills: "./skills/"` declaration in
  //    plugin.json and the layout used by codex's bundled plugins like
  //    ~/.codex/.tmp/plugins/plugins/linear/).
  let installed = 0;
  for (const entry of readdirSync(sourceSkillsDir, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    cpSync(
      join(sourceSkillsDir, entry.name),
      join(xdevSkillsDir, entry.name),
      { recursive: true, dereference: true },
    );
    installed++;
  }

  console.log(`[xdev] Installed ${installed} skill(s) to ${xdevSkillsDir}`);
  console.log(`[xdev] Plugin manifest: ${xdevManifestPath}`);
  console.log(`[xdev] Skills will be loaded as 'xdev:<skill-name>' in codex`);
  return projectRoot;
}

/**
 * Install xdev plugin into Coco / TRAE CLI at PROJECT scope.
 *
 * Coco's plugin protocol is compatible with Claude Code at the file format
 * level (the existing marketplace/.claude-plugin/marketplace.json and
 * plugins/xdev/.claude-plugin/plugin.json work as-is), but coco lacks an
 * equivalent of `claude plugin install --scope project`. We instead write the
 * project-level config directly: `.coco/coco.yaml` under the project root.
 *
 * Two non-obvious constraints from coco 0.120.16:
 *   1. Project config is loaded ONLY when cwd matches the directory containing
 *      `.coco/coco.yaml` — coco does NOT walk up parent dirs. Therefore the
 *      caller (launcher) must spawn coco with cwd set to the returned project
 *      root, otherwise running `xdev --coco` from a subdirectory would silently
 *      fail to load the plugin.
 *   2. The directory/filename pair must be `.coco/coco.yaml` (not
 *      `.coco/traecli.yaml` or `.trae/coco.yaml`). Confirmed by oncall: coco
 *      and trae configs have not yet been merged.
 *
 * Merge contract — never clobber the user's existing config:
 *   - Existing `.coco/coco.yaml` is parsed via `yaml.parseDocument` which
 *     preserves comments, key order, and indentation across the round-trip.
 *   - The xdev marketplace and plugin entries are upserted by `name`, leaving
 *     unrelated marketplaces/plugins/model/permission_mode/etc untouched.
 *   - If the YAML is malformed, we throw with a clear error rather than
 *     overwriting. The user's file is sacred.
 *
 * Refuses to write into $HOME (which would happen if the user runs xdev from
 * their home directory and has no git repo there) — that would silently leak
 * the xdev config into every shell session under coco.
 *
 * Returns the project root path so the launcher can spawn coco with cwd=root.
 */
export function installToCoco(): string {
  const projectRoot = findProjectRoot(process.cwd());
  if (projectRoot === homedir()) {
    throw new Error(
      `[xdev] Refusing to write .coco/coco.yaml in $HOME (${projectRoot}). ` +
        `Run xdev from inside a project directory.`,
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
        `[xdev] Failed to parse existing ${cocoYamlPath}: ${doc.errors[0].message}. ` +
          `Refusing to overwrite a malformed file — please fix it manually.`,
      );
    }
  } else {
    doc = new YAML.Document({});
  }

  upsertNamedItem(doc, 'marketplaces', {
    enabled: true,
    name: MARKETPLACE_NAME,
    source: MARKETPLACE_DIR,
    type: 'local',
  });
  upsertNamedItem(doc, 'plugins', {
    enabled: true,
    marketplace: MARKETPLACE_NAME,
    name: PLUGIN_NAME,
    source: PLUGIN_NAME,
    type: 'marketplace',
  });

  const addedHooks = injectHooksToCoco(doc);

  writeFileSync(cocoYamlPath, doc.toString());
  console.log(`[xdev] Wrote project Coco config: ${cocoYamlPath}`);
  if (addedHooks > 0) {
    console.log(`[xdev] Injected ${addedHooks} trace hook(s) into coco.yaml`);
  }
  return projectRoot;
}

/**
 * Upsert (insert or replace) a named map item inside a top-level sequence.
 *
 * Used to merge xdev's marketplace/plugin entries into the user's coco.yaml
 * without disturbing other entries. Lookup is by the `name` field; if no entry
 * with the same name exists, the new item is appended; otherwise the existing
 * entry is replaced in-place (keeping its position in the sequence).
 *
 * If the top-level key doesn't exist or isn't a sequence, we create a new
 * sequence containing just this item.
 */
function upsertNamedItem(doc: YAML.Document, key: string, item: Record<string, unknown>): void {
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

/**
 * Install xdev skills into Trae / Trae CN at project scope.
 *
 * Trae and Trae CN don't support a plugin mechanism, so we copy each skill
 * directly under <cwd>/.trae/skills/, prefixed with `xdev-` to isolate from
 * the user's own skills and to enable easy cleanup. The SKILL.md frontmatter
 * `name:` field is rewritten to match the new directory name (Trae loaders
 * generally require directory and frontmatter name to agree).
 *
 * Idempotent: each invocation first wipes any existing `xdev-*` directories
 * (so removed-upstream skills don't linger) and then re-copies. The user's
 * own non-prefixed skills under .trae/skills/ are left untouched.
 *
 * Same install for both Trae and Trae CN — the only difference is which
 * binary the launcher spawns.
 */
export function installToTrae(selectedSkills?: string[]): void {
  const sourceSkillsDir = join(PLUGIN_SRC_DIR, 'skills');
  const targetSkillsDir = join(process.cwd(), '.trae', 'skills');

  console.log('[xdev] Installing xdev skills to Trae...');
  mkdirSync(targetSkillsDir, { recursive: true });

  for (const entry of readdirSync(targetSkillsDir, { withFileTypes: true })) {
    if (entry.isDirectory() && entry.name.startsWith(TRAE_SKILL_PREFIX)) {
      rmSync(join(targetSkillsDir, entry.name), { recursive: true, force: true });
    }
  }

  const allSourceEntries = readdirSync(sourceSkillsDir, { withFileTypes: true }).filter(
    (e) => e.isDirectory(),
  );
  const total = allSourceEntries.length;

  let installed = 0;
  for (const entry of allSourceEntries) {
    const srcName = entry.name;
    if (selectedSkills && !selectedSkills.includes(srcName)) continue;

    const dstName = `${TRAE_SKILL_PREFIX}${srcName}`;
    const srcDir = join(sourceSkillsDir, srcName);
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

  console.log(`[xdev] Installed ${installed}/${total} skill(s) to ${targetSkillsDir} (prefix: ${TRAE_SKILL_PREFIX})`);
}

/**
 * Replace the `name:` field inside the frontmatter block of a SKILL.md.
 *
 * Anchored to the leading `---` ... `---` block so we don't accidentally rewrite
 * occurrences of the skill name in description / body text. Tolerates plain,
 * single-quoted and double-quoted name values.
 *
 * Returns the original content unchanged if there's no frontmatter or no
 * matching name line.
 */
export function rewriteFrontmatterName(content: string, oldName: string, newName: string): string {
  const fmMatch = content.match(/^---\r?\n([\s\S]*?)\r?\n---/);
  if (!fmMatch) return content;
  const fm = fmMatch[1];
  const escaped = oldName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const nameLineRe = new RegExp(`^name:[ \\t]*["']?${escaped}["']?[ \\t]*$`, 'm');
  if (!nameLineRe.test(fm)) return content;
  const newFm = fm.replace(nameLineRe, `name: ${newName}`);
  return content.replace(fm, newFm);
}
