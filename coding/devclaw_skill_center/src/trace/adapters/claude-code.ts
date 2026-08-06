import { createReadStream } from 'node:fs';
import { createInterface } from 'node:readline';
import { basename } from 'node:path';
import type { Adapter, AdapterMeta } from './types.js';

/**
 * Claude Code adapter.
 *
 * Claude Code stores conversation transcripts as JSONL files in:
 *   ~/.claude/projects/<project-hash>/<session-id>.jsonl
 *
 * Each line is a JSON object with fields like:
 *   { "type": "user"|"assistant"|"summary", "sessionId": "...", "uuid": "...", ... }
 */
export const claudeCodeAdapter: Adapter = {
  name: 'claude-code',

  detect(filePath: string): boolean {
    // Path-based detection: contains .claude/projects/ or .claude/
    const normalized = filePath.replace(/\\/g, '/');
    return (
      normalized.includes('.claude/projects/') ||
      normalized.includes('/.claude/') && normalized.endsWith('.jsonl')
    );
  },

  async extractMeta(filePath: string): Promise<AdapterMeta> {
    const meta: AdapterMeta = {
      toolName: 'claude-code',
      sessionId: '',
    };

    // Try to get session ID from filename (Claude uses session ID as filename)
    const filename = basename(filePath, '.jsonl');
    if (filename && !filename.startsWith('.')) {
      meta.sessionId = filename;
    }

    // Read first few lines to extract metadata
    const rl = createInterface({
      input: createReadStream(filePath, { encoding: 'utf-8' }),
      crlfDelay: Infinity,
    });

    let lineCount = 0;
    const MAX_LINES = 20; // Only scan first 20 lines

    for await (const line of rl) {
      if (lineCount >= MAX_LINES) break;
      lineCount++;

      if (!line.trim()) continue;

      try {
        const entry = JSON.parse(line) as Record<string, unknown>;

        // Extract sessionId from JSONL content (overrides filename if present)
        if (entry['sessionId'] && typeof entry['sessionId'] === 'string') {
          meta.sessionId = entry['sessionId'];
        }

        // Extract timestamp from first entry
        if (!meta.startedAt && entry['timestamp'] && typeof entry['timestamp'] === 'string') {
          meta.startedAt = entry['timestamp'];
        }

        // Extract tool version if available
        if (entry['cliVersion'] && typeof entry['cliVersion'] === 'string') {
          meta.toolVersion = entry['cliVersion'];
        }

        // If we have all we need, stop early
        if (meta.sessionId && meta.startedAt && meta.toolVersion) break;
      } catch {
        // Skip malformed lines
        continue;
      }
    }

    rl.close();

    // Fallback: generate session ID from filename if not found in content
    if (!meta.sessionId) {
      meta.sessionId = `claude-${filename || Date.now()}`;
    }

    return meta;
  },
};
