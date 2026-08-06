import { describe, expect, it, beforeEach, afterEach } from 'vitest';
import { existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { mergeTmuxConf } from './setup.js';
import { END_MARKER, START_MARKER, XDEV_TMUX_CONF_BLOCK } from './conf-template.js';

describe('mergeTmuxConf', () => {
  let tmpDir: string;
  let confPath: string;

  beforeEach(() => {
    tmpDir = mkdtempSync(join(tmpdir(), 'xdev-tmux-test-'));
    confPath = join(tmpDir, '.tmux.conf');
  });

  afterEach(() => {
    rmSync(tmpDir, { recursive: true, force: true });
  });

  it('creates the file with the block when no file exists', () => {
    expect(existsSync(confPath)).toBe(false);
    const replaced = mergeTmuxConf(confPath);
    expect(replaced).toBe(false);
    const content = readFileSync(confPath, 'utf-8');
    expect(content).toContain(START_MARKER);
    expect(content).toContain(END_MARKER);
    expect(content).toContain('set -g mouse on');
  });

  it('appends the block when file exists without it (preserves existing content)', () => {
    const existing = '# user custom config\nset -g history-limit 100000\n';
    writeFileSync(confPath, existing);
    const replaced = mergeTmuxConf(confPath);
    expect(replaced).toBe(false);
    const content = readFileSync(confPath, 'utf-8');
    expect(content.startsWith('# user custom config')).toBe(true);
    expect(content).toContain('set -g history-limit 100000');
    expect(content).toContain(XDEV_TMUX_CONF_BLOCK.trim());
  });

  it('idempotently replaces an existing XDEV-TMUX block (no duplication)', () => {
    mergeTmuxConf(confPath);
    const replaced = mergeTmuxConf(confPath);
    expect(replaced).toBe(true);
    const content = readFileSync(confPath, 'utf-8');
    const startCount = content.split(START_MARKER).length - 1;
    const endCount = content.split(END_MARKER).length - 1;
    expect(startCount).toBe(1);
    expect(endCount).toBe(1);
  });

  it('preserves user content around the block when replacing', () => {
    const head = '# user prefix\nset -g status on\n';
    const tail = '\n# user suffix\nset -g base-index 1\n';
    writeFileSync(confPath, head + XDEV_TMUX_CONF_BLOCK + tail);
    const replaced = mergeTmuxConf(confPath);
    expect(replaced).toBe(true);
    const content = readFileSync(confPath, 'utf-8');
    expect(content).toContain('# user prefix');
    expect(content).toContain('set -g status on');
    expect(content).toContain('# user suffix');
    expect(content).toContain('set -g base-index 1');
    expect(content.split(START_MARKER).length - 1).toBe(1);
  });
});
