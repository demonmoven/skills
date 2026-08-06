# Gloop v0.2.15

## 新建委托现在能感知 HOTL 模式

此前 warrior/mage 完全不知道当前是否处于 HOTL 自主闭环模式——`quest info` 不返回 hotl 字段，context pack 也不注入 HOTL 提示。agent 盲跑，不知道交付后会自动闭环还是等人工。

### 修复

**1. context pack 注入 HOTL 模式块（`prompt.InjectHOTLMode`）**

warrior 和 mage 的 context pack 现在都注入 `hotl_mode` 控制块：
- HOTL 开启：告知 agent "法师 pass 后自动 apply + 闭环，无需等人工审核"，行为预期是专注交付质量不留半成品，需介入时主动 quest ask
- HOTL 关闭：告知 agent "pass 后进用户终审，需手动确认"

在 `pipeline.go` 的 warrior context pack 和 mage review context pack 两处注入。

**2. `quest info` CLI 返回 `hotl_auto_close` 字段**

warrior 通过 `gloop quest info` 能读到当前 hotl_auto_close 状态（从 config 加载）。

### 技能安装

`gloop-iteration-workflow` 技能装到 `~/.claude/skills/`，所有 agent（不只 gloop warrior）都能用。
