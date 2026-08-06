# charge-atom-repo-router — 仓库与模块路由原子 SKILL

## 定位

读取 `settle_skill/config/repo_routing.yaml`,根据 PRD 解析产出的 `prd_title` + `prd_keywords` + 全文,输出**目标仓库列表 + 元信息 + 模块提示**。所有"PRD 涉及哪个 PSM、推荐用哪个子流程、改哪些参考文件、用什么策略"的判定,统一从这里出。

**不做**:
- 不解析 PRD(上游 `prd-analyzer` 已完成)
- 不调用其它原子 / 子流程
- 不写代码 / 不建 MR
- 不维护任何业务关键词(全部在 yaml 里)

## 输入参数（由 charge-dev-router 或子流程传入)

```yaml
prd_title: ${PRD标题}
prd_text: ${PRD全文,可选,提高 L2 模块词命中率}
prd_keywords: ${PRD解析产出的关键词数组,例如 ["渠道离线计费","退款计算器"]}
force_repos: []                          # 可选,人工强制指定仓库列表(跳过路由)
config_path: ${sandbox路径}/settle_skill/config/repo_routing.yaml  # 由调用方提供
```

## 输出（返回给调用方）

```yaml
matched_repos:                           # 数组,长度 0/1/N
  - repo_key: bytepay_charge_offline_engine
    repo_meta:                           # 直接来自 yaml 的 repos.<key>
      repo_id: 849505
      repo_path: bytepay/bytepay_charge_offline_engine
      default_base_branch: feature/ai_native
      fallback_base_branch: master
      build_system: scm
      scm_repo_name: caijing/bytepay/charge_offline_engine
      scm_build_type: offline
      mr_title_prefix: "[fee-rule]"
    matched_l1_keywords: [离线引擎]
    recommended_subflow: channel_charge
    suggested_change_type: new_calculator
    suggested_reference_files: [...]
    suggested_strategy: mira_patch
    matched_l2_keywords: [退款计算器]
  - repo_key: bytepay_charge
    repo_meta: { ... }
    matched_l1_keywords: [商户计收费]
    recommended_subflow: general_dev
    suggested_change_type: unknown
    suggested_reference_files: []
    suggested_strategy: coco_full
    matched_l2_keywords: []

execution_mode: parallel | single | none
needs_user_confirmation: false
confirmation_prompt: ""                  # 仅 needs_user_confirmation=true 时填
config_version: 2
config_loaded_from: <绝对路径>
```

## 执行流程

### Step 1. 加载配置

读取 `config_path` 指向的 yaml。校验 `version >= 2`,若文件不存在或解析失败 → 直接返回错误,**不要降级到硬编码兜底**(避免业务知识漂移)。

### Step 2. 处理 `force_repos`

若调用方传入 `force_repos` 非空:
- 跳过 L1 关键词匹配,直接以 `force_repos` 为 `matched_repos`
- 仍执行 L2 模块词匹配以补充 `suggested_*` 字段
- `execution_mode` 按数量决定:1 → single;≥2 → parallel
- `needs_user_confirmation = false`(用户已显式指定)

### Step 3. L1 仓库决定词扫描

构造扫描语料 = `prd_title + " " + prd_text + " " + join(prd_keywords)`,小写化。

遍历 `repo_determining_keywords` 中每个 `repo_key` 下的关键词:
- 任一关键词作为子串出现在语料中 → 该 `repo_key` 命中
- 记录命中的具体关键词到 `matched_l1_keywords`

收集所有命中的 `repo_key` → `matched_repos`(去重)。

### Step 4. 应用兜底策略

参照 yaml `fallback` 节:

**case A: `matched_repos` 为空**
- `on_no_l1_match=ask_user`(默认):
  - `execution_mode: none`
  - `needs_user_confirmation: true`
  - `confirmation_prompt`: "PRD 未匹配到任何已知仓库。请确认要改的仓库,或将新关键词加入 settle_skill/config/repo_routing.yaml 后重试。"
- `on_no_l1_match=use_default`:
  - 取 `fallback.default_repo` 作为 `matched_repos[0]`
  - 标注 `matched_l1_keywords: ["<fallback>"]`

**case B: `matched_repos` 长度 = 1**
- `execution_mode: single`
- `needs_user_confirmation: false`

**case C: `matched_repos` 长度 ≥ 2**
- `on_multi_l1_match=parallel_execute`(默认): `execution_mode: parallel`
- `on_multi_l1_match=sequential_execute`: `execution_mode: sequential`
- `on_multi_l1_match=ask_user`: `execution_mode: none`,`needs_user_confirmation: true`,`confirmation_prompt` 列出所有命中仓库让用户选

### Step 5. L2 模块/功能词扫描(为每个命中仓库补充提示)

对 `matched_repos` 中**每个**仓库:

1. 遍历 `module_hints` 数组,找出**所有满足以下两个条件的条目**:
   - 当前 `repo_key` 在该条目的 `applies_to_repos` 列表中(或 `applies_to_repos` 含 `"*"`)
   - 该条目 `keywords` 中至少一个出现在扫描语料中

2. 合并命中条目的提示:
   - `suggested_reference_files`:**合并去重**(保持首次出现顺序)
   - `suggested_change_type`:
     - 0 条命中 → 取 `fallback.default_change_type`
    "i�Y�K��(i"y�Nh�^�x~yJ�� ≥2 条命中且 change_type **相同** → 直接采用
     .(�S"i�Y�K��K�B6��vU�G�R��K��Y¢�(i"��>X{�K��X�~���G�S�G�S"������yKK��k��{�nzX��Z�Xk>Z�h�n��.�z��7VvvW7FVE�7G&FVw��� 任一条目为 `coco_full` → 取 `coco_full`(从严)
    "Y
nX��X�n�ini�Y�K��y�B7G&FVw��0 条命中 → 取 `fallback.default_strategy`
   - `matched_l2_keywords`:合并去重所有命中关键词

### Step 6. 装配输出

按上述结构返回。所有 `repo_meta` 字段必须**原样从 yaml 透传**,不要在原子内做任何字段名映射或加工(避免上游期望与配置漂移)。

---

## 关键设计原则

| 原则 | 含义 |
|---|---|
| **配置驱动** | 不在原子代码 / SKILL.md 内硬编码任何关键词 / 仓库 ID / 路径 |
| **扇出, 不收敛** | 多仓库命中时输出多条,**不做仓库择优**;并发执行交给上游 |
| **L1/L2 解耦** | L1 决定"去哪",L2 决定"在哪改";互不污染 |
| **L2 跨仓库共用** | 同一关键词可适用多个仓库,通过 `applies_to_repos` 列表表达 |
| **失败显式** | 配置加载失败 / 完全无命中 → 显式抛错或停下问人,**严禁悄悄兜底** |

## 异常处理

| 异常 | 处理 |
|---|---|
| `config_path` 不存在 | 返回错误"配置文件未找到, 请确认 settle_skill 已同步" |
| yaml 解析失败 | 返回错误,附 yaml 错误位置,**不降级** |
| `version < 2` | 返回错误"配置版本不兼容,期望 ≥ 2" |
| `force_repos` 中存在 yaml `repos` 未定义的 key | 返回错误"未知仓库 key: xxx" |
| `module_hints[*].applies_to_repos` 引用了未定义仓库 | 跳过该条 hint, 记入 warnings(不阻塞) |

## 调用示例

```yaml
# 输入
prd_title: "渠道离线计费 + 商户计收费 联合改造:新增退款计算器"
prd_keywords: [渠道离线计费, 退款计算器, 商户计收费, 月付贴息]

# 输出
matched_repos:
  - repo_key: bytepay_charge_offline_engine
    matched_l1_keywords: [渠道离线, 离线计费]
    matched_l2_keywords: [退款计算器]
    suggested_change_type: new_calculator
    suggested_strategy: mira_patch
    suggested_reference_files: [<7 个 java 文件>]
    repo_meta: { build_system: scm, ... }
    recommended_subflow: channel_charge
  - repo_key: bytepay_charge
    matched_l1_keywords: [商户计收费, 月付贴息]
    matched_l2_keywords: []
    suggested_change_type: unknown
    suggested_strategy: coco_full
    suggested_reference_files: []
    repo_meta: { build_system: bits, bits_devops_space_id: 749690368002, ... }
    recommended_subflow: general_dev
execution_mode: parallel
needs_user_confirmation: false
```

下游 `charge-dev-router` 收到此输出后,**并发派发**两个独立 Coco task,各自跑对应子流程,最后合并报告。
