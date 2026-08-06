import { existsSync, mkdirSync } from 'node:fs';
import { join } from 'node:path';
import { gitOrThrow, gitSpawn, runConcurrent, summarizeResults } from './git-spawn.js';
import { createWorkspaceScaffold } from './scaffold.js';
import { loadScene } from './scenes.js';

export interface WorkspaceCreateOptions {
  scene: string;
  parallelism?: number;
  skipSubRepos?: boolean;
}

const DEFAULT_PARALLELISM = 8;

export async function runWorkspaceCreate(opts: WorkspaceCreateOptions): Promise<void> {
  const cwd = process.cwd();
  const scene = loadScene(opts.scene);
  const parallelism = opts.parallelism ?? DEFAULT_PARALLELISM;

  console.log(`[xdev] workspace create — scene: ${scene.scene}`);
  console.log(`       工作目录: ${cwd}`);

  // Step 1/3: clone main repo
  const mainDst = join(cwd, scene.main_repo.dir);
  if (existsSync(mainDst)) {
    console.error(`✗ 主仓库目录已存在: ${mainDst}（拒绝覆盖，请先清理）`);
    process.exit(1);
  }
  console.log(`[1/3] clone 主仓库 ${scene.main_repo.url} → ./${scene.main_repo.dir}`);
  // inherit stdout so user sees git's progress bar; gitOrThrow will surface
  // exit code != 0 as a thrown error with the spawned cmd recorded.
  const mainResult = await gitSpawn(
    ['clone', scene.main_repo.url, mainDst],
    cwd,
    { inheritStdout: true },
  );
  if (mainResult.exitCode !== 0) {
    console.error(`✗ 主仓库 clone 失败 (exit ${mainResult.exitCode})。常见原因：远端仓库未创建 / SSH key 无权限 / 网络问题。`);
    process.exit(1);
  }

  // Step 2/3: scaffold workspace skeleton (in main repo root, BEFORE sub repos
  // populate `repos/`, so any path collision is loud and obvious).
  console.log(`[2/3] scaffold 目录结构`);
  const created = createWorkspaceScaffold(mainDst, scene.scaffold ?? {});
  for (const path of created) {
    console.log(`      + ${path}`);
  }
  if (created.length === 0) {
    console.log(`      （所有占位文件均已存在，跳过）`);
  }

  // Step 3/3: concurrent clone of sub repos
  if (opts.skipSubRepos) {
    console.log(`[3/3] 跳过 sub_repos clone（--skip-sub-repos）`);
  } else {
    const subDir = join(mainDst, scene.sub_repos_dir);
    mkdirSync(subDir, { recursive: true });
    console.log(`[3/3] 并发 clone ${scene.sub_repos.length} 个子仓库 → ./${scene.main_repo.dir}/${scene.sub_repos_dir}/ (parallelism=${parallelism})`);

    const tasks = scene.sub_repos.map((repo) => async () => {
      const dst = join(subDir, repo.name);
      if (existsSync(dst)) {
        // already cloned (re-run case): skip without erroring
        console.log(`      ↺ ${repo.name} 已存在，跳过`);
        return;
      }
      await gitOrThrow(['clone', repo.url, dst], subDir);
      console.log(`      ✓ ${repo.name}`);
    });

    const results = await runConcurrent(tasks, parallelism);
    const failed = summarizeResults(results, scene.sub_repos.map((r) => r.name), 'sub_repos clone');
    if (failed > 0) process.exitCode = 1;
  }

  console.log(`[xdev] Done. 工作区根: ${mainDst}`);
  console.log(`下一步: cd ${scene.main_repo.dir} && xdev workspace init --branch <your-branch>`);
}
