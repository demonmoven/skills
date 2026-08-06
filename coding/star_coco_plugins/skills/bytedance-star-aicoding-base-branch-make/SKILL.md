---
name: bytedance-star-aicoding-base-branch-make
description: |
  基于 GitLab MR 创建基座分支的自动化工具。支持单条和批量两种模式。
  当用户提到以下场景时触发此 skill：
  - 需要从 MR 创建基座分支 / base branch
  - 需要批量处理 MR 列表生成基座分支
  - 涉及 ai_coding_test/base 分支命名
  - 需要基于 MR 的 merge commit 创建工作分支
  - 提到 create_base_branch 或 batch_create 脚本
  - 需要从 CSV/飞书表格批量读取 MR 并生成分支
  - 需要对特定仓库 cherry-pick 额外 commit
  即使用户没有明确说"基座分支"，只要涉及从 MR 出发创建新分支用于 AI coding 测试的场景，都应使用此 skill。
---

# Bytedance Star AI Coding Base Branch Maker

基于 GitLab MR 自动创建基座分支，支持单条执行和批量处理，支持按仓库配置 cherry-pick commits。

## 核心流程

```
准备环境 → 检测默认分支 → Fetch MR → 计算分叉点 → 找 merge commit → 创建分支 → [cherry-pick] → 推送 → 清理
```

## 使用方式

### 单条模式

```bash
bash <skill-path>/scripts/create_base_branch.sh <REPO> <MR_ID> <VERSION> [CHERRY_PICK_CONFIG]
```

参数：
- **REPO**: 仓库地址（如 `https://code.byted.org/ad/star_settlement`）
- **MR_ID**: MR 编号（如 `844`）
- **VERSION**: 版本号（如 `v1`）
- **CHERRY_PICK_CONFIG**（可选）: cherry-pick 配置文件路径

### 批量模式

```bash
bash <skill-path>/scripts/batch_create.sh <CSV_FILE> <VERSION> [CHERRY_PICK_CONFIG]
```

CSV 需包含 `mr_iid` 和 `repo_web_url` 两列。

## Cherry-Pick 配置文件

通过 JSON 配置文件实现**按仓库独立配置** cherry-pick commits，支持随时插拔。

### 配置格式（cherry_pick_config.json）

```json
{
  "star_settlement": [
    "7ed9c36e7a30b3bb4541101713a37e35480fbb27",
    "4f066c965568f9f6192c8f87cf0a4d7ebfe42ce2"
  ],
  "star_orders": [
    "abcd1234..."
  ],
  "star_account": []
}
```

- **key**: 仓库名（即 URL 的最后一段，如 `star_settlement`）
- **value**: 该仓库需要 cherry-pick 的 commit SHA 列表
- 空数组 `[]` 或不配置该仓库 = 不做 cherry-pick
- 不提供配置文件 = 使用脚本内置的默认配置（star_settlement 的 9 个 commit）

### 新增/删除 commit

直接编辑 JSON 文件即可，无需改动任何脚本代码。

## 技术细节

### 退出码

| 退出码 | 含义 |
|--------|------|
| 0 | 成功 |
| 1 | 参数错误 |
| 2 | MR 引用未找到 |
| 3 | merge commit 未找到 |
| 4 | 推送失败 |
| 5 | cherry-pick 失败 |

### 批量模式输出

在 `batch_result_<timestamp>/` 目录下生成：
- `result.csv`: 每条 MR 的处理结果
- `logs/`: 每条 MR 的独立日志文件
