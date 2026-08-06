# [项目名] 文档索引

> 本目录承接仓库的专题文档和中层导航。
> Agent 总入口见 `../AGENTS.md`，人类入口见 `../README.md`。

## 1. 阅读顺序

- Agent：`../AGENTS.md` → 本页 → 模块入口 / 专题文档 / `docs/reference` / `docs/guidance`
- 人类：`../README.md` → 本页 → `../ARCHITECTURE.md` 或对应模块文档
- 需要代码边界：直接看 `../ARCHITECTURE.md`

## 2. 文档分层

| 层级 | 位置 | 用途 |
|---|---|---|
| 总入口 | `../AGENTS.md` | 仓库任务路由、协作约束 |
| 人类入口 | `../README.md` | 项目介绍、快速开始 |
| 架构入口 | `../ARCHITECTURE.md` | 代码地图、边界、不变量 |
| 中层索引 | `docs/AGENTS.md`（本文件） | 专题文档汇总、按任务导航 |
| 参考资料 | `reference/` | 稳定事实说明——API、架构、配置 |
| 操作手册 | `guidance/` | SOP——本地启动、测试、发布流程 |
| 执行计划 | `plans/` | 执行计划（proposal → active → completed） |
| 规则约束 | `rules/` | 架构不变量、工作原则 |
| 质量追踪 | `quality/` | 技术债日志 |
| 开发期 Skill | `../.skills/` | 仓库 Skill 单一事实来源 |

## 3. 按任务导航

| 场景 | 文档 |
|---|---|
| 仓库总览与快速开始 | `../README.md` |
| 架构和代码边界 | `../ARCHITECTURE.md` |
| _TODO: 根据仓库实际文档填充_ | |
| 代码惯例和模式 | `reference/code-patterns.md` |
| 运行时行为和数据流 | `reference/runtime-behavior.md` |
| 外部服务集成 | `reference/integrations.md` |
| 架构决策记录 | `reference/adr/` |
| 排查常见问题 | `guidance/debugging-playbook.md` |
| 避开已知坑点 | `guidance/common-pitfalls.md` |
| 知识缺口追踪 | `quality/knowledge-gaps.md` |

## 4. 本目录当前专题

```text
docs/
├── AGENTS.md              # 本文件
├── reference/             # 稳定事实说明
│   ├── code-patterns.md
│   ├── runtime-behavior.md
│   ├── integrations.md
│   └── adr/
├── guidance/              # 操作手册
│   ├── local-dev-setup.md
│   ├── debugging-playbook.md
│   └── common-pitfalls.md
├── plans/
│   ├── proposal/
│   ├── active/
│   ├── completed/
│   └── artifacts/
├── rules/
│   ├── invariants.md
│   └── golden-principles.md
└── quality/
    ├── debt-log.md
    └── knowledge-gaps.md
```

## 5. 维护规则

- 新增**事实说明型**文档（API、架构分析、配置参考）→ 放入 `reference/`
- 新增**操作型**文档（启动指南、测试手册、发布流程）→ 放入 `guidance/`
- 模块入口型文档（`README.md`、`AGENTS.md`）保留在模块根目录
- 代码目录里的操作型 README 应迁入 `docs/guidance/` 体系
- `reference/`、`guidance/`、`plans/`（含 `artifacts/`）统一收敛到 `docs/`
- 新增开发期 Skill 时，只改 `../.skills/`，不要在 `.claude/skills/`、`.opencode/skills/` 或 `.codex/skills/` 分别写一份
