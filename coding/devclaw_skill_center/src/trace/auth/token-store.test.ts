import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { mkdir, rm, stat } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import type { TokenPair } from './types.js';

// We need to mock homedir to use temp dirs for testing
const tempDir = join(tmpdir(), `trace-auth-test-${Date.now()}-${Math.random().toString(36).slice(2)}`);
const credPath = join(tempDir, '.trace', 'credentials.json');

vi.mock('node:os', async () => {
  const actual = await vi.importActual('node:os');
  return { ...actual, homedir: () => tempDir };
});

const { saveToken, loadToken, deleteToken, isTokenExpired, parseUserFromToken } = await import('./token-store.js');

describe('token-store', () => {
  beforeEach(async () => {
    await mkdir(join(tempDir, '.trace'), { recursive: true });
  });

  afterEach(async () => {
    await rm(tempDir, { recursive: true, force: true });
  });

  const sampleToken: TokenPair = {
    accessToken: 'test-access-token',
    refreshToken: 'test-refresh-token',
    expiresAt: Math.floor(Date.now() / 1000) + 3600,
  };

  it('saves and loads token', async () => {
    await saveToken(sampleToken);
    const loaded = await loadToken();
    expect(loaded).toEqual(sampleToken);
  });

  it('sets file permissions to 0600', async () => {
    await saveToken(sampleToken);
    const stats = await stat(credPath);
    // Check owner read/write only (0o600 = 0b110000000 = 384)
    expect(stats.mode & 0o777).toBe(0o600);
  });

  it('returns null when no credentials file exists', async () => {
    const loaded = await loadToken();
    expect(loaded).toBeNull();
  });

  it('deletes token', async () => {
    await saveToken(sampleToken);
    await deleteToken();
    const loaded = await loadToken();
    expect(loaded).toBeNull();
  });

  it('delete is idempotent (no error when file missing)', async () => {
    await expect(deleteToken()).resolves.toBeUndefined();
  });
});

describe('isTokenExpired', () => {
  it('returns false for non-expired token', () => {
    const token: TokenPair = {
      accessToken: 'a',
      refreshToken: 'r',
      expiresAt: Math.floor(Date.now() / 1000) + 3600,
    };
    expect(isTokenExpired(token)).toBe(false);
  });

  it('returns true for expired token', () => {
    const token: TokenPair = {
      accessToken: 'a',
      refreshToken: 'r',
      expiresAt: Math.floor(Date.now() / 1000) - 100,
    };
    expect(isTokenExpired(token)).toBe(true);
  });

  it('returns true when within 60s safety margin', () => {
    const token: TokenPair = {
      accessToken: 'a',
      refreshToken: 'r',
      expiresAt: Math.floor(Date.now() / 1000) + 30, // 30s left, but 60s margin
    };
    expect(isTokenExpired(token)).toBe(true);
  });
});

describe('parseUserFromToken', () => {
  function makeJwt(payload: Record<string, unknown>): string {
    const header = Buffer.from(JSON.stringify({ alg: 'RS256' })).toString('base64url');
    const body = Buffer.from(JSON.stringify(payload)).toString('base64url');
    return `${header}.${body}.fake-sig`;
  }

  it('extracts email and userId from JWT', () => {
    const token: TokenPair = {
      accessToken: makeJwt({ email: 'yaoqiyu@bytedance.com', sub: 'yaoqiyu@bytedance.com' }),
      refreshToken: 'r',
      expiresAt: 0,
    };
    const info = parseUserFromToken(token);
    expect(info.email).toBe('yaoqiyu@bytedance.com');
    expect(info.userId).toBe('yaoqiyu');
  });

  it('falls back to sub if no email', () => {
    const token: TokenPair = {
      accessToken: makeJwt({ sub: 'user123@example.com' }),
      refreshToken: 'r',
      expiresAt: 0,
    };
    const info = parseUserFromToken(token);
    expect(info.userId).toBe('user123');
  });

  it('returns empty object for malformed token', () => {
    const token: TokenPair = {
      accessToken: 'not-a-jwt',
      refreshToken: 'r',
      expiresAt: 0,
    };
    const info = parseUserFromToken(token);
    expect(info).toEqual({});
  });
});
