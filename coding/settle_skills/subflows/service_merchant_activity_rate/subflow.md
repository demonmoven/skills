---
name: service_merchant_activity_rate
display_name: 服务商活动费率配置
mode: natural
triggers:
  - 服务商活动费率配置
  - 服务商活动费率
  - 服务商费率活动配置

# ===== Facade 实现声明(核心) =====
# auto-develop 按通用 8-facade 顺序编排,每个 facade 的行为由本声明决定。
# 可选值:
#   implementation: default          → 走 manifest.yaml 中 facade 的 default_atom
#   implementation: skip             → 跳过该 facade,直接标 done/skip
#   implementation: replace_with_atom → 替换为指定原子
#   post_hook: ask_user(...)         → facade 完成后强制人在环路

facade_implementations:
  prd-facade:
    implementation: default
    required_outputs: [prd_title, prd_link, prd_requirements]
    on_missing:
      meego_url: stop_and_ask(prompt="本类需求需要 prd URL 用于sql生成,请提供")

  repo-route-facade:
    implementation: default
    failure_mode: warn_continue      # 0 命中不阻塞(本类需求不依赖业务仓库)

  tech-solution-facade:
    implementation: replace_with_atom
    atom: online-service-activity-sql
    inputs:
      prd_title: ${prd-facade.prd_title}
      prd_link: ${prd-facade.prd_link}
      prd_requirements: ${prd-facade.prd_requirements}
      out_dir: ${sandbox}/service_merchant_activity_rate/${session_id}/sql/
    required_outputs: [sql_file_path, sql_size_bytes, sql_excerpt]
    failure_mode: stop

  knowledge-facade:
    implementation: skip
    reason: 本类需求知识库已沉淀在 SQL 模板中,无需调原子

  coding-facade:
    implementation: replace_with_atom
    atom: boe-sql-ticket
    inputs:
      sql_file_path: ${tech-solution-facade.sql_file_path}
      db: caijing_bytepay_charge        # 硬锁定,不允许 override
      ticket_title: "[服务商活动费率配置] ${prd-facade.prd_title}"
      ticket_desc: "${prd-facade.prd_link} + ${prd-facade.prd_requirements}"
      approver_mode: self                # 申请人自审批
    required_outputs: [ticket_id, ticket_url, approver, approved_at]
    failure_mode: stop
    post_hook:
      type: ask_user
      blocking: true                     # 永久阻塞,不设超时
      prompt_template: |
        【服务商活动费率配置】BOE SQL 工单已审批
        标题:${prd-facade.prd_title}
        工单:${coding-facade.ticket_url}
        库:caijing_bytepay_charge
        SQL 摘要:${tech-solution-facade.sql_excerpt}

        请确认是否已在 BOE 环境执行该 SQL:
        - 已执行 → 回复「已执行 / 已生效 / ok」,我会继续触发 prd-testcase-workflow-trigger 测试用例生成
        - 未执行 → 回复「未执行」,我会停下来等你执行完成后再回复继续
      accept_keywords: [已执行, 已生效, ok, continue, 执行了, 跑过了]
      reject_keywords: [未执行, 还没, 没跑, 别催, 还没执行]
      on_reject: stop_and_wait(prompt="已记录,请在 BOE 执行 SQL 后回复『已执行』触发后续 prd-testcase-workflow-trigger")

  delivery-facade:
    implementation: skip
    reason: 本类需求无业务代码变更,不触发 BITS/SCM 流水线,不创建 MR

  qa-facade:
    implementation: replace_with_atom
    atom: prd-testcase-workflow-trigger, test-case-excute-agent
    inputs:
      prd_url: ${prd-facade.prd_url}
      prd_title: ${prd-facade.prd_title}
      extra_context: "SQL 工单=${coding-facade.ticket_url}, SQL摘要=${tech-solution-facade.sql_excerpt}"
    required_outputs: 
    failure_mode: warn_continue          # 执行失败不回滚已生效 SQL
    test_report: true, 产出原子测试报告、整体测试报告                  # 生成测试报告

  report-facade:
    implementation: default

# ===== 看板 stage 映射 =====
progress_mapping:
  s1: prd-facade
  s2: repo-route-facade
  s3: tech-solution-facade
  s4: knowledge-facade
  s5: coding-facade
  s6: coding-facade.post_hook           # 人在环路确认
  s7: qa-facade
  s8: report-facade
---

# service_merchant_activity_rate — 服务商活动费率配置

## 子流程概述

在 BOE 环境执行配置 SQL 完成服务商活动费率配置,并生成测试用例自动检测配置正确性。
不涉及业务代码变更,无 BITS/SCM 流水线,无 MR。

## 硬约束

- **不操作生产环境**:env ∈ {boe}
- **数据库锁死**:`data-rds-boe` 工单的目标数据库 **强制** = `caijing_bytepay_charge`,不允许其它库
- **审批人锁死**:工单提交后**使用申请人(本会话当前用户)的权限自审批**,不引入其他审批人
- **执行确认永久阻塞**:coding-facade 的 `post_hook` 为永久阻塞的人在环路;后续 test-case-excute-agent 强依赖 SQL 已执行的真实数据,无回复永远阻塞,不退化跳过

## 异常分支

| 节点 | 异常                                    | 处理 |
|---|---------------------------------------|---|
| prd-facade | prd_url 缺失                            | 停止询问用户补 URL |
| tech-solution-facade | SQL 渲染失败 / 产物为空                       | 停止,抛原子真实 stderr |
| coding-facade | data-rds-boe 鉴权失败                     | 停止,提示用户在主机登录 data-rds-boe |
| coding-facade | 工单 DB 字段被误传非 `caijing_bytepay_charge` | 原子内强校验,直接报错 |
| coding-facade.post_hook | 用户回复「未执行」                             | 停止等待 |
| qa-facade | prd-testcase-workflow-trigger或test-case-excute-agent 失败     | warn,报告标注 |

## 验证清单

- [ ] prd-facade 产出含非空 `prd_url`
- [ ] tech-solution-facade 已产 SQL 文件,大小 > 0
- [ ] coding-facade 工单状态 = APPROVED
- [ ] 已 emit「BOE 是否已执行」独立回复消息(post_hook)
- [ ] 用户已明确回复「已执行」或本流程在「未执行」分支停止
- [ ] qa-facade 已触发(已执行分支)
- [ ] report-facade 已生成飞书报告且置顶看板 URL
