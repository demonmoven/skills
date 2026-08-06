# Gloop v0.2.8

## 修复：readonly 模式不再卡 apply

readonly 模式语义 = 直接在基准目录只读运行，无隔离工作区，无 diff，无 apply 概念。此前 readonly 委托法师 pass 后会尝试 apply，报错"readonly 模式不支持 apply"并标记 apply_failed，卡进 user_review。

- **自主闭环**（runAutoCloseLoop / runAutoApplyProcessor）：readonly 委托法师 pass 后直接 success，跳过 apply（与 design quest 同路径）
- **手动 apply**（ApplyQuest）：readonly 改为 no-op 成功（applied=true，发 EvtQuestApplied skipped），不再 recordApplyFailure
- **前端 canApply**：readonly 委托不显示 apply 按钮（questSelectors 排除 readonly）
- **connector chip**：readonly 委托/自动化隐藏"闭环后动作"（无工作区改动可提交）
