# Git 操作完整指南

## 概述

本文档提供 Git 操作的完整指南，使用通用 Git CLI 工具进行版本控制。

## 智能提交 (Smart Commit)

### 基本用法

```bash
# 提交变更
git add .
git commit -m "feat: add user authentication"

# 提交并推送
git add .
git commit -m "feat: add user authentication"
git push origin <branch-name>
```

### 约定式提交

推荐使用约定式提交格式：

```
<type>(<scope>): <subject>

<body>

<footer>
```

#### 类型 (type)

| 类型 | 说明 |
|------|------|
| feat | 新功能 |
| fix | 修复 bug |
| docs | 文档更新 |
| style | 代码格式调整 |
| refactor | 代码重构 |
| test | 测试相关 |
| chore | 构建/工具相关 |

### 自动分析建议

在提交时，可以根据代码变更类型选择合适的提交信息：

- 检测新增文件 → `feat: add ...`
- 检测测试文件 → `test: add/update tests`
- 检测文档 → `docs: update documentation`
- 检测修复 → `fix: resolve ...`

## 分支管理 (Branch Management)

### 基本用法

```bash
# 创建普通分支
git checkout -b feature-name

# 创建特性分支
git checkout -b feature/user-auth

# 创建修复分支
git checkout -b bugfix/login-bug

# 创建热修复分支
git checkout -b hotfix/critical-fix

# 创建发布分支
git checkout -b release/v1.2.0
```

### 分支命名规范

| 类型 | 命名格式 | 示例 |
|------|----------|------|
| feature | feature/<name> | feature/user-auth |
| bugfix | bugfix/<name> | bugfix/login-error |
| hotfix | hotfix/<name> | hotfix/security-patch |
| release | release/<version> | release/v1.2.0 |

### 分支管理命令

```bash
# 查看所有分支
git branch -a

# 删除已合并分支
git branch -d <branch-name>

# 删除未合并分支（强制）
git branch -D <branch-name>

# 重命名分支
git branch -m <old-name> <new-name>

# 清理远程已删除的分支引用
git fetch -p
```

## 冲突解决 (Resolve Conflicts)

### 检测冲突

```bash
# 查看冲突文件
git status

# 查看冲突详情
git diff
```

### 解决策略

```bash
# 使用本地版本
git checkout --ours <file>

# 使用远程版本
git checkout --theirs <file>

# 手动解决后标记为已解决
git add <file>
```

### 冲突标记说明

```
<<<<<<< HEAD
本地版本内容
=======
远程版本内容
>>>>>>> branch-name
```

## 状态分析 (Status Analysis)

### 基本用法

```bash
# 查看工作区状态
git status

# 查看简短状态
git status -s

# 查看分支信息
git status -b
```

### 输出示例

```
On branch main
Changes to be committed:
  (use "git restore --staged <file>..." to unstage)
    new file:   file1.txt
    modified:   file2.txt

Changes not staged for commit:
  (use "git add <file>..." to update what will be committed)
  (use "git restore <file>..." to discard changes in working directory)
    modified:   file3.txt

Untracked files:
  (use "git add <file>..." to include in what will be committed)
    file4.txt
```

## 工作流示例

### 功能开发流程

```bash
# 1. 创建特性分支
git checkout -b feature/user-profile

# 2. 开发并提交
# ... 编写代码 ...
git add .
git commit -m "feat: add user profile feature"
git push origin feature/user-profile

# 3. 创建 Pull Request
# ... 在平台上操作 ...

# 4. 合并后清理
git checkout main
git pull origin main
git branch -d feature/user-profile
git push origin --delete feature/user-profile
```

### 热修复流程

```bash
# 1. 从 main 创建热修复分支
git checkout main
git pull origin main
git checkout -b hotfix/critical-bug

# 2. 修复并提交
# ... 修复代码 ...
git add .
git commit -m "fix: resolve critical security issue"
git push origin hotfix/critical-bug

# 3. 合并回 main 和 develop
# ... 在平台上操作 ...
```

### 发布流程

```bash
# 1. 创建发布分支
git checkout -b release/v1.0.0

# 2. 版本号更新和测试
# ... 更新版本号，运行测试 ...

# 3. 提交发布
git add .
git commit -m "chore: release v1.0.0"
git push origin release/v1.0.0

# 4. 打标签
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

## 最佳实践

### 1. 提交粒度

- 每次提交只做一件事
- 提交信息清晰描述变更
- 避免大批量提交

### 2. 分支策略

- main 分支保持稳定
- feature 分支短小精悍
- 及时删除已合并分支

### 3. 冲突预防

- 频繁同步上游变更
- 小步提交，减少冲突范围
- 及时沟通协调

### 4. 提交信息

- 使用约定式提交格式
- 说明"做了什么"和"为什么"
- 关联相关 Issue

## 常用命令速查

### 基础操作

```bash
# 初始化仓库
git init

<<<<<<< HEAD
# 克隆仓库 (⚠️ 强制使用 SSH 格式，禁止使用 HTTPS，防止卡在终端输入密码环节)
git clone git@code.byted.org:<project>/<repo>.git

# 查看状态
git status

# 添加文件
git add <file>
git add .

# 提交
git commit -m "message"

# 推送
git push origin <branch>

# 拉取
git pull origin <branch>
```

### 分支操作

```bash
# 创建分支
git branch <branch-name>
git checkout -b <branch-name>

# 切换分支
git checkout <branch-name>
git switch <branch-name>

# 合并分支
git merge <branch-name>

# 删除分支
git branch -d <branch-name>
```

### 远程操作

```bash
# 查看远程仓库
git remote -v

# 添加远程仓库
git remote add origin <repo-url>

# 更新远程仓库地址
git remote set-url origin <new-url>

# 删除远程仓库
git remote remove origin
```

### 撤销操作

```bash
# 撤销工作区修改
git restore <file>

# 撤销暂存区
git restore --staged <file>

# 撤销最近一次提交
git reset --soft HEAD~1

# 撤销最近一次提交并丢弃修改
git reset --hard HEAD~1
```

## 故障排除

### 提交失败

```bash
# 检查是否有未暂存的变更
git status

# 检查 pre-commit 钩子
ls -la .git/hooks/
```

### 推送失败

```bash
# 拉取远程变更
git pull --rebase origin <branch>

# 强制推送 (谨慎使用)
git push --force-with-lease origin <branch>
```

### 合并冲突

```bash
# 中止合并
git merge --abort

# 使用工具解决
git mergetool
```

### 查看历史

```bash
# 查看提交历史
git log --oneline --graph

# 查看文件修改历史
git log -p <file>

# 查看某次提交详情
git show <commit-hash>
```

## Git 配置

### 用户配置

```bash
# 设置用户名
git config --global user.name "Your Name"

# 设置邮箱
git config --global user.email "your.email@example.com"

# 查看配置
git config --list
```

### SSH 配置

```bash
# 生成 SSH 密钥
ssh-keygen -t rsa -b 4096 -C "your.email@example.com"

# 添加密钥到 SSH 代理
ssh-add ~/.ssh/id_rsa

# 测试 SSH 连接
ssh -T git@code.bytedance.net
```

### 别名配置

```bash
# 设置常用别名
git config --global alias.st status
git config --global alias.co checkout
git config --global alias.br branch
git config --global alias.ci commit
git config --global alias.lg "log --oneline --graph"
```

## 进阶技巧

### 交互式暂存

```bash
# 交互式添加
git add -p

# 交互式暂存
git reset -p
```

### 储藏 (Stash)

```bash
# 储藏当前修改
git stash

# 查看储藏列表
git stash list

# 应用储藏
git stash apply

# 删除储藏
git stash drop
```

### Cherry-pick

```bash
# 选择性合并提交
git cherry-pick <commit-hash>

# 选择性合并多个提交
git cherry-pick <commit-hash1> <commit-hash2>
```

### Rebase

```bash
# 变基到主分支
git rebase main

# 交互式变基
git rebase -i HEAD~3

# 中止变基
git rebase --abort

# 继续变基
git rebase --continue
```

## 联系支持

如果遇到无法解决的问题，请联系：

- **Git 官方文档**: https://git-scm.com/doc
- **代码平台支持**: code-platform@bytedance.com
- **IT支持**: it-helpdesk@bytedance.com
