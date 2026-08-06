import { existsSync, readFileSync } from 'node:fs';
import { CLAUDE_SETTINGS_TEMPLATE_BUNDLED, CLAUDE_SETTINGS_TEMPLATE_USER } from '../paths.js';
import type { ClaudeSettings } from './types.js';

/**
 * Resolve the claude-settings template. User override
 * (~/.xdev/claude-settings/settings.template.json) wins over the bundled
 * copy that ships with the npm package.
 */
export function loadTemplate(): { template: ClaudeSettings; source: string } {
  for (const path of [CLAUDE_SETTINGS_TEMPLATE_USER, CLAUDE_SETTINGS_TEMPLATE_BUNDLED]) {
    if (!existsSync(path)) continue;
    const raw = readFileSync(path, 'utf-8');
    let parsed: ClaudeSettings;
    try {
      parsed = JSON.parse(raw) as ClaudeSettings;
    } catch (e) {
      throw new Error(`模板 JSON 解析失败 (${path}): ${(e as Error).message}`);
    }
    return { template: parsed, source: path };
  }
  throw new Error(
    `claude-settings 模板未找到，已搜索: ${CLAUDE_SETTINGS_TEMPLATE_USER}, ${CLAUDE_SETTINGS_TEMPLATE_BUNDLED}`,
  );
}
