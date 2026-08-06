import { createRequire } from 'node:module';
import { readFile, writeFile, mkdir } from 'node:fs/promises';
import { join } from 'node:path';
import { homedir } from 'node:os';
import type { TosConfig } from '../config/schema.js';
import { logger } from '../utils/logger.js';

const require = createRequire(import.meta.url);

/** Shape of a user meta file stored in TOS at xtrace/_meta/users/{userId}.json */
export interface UserMetaRecord {
  userId: string;
  nickname?: string;
  department?: string;
  email?: string;
  updatedAt: string; // ISO date
}

// Local marker directory to avoid repeated TOS HEAD requests
const META_MARKER_DIR = join(homedir(), '.trace', 'meta-uploaded');

/**
 * Ensure the user's meta file exists in TOS.
 * Uses a local marker file to skip the TOS check on subsequent uploads.
 *
 * Flow:
 *   1. Check local marker (~/.trace/meta-uploaded/{userId})
 *   2. If marker exists → skip (already uploaded)
 *   3. If no marker → HEAD TOS to check if meta file exists
 *   4. If not on TOS → upload user meta JSON
 *   5. Write local marker
 */
export async function ensureUserMeta(
  config: TosConfig,
  userInfo: { userId?: string; nickname?: string; department?: string; email?: string },
): Promise<void> {
  const userId = userInfo.userId;
  if (!userId) return;

  // 1. Check local marker
  const markerPath = join(META_MARKER_DIR, userId);
  try {
    await readFile(markerPath, 'utf-8');
    logger.debug(`[user-meta] Local marker exists for ${userId}, skipping`);
    return;
  } catch {
    // no marker, continue
  }

  const objectKey = `xtrace/_meta/users/${userId}.json`;

  try {
    const Tos = require('@byted-service/tos');
    const tos = new Tos({
      bucket: config.bucket,
      accessKey: config.accessKey,
      secretKey: config.secretKey,
      signatureVersion: 'sign_v1',
      disableWatcher: true,
      reqTimeout: 10000,
    });

    try {
      // 2. HEAD check — does the meta file already exist on TOS?
      let exists = false;
      try {
        const headers = await tos.headObject(objectKey);
        exists = !!headers;
      } catch {
        // 404 or other error → treat as not existing
      }

      if (!exists) {
        // 3. Upload user meta JSON
        const record: UserMetaRecord = {
          userId,
          nickname: userInfo.nickname,
          department: userInfo.department,
          email: userInfo.email,
          updatedAt: new Date().toISOString().slice(0, 10),
        };
        const content = Buffer.from(JSON.stringify(record), 'utf-8');
        await tos.upload(content, objectKey, {
          headers: {
            'Content-Type': 'application/json',
            'x-tos-meta-type': 'user-meta',
          },
        });
        logger.info(`[user-meta] Uploaded meta for ${userId}`);
      } else {
        logger.debug(`[user-meta] Meta already exists on TOS for ${userId}`);
      }
    } finally {
      tos.destroy();
    }
  } catch (err) {
    // Best-effort: don't block the upload if meta write fails
    logger.debug(`[user-meta] Failed to ensure meta for ${userId}: ${err instanceof Error ? err.message : String(err)}`);
    return;
  }

  // 4. Write local marker (best-effort)
  try {
    await mkdir(META_MARKER_DIR, { recursive: true });
    await writeFile(markerPath, new Date().toISOString(), 'utf-8');
  } catch {
    // ignore marker write failures
  }
}
