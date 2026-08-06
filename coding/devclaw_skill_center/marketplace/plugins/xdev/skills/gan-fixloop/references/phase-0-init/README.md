# Phase 0: Init（初始化）

## 对应原 Stage
原 byted-gan-fixloop 的 phase-0-precheck（前置检查）+ gan-fixloop 新增的"交互式参数收集"。

## 干什么
gan-fixloop 的初始化阶段，分两步执行：

1. **Step 1 — 交互式参数收集**：通过 wizard 自动从 cwd / git / 文件系统扫描候选，让用户通过 AskUserQuestion 选择 7 个必填参数。
2. **Step 2 — 前置检查**：参数收集完成后，执行原有的清理输出 / 克隆仓库 / bytedcli 发现 / 双站点认证 / AGW service-id 查找。

## 执行位置
主 context（不启 subagent）。两步都直接在主 context 完成。

## 主要 prompts
| 文件 | 谁读 | 干什么 |
|------|------|------|
| prompts/interactive-collect.md | 主 context | wizard 主流程：5 步收集参数（含 PROFILE 选择） |
| prompts/precheck.md | 主 context | 通用前置检查流程（PROFILE 适配） |
| prompts/adapters/auth-bytedance.md | 主 context | bytedcli 认证 + AGW service-id 查找（PROFILE=bytedance-tce） |
| prompts/adapters/auth-none.md | 主 context | 跳过 byted 相关检查（PROFILE=none） |
| prompts/lib/path-discovery.md | 主 context | 路径自动发现算法 |
| prompts/lib/psm-discovery.md | 主 context | PSM 自动推断算法（仅 PROFILE=bytedance-tce） |
| prompts/lib/candidate-generation.md | 主 context | 候选标签格式（含 `(当前分支: xxx)` 显示） |

## PROFILE 适配（gan-fixloop 新增）

gan-fixloop 在 wizard 最前面新增 Step 0 让用户选择 PROFILE：

| PROFILE | 描述 | 必填参数 |
|---------|------|---------|
| `bytedance-tce` | 字节内场（默认推荐） | PSM, BRANCH, BUSINESS_REPO, TEST_REPO, SPEC_DIR, IDL_REPO, IDL_BRANCH |
| `none` | 自部署（最简化） | BUSINESS_REPO, BRANCH, TEST_REPO, SPEC_DIR, BASE_URL |

PROFILE 决定了：
- wizard 后续 step 发出哪些 question
- precheck.md 中调用哪个 auth adapter
- phase-2 各 stage 加载哪个 adapter（adapters/{bytedance-tce,none}.md）

## 输入
- `$ARGUMENTS`（可能为空，可能为部分参数，可能为完整参数）
- 当前 cwd
- git 状态（cwd 是否为 git 仓库、当前分支、remote URL 等）

## 产出
完成 Phase 0 后，主 context 应当持有：
- 7 个必填参数：BUSINESS_REPO / BRANCH / TEST_REPO / SPEC_DIR / IDL_REPO / IDL_BRANCH / PSM
- 高级参数：MAX_ITERATIONS / GENERATE_TESTS / SINGLE_TEST_RUN / TCE_LANE / SKIP_DEPLOY 等
- BUSINESS_REPO_PATH / TEST_REPO_PATH / IDL_REPO_PATH 已就绪（本地路径或克隆完成）
- BYTEDCLI_SKILLS_DIR 变量已缓存
- AGW_SERVICE_ID 变量已缓存（可能为空）
- TEST_ENV 初始化为 `fornax_boe`
- bytedcli 双站点（prod + BOE）认证已通过

## 跨外部 skill 依赖
- `$BYTEDCLI_SKILLS_DIR/bytedance-auth/SKILL.md`（双站点认证）
- `$BYTEDCLI_SKILLS_DIR/bytedance-agw/SKILL.md`（按 PSM 查 service_id）
- `$BYTEDCLI_SKILLS_DIR/bytedance-tce/SKILL.md`（可选：wizard 中 PSM 模糊搜索）

## 跳过条件
- **Step 1 wizard**：完全显式调用（7 个必填参数都在 $ARGUMENTS 里）时跳过 wizard 的所有 question，仅最终汇总确认
- **Step 2 precheck**：必须执行，无跳过条件

## 失败处理
- 用户取消 wizard → 优雅退出主流程，不留 .costudio 残留
- 路径验证失败 → wizard 重新询问该参数
- bytedcli skills 目录未找到 → 仅 warn，下游 stage 容错
- 双站点认证失败 → 阻断主流程
- AGW service-id 未找到 → 仅 warn，phase-2 的 stage-1.5 会跳过

## 关键 UX 要求
**列出仓库类候选时，每个候选标签必须显示其当前 git 分支**，例如：
```
○ /Users/x/cozeloop_backend  (当前分支: feat/my-feature)
○ /Users/x/another_repo      (当前分支: main)
```

这通过 `lib/candidate-generation.md` 里的"仓库候选标签格式"规范实现。
