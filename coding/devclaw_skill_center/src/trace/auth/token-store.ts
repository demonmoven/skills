import { readFile, writeFile, unlink, mkdir, chmod } from 'node:fs/promises';
import { join } from 'node:path';
import { homedir } from 'node:os';
import type { TokenPair, UserInfo } from './types.js';

const TRACE_DIR = join(homedir(), '.trace');
const CREDENTIALS_PATH = join(TRACE_DIR, 'credentials.json');

export async function saveToken(token: TokenPair): Promise<void> {
  await mkdir(TRACE_DIR, { recursive: true });
  await writeFile(CREDENTIALS_PATH, JSON.stringify(token, null, 2), 'utf-8');
  await chmod(CREDENTIALS_PATH, 0o600);
}

export async function loadToken(): Promise<TokenPair | null> {
  try {
    const content = await readFile(CREDENTIALS_PATH, 'utf-8');
    return JSON.parse(content) as TokenPair;
  } catch (err: unknown) {
    if (err instanceof Error && 'code' in err && (err as NodeJS.ErrnoException).code === 'ENOENT') {
      return null;
    }
    throw err;
  }
}

export async function deleteToken(): Promise<void> {
  try {
    await unlink(CREDENTIALS_PATH);
  } catch (err: unknown) {
    if (err instanceof Error && 'code' in err && (err as NodeJS.ErrnoException).code === 'ENOENT') {
      return; // already gone
    }
    throw err;
  }
}

export function isTokenExpired(token: TokenPair): boolean {
  const now = Math.floor(Date.now() / 1000);
  // Consider expired 60 seconds before actual expiry for safety margin
  return now >= token.expiresAt - 60;
}

/**
 * Get user info from stored token.
 * Reads cached userInfo (saved during login), falls back to JWT parsing for legacy tokens.
 */
export function parseUserFromToken(token: TokenPair): UserInfo {
  // Prefer cached userInfo (populated during login via SSO userinfo endpoint)
  if (token.userInfo?.userId || token.userInfo?.email) {
    return token.userInfo;
  }

  // Fallback: try parsing as JWT (for legacy or JWT-based tokens)
  try {
    const parts = token.accessToken.split('.');
    if (parts.length !== 3) return {};
    const payload = JSON.parse(Buffer.from(parts[1]!, 'base64url').toString('utf-8'));
    const email: string | undefined = payload.email ?? payload.sub;
    if (!email) return {};
    const userId = email.includes('@') ? email.split('@')[0] : email;
    return { userId, email };
  } catch {
    return {};
  }
}

export { TRACE_DIR, CREDENTIALS_PATH };
