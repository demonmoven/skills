import { copyFileSync, existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { homedir, platform as osPlatform } from 'node:os';
import { dirname, join } from 'node:path';
import { mergeClaudeSettings } from './merge.js';
import { loadTemplate } from './template.js';
import type { ClaudeSettings } from './types.js';

const ok = '\x1b[32m✅\x1b[0m';
const fail = '\x1b[31m❌\x1b[0m';
const point = '\x1b[35m👉\x1b[0m';

export interface SetupOptions {
  preferTemplate?: boolean;
  dryRun?: boolean;
}

export async function runClaudeSettingsSetup(opts: SetupOptions = {}): Promise<void> {
  console.log('[xdev] claude-settings mac setup');

  if (osPlatform() !== 'darwin') {
    console.error(`${fail} 仅支持 macOS（当前 platform: ${osPlatform()}）`);
    process.exit(1);
  }

  const target = join(homedir(), '.claude', 'settings.json');

  const { template, source } = loadTemplate();
  console.log(`  模板来源: ${source}`);

  const user: ClaudeSettings = existsSync(target)
    ? safeParseJson(target)
    : {};
  console.log(`  目标文件: ${target} ${existsSync(target) ? '(已存在)' : '(将创建)'}`);

  const { merged, report } = mergeClaudeSettings(user, template, {
    preferTemplate: opts.preferTemplate,
  });

  if (!report.changed) {
    console.log(`  ${ok} 已是合并完成状态，无变化`);
    return;
  }

  console.log(`  合并差异 (${report.changes.length} 项):`);
  for (const line of report.changes) {
    console.log(`     ${line}`);
  }

  if (opts.dryRun) {
    console.log(`  ${point} --dry-run 模式，不写文件。去掉 --dry-run 实际执行`);
    return;
  }

  // backup ONLY when target exists AND we're about to change it
  if (existsSync(target)) {
    const ts = new Date().toISOString().replace(/[:.]/g, '-');
    const backup = `${target}.xdev-bak.${ts}`;
    copyFileSync(target, backup);
    console.log(`  ${ok} 备份: ${backup}`);
  } else {
    const parentDir = dirname(target);
    if (!existsSync(parentDir)) mkdirSync(parentDir, { recursive: true });
  }

  writeFileSync(target, JSON.stringify(merged, null, 2) + '\n');
  console.log(`  ${ok} 已写入: ${target}`);
}

function safeParseJson(path: string): ClaudeSettings {
  const raw = readFileSync(path, 'utf-8').trim();
  if (!raw) return {};
  try {
    return JSON.parse(raw) as ClaudeSettings;
  } catch (e) {
    throw new Error(`无法解析 ${path}: ${(e as Error).message}（请手动修复 JSON 格式或备份后删除）`);
  }
}
