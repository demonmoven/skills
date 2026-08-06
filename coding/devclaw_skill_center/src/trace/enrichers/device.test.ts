import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { mkdir, rm, readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';

const tempDir = join(tmpdir(), `trace-device-test-${Date.now()}-${Math.random().toString(36).slice(2)}`);

vi.mock('node:os', async () => {
  const actual = await vi.importActual('node:os');
  return { ...actual, homedir: () => tempDir };
});

const { getClientId, resetClientIdCache } = await import('./device.js');

describe('device enricher', () => {
  beforeEach(async () => {
    resetClientIdCache();
    await mkdir(join(tempDir, '.trace'), { recursive: true });
  });

  afterEach(async () => {
    await rm(tempDir, { recursive: true, force: true });
  });

  it('generates a UUID client ID on first call', async () => {
    const id = await getClientId();
    expect(id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
  });

  it('persists client ID to device.json', async () => {
    const id = await getClientId();
    const content = JSON.parse(await readFile(join(tempDir, '.trace', 'device.json'), 'utf-8'));
    expect(content.clientId).toBe(id);
  });

  it('returns same ID on subsequent calls (cached)', async () => {
    const id1 = await getClientId();
    const id2 = await getClientId();
    expect(id1).toBe(id2);
  });

  it('reads existing ID from file after cache reset', async () => {
    const id1 = await getClientId();
    resetClientIdCache();
    const id2 = await getClientId();
    expect(id1).toBe(id2);
  });
});
