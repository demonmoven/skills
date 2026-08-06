// scripts/l1-tools/auto-install.ts
import { runCmd } from '../util/exec.js';
import type { PreflightResult, InstallVia } from './adapter.js';

export interface InstallStep {
  via: InstallVia;
  pkg: string;
  args?: string[];
}

export interface EnsureBinaryOpts {
  binary: string;
  installPlan: InstallStep[];
  checkCommand?: { cmd: string; args: string[] };
}

export async function ensureBinary(opts: EnsureBinaryOpts): Promise<PreflightResult> {
  const check = opts.checkCommand ?? { cmd: 'which', args: [opts.binary] };
  const r = await runCmd(check.cmd, check.args, { timeoutMs: 10_000 });
  if (r.status === 'ok' && r.stdout.trim()) {
    return { ready: true, installedVia: 'pre-existing' };
  }

  for (const step of opts.installPlan) {
    const res = await tryInstall(step);
    if (res.ok) {
      const verify = await runCmd(check.cmd, check.args, { timeoutMs: 10_000 });
      if (verify.status === 'ok' && verify.stdout.trim()) {
        return { ready: true, installedVia: step.via, installMessage: res.message };
      }
    }
  }

  return { ready: false, reason: 'install-failed' };
}

async function tryInstall(step: InstallStep): Promise<{ ok: boolean; message: string }> {
  switch (step.via) {
    case 'pipx': {
      const r = await runCmd('pipx', ['install', step.pkg, ...(step.args ?? [])], { timeoutMs: 120_000 });
      return { ok: r.status === 'ok', message: `pipx install ${step.pkg}` };
    }
    case 'go-install': {
      const r = await runCmd('go', ['install', step.pkg, ...(step.args ?? [])], { timeoutMs: 180_000 });
      return { ok: r.status === 'ok', message: `go install ${step.pkg}` };
    }
    case 'docker': {
      const r = await runCmd('docker', ['pull', step.pkg, ...(step.args ?? [])], { timeoutMs: 180_000 });
      return { ok: r.status === 'ok', message: `docker pull ${step.pkg}` };
    }
    case 'binary': {
      if (!step.args || step.args.length === 0) return { ok: false, message: 'binary install missing args' };
      const [cmd, ...rest] = step.args;
      const r = await runCmd(cmd, rest, { timeoutMs: 180_000 });
      return { ok: r.status === 'ok', message: `binary via ${cmd}` };
    }
    default:
      return { ok: false, message: `unknown install via=${step.via as string}` };
  }
}
