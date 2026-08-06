# Phase 0: Precheck（前置检查）

## 对应原 Stage
原 fixloop SKILL.md 行 152-209 的"前置检查"段（无 stage 编号，是循环外的初始化工作）。

## 干什么
fix-loop 的初始化阶段。准备所有后续 phase 需要的环境状态：
- 解析参数与仓库
- 发现并校验 bytedcli 工具链
- 缓存 AGW service-id
- 验证四个目录可达

## 执行位置
主 context（不启 subagent）

## 主要 prompts
| 文件 | 谁读 | 干什么 |
|------|------|------|
| prompts/precheck.md | 主 context | 完整的前置检查流程 |

## 输入
- `$ARGUMENTS` 解析后的参数（PSM / BRANCH / BUSINESS_REPO / TEST_REPO / SPEC_DIR / IDL_REPO / IDL_BRANCH 等）
- 参数定义参考 `_shared/parameter-defaults.md`

## 产出
- `$OUTPUT_DIR` 已创建并清理上次残留
- BUSINESS_REPO_PATH / TEST_REPO_PATH / IDL_REPO_PATH 已就绪（本地路径或克隆完成）
- BYTEDCLI_SKILLS_DIR 变量已缓存
- AGW_SERVICE_ID 变量已缓存（可能为空）
- TEST_ENV 初始化为 `fornax_boe`
- bytedcli 双站点（prod + BOE）认证已通过

## 跨外部 skill 依赖
- `$BYTEDCLI_SKILLS_DIR/bytedance-auth/SKILL.md`（双站点认证）
- `$BYTEDCLI_SKILLS_DIR/bytedance-agw/SKILL.md`（按 PSM 查 service_id）

## 跳过条件
无。前置检查必须执行。

## 失败处理
- bytedcli skills 目录未找到：仅 warn，不阻断（下游 stage 会瞎调命令容错）
- 双站点认证失败：阻断主流程
- AGW service-id 未找到：仅 warn，phase-2 的 stage-1.5（AGW IDL 同步）会跳过
