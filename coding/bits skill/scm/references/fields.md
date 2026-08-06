# Repo / Version 字段说明

本文件用于集中说明 `get_repo` 与 `get_version` 返回结果中的常见字段，供 `references/api.md` 等位置引用。

## Repo（get_repo）常见字段

- `id`：仓库在 SCM 系统内的唯一 ID
- `name`：SCM 仓库名称（例如 `toutiao/tce/artifact`）
- `repo_name`：Git 仓库名称（例如 `tce/artifact`）
- `http_url` / `git_url`：仓库拉取地址（HTTP / SSH）
- `desc`：仓库描述
- `type`：仓库编程语言，取值见 枚举值规则（`enums.md`）
- `create_user` / `create_date` / `update_date`：创建人与时间、更新时间
- `default_branch`：默认分支
- `arch`：支持构建的架构列表，取值见 枚举值规则（`enums.md`）
- `image` / `image_alias`：默认构建镜像与别名
- `compile_script_path` / `lint_check_script_path` / `unittest_script_path`：对应脚本路径
- `has_compile`：仓库构建方式，可选值见 枚举值规则（`enums.md`）
- `has_unittest` / `has_trigger`：是否启用单测、触发等能力
- `submodule` / `deep_clone`：是否包含子模块、是否深度克隆
- `repopermissions`：权限配置（`view_permission` / `download_permission` / `deploy_permission` / `configure_permission`）

## Version（get_version）常见字段

- `id`：版本在 SCM 系统内的唯一 ID
- `version`：版本号（例如 `1.0.0.3105`）
- `type`：版本类型，取值见 枚举值规则（`enums.md`）
- `repos` / `repo_name`：所属仓库 ID 与仓库名
- `create_user` / `create_date`：版本创建人与创建时间
- `pub_base`：发布基准（`branch_base` / `commit_base`）
- `branch_name` / `base_commit_hash` / `commit_url` / `git_url`：源码关联信息
- `builds`：构建流水线列表（不同架构通常对应不同元素）
- `builds[].arch`：流水线架构，取值见 枚举值规则（`enums.md`）
- `builds[].status` / `builds[].status_display`：流水线状态与展示文案，`status` 取值见 枚举值规则（`enums.md`）
- `builds[].job_id`：流水线 Job 标识
- `builds[].tar_url` / `builds[].tar_md5_url`：产物与校验文件地址
- `builds[].product_size` / `builds[].compile_time` / `builds[].finish_date`：产物大小、耗时、完成时间
- `builds[].error_message`：失败时错误信息
- `builds[].bytebuild`：ByteBuild 关联信息（`host` / `type` / `build_ids` 等，按实际返回为准）

## 枚举值

常见枚举取值说明见 枚举值规则（`enums.md`）。
