# Phase 1: Craft Test Case（测试用例构造）

## 对应原 Stage
原 Stage 0（含 7 个 step：0.1-0.7）。

## 干什么
基于 SPEC + IDL 自动构造测试用例。共 7 个内部 step：
1. 提取用户动线（user journeys）
2. 覆盖度分析（动线 → E2E）
3. 生成 E2E 测试代码
4. 同步 E2E 到 TEST_REPO 并 push（主 context bash）
5. 覆盖度分析（E2E → 单元测试）
6. 生成单元测试代码
7. 同步单元测试到 BUSINESS_REPO 并 push（主 context bash）

## 执行位置
- step1 / step2 / step3 / step5 / step6: subagent (general-purpose)
- step4 / step7: 主 context bash（无 prompts 文件）

## 主要 prompts
| 文件 | 谁读 | 干什么 |
|------|------|------|
| prompts/step1-user-journeys.md | subagent | 从 SPEC + IDL 提取用户动线 |
| prompts/coverage-analysis.md | subagent | 覆盖度差距分析（被 step2 / step5 通过 mode 参数复用） |
| prompts/step3-e2e-generation.md | subagent | 生成 Go E2E 测试代码 |
| prompts/step6-unit-tests.md | subagent | 生成 Go 单元测试代码 |

## 输入
- `SPEC_DIR`：SPEC 需求文档目录
- `IDL_REPO_PATH`：IDL 仓库（thrift / protobuf）
- `BUSINESS_REPO_PATH`：业务代码仓库
- `TEST_REPO_PATH`：测试代码仓库

## 产出
- `$OUTPUT_DIR/craft/user_journeys.md`
- `$OUTPUT_DIR/craft/stage1_to_stage2_coverage.md`
- `$OUTPUT_DIR/craft/e2e_work_copy/` + `generated_test_cases.jsonl`
- `$OUTPUT_DIR/craft/test_dirs.txt`（精确测试列表）
- `$OUTPUT_DIR/craft/stage2_to_stage3_coverage.md`
- `$OUTPUT_DIR/craft/unit_tests/` + `generated_unit_test_cases.jsonl`
- `$OUTPUT_DIR/craft/unit_test_dirs.txt`
- TEST_REPO 上新建的 `$BRANCH` 分支（含生成的 E2E 测试）
- BUSINESS_REPO 上 `$BRANCH` 的新提交（含生成的单元测试）

## 跨外部 skill 依赖
无（craft 阶段不调 bytedcli）

## 跳过条件
- `GENERATE_TESTS=false` → 整个 phase-1 跳过，直接进入 phase-2 的循环
