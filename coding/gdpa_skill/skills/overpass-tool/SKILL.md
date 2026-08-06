---
name: overpass-tool
description: 当需要获取服务 IDL 信息、接口方法列表或生成 overpass 代码时使用此 SKILL。Overpass 是公司内部的代码生成平台，此工具支持获取 IDL 基础信息、代码生成仓库信息、IDL 方法列表（包含 AI 增强的方法注释），以及触发代码生成。适用于需要了解服务接口定义、生成客户端代码的场景。
---

> **session_id 传递**：如果之前已经调用过 `gdpa-cli`，你将获得一个 SessionID，务必在之后的调用中都携带上此 session_id，并在 input 中加上此参数: `--input '{..., "session_id": "sess_xxx_xxx"}'`

# Overpass 工具

> **何时使用**: 当需要获取服务 IDL 信息、查看接口方法列表或生成 overpass 代码时调用此 SKILL。

## 使用方法

```bash
gdpa-cli run overpass --input '{"action": "<action>", "psm": "<psm>"}'
```

## 支持的 Action

| Action | 描述 | 必填参数 | 可选参数 |
|--------|------|----------|----------|
| `get_psm_idl_info` | 获取服务 IDL 路径和仓库信息（GitType: 0=gitlab, 1=gerrit） | psm | |
| `get_psm_repo_info` | 获取服务 overpass repo 路径、method 路径和主结构体路径 | psm | biz, branch |
| `get_psm_method_list` | 获取服务的 IDL 方法列表（含 AI 生成注释） | psm | branch |
| `generate_psm_repo` | 生成 PSM 对应的 overpass 代码（耗时分钟级） | psm | biz, branch |

## 输入参数

| 参数 | 类型 | 描述 |
|------|------|------|
| action | string | 必填，要执行的操作 |
| psm | string | 必填，PSM（项目服务名） |
| biz | string | 可选，业务线，默认 overpass（支持 overpass、webcastv2、tikcast、oecv2） |
| branch | string | 可选，分支，默认 master |

## 输出格式

```json
{
  "success": true,
  "action": "get_psm_idl_info",
  "data": { ... }
}
```

## 示例

```bash
# 获取服务 IDL 基础信息
gdpa-cli run overpass --input '{"action": "get_psm_idl_info", "psm": "tiktok.user.service"}'

# 获取服务 overpass 代码生成信息
gdpa-cli run overpass --input '{"action": "get_psm_repo_info", "psm": "tiktok.user.service"}'

# 指定业务线和分支
gdpa-cli run overpass --input '{"action": "get_psm_repo_info", "psm": "tiktok.user.service", "biz": "overpass", "branch": "develop"}'

# 获取服务的 IDL 方法列表（包含 IDL 方法名、GO 方法名、AI 分析的方法注释）
gdpa-cli run overpass --input '{"action": "get_psm_method_list", "psm": "tiktok.user.service"}'

# 生成 PSM 对应的 overpass 代码（注意：耗时可能是分钟级别）
gdpa-cli run overpass --input '{"action": "generate_psm_repo", "psm": "tiktok.user.service"}'

# 指定业务线和分支
gdpa-cli run overpass --input '{"action": "generate_psm_repo", "psm": "tiktok.user.service", "biz": "webcastv2", "branch": "develop"}'
```

## 注意事项

- `generate_psm_repo` 会硬等代码生成完成，耗时可能是分钟级别，请耐心等待
- 需要确保网络可以访问 MCP 服务器
- 需要安装 `npx` 命令（通常随 Node.js 一起安装）
