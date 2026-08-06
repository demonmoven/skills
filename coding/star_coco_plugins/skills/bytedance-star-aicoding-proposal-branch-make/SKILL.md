---
name: bytedance-star-aicoding-proposal-branch-make
description: |
  基于已存在的 AI Coding 基座分支，自动创建 proposal 分支并执行 OpenSpec 提案流程，最后推送到远端。
  当用户提到以下场景时触发此 skill：
  - 需要从 `ai_coding_test/base/<version>/<mr_id>` 生成 `ai_coding_test/proposal/<version>/<mr_id>`
  - 需要跑 `openspec:proposal` / “发起提案” 并把结果推到 proposal 分支
  - 需要批量读取 CSV，对多条 MR 生成 proposal 分支并执行提案
  - 提到 `ai_coding_pipeline.sh` / `ai_coding_batch.sh` 脚本
---

# Bytedance Star AI Coding Proposal Branch Maker

本 skill 用于在 **base 分支已存在** 的前提下：拉取代码 → 安装 Coco/OpenSpec 环境 → 切换/创建分支 → 执行提案 → 提交并推送 proposal 分支 → 清理工作目录。

## 前置条件

- 远端必须已存在基座分支：`ai_coding_test/base/<version>/<mr_id>`
  - 该分支通常由 `bytedance-star-aicoding-base-branch-make` 生成。

## 使用方式

### 单条模式

```bash
bash <skill-path>/scripts/ai_coding_pipeline.sh \
  --repo-url  <代码仓库URL> \
  --version   <版本号> \
  --mr-id     <MR_ID> \
  --spec-url  <需求规格文档URL>
```

参数说明：
- `--repo-url`: 仓库的 Git clone 地址或 Web URL（脚本内部用 `git clone`）
- `--version`: 版本号（用于拼接分支名）
- `--mr-id`: MR 编号（用于拼接分支名）
- `--spec-url`: 提案输入的 Spec 文档链接

脚本生成的分支：
- `BASE_BRANCH`: `ai_coding_test/base/<version>/<mr_id>`（要求已存在）
- `PROPOSAL_BRANCH`: `ai_coding_test/proposal/<version>/<mr_id>`（要求远端不存在）

### 批量模式

```bash
bash <skill-path>/scripts/ai_coding_batch.sh \
  --version <版本号> \
  --csv     <CSV文件路径>
```

CSV 格式（首行为表头，至少包含以下三列）：

```csv
mr_iid,repo_web_url,spec_url
592,https://code.byted.org/ad/star_settlement,https://bytedance.larkoffice.com/docx/xxxx
```

说明：
- `spec_url` 可能包含逗号，脚本会把第 3 列到行尾作为 `spec_url`。
- 批量模式默认单行失败不中断整体流程，最后按失败数决定退出码。

## 注意事项

- `scripts/ai_coding_pipeline.sh` 会在本机安装/配置 `nvm`、`node`、`openspec`、`coco` 以及 `ad/star_coco_plugin`（若已存在会跳过）。
- 推送 proposal 分支使用 `--force-with-lease`，适用于首次创建/幂等重试，但仍建议确认远端分支不存在。
- 工作目录默认在当前目录下的 `ai_coding_workspace/`，流程结束会删除该目录。

