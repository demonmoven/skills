# AI 友好度仓库巡检

对团队所有仓库进行 AI 友好度扫描，更新飞书多维表格并发送通知。

## 配置

| 项目 | 值 |
|------|-----|
| 仓库列表 | `scripts/repos.txt` |
| 扫描脚本 | `scripts/scan_repos.py` |
| 包装脚本 | `scripts/scheduled_scan.sh` |
| Bitable | `LUl0bFMooanjGoszjhkcyhQyn8d` |
| 快照表 | `tbl73jFL8zFz137d` |
| 历史表 | `tblDbwTLLhm0PUnf` |
| CSV 归档 | `scripts/scan_history/` |

## 执行模式

根据用户指令选择模式：

### 模式 1：完整巡检（默认）

用户说"扫描"、"巡检"、"跑一次"时执行：

```bash
cd $PROJECT_DIR && bash scripts/scheduled_scan.sh
```

完整流程：克隆仓库 → 扫描评分 → 清空快照表 → 写入快照 → 追加历史表 → 对比上次 → 发通知

### 模式 2：仅扫描不写表

用户说"只扫描"、"试跑"、"dry run"时执行：

```bash
python3 scripts/scan_repos.py \
  --input scripts/repos.txt \
  --output scripts/scan_history/scan_$(date +%Y-%m-%d).csv \
  --skip-bitable
```

### 模式 3：质量评估

用户说"评估质量"、"评估 CLAUDE.md"时：

1. 先执行模式 2 的扫描（加 `--keep-clones --workdir /tmp/repo-scan-all`）
2. 扫描完成后，启动并行 Agent 对有 CLAUDE.md / 架构文档的仓库进行内容质量评估
3. 评估维度：
   - CLAUDE.md 质量（7 维度）：构建命令、架构概览、代码规范、错误处理、测试约定、生成代码标注、关键依赖
   - 架构文档质量（5 维度）：模块描述、模块依赖、数据流向、设计决策、目录结构
4. 将质量评估结果回填到 CSV 的 `claude_md_quality` 和 `architecture_doc_quality` 列

### 模式 4：更新仓库列表

用户说"更新仓库列表"、"加仓库"时：

编辑 `scripts/repos.txt`，每行一个 git 地址，格式：
```
git@code.byted.org:org/repo.git
```

## 扫描指标说明

### 代码得分（满分 5.0，权重 55%）
- 起始 5.0 分，按以下规则扣减：
- 超 500 行文件占比 >15%：-1.0；>5%：-0.5
- 类型违规数 >20：-1.5；>10：-1.0；>5：-0.5
- 平均文件行数 >300：-0.5

### 测试得分（满分 5.0，权重 45%）
- 起始 0.0 分，按以下规则累加：
- 有测试文件：+2.0
- 测试文件占比 >10%：+1.5；>5%：+0.5
- 有 CI 配置：+1.0
- 有构建脚本：+0.5

### 等级
- A：总分 >= 4.0
- B：总分 >= 3.0
- C：总分 >= 2.0
- D：总分 < 2.0

## 注意事项

- 扫描 71 个仓库大约需要 5-10 分钟（取决于网络和仓库大小）
- 质量评估（模式 3）需要额外时间，建议用 `--keep-clones` 避免重复克隆
- 飞书通知需要在 `scheduled_scan.sh` 中配置 WEBHOOK 变量
- 历史表每次扫描追加记录，不会覆盖，用于趋势追踪
