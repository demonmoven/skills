import type { RawJsonlLine } from './types.js';

/**
 * Parse JSONL text into an array of raw JSON objects.
 * Silently skips blank lines and malformed JSON lines.
 */
export function parseJsonl(content: string): RawJsonlLine[] {
  const lines = content.split('\n');
  const results: RawJsonlLine[] = [];

  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed) continue;
    try {
      results.push(JSON.parse(trimmed) as RawJsonlLine);
    } catch {
      // skip malformed lines
    }
  }

  return results;
}

/**
 * Parse an ISO timestamp string to a Date object.
 * Handles both 'Z' suffix and '+HH:MM' timezone offsets.
 */
export function parseTimestamp(ts: string): Date {
  let normalized = ts;
  if (normalized.endsWith('Z')) {
    normalized = normalized.slice(0, -1) + '+00:00';
  }
  return new Date(normalized);
}
