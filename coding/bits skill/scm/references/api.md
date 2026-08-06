# SCM 工具 API 参考手册

本文档详细介绍了 SCM 管理工具集中的每个命令及其参数。

## 1. 获取仓库信息 (`get_repo`)
获取 SCM 仓库的详细信息。

**参数：**
- `--repo_id` (可选): 仓库的唯一 ID（整数）。
- `--repo_name` (可选): 仓库的语义化名称（例如 `namespace/repo`）。

> **注意**：必须且只能提供 `repo_id` 或 `repo_name` 其中之一。

字段说明见 Repo / Version 字段说明（`fields.md`）。

---

## 2. 获取版本详情 (`get_version`)
获取特定版本的详细信息。

**参数：**
- `--version_id` (可选): 版本的唯一 ID。
- `--version_number` (可选): 语义化版本号（如 `1.0.0.1`）。
- `--repo_id` / `--repo_name`: 如果使用 `version_number` 查询，必须同时提供仓库标识。

字段说明见 Repo / Version 字段说明（`fields.md`）。

---

## 3. 创建新版本 (`create_version`)
创建一个新的 SCM 版本。

**参数：**
- `--repo_id` / `--repo_name`: 目标仓库标识。
- `--type` (默认: `online`): 环境类型，可选值为 `online`（线上）, `offline`（线下）, `test`（测试）。
- `--pub_base` (默认: `branch_base`): 发布基准，可选值为 `branch_base`（基于分支）或 `commit_base`（基于提交）。
- `--branch_name`: Git 分支名称（当 `pub_base` 为 `branch_base` 时必填）。
- `--commit_hash`: Git 提交哈希（当 `pub_base` 为 `commit_base` 时必填）。
- `--build_image`: 自定义构建镜像（可选）。

**认证：**
- 需要设置环境变量 `SCM_JWT_TOKEN`（`X-Jwt-Token`）。

**行为：**
- 请求会自动从 `SCM_JWT_TOKEN` 解码出 `create_user` 并随请求一起提交。

---

## 4. 获取版本差异 (`get_version_diff`)
对比两个版本的配置、环境变量和构建环境差异。

**参数：**
- `--repo_id` / `--repo_name`: 仓库标识。
- `--left_version_number`: 基准版本号（旧版本）。
- `--right_version_number`: 目标版本号（新版本）。

**认证：**
- 需要设置环境变量 `SCM_JWT_TOKEN`（`X-Jwt-Token`）。

> **注意**：右侧版本必须晚于左侧版本创建。

---

## 5. 获取构建日志 (`get_build_log`)
获取特定版本的构建日志。

**参数：**
- `--repo_id` / `--repo_name`: 仓库标识。
- `--version_number`: 版本号。
- `--step_name` (默认: `building`): 构建步骤名称。
- `--arch` (默认: `x86_64`): 构建架构，取值见 枚举值规则（`enums.md`）。

---

## 6. 其他辅助命令

### 获取版本状态 (`get_version_status`)
查询版本的 CI/CD 流水线状态。
- 参数同 `get_version`。
- `state` 聚合规则、`status` 分类取值见 枚举值规则（`enums.md`）。

### 获取版本依赖 (`get_version_dependencies`)
查询版本的程序依赖列表。
- 参数同 `get_version`。

### 获取版本列表 (`get_version_list`)
列出仓库的历史版本。
- 参数：`--repo_id` / `--repo_name`，以及 `--limit`（默认 10 条）。

字段说明见 Repo / Version 字段说明（`fields.md`）。
