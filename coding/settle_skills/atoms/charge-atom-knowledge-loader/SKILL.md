---
name: charge-atom-knowledge-loader
description: 计费域知识源加载原子。读取 settle_skill/config/knowledge_sources.yaml,按清单逐项拉取文档内容到本地缓存,返回已加载路径数组。仅供 auto-develop 路由层和子流程调用。
---

# charge-atom-knowledge-loader

## 定位
计费域**强基线知识**的统一加载器。每次 auto-develop 路由执行时调用一次,从 `config/knowledge_sources.yaml` 读清单,按 `loader` 字段分发到对应拉取器,把文档内容缓存到本地。

**仓库只感知文档地址,不感知内容**;内容每次运行实时拉取。

## 调用契约

### 输入
- `--config`:yaml 清单路径(默认 `<repo>/config/knowledge_sources.yaml`)
- `--scope`:逗号分隔的 scope 过滤(默认 `charge`,只加载 scope 命中的源)
- `--refresh`:`auto`(默认,按 yaml 中 `refresh_policy`)| `force`(强制重拉)| `cache-only`(只读缓存,失败即 abort)

### 输出(stdout JSON)
```json
{
  "loaded": [
    {"key":"payment_charge_whitepaper","cache_path":"/home/mira/files/charge-knowledge/payment_charge_whitepaper.md","bytes":26788,"hit_cache":true,"loader":"lark-docs-skill"}
  ],
  "failed": [],
  "warnings": []
}
```

### 失败语义
- 任一 `required: true` 的源加载失败 → exit 1,把错误打到 stderr,调用方按"硬阻断"处理
- `required: false` 失败 → 进入 `warnings`,exit 0

## 使用方式

```bash
python3 load.py --config <repo>/config/knowledge_sources.yaml --scope charge
```

## Loader 实现说明

| loader 字段 | 调用 |
|---|---|
| `lark-docs-skill` | `cd /data/plugins/market/lark-docs-skill/skills/lark-docs-skill && python3.11 -m lark_docs read-doc <url> --extract-field content` |
| `bytedance-feishu` | `bytedcli --json bytedance-feishu docs read --url <url>`(预留,未实现) |
| `web-fetch` | `curl -fsSL <url>`(预留,未实现) |

当前仅实现 `lark-docs-skill`,其它 loader 返回 `not_implemented` 错误。

