import type { DeviceAuthResponse, TokenResponse } from './types.js';
import { logger } from '../utils/logger.js';

// SSO endpoints - aligned with bytedcli's ByteDance SSO
const SSO_HOST = process.env['TRACE_SSO_HOST'] ?? 'https://sso.bytedance.com';
const CLIENT_ID = process.env['TRACE_SSO_CLIENT_ID'] ?? 'cd1k8uzbde1i1aa1gy0f';
const DEVICE_CODE_PATH = '/oauth2/device/code';
const TOKEN_PATH = '/oauth2/access_token';
const USERINFO_PATH = '/oauth2/userinfo';
const DEFAULT_SCOPES = ['read', 'ciam.device.read'];

// ByteCloud JWT (for TOS auth)
const BYTECLOUD_HOST = process.env['TRACE_BYTECLOUD_HOST'] ?? 'https://cloud.bytedance.net';
const JWT_PATH = '/auth/api/v1/jwt';

/**
 * Start Device Code authorization flow.
 */
export async function startDeviceAuth(): Promise<DeviceAuthResponse> {
  const url = `${SSO_HOST}${DEVICE_CODE_PATH}`;

  const body = new URLSearchParams({
    client_id: CLIENT_ID,
    scope: DEFAULT_SCOPES.join(' '),
  });

  const response = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: body.toString(),
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(`SSO device code request failed: ${response.status} ${text}`);
  }

  const data = await response.json() as Record<string, unknown>;
  return {
    deviceCode: data['device_code'] as string,
    userCode: data['user_code'] as string,
    verificationUri: data['verification_uri'] as string,
    verificationUriComplete: (data['verification_uri_complete'] as string) ?? undefined,
    expiresIn: (data['expires_in'] as number) ?? 300,
    interval: (data['interval'] as number) ?? 5,
  };
}

/**
 * Poll SSO service for token using device_code.
 */
export async function pollForToken(
  deviceCode: string,
  interval: number,
  expiresIn: number,
): Promise<TokenResponse> {
  const url = `${SSO_HOST}${TOKEN_PATH}`;
  const deadline = Date.now() + expiresIn * 1000;
  let pollInterval = interval;

  while (Date.now() < deadline) {
    await sleep(pollInterval * 1000);

    const body = new URLSearchParams({
      client_id: CLIENT_ID,
      device_code: deviceCode,
      grant_type: 'urn:ietf:params:oauth:grant-type:device_code',
    });

    const response = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: body.toString(),
    });

    if (response.ok) {
      const data = await response.json() as Record<string, unknown>;
      return {
        accessToken: data['access_token'] as string,
        refreshToken: (data['refresh_token'] as string) ?? '',
        expiresIn: (data['expires_in'] as number) ?? 3600,
      };
    }

    // Check error type
    let errorType = '';
    try {
      const errorData = await response.json() as Record<string, unknown>;
      errorType = (errorData['error'] as string) ?? '';
    } catch {
      const text = await response.text();
      if (text.includes('authorization_pending')) errorType = 'authorization_pending';
    }

    if (errorType === 'authorization_pending' || errorType.includes('pending')) {
      logger.debug('Waiting for user authorization...');
      continue;
    } else if (errorType === 'slow_down') {
      pollInterval = Math.min(pollInterval + 1, 10);
      continue;
    } else if (errorType === 'expired_token') {
      throw new Error('SSO authorization expired. Please try again.');
    } else {
      throw new Error(`SSO token polling failed: ${errorType || response.statusText}`);
    }
  }

  throw new Error('SSO authorization timed out. Please try again.');
}

/**
 * Refresh access token using refresh_token.
 */
export async function refreshAccessToken(refreshToken: string): Promise<TokenResponse> {
  const url = `${SSO_HOST}${TOKEN_PATH}`;
  const body = new URLSearchParams({
    client_id: CLIENT_ID,
    grant_type: 'refresh_token',
    refresh_token: refreshToken,
  });

  const response = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: body.toString(),
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(`Token refresh failed: ${response.status} ${text}`);
  }

  const data = await response.json() as Record<string, unknown>;
  return {
    accessToken: data['access_token'] as string,
    refreshToken: (data['refresh_token'] as string) ?? refreshToken,
    expiresIn: (data['expires_in'] as number) ?? 3600,
  };
}

/**
 * Get ByteCloud JWT using SSO access token.
 * This JWT is used to authenticate with internal services like TOS.
 */
export async function getBytecloudJwt(accessToken: string): Promise<string> {
  const url = `${BYTECLOUD_HOST}${JWT_PATH}?sso_access_token=${accessToken}&sso_client_id=${CLIENT_ID}`;

  const response = await fetch(url, {
    headers: { 'User-Agent': 'trace-cli/0.1.0' },
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(`ByteCloud JWT request failed: ${response.status} ${text}`);
  }

  // JWT may be in header or body
  const headerJwt = response.headers.get('x-jwt-token');
  if (headerJwt) return headerJwt;

  const bodyText = await response.text();
  try {
    const parsed = JSON.parse(bodyText) as Record<string, unknown>;
    const data = parsed['data'] as Record<string, unknown> | undefined;
    const jwt = (parsed['jwt'] as string) ?? (parsed['token'] as string) ?? (data?.['jwt'] as string) ?? (data?.['token'] as string);
    if (jwt) return jwt;
  } catch {
    // not JSON
  }

  throw new Error('ByteCloud JWT not found in response');
}

/**
 * Get user info from SSO using access token.
 */
export interface SSOUserInfo {
  email: string;
  username: string;
  nickname?: string;      // real name (e.g. "钟智伟")
  department?: string;    // department path (e.g. "产品研发和工程架构-Dev Infra-AI Platform-X")
  departmentEn?: string;  // english department path
}

export async function getSSOUserInfo(accessToken: string): Promise<SSOUserInfo> {
  const url = `${SSO_HOST}${USERINFO_PATH}`;

  const response = await fetch(url, {
    headers: { 'Authorization': `Bearer ${accessToken}` },
  });

  if (!response.ok) {
    throw new Error(`SSO userinfo request failed: ${response.status}`);
  }

  const data = await response.json() as Record<string, unknown>;
  const dept = data['department'] as Record<string, string> | undefined;
  return {
    email: (data['email'] as string) ?? '',
    username: (data['username'] as string) ?? (data['name'] as string) ?? '',
    nickname: (data['nickname'] as string) ?? undefined,
    department: dept?.['name'] ?? undefined,
    departmentEn: dept?.['en_name'] ?? undefined,
  };
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export { SSO_HOST, CLIENT_ID, BYTECLOUD_HOST };
