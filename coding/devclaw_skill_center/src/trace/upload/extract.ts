import { execFile as execFileCb } from 'node:child_process';
import { promisify } from 'node:util';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { readdir, readFile, writeFile, mkdir, rm } from 'node:fs/promises';
import { randomUUID } from 'node:crypto';
import type { SessionTree } from '../analyzer/types.js';
import { logger } from '../utils/logger.js';

const execFile = promisify(execFileCb);

/**
 * Extract a SessionTree from a session bundle (tar.gz buffer).
 *
 * Reads the main JSONL and all subagent files from the archive.
 * Skips tool-results/ files (archive-only, not analyzed).
 *
 * @param tarGzBuffer - Raw tar.gz content
 * @returns SessionTree with main JSONL and subagent data
 */
export async function extractSessionTree(tarGzBuffer: Buffer): Promise<SessionTree> {
  // Write buffer to temp file
  const extractDir = join(tmpdir(), `xtrace-extract-${randomUUID()}`);
  await mkdir(extractDir, { recursive: true });
  const tarPath = join(extractDir, 'session.tar.gz');
  await writeFile(tarPath, tarGzBuffer);

  // Extract tar.gz
  await execFile('tar', ['-xzf', tarPath, '-C', extractDir], {
    timeout: 60_000,
  });

  // Find the main JSONL file (top-level *.jsonl)
  const topEntries = await readdir(extractDir);
  const mainJsonlFile = topEntries.find(e => e.endsWith('.jsonl'));
  if (!mainJsonlFile) {
    await rm(extractDir, { recursive: true, force: true }).catch(() => {});
    throw new Error('No main JSONL file found in session bundle');
  }

  const sessionId = mainJsonlFile.replace('.jsonl', '');
  const mainJsonlContent = await readFile(join(extractDir, mainJsonlFile), 'utf-8');

  // Read subagent files (skip tool-results/)
  const subagents: SessionTree['subagents'] = [];
  const subagentsDir = join(extractDir, sessionId, 'subagents');

  try {
    const subEntries = await readdir(subagentsDir);
    const jsonlFiles = subEntries.filter(e => e.startsWith('agent-') && e.endsWith('.jsonl'));

    for (const jsonlFile of jsonlFiles) {
      const agentId = jsonlFile.slice('agent-'.length, -'.jsonl'.length);
      const jsonlContent = await readFile(join(subagentsDir, jsonlFile), 'utf-8');

      // Read paired meta.json if available
      let meta = { agentType: 'unknown', description: '' };
      const metaFile = `agent-${agentId}.meta.json`;
      if (subEntries.includes(metaFile)) {
        try {
          const raw = JSON.parse(await readFile(join(subagentsDir, metaFile), 'utf-8')) as Record<string, unknown>;
          meta = {
            agentType: (raw['agentType'] as string) ?? 'unknown',
            description: (raw['description'] as string) ?? '',
          };
        } catch { /* malformed meta */ }
      }

      subagents.push({ agentId, jsonlContent, meta });
    }
  } catch {
    // No subagents directory — that's fine
  }

  // Cleanup temp directory
  await rm(extractDir, { recursive: true, force: true }).catch(() => {});

  logger.info(`[extract] Extracted session bundle: main JSONL + ${subagents.length} subagents (tool-results skipped)`);
  return { mainJsonlContent, subagents };
}
