# Phase 1 / Bug Craft / Step 5 — 同步推送

> 主 context 直接执行 bash，提交生成的测试代码到 BUSINESS_REPO。
> 不启动 subagent。

## 输入

- `BUSINESS_REPO_PATH`：业务代码仓库路径
- `BRANCH`：目标分支
- `OUTPUT_DIR`：输出基目录

可以读取的文件：
- `$OUTPUT_DIR/craft/generated_unit_test_cases.jsonl`
- `$OUTPUT_DIR/craft/unit_tests/`（生成的测试代码副本）

## 流程

### Step 1: 校验 BUSINESS_REPO 状态

```bash
cd $BUSINESS_REPO_PATH

# 确认分支
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "$BRANCH" ]; then
  echo "切换到 $BRANCH"
  git checkout "$BRANCH"
fi
```

### Step 2: 同步 unit_tests/ 到业务仓库

> **注意**：step 4 已经把测试文件生成到 BUSINESS_REPO 并做了编译校验。这里的 cp 是为了防止 step 4 执行中途失败导致文件缺失，做一次幂等同步。

```bash
# 幂等同步（step 4 可能已经 cp 过）
if [ -d "$OUTPUT_DIR/craft/unit_tests" ]; then
  cp -r "$OUTPUT_DIR/craft/unit_tests/"* "$BUSINESS_REPO_PATH/" 2>/dev/null || true
fi
```

### Step 3: git 状态检查

```bash
cd $BUSINESS_REPO_PATH
echo "=== git 状态 ==="
git status --short
echo "==="

# 如果没有改动（无新测试文件），报错
if git diff --quiet --cached && git diff --quiet; then
  echo "ERROR: 生成的测试文件似乎没有产生 git 变更"
  echo "可能原因："
  echo "  1. step 4 没有成功生成测试文件"
  echo "  2. 测试文件被忽略（在 .gitignore 中）"
  exit 1
fi
```

### Step 4: commit + push

```bash
cd $BUSINESS_REPO_PATH

# 只 add 新增的 _test.go 文件（保险起见，不 add 其它改动）
git add $(git status --porcelain | grep -E '\.go$' | grep '_test\.go' | awk '{print $2}')

# 或者更简单：直接 add 所有 _test.go 类型的改动
# git add "*_test.go" # 这种 pattern 可能不工作，用上面的方式更可靠

git commit -m "test: auto-generated regression tests for $COMMIT_RANGE

由 bug-fixloop 自动生成的单元测试,覆盖 $COMMIT_RANGE 范围的代码改动。
测试需求: \$OUTPUT_DIR/craft/test_requirements.md
生成清单: \$OUTPUT_DIR/craft/generated_unit_test_cases.jsonl
"

# Push
git push origin "$BRANCH"
```

### Step 5: 生成 unit_test_dirs.txt

```bash
cd $BUSINESS_REPO_PATH
MODULE_NAME=$(head -1 go.mod | awk '{print $2}')

# 从 jsonl 提取每个测试的相对目录
python3 -c "
import json, sys
module = sys.argv[1]
seen = set()
with open(sys.argv[2]) as f:
    for line in f:
        line = line.strip()
        if not line: continue
        case = json.loads(line)
        test = case.get('Test', '')
        pkg = case.get('Package', '')
        if test and pkg:
            if pkg.startswith(module):
                rel = pkg[len(module):].lstrip('/')
                rel_dir = './' + rel if rel else '.'
            else:
                rel_dir = './' + pkg
            key = f'{test} {rel_dir}'
            if key not in seen:
                seen.add(key)
                print(key)
" "$MODULE_NAME" "$OUTPUT_DIR/craft/generated_unit_test_cases.jsonl" \
  > "$OUTPUT_DIR/craft/unit_test_dirs.txt"

echo "=== unit_test_dirs.txt ==="
cat "$OUTPUT_DIR/craft/unit_test_dirs.txt"
echo "==="
```

### Step 6: 注意：bug-fixloop 不生成 test_dirs.txt（E2E 列表）

bug-fixloop MVP 只生成单元测试，**不生成 E2E 测试**，所以 `$OUTPUT_DIR/craft/test_dirs.txt` 不会被创建。

phase-2 的 stage-2-test 会检测到 `test_dirs.txt` 不存在，自动跳过 E2E 测试，只跑 `unit_test_dirs.txt` 中的单元测试。

## 产出

- BUSINESS_REPO 上的新提交（含生成的 `_test.go` 文件）
- `$OUTPUT_DIR/craft/unit_test_dirs.txt`
- 已 push 到 `$BRANCH` 分支

## 失败处理

| 情况 | 处理 |
|------|------|
| 没有文件变更 | 阻断,提示 step 4 未成功 |
| git push 失败（权限/网络） | 阻断,让用户手动检查 |
| 生成的测试文件在 .gitignore 中 | 阻断,提示用户调整 .gitignore 或改 OUTPUT_DIR |

## 后续

Step 5 完成后，Phase 1（bug craft）结束，进入 Phase 2（gan-fix-loop 循环）。由于只生成了单元测试，phase-2 的 stage-2 会跳过 E2E 测试，直接跑单元测试，如果失败触发 stage-3 Judge 和 stage-4 Fixer 修复。
