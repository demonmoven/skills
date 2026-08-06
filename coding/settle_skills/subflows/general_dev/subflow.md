---
name: general_dev
display_name: 通用研发子流程
mode: natural
triggers:
  - 通用研发
  - 兜底子流程
  - BITS 流水线
  - 自动化测试循环

# ===== Facade 实现声明(核心) =====
facade_implementations:
  prd-facade:
    implementation: default
    required_outputs: [prd_title, prd_link, prd_requirements, change_type, reference_files, uncertain_points]
    on_missing:
      uncertain_points: stop_and_ask(prompt="PRD 存在不确定点,请确认后继续")

  repo-route-facade:
    implementation: default
    required_outputs: [repo_id, repo_path, repo_meta, hit_keywords, hit_count]
    failure_mode: stop_and_ask          # 0 命中或 ≥2 命中 → 停

  tech-solution-facade:
    implementation: default              # 走 manifest 中 routing_rules(simple→内置模板,complex→prd-tech-design 编排 Agent)
    inputs:
      prd_requirements: ${prd-facade.prd_requirements}
      change_type: ${prd-facade.change_type}
      reference_files: ${prd-facade.reference_files}
      repo_meta: ${repo-route-facade.repo_meta}
      requirement_brief: ${prd-facade.prd_requirements}
      knowledge_context: ${knowledge-facade.outputs}   # 硬依赖: 取 knowledge_md/sources; 由 knowledge-facade stop_and_ask 保证 status=ok
    required_outputs: [tech_solution_md, complexity, complex_reasons, decisions, open_questions]
    failure_mode: fallback_to_simple     # 原子失败 → 走 simple 模板,warn

  knowledge-facade:
    implementation: default
    inputs:
      keywords: ${prd-facade.prd_requirements}
      requirement_brief: ${prd-facade.prd_requirements}
      scope: charge
    failure_mode: stop_and_ask           # 检索失败 / 空命中即停, 等用户 (llm-wiki-git query)

  coding-facade:
    implementation: default              # 走 manifest 中 coding-facade.default_atom (charge-atom-code-gen / Coco)
    inputs:
      repo_id: ${repo-route-facade.repo_id}
      repo_path: ${repo-route-facade.repo_path}
      base_branch: master
      new_branch: "feature/auto_dev-${slug}"
      target_branch: master
      prd_title: ${prd-facade.prd_title}
      prd_link: ${prd-facade.prd_link}
      prd_requirements: ${prd-facade.prd_requirements}
      change_type: ${prd-facade.change_type}
      reference_files: ${prd-facade.reference_files}
      tech_solution_md: ${tech-solution-facade.tech_solution_md}
      mr_title_prefix: ${repo-route-facade.repo_meta.mr_title_prefix}
      # 循环修复时由框架自动注入:
      # incremental_fix: true
      # existing_coco_task_id: ${prev.coco_task_id}
      # existing_branch: ${prev.branch}
      # fix_instructions: ${delivery-facade.fix_instructions}
    required_outputs: [strategy_used, coco_task_id, commit_id, branch, changed_files, bypass_changes, compile_pass]
    failure_mode: stop_and_ask

  delivery-facade:
    implementation: default              # 走 manifest 中 delivery-facade routing_rules(bits/scm)
    inputs:
      repo_id: ${repo-route-facade.repo_id}
      repo_path: ${repo-route-facade.repo_path}
      branch: ${coding-facade.branch}
      commit_id: ${coding-facade.commit_id}
      target_branch: master
      prd_title: ${prd-facade.prd_title}
      prd_link: ${prd-facade.prd_link}
      mr_title_prefix: ${repo-route-facade.repo_meta.mr_title_prefix}
      bits_service: ${repo-route-facade.repo_meta.bits_service}
      # 循环修复时由框架自动注入:
      # existing_dev_id: ${prev.dev_id}
      # existing_mr_iid: ${prev.mr_iid}
    required_outputs: [dev_id, pipeline_status, mr_url, mr_iid]
    failure_mode: loop_back_to_coding    # pipeline failed → 回 coding-facade 增量修复
    auto_repair:
      enabled: true
      max_rounds: 3
      on_exhaust: stop_and_ask(prompt="自动修复 3 轮仍失败,请人工介入")

  qa-facade:
    implementation: skip                 # 当前占位,待接入自动化测试
    reason: 自动化测试待接入,本轮跳过;由人工确认是否可建 MR
    post_hook:
      type: ask_user
      blocking: true
      timeout_minutes: 30
      prompt_template: |
        流水线已通过,请确认本轮代码是否可用并完成 MR?
        - 回复「确认 / ok / 可以」→ 我将完成 MR 节点收尾
        - 回复「不行 / 还有问题」→ 请附上修复说明,我将触发增量修复
      accept_keywords: [确认, ok, 可以, approve, 没问题]
      reject_keywords: [不行, 还有问题, 不可以, reject]
      on_reject: loop_back_to_coding
      on_timeout: warn_and_proceed

  report-facade:
    implementation: default

# ===== 看板 stage 映射 =====
progress_mapping:
  s1: prd-facade
  s2: repo-route-facade
  s3: tech-solution-facade
  s4: knowledge-facade
  s5: coding-facade
  s6: delivery-facade                   # 流水线
  s7: delivery-facade.mr                # MR 创建
  s8: report-facade
---

# general_dev — 通用研发子流程

## 子流程概述

不绑定特定仓库的通用研发流程。覆盖 **PRD 理解 → 仓库路由 → 技术方案 → 知识检索 → 编码 → 流水线 + MR → 报告** 全链路。仓库由 `repo-route-facade` 决定,构建系统由 `delivery-facade` 自动路由(BITS/SCM)。

## 硬约束

- **不绑定仓库**:仓库由 `repo-route-facade` 决定
- **不部署到 prod**:本子流程只覆盖到「流水线 success + MR 创建」,不触发上线
- **测试通过后才创建 MR**:`delivery-facade` 在 build_system=bits 路径下默认先跑流水线再建 MR
- **同一 MR + 同一 Coco task-id + 同一 dev_id**:循环修复时,coding-facade 复用 task-id 续聊;delivery-facade 复用同 dev_id,只产生新 pipeline_run_seq
- **人在环路**:任一 facade 输出 `open_questions` / `uncertain_points` / `bypass_changes` 非空 → 立即停止等待
- **循环至人满意为止**:无最大循环次数硬性上限(auto_repair 仅限流水线失败的自动回环),每轮必产出 report-facade 报告等用户判断

## 循环修复机制

| 维度 | 约束 | 由哪个 facade 字段控制 |
|---|---|---|
| Coco task-id | 复用同一 task-id 续聊 | `coding-facade.existing_coco_task_id` |
| 分支 | 同一分支,新增 commit | `coding-facade.existing_branch` |
| MR | 不重新建;若已建则在评论补述 | `delivery-facade.existing_mr_iid` |
| dev_id | 复用同一 dev_id,产生新 pipeline_run_seq | `delivery-facade.existing_dev_id` |
| 修复指令 | 上轮失败日志转化为 fix_instructions | `coding-facade.fix_instructions` |

## 异常分支

| 节点 | 异常 | 处理 |
|---|---|---|
| prd-facade | uncertain_points 非空 | 停止等用户 |
| repo-route-facade | 0 命中 / ≥2 命中 | 停止等用户 |
| tech-solution-facade | 原子失败 | fallback 到 simple 模板,warn |
| knowledge-facade | 未授权 / 空命中 / 检索失败 | 停止等用户(stop_and_ask) |
| coding-facade | AUTH_REQUIRED / 持续失败 | 停止 |
| delivery-facade | pipeline failed | 自动回环修复(≤3 轮) |
| delivery-facade | push 被拒绝 / MR 冲突 | 停止,报告用户 |
| qa-facade | 测试不通过 | 回环 coding-facade |
| report-facade | 失败 | mira 上传 .md fallback |

## 验证清单

- [ ] prd-facade 解析无悬挂 uncertain_points
- [ ] repo-route-facade 单仓命中
- [ ] tech-solution-facade 已产出 tech_solution_md
- [ ] knowledge-facade 已尝试
- [ ] coding-facade compile + push PASS
- [ ] delivery-facade 流水线 success(BITS)或本地打包 PASS(SCM)
- [ ] 人工确认代码可用
- [ ] MR 已创建且 body 含完整链路 + 看板 URL
- [ ] report-facade 已生成飞书文档且置顶看板 URL
- [ ] 看板 8 个 stage 均已 patch 到终态
