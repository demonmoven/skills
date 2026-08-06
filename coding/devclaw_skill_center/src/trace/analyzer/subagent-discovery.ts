import { readdir, readFile, stat } from 'node:fs/promises';
import { join, dirname, basename } from 'node:path';
import type { SessionTree } from './types.js';

// ─── Types ────────────────────────────────────────────────────────

export interface SubagentFileInfo {
  agentId: string;
  jsonlPath: string;
  metaPath: string | null;
  meta: { agentType: string; description: string };
}

// ─── Discovery Functions ──────────────────────────────────────────

/**
 * Given a main JSONL file path, check if a sibling session directory exists.
 * Claude Code stores session data at:
 *   <project-dir>/<session-id>.jsonl     (main session)
 *   <project-dir>/<session-id>/          (session directory)
 *
 * @returns The session directory path, or null if it doesn't exist
 */
export async function discoverSessionDir(mainJsonlPath: string): Promise<string | null> {
  const sessionId = basename(mainJsonlPath, '.jsonl');
  const sessionDir = join(dirname(mainJsonlPath), sessionId);

  try {
    const dirStat = await stat(sessionDir);
    return dirStat.isDirectory() ? sessionDir : null;
  } catch {
    return null;
  }
}

/**
 * List all subagent JSONL files within a session directory.
 * Looks for `subagents/agent-*.jsonl` and paired `agent-*.meta.json`.
 */
export async function discoverSubagents(sessionDir: string): Promise<SubagentFileInfo[]> {
  const subagentsDir = join(sessionDir, 'subagents');

  let entries: string[];
  try {
    entries = await readdir(subagentsDir);
  } catch {
    return []; // subagents/ directory doesn't exist
  }

  const jsonlFiles = entries.filter(e => e.startsWith('agent-') && e.endsWith('.jsonl'));
  const results: SubagentFileInfo[] = [];

  for (const jsonlFile of jsonlFiles) {
    // Extract agentId from filename: agent-<agentId>.jsonl
    const agentId = jsonlFile.slice('agent-'.length, -'.jsonl'.length);
    const jsonlPath = join(subagentsDir, jsonlFile);
    const metaFile = `agent-${agentId}.meta.json`;
    const metaPath = entries.includes(metaFile) ? join(subagentsDir, metaFile) : null;

    // Read meta.json if available
    let meta = { agentType: 'unknown', description: '' };
    if (metaPath) {
      try {
        const raw = await readFile(metaPath, 'utf-8');
        const parsed = JSON.parse(raw) as Record<string, unknown>;
        meta = {
          agentType: (parsed['agentType'] as string) ?? 'unknown',
          description: (parsed['description'] as string) ?? '',
        };
      } catch {
        // malformed meta.json — use defaults
      }
    }

    results.push({ agentId, jsonlPath, metaPath, meta });
  }

  return results;
}

/**
 * Build a SessionTree from a main JSONL file path.
 * Discovers the session directory and reads all subagent files.
 *
 * Graceful degradation: if no session directory exists, returns a
 * SessionTree with only the main JSONL content (zero subagents).
 */
export async function buildSessionTree(mainJsonlPath: string): Promise<SessionTree> {
  const mainJsonlContent = await readFile(mainJsonlPath, 'utf-8');

  const sessionDir = await discoverSessionDir(mainJsonlPath);
  if (!sessionDir) {
    return { mainJsonlContent, subagents: [] };
  }

  const subagentFiles = await discoverSubagents(sessionDir);
  const subagents: SessionTree['subagents'] = [];

  for (const sub of subagentFiles) {
    try {
      const jsonlContent = await readFile(sub.jsonlPath, 'utf-8');
      subagents.push({
        agentId: sub.agentId,
        jsonlContent,
        meta: sub.meta,
      });
    } catch {
      // skip unreadable subagent files
    }
  }

  return { mainJsonlContent, subagents };
}
