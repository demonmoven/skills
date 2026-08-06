/**
 * OpenCode plugin for trace (formerly xtrace).
 *
 * Usage: add "@byted/xdex/trace/opencode" to the plugin array
 * in ~/.config/opencode/opencode.json
 */

import { writeFile, mkdir } from 'node:fs/promises';
import { join } from 'node:path';
import { execSync } from 'node:child_process';
import { tmpdir } from 'node:os';

const LOG_FILE = '/tmp/trace-opencode-hook.log';

function log(msg: string): void {
  const ts = new Date().toISOString();
  const line = `[${ts}] ${msg}\n`;
  writeFile(LOG_FILE, line, { flag: 'a' }).catch(() => {});
}

// Use a generic type to avoid depending on @opencode-ai/plugin at compile time.
// At runtime OpenCode provides the typed input.
type PluginInput = {
  client: {
    session: {
      messages(opts: { path: { id: string } }): Promise<{ data?: unknown[] }>;
    };
  };
};

type PluginEvent = {
  event: {
    type: string;
    properties: Record<string, unknown>;
  };
};

export const TracePlugin = async ({ client }: PluginInput) => {
  return {
    event: async ({ event }: PluginEvent) => {
      if (event.type === 'session.created') {
        const info = event.properties.info as { id: string } | undefined;
        const sessionId = info?.id;
        if (!sessionId) return;

        log(`session.created: ${sessionId}`);
        try {
          execSync(`xdev trace session-start --session-id "${sessionId}"`, {
            timeout: 10000,
            stdio: 'pipe',
          });
          log(`session-start OK: ${sessionId}`);
        } catch (err: unknown) {
          const message = err instanceof Error ? err.message : String(err);
          log(`session-start FAILED: ${message}`);
        }
      }

      if (event.type === 'session.idle') {
        const sessionId = event.properties.sessionID as string | undefined;
        if (!sessionId) return;

        log(`session.idle: ${sessionId}`);
        try {
          const res = await client.session.messages({ path: { id: sessionId } });
          const messages = res.data ?? [];

          if (messages.length === 0) {
            log(`no messages for session ${sessionId}, skipping`);
            return;
          }

          const tmpDir = join(tmpdir(), 'trace-opencode');
          await mkdir(tmpDir, { recursive: true });
          const jsonlPath = join(tmpDir, `${sessionId}.jsonl`);
          const lines = messages.map((msg: unknown) => JSON.stringify(msg)).join('\n') + '\n';
          await writeFile(jsonlPath, lines);
          log(`wrote ${messages.length} messages to ${jsonlPath}`);

          execSync(`xdev trace forward --file "${jsonlPath}" --source opencode`, {
            timeout: 30000,
            stdio: 'pipe',
          });
          log(`forward OK: ${sessionId}`);
        } catch (err: unknown) {
          const message = err instanceof Error ? err.message : String(err);
          log(`forward FAILED: ${message}`);
        }
      }
    },
  };
};

export default TracePlugin;
