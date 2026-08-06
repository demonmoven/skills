# Environment Management Guide

## SearchEnvs API 详细说明

### Request Parameters

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|:---|:---|:---|:---|:---|
| `search` | string | Yes | - | 环境名称搜索关键词 |
| `page_num` | integer | No | 1 | 页码 |
| `page_size` | integer | No | 10 | 每页数量 |
| `detail` | boolean | No | true | 是否返回详细信息 |
| `service_meta` | boolean | No | false | 是否返回服务元数据 |
| `with_auth_check` | boolean | No | false | 是否检查权限 |
| `with_auth_list` | boolean | No | true | 是否返回权限列表 |

### Region 参数映射

`--region` 参数决定了查询的 ControlPanel 和环境类型：

| Region 参数 | standard_envs | type | 说明 |
|:---|:---|:---|:---|
| `BOE` | (不传) | `boe_feature,boe_base` | BOE 预发环境 |
| `CN` | `online_cn` | `ppe` | CN 正式环境 |
| `I18N` | `online_i18n` | `ppe` | I18N TikTok 环境 |
| `I18N-BD` | `online_i18nbd` | `ppe` | I18N ByteDance 环境 |

### Response Structure

```json
{
  "items": [
    {
      "id": 255034413,
      "name": "boe_fund",
      "status": 0,
      "type": "boe_feature",
      "standard_env": "boe",
      "env_managers": ["goofydeploy", "suwei.8"],
      "env_collaborators": ["字节跳动", "懂车帝"],
      "services": [
        {
          "service": "14663|3M-资金运营平台前端",
          "service_type": "web",
          "env": "boe_fund"
        }
      ],
      "created_at": "2025-09-15T10:31:58+08:00",
      "updated_at": "2025-09-15T10:31:58+08:00",
      "auth_type": 0,
      "desc": ""
    }
  ],
  "pagination": {
    "total": 91,
    "total_page": 10,
    "page_num": 1,
    "page_size": 10
  }
}
```

### Status 含义

| status | 说明 |
|:---|:---|
| 0 | 正常 |
| 2 | 已销毁 |

### auth_type 含义

| auth_type | 说明 |
|:---|:---|
| 0 | 无权限限制 |
| 1 | 需要申请 |
| 2 | 受限访问 |

## Create Env

环境创建流程在环境平台 UI 完成：https://bits.bytedance.net/env/life/create

创建时需要填写：
- **环境名称**: 以 `boe_` 为前缀
- **环境用途**: 通常选 `test_function`
- **隔离方式**: 通常选 `env`
- **流量基线**: 可选 `prod` (基于生产流量)
- **管理员**: 指定环境管理人

## Manage Env

环境管理（添加/移除服务）在环境平台完成：

URL 模板: `https://bits.bytedance.net/env/life/{standard_env}/boe/detail/{env_name}/service`

支持的服务类型：
- **TCE**: 容器化服务 (service_type: `tce`)
- **TCC**: 配置中心
- **FaaS**: 无服务器函数
- **Web**: 前端应用 (service_type: `web`)
- **Mock TCE**: 模拟服务 (service_type: `mock_tce`)
