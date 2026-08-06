---
name: byte-env
description: >-
  This skill should be used when the user asks to "查询环境", "搜索环境",
  "search env", "创建环境", "create env", "管理环境", "manage env",
  "查看 BOE 环境", "查看 PPE 环境", "环境列表", or needs to search, create,
  or manage ByteDance environments (BOE/CN/I18N/I18N-BD).
tags:
  - byte-skill
---

# ByteDance Environment Management

管理字节跳动环境平台上的环境（BOE / CN / I18N / I18N-BD）。

## ControlPanel 与 standard_env 映射

| ControlPanel | standard_env | env type | 说明 |
|:---|:---|:---|:---|
| BOE | `boe_feature` | `boe_feature,boe_base` | 预发布测试环境 |
| CN | `online_cn` | `ppe` | 中国大陆正式环境 |
| I18N (I18N-TT) | `online_i18n` | `ppe` | 国际化 TikTok 环境 |
| I18N-BD | `online_i18nbd` | `ppe` | 国际化 ByteDance 环境 |

## Operations

### 1. Search Envs (搜索环境)

通过环境平台 API 按名称搜索环境。

1. 确认用户要搜索的 ControlPanel（BOE / CN / I18N / I18N-BD），默认 BOE
2. 确认搜索关键词（env 名称或前缀）
3. 使用 `byte-cli` 执行 `SearchEnvs` 请求：

   当调用本 skill 时，系统会返回 "Base directory for this skill"。

   需要将此 base directory 与相对路径 `assets/config.json` 拼接成完整路径：

   ```bash
   # <base-dir> 为系统返回的 Base directory for this skill
   CONFIG="<base-dir>/assets/config.json"

   byte-cli --config "$CONFIG" \
     --component ByteEnv \
     --endpoint SearchEnvs \
     --region BOE \
     --params '{"search": "<keyword>", "page_num": 1, "page_size": 10}'
   ```

4. 从返回的 `items` 数组中提取关键信息展示给用户：
   - `name`: 环境名称
   - `id`: 环境 ID
   - `status`: 状态 (0=正常, 2=已销毁)
   - `env_managers`: 管理员列表
   - `services`: 绑定的服务列表（每个含 `service` 名称和 `service_type`）
   - `created_at` / `updated_at`: 时间

如需翻页，调整 `page_num` 参数。

### 2. Create Env (创建环境)

环境创建流程较为复杂，引导用户手动操作：

1. 告知用户打开环境平台创建页面：https://bits.bytedance.net/env/life/create
2. 提供创建环境时需要填写的关键字段说明：
   - 环境名称（`boe_` 前缀）
   - 环境用途（如 `test_function`）
   - 隔离方式（通常为 `env`）
   - 管理员

### 3. Manage Env (管理环境)

环境管理（添加 TCE/TCC/FaaS 服务）操作在环境平台完成：

1. 引导用户打开环境详情页：
   `https://bits.bytedance.net/env/life/{standard_env}/boe/detail/{env_name}/service`
2. 其中 `{standard_env}` 根据 ControlPanel 映射表确定，`{env_name}` 为具体环境名
3. 用户可在该页面添加/移除服务绑定

## Reference

详细的 API 参数说明参见 `references/env-guide.md`。
