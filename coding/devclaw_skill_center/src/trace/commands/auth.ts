import { startDeviceAuth, pollForToken, getSSOUserInfo } from '../auth/sso.js';
import { saveToken, loadToken, deleteToken, isTokenExpired, parseUserFromToken } from '../auth/token-store.js';
import { refreshAccessToken } from '../auth/sso.js';
import { openInBrowser, copyToClipboard } from '../utils/open.js';
import { logger } from '../utils/logger.js';
import type { TokenPair } from '../auth/types.js';

export async function runAuthLogin(): Promise<void> {
  logger.info('Starting Device Code SSO login...');

  const deviceAuth = await startDeviceAuth();

  // Prefer verification_uri_complete (RFC 8628: user_code pre-embedded in URL)
  const browserUrl = deviceAuth.verificationUriComplete ?? deviceAuth.verificationUri;
  await copyToClipboard(deviceAuth.userCode);
  openInBrowser(browserUrl);

  console.log();
  console.log(`Opening browser: ${browserUrl}`);
  if (deviceAuth.verificationUriComplete) {
    console.log(`Code: ${deviceAuth.userCode} (pre-filled in browser)`);
    console.log();
    console.log('Confirm authorization in the browser. Waiting...');
  } else {
    console.log(`Code copied to clipboard: ${deviceAuth.userCode}  (Cmd+V to paste)`);
    console.log();
    console.log('Paste the code and authorize in the browser. Waiting...');
  }

  const tokenResponse = await pollForToken(
    deviceAuth.deviceCode,
    deviceAuth.interval,
    deviceAuth.expiresIn,
  );

  const now = Math.floor(Date.now() / 1000);
  const tokenPair: TokenPair = {
    accessToken: tokenResponse.accessToken,
    refreshToken: tokenResponse.refreshToken,
    expiresAt: now + tokenResponse.expiresIn,
  };

  // Fetch user info from SSO userinfo endpoint
  try {
    const ssoUser = await getSSOUserInfo(tokenResponse.accessToken);
    const userId = ssoUser.email?.includes('@') ? ssoUser.email.split('@')[0] : ssoUser.username;
    tokenPair.userInfo = {
      userId: userId || ssoUser.username,
      email: ssoUser.email,
      nickname: ssoUser.nickname,
      department: ssoUser.department,
    };
    logger.info(`Login successful! User: ${tokenPair.userInfo.userId} (${ssoUser.nickname ?? ''}, ${ssoUser.department ?? ''})`);
  } catch (err) {
    logger.warn(`Could not fetch user info: ${err instanceof Error ? err.message : String(err)}`);
    logger.info('Login successful! (user info unavailable)');
  }

  await saveToken(tokenPair);
}

export async function runAuthStatus(): Promise<void> {
  const token = await loadToken();
  if (!token) {
    logger.info('Not logged in. Run `xdev trace auth login` to authenticate.');
    return;
  }

  const userInfo = parseUserFromToken(token);
  const expired = isTokenExpired(token);
  const expiresAt = new Date(token.expiresAt * 1000).toISOString();

  console.log(`User: ${userInfo.userId ?? userInfo.email ?? 'unknown'}`);
  console.log(`Token expires: ${expiresAt}`);
  console.log(`Status: ${expired ? 'EXPIRED' : 'VALID'}`);
}

export async function runAuthLogout(): Promise<void> {
  await deleteToken();
  logger.info('Logged out. Local credentials removed.');
}

/**
 * Ensure we have a valid (non-expired) token.
 * If expired, attempts to refresh automatically.
 * Returns the valid token or null if not authenticated.
 */
export async function ensureValidToken(): Promise<TokenPair | null> {
  const token = await loadToken();
  if (!token) return null;

  if (!isTokenExpired(token)) return token;

  // Try to refresh
  try {
    logger.debug('Access token expired, refreshing...');
    const refreshed = await refreshAccessToken(token.refreshToken);
    const now = Math.floor(Date.now() / 1000);
    const newToken: TokenPair = {
      accessToken: refreshed.accessToken,
      refreshToken: refreshed.refreshToken,
      expiresAt: now + refreshed.expiresIn,
    };
    await saveToken(newToken);
    logger.debug('Token refreshed successfully.');
    return newToken;
  } catch (err) {
    logger.warn(`Token refresh failed: ${err instanceof Error ? err.message : String(err)}`);
    return null;
  }
}

/**
 * Ensure the user is authenticated before launching an agent.
 *
 * Flow:
 *  1. Valid token exists → print user info, return.
 *  2. Token expired → attempt silent refresh → success → return.
 *  3. No token / refresh failed → interactive Device Code login.
 *  4. Login failed or non-TTY without valid token → process.exit(1).
 *
 * Called from runLaunch() so that all subsequent trace uploads carry a real userId.
 */
export async function ensureTraceAuth(): Promise<void> {
  // 1. Try loading existing token
  const token = await loadToken();

  if (token && !isTokenExpired(token)) {
    // Backfill department/nickname for tokens cached before this field was added
    if (!token.userInfo?.department) {
      try {
        const ssoUser = await getSSOUserInfo(token.accessToken);
        const userId = ssoUser.email?.includes('@') ? ssoUser.email.split('@')[0] : ssoUser.username;
        token.userInfo = {
          ...token.userInfo,
          userId: token.userInfo?.userId || userId || ssoUser.username,
          email: token.userInfo?.email || ssoUser.email,
          nickname: ssoUser.nickname,
          department: ssoUser.department,
        };
        await saveToken(token);
        logger.debug('Trace auth: backfilled department/nickname into cached token');
      } catch {
        // Best-effort: don't block on backfill failure
      }
    }

    const user = parseUserFromToken(token);
    console.log(`[xdev] Trace auth: logged in as ${user.userId ?? user.email ?? 'unknown'}`);
    return;
  }

  // 2. Token expired → silent refresh
  if (token) {
    try {
      logger.debug('Trace auth: token expired, attempting refresh...');
      const refreshed = await refreshAccessToken(token.refreshToken);
      const now = Math.floor(Date.now() / 1000);
      const newToken: TokenPair = {
        accessToken: refreshed.accessToken,
        refreshToken: refreshed.refreshToken,
        expiresAt: now + refreshed.expiresIn,
      };

      // Re-fetch user info in case the old token didn't have it cached
      try {
        const ssoUser = await getSSOUserInfo(refreshed.accessToken);
        const userId = ssoUser.email?.includes('@') ? ssoUser.email.split('@')[0] : ssoUser.username;
        newToken.userInfo = {
          userId: userId || ssoUser.username,
          email: ssoUser.email,
          nickname: ssoUser.nickname,
          department: ssoUser.department,
        };
      } catch {
        // Keep whatever userInfo was on the old token
        newToken.userInfo = token.userInfo;
      }

      await saveToken(newToken);
      const user = parseUserFromToken(newToken);
      console.log(`[xdev] Trace auth: token refreshed, logged in as ${user.userId ?? user.email ?? 'unknown'}`);
      return;
    } catch (err) {
      logger.debug(`Trace auth: refresh failed (${err instanceof Error ? err.message : String(err)}), need re-login`);
    }
  }

  // 3. No valid token → need interactive login
  if (!process.stdin.isTTY) {
    console.error('[xdev] Trace auth: not logged in and no TTY available for interactive login.');
    console.error('[xdev] Please run `xdev trace auth login` manually first, or use --no-trace-auth to skip.');
    process.exit(1);
  }

  console.log('[xdev] Trace auth: not logged in. Starting SSO login...');
  try {
    await runAuthLogin();
  } catch (err) {
    console.error(`[xdev] Trace auth: login failed — ${err instanceof Error ? err.message : String(err)}`);
    console.error('[xdev] Cannot start agent without trace authentication. Please retry or use --no-trace-auth to skip.');
    process.exit(1);
  }
}
