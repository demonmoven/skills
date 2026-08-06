# CHANGELOG


## 2026-05-16 至 2026-05-22 <!-- commit: 641c3b48 -->

### 研发流程与 Skill
- 新增 Flou Trae Hook 机制：通过 `flou-cli hook` 在 Trae 中接入 Flou，让 agent 在编辑器侧自动感知任务阶段并执行配套动作
- 新增知识插件能力，agent 调用前会先经过项目记忆与知识检索，提示词与回流知识可以被多个流程复用
- 强化「先查记忆再调用工具」的环境感知约束，更新 `SKILL.md`、`references/project-memory.md`、`references/workflow-management.md`、`statics/interactive_flow.md`
- 沉淀「技能评测最优使用方法」项目记忆，便于后续 agent 直接参考最佳实践


### CLI 体验
- 修复 `project init` 在 Linux 下由于安装路径处理不当导致的报错
- 同步修复 `SKILL.md`、`statics/sdd-plus-flow.md`、`statics/sdd_flow_spec.md`、`references/todo-flow.md` 与 CLI 行为不一致的文档问题


## 2026-05-08 至 2026-05-15 <!-- commit: 89da5b2 -->

### Flou CLI 安装与元信息
- 安装脚本改为通过安装目录中的 `flou-cli` 校验版本，避免新 PATH 尚未生效时误判安装失败
- `flou-cli install` 支持不可写 npm 全局目录时回退到用户目录，并修复 `gdpa-cli` 安装后登录命令不在 PATH 时的执行路径
- 扩展安装目标 agent 识别范围，补充 codex、cursor、gemini、openclaw、opencode、ttadk 等 skill 目录落点
- 任务初始化和 workspace snapshot 增加 Flou CLI 路径与版本信息，便于后续追踪任务由哪个本地 CLI 版本生成

### 任务初始化与工作区
- 放宽 `task init` 对空工作目录的判断，允许仅包含点号文件或空子目录的目录作为初始化目标，同时继续拒绝 git/worktree 目录
- 多仓工作目录复用增加 git artifact 检测与错误提示，降低误把已有仓库当作 Flou 工作区的风险
- 补充 task init、workspace API、TOS snapshot 等路径的版本元数据与回归测试

### Agent Box Bridge 与协作工作区
- 初始化 Agent Box host tool bridge 投影，新增 Dashboard、Memory、Kanban、Coding Agent、todo、notebook、channel、skill、search 等宿主能力的桥接说明与脚本
- 更新 AGENTS/SOUL 协作约定，要求宿主能力优先通过 `.gdpa-agent-box/dialogue-tools/agentbox_dialogue_tool.py` 调用真实注册工具
- 补充 Agent Box skill bridge 投影，支持外部对话 agent 查询、查看、创建和维护可复用 skill

### 研发流程与 Skill
- 新增 `flou-issue-management` skill，用于创建、查询、更新和跟踪 Flou 项目 issue
- 新增 TCC 配置自助接入流程 `tcc_config_flow`，覆盖配置分析、热点配置推荐、开发验证和 Flou 内置测试流程门禁
- 更新 `interactive_flow` 与 Flou skill 入口，使任务初始化、issue 管理和 TCC 配置流程能够被正确路由

### CI 与回归
- 新增 Codebase CI 配置，自动执行 `go build ./...` 与 `go test -coverprofile=coverage.out ./...`
- 增加 `main.go` 自更新生命周期守护测试，防止入口链路误删 `selfupdate` 关键调用
- 补充安装 fallback、task init 目录识别、agent 校验等单元测试

## 2026-04-20 至 2026-05-07 <!-- commit: 74809c1 -->

### Workspace API 与 VS Code 扩展
- 新增隐藏 workspace API 服务，支持工作区发现、管理、远程访问、记忆视图、快照和任务初始化等接口
- 新增 VS Code 扩展入口，支持在编辑器内查看任务信息、推进任务初始化、展示 mermaid/todo-flow 等内容
- `flou server` 增加 `stop`、`status` 能力，便于管理本地服务进程
- 新增工作区快照 TOS 上传链路，支持任务初始化和项目初始化时产出可共享的 workspace 资源

### 任务初始化与项目接入
- 重构 `task init` 单仓/多仓初始化架构，拆分 orchestration、single、multi、shared 等逻辑并补充测试
- 单仓库识别到 submodule 时支持自动拉取，提升初始化后的代码跳转可用性
- 优化项目工作目录推荐：空目录优先作为当前工作目录，任务命令补充相关校验与测试
- 项目初始化补充 workspace API 配置落盘和 TOS 上传相关逻辑

### SDD-Plus 与依赖安装
- 串联 SDD-Plus 流程，新增 `sdd-plus-flow.md` 并接入 spec-coder 场景路由
- `flou-cli install` 自动安装 spec-coder 依赖工具，支持 `feature_docs`、`scc` 等工具检查与安装
- 优化 `feature_docs` 的 go install 依赖处理和查找 skill 文档说明
- 修复本地不存在 flou skill 时 `flou-cli install` 报错的问题

### 配置与记忆
- 配置写入采用固定 JSON 顺序，减少 `.flou/config.json` diff 抖动
- memory 查询结果返回记忆来源路径，便于定位命中的研发记忆
- 更新 AGENTS/interactive flow 相关提示，使任务初始化、记忆和流程约束与 CLI 行为保持一致

### 安装脚本
- 更新 `install-skill-and-bin.sh` 与构建脚本，改进本地 skill/CLI 安装体验
- 同步更新 Flou CLI 版本标记和安装脚本中的环境处理

## 2026-03-27 至 2026-04-03 <!-- commit: 20a3b0f -->

### 新增SDD-Plus流程
- 支持并发任务执行
- 支持spec强规格文件约束
- 支持阶段门禁控制

### 多仓能力优化
- 多仓工作区文件固化，避免代码不可跳转
- AGENT.md 文件支持多仓配置
  
## 2026-03-24 至 2026-03-26 <!-- commit: 5129b90 -->

### task init
- 新增多仓库分支支持

### 测试能力
- 新增 test 命令，支持网页模块自动化测试
- 新增会话录制和快照功能

### 其他
- 调整 skill 目录结构
- 增加发布脚本

## 2026-03-18 至 2026-03-22 <!-- commit: ba945bd -->

### 记忆管理
- 新增日志异步分析和记忆检索功能
- 修复 task init 搜索记忆时加载完整配置

### 项目初始化
- 新增 project init 命令，支持AGENTS.md 文件注入 Flou 能力
- task 命令新增 --create-worktree 支持

### 环境初始化
- 添加默认 API key，避免环境变量未设置时报错
- 环境澄清功能优化

### 安装
- 本地 skill 安装功能
- 重构 skill 安装逻辑，移除 npx skills add 依赖
- 支持多技能目录安装并增强软链接回滚

### 其他
- 压缩 skill 文档
- 更新交互流程说明和输出提示

## 2026-03-13 至 2026-03-17 
#### 配置管理
- 支持配置包目录
- 支持配置记忆读取方式

#### 记忆管理
- 修复临时目录损坏问题
- 支持 flou 通用研发记忆加载
- 支持外部项目记忆加载

#### 日志录制
- 新增日志录制功能

#### Sub-Agent
- 新增 sub-agent chat 功能，支持 function calling 工具调用


## 2026-03-18 至 2026-03-22

### 新增内置研发流程
- Hotfix 热修复流程 - 紧急修复线上问题
- Oncall 值班响应流程 - 值班问题处理
- Spike 技术预研流程 - 技术方案调研
- 
### 安装命令优化
- 修复安装依赖检查问题
- 
### Bug 修复
- 修复 memory 命令 panic 问题

## 1.0.3 - 2026-03-10
- 工作流程优化
- 研发流程规范
