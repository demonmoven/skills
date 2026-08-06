# Gloop v0.2.17

## 重构：~ 展开单点收敛到 LoadQuest

v0.2.16 在 CreateQuest 和 Prepare 两处分散展开 `~`，不是全局最优——automation update 路径仍漏网，且每加一个写入点都要记得展开。

**重构为单点收敛**：在 `QuestStore.LoadQuest` 出口统一展开 `BaseWorkingDir`。

### 为什么这是全局最优

- **单点收敛**：LoadQuest 是所有 quest meta 读取的唯一出口（ListQuests、adapter、engine 全走它）。在这一处展开，所有下游（Prepare/apply/diff/cleanup/automation/connector）拿到的都是绝对路径，无需各自展开。
- **不丢信息**：存储保留用户原始输入（`~` 不变），只在读取时展开成绝对路径。
- **覆盖所有写入路径**：无论 CreateQuest、automation update、还是未来新增的写入点，读出来都是展开的。不再有"漏网之鱼"。
- **去掉冗余**：CreateQuest 和 Prepare 里的分散展开移除（LoadQuest 已处理）。

### 变更

- `quests.go` LoadQuest：加 `m.BaseWorkingDir = ExpandDir(m.BaseWorkingDir)`
- `quest_lifecycle.go` CreateQuest：移除 `wd = ExpandDir(wd)`（冗余）
- `workspace_prepare.go` Prepare：移除 `baseDir = ExpandDir(baseDir)`（冗余）
