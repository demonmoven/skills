# 使用示例

本 Skill 的核心逻辑已封装在 `scripts/query_charge_config.py` 脚本中，推荐直接调用。脚本会自动处理查询路径的选择（优先 MCP，失败则兜底）和结果的格式化。

## 调用方式

### 调用命令

```bash
python3 scripts/query_charge_config.py <contract_inst>
```

**参数说明:**
- `<contract_inst>`: 需要查询的主体实例 ID。

**示例调用:**
```bash
python3 scripts/query_charge_config.py MSO202602105645
```

## 执行路径与依赖

### 主路径：通过 RDS MCP 执行（推荐）

这是首选的执行方式，它利用平台能力进行安全、可靠的数据库查询。

- **核心依赖**:
  - `byted_aime_sdk` 库。
  - `AIME_USER_CLOUD_JWT` 环境变量。

- **如何启用**:
  - 在调用工具（如 `bash`）时，**必须** 设置 `include_secrets=true`。这将使平台能够注入所需的 `AIME_USER_CLOUD_JWT`，从而完成身份认证。
  - **示例**: `bash(command: "...", include_secrets=true)`

- **连接信息**:
  - `db_name`: `caijing_bytepay_charge_union`
  - `region`: `bytedance`
  - VRegion `China-Pay` 的信息已包含在平台能力中，无需额外配置。

### 兜底路径：通过数据库直连执行

当主路径（MCP）因任何原因（如 SDK 未安装、JWT 缺失等）失败时，脚本会自动尝试此方法。

- **核心依赖**:
  - `pymysql` 库。
  - 数据库连接相关的环境变量。

- **环境变量配置**:
  - `CHARGE_CONFIG_DB_HOST`
  - `CHARGE_CONFIG_DB_USER`
  - `CHARGE_CONFIG_DB_PASSWORD`
  - `CHARGE_CONFIG_DB_PORT` (可选, 默认为 3306)
  - `CHARGE_CONFIG_DB_NAME` (可选, 默认为 `caijing_bytepay_charge_union`)

**注意**: 正常情况下，应优先确保主路径（MCP）可用。仅在调试或特殊场景下，才需要配置直连所需的环境变量。

## 预期输出

脚本会直接在标准输出打印 Markdown 格式的表格。

**成功时:**
```markdown
| contract_inst | scene | match_rule | charge_rule_content | collect_fee_type | collect_fee_mode |
|---|---|---|---|---|---|\n| MSO202602105645 | ... | ... | ... | ... | ... |
```

**无结果时:**
```
无匹配记录
```

**失败时:**
会在标准错误流打印具体的错误信息，例如：
- `RDS MCP 查询失败: 缺少 AIME_USER_CLOUD_JWT 环境变量...`
- `数据库配置错误: 缺少数据库连接配置环境变量...`
- `查询失败: ...`
