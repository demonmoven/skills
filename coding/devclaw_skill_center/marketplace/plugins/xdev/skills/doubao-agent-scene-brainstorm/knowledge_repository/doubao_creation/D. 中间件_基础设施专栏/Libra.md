# Libra

> 字节内部的**实验切流平台**（A/B testing / 灰度发布）。

---

## 一、概念区分

| 概念 | 含义 |
|---|---|
| **Libra**（平台） | 字节内部 A/B 实验平台（外部基础设施） |
| **AB 实验参数**（`option.XXX`） | 通过 Libra 平台下发到代码的实验参数 |
| **AB 维度常量**（`constant/ab_params.go`） | 代码中定义的 AB key 常量 |

---

## 二、用途（豆包-创作链路）

参考 [`../../../locators/libra_locator.md`](../../../locators/libra_locator.md)。

| 用途 | 散落点 |
|---|---|
| **架构代际灰度**（新老 ReAct 切换） | `creation_agent/handler/handler.go::SupportAgent26` ★最重要 |
| 模型实现选择（Ark vs ModelAPI） | `SupportAgent26Ark` |
| Libra 常量定义 | `creation_access/constant/libra.go` |
| 评测专用 AB 维度 | `creation_evaluation/constant/ab_params.go` |
| SP 来源选择（Libra 实验 SP 版本） | phase3 `sp_manager.go` |

详见 [libra locator §三](../../../locators/libra_locator.md#三7-处散落点全清单按密集度--关键度排序) 7 处全清单。

---

## 三、代码引用方式

```go
// 通过 option 包装（推荐）
if option.SupportAgent26(ctx) {
    // 命中实验
}

// 直接调 SDK（低级用法）
val := libra_sdk.GetParam(ctx, "experiment_key")
```

> 具体的 Libra SDK 包路径以代码为准（`creation_access/constant/libra.go` 中 `LibraToken` / `LibraDoubaoAppID` 等常量是 SDK 接入参数）。

---

## 四、Libra vs TCC 的边界

| 维度 | Libra | TCC |
|---|---|---|
| 适合 | 短期 A/B 对照、灰度 | 长期固化、租户级开关、降级兜底 |
| 配置生效 | 实时 | 实时 |
| 平台 | Libra 平台 | TCC 平台 |
| 创作链路使用密度 | 高（上游） | 高（上下游） |

---

## 五、写需求时的"完整 Libra 标准动作"

参考 [libra locator §六](../../../locators/libra_locator.md#六libra-灰度的标准动作模板)。

### 5.1 新增 Libra 实验

- [ ] 命名遵循 `creation_<能力>_<场景>_v<版本>`
- [ ] Libra 平台创建实验
- [ ] 代码加 option / 常量
- [ ] 使用点加 if 分支
- [ ] 上线前 dry-run 切流 0%
- [ ] 推全 100%

### 5.2 下线 Libra 实验（**最容易遗漏，导致屎山的源头**）

- [ ] grep 实验 key
- [ ] 删除所有 if 分支
- [ ] Libra 平台关闭实验
- [ ] 避免遗留 dead code

---

## 六、写需求时的检查

- [ ] 改动是否涉及**架构代际灰度**？（# 1 SupportAgent26）
- [ ] 改动是否需要新建实验？走 Libra 不走 TCC？
- [ ] 实验 key 命名是否符合约定？
- [ ] 实验**下线计划**是否写入方案？

---

## 七、未来：`skill_libra_change`

王瑶访谈明确希望"Libra 增/删的标准化 skill"。

- 本期 brainstorm skill 仅做 **read 侧**（盘点 / 导览）
- write 侧 skill `skill_libra_change` 是后续工作

详见本 skill 的调研报告（在 `stone/devclaw_skills_center/feat-dev/2026/04/27/183429-doubao-agent-scene-brainstorm-skill/research.md` § 七 范围外）。该文件不在 worktree 内，不做 markdown 内部链接。
