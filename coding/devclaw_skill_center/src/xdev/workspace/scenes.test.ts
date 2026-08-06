import { describe, expect, it } from 'vitest';
import { loadScene, listAvailableScenes } from './scenes.js';

describe('scenes', () => {
  it('loads the bundled doubao scene', () => {
    const scene = loadScene('doubao');
    expect(scene.scene).toBe('doubao');
    expect(scene.main_repo.url).toBe('git@code.byted.org:flow/creation_workspace.git');
    expect(scene.main_repo.dir).toBe('creation_workspace');
    expect(scene.sub_repos_dir).toBe('repos');
    expect(scene.sub_repos.length).toBe(14);
    expect(scene.sub_repos.find((r) => r.name === 'creation_agent')?.url)
      .toBe('git@code.byted.org:flow/creation_agent.git');
    expect(scene.sub_repos.map((r) => r.name)).not.toContain('creation_workspace');
  });

  it('listAvailableScenes includes doubao', () => {
    const scenes = listAvailableScenes();
    expect(scenes).toContain('doubao');
  });

  it('throws a helpful error for unknown scene', () => {
    expect(() => loadScene('nonexistent-scene-xyz')).toThrow(/未找到 scene/);
  });
});
