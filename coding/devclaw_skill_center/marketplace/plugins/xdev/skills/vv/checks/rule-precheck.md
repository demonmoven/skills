# Rule Precheck（确定性规则预检查）

你是一位确定性检查器，负责执行不依赖 LLM 判断的编译和规则检查。

**重要**：本检查不做主观判断，所有检查项均有明确的 pass/fail 标准。

## 上下文

### 参数
- **TARGET_PATH**: {{TARGET_PATH}} — 待验证代码目录
- **BASE_SHA**: {{BASE_SHA}} — 基准 commit（默认 HEAD~1）
- **OUTPUT_DIR**: {{OUTPUT_DIR}} — 输出目录

## 你的任务

依次执行以下 5 项确定性检查，将每项结果记录为 finding，最后输出结构化报告。

### 检查 1: go vet 静态分析

```bash
cd {{TARGET_PATH}} && go vet ./... 2>&1
```

**判定**：
- exit 0 → PASS
- exit 非 0 → 每个 vet 警告记录为一个 finding（severity: high）

### 检查 2: go build 编译检查

```bash
cd {{TARGET_PATH}} && go build ./... 2>&1
```

**判定**：
- exit 0 → PASS
- exit 非 0 → 编译失败记录为 finding（severity: critical）

### 检查 3: Secrets Scan（硬编码密钥/Token 扫描）

对 `git diff --name-only {{BASE_SHA}}..HEAD` 中变更的文件执行正则扫描：

```bash
cd {{TARGET_PATH}} && git diff --name-only {{BASE_SHA}}..HEAD 2>/dev/null | while read f; do
  if [ -f "$f" ]; then
    grep -nE '(password|passwd|secret|token|api_key|apikey|access_key|private_key)\s*[:=]\s*["\x27][^"\x27]{8,}' "$f" 2>/dev/null
  fi
done
```

**扫描模式**（不区分大小写）：
- `password\s*[:=]\s*"[^"]{8,}"`
- `secret\s*[:=]\s*"[^"]{8,}"`
- `token\s*[:=]\s*"[^"]{8,}"`
- `api[_-]?key\s*[:=]\s*"[^"]{8,}"`
- `access[_-]?key\s*[:=]\s*"[^"]{8,}"`
- `private[_-]?key\s*[:=]\s*"[^"]{8,}"`

**排除**：
- `_test.go` 文件
- 注释中的内容
- 明确标注为示例/占位符的值（如 `"your-secret-here"`, `"placeholder"`, `"xxx"`, `"changeme"`）

**判定**：
- 无匹配 → PASS
- 有匹配 → 每个匹配记录为 finding（severity: critical）

### 检查 4: 禁用依赖检查

读取 go.mod 文件，检查是否引入了黑名单依赖：

```bash
cd {{TARGET_PATH}} && cat go.mod
```

**默认黑名单**（可通过 OUTPUT_DIR/vv/config.json 的 `deny_deps` 字段覆盖）：
- `github.com/astaxie/beego` — 已废弃框架
- `github.com/go-xorm/xorm` — 已废弃 ORM
- `github.com/Sirupsen/logrus` — 大小写变化的旧包（应使用 `sirupsen/logrus`）

检查变更中是否新增了黑名单依赖：
```bash
cd {{TARGET_PATH}} && git diff {{BASE_SHA}}..HEAD -- go.mod 2>/dev/null | grep "^+" | grep -v "^+++"
```

**判定**：
- 无黑名单依赖 → PASS
- 新增黑名单依赖 → 每个匹配记录为 finding（severity: high）

### 检查 5: 禁止修改目录检查

基于 `git diff --name-only` 检查是否修改了禁止目录：

```bash
cd {{TARGET_PATH}} && git diff --name-only {{BASE_SHA}}..HEAD 2>/dev/null
```

**默认禁止修改目录**（可通过 config.json 的 `forbidden_dirs` 覆盖）：
- `vendor/` — 应通过 go mod 管理
- `kitex_gen/` — 自动生成代码
- `loop_gen/` — 自动生成代码
- `.github/workflows/` — CI/CD 配置（需专门审批）

**判定**：
- 未修改禁止目录 → PASS
- 修改了禁止目录 → 每个违规文件记录为 finding（severity: high）

---

## 评分标准

> 使用**门禁模式评分**（区别于 LLM 审查模式，medium 扣 5 分而非 3 分）。

### 分数计算 (0-100 分)

基础分 100 分：

| 问题级别 | 扣分 |
|----------|------|
| critical | 每个 -25 分 |
| high | 每个 -10 分 |
| medium | 每个 -5 分 |
| low | 每个 -0 分 |

最低 0 分。

### 通过标准

以 `references/gate-policies.json` 中当前 `GATE_LEVEL` 的阈值配置为准。例如：
- PR 级：score >= 70，零 critical（允许 high/medium/low）
- Nightly 级：score >= 80，零 critical，零 high
- Release 级：score >= 90，零 critical，零 high，零 medium

---

## 输出格式

将报告写入文件：`{{OUTPUT_DIR}}/rule-precheck.md`

```markdown
# 规则预检查报告

## 检查结果摘要

| 检查项 | 状态 | 发现数 |
|--------|------|--------|
| go vet | PASS/FAIL | {n} |
| go build | PASS/FAIL | {n} |
| Secrets Scan | PASS/FAIL | {n} |
| 禁用依赖 | PASS/FAIL | {n} |
| 禁止目录 | PASS/FAIL | {n} |

## 发现的问题

### RC-{NNN}: {标题}
- **严重程度**: critical / high / medium / low
- **检查项**: go_vet / go_build / secrets_scan / deny_deps / forbidden_dirs
- **文件**: `{path}:{line}`
- **描述**: {详细描述}
- **修复建议**: {如何修复}

...

## 评分

<score>{0-100}</score>
<pass>{true/false}</pass>
```

同时写入结构化数据到 `{{OUTPUT_DIR}}/rule-precheck-findings.json`：

```json
[
  {
    "id": "RC-001",
    "severity": "critical",
    "category": "go_build_failure",
    "file": "cmd/server/main.go",
    "line": 15,
    "title": "编译失败",
    "description": "undefined: NewService",
    "fix_hint": "检查 NewService 函数是否存在或导入是否正确",
    "failure_type": "POLICY_VIOLATION"
  }
]
```
