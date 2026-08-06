import { existsSync, readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { SCENES_DIR_BUNDLED, SCENES_DIR_USER } from '../paths.js';
import type { SceneSpec } from './types.js';

/**
 * Resolve scene config by name. User override (~/.xdev/scenes) takes
 * precedence over the bundled copy shipped inside the npm package
 * (marketplace/plugins/xdev/scenes).
 */
export function loadScene(name: string): SceneSpec {
  const candidates = [
    join(SCENES_DIR_USER, `${name}.json`),
    join(SCENES_DIR_BUNDLED, `${name}.json`),
  ];
  for (const path of candidates) {
    if (!existsSync(path)) continue;
    const raw = readFileSync(path, 'utf-8');
    let parsed: SceneSpec;
    try {
      parsed = JSON.parse(raw) as SceneSpec;
    } catch (e) {
      throw new Error(`scene 文件 JSON 解析失败：${path}\n${(e as Error).message}`);
    }
    validateScene(parsed, path);
    return parsed;
  }
  const available = listAvailableScenes();
  const hint = available.length > 0
    ? `\n可用 scenes：${available.join(', ')}`
    : '';
  throw new Error(`未找到 scene '${name}'，已搜索：${candidates.join(', ')}${hint}`);
}

export function listAvailableScenes(): string[] {
  const seen = new Set<string>();
  for (const dir of [SCENES_DIR_USER, SCENES_DIR_BUNDLED]) {
    if (!existsSync(dir)) continue;
    for (const f of readdirSync(dir)) {
      if (f.endsWith('.json')) seen.add(f.replace(/\.json$/, ''));
    }
  }
  return Array.from(seen).sort();
}

function validateScene(scene: SceneSpec, path: string): void {
  const errs: string[] = [];
  if (!scene.scene) errs.push('缺少 scene 字段');
  if (!scene.main_repo?.url) errs.push('缺少 main_repo.url');
  if (!scene.main_repo?.dir) errs.push('缺少 main_repo.dir');
  if (!scene.sub_repos_dir) errs.push('缺少 sub_repos_dir');
  if (!Array.isArray(scene.sub_repos)) errs.push('sub_repos 必须是数组');
  else {
    scene.sub_repos.forEach((r, i) => {
      if (!r.name) errs.push(`sub_repos[${i}].name 缺失`);
      if (!r.url) errs.push(`sub_repos[${i}].url 缺失`);
    });
  }
  if (errs.length > 0) {
    throw new Error(`scene 文件校验失败：${path}\n  - ${errs.join('\n  - ')}`);
  }
}
