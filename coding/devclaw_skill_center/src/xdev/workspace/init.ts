import { existsSync, readdirSync, statSync } from 'node:fs';
import { basename, join } from 'node:path';
import { gitSpawn, runConcurrent, summarizeResults } from './git-spawn.js';
import { appendGitignoreLines } from './gitignore-edit.js';

export interface WorkspaceInitOptions {
  branch: string;
  parallelism?: number;
}

const DEFAULT_PARALLELISM = 8;

export async function runWorkspaceInit(opts: WorkspaceInitOptions): Promise<void> {
  const root = process.cwd();
  const parallelism = opts.parallelism ?? DEFAULT_PARALLELISM;
  const branch = opts.branch;

  console.log(`[xdev] workspace init — branch: ${branch}`);
  console.log(`       工作区根: ${root}`);

  // ─── Validate workspace shape ────────────────────────────────────────
  if (!existsSync(join(root, '.git'))) {
    console.error(`✗ 当前目录不是 git 仓库（缺 .git/）。请在 workspace 根目录运行（即 xdev workspace create 创建出来的主仓库目录）。`);
    process.exit(1);
  }
  const reposDir = join(root, 'repos');
  if (!existsSync(reposDir)) {
    console.error(`✗ 缺 repos/ 子目录。请在 workspace 根目录运行。`);
    process.exit(1);
  }

  const subRepos: { name: string; cwd: string }[] = readdirSync(reposDir)
    .filter((n) => {
      const p = join(reposDir, n);
      return statSync(p).isDirectory() && existsSync(join(p, '.git'));
    })
    .map((n) => ({ name: n, cwd: join(reposDir, n) }));

  console.log(`       检测到 ${subRepos.length} 个子仓库：${subRepos.map((r) => r.name).join(', ') || '<无>'}`);

  const allRepos: { name: string; cwd: string }[] = [
    { name: `<main:${basename(root)}>`, cwd: root },
    ...subRepos,
  ];

  // ─── Step 1/4: checkout branch ───────────────────────────────────────
  console.log(`[1/4] checkout -b ${branch}（已存在则切换）`);
  const checkoutTasks = allRepos.map((repo) => async () => {
    await checkoutBranchSafe(repo.cwd, branch);
  });
  const checkoutResults = await runConcurrent(checkoutTasks, parallelism);
  const checkoutFailed = summarizeResults(checkoutResults, allRepos.map((r) => r.name), 'checkout');
  if (checkoutFailed > 0) {
    console.error(`✗ 中止：checkout 阶段有仓库失败。`);
    process.exit(1);
  }

  // ─── Step 2/4: .gitignore edits ──────────────────────────────────────
  console.log(`[2/4] 追加 .gitignore`);
  const mainAdded = appendGitignoreLines(root, ['.claude/', 'repos/']);
  console.log(`      [main] +${mainAdded.length} 行 ${mainAdded.length > 0 ? '(' + mainAdded.join(', ') + ')' : '(无变化)'}`);
  for (const sub of subRepos) {
    const added = appendGitignoreLines(sub.cwd, ['.claude/']);
    console.log(`      [${sub.name}] +${added.length} 行 ${added.length > 0 ? '(' + added.join(', ') + ')' : '(无变化)'}`);
  }

  // ─── Step 3/4: stage + commit (skip clean repos) ─────────────────────
  // Main repo commits ALL changes (scaffold files from `workspace create` +
  // .gitignore). The .gitignore we just wrote excludes repos/ so sub repo
  // working trees never get swept up. Sub repos only commit their own
  // .gitignore addition to keep the change set tight.
  console.log(`[3/4] commit 变更（无变更则跳过）`);
  const mainMsg = `chore: workspace init ${branch} — 初始化 workspace（.gitignore + AGENTS.md / docs / scaffold，repos/ 已排除）`;
  const subMsg = `chore: workspace init ${branch} — 初始化 .gitignore（.claude/ 排除）`;
  const commitTasks: Array<() => Promise<string>> = [
    () => commitAllIfDirty(root, mainMsg),
    ...subRepos.map((sub) => () => commitIfDirty(sub.cwd, '.gitignore', subMsg)),
  ];
  const commitResults = await runConcurrent(commitTasks, parallelism);
  summarizeResults(commitResults, allRepos.map((r) => r.name), 'commit');

  // ─── Step 4/4: push -u origin <branch> ───────────────────────────────
  console.log(`[4/4] push -u origin ${branch}（并发，失败汇总，不中断）`);
  const pushTasks = allRepos.map((repo) => async () => {
    const r = await gitSpawn(['push', '-u', 'origin', branch], repo.cwd);
    if (r.exitCode !== 0) {
      throw new Error(
        `push 失败 [exit ${r.exitCode}]：${(r.stderr || r.stdout).split('\n').slice(0, 3).join(' / ')}`,
      );
    }
    return 'pushed';
  });
  const pushResults = await runConcurrent(pushTasks, parallelism);
  const pushFailed = summarizeResults(pushResults, allRepos.map((r) => r.name), 'push');
  if (pushFailed > 0) {
    console.error(`⚠ ${pushFailed} 个仓库 push 失败 — 常见原因：远端无该仓库（main 仓库需先在 web 端建好）/ 无 push 权限 / 远端 hook 拒绝。`);
    process.exitCode = 1;
  }

  console.log(`[xdev] workspace init done.`);
}

/**
 * Checkout `branch`. If a local branch with that name already exists, switch
 * to it (idempotent re-run); otherwise create it (`-b`). Throws on any other
 * git error.
 */
async function checkoutBranchSafe(cwd: string, branch: string): Promise<void> {
  const exists = await gitSpawn(['rev-parse', '--verify', `refs/heads/${branch}`], cwd);
  if (exists.exitCode === 0) {
    const r = await gitSpawn(['checkout', branch], cwd);
    if (r.exitCode !== 0) throw new Error(`checkout 失败：${r.stderr.trim() || r.stdout.trim()}`);
  } else {
    const r = await gitSpawn(['checkout', '-b', branch], cwd);
    if (r.exitCode !== 0) throw new Error(`checkout -b 失败：${r.stderr.trim() || r.stdout.trim()}`);
  }
}

/**
 * Stage all changes (`git add .`) and commit. Used for the main repo so
 * scaffold files created by `workspace create` get committed alongside the
 * .gitignore additions. The .gitignore excludes `repos/`, so sub repo
 * working trees are not pulled in.
 */
async function commitAllIfDirty(cwd: string, msg: string): Promise<string> {
  const status = await gitSpawn(['status', '--porcelain'], cwd);
  if (status.exitCode !== 0) {
    throw new Error(`git status 失败：${status.stderr.trim() || status.stdout.trim()}`);
  }
  if (!status.stdout.trim()) return 'skipped (clean)';
  const add = await gitSpawn(['add', '.'], cwd);
  if (add.exitCode !== 0) throw new Error(`git add . 失败：${add.stderr.trim()}`);
  const commit = await gitSpawn(['commit', '-m', msg], cwd);
  if (commit.exitCode !== 0) throw new Error(`git commit 失败：${commit.stderr.trim() || commit.stdout.trim()}`);
  return 'committed';
}

/**
 * Stage `file` and commit with `msg` if and only if there is something to
 * commit (file appears in git status --porcelain). Idempotent on re-run:
 * if the working tree is clean, returns 'skipped' without erroring.
 */
async function commitIfDirty(cwd: string, file: string, msg: string): Promise<string> {
  const status = await gitSpawn(['status', '--porcelain', '--', file], cwd);
  if (status.exitCode !== 0) {
    throw new Error(`git status 失败：${status.stderr.trim() || status.stdout.trim()}`);
  }
  if (!status.stdout.trim()) return 'skipped (clean)';
  const add = await gitSpawn(['add', file], cwd);
  if (add.exitCode !== 0) throw new Error(`git add 失败：${add.stderr.trim()}`);
  const commit = await gitSpawn(['commit', '-m', msg], cwd);
  if (commit.exitCode !== 0) throw new Error(`git commit 失败：${commit.stderr.trim() || commit.stdout.trim()}`);
  return 'committed';
}
