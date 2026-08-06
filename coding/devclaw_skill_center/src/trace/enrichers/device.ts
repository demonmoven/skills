import { readFile, writeFile, mkdir } from 'node:fs/promises';
import { join } from 'node:path';
import { homedir } from 'node:os';
import { randomUUID } from 'node:crypto';

const DEVICE_FILE = join(homedir(), '.trace', 'device.json');

let cachedClientId: string | null = null;

/**
 * Get or generate a stable device/client ID.
 * Generated once on first run, persisted to ~/.trace/device.json.
 */
export async function getClientId(): Promise<string> {
  if (cachedClientId) return cachedClientId;

  try {
    const content = await readFile(DEVICE_FILE, 'utf-8');
    const data = JSON.parse(content) as { clientId: string };
    if (data.clientId) {
      cachedClientId = data.clientId;
      return cachedClientId;
    }
  } catch {
    // File doesn't exist or is invalid, generate new
  }

  const clientId = randomUUID();
  await mkdir(join(homedir(), '.trace'), { recursive: true });
  await writeFile(DEVICE_FILE, JSON.stringify({ clientId }, null, 2), 'utf-8');
  cachedClientId = clientId;
  return clientId;
}

/**
 * Reset cached client ID (for testing).
 */
export function resetClientIdCache(): void {
  cachedClientId = null;
}

export { DEVICE_FILE };
