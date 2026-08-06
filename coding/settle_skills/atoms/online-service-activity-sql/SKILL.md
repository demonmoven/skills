---
name: online-service-activity-sql
description: 服务商活动费率配置 SQL 生成原子。按 PRD 直接渲染产出可执行的 SQL 文件(SQL 模板内置于本原子,无外挂业务仓库分支依赖),供 boe-sql-ticket 原子提交 BOE 工单。仅服务商活动费率配置子流程使用。
---

# online-service-activity-sql

## 何时调用

仅 **`service_merchant_activity_rate`** 子流程的第 3 步调用本原子。其它子流程严禁调用。

## 输入

| 参数 | 必填 | 说明 |
|---|---|---|
| `prd_title` | 是 | PRD 标题(由 prd-facade 产出) |
| `prd_link` | 是 | PRD 链接 |
| `prd_requirements` | 是 | PRD 摘要(自然语言) |
| `out_dir` | 否 | 默认 `<sandbox>/service_merchant_activity_rate/<session_id>/sql/` |

## 输出

```json
{
  "sql_file_path": "<绝对路径>",
  "sql_size_bytes": 1234,
  "sql_excerpt": "<前 50 行 + 末 20 行>",
  "rendered_at": "<ISO8601>"
}
```

## 执行步骤

### Step 1:准备 SQL 模板与字段映射(原子内置)

SQL 模板与字段映射 yaml **内置于本原子目录**(`template.sql.j2` / `fields.yaml` / `render.py`),
无需也不应该从外部业务仓库分支拉取。原子目录结构:

```
atoms/online-service-activity-sql/
  ├── SKILL.md           # 本文件
  ├── template.sql.j2    # SQL 模板(Jinja2)
  ├── fields.yaml        # 字段映射 / 默认值
  └── render.py          # 渲染脚本
```

### Step 2:按 PRD 渲染 SQL

调用本原子目录下的 `render.py`,输入 `prd_title` / `prd_requirements` / `fields.yaml`,
输出到 `<out_dir>/<slug>.sql`,`<slug>` 由 PRD 标题 kebab-case 而来。

```bash
ATOM_DIR="$(dirname "$0")"
python3 "$ATOM_DIR/render.py" \
  --prd-title "$PRD_TITLE" \
  --prd-requirements-file "$REQ_FILE" \
  --template "$ATOM_DIR/template.sql.j2" \
  --fields "$ATOM_DIR/fields.yaml" \
  --out "$OUT_DIR/$SLUG.sql"
```

> 若 PRD 描述需要的字段在 `fields.yaml` 中缺失,原子内 LLM 子调用按 PRD 文本补全后再渲染。

### Step 3:返回结果

读 `$OUT_DIR/$SLUG.sql`:
- 大小 0 → 报错
- 前 50 行 + 末 20 行拼成 `sql_excerpt`(中间省略)
- 返回上述 JSON

## 异常分支

| 情况 | 处理 |
|---|---|
| 渲染脚本报错 | 抛原始 stderr |
| SQL 大小 = 0 | 抛错「渲染产物为空,模板或 PRD 输入有问题」 |
| `fields.yaml` 字段缺失且 LLM 无法补全 | 抛错并提示需要补充的字段名 |

## 关键约束

- **不依赖任何外部业务仓库分支**(SQL 模板内聚于本原子)
- **不操作 BOE 数据库**(本原子只产 SQL 文件,执行交给 `boe-sql-ticket`)
- **不写文件到用户家目录之外的位置**
