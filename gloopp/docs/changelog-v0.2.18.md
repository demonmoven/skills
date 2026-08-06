# Gloop v0.2.18

## 修复：warrior 产物声明链路打通（产物 md 看不到 / 冒险者结论渲染 inlinecode）

### 现象

design quest 普遍存在两个问题：
1. 「冒险者结论」区块把 `docs/xxx.md` 渲染成 inline code（一段短文本）
2. 「交付产物」区空，产物 md 看不到、「查看产物」按钮不出现

### 根因

`gloop phase done` CLI 命令**硬编码** `deliverables = nil`（`cmd_agent_syscall.go` 直接传 nil 给 `phaseCheckpoint`），warrior 即使写了 md 文件也无法通过 CLI 注册成 quest output。导致：

- `meta.outputs` 永远为空 → `pickSummaryArtifact` 返回 null → `summaryArtifactText` 为空
- `selectQuestProducerArtifact` 一路回退到 warrior 的 turn summary 文本（那段含 `docs/xxx.md`）
- `AgentMessageRenderer` 把 `docs/xxx.md` 当 inline code 渲染 → 用户看到「冒险者结论是 inlinecode」
- 「交付产物」区因 outputs 空而无内容

后端 `extractWarriorDeliverables` 机制本身完整（会把 deliverables 写入 q.Outputs），**只差 CLI 入口**。

### 修复

**CLI 入口**：`gloop phase done` 新增 `--deliverable` 可重复参数，格式 `name=<显示名>,kind=<document|file|link>,path=<相对当前目录的文件路径>或url=<外部链接>,description=<说明>`。

**path 归一化**（第一性原理）：warrior 在 worktree 里 cwd 是工作区根，它看到的路径是 `docs/x.md`；但 `OpenArtifact` 用 `qs.dir(qid)` + StoragePath 拼接，StoragePath 必须能定位文件。归一化策略：
- 优先转**相对 quest 目录的路径**（如 `work/docs/x.md`）——可移植，directory_copy 模式（workspace 在 quest 目录内）走这条
- worktree 模式（workspace 在 quest 目录外）时 `filepath.Rel` 得到 `..` 前缀，fallback **绝对路径**

**后端统一 IsAbs**：`OpenArtifact` 和 `ArtifactPath` 此前直接 `filepath.Join(questRoot, StoragePath)`，不支持绝对路径；`verifyOutputArtifact` 已支持。三处统一加 `IsAbs` 判断：绝对路径直接用，相对路径 Join quest 目录。worktree 模式 workspace 在 quest 目录外也能正确读取。

**warrior prompt**：`warrior_execution_instruction.md` + pipeline hint + prompt fallback 三处告知 warrior 用 `--deliverable` 声明产出物，否则评审与首页看不到产物。

### 为什么这是全局最优

- **warrior 不感知 quest 目录结构**：只传相对 cwd 的路径，CLI 负责归一化，抽象不泄漏
- **路径可移植优先**：directory_copy 模式存相对路径，meta.json 不含机器特定绝对前缀；worktree 模式才 fallback 绝对路径
- **后端三处一致**：OpenArtifact / ArtifactPath / verifyOutputArtifact 统一 IsAbs 语义，不再分裂
- **前端零改动**：`isTextualArtifact` 只看 `artifact.name` 后缀（不看 StoragePath），`artifactURL` 走后端按 id 打开，StoragePath 绝对/相对都透明

### 变更

- `internal/cli/cmd_agent_syscall.go`：新增 `deliverableFlag`（flag.Value）解析可重复 `--deliverable`；`runPhaseDone` 透传 deliverables + path 归一化（相对 quest 目录优先，worktree fallback 绝对）
- `internal/orchestrator/macro_loop.go`：`extractWarriorDeliverables` 支持 `path` 字段（优先 path 回退 url），映射到 `DeclaredOutput.URL` 作为 StoragePath
- `internal/fsstore/artifacts.go`：`OpenArtifact` / `ArtifactPath` 加 `IsAbs` 判断，与 `verifyOutputArtifact` 一致
- `internal/prompt/prompt.go` + `prompts/warrior_execution_instruction.md` + `internal/orchestrator/pipeline.go`：warrior 执行指令告知用 `--deliverable` 声明产物

### 数据回填

历史 design quest（qst_2606257340、qst_2606255105）outputs 为空，已补入对应核心产物 md（StoragePath 用相对 quest 目录路径），前端可立即查看。

### 评估为不做的

- **HOTL 模式徽标**：从全局最优看是 UI 噪音，FocusView 已按 HOTL 心智深度适配，不需要额外徽标区分
- **首页"空"UX 问题**：HOTL 下 user_review 常空导致信息密度低，非 bug，单独评估
