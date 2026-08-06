import { Command } from 'commander';
import { runWorkspaceCreate } from './create.js';
import { runWorkspaceInit } from './init.js';
import { listAvailableScenes } from './scenes.js';

interface CreateCliOptions {
  scene: string;
  parallelism?: string;
  skipSubRepos?: boolean;
}

interface InitCliOptions {
  branch: string;
  parallelism?: string;
}

export function registerWorkspaceCommand(parent: Command): void {
  const ws = parent
    .command('workspace')
    .description('跨仓库工作区管理（create / init）');

  ws.command('create')
    .description('按 scene 配置 clone 主仓库 + 并发 clone 全部子仓库 + scaffold 目录结构')
    .requiredOption('--scene <name>', `场景名（可用：${listAvailableScenes().join(', ') || '<尚无>'}）`)
    .option('--parallelism <n>', '子仓库并发 clone 数', '8')
    .option('--skip-sub-repos', '只 clone 主仓库 + scaffold，不 clone 子仓库')
    .action(async (opts: CreateCliOptions) => {
      await runWorkspaceCreate({
        scene: opts.scene,
        parallelism: opts.parallelism ? Number(opts.parallelism) : undefined,
        skipSubRepos: opts.skipSubRepos,
      });
    });

  ws.command('init')
    .description('在 workspace 根目录运行：主+子仓库 checkout -b <branch>，追加 .gitignore（.claude/ 等），commit 并并发 push -u origin <branch>')
    .requiredOption('--branch <name>', '目标分支名（已存在则切换，否则 -b 新建）')
    .option('--parallelism <n>', '子仓库并发数', '8')
    .action(async (opts: InitCliOptions) => {
      await runWorkspaceInit({
        branch: opts.branch,
        parallelism: opts.parallelism ? Number(opts.parallelism) : undefined,
      });
    });
}
