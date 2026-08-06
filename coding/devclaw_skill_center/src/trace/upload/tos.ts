import { readFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import type { UploadMeta, UploadResult } from './types.js';
import type { TosConfig } from '../config/schema.js';
import { logger } from '../utils/logger.js';

// @byted-service/tos is a CJS module, need createRequire for ESM compat
const require = createRequire(import.meta.url);

/**
 * Build the TOS object key path.
 * Format: xtrace/${user_id}/${yyyy-mm-dd}/${tool_name}/${session_id}${suffix}
 *
 * NOTE: The leading "xtrace/" segment is intentionally kept (not renamed to "trace/")
 * because it is part of the data contract with downstream consumers (Fornax /
 * analytics jobs that read the TOS bucket). Renaming it would split historical and
 * new data into two prefixes and break those consumers. The CLI itself is renamed
 * to `trace`, but the on-disk object layout stays stable.
 *
 * No leading slash — @byted-service/tos prepends /<bucket>/.
 */
export function buildObjectKey(meta: UploadMeta, suffix: string): string {
  // Use session start date if available, fallback to current date
  const date = meta.sessionStartedAt
    ? new Date(meta.sessionStartedAt).toISOString().slice(0, 10)
    : new Date().toISOString().slice(0, 10);
  const userId = meta.userId ?? 'anonymous';
  return `xtrace/${userId}/${date}/${meta.toolName}/${meta.sessionId}${suffix}`;
}

/**
 * Build metadata headers for TOS object.
 */
export function buildObjectMetadata(meta: UploadMeta): Record<string, string> {
  const result: Record<string, string> = {
    'x-tos-meta-trace-id': meta.traceId,
    'x-tos-meta-tool-name': meta.toolName,
    'x-tos-meta-session-id': meta.sessionId,
    'x-tos-meta-device-id': meta.deviceId,
    'x-tos-meta-cli-version': meta.cliVersion,
  };

  if (meta.userId) result['x-tos-meta-user-id'] = meta.userId;
  if (meta.nickname) result['x-tos-meta-nickname'] = encodeURIComponent(meta.nickname);
  if (meta.department) result['x-tos-meta-department'] = encodeURIComponent(meta.department);
  if (meta.gitUrl) result['x-tos-meta-git-url'] = meta.gitUrl;
  if (meta.gitBranch) result['x-tos-meta-git-branch'] = meta.gitBranch;
  if (meta.gitCommit) result['x-tos-meta-git-commit'] = meta.gitCommit;
  if (meta.workspaceRoot) result['x-tos-meta-workspace-root'] = meta.workspaceRoot;
  if (meta.repoDirty !== undefined) result['x-tos-meta-repo-dirty'] = String(meta.repoDirty);
  if (meta.gitStartCommit) result['x-tos-meta-git-start-commit'] = meta.gitStartCommit;

  return result;
}

/**
 * Generate a pre-signed URL for downloading a TOS object.
 * Requires config.endpoint to be set (BOE: tos-cn-north-boe.byted.org).
 * Returns a self-contained HTTPS/HTTP URL that any browser or curl can open
 * until the expiresSec deadline.
 */
export function generatePresignedUrl(
  config: TosConfig,
  objectKey: string,
  expiresSec: number = 24 * 3600,
  sigName: string = 'xdev-url',
): string {
  if (!config.endpoint) {
    throw new Error('tos.endpoint is not configured — set via `xdev trace config set tos.endpoint <endpoint>`');
  }
  const Tos = require('@byted-service/tos');
  const tos = new Tos({
    bucket: config.bucket,
    accessKey: config.accessKey,
    secretKey: config.secretKey,
    signature: true,
    disableWatcher: true,
    endpoints: config.endpoint,
  });
  try {
    return tos.signatureSubdomainLink({
      subdomain: config.endpoint,
      name: objectKey.replace(/^\//, ''),
      sigName,
      expiredAt: Math.floor(Date.now() / 1000) + expiresSec,
    });
  } finally {
    if (typeof tos.destroy === 'function') tos.destroy();
  }
}

/**
 * Upload a file to TOS using @byted-service/tos SDK.
 */
export async function uploadToTOS(
  filePath: string,
  objectKey: string,
  meta: UploadMeta,
  config: TosConfig,
): Promise<UploadResult> {
  const Tos = require('@byted-service/tos');

  const tos = new Tos({
    bucket: config.bucket,
    accessKey: config.accessKey,
    secretKey: config.secretKey,
    signatureVersion: 'sign_v1',
    disableWatcher: true,
    reqTimeout: 30000,
    ...(config.endpoint ? { endpoints: config.endpoint } : {}),
  });

  try {
    const fileContent = await readFile(filePath);
    const headers = buildObjectMetadata(meta);

    const result = await tos.upload(fileContent, objectKey, { headers });

    logger.debug(`TOS response: ${JSON.stringify(result)}`);

    const etag = result.headers?.etag ?? result.headers?.['x-tos-md5'] ?? '';

    return {
      objectKey: result.tosKey ?? `/${config.bucket}/${objectKey}`,
      bucket: config.bucket,
      etag,
    };
  } finally {
    tos.destroy();
  }
}
