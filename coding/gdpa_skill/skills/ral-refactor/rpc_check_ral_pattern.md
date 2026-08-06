## Goals:

Analyze RPC calls using `WithIDC` options in Go project, classify routing patterns and summary core information.
Note: 用户提供的代码会多于分析的需求，只需分析用户要求的目标 WithIDC 点位，所有结论也都针对此点位

## Output Format

分析结果必须输出到 `pattern_analysis.md` 文档，格式如下：

```markdown
# Pattern Analysis Report

## Summary
- Total WithIDC points: N
- Categories breakdown:
  - Deployment-induced: X
  - Storage-XXX: Y
  - RoutingByEnt-XXX: Z
  - Other-XXX: W

## Points

### Point 1: {file_path}:{line_number}
- **Category**: Deployment-induced / Storage-XXX / RoutingByEnt-XXX / Other-XXX
- **va_involved**: true/false
- **ap_involved**: true/false
- **routingCodeMap**:
  - WithIDC 位置: `{file}:{line}` - `WithIDC({value})`
  - RPC 调用位置: `{file}:{line}` - `{method_call}`
  - IDC 来源: {description of how IDC value is determined}
  - 路由策略: {description of routing logic}

### Point 2: {file_path}:{line_number}
...
```

每个点位必须包含完整的分析信息，以便后续 refactor 阶段能够独立处理每个点位。

## 规则

1. **PSM 定义**: 服务唯一标识，格式为 `{PRODUCT}.{SUBSYSTEM}.{MODULE}` (如 `tiktok.devflow.api`)。**⚠️ 禁止从 package path 猜测 PSM，必须通过工具获取**: [get_ral_info_by_package_path.md](./tools/get_ral_info_by_package_path.md)
2. **代码分析**: 按要求严格分析/跟踪目标内容，忽略无关代码

## Context

{{.arch_3v}}
{{.normal_rpc}}
### IDC Routing Definition:
    - `{toIDC} = routingFunction({Input})`
    - `{Input}`: Determinant of target IDC (e.g., request params, configs, entity fields, constants)
    - `{toIDC}`: Target environment identifier (IDC/VRegion/VGeo)

## Classification Criteria

### Category Identification Keys

> A `Exclusive` label will be added when a category is exclusive.
> Non-Exclusive categories can be selected in multiple ways.

| Category                       | Identification Criteria.                                                                                                       | 
|--------------------------------|--------------------------------------------------------------------------------------------------------------------------------| 
| **NoWithIDC**                  | NO `WithIDC()` option in the provided code. [Exclusive]                                                                        | 
| **Deployment-induced Routing** | {Input} ONLY depends on `deployStatus` (see definition below), and only a single {toIDC} is involved per request.  [Exclusive] |  
| **DisasterRecover**            | There must some vars/functions that contain `failover`/`disaster_recovery`/`DR` symbols                                        |
| **Storage-XXX**                | Storage related routing.                                                                                                       |
| └─ Storage-TOS                 | The {input} contains constants/vars/functions with names of `tos`, `bucket` etc.                                               |
| **RoutingByEnt-XXX**           | The IDC is deduced from entities, like user, video, music, etc., it's like entity.Field->(xxx, usually a remote call)->idc.    |
| ├─ RoutingByEnt-StoreRegion    | The Entity is user and the routing is to find user's storeRegion, ttregion(toutiao_ttregion_manager.XXX) is used to get idc    |
| └─ RoutingByEnt-{XXX}          | Other entities, such as `video`, `music`, `anchor`, etc.                                                                       |
| **Other-XXX**                  | Others...                                                                                                                      |
| ├─ Other-AbTesting             | {Input} must contains vars/functions with names similar to `abtest`/`exp_id`                                                   |
| ├─ Other-ReqIDC                | IDC directly comes from request, such as `req.IDC`/`ctx.Value("idc")`/`extractStrFrom(req.Field)`                              |
| ├─ Other-ReqRegion             | {Input} directly comes from request, such as `req.Region`/`req.VRegion`                                                        |
| ├─ Other-ReqField              | {Input} inferred from request, such as `req.XID -> VDC`                                                                        |
| ├─ Other-Broadcast             | Same RPC method always iterates over >1 {toIDC}, (e.g., range over IDCs, call(IDC_A) then call(IDC_B))                         |
| ├─ Other-Fallback              | Same RPC method may iterates over >1 {toIDC}, (e.g., call(IDC_A) if no_result call(IDC_B))                                     |                                        
| └─ Other-Unknown               | Pattern doesn't match any criteria or your confidence is less than 50%                                                         |

### Key Definitions

**deployStatus Indicators**:
Only the following information is used to decide the {toIDC}. Explicitly exclude any dependence on request parameters (
e.g. req.appID, userID) or dynamic call context.

1. get a hardcode {toIDC} without any conditions  
   • Fixed value from code constant (same in all regions/envs)

2. depend on current deploy env  
   • Only infrastructure-level functions:  
   › `{region_lib|env}.Is{Env_tag}()`  
   › `{region_lib|env}.{Env_tag}()`  
   › `region_lib.GetCurrentVRegion()`  
   • Pure env detection without mixing other factors
   • e.g. `if env.IsVGeoUS() { return "us" }`  
   • Exclude: any `req.*` or `ctx.*` references

3. config from static sources (immutable at runtime)  
   • File: `conf/conf.{Env_tag}.yml`  
   • Env var: `os.Getenv("IDC")`  
   • Pre-loaded config: `tcc.Get().ToIDC`  
   • Must be read-once during initialization
   • Exclude: config with per-request overrides

## Core Information

1. `va_involved`: WithIDC 中的 toIDC 涉及 US-East (VRegion) 区域 (别名：va，包含的 VDC：maliva)
2. `ap_involved`: WithIDC 中的 toIDC 涉及 Singapore-Central (VRegion) 区域 (别名：aliag，包含的 VDC：my, my2, my3, sg1)
3. `routingCodeMap`: 收集和目标 WithIDC 点位相关的代码信息，包括：WithIDC Option 定义和被使用的位置、使用 WithIDC 的 RPC 位置、IDC 来源、路由选择策略等

{{- /* Conditional block for the safety check summary fields */}}
{{- if .enabel_with_vregion }}
4.  `checkCategory` (string): 对安全检查结果的最终分类。**必须**是以下值之一：
    -   `Unresolvable Dynamic IDC`  
    -   `Intra-VRegion Multi-IDC Call` 
    -   `Non-3V IDC` 
    -   `Normal` 
5.  `checkExplain` (string): 对 `checkCategory` 的详细文字解释，说明判断依据。
    {{- end }}

{{- /* Conditional rules section that defines how to perform the checks */}}
{{- if .enabel_with_vregion }}

### Safety Check Execution Plan (Active if `enabel_with_vregion` is "1")

#### Step 1: ANALYZE - Trace & Resolve IDC Value Set
For every target `WithIDC(x)` call, you must determine the **Candidate Value Set** of `x`.
-   **Literal**: If `x` is a string literal (e.g., "sg1"), the set is `["sg1"]`.
-   **Resolvable Variable (TCC/Constants)**:
    -   If `x` is a constant or enum, add all defined values to the set.
    -   If `x` comes from a TCC (Configuration) switch and you can see the configured values (or default values) in the context/code, **YOU MUST EXPAND** these values into the set. (e.g., if TCC key `target_idc` has values `sg1,my2`, the set is `["sg1", "my2"]`).
-   **Unresolvable Variable**: If `x` comes from external I/O (e.g., `req.Header.Get("idc")`, `RPC response`) and logic does not restrict it to a known list, mark it as `UNRESOLVABLE`.

#### Step 2: CHECK - Apply Safety Checks with Priority
Based on the **Candidate Value Set** from Step 1, perform the checks.

**Rule 1 (Highest Priority): Unresolvable Dynamic IDC**
-   **Condition**: The variable `x` is marked as `UNRESOLVABLE` in Step 1. (e.g., purely dynamic input from User Request or external opaque package).
    -   *Note*: Do NOT match this rule if `x` is from TCC but you successfully resolved its possible values.
-   **Action**:
    -   Set `checkCategory`: `"Unresolvable Dynamic IDC"`
    -   Set `checkExplain`: `"The IDC source is purely dynamic (e.g., request parameter) and cannot be enumerated statically. Safe refactoring requires manual verification."`

**Rule 2: Intra-VRegion Multi-IDC Call**
-   **Condition**:
    1.  The **Candidate Value Set** contains multiple distinct IDCs.
    2.  AND these IDCs belong to the **same VRegion** (e.g., `sg1` and `sg2` both map to `Singapore-Central`).
-   **scenarios**:
    -   *TCC Fan-out*: A TCC switch allows switching between `my` and `my2` (both Malaysia).
    -   *Loop/Fallback*: Logic iterates `["sg1", "sg2"]` or falls back from one to the other.
-   **Action**:
    -   Set `checkCategory`: `"Intra-VRegion Multi-IDC Call"`
    -   Set `checkExplain`: `"The variable can resolve to multiple IDCs {{print .CandidateValueSet}} which belong to the same VRegion. This pattern forbids VRegion abstraction."`

**Rule 3: Non-3V IDC**
-   **Condition**: Any value in the **Candidate Value Set** is **not** a valid 3V architecture VDC.
-   **Action**:
    -   Set `checkCategory`: `"Non-3V IDC"`
    -   Set `checkExplain`: `"The potential IDC value '<value>' is not a valid 3V IDC."`

**Rule 4 (Default): Normal**
-   **Condition**: The **Candidate Value Set** is fully resolvable, all values are valid 3V IDCs, and no Intra-VRegion conflicts exist.
-   **Action**:
    -   Set `checkCategory`: `"Normal"`
    -   Set `checkExplain`: `"All potential IDC values are 3V-compliant and safe for VRegion abstraction."`

#### Step 3: Final Output Formatting
-   **If `checkCategory` is "Normal"**: Proceed with your tasks and provide the complete analysis report with all fields populated.
-   **If `checkCategory` is NOT "Normal" (i.e., a check failed)**:
    -   You MUST format your final output according to the `IMPORTANT (If Checks Fails)` section below.
    -   This means the primary task (e.g., refactoring) is considered failed.

{{- end }}
