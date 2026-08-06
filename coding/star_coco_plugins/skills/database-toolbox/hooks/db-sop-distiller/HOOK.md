---
name: db-sop-distiller
description: "数据库排查 SOP 知识蒸馏 — 会话结束时自动分析排查链路，合并更新场景化 SOP 到 memory/db-sop/"
metadata:
  openclaw:
    emoji: "🧠"
    events: ["command:new"]
    requires:
      bins: ["node"]
      config: ["workspace.dir"]
---

# db-sop-distiller

在 `/new` 开启新会话时触发，自动分析**上一次**的 database-toolbox 排查对话：

1. **粗筛**：检测是否使用了 DatabaseToolbox 方法、对话轮次是否 ≥ 4
2. **多场景识别**：一次会话可能涉及多个场景（如慢查询 + 索引优化），均会生成对应 SOP
3. **纠偏检测**：收紧的信号词 + 排除词 + 上下文差异验证，降低误报
4. **冗余检测**：按 函数+实例+库 三元组签名去重，同参数调用 >2 次才标记
5. **SOP 合并**：同场景同 dbType 写入同一文件（如 `slow-query-rds.md`），多次会话的经验累积合并
6. **Promote 提示**：同一纠偏出现 ≥ 3 次时，标注"建议 promote 到 AGENTS.md"

产出文件带 frontmatter（`type: reference`），由 agent 在后续会话中通过 `memory_search` 按需检索。
