# Change Name 解析指令

> 把用户的自由文本输入解析为符合 OpenSpec 约定的 kebab-case change-name。

## 输入参数

- `ARG`（可选）：用户传入的文本，可能是 change-name（已经 kebab-case）或自由描述
- `PRIMARY_REPO`：主仓库根目录，用于检查重名

## 输出

- `CHANGE_NAME`：最终 kebab-case 名称
- `IS_NEW_CHANGE`：布尔值
- `CHANGE_DIR`：`{PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}` 的绝对路径

---

## 步骤 1：参数判定

### 1a. 无参数

提示用户输入需求描述：

> "请用一句话描述你想做的变更（例如：加一个 dark mode 切换）："

收到用户输入后，作为 `ARG` 进入步骤 2。

### 1b. 有参数

判断 `ARG` 是否已是 kebab-case 形式：

- 全小写
- 仅含字母 / 数字 / 连字符 `-`
- 不含空格 / 中文 / 大写

满足以上条件 → `CHANGE_NAME = ARG`，跳到步骤 3。

否则进入步骤 2。

## 步骤 2：从描述生成 kebab-case 名称

1. 翻译为英文（如果输入是中文）
2. 提取 2-5 个关键词
3. 全部小写
4. 用 `-` 连接
5. 仅保留 `[a-z0-9-]`

示例：

```
"加一个 dark mode 切换" → "add-dark-mode"
"用户登录优化"          → "improve-user-login"
"修复 spec 校验 bug"    → "fix-spec-validation"
```

`CHANGE_NAME = {生成的 kebab-case}`

## 步骤 3：检查重名

```bash
ls -d "{PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}/" 2>/dev/null
```

- **不存在** → `IS_NEW_CHANGE = true`，**完成**
- **已存在** → 用 AskUserQuestion 反问：

  > "Change `{CHANGE_NAME}` 已存在。请选择："
  > [1] 继续基于已有 change（续接已有 artifacts）
  > [2] 换一个名字（输入新名称）
  > [3] 删除已有 change 并重新开始（**危险**）

  - 选 [1] → `IS_NEW_CHANGE = false`，**完成**
  - 选 [2] → 输入新名称作为新 `ARG`，回到步骤 1b
  - 选 [3] → 二次确认 → `rm -rf "{PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}/"` → `IS_NEW_CHANGE = true`，**完成**

## 步骤 4：完成

```text
CHANGE_DIR = {PRIMARY_REPO}/docs/xdev/openspec/changes/{CHANGE_NAME}
```

返回 `CHANGE_NAME` / `IS_NEW_CHANGE` / `CHANGE_DIR`。
