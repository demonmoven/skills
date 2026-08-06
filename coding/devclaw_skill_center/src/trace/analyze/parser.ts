import { readFile, readdir } from 'node:fs/promises';
import { join, dirname, basename, extname } from 'node:path';
import type { LogEntry, SubagentMeta, SubagentTrace } from './types.js';
import { logger } from '../utils/logger.js';

/**
 * Parse a JSONL file into LogEntry objects.
 *
 * Malformed lines are skipped with a warning; never throws for bad lines.
 */
export async function parseJsonl(filePath: string): Promise<LogEntry[]> {
  const raw = await readFile(filePath, 'utf-8');
  const lines = raw.split('\n');
  const entries: LogEntry[] = [];
  let skipped = 0;
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    if (!line || !line.trim()) continue;
    try {
      entries.push(JSON.parse(line) as LogEntry);
    } catch (err) {
      skipped++;
      logger.debug(`Malformed JSONL at ${filePath}:${i + 1}: ${(err as Error).message}`);
    }
  }
  if (skipped > 0) {
    logger.warn(`Skipped ${skipped} malformed line(s) in ${basename(filePath)}`);
  }
  return entries;
}

/**
 * Load subagent traces from the conventional layout:
 *   <dir>/<sessionId>.jsonl          — main
 *   <dir>/<sessionId>/subagents/     — subagent dir
 *     ├─ agent-<agentId>.jsonl       — subagent full trace
 *     └─ agent-<agentId>.meta.json   — { agentType, description }
 *
 * Returns an empty Map if no subagents directory exists. Does not throw.
 */
export async function loadSubagents(mainJsonlPath: string): Promise<Map<string, SubagentTrace>> {
  const subagents = new Map<string, SubagentTrace>();

  const baseName = basename(mainJsonlPath, extname(mainJsonlPath));
  const subagentsDir = join(dirname(mainJsonlPath), baseName, 'subagents');

  let files: string[];
  try {
    files = await readdir(subagentsDir);
  } catch {
    logger.debug(`No subagents directory at ${subagentsDir}`);
    return subagents;
  }

  for (const file of files) {
    if (!file.endsWith('.jsonl')) continue;
    const agentId = file.replace(/^agent-/, '').replace(/\.jsonl$/, '');
    const jsonlPath = join(subagentsDir, file);
    const metaPath = join(subagentsDir, `agent-${agentId}.meta.json`);

    try {
      const entries = await parseJsonl(jsonlPath);

      let meta: SubagentMeta = { agentType: 'unknown', description: '' };
      try {
        const metaRaw = await readFile(metaPath, 'utf-8');
        const parsed = JSON.parse(metaRaw) as { agentType?: unknown; description?: unknown };
        meta = {
          agentType: typeof parsed.agentType === 'string' ? parsed.agentType : 'unknown',
          description: typeof parsed.description === 'string' ? parsed.description : '',
        };
      } catch (err) {
        logger.debug(`No meta.json for subagent ${agentId}: ${(err as Error).message}`);
      }

      subagents.set(agentId, { agentId, meta, entries });
      logger.debug(`Loaded subagent ${agentId} (${meta.agentType}): ${entries.length} entries`);
    } catch (err) {
      logger.warn(`Failed to load subagent ${agentId}: ${(err as Error).message}`);
    }
  }

  return subagents;
}
