import { describe, it, expect } from 'vitest';
import { isNewer, formatRelativeTime, formatVersionBanner, type VersionInfo } from './version-check.js';

// ---------------------------------------------------------------------------
// isNewer (preserved from original test suite)
// ---------------------------------------------------------------------------

describe('isNewer', () => {
  it('returns true when patch is strictly greater', () => {
    expect(isNewer('0.0.5', '0.0.4')).toBe(true);
    expect(isNewer('1.2.4', '1.2.3')).toBe(true);
  });

  it('returns true when minor is strictly greater', () => {
    expect(isNewer('0.1.0', '0.0.99')).toBe(true);
    expect(isNewer('1.3.0', '1.2.99')).toBe(true);
  });

  it('returns true when major is strictly greater', () => {
    expect(isNewer('1.0.0', '0.99.99')).toBe(true);
    expect(isNewer('2.0.0', '1.99.99')).toBe(true);
  });

  it('returns false when patch is strictly less (no downgrade)', () => {
    expect(isNewer('0.0.4', '0.0.5')).toBe(false);
  });

  it('returns false when minor is strictly less', () => {
    expect(isNewer('0.0.99', '0.1.0')).toBe(false);
  });

  it('returns false when major is strictly less', () => {
    expect(isNewer('0.99.99', '1.0.0')).toBe(false);
  });

  it('returns false when versions are equal', () => {
    expect(isNewer('0.0.5', '0.0.5')).toBe(false);
    expect(isNewer('1.2.3', '1.2.3')).toBe(false);
  });

  it('strips leading "v" prefix', () => {
    expect(isNewer('v0.0.5', '0.0.4')).toBe(true);
    expect(isNewer('0.0.5', 'v0.0.4')).toBe(true);
    expect(isNewer('v0.0.4', 'v0.0.5')).toBe(false);
  });

  it('strips "-pre" / "+build" suffix and compares the numeric core', () => {
    expect(isNewer('0.0.5-rc.1', '0.0.4')).toBe(true);
    expect(isNewer('0.0.5+build.42', '0.0.4')).toBe(true);
    expect(isNewer('0.0.5-rc.2', '0.0.5-rc.1')).toBe(false);
  });

  it('returns false on invalid input rather than throwing', () => {
    expect(isNewer('not-a-version', '0.0.5')).toBe(false);
    expect(isNewer('0.0.5', 'not-a-version')).toBe(false);
    expect(isNewer('1.2', '1.2.3')).toBe(false);
    expect(isNewer('1.2.3.4', '1.2.3')).toBe(false);
    expect(isNewer('', '0.0.5')).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// formatRelativeTime
// ---------------------------------------------------------------------------

describe('formatRelativeTime', () => {
  it('returns "刚刚" for < 1 minute', () => {
    expect(formatRelativeTime(0)).toBe('刚刚');
    expect(formatRelativeTime(30_000)).toBe('刚刚');
    expect(formatRelativeTime(59_999)).toBe('刚刚');
  });

  it('returns minutes for < 1 hour', () => {
    expect(formatRelativeTime(60_000)).toBe('1m 前');
    expect(formatRelativeTime(5 * 60_000)).toBe('5m 前');
    expect(formatRelativeTime(59 * 60_000 + 59_999)).toBe('59m 前');
  });

  it('returns hours for < 1 day', () => {
    expect(formatRelativeTime(60 * 60_000)).toBe('1h 前');
    expect(formatRelativeTime(3 * 60 * 60_000)).toBe('3h 前');
    expect(formatRelativeTime(23 * 60 * 60_000 + 59 * 60_000)).toBe('23h 前');
  });

  it('returns days for < 30 days', () => {
    expect(formatRelativeTime(24 * 60 * 60_000)).toBe('1d 前');
    expect(formatRelativeTime(7 * 24 * 60 * 60_000)).toBe('7d 前');
    expect(formatRelativeTime(29 * 24 * 60 * 60_000)).toBe('29d 前');
  });

  it('returns "很久前" for >= 30 days', () => {
    expect(formatRelativeTime(30 * 24 * 60 * 60_000)).toBe('很久前');
    expect(formatRelativeTime(365 * 24 * 60 * 60_000)).toBe('很久前');
  });
});

// ---------------------------------------------------------------------------
// formatVersionBanner
// ---------------------------------------------------------------------------

const baseInfo: VersionInfo = {
  current: '0.0.7',
  latest: '0.0.7',
  lastCheckedAt: Date.now() - 2 * 60 * 60_000, // 2h ago
  source: 'cache',
  needsUpgrade: false,
  autoUpdateDisabled: false,
};

describe('formatVersionBanner', () => {
  describe('compact mode', () => {
    it('shows up-to-date when current === latest', () => {
      const out = formatVersionBanner(baseInfo, 'compact');
      expect(out).toContain('current:');
      expect(out).toContain('0.0.7');
      expect(out).toContain('remote:');
      expect(out).toContain('已是最新');
    });

    it('shows upgrade hint when needsUpgrade', () => {
      const info: VersionInfo = { ...baseInfo, latest: '0.0.9', needsUpgrade: true };
      const out = formatVersionBanner(info, 'compact');
      expect(out).toContain('0.0.9');
      expect(out).toContain('发现新版本');
    });

    it('shows unknown when latest is null', () => {
      const info: VersionInfo = { ...baseInfo, latest: null, source: 'unavailable' };
      const out = formatVersionBanner(info, 'compact');
      expect(out).toContain('未知');
      expect(out).toContain('离线');
    });

    it('shows disabled hint when autoUpdateDisabled', () => {
      const info: VersionInfo = { ...baseInfo, autoUpdateDisabled: true, latest: null, source: 'disabled' };
      const out = formatVersionBanner(info, 'compact');
      expect(out).toContain('XDEV_NO_UPDATE');
      expect(out).toContain('自动升级已禁用');
    });
  });

  describe('detailed mode', () => {
    it('shows structured info when up-to-date', () => {
      const out = formatVersionBanner(baseInfo, 'detailed');
      expect(out).toContain('xdev (@byted/xdex)');
      expect(out).toContain('current (本地)');
      expect(out).toContain('remote  (远端)');
      expect(out).toContain('已是最新');
      expect(out).toContain('registry');
      expect(out).toContain('bnpm.byted.org');
    });

    it('shows upgrade available when needsUpgrade', () => {
      const info: VersionInfo = { ...baseInfo, latest: '0.0.9', needsUpgrade: true };
      const out = formatVersionBanner(info, 'detailed');
      expect(out).toContain('0.0.9');
      expect(out).toContain('有新版本可升级');
    });

    it('shows unknown remote when offline', () => {
      const info: VersionInfo = { ...baseInfo, latest: null, source: 'unavailable' };
      const out = formatVersionBanner(info, 'detailed');
      expect(out).toContain('未知');
    });

    it('shows disabled state', () => {
      const info: VersionInfo = { ...baseInfo, autoUpdateDisabled: true, latest: null, source: 'disabled' };
      const out = formatVersionBanner(info, 'detailed');
      expect(out).toContain('已禁用');
    });

    it('includes manual upgrade command', () => {
      const out = formatVersionBanner(baseInfo, 'detailed');
      expect(out).toContain('npm install -g @byted/xdex@latest');
    });
  });
});
