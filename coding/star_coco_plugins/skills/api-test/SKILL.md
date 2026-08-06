---
name: api-test
description: 当用户要求"接口测试"、"API 测试"、"发送请求"、"测试 RPC 接口"、"测试 HTTP 接口"、"调用接口"时使用此 skill。支持 RPC 和 HTTP 两种接口测试方式。
tags:
  - byte-skill
---

# API Test

用于发送接口测试请求，支持 RPC 与 HTTP 两种方式。

## Prerequisites

当调用本 skill 时，系统会返回 "Base directory for this skill"。

需要将此 base directory 与相对路径 `assets/config.json` 拼接成完整路径传给 `--config` 参数：

```bash
# <base-dir> 为系统返回的 Base directory for this skill
CONFIG="<base-dir>/assets/config.json"
```

## 流程
0. **依赖安装**：
   - Run `curl -LsSf https://astral.sh/uv/install.sh | sh`
   - Run `uv tool install --force "git+https://code.byted.org/bytedance/byte-skill.git#subdirectory=byte-cli" --index https://bytedpypi.byted.org/simple`

1. **确认服务信息**：检查项目的 AGENTS.md 或 CLAUDE.md 中是否存在 `Project Meta` 章节，包含 PSM 和服务类型（RPC / HTTP）信息。
   - 如果存在，直接使用。
   - 如果不存在，**必须先询问用户**以下信息：
     - 服务的 PSM（Process Service Module）名称
     - 服务类型：RPC 还是 HTTP
   - 获取信息后，**询问用户是否将这些信息补充到项目的 AGENTS.md / CLAUDE.md 文件的 `Project Meta` 章节中**（推荐，避免重复询问）。

2. **加载对应指引文档**：根据服务类型，读取对应的 references 文档：
   - RPC 服务：读取 `references/rpc-guide.md`
   - HTTP 服务：读取 `references/http-guide.md`

3. **按指引文档执行测试流程**。
