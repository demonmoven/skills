import type {
  ClaudeSettings,
  HookEntry,
  MergeOptions,
  MergeReport,
  PermissionsBlock,
} from './types.js';

const SCALAR_FIELDS = [
  'model',
  'language',
  'alwaysThinkingEnabled',
  'autoUpdatesChannel',
  'skipDangerousModePermissionPrompt',
] as const;

const PERMISSION_ARRAYS = ['allow', 'deny', 'ask'] as const;

/**
 * Increment-merge `template` into `user` and return a merged copy along with
 * a human-readable change report. The original `user` object is NOT mutated.
 *
 * Per-field semantics (see plan.md §2):
 *   - extraKnownMarketplaces:    NEVER touched (skipped — host-absolute path)
 *   - env / enabledPlugins:      per-key user-wins (preferTemplate flips)
 *   - permissions.allow/deny/ask: union dedupe
 *   - permissions.defaultMode + scalar fields: user-wins (preferTemplate flips)
 *   - hooks:                     per event, group by matcher; inner hooks
 *                                deduped by `${type}::${command}` signature
 *   - unknown top-level keys:    user value wins; template adds missing
 *
 * Idempotent: running merge(merged, template) again produces the same merged.
 */
export function mergeClaudeSettings(
  user: ClaudeSettings,
  template: ClaudeSettings,
  opts: MergeOptions = {},
): { merged: ClaudeSettings; report: MergeReport } {
  const merged: ClaudeSettings = deepClone(user);
  const report: MergeReport = { changes: [], changed: false };
  const preferTemplate = opts.preferTemplate ?? false;

  // env — per-key
  if (template.env) {
    merged.env ??= {};
    for (const [k, v] of Object.entries(template.env)) {
      mergeRecordKey(merged.env, k, v, `env.${k}`, preferTemplate, report);
    }
  }

  // permissions
  if (template.permissions) {
    merged.permissions ??= {};
    mergePermissions(merged.permissions, template.permissions, preferTemplate, report);
  }

  // scalar top-level
  for (const field of SCALAR_FIELDS) {
    const tplVal = template[field];
    if (tplVal === undefined) continue;
    mergeScalarKey(merged, field, tplVal, field, preferTemplate, report);
  }

  // hooks
  if (template.hooks) {
    merged.hooks ??= {};
    mergeHooks(merged.hooks, template.hooks, report);
  }

  // enabledPlugins — per-key
  if (template.enabledPlugins) {
    merged.enabledPlugins ??= {};
    for (const [k, v] of Object.entries(template.enabledPlugins)) {
      mergeRecordKey(merged.enabledPlugins, k, v, `enabledPlugins.${k}`, preferTemplate, report);
    }
  }

  // extraKnownMarketplaces — intentionally skipped (host-absolute paths)

  // unknown top-level fields — additive, never overwrite
  for (const [k, v] of Object.entries(template)) {
    if (k in merged) continue;
    if (
      k === 'env' || k === 'permissions' || k === 'hooks' ||
      k === 'enabledPlugins' || k === 'extraKnownMarketplaces' ||
      (SCALAR_FIELDS as readonly string[]).includes(k)
    ) continue;
    merged[k] = v;
    report.changes.push(`+ ${k}（unknown 字段，从模板透传）`);
    report.changed = true;
  }

  return { merged, report };
}

function deepClone<T>(v: T): T {
  return JSON.parse(JSON.stringify(v)) as T;
}

function mergeRecordKey<T>(
  target: Record<string, T>,
  key: string,
  tplVal: T,
  path: string,
  preferTemplate: boolean,
  report: MergeReport,
): void {
  if (target[key] === undefined) {
    target[key] = tplVal;
    report.changes.push(`+ ${path} = ${formatVal(tplVal)}`);
    report.changed = true;
    return;
  }
  if (preferTemplate && !deepEqual(target[key], tplVal)) {
    report.changes.push(`~ ${path}: ${formatVal(target[key])} → ${formatVal(tplVal)}`);
    target[key] = tplVal;
    report.changed = true;
  }
}

function mergeScalarKey(
  target: ClaudeSettings,
  key: string,
  tplVal: unknown,
  path: string,
  preferTemplate: boolean,
  report: MergeReport,
): void {
  if (target[key] === undefined) {
    target[key] = tplVal;
    report.changes.push(`+ ${path} = ${formatVal(tplVal)}`);
    report.changed = true;
    return;
  }
  if (preferTemplate && !deepEqual(target[key], tplVal)) {
    report.changes.push(`~ ${path}: ${formatVal(target[key])} → ${formatVal(tplVal)}`);
    target[key] = tplVal;
    report.changed = true;
  }
}

function mergePermissions(
  user: PermissionsBlock,
  template: PermissionsBlock,
  preferTemplate: boolean,
  report: MergeReport,
): void {
  for (const arr of PERMISSION_ARRAYS) {
    const tplArr = template[arr];
    if (!Array.isArray(tplArr) || tplArr.length === 0) continue;
    user[arr] ??= [];
    const seen = new Set(user[arr]);
    let added = 0;
    for (const item of tplArr) {
      if (seen.has(item)) continue;
      user[arr]!.push(item);
      seen.add(item);
      added++;
    }
    if (added > 0) {
      report.changes.push(`+ permissions.${arr}: ${added} 项新增`);
      report.changed = true;
    }
  }

  if (template.defaultMode !== undefined) {
    if (user.defaultMode === undefined) {
      user.defaultMode = template.defaultMode;
      report.changes.push(`+ permissions.defaultMode = ${formatVal(template.defaultMode)}`);
      report.changed = true;
    } else if (preferTemplate && user.defaultMode !== template.defaultMode) {
      report.changes.push(
        `~ permissions.defaultMode: ${formatVal(user.defaultMode)} → ${formatVal(template.defaultMode)}`,
      );
      user.defaultMode = template.defaultMode;
      report.changed = true;
    }
  }
}

function mergeHooks(
  user: Record<string, HookEntry[]>,
  template: Record<string, HookEntry[]>,
  report: MergeReport,
): void {
  for (const [event, tplEntries] of Object.entries(template)) {
    if (!Array.isArray(tplEntries)) continue;
    user[event] ??= [];
    let added = 0;
    for (const tplEntry of tplEntries) {
      let userEntry = user[event].find((e) => e.matcher === tplEntry.matcher);
      if (!userEntry) {
        userEntry = { matcher: tplEntry.matcher, hooks: [] };
        user[event].push(userEntry);
      }
      const sigs = new Set(userEntry.hooks.map(hookSig));
      for (const h of tplEntry.hooks) {
        const sig = hookSig(h);
        if (sigs.has(sig)) continue;
        userEntry.hooks.push(h);
        sigs.add(sig);
        added++;
      }
    }
    if (added > 0) {
      report.changes.push(`+ hooks.${event}: ${added} 个 hook 新增`);
      report.changed = true;
    }
  }
}

function hookSig(h: { type: string; command: string }): string {
  return `${h.type}::${h.command}`;
}

function deepEqual(a: unknown, b: unknown): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}

function formatVal(v: unknown): string {
  if (typeof v === 'string') return JSON.stringify(v);
  if (typeof v === 'boolean' || typeof v === 'number') return String(v);
  return JSON.stringify(v);
}
