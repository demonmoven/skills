import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { mkdir, writeFile, rm } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';

const tempDir = join(tmpdir(), `trace-config-test-${Date.now()}-${Math.random().toString(36).slice(2)}`);

// Mock homedir so global config reads from temp dir, not real ~/.trace/
vi.mock('node:os', async () => {
  const actual = await vi.importActual('node:os');
  return { ...actual, homedir: () => tempDir };
});

const { loadConfig } = await import('./loader.js');
const { defaultConfig } = await import('./defaults.js');

describe('loadConfig', () => {
  beforeEach(async () => {
    await mkdir(tempDir, { recursive: true });
  });

  afterEach(async () => {
    await rm(tempDir, { recursive: true, force: true });
  });

  it('returns default config when no config files exist', async () => {
    const config = await loadConfig(tempDir);
    expect(config.tos.bucket).toBe(defaultConfig.tos.bucket);
    expect(config.tos.region).toBe(defaultConfig.tos.region);
    expect(config.privacy.collect_content).toBe(false);
    expect(config.privacy.allowed_dirs).toEqual([]);
  });

  it('loads project-level config and overrides defaults', async () => {
    const configDir = join(tempDir, '.trace');
    await mkdir(configDir, { recursive: true });
    await writeFile(
      join(configDir, 'config.yaml'),
      'tos:\n  bucket: "my-custom-bucket"\n  region: "us-east-1"\n',
    );

    const config = await loadConfig(tempDir);
    expect(config.tos.bucket).toBe('my-custom-bucket');
    expect(config.tos.region).toBe('us-east-1');
    expect(config.privacy.collect_content).toBe(false);
  });

  it('merges partial project config with defaults', async () => {
    const configDir = join(tempDir, '.trace');
    await mkdir(configDir, { recursive: true });
    await writeFile(
      join(configDir, 'config.yaml'),
      'privacy:\n  collect_content: true\n',
    );

    const config = await loadConfig(tempDir);
    expect(config.tos.bucket).toBe(defaultConfig.tos.bucket);
    expect(config.privacy.collect_content).toBe(true);
  });

  it('throws on invalid YAML', async () => {
    const configDir = join(tempDir, '.trace');
    await mkdir(configDir, { recursive: true });
    await writeFile(
      join(configDir, 'config.yaml'),
      ':\ninvalid: [yaml\n',
    );

    await expect(loadConfig(tempDir)).rejects.toThrow();
  });
});
