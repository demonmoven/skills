# Stage 4 · 代码修复

根据 Stage 3 的 `analysis_report.md`，按根因类型分派到业务仓库 / 测试仓库。

## 输入

- `$JUDGE_REPORT_PATH = $OUTPUT_DIR/iteration_$ITERATION/analysis_report.md`
- `$BUSINESS_REPO_PATH`：可能是单仓或 workspace
- `$TEST_REPO_PATH`：qa_model_effect 本地路径
- `$OUTPUT_DIR/detected_targets.jsonl`：Section 6 嗅探结果
- `$FAILED_CASES_FILE = $OUTPUT_DIR/iteration_$ITERATION/failed_cases.jsonl`
- 可选 `$HISTORY_SUMMARY_FILE`：上一轮 `iteration_$((ITERATION-1))/iteration_summary.md`
- `$BRANCH`
- `$BYTEDCLI_SKILLS_DIR`

## 根因 → 修复动作分派

`analysis_report.md` 的每条 fix_item 含：
- `nodeid`
- `root_cause_type` ∈ `BIZ` / `TEST` / `IFACE` / `INFRA` / `FLAKY`
- `target_repo`（具体路径）
- `target_file` + `target_line_range`
- `suggested_action`

| 根因 | 目标 | 修复规则 |
|---|---|---|
| `[BIZ]` | `target_repo` 下的 `.go` | Go 惯例：保持接口签名；触发 `go fmt`；新增字段加 `// comment`（仅在非显然时） |
| `[TEST]` | `qa_model_effect/testcases/*.py` 或 `*.online.json` | JSON schema 对齐邻居；`@test.runtime` 装饰器仅动 `file=`；断言语句改用 `common/im/asserts.py` 等现有工具 |
| `[IFACE]` | 两边：`target_repo` 的 Go 实现 + 测试 stub/JSON | 先改业务再改测试；若 IDL 需要动，更新 `$IDL_REPO` 并标注（不在本 skill 自动提交 IDL） |
| `[INFRA]` | 按建议：换账号池 or 在 JSON 里置 `skip: true` | 必须填 `skip_reason`；不阻断循环 |
| `[FLAKY]` | 不改代码 | 只在本轮 fix_summary.md 标记，供 Stage 5 合成 |

## 修复执行（按根因分组）

subagent 按 fix_item 分组处理。每条 fix_item：

1. 读 `target_file`（用 Read 工具）
2. 按 `suggested_action` 生成补丁（保持既有风格）
3. 用 Edit 工具应用（禁止直接 Write 覆盖非一次性文件）
4. 执行对应根因的验证：

**BIZ**：

```bash
cd $target_repo
go build ./... 2>&1 | head -30
go vet ./... 2>&1 | head -30
```

**TEST**（仅 pytest 侧）：

```bash
cd $TEST_REPO_PATH
python3 -m py_compile $target_file   # 或 .online.json 改的话用 json.load 校验
```

**IFACE**：两个验证都跑。

## 跨仓 commit/push

修复完成后：

```bash
# 1. 业务侧：多 sub-repo 串行 commit + push
for target in $(cat $OUTPUT_DIR/detected_targets.jsonl); do
  sub_repo=$(echo "$target" | jq -r .sub_repo_path)
  cd "$sub_repo"
  git add -A
  if git diff --cached --quiet; then
    continue  # 此 sub-repo 本轮无改动
  fi
  git commit -m "fix(iter $ITERATION): $(basename $sub_repo)"
  git push origin "$BRANCH"
done

# 2. 测试侧（如有改动）
cd $TEST_REPO_PATH
git add -A
if ! git diff --cached --quiet; then
  git commit -m "fix(iter $ITERATION): test fixes"
  git push origin "HEAD:$BRANCH"
fi
```

## 历史经验传递（防止陷入循环修同一处）

如果 `$HISTORY_SUMMARY_FILE` 存在：

1. 读取上一轮的修复清单
2. 与本轮 analysis_report.md 比对：
   - 如果同一个 `(target_file, target_line_range, root_cause_type)` 已在连续 ≥ 2 轮出现 → 标记为"修复循环"
   - 修复循环：**不修代码**，在 `fix_summary.md` 里标 `ESCALATION_NEEDED: 可能需要人工介入` 并继续下一条
3. 如果上一轮已标 `[FLAKY]` 的 nodeid 本轮还是失败，`[FLAKY]` 升级为 `[TEST]`（稳定复现 → 真实 bug），正常修复

## 产出 fix_summary.md

```markdown
# 迭代 $ITERATION · 修复摘要

## 本轮修复清单
| nodeid | 根因 | target_repo | target_file | 动作 |
|--------|------|-------------|-------------|------|
| ...    | BIZ  | creativity  | handler/xxx.go | ... |

## 无法修复的（跳过 / 升级）
| nodeid | 原因 | 动作 |
|--------|------|------|
| ...    | FLAKY | 本轮未修 |
| ...    | 修复循环 | ESCALATION_NEEDED |

## 涉及 sub-repo
- creativity: push 1 个 commit
- alice_ugc_plugin: push 0 个 commit（未动）

## 涉及测试仓库
qa_model_effect: push 1 个 commit
```

## 与原 fixloop fix-procedure.md 的差异

- 新增 `[TEST]` 类型修复（Python + JSON，不是 Go）
- 跨多 sub-repo commit/push（原 fixloop 只管单个 `$BUSINESS_REPO_PATH`）
- 历史经验传递从 `iteration_$((N-1))/iteration_summary.md` 读取（原 fixloop 已有此机制，此处强调并明确格式）
