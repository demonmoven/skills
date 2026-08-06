import { execFileSync, spawnSync } from 'node:child_process';
import {
  cpSync,
  existsSync,
  lstatSync,
  mkdirSync,
  readdirSync,
  readFileSync,
  realpathSync,
  rmSync,
  writeFileSync,
} from 'node:fs';
import { dirname, join } from 'node:path';
import { BYTEDCLI_CHECK_CACHE, findProjectRoot } from './paths.js';
import type { Agent } from './prompt.js';

const BYTEDCLI_PACKAGE = '@bytedance-dev/bytedcli';
const BYTEDCLI_REGISTRY = 'https://bnpm.byted.org';
const BYTEDCLI_GIT = 'git@code.byted.org:byteapi/bytedcli.git';

const REQUIRED_SKILLS = [
  'bytedance-auth',
  'bytedance-tce',
  'bytedance-agw',
  'bytedance-log',
  'bytedance-tools',
] as const;

const AGENT_SKILL_MAP: Record<Agent, { dirName: string; skillsAgent: string }> = {
  cc: { dirName: '.claude', skillsAgent: 'claude-code' },
  cdx: { dirName: '.codex', skillsAgent: 'codex' },
  trae: { dirName: '.trae', skillsAgent: 'trae' },
  'trae-cn': { dirName: '.trae', skillsAgent: 'trae' },
  coco: { dirName: '.coco', skillsAgent: 'coco' },
};

const UPDATE_CHECK_TTL_MS = 12 * 60 * 60 * 1000;
const NPM_VIEW_TIMEOUT_MS = 10_000;

interface CheckCache {
  checked_at: number;
  latest_version?: string;
}

export async function ensureBytedcliDeps(agent: Agent): Promise<void> {
  if (process.env.XDEV_SKIP_BYTEDCLI_DEPS === '1') {
    console.log('[xdev] XDEV_SKIP_BYTEDCLI_DEPS=1 — skipping bytedcli dependency check');
    return;
  }

  console.log('[xdev] Checking bytedcli dependencies (fixloop prerequisite)...');

  ensureBytedcliBinary();
  maybeUpgradeBytedcli();
  ensureBytedcliSkillPacks(agent);
}

function detectBytedcliVersion(): string | null {
  try {
    const out = execFileSync('bytedcli', ['--version'], {
      stdio: ['ignore', 'pipe', 'ignore'],
      encoding: 'utf-8',
      timeout: 10_000,
    });
    const match = out.trim().match(/(\d+\.\d+\.\d+)/);
    return match ? match[1] : null;
  } catch {
    return null;
  }
}

function ensureBytedcliBinary(): void {
  const version = detectBytedcliVersion();
  if (version) {
    console.log(`[xdev] bytedcli ${version} detected`);
    return;
  }
  console.log('[xdev] bytedcli not found — installing globally via npm...');
  installOrUpgradeBytedcliGlobal();
  const installed = detectBytedcliVersion();
  if (!installed) {
    throw new Error(
      '[xdev] bytedcli install succeeded but `bytedcli --version` still fails. ' +
        'Check that your global npm bin directory is on PATH. ' +
        `Manual: npm install -g ${BYTEDCLI_PACKAGE}@latest --registry ${BYTEDCLI_REGISTRY}`,
    );
  }
  console.log(`[xdev] bytedcli ${installed} installed`);
}

function installOrUpgradeBytedcliGlobal(): void {
  const result = spawnSync(
    'npm',
    ['install', '-g', `${BYTEDCLI_PACKAGE}@latest`, '--registry', BYTEDCLI_REGISTRY],
    { stdio: ['ignore', 'inherit', 'pipe'], encoding: 'utf-8' },
  );

  if (result.status === 0) return;

  const stderr = (result.stderr || '').toString();
  const combinedErr = stderr + ' ' + (result.error?.message ?? '');

  if (/EACCES|permission denied/i.test(combinedErr)) {
    throw new Error(
      '[xdev] `npm install -g` was denied by the OS (EACCES). Re-run xdev with elevated privileges:\n' +
        `    sudo npm install -g ${BYTEDCLI_PACKAGE}@latest --registry ${BYTEDCLI_REGISTRY}\n` +
        'Or fix your global npm prefix so a non-root user can write to it.',
    );
  }

  throw new Error(
    `[xdev] npm install of ${BYTEDCLI_PACKAGE} failed (exit ${result.status}). ${stderr.trim() || result.error?.message || ''}`.trim(),
  );
}

function maybeUpgradeBytedcli(): void {
  const now = Date.now();
  const cache = readCheckCache();
  if (cache && now - cache.checked_at < UPDATE_CHECK_TTL_MS) return;

  const latest = fetchLatestVersion();
  if (!latest) {
    writeCheckCache({ checked_at: now });
    return;
  }

  const current = detectBytedcliVersion();
  if (current && compareVersions(current, latest) < 0) {
    console.log(`[xdev] Upgrading bytedcli: ${current} -> ${latest}`);
    const result = spawnSync('bytedcli', ['self', 'update'], { stdio: 'inherit' });
    if (result.status !== 0) {
      console.log('[xdev] `bytedcli self update` failed, falling back to `npm install -g`...');
      installOrUpgradeBytedcliGlobal();
    }
  }

  writeCheckCache({ checked_at: now, latest_version: latest });
}

function fetchLatestVersion(): string | null {
  const result = spawnSync(
    'npm',
    ['view', BYTEDCLI_PACKAGE, 'version', '--registry', BYTEDCLI_REGISTRY],
    { stdio: ['ignore', 'pipe', 'pipe'], encoding: 'utf-8', timeout: NPM_VIEW_TIMEOUT_MS },
  );
  if (result.status !== 0) {
    console.warn('[xdev] Could not reach bnpm to check bytedcli latest version (offline?), skipping upgrade check');
    return null;
  }
  const out = (result.stdout || '').toString().trim();
  return /^\d+\.\d+\.\d+/.test(out) ? out : null;
}

const VERSION_MARKER = '.bytedcli-version';

function ensureBytedcliSkillPacks(agent: Agent): void {
  const { dirName, skillsAgent } = AGENT_SKILL_MAP[agent];
  const projectRoot = findProjectRoot(process.cwd());
  const skillsDir = join(projectRoot, dirName, 'skills');
  const markerPath = join(skillsDir, VERSION_MARKER);
  const currentCliVersion = detectBytedcliVersion();

  // If bytedcli CLI has changed since we last installed the skill pack at the
  // project level, wipe the 5 skill dirs so `missing` becomes the full set
  // and we redownload fresh SKILL.md content aligned with the new CLI.
  if (currentCliVersion && existsSync(markerPath)) {
    const installedFor = readFileSync(markerPath, 'utf-8').trim();
    if (installedFor && installedFor !== currentCliVersion) {
      console.log(
        `[xdev] bytedcli CLI changed (${installedFor} -> ${currentCliVersion}), refreshing project-level skills...`,
      );
      for (const skill of REQUIRED_SKILLS) {
        rmSync(join(skillsDir, skill), { recursive: true, force: true });
      }
    }
  }

  const missing = REQUIRED_SKILLS.filter(
    (s) => !existsSync(join(skillsDir, s, 'SKILL.md')),
  );
  if (missing.length === 0) {
    writeVersionMarker(markerPath, currentCliVersion);
    console.log(`[xdev] bytedcli skills already present in ${dirName}/skills/ (${REQUIRED_SKILLS.length} skills)`);
    return;
  }

  console.log(
    `[xdev] Installing ${missing.length} bytedcli skill(s) into ${dirName}/skills/: ${missing.join(', ')}`,
  );

  const failures: string[] = [];
  for (const skill of missing) {
    const result = spawnSync(
      'npx',
      ['-y', 'skills', 'add', BYTEDCLI_GIT, '--skill', skill, '-a', skillsAgent, '-y'],
      { stdio: 'inherit', cwd: projectRoot },
    );
    if (result.status !== 0) {
      failures.push(skill);
    }
  }

  if (failures.length > 0) {
    throw new Error(
      `[xdev] Failed to install bytedcli skill(s): ${failures.join(', ')}. ` +
        `You can retry manually with: npx -y skills add ${BYTEDCLI_GIT} --skill <name> -a ${skillsAgent} -y ` +
        `(from ${projectRoot})`,
    );
  }

  // `skills add` places the real files under .agents/skills/<name>/ and creates
  // symlinks at .<agent>/skills/<name> pointing there. We want each coding
  // agent's own .<agent>/skills/ to hold the real files directly (no shared
  // .agents/ dir, no skills-lock.json). Materialize and clean up.
  materializeAndCleanupAgentsDir(projectRoot, skillsDir);

  writeVersionMarker(markerPath, currentCliVersion);
  console.log(`[xdev] Installed ${missing.length} bytedcli skill(s) into ${skillsDir}`);
}

function writeVersionMarker(markerPath: string, version: string | null): void {
  if (!version) return;
  try {
    mkdirSync(dirname(markerPath), { recursive: true });
    writeFileSync(markerPath, version);
  } catch {
    // best-effort only — a missing marker just means we'll refresh on the
    // next bytedcli upgrade detection, no harm done
  }
}

function materializeAndCleanupAgentsDir(projectRoot: string, skillsDir: string): void {
  for (const skill of REQUIRED_SKILLS) {
    const linkPath = join(skillsDir, skill);
    if (!existsSync(linkPath)) continue;
    let isLink = false;
    try {
      isLink = lstatSync(linkPath).isSymbolicLink();
    } catch {
      continue;
    }
    if (!isLink) continue;

    let realPath: string;
    try {
      realPath = realpathSync(linkPath);
    } catch {
      rmSync(linkPath, { recursive: true, force: true });
      continue;
    }
    rmSync(linkPath, { recursive: true, force: true });
    cpSync(realPath, linkPath, { recursive: true, dereference: true });
  }

  const agentsSkillsDir = join(projectRoot, '.agents', 'skills');
  for (const skill of REQUIRED_SKILLS) {
    const srcPath = join(agentsSkillsDir, skill);
    if (existsSync(srcPath)) {
      rmSync(srcPath, { recursive: true, force: true });
    }
  }

  removeIfEmpty(agentsSkillsDir);
  removeIfEmpty(join(projectRoot, '.agents'));

  const lockFile = join(projectRoot, 'skills-lock.json');
  if (existsSync(lockFile)) {
    try {
      const lock = JSON.parse(readFileSync(lockFile, 'utf-8')) as {
        skills?: Record<string, unknown>;
      };
      if (lock?.skills) {
        for (const skill of REQUIRED_SKILLS) {
          delete lock.skills[skill];
        }
        if (Object.keys(lock.skills).length === 0) {
          rmSync(lockFile, { force: true });
        } else {
          writeFileSync(lockFile, JSON.stringify(lock, null, 2));
        }
      }
    } catch {
      // malformed lock file — leave it alone
    }
  }
}

function removeIfEmpty(dir: string): void {
  if (!existsSync(dir)) return;
  try {
    if (readdirSync(dir).length === 0) {
      rmSync(dir, { recursive: true, force: true });
    }
  } catch {
    // directory vanished or unreadable — ignore
  }
}

function readCheckCache(): CheckCache | null {
  if (!existsSync(BYTEDCLI_CHECK_CACHE)) return null;
  try {
    const parsed = JSON.parse(readFileSync(BYTEDCLI_CHECK_CACHE, 'utf-8'));
    if (typeof parsed?.checked_at === 'number') return parsed as CheckCache;
    return null;
  } catch {
    return null;
  }
}

function writeCheckCache(entry: CheckCache): void {
  mkdirSync(dirname(BYTEDCLI_CHECK_CACHE), { recursive: true });
  writeFileSync(BYTEDCLI_CHECK_CACHE, JSON.stringify(entry));
}

export function compareVersions(a: string, b: string): number {
  const parse = (v: string): number[] => v.split('.').map((n) => Number(n) || 0);
  const [am, ai, ap] = parse(a);
  const [bm, bi, bp] = parse(b);
  if (am !== bm) return am - bm;
  if (ai !== bi) return ai - bi;
  return ap - bp;
}
