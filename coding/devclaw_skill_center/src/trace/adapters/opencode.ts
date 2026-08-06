import { createReadStream } from 'node:fs';
import { createInterface } from 'node:readline';
import { basename } from 'node:path';
import type { Adapter, AdapterMeta } from './types.js';

/**
 * OpenCode adapter.
 *
 * OpenCode stores session data in its working directory structure.
 * Detection primarily through --source flag or path patterns.
 */
export const opencodeAdapter: Adapter = {
  name: 'opencode',

  detect(filePath: string): boolean {
    const normalized = filePath.replace(/\\/g, '/');
    return (
      normalized.includes('opencode/') ||
      normalized.includes('.opencode/')
    ) && normalized.endsWith('.jsonl');
  },

  async extractMeta(filePath: string): Promise<AdapterMeta> {
    const meta: AdapterMeta = {
      toolName: 'opencode',
      sessionId: '',
    };

    // Try session ID from filename
    const filename = basename(filePath, '.jsonl');
    if (filename && !filename.startsWith('.')) {
      meta.sessionId = filename;
    }

    // Read first few lines
    const rl = createInterface({
      input: createReadStream(filePath, { encoding: 'utf-8' }),
      crlfDelay: Infinity,
    });

    let lineCount = 0;
    const MAX_LINES = 20;

    for await (const line of rl) {
      if (lineCount >= MAX_LINES) break;
      lineCount++;

      if (!line.trim()) continue;

      try {
        const entry = JSON.parse(line) as Record<string, unknown>;

        if (entry['sessionId'] && typeof entry['sessionId'] === 'string') {
          meta.sessionId = entry['sessionId'];
        }
        if (entry['session_id'] && typeof entry['session_id'] === 'string') {
          meta.sessionId = entry['session_id'];
        }

        if (!meta.startedAt && entry['timestamp'] && typeof entry['timestamp'] === 'string') {
          meta.startedAt = entry['timestamp'];
        }
        if (!meta.startedAt && entry['created_at'] && typeof entry['created_at'] === 'string') {
          meta.startedAt = entry['created_at'];
        }

        if (entry['version'] && typeof entry['version'] === 'string') {
          meta.toolVersion = entry['version'];
        }

        if (meta.sessionId && meta.startedAt) break;
      } catch {
        continue;
      }
    }

    rl.close();

    if (!meta.sessionId) {
      meta.sessionId = `opencode-${filename || Date.now()}`;
    }

    return meta;
  },
};
