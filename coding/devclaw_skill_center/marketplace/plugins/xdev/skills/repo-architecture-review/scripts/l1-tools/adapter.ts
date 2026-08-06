// scripts/l1-tools/adapter.ts
import { createHash } from 'node:crypto';
import type { RepoProfile, ToolRunResult, Violation } from '../types.js';

export type InstallVia = 'pipx' | 'go-install' | 'docker' | 'binary' | 'pre-existing';

export interface PreflightResult {
  ready: boolean;
  reason?: 'no-config' | 'no-binary' | 'not-applicable' | 'install-failed';
  installedVia?: InstallVia;
  installMessage?: string;
  generatedConfigPath?: string;
}

export interface AdapterContext {
  profile: RepoProfile;
  tmpDir: string;
  rulesFromBuiltin: boolean;
  timeoutMs: number;
  disabledToolIds: Set<string>;
}

export interface ToolAdapter {
  id: string;
  detectApplies(profile: RepoProfile): boolean;
  preflight(ctx: AdapterContext): Promise<PreflightResult>;
  run(ctx: AdapterContext): Promise<RawRunOutput>;
  normalize(raw: RawRunOutput, ctx: AdapterContext): Violation[];
}

export interface RawRunOutput {
  rawOutputPath: string;
  exitCode: number;
  stdout: string;
  stderr: string;
  durationMs: number;
}

export function hashRuleSource(adapterId: string, ruleId: string, file: string, symbol?: string): string {
  return createHash('sha256')
    .update(`${adapterId}|${ruleId}|${file}|${symbol ?? ''}`)
    .digest('hex')
    .slice(0, 16);
}

export async function runAdapterSafely(
  adapter: ToolAdapter,
  ctx: AdapterContext,
): Promise<ToolRunResult> {
  const started = Date.now();
  if (!adapter.detectApplies(ctx.profile)) {
    return { toolId: adapter.id, status: 'skipped', reason: 'not-applicable', durationMs: 0, violations: [] };
  }
  if (ctx.disabledToolIds.has(adapter.id)) {
    return { toolId: adapter.id, status: 'skipped', reason: 'no-binary', durationMs: 0, violations: [] };
  }
  try {
    const pf = await adapter.preflight(ctx);
    if (!pf.ready) {
      return {
        toolId: adapter.id,
        status: 'skipped',
        reason: pf.reason ?? 'no-binary',
        durationMs: Date.now() - started,
        violations: [],
        generatedConfigPath: pf.generatedConfigPath,
        installedVia: pf.installedVia,
      };
    }
    const raw = await adapter.run(ctx);
    const violations = adapter.normalize(raw, ctx);
    return {
      toolId: adapter.id,
      status: 'ok',
      durationMs: Date.now() - started,
      violations,
      rawOutputPath: raw.rawOutputPath,
      installedVia: pf.installedVia,
      generatedConfigPath: pf.generatedConfigPath,
    };
  } catch (e) {
    return {
      toolId: adapter.id,
      status: 'failed',
      reason: 'exec-failed',
      durationMs: Date.now() - started,
      violations: [],
      error: e instanceof Error ? e.message : String(e),
    };
  }
}
