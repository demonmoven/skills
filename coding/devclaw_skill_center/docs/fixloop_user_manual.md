# Fixloop 用户手册

## 一句话介绍

Fixloop 是一个**自动化测试修复循环工具**——你给它代码仓库和需求文档，它帮你生成测试、部署服务、跑测试、分析失败原因、修复代码，循环往复直到测试全部通过。

---

## 它能帮你做什么

| 能力 | 说明 |
|------|------|
| 自动生成测试 | 根据需求文档自动生成 E2E 集成测试和单元测试 |
| 自动部署 | 将代码部署到 TCE BOE 泳道环境 |
| 自动跑测试 | 执行测试并收集结果 |
| 自动分析失败 | 分析每个失败用例的根因，区分是业务代码 bug 还是测试写错了 |
| 自动修复代码 | 根据分析结果自动修复代码并提交 |
| 循环迭代 | 部署→测试→分析→修复，反复循环直到全部通过（或达到最大轮次） |

---

## 工作流程

```
你提供: 需求文档 + 业务仓库 + 测试仓库
            │
            ▼
    ┌── 生成测试（可跳过）
    │   根据需求文档自动生成 E2E 测试和单元测试
    │
    ▼
┌─────────── 迭代循环 ───────────┐
│                                │
│  1. 部署 → 代码部署到泳道      │
│       ↓                        │
│  2. 跑测试 → 收集测试结果      │
│       ↓                        │
│  全部通过？→ 是 → 结束 🎉      │
│       ↓ 否                     │
│  3. 分析失败 → 定位根因        │
│       ↓                        │
│  4. 修复代码 → 提交推送        │
│       ↓                        │
│  回到第 1 步（最多 10 轮）     │
└────────────────────────────────┘
            │
            ▼
    输出：最终测试通过率 + 各轮修复记录
```

---

## 参数说明

### 必填参数

你必须提供以下 5 个参数。如果漏了，fixloop 会交互式问你补充。

| 参数 | 说明 | 示例 |
|------|------|------|
| `PSM` | 你的 TCE 服务标识 | `stone.cozeloop.prompt` |
| `BRANCH` | 业务仓库的分支名 | `feat/my-feature` |
| `BUSINESS_REPO` | 业务代码仓库路径 | `/home/user/backend` |
| `TEST_REPO` | 测试代码仓库路径 | `/home/user/api_test` |
| `SPEC_DIR` | 需求文档所在目录（API 文档、PRD 等） | `/home/user/specs` |

> 仓库路径支持本地绝对路径（推荐）或 Git 地址（`git@...`），Git 地址会自动 clone。

### 可选参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `MAX_ITERATIONS` | `10` | 最大迭代轮次 |
| `GENERATE_TESTS` | `true` | 是否自动生成测试，已有测试时设 `false` |
| `SINGLE_TEST_RUN` | `false` | 只跑一次测试看结果，不修复 |
| `TCE_LANE` | 自动创建 | 复用已有的泳道名 |
| `SKIP_DEPLOY` | `false` | 跳过部署（需配合 `TCE_LANE` 使用） |

---

## 使用示例

### 最常用：全流程自动化

```
/fixloop PSM=stone.cozeloop.prompt BRANCH=feat/my-feature BUSINESS_REPO=/home/user/backend TEST_REPO=/home/user/api_test SPEC_DIR=/home/user/specs
```

从生成测试到修复代码，全自动完成。

### 已有测试，直接跑修复循环

```
/fixloop PSM=stone.cozeloop.prompt BRANCH=main BUSINESS_REPO=/home/user/backend TEST_REPO=/home/user/api_test SPEC_DIR=/home/user/specs GENERATE_TESTS=false
```

### 只看测试结果，不自动修复

```
/fixloop ... SINGLE_TEST_RUN=true GENERATE_TESTS=false
```

### 复用已有泳道，跳过部署

```
/fixloop ... GENERATE_TESTS=false TCE_LANE=boe_costudio_a1b2 SKIP_DEPLOY=true
```

### 只生成测试，不执行

```
/fixloop ... MAX_ITERATIONS=0
```

### 快速试跑 3 轮

```
/fixloop ... MAX_ITERATIONS=3
```

---

## 怎么选参数？

```
你有现成的测试代码吗？
├── 有 → GENERATE_TESTS=false
│   ├── 服务已在泳道中？ → SKIP_DEPLOY=true TCE_LANE=<泳道名>
│   ├── 只想看结果？     → SINGLE_TEST_RUN=true
│   └── 想限制轮数？     → MAX_ITERATIONS=3
│
└── 没有 → 用默认参数即可（自动生成测试 + 跑修复循环）
    └── 只想生成测试？   → MAX_ITERATIONS=0
```

---

## 常见问题

**Q: 运行前需要准备什么？**
确保 bytedcli 已认证（fixloop 通过 npx 自动调用，不需要预装），仓库路径可访问，SPEC 目录下有需求文档。

**Q: 中途失败了怎么办？**
利用参数跳过已完成的阶段重新运行，比如 `GENERATE_TESTS=false TCE_LANE=<上次的泳道> SKIP_DEPLOY=true`。

**Q: 修了好多轮还是不通过？**
设 `MAX_ITERATIONS=3` 先跑几轮观察趋势，查看输出目录下的分析报告，标为"不确定"的问题通常需要人工介入。

**Q: 输出在哪里看？**
默认在 `.costudio/` 目录下，每轮迭代有独立文件夹（`iteration_1/`、`iteration_2/`...），包含测试结果、失败分析、修复记录等。
