# Gloop v0.2.7

## 修复

- **端口冲突不再静默换端口**：`bindWithFallback` 在指定端口（含默认 37317）被占时直接报错退出，提示用 `gloop status` / `gloop stop` 处理。此前端口冲突会静默扫描下一个端口（MaxPortScan=20），导致多实例并立、pidfile 互相覆盖、用户不知实际端口。仅 `port<=0`（显式请求自动选）时保留扫描能力。
- **新建自动化委托流程排版**：委托流程（3 个 option-card）从 AdvancedSection 的 `grid-2` 半宽列提到主区 `form-grid`，靠 `flow-field` 独占整行；对齐「发起委托」页。AdvancedSection 只留工作目录、工作区模式等紧凑字段。根因是委托流程 label 漏标 `flow-field` 且被塞进半宽列，3 卡挤爆。

## 新增：Connector 感知

不单独开页面，按「配置 / 声明 / 展示」三层分散到现有页面：

- **配置态**：Settings 新增 Connectors 区，可启用 Git connector 并配置 remote / base_branch / author / push。后端 `GlobalConfigPatch` 补 `Connectors` 字段 + Apply 合并。
- **声明态**：新建自动化、编辑自动化补「闭环后动作」chip（对齐发起委托页）。后端 automation update 链路（`updateAutomationReq` / `AutomationUpdate` / `UpdateAutomation`）补 `Connectors` 字段。
- **展示态**：委托详情 facts 显示该 quest 声明的 connectors（如「提交到 gloop 分支」），让用户在详情页感知 quest 闭环后会触发哪些外部动作。

Connector 运行态结果（派发成功/失败、分支名）回写到 quest meta 并在详情页展示，作为后续增强。
