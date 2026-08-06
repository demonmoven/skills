# Find-Skills 技能检索

从多个技能来源中检索技能，检索结果需使用 install 命令安装。

## 命令

```bash
# 列出所有可用的技能来源
flou-cli install find-skills --list

# 搜索技能
flou-cli install find-skills <query>

# 指定来源搜索
flou-cli install find-skills --source <source-name> <query>

# 安装指定技能（需配合 --source）
flou-cli install --skills <skill-name> --source <source-name>

# 安装指定 CLI 依赖
flou-cli install --global-tools <tool-name>
```

## 参数

### find-skills 子命令

| 参数 | 说明 |
|------|------|
| `query` | 查询关键词 |
| `--list, -l` | 列出所有可用的技能来源 |
| `--source, -s` | 指定技能来源进行检索 |

### install 命令

| 参数 | 说明 |
|------|------|
| `--skills <skill>` | 指定要安装的技能名称 |
| `--global-tools <tool>` | 指定要安装的 CLI 依赖名称 |
| `--source, -s <source>` | 指定技能来源（仅安装 skill 时必填） |

## 技能来源

| 名称 | 仓库 |
|------|------|
| bytedcli | git@code.byted.org:byteapi/bytedcli.git |
| gdpa-cli | git@code.byted.org:tiktok/gdpa_skills.git |
| ai_coding_marketplace | git@code.byted.org:ies-cs/ai_coding_marketplace |
| feature-doc | git@code.byted.org:ies-cs/feature-doc.git |

## 示例

```bash
# 搜索 log 相关技能
flou-cli install find-skills rds

# 搜索 es 相关技能
flou-cli install find-skills es

# 列出所有技能来源
flou-cli install find-skills --list

# 安装指定技能
flou-cli install --skills bytedance-log --source bytedcli

# 安装指定 CLI 依赖
flou-cli install --global-tools feature_docs
```

## 注意事项

- 原 `flou-cli find-skills` 命令已废弃，请使用 `flou-cli install find-skills` 代替
- 安装技能时必须指定 `--source` 参数
- 安装 CLI 依赖时不需要 `--source`
