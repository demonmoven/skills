import { readFile, writeFile, mkdir, chmod } from 'node:fs/promises';
import { parse as parseYaml, stringify as stringifyYaml } from 'yaml';
import { GLOBAL_CONFIG_DIR, GLOBAL_CONFIG_PATH } from '../config/loader.js';
import { loadConfig } from '../config/loader.js';
import { logger } from '../utils/logger.js';

/**
 * Set a config key in the global config file.
 * Supports dotted keys like "tos.bucket".
 */
export async function runConfigSet(key: string, value: string): Promise<void> {
  await mkdir(GLOBAL_CONFIG_DIR, { recursive: true });

  let config: Record<string, unknown> = {};
  try {
    const content = await readFile(GLOBAL_CONFIG_PATH, 'utf-8');
    config = (parseYaml(content) as Record<string, unknown>) ?? {};
  } catch {
    // File doesn't exist, start fresh
  }

  // Support dotted keys: "tos.bucket" → config.tos.bucket
  const parts = key.split('.');
  let current: Record<string, unknown> = config;
  for (let i = 0; i < parts.length - 1; i++) {
    const part = parts[i]!;
    if (typeof current[part] !== 'object' || current[part] === null) {
      current[part] = {};
    }
    current = current[part] as Record<string, unknown>;
  }

  // Parse value: try boolean, number, then string
  const lastKey = parts[parts.length - 1]!;
  if (value === 'true') {
    current[lastKey] = true;
  } else if (value === 'false') {
    current[lastKey] = false;
  } else if (!isNaN(Number(value)) && value.trim() !== '') {
    current[lastKey] = Number(value);
  } else {
    current[lastKey] = value;
  }

  await writeFile(GLOBAL_CONFIG_PATH, stringifyYaml(config), 'utf-8');
  await chmod(GLOBAL_CONFIG_PATH, 0o600);
  logger.info(`Set ${key} = ${value}`);
}

/**
 * Display the current effective config (merged defaults + global + project).
 */
export async function runConfigView(): Promise<void> {
  const config = await loadConfig();
  console.log(stringifyYaml(config));
}
