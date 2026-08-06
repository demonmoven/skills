# 枚举值规则

本文件用于说明 `get_repo` / `get_version` / `get_version_list` 等返回中常见字段的枚举取值含义。

## Version 相关枚举

### type（版本类型）

| key | desc |
| --- | --- |
| `online` | 线上版本 |
| `offline` | 线下版本 |
| `test` | 测试版本 |

### status（流水线状态）

| key | desc |
| --- | --- |
| `not_build` | 未触发构建或该架构未参与构建 |
| `prepare` | 构建准备中（资源/环境准备阶段） |
| `queue` | 已进入队列，等待调度执行 |
| `building` | 编译/构建中 |
| `build_ok` | 构建成功，产物已生成 |
| `uploading` | 资源文件上传中 |
| `upload_failed` | 资源文件上传失败 |
| `build_failed` | 构建失败（编译或构建步骤失败） |

### state（版本状态聚合）

`get_version_status` 会将底层 `status` 聚合为 `state`

**status 分类取值：**
- SUCCESSFUL_STATUS：`build_ok`
- FAILED_STATUS：`build_failed`、`upload_failed`
- RUNNING_STATUS：`prepare`、`building`、`uploading`、`queue`

| status 分类 | state | desc | error |
| --- | --- | --- | --- |
| SUCCESSFUL_STATUS | `passed` | 构建成功 |  |
| FAILED_STATUS | `failed` | 构建失败 | `build version failed` |
| RUNNING_STATUS | `running` | 构建进行中 |  |
| `not_build` | `not_build` | 未触发构建 |  |
| 其他 | `failed` | 未识别的构建状态 | `unknown build status` |

### arch（构建架构）

| key | desc |
| --- | --- |
| `x86_64` | x86_64 架构 |
| `aarch64` | ARM64 架构（通用） |
| `aarch64_v8` | ARM64 v8 架构 |
| `aarch64_v9` | ARM64 v9 架构 |

## Repo 相关枚举

### type（仓库编程语言）

| key | desc |
| --- | --- |
| `go` | Go |
| `nodejs` | Node.js |
| `cpp` | C++ |
| `java` | Java |
| `python` | Python |
| `rust` | Rust |

### has_compile（仓库构建方式）

| key | desc |
| --- | --- |
| `0` | 跳过编译 |
| `1` | 使用构建脚本 |
| `2` | 使用 `atum.yaml` |
| `3` | 使用 `bazelisk` |
