import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtempSync, rmSync, readFileSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import YAML from 'yaml';
import { applyBundledCredsIfMissing, type BundledTosCreds } from './bundled-creds.js';

const REAL_CREDS: BundledTosCreds = {
  ak: 'test-ak',
  sk: 'test-sk',
  bucket: 'test-bucket',
  region: 'cn-beijing',
};

let root: string;
let configPath: string;

beforeEach(() => {
  root = mkdtempSync(join(tmpdir(), 'xdev-creds-'));
  configPath = join(root, 'config.yaml');
});

afterEach(() => {
  rmSync(root, { recursive: true, force: true });
});

describe('applyBundledCredsIfMissing', () => {
  it('returns skipped when all creds empty (placeholder state)', () => {
    const result = applyBundledCredsIfMissing(configPath, { ak: '', sk: '', bucket: '', region: '' });
    expect(result).toBe('skipped');
  });

  it('writes all fields into new config.yaml', () => {
    const result = applyBundledCredsIfMissing(configPath, REAL_CREDS);
    expect(result).toBe('written');
    const parsed = YAML.parse(readFileSync(configPath, 'utf-8')) as {
      tos: { accessKey: string; secretKey: string; bucket: string; region: string };
    };
    expect(parsed.tos.accessKey).toBe('test-ak');
    expect(parsed.tos.secretKey).toBe('test-sk');
    expect(parsed.tos.bucket).toBe('test-bucket');
    expect(parsed.tos.region).toBe('cn-beijing');
  });

  it('does NOT overwrite user-set fields (only fills missing)', () => {
    writeFileSync(configPath, `tos:\n  accessKey: user-set-ak\n  region: cn-shanghai\n`);
    const result = applyBundledCredsIfMissing(configPath, REAL_CREDS);
    expect(result).toBe('written');
    const parsed = YAML.parse(readFileSync(configPath, 'utf-8')) as {
      tos: { accessKey: string; secretKey: string; bucket: string; region: string };
    };
    expect(parsed.tos.accessKey).toBe('user-set-ak');
    expect(parsed.tos.region).toBe('cn-shanghai');
    expect(parsed.tos.secretKey).toBe('test-sk');
    expect(parsed.tos.bucket).toBe('test-bucket');
  });

  it('returns unchanged when all fields already user-set', () => {
    writeFileSync(
      configPath,
      `tos:\n  accessKey: user-ak\n  secretKey: user-sk\n  bucket: user-bucket\n  region: user-region\n`,
    );
    const result = applyBundledCredsIfMissing(configPath, REAL_CREDS);
    expect(result).toBe('unchanged');
    const parsed = YAML.parse(readFileSync(configPath, 'utf-8')) as { tos: { accessKey: string } };
    expect(parsed.tos.accessKey).toBe('user-ak');
  });

  it('preserves unrelated top-level fields', () => {
    writeFileSync(configPath, `privacy:\n  collect_content: true\n`);
    applyBundledCredsIfMissing(configPath, REAL_CREDS);
    const parsed = YAML.parse(readFileSync(configPath, 'utf-8')) as {
      privacy: { collect_content: boolean };
      tos: { accessKey: string };
    };
    expect(parsed.privacy.collect_content).toBe(true);
    expect(parsed.tos.accessKey).toBe('test-ak');
  });
});

