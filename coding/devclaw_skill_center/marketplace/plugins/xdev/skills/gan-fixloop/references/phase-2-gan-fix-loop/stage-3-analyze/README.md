# Phase 2 / Stage 3: Analyze（失败根因分析）

## 对应原 Stage
原 Stage 3。

## 干什么
扮演 **Judge** 角色，分析 stage-2 产出的失败用例，判断每个失败/跳过是「测试代码问题」还是「业务代码问题」，并给出修复建议。

## 执行位置
subagent (Judge 角色)

## 主要 prompts

| 文件 | 谁读 | 干什么 |
|------|------|------|
| prompts/failure-analysis.md | subagent | 完整的 Judge 分析流程（含 5 种无效测试模式 + 系统性遗漏检查） |

## lib
（无 — go-test-json-format.md 放在 `_shared/`）

## 输入
- `FAILED_CASES_FILE`：来自 stage-2 的 `iteration_$ITERATION/failed_cases.jsonl`
- `BUSINESS_REPO_PATH`、`TEST_REPO_PATH`
- `SPEC_DIR`（用于校验测试预期值是否符合规格）
- `PSM`、`BYTEDCLI_SITE`（可选，LogID 查询用）

## 产出
- `$OUTPUT_DIR/iteration_$ITERATION/analysis_report.md`
  - 含 markdown 分析正文
  - 末尾结构化 JSON 块（`<!-- JUDGE_STRUCTURED_OUTPUT_START -->...END -->`）
  - JSON 中的 `fix_priority` 数组最多 5 条

## 关键动作
1. Read failed_cases.jsonl
2. **可选：通过 LogID 查询服务端日志**（当 log_ids 非空时）
3. 自动定位实现文件（从 TestXxx 名提取接口名 grep 业务代码）
4. 逐个失败用例分类：E2E_TEST_CODE / BUSINESS_CODE / UNCERTAIN
5. 检查 5 种无效测试模式（占位符 ID、软断言、条件断言、跳过验证、错误静默）
6. 检查系统性遗漏清单（seed 数据、异步状态、空数据降级、nil 降级、路由注册、DTO 转换、参数绑定、配置数据）
7. 验证 `files_to_modify` 中每个路径存在
8. 写出报告

## 跨外部 skill 依赖（条件性）
- `$BYTEDCLI_SKILLS_DIR/bytedance-log/SKILL.md`（步骤 2.5 LogID 查询，仅当 log_ids 非空时）
- `$BYTEDCLI_SKILLS_DIR/bytedance-tce/SKILL.md`（冷启动检查，仅 ITERATION>=3 且含 404/connection refused 时）

## 跳过条件
- 仅在 stage-2 判断未通过且未提前退出时执行（即 stage-2 → 这里）
