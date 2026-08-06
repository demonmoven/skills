import { loadToken, saveToken, parseUserFromToken } from '../auth/token-store.js';
import { getSSOUserInfo } from '../auth/sso.js';
import { logger } from '../utils/logger.js';
import type { UserInfo } from '../auth/types.js';

/**
 * Get user info from stored SSO token.
 * If the cached token is missing department (legacy credentials), backfills it
 * by calling the SSO userinfo endpoint once and updating the stored token.
 * Returns empty UserInfo if not logged in, does not block on backfill failure.
 */
export async function getUserInfo(): Promise<UserInfo> {
  const token = await loadToken();
  if (!token) return {};

  const userInfo = parseUserFromToken(token);

  // Backfill department/nickname for tokens cached before these fields were added
  if (!userInfo.department && token.accessToken) {
    try {
      const ssoUser = await getSSOUserInfo(token.accessToken);
      userInfo.nickname = ssoUser.nickname;
      userInfo.department = ssoUser.department;
      token.userInfo = { ...token.userInfo, nickname: ssoUser.nickname, department: ssoUser.department };
      await saveToken(token);
      logger.debug('[user] Backfilled department/nickname into cached token');
    } catch {
      // Best-effort: don't block forward on backfill failure
    }
  }

  return userInfo;
}
