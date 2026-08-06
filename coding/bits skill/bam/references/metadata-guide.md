# BAM 元数据管理指南

> 本文档描述 BAM 元数据管理的完整使用方式。
>
> **执行前缀**：参考 `references/invocation.md`；下面示例直接写 `bits-cli`。

## When to use

- PSM 搜索、收藏、最近查看
- 方法列表 / 方法详情
- 服务版本查询
- 创建服务版本（IDL 版本更新）

## Quick start

Commands are grouped under `bam psm`, `bam method`, `bam version`, and `bam idl`. Old flat names (e.g. `bam list-recent-psm`, `bam search-psm`, `bam list-method`, `bam get-method`, `bam versions`, `bam update-idl-version`) still work as hidden aliases.

### PSM 列表

```bash
# 列出最近访问的 PSM
bits-cli bam psm list --cluster default

# 列出收藏的 PSM
bits-cli bam psm list --cluster default --starred

# 搜索 PSM
bits-cli bam psm search "example.service.api" --cluster default
```

### 方法列表 / 详情

```bash
# 列出某 PSM 的所有方法
bits-cli bam method list --psm "example.service.api"

# 通过 endpoint-id 获取方法详情
bits-cli bam method get --endpoint-id 123456 --version 1.2.3

# 通过 PSM + 方法名获取详情
bits-cli bam method get --psm "example.service.api" --method "DemoMethod"
```

### 版本历史

```bash
# 查询服务版本历史
bits-cli bam version list "example.service.api" --cluster default
```

### 创建/更新 IDL 版本

```bash
# 自动递增版本号（patch +1）
bits-cli bam idl update --psm "example.service.api" --branch master --next-version

# 指定版本号
bits-cli bam idl update --psm "example.service.api" --branch "codex/fix-idl" --version "1.2.4" --commit-id "abc1234" --commit-msg "update idl"
```

## 命令详解

### bam psm

| 子命令 | 说明 | 参数 |
|-------|------|------|
| `list` | 列出 PSM | `--cluster`, `--starred` |
| `search` | 搜索 PSM | `--cluster`, `<keyword>` |

### bam method

| 子命令 | 说明 | 参数 |
|-------|------|------|
| `list` | 列出方法 | `--psm`, `--ep-type` |
| `get` | 获取方法详情 | `--endpoint-id` 或 `--psm` + `--method`, `--version`, `--schema` |

### bam version

| 子命令 | 说明 | 参数 |
|-------|------|------|
| `list` | 列出版本历史 | `<psm>`, `--cluster` |

### bam idl

| 子命令 | 说明 | 参数 |
|-------|------|------|
| `update` | 创建/更新 IDL 版本 | `--psm`, `--branch`, `--version` 或 `--next-version`, `--commit-id`, `--commit-msg` |

## Notes

- `method get` 支持 `--endpoint-id` 或 `--psm` + `--method` 两种定位方式
- `--schema ref|raw` 控制 schema 展示方式
- `idl update` 必须提供 `--psm` 和 `--branch`，版本号通过 `--version` 指定或 `--next-version` 自动在最新版本基础上 patch +1
- 缺少必填参数会自动输出帮助信息
- 需要结构化输出加 `--json`（全局选项，放在子命令之前，如 `bits-cli --json bam method get ...`）

## ⚠️ 约束

- **不要** 未经确认执行 IDL 版本创建 (`bam idl update`)
- 创建版本前应先确认本地 IDL 与 BAM 远端差异
