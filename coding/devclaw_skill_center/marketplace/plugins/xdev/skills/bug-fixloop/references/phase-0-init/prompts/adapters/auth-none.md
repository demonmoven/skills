# Phase 0 / Auth / Adapter: none

> PROFILE=none 的认证 adapter。**直接跳过所有认证检查**。
>
> 由 `precheck.md` 在 PROFILE=none 时调用。

## 适用场景

PROFILE=none 时使用。用户的服务自部署在 BASE_URL，不需要 bytedcli / 不需要双站点认证 / 不需要 AGW service-id。

## 流程

什么都不做。

```bash
echo "PROFILE=none: 跳过 bytedcli 检查 / 双站点认证 / AGW service-id 查找"
```

## 产出

主 context 不需要持有任何 bytedcli 相关变量。后续 phase 的 adapter 都是 none 版本，它们也不依赖 bytedcli。

## 与 bytedance adapter 的差异

| 维度 | bytedance | none |
|------|--------|------|
| bytedcli 二进制检查 | 是 | 否 |
| bytedcli sub-skills 检查 | 是 | 否 |
| 双站点认证 | 是 | 否 |
| AGW service-id 查找 | 是 | 否 |
| BYTEDCLI_SKILLS_DIR 变量 | 必须设置 | 不设置 |

## 失败处理

无（noop adapter 不会失败）。

## 注意

PROFILE=none 时，整个 fix-loop 流程都不会用到任何 byted 内场工具。如果用户在 wizard 中误选 none 但实际服务在字节内场，建议改回 bytedance-tce 重试。
