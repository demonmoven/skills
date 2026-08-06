import { readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { homedir } from 'node:os';
import { parse as parseYaml } from 'yaml';
import type { TraceConfig } from './schema.js';
import { defaultConfig } from './defaults.js';

const GLOBAL_CONFIG_DIR = join(homedir(), '.trace');
const GLOBAL_CONFIG_PATH = join(GLOBAL_CONFIG_DIR, 'config.yaml');
const PROJECT_CONFIG_DIR = '.trace';
const PROJECT_CONFIG_FILE = 'config.yaml';

/**
 * Deep merge two objects. `override` values take precedence over `base`.
 * Only merges plain objects; arrays and primitives from override replace base.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function deepMerge(base: any, override: any): any {
  const result = { ...base };
  for (const key of Object.keys(override)) {
    const baseVal = base[key];
    const overVal = override[key];
    if (
      baseVal !== null &&
      overVal !== null &&
      typeof baseVal === 'object' &&
      typeof overVal === 'object' &&
      !Array.isArray(baseVal) &&
      !Array.isArray(overVal)
    ) {
      result[key] = deepMerge(baseVal, overVal);
    } else if (overVal !== undefined) {
      result[key] = overVal;
    }
  }
  return result;
}

async function readYamlFile(filePath: string): Promise<Record<string, unknown> | null> {
  try {
    const content = await readFile(filePath, 'utf-8');
    const parsed: unknown = parseYaml(content);
    if (parsed !== null && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
    return null;
  } catch (err: unknown) {
    if (err instanceof Error && 'code' in err && (err as NodeJS.ErrnoException).code === 'ENOENT') {
      return null;
    }
    throw err;
  }
}

/**
 * Load trace configuration with layered merging:
 * defaults → global (~/.trace/config.yaml) → project (.trace/config.yaml)
 *
 * @param cwd - project directory to look for project-level config (defaults to process.cwd())
 */
export async function loadConfig(cwd?: string): Promise<TraceConfig> {
  const projectDir = cwd ?? process.cwd();

  let config: TraceConfig = { ...defaultConfig, tos: { ...defaultConfig.tos }, privacy: { ...defaultConfig.privacy, allowed_dirs: [...defaultConfig.privacy.allowed_dirs] } };

  const globalData = await readYamlFile(GLOBAL_CONFIG_PATH);
  if (globalData) {
    config = deepMerge(config, globalData) as TraceConfig;
  }

  const projectConfigPath = join(projectDir, PROJECT_CONFIG_DIR, PROJECT_CONFIG_FILE);
  const projectData = await readYamlFile(projectConfigPath);
  if (projectData) {
    config = deepMerge(config, projectData) as TraceConfig;
  }

  // Environment variables override config file (highest priority)
  if (process.env['TRACE_TOS_BUCKET']) config.tos.bucket = process.env['TRACE_TOS_BUCKET'];
  if (process.env['TRACE_TOS_REGION']) config.tos.region = process.env['TRACE_TOS_REGION'];
  if (process.env['TRACE_TOS_AK']) config.tos.accessKey = process.env['TRACE_TOS_AK'];
  if (process.env['TRACE_TOS_SK']) config.tos.secretKey = process.env['TRACE_TOS_SK'];

  return config;
}

export { GLOBAL_CONFIG_DIR, GLOBAL_CONFIG_PATH };
