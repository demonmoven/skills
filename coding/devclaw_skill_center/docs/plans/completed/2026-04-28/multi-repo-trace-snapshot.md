# 多仓库 Trace Snapshot — 数据采集侧改造

本 ExecPlan 是一份活文档。Progress、Surprises & Discoveries、Decision Log 和 Outcomes & Retrospective 章节必须随工作推进持续更新。

**创建时代码基线：**
- 分支：opencode/stellar-garden
- Commit SHA：89c51223668cae774356bd4d000d455558077b2a


## Purpose / Big Picture

当前 trace 系统只记录 `cwd` 所在的单个 git 仓库的起始 commit 和 diff。在「中心化文档仓库 + repos/ 子仓库」或「普通目录 + 多个 git 仓库」的场景下，子仓库的起始 commit 完全丢失。

改造完成后，trace 系统能够：
1. 在 session 开始时发现并记录工作区下所有 git 仓库的起始 commit、branch、dirty 状态
2. 在 session 结束（forward）时为每个仓库计算 diff，上传结构化的 manifest 和 diffs 文件
3. 后续可以根据 manifest 中的 startCommit 对每个仓库执行 `git checkout {startCommit}` 恢复精确的起始状态，配合 transcript 重放任务

验证方式：用 `xdev trace forward --dry-run` 在多仓库目录下运行，检查输出的 `.manifest.json.gz` 和 `.diffs.json.gz` 文件内容是否包含所有子仓库的信息。


## Progress

- [x] (2026-04-28 17:20:00+08:00) Milestone 1: 数据结构与配置扩展
- [x] (2026-04-28 17:22:00+08:00) Milestone 2: 子仓库发现机制
- [x] (2026-04-28 17:23:00+08:00) Milestone 3: session-start 多仓库快照采集
- [x] (2026-04-28 17:25:00+08:00) Milestone 4: forward 多仓库 diff 采集与上传
- [x] (2026-04-28 17:27:00+08:00) Milestone 5: 集成测试与 dry-run 验证
- [x] (2026-04-28 17:30:00+08:00) Milestone 6: 文档更新


## Surprises & Discoveries

- 观察：`npx tsc --noEmit` 报出 4 个错误，全部在 `src/xdev/workspace/git-spawn.ts` 中，是预先存在的类型问题（readonly tuple 不兼容 StdioOptions），与本次改动无关。
  证据：错误信息中的文件路径和行号均在 git-spawn.ts:29-34，本次只改了 trace/ 目录下的文件。


## Decision Log

- 决策：manifest 和 diffs 拆为两个独立文件上传
  理由：manifest 是轻量元数据（几百字节），重放时只需读 manifest 就能拿到各仓库 startCommit，不需要下载可能很大的 diff 内容。分开后支持按需消费。
  日期/作者：2026-04-28 / 用户

- 决策：子仓库发现采用「配置优先 + 自动扫描兜底」
  理由：有配置时精确可控；无配置时兜底扫描保证开箱即用。扫描深度最多 2 层，避免扫描过深。
  日期/作者：2026-04-28 / 用户

- 决策：保留现有 `.diff.gz` 上传逻辑不变
  理由：当前 `.diff.gz` 没有消费者做格式解析（web dashboard 刻意排除，download/open/get 只做二进制透传），所以保持原有行为不会产生兼容问题。新增的 manifest 和 diffs 是增量产物。
  日期/作者：2026-04-28 / 分析代码后确认

- 决策：不在 snapshot 中记录 endCommit
  理由：session 期间用户可能只做了修改但没有 git commit，此时 endCommit 等于 startCommit，没有信息量。真正的变更通过 `git diff {startCommit}`（包含 uncommitted changes）来记录。
  日期/作者：2026-04-28 / 用户


## Outcomes & Retrospective

全部 6 个里程碑完成。

**成果**：
- 新增 `src/trace/enrichers/repo-discovery.ts`：子仓库发现机制（配置优先 + 自动扫描兜底，最多 2 层）
- 扩展 `SessionState` 支持 `repoSnapshots`，`TraceConfig` 支持 `workspace.repos`
- `session-start` 命令在启动时自动发现并记录所有子仓库的 startCommit/branch/dirty 状态
- `forward` 命令构建并上传 `manifest.json.gz`（元数据）和 `diffs.json.gz`（各仓库 diff），同时在 `UploadMeta` 中新增 `gitStartCommit`
- 所有改动向后兼容，现有 `.diff.gz` 上传逻辑不变
- 新增 12 个测试用例，总测试从 263 增至 275，全部 PASS

**经验**：
- 现有 forward.test.ts 的 mock 需要同步更新（`loadSessionState`、`collectMultiRepoDiffs`），这是添加新依赖时常见的遗漏
- `runGit` 从 file-private 改为 exported 是正确的决策，避免了在新文件中重复实现 git 命令执行逻辑


## Context and Orientation

本仓库名为 `devclaw_skills_center`（别名 `stellar-garden`），是 xdev CLI 和 Agent Skills 的中心仓库。技术栈为 TypeScript + esbuild + vitest。

trace 子系统位于 `src/trace/`，负责 AI coding session 的 trace 采集和上报。核心流程：

1. coding CLI（Claude Code / OpenCode / Coco）启动 session 时触发 hook，调用 `xdev trace session-start --session-id <ID>`
2. session-start 命令记录当前 git 仓库的 HEAD commit，保存到 `~/.trace/sessions/{sessionId}.json`
3. session 结束时触发 hook，调用 `xdev trace forward --file <JSONL_PATH>`
4. forward 命令收集 git context（branch、commit、diff）、user info、device ID，将 JSONL 和 diff 上传到 TOS

关键文件路径（均相对于仓库根目录）：

    src/trace/config/schema.ts        — TraceConfig、TosConfig 等类型定义
    src/trace/config/loader.ts        — 分层配置加载：defaults → ~/.trace/config.yaml → .trace/config.yaml → env
    src/trace/config/defaults.ts      — 默认配置值
    src/trace/session/state.ts        — SessionState 类型和持久化（~/.trace/sessions/*.json）
    src/trace/commands/session-start.ts — session-start 命令实现
    src/trace/commands/forward.ts     — forward 命令实现（enrichment + compress + upload）
    src/trace/enrichers/git.ts        — git 上下文采集（repo URL、branch、commit、dirty、diff）
    src/trace/upload/types.ts         — UploadMeta、ForwardResult 等上传类型
    src/trace/upload/tos.ts           — TOS 上传、object key 构建、metadata 构建
    src/trace/utils/hook-log.ts       — hook 日志记录（HookLogEntry 类型）
    src/trace/utils/compress.ts       — gzip 压缩工具函数
    src/trace/cli.ts                  — CLI 命令注册

已有测试文件（vitest）：

    src/trace/enrichers/git.test.ts   — collectGitContext 的单元测试，mock child_process
    src/trace/session/state.test.ts   — SessionState 持久化测试
    src/trace/upload/tos.test.ts      — buildObjectKey、buildObjectMetadata 测试
    src/trace/config/loader.test.ts   — 配置加载测试
    src/trace/commands/forward.test.ts — forward 命令测试

三种目标场景：

    场景 A — 传统单仓库：cwd 本身是 git repo，无子仓库。repos 数组只包含 relativePath="."。
    场景 B — 中心化文档仓库 + repos/：cwd 是 git repo，repos/ 下有多个独立 git 仓库。
    场景 C — 普通目录 + 多仓库：cwd 不是 git repo，子目录下有多个 git 仓库。


## Plan of Work

改造分为 6 个里程碑，按顺序执行。每个里程碑完成后跑 `npm test` 确认无回归。

### Milestone 1: 数据结构与配置扩展

**范围**：扩展 TypeScript 类型定义，为后续功能提供数据基础。不涉及任何运行时行为变更。

**工作内容**：

在 `src/trace/session/state.ts` 中新增 `RepoSnapshot` 接口并扩展 `SessionState`：

    export interface RepoSnapshot {
      relativePath: string;
      repoUrl?: string;
      branch: string;
      startCommit: string;
      dirty: boolean;
    }

    // SessionState 新增 optional 字段
    repoSnapshots?: RepoSnapshot[];

在 `src/trace/config/schema.ts` 中新增 `WorkspaceConfig` 并扩展 `TraceConfig`：

    export interface WorkspaceConfig {
      repos?: string[];
    }

    // TraceConfig 新增 optional 字段
    workspace?: WorkspaceConfig;

在 `src/trace/upload/types.ts` 的 `UploadMeta` 中新增：

    gitStartCommit?: string;

在 `src/trace/upload/tos.ts` 的 `buildObjectMetadata` 函数中新增：

    if (meta.gitStartCommit) result['x-tos-meta-git-start-commit'] = meta.gitStartCommit;

在 `src/trace/utils/hook-log.ts` 的 `HookLogEntry` 中新增（利用已有的 `[extra: string]: unknown` 索引签名，无需修改接口定义，但为了可读性在注释中注明新字段）：

    snapshotManifestKey?: string;
    snapshotDiffsKey?: string;

**成果**：所有类型定义就位，现有测试继续通过。

**验收**：`npm test` 全部 PASS，0 failures。`npx tsc --noEmit` 零错误。

### Milestone 2: 子仓库发现机制

**范围**：新建 `src/trace/enrichers/repo-discovery.ts`，实现配置声明 + 自动扫描两种发现方式。

**工作内容**：

新建文件 `src/trace/enrichers/repo-discovery.ts`，导出一个函数：

    export async function discoverRepos(
      cwd: string,
      configuredRepos?: string[],
    ): Promise<DiscoveredRepo[]>

    export interface DiscoveredRepo {
      relativePath: string;   // 相对于 cwd
      absolutePath: string;   // 绝对路径
    }

函数逻辑：

1. 检测 cwd 自身是否是 git 仓库（检查 `git rev-parse --show-toplevel` 是否成功）。如果是，加入结果列表，relativePath 为 `"."`。
2. 如果 `configuredRepos` 非空（来自 `.trace/config.yaml` 的 `workspace.repos`），遍历每个路径，用 `path.resolve(cwd, repoPath)` 得到绝对路径，检查该目录是否存在 `.git`（用 `fs.access(path.join(abs, '.git'))`)。有效的加入结果列表。
3. 如果 `configuredRepos` 为空或 undefined，执行自动扫描：从 cwd 开始，用 `fs.readdir` 递归扫描最多 2 层子目录，排除 `node_modules`、`.git`、`vendor`、`dist`、`build`、`__pycache__` 以及以 `.` 开头的隐藏目录。对每个目录检查是否存在 `.git` 子目录。总超时 5 秒（用 `AbortController` + `setTimeout`）。
4. 去重：如果 cwd 自身是 git repo（relativePath="."），确保它不在子目录扫描结果中重复出现。去重方式为比较 absolutePath 的 `fs.realpathSync`。

新建测试文件 `src/trace/enrichers/repo-discovery.test.ts`。用 `vi.mock('node:fs/promises')` 和 `vi.mock('node:child_process')` 模拟文件系统和 git 命令。测试用例覆盖：

- cwd 是 git repo + 无配置 + 子目录有 2 个 git repo → 返回 3 个 DiscoveredRepo
- cwd 不是 git repo + 无配置 + 子目录有 git repo → 返回子仓库，不含 "."
- cwd 是 git repo + 有配置 → 只返回配置中的仓库 + "."
- 配置中的路径不存在 → 跳过，不报错
- 无子仓库 → 只返回 cwd 自身（如果是 git repo）

**成果**：`discoverRepos()` 函数和完整测试。

**验收**：`npm test` 全部 PASS。`npx tsc --noEmit` 零错误。

### Milestone 3: session-start 多仓库快照采集

**范围**：改造 `src/trace/commands/session-start.ts`，在记录 cwd 自身 commit 的同时，发现并记录所有子仓库的快照。

**工作内容**：

在 `src/trace/commands/session-start.ts` 的 `runSessionStart` 函数中，在现有的 startCommit 采集逻辑之后（第 36 行之后、`saveSessionState` 调用之前），新增以下步骤：

1. 调用 `loadConfig(workDir)` 获取配置（需从 `../config/loader.js` 导入）
2. 调用 `discoverRepos(workDir, config.workspace?.repos)` 获取仓库列表
3. 对每个发现的仓库并行采集快照信息（使用 `Promise.all`）：
   - `git rev-parse HEAD` → startCommit
   - `git rev-parse --abbrev-ref HEAD` → branch
   - `git config --get remote.origin.url` → repoUrl（可选，失败不影响）
   - `git status --porcelain` → dirty（输出长度 > 0 则 dirty=true）
   每个 git 命令超时 3 秒，单个字段失败不影响其他字段。
4. 将采集结果组装为 `RepoSnapshot[]`，保存到 `SessionState.repoSnapshots`

将快照采集逻辑抽取为独立函数放在 `src/trace/enrichers/git.ts` 中：

    export async function collectRepoSnapshots(
      repos: DiscoveredRepo[],
    ): Promise<RepoSnapshot[]>

这个函数对每个 repo 并行调用已有的 `runGit` 辅助函数（需要先将 `runGit` 的可见性从 file-private 改为 module-exported，或者在 repo-discovery 中重新实现一个轻量版）。

**成果**：session-start 命令在多仓库目录下运行后，`~/.trace/sessions/{sessionId}.json` 中包含 `repoSnapshots` 数组。

**验收**：`npm test` 全部 PASS。手动在一个包含子仓库的目录运行 `xdev trace session-start --session-id test-123`，检查 `~/.trace/sessions/test-123.json` 文件中 `repoSnapshots` 包含所有预期仓库。

### Milestone 4: forward 多仓库 diff 采集与上传

**范围**：改造 `src/trace/commands/forward.ts`，在现有上传逻辑基础上新增 manifest 和 diffs 文件的构建和上传。

**工作内容**：

在 `src/trace/commands/forward.ts` 的 `runForwardInner` 函数中，步骤 5（collect enrichment data）之后、步骤 6（build upload metadata）之前，新增以下逻辑：

1. 读取 session state：`loadSessionState(adapterMeta.sessionId)`（已有 import，但当前只在 enrichers/git.ts 中使用）

2. 如果 sessionState 存在且有 `repoSnapshots`：
   a. 对每个 repoSnapshot 并行计算 diff：`git diff {startCommit}` in 对应目录
   b. 构建 manifest 对象：

        {
          version: 1,
          workspace: {
            path: workDir,
            isRepo: <sessionState.startCommit 不为空>
          },
          repos: sessionState.repoSnapshots.map(r => ({
            relativePath: r.relativePath,
            repoUrl: r.repoUrl,
            branch: r.branch,
            startCommit: r.startCommit,
            dirty: r.dirty,
          }))
        }

   c. 构建 diffs 对象：

        {
          version: 1,
          diffs: { [relativePath]: diffContent, ... }  // 只包含非空 diff
        }

   d. 将 manifest 和 diffs 各自 JSON.stringify → gzipString → 上传到 TOS
   e. TOS object key 使用 `buildObjectKey(uploadMeta, '.{timestamp}.manifest.json.gz')` 和 `buildObjectKey(uploadMeta, '.{timestamp}.diffs.json.gz')`

3. 在 `uploadMeta` 中填充 `gitStartCommit`（来自 sessionState.startCommit）

4. 在 `appendHookLog` 中新增 `snapshotManifestKey` 和 `snapshotDiffsKey` 字段

5. dry-run 模式同步支持：在 `dryRunForward` 函数中，将 manifest 和 diffs 写到本地输出目录

将 diff 采集逻辑抽取为 `src/trace/enrichers/git.ts` 中的新函数：

    export async function collectMultiRepoDiffs(
      cwd: string,
      repoSnapshots: RepoSnapshot[],
    ): Promise<Map<string, string>>

该函数对每个 snapshot 并行执行 `git diff {startCommit}`，返回 relativePath → diff 内容的 Map。

更新 `src/trace/upload/tos.test.ts` 中 `buildObjectMetadata` 的测试，验证新增的 `x-tos-meta-git-start-commit` header。

**成果**：forward 命令在多仓库场景下会额外上传 manifest.json.gz 和 diffs.json.gz。

**验收**：`npm test` 全部 PASS。`npx tsc --noEmit` 零错误。

### Milestone 5: 集成测试与 dry-run 验证

**范围**：用 dry-run 模式端到端验证三种场景。

**工作内容**：

创建一个集成测试脚本或测试用例，模拟三种场景。由于集成测试需要真实的 git 仓库，采用 vitest 中在临时目录创建 git repo 的方式：

在 `src/trace/commands/forward.test.ts` 中新增测试 suite `describe('multi-repo snapshot')`，测试：

- mock 场景：sessionState 有 repoSnapshots 时，forward 会调用 gzipString 和 uploadToTOS 上传 manifest 和 diffs
- mock 场景：sessionState 无 repoSnapshots 时（旧数据），forward 行为不变，不上传 manifest/diffs
- dry-run 场景：检查输出目录中是否生成了 `{sessionId}.manifest.json.gz` 和 `{sessionId}.diffs.json.gz`

手动验证步骤（不纳入自动化测试，但在 Concrete Steps 中记录命令）：

1. 创建临时多仓库目录结构
2. 运行 `xdev trace session-start --session-id multi-test --cwd <tmpdir>`
3. 在子仓库中做一些修改
4. 创建一个假的 JSONL 文件
5. 运行 `xdev trace forward --file <jsonl> --cwd <tmpdir> --dry-run --dry-run-output <outdir>`
6. 检查输出目录中的 manifest 和 diffs 文件内容

**成果**：自动化测试 + 手动验证均通过。

**验收**：`npm test` 全部 PASS，0 failures，0 skipped。

### Milestone 6: 文档更新

**范围**：更新设计文档，确保与最终实现一致。

**工作内容**：

更新 `docs/2026-04-28-multi-repo-trace-snapshot-design.md`，确保文档中的类型定义、文件路径、流程描述与实际代码一致。如有实现过程中的偏差（记录在 Decision Log 中），同步反映到设计文档。

**成果**：设计文档与代码完全对齐。

**验收**：文档中描述的每个类型、函数、文件路径都能在代码中找到对应。


## Concrete Steps

所有命令在仓库根目录执行：`/Users/bytedance/.local/share/opencode/worktree/0003cfff989dc1f20ad6f9e6a550ca2e217a8c43/stellar-garden`

Milestone 1 完成后验证：

    npx tsc --noEmit
    # 预期：无错误输出

    npm test
    # 预期：所有测试 PASS

Milestone 2 完成后验证：

    npx tsc --noEmit
    npm test
    # 预期：包含 repo-discovery.test.ts 的新测试，全部 PASS

Milestone 3 完成后验证：

    npx tsc --noEmit
    npm test

    # 手动验证（可选）：
    mkdir -p /tmp/multi-repo-test/repos/repo-a
    cd /tmp/multi-repo-test/repos/repo-a && git init && git commit --allow-empty -m "init"
    mkdir -p /tmp/multi-repo-test/repos/repo-b
    cd /tmp/multi-repo-test/repos/repo-b && git init && git commit --allow-empty -m "init"
    cd /tmp/multi-repo-test && git init && git commit --allow-empty -m "init"
    xdev trace session-start --session-id test-multi --cwd /tmp/multi-repo-test
    cat ~/.trace/sessions/test-multi.json
    # 预期：repoSnapshots 包含 3 个条目（"."、"repos/repo-a"、"repos/repo-b"）

Milestone 4 完成后验证：

    npx tsc --noEmit
    npm test

Milestone 5 完成后验证：

    npm test
    # 预期：全部 PASS，0 failures，0 skipped

    # 手动 dry-run 验证：
    echo '{"role":"user","content":"test"}' > /tmp/test-session.jsonl
    xdev trace forward --file /tmp/test-session.jsonl --source opencode --cwd /tmp/multi-repo-test --dry-run --dry-run-output /tmp/trace-dry-run
    ls /tmp/trace-dry-run/
    # 预期：包含 *.manifest.json.gz 和 *.diffs.json.gz 文件
    # 解压 manifest 验证内容：
    gunzip -c /tmp/trace-dry-run/*.manifest.json.gz | python3 -m json.tool
    # 预期：JSON 包含 version=1、workspace.isRepo=true、repos 数组含 3 个条目


## Validation and Acceptance

1. `npx tsc --noEmit` 零错误
2. `npm test` 全部 PASS，0 failures，0 skipped
3. dry-run 模式在多仓库目录下输出的 manifest.json.gz 解压后包含所有子仓库的 startCommit
4. dry-run 模式输出的 diffs.json.gz 解压后包含有修改的仓库的 diff 内容
5. 在传统单仓库目录下 dry-run，manifest repos 数组只包含 `"."`，行为与改造前等价
6. 在非 git 普通目录下 dry-run（子目录有 git repo），manifest.workspace.isRepo=false，repos 只包含子仓库


## Documentation Update

更新 `docs/2026-04-28-multi-repo-trace-snapshot-design.md`，确保与最终实现对齐。该文件在 Milestone 6 中处理。

如果实现过程中新增了在设计文档未覆盖的函数或类型，也需要补充到设计文档中。


## Idempotence and Recovery

所有里程碑都是幂等的：

- 类型定义的修改是声明式的，重复执行无副作用
- `discoverRepos()` 是纯函数（给定输入返回确定输出），可安全重复调用
- session-start 写入的 JSON 文件会覆盖同名文件，幂等
- forward 上传到 TOS 的 object key 包含时间戳，不会覆盖已有文件
- dry-run 输出到本地目录，可随时删除重跑

回滚路径：所有改动都是 additive（新增 optional 字段和新增文件），`git checkout` 到基线 commit 即可完全回退。


## Artifacts and Notes

manifest.json 示例（中心化文档仓库场景）：

    {
      "version": 1,
      "workspace": {
        "path": "/Users/user/stellar-garden",
        "isRepo": true
      },
      "repos": [
        {
          "relativePath": ".",
          "repoUrl": "git@code.byted.org:stone/devclaw_skills_center.git",
          "branch": "main",
          "startCommit": "89c51223668cae774356bd4d000d455558077b2a",
          "dirty": false
        },
        {
          "relativePath": "repos/repo-a",
          "repoUrl": "git@code.byted.org:team/repo-a.git",
          "branch": "feature/x",
          "startCommit": "def5678defabc1234567890abcdef1234567890ab",
          "dirty": true
        }
      ]
    }

diffs.json 示例：

    {
      "version": 1,
      "diffs": {
        "repos/repo-a": "diff --git a/src/main.ts b/src/main.ts\nindex abc..def 100644\n--- a/src/main.ts\n+++ b/src/main.ts\n@@ -1,3 +1,4 @@\n+import { foo } from './foo';\n ..."
      }
    }

TOS 上传路径示例：

    xtrace/zhongzhiwei/2026-04-28/opencode/sess-abc123.2026-04-28T09-07-59-000Z.manifest.json.gz
    xtrace/zhongzhiwei/2026-04-28/opencode/sess-abc123.2026-04-28T09-07-59-000Z.diffs.json.gz


## Interfaces and Dependencies

**现有依赖（不新增）**：

- `node:child_process`（execFile）— 执行 git 命令
- `node:fs/promises`（readdir、access、readFile、writeFile、mkdir）— 文件系统操作
- `node:path`（join、resolve、relative）— 路径处理
- `yaml`（parse）— YAML 配置解析
- `@byted-service/tos` — TOS 上传（已有）
- `vitest` — 测试框架

**新增导出的函数签名**：

    // src/trace/enrichers/repo-discovery.ts
    export interface DiscoveredRepo {
      relativePath: string;
      absolutePath: string;
    }
    export async function discoverRepos(
      cwd: string,
      configuredRepos?: string[],
    ): Promise<DiscoveredRepo[]>

    // src/trace/enrichers/git.ts（新增）
    export async function collectRepoSnapshots(
      repos: DiscoveredRepo[],
    ): Promise<RepoSnapshot[]>

    export async function collectMultiRepoDiffs(
      cwd: string,
      repoSnapshots: RepoSnapshot[],
    ): Promise<Map<string, string>>

**扩展的类型（详见 Milestone 1）**：

    // src/trace/session/state.ts
    RepoSnapshot（新增接口）
    SessionState.repoSnapshots（新增 optional 字段）

    // src/trace/config/schema.ts
    WorkspaceConfig（新增接口）
    TraceConfig.workspace（新增 optional 字段）

    // src/trace/upload/types.ts
    UploadMeta.gitStartCommit（新增 optional 字段）
