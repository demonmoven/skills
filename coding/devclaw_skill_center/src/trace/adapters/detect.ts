import type { Adapter, ToolName } from './types.js';
import { claudeCodeAdapter } from './claude-code.js';
import { opencodeAdapter } from './opencode.js';

/** All registered adapters, ordered by detection priority */
const adapters: Adapter[] = [
  claudeCodeAdapter,
  opencodeAdapter,
];

/**
 * Detect which tool produced the given file.
 *
 * @param filePath - path to the JSONL file
 * @param source - optional explicit tool name (from --source flag)
 * @returns the matching adapter, or null if no match
 */
export function detectAdapter(filePath: string, source?: string): Adapter | null {
  // If source is explicitly specified, find the matching adapter
  if (source) {
    const adapter = adapters.find((a) => a.name === source);
    return adapter ?? null;
  }

  // Auto-detect by trying each adapter's detect() in priority order
  for (const adapter of adapters) {
    if (adapter.detect(filePath)) {
      return adapter;
    }
  }

  return null;
}

/**
 * Get adapter by tool name.
 */
export function getAdapterByName(name: ToolName): Adapter | null {
  return adapters.find((a) => a.name === name) ?? null;
}

export { adapters };
