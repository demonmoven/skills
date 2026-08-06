---
name: channel_charge
display_name: 渠道离线计费
mode: natural
triggers:
  - 三方渠道离线计费
  - 退款计算器
  - 渠道计费
  - 费率规则
  - 结算引擎

# ===== Facade 实现声明(核心) =====
facade_implementations:
  prd-facade:
    implementation: default
    required_outputs: [prd_title, prd_link, prd_requirements, change_type, reference_files]

  repo-route-facade:
    implementation: skip
    reason: 仓库固定为 bytepay/bytepay_charge_offline_engine(repo_id=849505),无需路由
    static_outputs:
      repo_id: 849505
      repo_path: bytepay/bytepay_charge_offline_engine
      repo_meta:
        build_system: scm
        mr_title_prefix: "[fee-rule]"
        base_branch_candidates: [feature/ai_native, master]   # 优先 feature/ai_native,不存在则回落 master
        scm_repo_name: caijing/bytepay/charge_offline_engine
        scm_repo_id: 466820

  tech-solution-facade:
    implementation: skip
    reason: 渠道离线计费需求较简单,直接按 change_type 映射 reference_files 进入编码

  knowledge-facade:
    implementation: default
    inputs:
      keywords: ${prd-facade.prd_requirements}
      requirement_brief: ${prd-facade.prd_requirements}
      scope: charge
    failure_mode: stop_and_ask           # 未授权 / 空命中 / 检索失败即停, 等用户 (llm-wiki-git query)

  coding-facade:
    implementation: default              # 走 charge-atom-code-gen (Coco)
    inputs:
      repo_id: 849505
      repo_path: bytepay/bytepay_charge_offline_engine
      base_branch: ${repo-route-facade.repo_meta.base_branch_candidates[0]}   # 优先 feature/ai_native
      new_branch: "feature/${slug}"
      target_branch: ${coding-facade.base_branch}      # MR 目标与基线一致
      prd_title: ${prd-facade.prd_title}
      prd_link: ${prd-facade.prd_link}
      prd_requirements: ${prd-facade.prd_requirements}
      change_type: ${prd-facade.change_type}
      reference_files: ${change_type_mapping[change_type]}   # 由下文 change_type 映射表决定
      mr_title_prefix: "[fee-rule]"
    required_outputs: [strategy_used, coco_task_id, commit_id, branch, changed_files, compile_pass]
    failure_mode: stop_and_ask

  delivery-facade:
    implementation: default
    inputs:
      repo_id: 849505
      repo_path: bytepay/bytepay_charge_offline_engine
      branch: ${coding-facade.branch}
      commit_id: ${coding-facade.commit_id}
      target_branch: ${coding-facade.base_branch}
      prd_title: ${prd-facade.prd_title}
      prd_link: ${prd-facade.prd_link}
      mr_title_prefix: "[fee-rule]"
      # SCM 打包专属:
      scm_repo_name: caijing/bytepay/charge_offline_engine
      scm_build_type: offline
    required_outputs: [mr_url, mr_iid]
    failure_mode: stop_and_ask
    # 注:SCM 路径下 delivery-facade 内部先 mr create 再 scm build(与 BITS 路径相反)

  qa-facade:
    implementation: skip
    reason: 渠道离线计费当前仅走 mvn test(已内含在 coding-facade 的编译门禁中),无独立 QA 原子

  report-facade:
    implementation: default

# ===== 看板 stage 映射 =====
progress_mapping:
  s1: prd-facade
  s2: repo-route-facade                 # skip,直接 done
  s3: tech-solution-facade              # skip,直接 done
  s4: knowledge-facade
  s5: coding-facade
  s6: delivery-facade.mr                # MR 创建
  s7: delivery-facade.scm_build         # SCM 打包
  s8: report-facade
---

# channel_charge — 渠道离线计费

## 子流程概述

渠道离线计费(bytepay_charge_offline_engine)的后端变更流程。仓库固定、构建系统为 SCM。
覆盖:知识检索 → 编码 → MR 创建 → SCM offline 打包 → 报告。

## 硬约束

- **仓库固定**:`bytepay/bytepay_charge_offline_engine`(repo_id=849505)
- **构建系统**:SCM offline 打包,不走 BITS
- **基线分支**:优先 `feature/ai_native`;该分支在业务仓库不存在则回落 `master`
- **MR 目标分支**:与基线分支一致
- **MR 标题前缀**:`[fee-rule]`
- **不部署到 prod**

## change_type 与 reference_files 映射表

| change_type | 默认 reference_files | 默认策略 |
|---|---|---|
| new_calculator | CalculatorAdapter, Enum, 同类Calculator, Base, Interface, FeeResult, Context | Plan A |
| modify_calculator | 目标Calculator, Adapter, FeeResult | Plan A |
| new_enum_value | 目标Enum, 注册文件 | Plan A |
| config_change | 配置类, Calculator | Plan A |
| new_rpc_interface | — | Plan B |
| cross_module_refactor | — | Plan B |
| unknown | — | Plan B |

## 异常分支

| 节点 | 异常 | 处理 |
|---|---|---|
| coding-facade | AUTH_REQUIRED | 报错,不创建 PAT |
| coding-facade | 编译失败(已降级仍失败) | 中断 |
| delivery-facade | push 被拒 | 报告用户 |
| delivery-facade | MR 冲突 | 报告用户 |
| delivery-facade | SCM 超时 | 报告版本号 |
| delivery-facade | SCM 鉴权错误 | 提示权限 |

## 验证清单

- [ ] MR open
- [ ] mvn compile PASS
- [ ] mvn test PASS
- [ ] changed_files 仅涉及目标模块
- [ ] MR 标题前缀正确
- [ ] 旁路改动已列出
- [ ] SCM 构建成功
- [ ] 报告标注编码策略
