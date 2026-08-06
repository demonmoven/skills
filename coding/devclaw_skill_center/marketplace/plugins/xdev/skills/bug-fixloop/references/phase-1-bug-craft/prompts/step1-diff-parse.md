# Phase 1 / Bug Craft / Step 1 — Diff 解析

> 主 context 直接执行 bash，解析 git diff 并把结果写入 `$OUTPUT_DIR/craft/`。
> 不启动 subagent。

## 输入

- `BUSINESS_REPO_PATH`：业务代码仓库路径（来自 phase-0）
- `COMMIT_RANGE`：git diff 范围，默认 `HEAD~1..HEAD`
- `OUTPUT_DIR`：输出基目录（默认 `.costudio`）

## 流程

### Step 1: 校验 COMMIT_RANGE 合法

```bash
cd "$BUSINESS_REPO_PATH"

# 校验 COMMIT_RANGE 的两端都能解析
if ! git rev-parse --verify "${COMMIT_RANGE%..*}^{commit}" >/dev/null 2>&1; then
  echo "ERROR: COMMIT_RANGE 起点无效: ${COMMIT_RANGE%..*}"
  exit 1
fi
if ! git rev-parse --verify "${COMMIT_RANGE#*..}^{commit}" >/dev/null 2>&1; then
  echo "ERROR: COMMIT_RANGE 终点无效: ${COMMIT_RANGE#*..}"
  exit 1
fi
```

### Step 2: 生成 diff stat（改动文件汇总）

```bash
mkdir -p "$OUTPUT_DIR/craft"

git diff "$COMMIT_RANGE" --stat > "$OUTPUT_DIR/craft/diff_stat.txt"

echo "=== diff stat ==="
cat "$OUTPUT_DIR/craft/diff_stat.txt"
echo "==="
```

### Step 3: 生成完整 diff

```bash
git diff "$COMMIT_RANGE" > "$OUTPUT_DIR/craft/full_diff.txt"

# 检查 diff 大小
diff_lines=$(wc -l < "$OUTPUT_DIR/craft/full_diff.txt")
echo "diff 总行数: $diff_lines"

if [ "$diff_lines" -gt 5000 ]; then
  echo "警告：diff 规模较大（$diff_lines 行），生成测试可能耗时较长。"
fi

if [ "$diff_lines" -eq 0 ]; then
  echo "ERROR: COMMIT_RANGE=$COMMIT_RANGE 无任何改动。请检查范围是否正确。"
  exit 1
fi
```

### Step 4: 提取改动文件列表（含改动类型）

```bash
# 格式: M|A|D|R<number>\t<path>
git diff "$COMMIT_RANGE" --name-status > "$OUTPUT_DIR/craft/changed_files.txt"

echo "=== 改动文件列表 ==="
cat "$OUTPUT_DIR/craft/changed_files.txt"
echo "==="
```

### Step 5: 过滤出相关的 Go 源文件（MVP 限制）

```bash
# 只保留：
# - 状态为 M (修改) 或 A (新增)
# - 后缀为 .go
# - 不是 _test.go（不是已有的测试文件）
# - 不在 vendor/ 目录

grep -E '^(M|A)[[:space:]]+.*\.go$' "$OUTPUT_DIR/craft/changed_files.txt" \
  | grep -v '_test\.go$' \
  | grep -v 'vendor/' \
  | grep -v '/generated_' \
  | awk '{print $2}' > "$OUTPUT_DIR/craft/go_changed_files.txt"

count=$(wc -l < "$OUTPUT_DIR/craft/go_changed_files.txt")
echo "需要生成测试的 Go 文件数: $count"

if [ "$count" -eq 0 ]; then
  echo "警告：COMMIT_RANGE 中没有改动 Go 源文件（排除 vendor / 已有测试文件）。"
  echo "bug-fixloop 的 MVP 版本只支持 Go 单元测试生成。如果你的改动不是 Go 代码，"
  echo "请考虑使用 gan-fixloop（spec 驱动）代替。"
  exit 1
fi

echo "=== Go 改动文件 ==="
cat "$OUTPUT_DIR/craft/go_changed_files.txt"
echo "==="
```

## 产出

| 文件 | 内容 |
|------|------|
| `$OUTPUT_DIR/craft/diff_stat.txt` | `git diff --stat` 输出 |
| `$OUTPUT_DIR/craft/full_diff.txt` | 完整 diff（subagent 在 step-2 读） |
| `$OUTPUT_DIR/craft/changed_files.txt` | `git diff --name-status` 输出 |
| `$OUTPUT_DIR/craft/go_changed_files.txt` | 过滤后的 Go 源文件列表（一行一个相对路径） |

## 失败处理

| 情况 | 处理 |
|------|------|
| COMMIT_RANGE 无效 | 阻断,提示用户检查范围 |
| diff 为空 | 阻断,提示用户 commit 范围可能错了 |
| diff 过大（> 5000 行） | 仅警告,不阻断 |
| 没有 Go 源文件改动 | 阻断,提示 MVP 仅支持 Go |

## 后续

Step 1 完成后进入 Step 2 改动点抽取（subagent）。
