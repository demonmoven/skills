import { spawnSync } from 'node:child_process';
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname } from 'node:path';
import { VERSION_CHECK_CACHE } from './paths.js';

declare const XDEV_VERSION: string;

const PACKAGE_NAME = '@byted/xdex';
const REGISTRY = 'https://bnpm.byted.org';
const REGISTRY_URL = `${REGISTRY}/${PACKAGE_NAME}`;
const CACHE_TTL_MS = 24 * 60 * 60 * 1000; // 24 hours
const FETCH_TIMEOUT_MS = 3000;
const INSTALL_TIMEOUT_MS = 60_000;

// ---------------------------------------------------------------------------
// ANSI helpers (no chalk dependency — keep bundle small)
// ---------------------------------------------------------------------------

const isTTY = () => Boolean(process.stderr.isTTY);

function c(text: string, code: string): string {
  return isTTY() ? `${code}${text}\x1b[0m` : text;
}

const dim = (s: string) => c(s, '\x1b[2m');
const green = (s: string) => c(s, '\x1b[32m');
const yellow = (s: string) => c(s, '\x1b[33m');
const bold = (s: string) => c(s, '\x1b[1m');

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface VersionCacheEntry {
  checked_at: number;
  current_version: string;
  latest_version: string | null;
}

export interface VersionInfo {
  current: string;
  latest: string | null;
  lastCheckedAt: number | null;
  source: 'cache' | 'fresh' | 'unavailable' | 'disabled';
  needsUpgrade: boolean;
  autoUpdateDisabled: boolean;
}

// ---------------------------------------------------------------------------
// Semver comparison
// ---------------------------------------------------------------------------

export function isNewer(a: string, b: string): boolean {
  const parse = (v: string): [number, number, number] | null => {
    const core = v.replace(/^v/, '').split(/[-+]/)[0];
    const parts = core.split('.').map((s) => Number(s));
    if (parts.length !== 3 || parts.some((n) => !Number.isFinite(n) || n < 0)) {
      return null;
    }
    return [parts[0], parts[1], parts[2]];
  };
  const pa = parse(a);
  const pb = parse(b);
  if (!pa || !pb) return false;
  for (let i = 0; i < 3; i++) {
    if (pa[i] > pb[i]) return true;
    if (pa[i] < pb[i]) return false;
  }
  return false;
}

// ---------------------------------------------------------------------------
// Relative time formatting
// ---------------------------------------------------------------------------

export function formatRelativeTime(ms: number): string {
  const SEC = 1000;
  const MIN = 60 * SEC;
  const HOUR = 60 * MIN;
  const DAY = 24 * HOUR;

  if (ms < MIN) return '刚刚';
  if (ms < HOUR) return `${Math.floor(ms / MIN)}m 前`;
  if (ms < DAY) return `${Math.floor(ms / HOUR)}h 前`;
  if (ms < 30 * DAY) return `${Math.floor(ms / DAY)}d 前`;
  return '很久前';
}

// ---------------------------------------------------------------------------
// getVersionInfo — pure data collection, no side effects
// ---------------------------------------------------------------------------

export function getVersionInfo(): VersionInfo {
  const disabled =
    process.env['XDEV_NO_UPDATE'] === '1' ||
    process.env['XDEV_NO_UPDATE_NOTIFIER'] === '1';

  if (disabled) {
    return {
      current: XDEV_VERSION,
      latest: null,
      lastCheckedAt: null,
      source: 'disabled',
      needsUpgrade: false,
      autoUpdateDisabled: true,
    };
  }

  const cache = readCache();
  const cacheFresh = cache && Date.now() - cache.checked_at < CACHE_TTL_MS;
  const cacheStale = cacheFresh && cache.latest_version && isNewer(XDEV_VERSION, cache.latest_version);

  if (cacheFresh && !cacheStale) {
    const needs = Boolean(cache.latest_version && isNewer(cache.latest_version, XDEV_VERSION));
    return {
      current: XDEV_VERSION,
      latest: cache.latest_version,
      lastCheckedAt: cache.checked_at,
      source: 'cache',
      needsUpgrade: needs,
      autoUpdateDisabled: false,
    };
  }

  const latest = fetchLatestSync();
  writeCache({
    checked_at: Date.now(),
    current_version: XDEV_VERSION,
    latest_version: latest,
  });

  if (!latest) {
    return {
      current: XDEV_VERSION,
      latest: null,
      lastCheckedAt: Date.now(),
      source: 'unavailable',
      needsUpgrade: false,
      autoUpdateDisabled: false,
    };
  }

  return {
    current: XDEV_VERSION,
    latest,
    lastCheckedAt: Date.now(),
    source: 'fresh',
    needsUpgrade: isNewer(latest, XDEV_VERSION),
    autoUpdateDisabled: false,
  };
}

// ---------------------------------------------------------------------------
// formatVersionBanner — pure string rendering
// ---------------------------------------------------------------------------

export function formatVersionBanner(info: VersionInfo, mode: 'compact' | 'detailed'): string {
  return mode === 'compact' ? formatCompact(info) : formatDetailed(info);
}

function formatCompact(info: VersionInfo): string {
  const cur = `current: ${bold(info.current)}`;

  if (info.autoUpdateDisabled) {
    return `[xdev] ${cur} ${dim('(自动升级已禁用，XDEV_NO_UPDATE=1)')}`;
  }

  if (info.latest === null) {
    return `[xdev] ${cur} · remote: ${dim('未知（离线或 registry 不可达）')}`;
  }

  const timeHint = info.lastCheckedAt
    ? dim(`(${formatRelativeTime(Date.now() - info.lastCheckedAt)}检查)`)
    : '';

  if (info.needsUpgrade) {
    return `[xdev] ${cur} · remote: ${yellow(info.latest)}  ${yellow('↑ 发现新版本，正在自动升级…')}`;
  }

  return `[xdev] ${cur} · remote: ${info.latest} ${timeHint}  ${green('✓ 已是最新')}`;
}

function formatDetailed(info: VersionInfo): string {
  const lines: string[] = [bold('xdev (@byted/xdex)'), ''];

  lines.push(`  current (本地) : ${bold(info.current)}`);

  if (info.autoUpdateDisabled) {
    lines.push(`  remote  (远端) : ${dim('—（自动升级已禁用）')}`);
  } else if (info.latest === null) {
    lines.push(`  remote  (远端) : ${dim('未知（离线或 registry 不可达）')}`);
  } else if (info.needsUpgrade) {
    lines.push(`  remote  (远端) : ${yellow(info.latest)}    ${yellow('↑ 有新版本可升级')}`);
  } else {
    lines.push(`  remote  (远端) : ${green(info.latest)}    ${green('✓ 已是最新')}`);
  }

  if (info.lastCheckedAt) {
    const rel = formatRelativeTime(Date.now() - info.lastCheckedAt);
    const abs = new Date(info.lastCheckedAt).toLocaleString('zh-CN', { hour12: false });
    const srcLabel = info.source === 'cache' ? '缓存' : '刚刚';
    lines.push(`  上次检查       : ${abs} (${rel}，${srcLabel})`);
  } else {
    lines.push(`  上次检查       : ${dim('—')}`);
  }

  lines.push(`  registry       : ${dim(REGISTRY)}`);
  lines.push('');

  if (info.autoUpdateDisabled) {
    lines.push(`  自动升级       : ${dim('已禁用 (XDEV_NO_UPDATE=1)')}`);
  } else {
    lines.push(`  自动升级       : 启动时若发现新版会自动升级`);
    lines.push(`  禁用自动       : export XDEV_NO_UPDATE=1`);
  }
  lines.push(`  手动升级       : npm install -g ${PACKAGE_NAME}@latest --registry ${REGISTRY}`);

  return lines.join('\n');
}

// ---------------------------------------------------------------------------
// runSelfUpdateCheck — thin orchestrator (entry point from index.ts)
// ---------------------------------------------------------------------------

export function runSelfUpdateCheck(): void {
  try {
    const info = getVersionInfo();

    if (!shouldQuiet(info)) {
      const mode = detectMode();
      console.error(formatVersionBanner(info, mode));
      console.error('');
    }

    if (info.needsUpgrade && info.latest) {
      attemptUpgrade(info.latest);
    }
  } catch (err) {
    console.error(
      `[xdev] update check failed: ${err instanceof Error ? err.message : String(err)}`,
    );
  }
}

function shouldQuiet(info: VersionInfo): boolean {
  if (process.env['XDEV_QUIET'] === '1') return true;
  if (detectMode() === 'detailed') return false;
  if (!isTTY()) return true;
  return false;
}

function detectMode(): 'compact' | 'detailed' {
  const args = process.argv.slice(2);
  if (args.includes('--version') || args.includes('-V')) return 'detailed';
  return 'compact';
}

// ---------------------------------------------------------------------------
// attemptUpgrade (preserved from original)
// ---------------------------------------------------------------------------

function attemptUpgrade(latestVersion: string): void {
  console.error(`[xdev] upgrading via: npm install -g ${PACKAGE_NAME}@${latestVersion} --registry ${REGISTRY}`);
  console.error('');

  const result = spawnSync(
    'npm',
    ['install', '-g', `${PACKAGE_NAME}@${latestVersion}`, '--registry', REGISTRY],
    {
      stdio: ['ignore', 'inherit', 'inherit'],
      timeout: INSTALL_TIMEOUT_MS,
    },
  );

  if (result.error || result.status !== 0) {
    const reason = result.error?.message ?? `exit code ${result.status}`;
    console.error(
      `\n[xdev] auto-upgrade failed (${reason}).\n` +
        `[xdev] continuing with current version ${XDEV_VERSION}.\n` +
        `[xdev] you can manually upgrade with:\n` +
        `       npm install -g ${PACKAGE_NAME}@latest --registry ${REGISTRY}\n`,
    );
    return;
  }

  try {
    writeCache({
      checked_at: 0,
      current_version: latestVersion,
      latest_version: latestVersion,
    });
  } catch {
    /* not critical */
  }

  console.error(
    `\n[xdev] ${green('✓')} upgraded ${XDEV_VERSION} → ${latestVersion}\n` +
      `[xdev] please re-run your command to use the new version\n`,
  );
  process.exit(0);
}

// ---------------------------------------------------------------------------
// Cache I/O (unchanged)
// ---------------------------------------------------------------------------

function fetchLatestSync(): string | null {
  try {
    const result = spawnSync(
      'curl',
      [
        '-fsSL',
        '--max-time',
        String(Math.ceil(FETCH_TIMEOUT_MS / 1000)),
        '-H',
        'Accept: application/json',
        REGISTRY_URL,
      ],
      {
        encoding: 'utf-8',
        timeout: FETCH_TIMEOUT_MS + 1000,
      },
    );
    if (result.error || result.status !== 0 || !result.stdout) return null;
    const data = JSON.parse(result.stdout) as { 'dist-tags'?: { latest?: string } };
    return data['dist-tags']?.latest ?? null;
  } catch {
    return null;
  }
}

function readCache(): VersionCacheEntry | null {
  try {
    if (!existsSync(VERSION_CHECK_CACHE)) return null;
    return JSON.parse(readFileSync(VERSION_CHECK_CACHE, 'utf-8')) as VersionCacheEntry;
  } catch {
    return null;
  }
}

function writeCache(entry: VersionCacheEntry): void {
  try {
    mkdirSync(dirname(VERSION_CHECK_CACHE), { recursive: true });
    writeFileSync(VERSION_CHECK_CACHE, JSON.stringify(entry, null, 2));
  } catch {
    /* silent */
  }
}
