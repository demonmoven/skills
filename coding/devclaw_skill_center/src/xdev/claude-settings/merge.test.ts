import { describe, expect, it } from 'vitest';
import { mergeClaudeSettings } from './merge.js';
import type { ClaudeSettings } from './types.js';

const tpl: ClaudeSettings = {
  env: { CLAUDE_CODE_EFFORT_LEVEL: 'max' },
  permissions: {
    allow: ['Bash(go test:*)', 'mcp__fetch__fetch'],
    deny: [],
    ask: [],
    defaultMode: 'default',
  },
  model: 'opus[1m]',
  hooks: {
    PostToolUse: [
      {
        matcher: 'Write|Edit|MultiEdit',
        hooks: [{ type: 'command', command: 'echo upload' }],
      },
    ],
    Stop: [{ matcher: '*', hooks: [{ type: 'command', command: 'echo stop' }] }],
  },
  enabledPlugins: { 'ralph-wiggum@claude-code-plugins': true },
  language: '中文',
  alwaysThinkingEnabled: true,
  autoUpdatesChannel: 'latest',
  skipDangerousModePermissionPrompt: true,
};

describe('mergeClaudeSettings', () => {
  it('writes the full template when user is empty', () => {
    const { merged, report } = mergeClaudeSettings({}, tpl);
    expect(report.changed).toBe(true);
    expect(merged.model).toBe('opus[1m]');
    expect(merged.permissions?.allow).toEqual(['Bash(go test:*)', 'mcp__fetch__fetch']);
    expect(merged.skipDangerousModePermissionPrompt).toBe(true);
  });

  it('user-wins for scalar fields by default', () => {
    const user: ClaudeSettings = { model: 'sonnet', language: 'English' };
    const { merged } = mergeClaudeSettings(user, tpl);
    expect(merged.model).toBe('sonnet');
    expect(merged.language).toBe('English');
  });

  it('--prefer-template flips scalar conflicts', () => {
    const user: ClaudeSettings = { model: 'sonnet', language: 'English' };
    const { merged, report } = mergeClaudeSettings(user, tpl, { preferTemplate: true });
    expect(merged.model).toBe('opus[1m]');
    expect(merged.language).toBe('中文');
    expect(report.changes.some((l) => l.includes('model'))).toBe(true);
  });

  it('permissions.allow is union-deduped', () => {
    const user: ClaudeSettings = {
      permissions: { allow: ['mcp__fetch__fetch', 'Bash(ls:*)'] },
    };
    const { merged } = mergeClaudeSettings(user, tpl);
    expect(merged.permissions?.allow?.sort()).toEqual(
      ['Bash(go test:*)', 'Bash(ls:*)', 'mcp__fetch__fetch'].sort(),
    );
  });

  it('enabledPlugins per-key user-wins', () => {
    const user: ClaudeSettings = {
      enabledPlugins: {
        'ralph-wiggum@claude-code-plugins': false,
        'my-plugin@me': true,
      },
    };
    const { merged } = mergeClaudeSettings(user, tpl);
    expect(merged.enabledPlugins?.['ralph-wiggum@claude-code-plugins']).toBe(false);
    expect(merged.enabledPlugins?.['my-plugin@me']).toBe(true);
  });

  it('extraKnownMarketplaces is preserved on user side, never copied from template', () => {
    const user: ClaudeSettings = {
      extraKnownMarketplaces: { mine: { source: { source: 'directory', path: '/x' } } },
    };
    const tplWithExtra = { ...tpl, extraKnownMarketplaces: { other: { source: { source: 'directory', path: '/leak' } } } };
    const { merged } = mergeClaudeSettings(user, tplWithExtra);
    expect(merged.extraKnownMarketplaces).toEqual({ mine: { source: { source: 'directory', path: '/x' } } });
  });

  it('hooks merge by matcher and dedupe by {type, command} signature', () => {
    const user: ClaudeSettings = {
      hooks: {
        PostToolUse: [
          {
            matcher: 'Write|Edit|MultiEdit',
            hooks: [
              { type: 'command', command: 'echo upload' },
              { type: 'command', command: 'echo other' },
            ],
          },
        ],
      },
    };
    const { merged } = mergeClaudeSettings(user, tpl);
    expect(merged.hooks?.PostToolUse).toHaveLength(1);
    expect(merged.hooks?.PostToolUse?.[0].hooks).toHaveLength(2);
    expect(merged.hooks?.Stop).toHaveLength(1);
  });

  it('hooks adds new matcher when user has different matcher only', () => {
    const user: ClaudeSettings = {
      hooks: {
        PostToolUse: [{ matcher: 'Bash', hooks: [{ type: 'command', command: 'echo bash' }] }],
      },
    };
    const { merged } = mergeClaudeSettings(user, tpl);
    expect(merged.hooks?.PostToolUse).toHaveLength(2);
    const matchers = merged.hooks!.PostToolUse!.map((e) => e.matcher).sort();
    expect(matchers).toEqual(['Bash', 'Write|Edit|MultiEdit']);
  });

  it('is idempotent — second merge produces the same merged + no changes', () => {
    const { merged: m1 } = mergeClaudeSettings({}, tpl);
    const { merged: m2, report: r2 } = mergeClaudeSettings(m1, tpl);
    expect(JSON.stringify(m2)).toBe(JSON.stringify(m1));
    expect(r2.changed).toBe(false);
    expect(r2.changes).toEqual([]);
  });

  it('does not mutate the user input object', () => {
    const user: ClaudeSettings = { model: 'sonnet', permissions: { allow: ['x'] } };
    const snapshot = JSON.stringify(user);
    mergeClaudeSettings(user, tpl);
    expect(JSON.stringify(user)).toBe(snapshot);
  });

  it('reports unchanged when template is empty {}', () => {
    const user: ClaudeSettings = { model: 'sonnet' };
    const { merged, report } = mergeClaudeSettings(user, {});
    expect(report.changed).toBe(false);
    expect(merged).toEqual(user);
  });
});
