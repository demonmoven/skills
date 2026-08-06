# Phase 2 / Stage 1: Deploy（部署）

## 对应原 Stage
原 Stage 1（TCE 部署）+ 原 Stage 1.5（AGW IDL 同步）。**两个原 stage 在新结构中合并为一个 stage**。

## 干什么
把业务代码部署到 TCE 泳道并同步 AGW 网关 IDL/路由配置，让后续 stage-2 的测试请求能够路由到正确的服务实例。

## 执行位置
- prompts/tce-deploy.md → **subagent** 执行（涉及大量代码探索和失败修复子循环）
- prompts/agw-idl-sync.md → **主 context** 直接执行（逻辑简单，无代码探索）

## 主要 prompts

| 文件 | 谁读 | 干什么 |
|------|------|------|
| prompts/tce-deploy.md | subagent | TCE 部署 + 实例状态轮询 + 失败修复子循环（最多 3 次） |
| prompts/agw-idl-sync.md | 主 context | AGW IDL/路由同步 + 泳道未注册自动恢复 |

## lib（参考资料，被 prompts 二级引用）

| 文件 | 用途 |
|------|------|
| lib/tce-commands.md | TCE 命令参考速查 |
| lib/tce-troubleshooting.md | TCE 常见问题排查手册 |

## 输入
- `BUSINESS_REPO_PATH` 当前在 `BRANCH` 分支
- `IDL_REPO_PATH` 当前在 `IDL_BRANCH` 分支
- `AGW_SERVICE_ID`（从 phase-0 缓存）
- `BYTEDCLI_SKILLS_DIR`（从 phase-0 缓存）
- `TCE_LANE`（首轮自动生成 `boe_costudio_<rand4>`，后续轮复用）

## 产出
- TCE_LANE 泳道名（写回主 context，整个循环都复用）
- 部署的服务实例（TCE 上的运行实例）
- 同步的 AGW IDL/路由配置

## 跨外部 skill 依赖
- `$BYTEDCLI_SKILLS_DIR/bytedance-tce/SKILL.md`
- `$BYTEDCLI_SKILLS_DIR/bytedance-agw/SKILL.md`
- `$BYTEDCLI_SKILLS_DIR/bytedance-tools/references/invocation.md`

## 跳过条件
- **TCE 部署**：`SKIP_DEPLOY=true` AND `TCE_LANE` 已指定 AND `ITERATION==1` 三者全满足时跳过（仅首轮）
- **AGW IDL 同步**：`AGW_SERVICE_ID` 为空 OR `ITERATION>1` 时跳过（仅首轮做）

> 注意：从第 2 轮开始，stage-4 修复了代码并 push 后必须重新部署才能验证修复效果，所以 ITERATION>=2 时强制执行 TCE 部署（使用 upgrade 动作）。
