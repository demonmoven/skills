import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtempSync, rmSync, readFileSync, writeFileSync, mkdirSync, existsSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import YAML from 'yaml';
import { parse as parseToml } from 'smol-toml';
import {
  injectHooksToClaudeCode,
  injectHooksToCodex,
  injectHooksToCoco,
} from './hook-injector.js';
import {
  CODE_STAT_CMD,
  TRACE_FORWARD_CMD,
  TRACE_SESSION_START_CMD,
} from './bundled-hooks.js';

let root: string;

beforeEach(() => {
  root = mkdtempSync(join(tmpdir(), 'xdev-hook-'));
});

afterEach(() => {
  rmSync(root, { recursive: true, force: true });
});

describe('injectHooksToClaudeCode', () => {
  it('creates .claude/settings.json with 4 bundled hook entries on empty project', () => {
    const added = injectHooksToClaudeCode(root);
    expect(added).toBe(4);
    const settingsPath = join(root, '.claude', 'settings.json');
    const settings = JSON.parse(readFileSync(settingsPath, 'utf-8'));
    expect(settings.hooks.SessionStart).toHaveLength(1);
    expect(settings.hooks.SessionStart[0].hooks[0].command).toContain(TRACE_SESSION_START_CMD);
    // Stop has 2 groups with different matchers ('*' for CODE_STAT, '' for TRACE_FORWARD)
    expect(settings.hooks.Stop).toHaveLength(2);
    expect(settings.hooks.PostToolUse).toHaveLength(1);
    expect(settings.hooks.PostToolUse[0].matcher).toBe('Write|Edit|MultiEdit');
  });

  it('is idempotent: second run produces same result, no duplicates', () => {
    injectHooksToClaudeCode(root);
    injectHooksToClaudeCode(root);
    const settings = JSON.parse(readFileSync(join(root, '.claude', 'settings.json'), 'utf-8'));
    // Purge+re-inject means added count is always 4, but no duplicates in output
    expect(settings.hooks.Stop).toHaveLength(2);
    expect(settings.hooks.PostToolUse).toHaveLength(1);
    expect(settings.hooks.SessionStart).toHaveLength(1);
    expect(settings.hooks.SessionStart[0].hooks).toHaveLength(1);
  });

  it('preserves existing unrelated hooks and settings', () => {
    mkdirSync(join(root, '.claude'), { recursive: true });
    writeFileSync(
      join(root, '.claude', 'settings.json'),
      JSON.stringify(
        {
          model: 'opus',
          hooks: {
            SessionStart: [
              { matcher: 'custom', hooks: [{ type: 'command', command: 'my-hook' }] },
            ],
          },
        },
        null,
        2,
      ),
    );
    const added = injectHooksToClaudeCode(root);
    expect(added).toBe(4);
    const settings = JSON.parse(readFileSync(join(root, '.claude', 'settings.json'), 'utf-8'));
    expect(settings.model).toBe('opus');
    expect(settings.hooks.SessionStart).toHaveLength(2);
    const customStill = settings.hooks.SessionStart.find((g: { matcher?: string }) => g.matcher === 'custom');
    expect(customStill).toBeDefined();
  });

  it('merges into existing matcher-equal group rather than creating duplicate', () => {
    mkdirSync(join(root, '.claude'), { recursive: true });
    writeFileSync(
      join(root, '.claude', 'settings.json'),
      JSON.stringify(
        {
          hooks: {
            Stop: [{ matcher: '', hooks: [{ type: 'command', command: 'unrelated-cmd' }] }],
          },
        },
        null,
        2,
      ),
    );
    injectHooksToClaudeCode(root);
    const settings = JSON.parse(readFileSync(join(root, '.claude', 'settings.json'), 'utf-8'));
    // matcher='' group gets unrelated-cmd + TRACE_FORWARD_CMD, matcher='*' group gets CODE_STAT_CMD
    const emptyMatcherGroup = settings.hooks.Stop.find((g: { matcher?: string }) => g.matcher === '');
    expect(emptyMatcherGroup.hooks).toHaveLength(2);
    expect(emptyMatcherGroup.hooks.some((h: { command: string }) => h.command === 'unrelated-cmd')).toBe(true);
    expect(emptyMatcherGroup.hooks.some((h: { command: string }) => h.command === TRACE_FORWARD_CMD)).toBe(true);
    const starMatcherGroup = settings.hooks.Stop.find((g: { matcher?: string }) => g.matcher === '*');
    expect(starMatcherGroup.hooks).toHaveLength(1);
    expect(starMatcherGroup.hooks[0].command).toBe(CODE_STAT_CMD);
  });
});

describe('injectHooksToCodex', () => {
  it('creates .codex/hooks.json and enables codex_hooks feature on empty project', () => {
    const added = injectHooksToCodex(root);
    expect(added).toBe(4);
    const hooks = JSON.parse(readFileSync(join(root, '.codex', 'hooks.json'), 'utf-8'));
    expect(hooks.hooks.SessionStart[0].matcher).toBe('startup|resume');
    expect(hooks.hooks.Stop).toHaveLength(2);
    const config = parseToml(readFileSync(join(root, '.codex', 'config.toml'), 'utf-8')) as {
      features: { codex_hooks: boolean };
    };
    expect(config.features.codex_hooks).toBe(true);
  });

  it('is idempotent: codex_hooks stays true, no duplicate hooks', () => {
    injectHooksToCodex(root);
    injectHooksToCodex(root);
    const hooks = JSON.parse(readFileSync(join(root, '.codex', 'hooks.json'), 'utf-8'));
    expect(hooks.hooks.Stop).toHaveLength(2);
    expect(hooks.hooks.SessionStart).toHaveLength(1);
    expect(hooks.hooks.SessionStart[0].hooks).toHaveLength(1);
  });

  it('preserves unrelated TOML sections when enabling feature flag', () => {
    mkdirSync(join(root, '.codex'), { recursive: true });
    writeFileSync(join(root, '.codex', 'config.toml'), `model = "gpt-5"\n\n[profile.prod]\napproval_policy = "strict"\n`);
    injectHooksToCodex(root);
    const config = parseToml(readFileSync(join(root, '.codex', 'config.toml'), 'utf-8')) as {
      model: string;
      profile: { prod: { approval_policy: string } };
      features: { codex_hooks: boolean };
    };
    expect(config.model).toBe('gpt-5');
    expect(config.profile.prod.approval_policy).toBe('strict');
    expect(config.features.codex_hooks).toBe(true);
  });
});

describe('injectHooksToCoco', () => {
  it('injects exactly 2 trace hooks (no code-stat) into empty doc', () => {
    const doc = new YAML.Document({});
    const added = injectHooksToCoco(doc);
    expect(added).toBe(2);
    const yamlStr = doc.toString();
    const parsed = YAML.parse(yamlStr) as { hooks: Array<{ command: string; matchers: Array<{ event: string }> }> };
    expect(parsed.hooks).toHaveLength(2);
    expect(parsed.hooks[0].matchers[0].event).toBe('session_start');
    expect(parsed.hooks[1].matchers[0].event).toBe('stop');
    expect(yamlStr.includes(CODE_STAT_CMD)).toBe(false);
  });

  it('is idempotent when called twice', () => {
    const doc = new YAML.Document({});
    injectHooksToCoco(doc);
    injectHooksToCoco(doc);
    const parsed = YAML.parse(doc.toString()) as { hooks: unknown[] };
    // Purge+re-inject: always 2 hooks, no duplicates
    expect(parsed.hooks).toHaveLength(2);
  });

  it('appends to existing hooks sequence without disturbing user hooks', () => {
    const doc = YAML.parseDocument(
      `hooks:\n  - type: command\n    command: user-custom-hook\n    matchers:\n      - event: stop\n`,
    );
    injectHooksToCoco(doc);
    const parsed = YAML.parse(doc.toString()) as {
      hooks: Array<{ command: string; matchers: Array<{ event: string }> }>;
    };
    expect(parsed.hooks).toHaveLength(3);
    expect(parsed.hooks[0].command).toBe('user-custom-hook');
    expect(parsed.hooks[1].command).toBe(TRACE_SESSION_START_CMD);
    expect(parsed.hooks[2].command).toBe(TRACE_FORWARD_CMD);
  });
});

describe('hook file creation', () => {
  it('creates parent directories as needed for Claude Code', () => {
    expect(existsSync(join(root, '.claude'))).toBe(false);
    injectHooksToClaudeCode(root);
    expect(existsSync(join(root, '.claude', 'settings.json'))).toBe(true);
  });

  it('creates parent directories as needed for Codex', () => {
    expect(existsSync(join(root, '.codex'))).toBe(false);
    injectHooksToCodex(root);
    expect(existsSync(join(root, '.codex', 'hooks.json'))).toBe(true);
    expect(existsSync(join(root, '.codex', 'config.toml'))).toBe(true);
  });
});
