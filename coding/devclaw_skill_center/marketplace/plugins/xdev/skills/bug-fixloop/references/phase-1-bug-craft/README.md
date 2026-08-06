# Phase 1: Bug Craft（diff 驱动的测试用例构造）

## 对应原版本
bug-fixloop 的 phase-1-bug-craft **替换**了 gan-fixloop 的 phase-1-craft-test-case。两者功能等价但驱动源完全不同：

| 维度 | gan-fixloop phase-1 | bug-fixloop phase-1 |
|------|-----|-----|
| 驱动源 | SPEC + IDL | git diff |
| 动线概念 | User Journey | Change Point |
| 生成的测试 | E2E + 单元测试 | 仅单元测试（MVP） |
| 适用场景 | 新功能开发 / spec 完善 | bug 修复 / 小步改动 |

## 干什么
基于 `COMMIT_RANGE`（默认 `HEAD~1..HEAD`）解析 git diff，为每个改动的函数生成对应的 Go 单元测试。共 5 个内部 step：

1. **step1 - diff 解析**：执行 `git diff`，写入 `full_diff.txt` / `go_changed_files.txt` 等（主 context bash）
2. **step2 - 改动点抽取**：subagent 从 diff 提取每个函数的 change point
3. **step3 - 测试需求生成**：subagent 为每个 change point 决定测试需求（happy path / 边界 / 回归）
4. **step4 - 测试代码生成**：subagent 生成 Go 单元测试代码 + `go build` 校验
5. **step5 - 同步推送**：主 context bash 把测试 commit 到 BUSINESS_REPO

## 执行位置
- step1 / step5: 主 context bash（不启 subagent）
- step2 / step3 / step4: subagent (general-purpose)

## 主要 prompts
| 文件 | 谁读 | 干什么 |
|------|------|------|
| prompts/step1-diff-parse.md | 主 context | 执行 git diff 并解析出改动文件列表 |
| prompts/step2-change-points.md | subagent | 从 diff 提取 change points（函数级） |
| prompts/step3-test-requirements.md | subagent | 为每个 change point 决定测试需求 |
| prompts/step4-test-generation.md | subagent | 生成 Go 单元测试 + 编译校验 |
| prompts/step5-sync-push.md | 主 context | git add/commit/push 测试代码 |

## 输入
- `BUSINESS_REPO_PATH`：业务代码仓库（含要测试的 Go 代码）
- `BRANCH`：目标分支
- `COMMIT_RANGE`：git diff 范围（默认 `HEAD~1..HEAD`）
- `OUTPUT_DIR`：输出基目录

**不再需要**（与 gan-fixloop 不同）：
- ~~`SPEC_DIR`~~（bug-fixloop 不读 spec）
- ~~`IDL_REPO_PATH`~~（diff 已含接口变更信息）
- ~~`TEST_REPO_PATH`~~（MVP 只生成单元测试,单元测试在 BUSINESS_REPO 内）

## 产出
- `$OUTPUT_DIR/craft/diff_stat.txt`
- `$OUTPUT_DIR/craft/full_diff.txt`
- `$OUTPUT_DIR/craft/changed_files.txt`
- `$OUTPUT_DIR/craft/go_changed_files.txt`
- `$OUTPUT_DIR/craft/change_points.md`
- `$OUTPUT_DIR/craft/test_requirements.md`
- `$OUTPUT_DIR/craft/unit_tests/` + `generated_unit_test_cases.jsonl`
- `$OUTPUT_DIR/craft/unit_test_dirs.txt`
- BUSINESS_REPO 上 `$BRANCH` 的新提交（含生成的 `_test.go` 文件）

> **注意**：bug-fixloop MVP **不生成** `test_dirs.txt`（E2E 测试列表），phase-2 的 stage-2-test 会自动跳过 E2E，只跑单元测试。

## 跨外部 skill 依赖
无（bug craft 与字节内场无关）

## 跳过条件
- `GENERATE_TESTS=false` → 整个 phase-1 跳过，直接进入 phase-2 循环
- `COMMIT_RANGE` 中无 Go 源文件改动 → 阻断（MVP 仅支持 Go）

## 约束（MVP 限制）
- 只处理 `.go` 源文件改动
- 只生成**单元测试**（不生成 E2E）
- 不支持 staged / unstaged diff（COMMIT_RANGE 必须是已 commit 的范围）
- 每个 change point 最多 5 个测试，总共最多 30 个测试
