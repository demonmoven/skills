---
name: devclaw-skills-center-publish
description: 发布 devclaw_skills_center 仓库到 npm/bnpm 的固定流程技能。Use when user asks to 发版 devclaw_skills_center、publish devclaw_skills_center、为 devclaw_skills_center 发布 npm 包、推送 publish 分支、或生成该仓库指向 main 的 MR 链接。
---

# devclaw-skills-center-publish

按下面固定流程执行 `devclaw_skills_center` 的发版操作，尽量减少随意发挥。

## 固定仓库信息

- 仓库目录：`/home/yukaige/.openclaw/workspace/projects/devclaw_skills_center`
- 默认目标分支：`main`
- 远端仓库：`git@code.byted.org:stone/devclaw_skills_center.git`
- MR 链接模板：
  - `https://code.byted.org/stone/devclaw_skills_center/merge_requests/new?target_branch=main&source_branch=<url_encoded_branch>`

## 执行步骤

按顺序执行，不要跳步。

### 1. 切到 main 并强制同步最新

在仓库根目录执行：

```bash
git fetch origin --prune
git checkout main
git reset --hard origin/main
```

要求：
- `main` 必须和 `origin/main` 完全一致
- 这一步会丢弃本地未提交改动，属于高风险动作；如果用户没有明确授权，不要执行

### 2. 在项目根目录执行 npm publish

先检查：

```bash
npm config get registry
npm whoami
```

然后直接在项目根目录执行：

```bash
npm publish
```

要求：
- 不额外改源码逻辑
- 如果仓库自己的发布脚本会自动 build / bump version，允许其执行
- 如果发布失败，立即停止后续步骤，并汇报失败原因

### 3. 创建新的 publish 分支

发布成功后，检查工作区是否有因发布脚本产生的版本号或 lockfile 变更。

分支命名建议：

```bash
publish/$(date +%Y%m%d-%H%M%S)
```

例如：

```bash
git checkout -b publish/20260323-123000
```

要求：
- 新分支必须从最新 `main` 派生
- 如果没有任何变更，也要如实说明，不要伪造提交

### 4. 提交版本号变更 commit

如果存在变更：

```bash
git status --short
git add package.json package-lock.json npm-shrinkwrap.json .
git commit -m "chore: publish devclaw-skills-center"
```

要求：
- 只提交本次发布产生的合理变更
- 如果没有变更，不要强行 commit

### 5. 推送 publish 分支到远端

```bash
git push -u origin <publish_branch>
```

### 6. 拼接 MR 链接

把分支名做 URL encode 后拼接：

```text
https://code.byted.org/stone/devclaw_skills_center/merge_requests/new?target_branch=main&source_branch=<url_encoded_branch>
```

示例：

```text
https://code.byted.org/stone/devclaw_skills_center/merge_requests/new?target_branch=main&source_branch=publish%2F20260323-123000
```

## 输出要求

最终至少返回：

- 仓库路径
- 发布前所在分支 / 发布后所在分支
- npm registry
- npm publish 是否成功
- 新分支名
- 是否有版本号或 lockfile 变更
- git push 是否成功
- MR 链接

MR 链接在面向飞书消息输出时，必须使用飞书友好的 Markdown 超链接展示，不要直接裸贴长链接。

推荐格式：

```md
[创建 MR](https://code.byted.org/stone/devclaw_skills_center/merge_requests/new?target_branch=main&source_branch=publish%2F20260323-123000)
```

如果任一步失败，明确指出失败步骤、命令摘要和错误原因。

## 安全与边界

- `git reset --hard origin/main` 是破坏性操作，只有在用户明确要求“强行拉取到最新提交”时才能执行
- 没有用户明确要求时，不要自动执行真实 `npm publish`
- 如果只是验证能否发布，用：

```bash
npm publish --dry-run
```

## 衔接下一步

发版完成后，用下面格式在总结末尾提示用户：

- 如果成功：`如果你要，我可以继续直接帮你打开 MR 链接，或者顺手检查这次 publish 分支里的变更是否干净。`
- 如果失败：`如果你要，我可以继续帮你定位卡住的具体步骤，并把发版流程修到可正式执行。`
