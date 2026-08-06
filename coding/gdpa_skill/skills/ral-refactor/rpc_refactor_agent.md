# RPC Refactoring: Replace Hard-coded IDC Routing with RAL Framework

## Overview
We need to migrate from hard-coded datacenter routing in RPC calls (`calloption.WithIDC("idc")`) to the configuration-based Resource Access Layer (RAL) framework. This refactoring will improve maintainability and reliability of cross-datacenter service calls by moving routing rules to configuration files.

## Pre-requisite: Read Pattern Analysis Report

在开始重构之前，必须先读取 `pattern_analysis.md` 文档获取所有待重构的 WithIDC 点位信息。

### 读取步骤：
1. 读取项目根目录下的 `pattern_analysis.md` 文件
2. 解析文档中的每个 Point 信息
3. 对每个点位按顺序执行重构

### 遍历处理逻辑：
```
for each Point in pattern_analysis.md:
    1. 读取该点位的 Category、va_involved、ap_involved、routingCodeMap 信息
    2. 根据 Category 判断是否需要重构（跳过 NoWithIDC 和不支持的场景）
    3. 根据 routingCodeMap 中的信息定位代码位置
    4. 执行重构操作（创建/更新 RAL 配置，修改代码）
    5. 记录重构结果
```

### 输出要求：
- 完成所有点位重构后，输出重构结果摘要
- 包含成功重构的点位数量、跳过的点位数量及原因、失败的点位数量及原因

## Background: 3V Datacenter Structure
Our infrastructure uses a hierarchical "3V" structure:
- **VGeo** (Compliance Zones): Major geographic regions
- **VRegion** (Logical Regions): Subdivisions within VGeos
- **VDC** (Data Centers): Physical facilities

```yaml
VGeo-EU: # Previous as Clover. VGeo Name is VGeo-EU, including VGeo- prefix.
  EU-TTP: # VRegion Name is EU-TTP, excluding VRegion- prefix.
    - ie
  EU-TTP2:
    - no1a
  US-EastRed: # Previously referred to as `i18n` in the context of Region; also known as `gcp`
    - useast2a
    - useast2b

VGeo-ROW: 
  Singapore-Central: # Previously referred to as `alisg` in the context of Region
    - my
    - my2
    - my3
    - sg1
  US-East: # Also known as `va`
    - maliva

VGeo-US: # Previously as Texas
  US-TTP: # Also known as `ttp`
    - useast5
  US-TTP2: # Also known as `ttp2`
    - useast8

NO-VGeo-For-Testing-Environments:
  US-BOE:
    - boei18n
    - boettp
  China-BOE:
    - boe

# Our service will not be deployed in the following data centers by default, unless they are explicitly referenced in the code.
NO-VGeo-For-These-VRegions:
  Asia-SouthEastBD:
    - mya
    - myb
    - myc
  US-EastBD:
    - useast9a
    - useast12a
    - useast13a
    - useast14a
  Asia-CIS:
    - mycisa
    - mycisb
  China-North:
    - hl
    - lf
    - lq
    - yg
    - gl
  China-East:
    - pd
    - hj
    - yz
    - zjg
```

3V Relevant Functions:
```go
package main

import (
    "code.byted.org/gopkg/env" // Both gopkg/env and gdp/env are ok. Follow original codes.
    "code.byted.org/tiktok/region_lib"
)

func Example() {
  env.IDC() // return current VDC
  env.DC_${VDC.upper()} // constants for each VDC

  env.GetCurrentVRegion() // return current VRegion
  env.VRegion_${VRegion.replace('-', '').upper() } // constants for each VRegion

  region_lib.VGeo() // return current VGeo
  region_lib.VGEO_ROW, region_lib.VGEO_US,  region_lib.VGEO_EU // constant for VGeos

  // Deprecated: Use env.GetCurrentVRegion() instead
  env.Region() // return current Region
  env.R_MALIVA (env.VREGION_USEAST), env.R_ALISG (env.VREGION_SINGAPORECENTRAL), env.R_USTTP, env.R_USTTP2, env.I18N (env.VREGION_USEASTRED), env.R_EUTTP, env.R_EUTTP2 // constants for each Region
  
  env.IsBoe() // return true if in "US-BOE" or "China-BOE".
  env.IsProduct()  // equivalent to !env.IsBoe()
}
```

For environment functions not listed above, search its definition to understand its behavior as needed.

## 1. Current Implementation (To Be Replaced)

### Pattern A: Using overpass/p_s_m RPC Stub SDK
```go
package main

// overpass/p_s_m is the rpc stub code repository
// The prefix overpass/ is fixed.
// In this example, iesarch.devflow.api is the rpc target service name, which is called PSM (product.subsystem.module).
// So the overpass rpc stub code repository is overpass/iesarch_devflow_api
import (
	"code.byted.org/overpass/common/option/calloption"   // option for per rpc call
	"code.byted.org/overpass/common/option/clientoption" // option for client

	"code.byted.org/overpass/iesarch_devflow_api/kitex_gen/iesarch/devflow/api" // rpc request and response definition in code.byted.org/overpass/p_s_m/kitex_gen/x/x, x/x is related to the thrift-idl namespace.
	"code.byted.org/overpass/iesarch_devflow_api/rpc/iesarch_devflow_api"       // rpc methols in code.byted.org/overpass/p_s_m/rpc/p_s_m package
)

func Example() {
	req := &api.GetUserRequest{
		UserID: userID,
	}

	// If WithIDC is not set, this rpc call will be routed to the same IDC where the caller is. Equals to `calloption.WithIDC(env.IDC())`
	resp, err := iesarch_devflow_api.RawCall.GetUser(ctx, req)
	resp, err := iesarch_devflow_api.GetUser(ctx, userID) // flatten req fields

	// Hard-coded IDC routing with per-call option
	resp, err := iesarch_devflow_api.RawCall.GetUser(ctx, req, calloption.WithIDC("my"))
	resp, err := iesarch_devflow_api.GetUser(ctx, userID, calloption.WithIDC("my"))

	// OR with global client option
	iesarch_devflow_api.InitDefaultClientOptions(clientoption.WithIDC("my"))

	// OR with client-level option
	cli := iesarch_devflow_api.NewClient("iesarch.devflow.api", clientoption.WithIDC("my")) // equivalent to `iesarch_devflow_api.NewClient("iesarch.devflow.api.service.my")`
	resp, err := cli.RawCall.GetUser(ctx, req) // route to my IDC
	resp, err := cli.GetUser(ctx, userID)      // route to my IDC
}
```

### Pattern B: Using Non-Overpass RPC Stub Codes

```go
package main

import (
   "code.byted.org/kite/kitex/client"
   "code.byted.org/kite/kitex/client/callopt"

   "code.byted.org/your/project/kitex_gen/iesarch/devflow/api/xservice" // rpc service and methods definition in this project
)

func Example() {
   // Hard-coded IDC with client option
   cli := xservice.NewClient("iesarch.devflow.api", client.WithIDC("my")) // equivalent to `xservice.NewClient("iesarch.devflow.api.service.my")`
   resp, err := cli.GetUser(ctx, req)

   // OR with per-call option
   cli := xservice.NewClient("iesarch.devflow.api")
   resp, err := cli.GetUser(ctx, req, callopt.WithIDC("my"))
}
```

## 2. Target Implementation (RAL Framework)

{{ .delete_withidc_instruction }}

### Step 1: Define RAL Configuration Files

Create/update configuration files at: `{workdir}/conf/ral/services/`. Transform the hard-coded IDC routing logic into RAL configuration files while preserving the original routing rules.

```
conf/
|--- ral/
|   |-- services/
|       |-- rpc.yaml               # Default routing rules
|       |-- rpc.<VRegion>.yaml     # VRegion-specific overrides (optional)
|       |-- rpc.<VGeo>.yaml        # VGeo-specific overrides (optional)
```

Example configuration:

```yaml
# {workdir}/conf/ral/services/rpc.yaml (Required).
a_meaningful_identifier: # Prefer use 'p.s.m'.replace('.', '_')
  PSM: p.s.m             # Required. ⚠️ 禁止猜测 PSM，必须通过工具获取: [get_ral_info_by_package_path.md](./tools/get_ral_info_by_package_path.md)
  Protocol: rpc          # Required. Always "rpc"
  # ConnTimeout/ReqTimeout are optional client-level timeout settings. Corresponding to original clientoption.WithXTimeout (client-level). For original calloption.WithXTimeout (method-level), preserve them in codes, DO NOT set them to ConnTimeout/ReqTimeout.
  ConnTimeout: 50ms      # Optional. Default: 50ms. Corresponding to clientoption.WithConnectTimeout.
  ReqTimeout: 1s         # Optional. Default: 1s. Corresponding to clientoption.WithRPCTimeout.
  AdditionalCfg:
    RegionRouter:
      Methods:
        MethodName:     # Specific method that needs routing.
          TargetVRegion: ~  # "~" means don't set TargetVRegion, thus route to the caller's current VDC. Corresponding to original rpc call without WithIDC or WithIDC(env.IDC()).
        AnotherMethod:  
          TargetVRegion: US-East  # Specific target VRegion. No matter where the caller is, it will always route to US-East if not overridden by rpc.<3V>.yaml.
```

```yaml
# {workdir}/conf/ral/services/rpc.<3V>.yaml (Optional)
# Only needed when caller in <3V> (VGeo/VRegion/VDC) requires different routing or timeout setting than default
a_meaningful_identifier: # Be Same with that in rpc.yaml.
  # PSM and Protocol cannot be overridden, thus should not be set here.
  # Optional client-level timeout setting. If and only if it is client-level timeout and different from that in rpc.yaml.
  ConnTimeout: 100ms
  ReqTimeout: 2s
  AdditionalCfg:
    RegionRouter:
      Methods:
        MethodName:
          TargetVRegion: Singapore-Central  # Override for this VRegion
```

The above routing rule configuration files equals to original RPC call using `calloption.WithIDC("x")` or `calloption.WithVRegion("xRegion")` like this (ignoring timeout settings):
```go
if env.GetCurrentVRegion() == <VRegion> {
    cli.RawCall.MethodName(ctx, req, calloption.WithVRegion("Singapore-Central"))
} else {
    cli.RawCall.MethodName(ctx, req)
}

cli.RawCall.AnotherMethod(ctx, req, calloption.WithVRegion("US-East"))
```

* When evaluating condition functions that determine routing logic:
  - IMPORTANT: Use the `read_definition` tool to confirm its behavior since function name can be deceptive.
* Original condition expression can be expressed using `rpc.<3V>.yaml`
  - `3V` in `rpc.<3V>.yaml` in RAL configurations can be any VGeo/VRegion/VDC, which means it only take effects when caller in that VGeo/VRegion/VDC.
  - Priorities of `3V` in `rpc.<3V>.yaml`: VDC > VRegion > VGeo > default.
  - `rpc.<VDC>.yaml` is not recommended, use `rpc.<VGeo>.yaml` (most preferred) or `rpc.<VRegion>.yaml`.
  - `env.IDC()=="X"` can be expressed as `rpc<VRegionOfX>.yaml`. Upgrade VDC to VRegion granularity is acceptable since our infrastructure has erased differences between multiple VDCs in the same VRegion.
  - Strictly use the exact VGeo/VRegion names (not alias); Case insensitive (prefer lower case) for `rpc<3V>.yaml`. e.g.: `rpc.vgeo-eu.yaml`  instead of `rpc.eu.yaml`, `rpc.eu-ttp.yaml` instead of `rpc.euttp.yaml` nor `rpc.vregion-eu-ttp.yaml`, `rpc.us-ttp2.yaml` instead of `rpc.ttp2.yaml`.
* Original WithIDC can be substituted by TargetVRegion
  - RAL use targetVRegion instead of targetIDC. 
  - Original `calloption.WithIDC("x")` should be replaced with `TargetVRegion: <VRegionOfX>` in the configuration files.
  - Strictly use the VRegion names (case-sensitive) for `TargetVRegion`.
* Note:
  - Every called rpc method must have a default routing rule in `rpc.yaml`. Set `TargetVRegion: ~` when default behavior is not set WithIDC.
  - Ignore `cn` environment. Don't use `rpc.cn.yaml`.
  - Original `dev`, `test`, `boe` keywords can be represented as `rpc.us-boe.yaml` (`China-BOE` can be omitted, since we now only use `US-BOE` as Testing Environment).
  - If original codes use IDCs not listed in the 3V structure, you can put a reasonable placeholder in the configuration file with a TODO comment.
  - Make configuration files concise:
    - **Don't** use <3V> when the content is the same with default value in `rpc.yaml`.
    - Replace multiple `rpc<VRegion>.yaml` with the same content and in the same VGeo to a single `rpc.<VGeo>.yaml`
  - Criterial of ConnTimeout/ReqTimeout:
    - Omit it when not given in original codes.
    - Only client-level is allowed in ral configuration files. For method-level timeout, set it in codes.
    - Preserve the timeout logic as original codes. For example: original codes tend to add extra timeout for cross-datacenter calls, in this case, you should also keep this logic.
    - If set timeout in `rpc.<3V>.yaml`, must also set it in `rpc.yaml` (When no global timeout exists in original codes, place a null value in rpc.yaml like `ConnTimeout: ~`).
    - Timeout in `rpc.yaml` takes effect globally. 
    - To specially unset timeout in VRegionA, set `ConnTimeout: ~` in `rpc.VRegionA.yaml`.
    - Omit other timeout settings in original codes. Only keep `ConnTimeout` and `ReqTimeout`.
  - Criterial of `MethodName` in `AdditionalCfg.RegionRouter.Methods`:
    - Can be "*", which means all methods. In YAML, you must enclose * in double quotes ("").
    - IMPORTANT: Use * when and only when original codes use client-level WithIDC like `NewClient("p.s.m", clientoption.WithIDC("x"))` or `InitDefaultClientOptions(clientoption.WithIDC("x"))`.
    - DO NOT combine MethodName from multiple rpc call level WithIDC to "*".
    - Priority: SpecificMethodName > *. If both are set for the same psm in the same file, then SpecificMethodName will override *.
    - `MethodName` in `rpc.<3V>.yaml` must be the same as that in `rpc.yaml`. DO NOT use SpecificMethodName in `rpc.<3V>.yaml` to override * in `rpc.yaml`.

### Step 2: Update RPC Call Implementation

Replace existing RPC calls with RAL RPC Stub implementations.

1. Locate rpc call sites to be refactored.
   - For WithIDC options in client initialization (`NewClient` or `InitDefaultClientOptions`), get references of the client to find rpc call sites.
   - For `InitDefaultClientOptions`, you must grep `code.byted.org/overpass/p_s_m/rpc/p_s_m` to find **ALL** files that import this package, then locate **ALL** related rpc call sites in these files.
2. Replace every original RPC call (`code.byted.org/overpass/p_s_m/rpc/p_s_m.RawCall.Method()` and `cli.Method()`) with RAL RPC Sub (`code.byted.org/overpass/p_s_m/gdp/rpc/p_s_m.Method()`).
   - There is no `NewClient`, `Client` in RAL. The only way to call RPC is `code.byted.org/overpass/p_s_m/gdp/rpc/p_s_m.Method()`.
3. Remove legacy client initialization if any.
   - Target service has been declared in RAL configuration files, so client variable is not needed anymore.
   - There is no `NewClient`, `Client` in RAL. The only way to call RPC is `code.byted.org/overpass/p_s_m/gdp/rpc/p_s_m.Method()`.

Guidelines:
* RPC Stub for RAL
  - RAL must use RPC Stub in `code.byted.org/overpass/p_s_m/gdp/rpc/p_s_m`
  - RAL must use Request/Response definitions in `code.byted.org/overpass/p_s_m/kitex_gen/<sub_path>` (same with Pattern A)
* How determine `p_s_m` and PSM in package name:
  - **⚠️ 禁止从 package path 猜测 PSM**，必须通过工具获取: [get_ral_info_by_package_path.md](./tools/get_ral_info_by_package_path.md)
  - For pattern A, original codes use `code.byted.org/overpass/p_s_m/rpc/p_s_m`
  - For pattern B, original codes use `xservice.NewClient("p.s.m")`, so it should be `code.byted.org/overpass/p_s_m/gdp/rpc/p_s_m`.
* How determine `<sub_path>` under `kitex_gen`:
  - `<sub_path>` are determined by namespace defined in the thrift IDL files.
  - For pattern A, and pattern B, and RAL, `<sub_path>` in `kitex_gen/<sub_path>` should be the same.

```go
package main

import (
   "code.byted.org/gdp/ral/c/rpc"
   "code.byted.org/overpass/iesarch_devflow_api/kitex_gen/iesarch/devflow/api" // rpc request and response definition in code.byted.org/overpass/p_s_m/kitex_gen/x/x, x/x is related to the thrift-idl namespace.
   // Always add import alias for code.byted.org/overpass/p_s_m/gdp/rpc/p_s_m to avoid confusion with previous raw overpass rpc stub package.
   gdp_iesarch_devflow_api "code.byted.org/overpass/iesarch_devflow_api/gdp/rpc/iesarch_devflow_api" // rpc methods in code.byted.org/overpass/p_s_m/gdp/rpc/p_s_m package. Note: it is p_s_m/gdp/rpc/p_s_m, which is different from previous raw overpass rpc stub package p_s_m/rpc/p_s_m
)

func Example() {
   req := &api.GetUserRequest{
      UserID: userID,
   }
   resp, err := gdp_iesarch_devflow_api.GetUser(ctx, req, // Note: parameters are the same to original iesarch_devflow_api.RawCall.GetUser, not iesarch_devflow_api.GetUser
      // If origin code use something like `err.(*rpc_error.RPCError)`, `rpcErr.Is(rpc_error.RPC_STATUS_CODE_NOT_ZERO)` to check error type,
      // then you must use `rpc.WithOverpassError()` to force ral returns the same error, so that the original error handling logic can still work.
	  // Otherwise, you should not add this option.
      rpc.WithOverpassError(),
	  
	  // For rpc call options (calloption.WithX, callopt.WithX) and client options (if not set in RAL YAML files), you should replace them to RAL options as shown below.
      // leave empty if none in original codes.
      rpc.WithRPCTimeout(10*time.Second),
      rpc.WithConnTimeout(1000*time.Millisecond),
      rpc.WithCluster("cluster"),
      rpc.WithRetry(3),
	  rpc.WithoutExtractBizError(), // corresponding to `calloption.WithoutOverpassErrHandler`, `clientoption.WithoutOverpassErrHandler()` in original code
	  
	  // Omit options not listed above
   )
   
   // IMPORTANT: NEVER USE NewXClient nor InitDefaultClientOptions in RAL. Instead, you should declare target service x in configuration files and use gdp_x.Method() directly.
}
```

Note:

1. Tips for timeout etc. option handling:

Preserve the VDC/VRegion-specific timeout and retry configurations exactly as they are in the original implementation.

Example 1 (calloption.WithXOption, method-level):

Original code:

```go

if env.IDC() == "XIDC" {
    x.RawCall.Method(ctx, req, calloption.WithConnectTimeout(1*time.Second), calloption.WithRPCTimeout(3*time.Second), calloption.WithRetry(3), calloption.WithIDC("TargetIDC"))
}else{
    x.Method(ctx, req)
}
```

The refactored code should look like this:
  1. Move the routing logic to the RAL configuration file. 
  2. Preserve timeout and retry settings in codes. DO NOT set them in RAL configuration file, because they are method-level options, not client-level settings in original codes.
  3. IMPORTANT: Prefer to use `region_lib.VGeo()` or `env.GetCurrentVRegion()` instead of `env.IDC()` in codes to determine the current environment.

```yaml
# {workdir}/conf/ral/services/rpc.yaml
a_meaningful_identifier:
    PSM: p.s.m
    Protocol: rpc
    # DO NOT set method-level timeout at here. Preserve them in codes.
    AdditionalCfg:
      RegionRouter:
        Methods:
          Method: #  DO NOT use * here, since it is method-level option.
            TargetVRegion: ~
```
```yaml
# {workdir}/conf/ral/services/rpc.VRegionOfXIDC.yaml or rpc.XIDC.yaml. Prefer to use rpc.VRegionOfXIDC.yaml if compatible with existing logic.
a_meaningful_identifier:
    # DO NOT set method-level timeout at here. Preserve them in codes.
    AdditionalCfg:
      RegionRouter:
        Methods:
          Method:
            TargetVRegion: VRegionOfTargetIDC
```

```go
// Never replace these conditional configurations with global/unconditional timeout settings. The VRegion-specific behavior must be preserved exactly as in the original code, just adapted to the new VRegion paradigm.
if env.GetCurrentVRegion() == "VRegionOfXIDC" { // This conditional logic must be preserved. But should be adapted to use region_lib.VGeo() or env.GetCurrentVRegion() instead of env.IDC().
    opts = append(rpc.WithConnTimeout(1*time.Second), rpc.WithRPCTimeout(3*time.Second), rpc.WithRetry(3))
}
gdp_x.Method(ctx, req, opts)
```

Example 2 (clientoption.WithXOption, client-level):

Original code:

```go
if env.IDC() == "XIDC" {
    cli = x.NewClient("p.s.m", clientoption.WithIDC("TargetIDC"), clientoption.WithConnectTimeout(1*time.Second), clientoption.WithRPCTimeout(3*time.Second))
    // OR x.InitDefaultClientOptions(...)
}else{
	cli = x.NewClient("p.s.m")
}

cli.RawCall.Method(ctx, req)
```

The refactored code should look like this:
  - Move routing logic and client-level timeout options to RAL configuration file.

```yaml
# {workdir}/conf/ral/services/rpc.yaml
a_meaningful_identifier:
    PSM: p.s.m
    Protocol: rpc
    ConnTimeout: ~
    ReqTimeout: ~
    AdditionalCfg:
      RegionRouter:
        Methods:
          "*":
            TargetVRegion: ~
```
```yaml
# {workdir}/conf/ral/services/rpc.VRegionOfXIDC.yaml or rpc.XIDC.yaml. Prefer to use rpc.VRegionOfXIDC.yaml if compatible with existing logic.
a_meaningful_identifier:
    ConnTimeout: 1s
    ReqTimeout: 3s
    AdditionalCfg:
      RegionRouter:
        Methods:
          "*":
            TargetVRegion: VRegionOfTargetIDC
```

```go
gdp_x.Method(ctx, req)
```

2. Tips when handling conditional rpc call.

If original code makes an RPC call only if a specific condition is met (e.g., the current virtual region is singapore-central)

```go
if env.GetCurrentVRegion() == env.VREGION_SINGAPORECENTRAL {
    x.RawCall.Method(ctx, req, calloption.WithIDC("my"))
}
// no rpc call in other conditions
```

You must preserve the condition. It should be refactored to:

```go
if env.GetCurrentVRegion() == env.VREGION_SINGAPORECENTRAL { // Maintain the original if condition
    gdp_x.Method(ctx, req)
}
```
And set `TargetVRegion: ~` in `rpc.yaml`, `TargetVRegion: Singapore-Central` in `rpc.singapore-central.yaml`.


3. Tips when handling global rpc clients like `cli := x.NewClient("p.s.m", clientoption.WithIDC("x"))`
  - Firstly, locate all RPC call sites using the `cli`, like `cli.<Method>` or `cli.RawCall.<Method>`.
  - Secondly, replace original `cli.<Method>` or `cli.RawCall.<Method>` with `gdp_x.<Method>`. Note: You only need to refactor the references of `cli`, because in the next step, `cli` will be deleted, and other codes will preserve unchanged.
  - Thirdly, use minimal refactoring to remove the `cli` variable and it's initialization.
  - IMPORTANT: RAL does not support `NewClient`. Besides, never replace `NewClient` with a wrapper client type that encapsulates RPC calls using RAL. You should refactor all references of `cli` to use `gdp_x.<Method>` directly, and then delete `cli` variable.

4. Tips for analyzing existing routing rules:
  - If the targetIDC in `calloption.WithIDC(targetIDC)` is not a constant string, you should trace and analyze the code to find out how it is determined.
  - If the targetIDC variable is determined by configuration:
    - Our static configuration files always locate in `conf/` directory (project root). You can access them.
      - Original codes may use `os.ReadFile`, `viper`, `gdp/config` etc. to read configuration files.
    - Our dynamic configuration platform is called TCC.
      - Original codes may use `gopkg/tccclient`, `gdp/config` etc. to get remote dynamic configurations.
      - Since you cannot access TCC currently, you can add TODO comments in `rpc.yaml` to indicate that it should be added later.
      - Tips: Supplementary information at the end may include the configuration content on TCC. If not, use placeholder with TODO comment.

## Deliverables

1. RAL configuration files with routing rules that match current behavior.
2. Refactored code that uses RAL framework instead of hard-coded IDC routing
3. Removal of all hard-coded IDC routing code being refactored

## Reference:

### Common Refactoring Best Practice

* DO NOT ADD COMMENTS unless asked.
* ⚠️Audit Call Sites Before Deleting: Before deleting a function, verify that it is no longer referenced or called anywhere in the codebase to avoid unintended errors.
* Run `go build -o /dev/null .` to prevent build artifacts from being accidentally committed.
* Never run `go mod init`, since you have already under a module. If not `go.mod` exists under the current directory, it must exist in the parent directory, but you cannot access it currently.
* git
  * Use `git add . && git diff HEAD --relative -- '*.go' '*.yaml'` to review your changes.
  * Use `git checkout -- path/filename` to discard changes in a specific file if needed.
  * Never run `git commit` to commit changes.
* Code Search
  * Primary: Use `get_references`, `read_definition` to search workspace symbols.
  * Fallback: Use `gopls references path/to/file:line:column` or `grep_codebase`, especially for searching struct fields, local variables, etc.
* Code Edit
  * When editing a .go file using the `replace_in_file` tool, it will automatically apply standard Go formatting (gofmt) and remove any unused imports. Generally, you need not worry about formatting or import cleanup.
  * DO NOT add imports before using them, since `replace_in_file` will automatically remove unused imports.

### Type Conversion Functions

RAL and Pattern A both use Request/Response types from `code.byted.org/overpass/p_s_m/kitex_gen/<sub_path>`
Pattern B currently uses types from other modules

When refactoring Pattern B to use RAL, we need to:

* Change its types to match `code.byted.org/overpass/p_s_m/kitex_gen/<sub_path>`
* Handle potential type mismatches:
  - Our RAL use types from `code.byted.org/overpass/p_s_m/kitex_gen/<sub_path>`
  - Other code (either within our project or third-party) may use types from `code.byted.org/another/module/kitex_gen/<sub_path>`

This creates type mismatches that need resolution.

Resolution Rules:

* Primary Approach: Standardize all codes to use `code.byted.org/overpass/p_s_m/kitex_gen/<sub_path>`
  - Apply this whenever it doesn't introduce compatibility issues with existing code
* Fallback Approach: Implement type conversion functions using the conv package like below:

```go
package main

import (
	// always add import alias for tangofeatures/conv to avoid confusion
	typconv "code.byted.org/lang/tangofeatures/conv"

	anotherexample `code.byted.org/another/module/kitex_gen/example`
	`code.byted.org/overpass/p_s_m/kitex_gen/example`
)

func ConvExample(req *example.GetUserRequest) (*anotherexample.GetUserRequest, error) {
	anotherReq := anotherexample.GetUserRequest{}
	if err := (&typconv.Converter{}).TypeToType(req, &anotherReq); err != nil {
		return nil, err
	}

	return &anotherReq, nil
}

```

### Cases Not Supported by the RAL Framework

The following scenarios should not be refactored to use the RAL framework. Stop task with message "No refactoring performed. [details]" and **failed** status.

* Special RPC Stub Repository
  - `code.byted.org/tikcast/rpc_gen` 
  - `code.byted.org/tikcast/rpc_p_s_m` 
  - These are custom RPC stub repositories exclusively for the `tikcast` project.

* Dynamic RPC Calls
  - Any usage of `code.byted.org/iesarch/dynamic_thrift` `code.byted.org/iesarch/dynamic_thrift/kitex_rpc_client.Client.RpcCall`
  - These dynamic call patterns should remain unchanged.

* Non-RPC-Call:
  - Do not refactor these patterns: `code.byted.org/gopkg/tos.WithIDC`, `code.byted.org/content/doorgod-sdk-go/clients.WithIDC`, `code.byted.org/tiktok/passport_golib/middleware.NewTTPProxyMWWithServiceInfo` etc.
  - Note: Only refactor RPC calls using `overpass` or `kitex_gen`.

### Add Third-Party Package Dependencies
To safely add a new third-party package dependency:
- Run `go list -m <package_path> || go get <package_path>` This will only add the package if it's not already present.
- When go get fails:
  - Verify the package path is correct.
  - If you are sure the package path is correct, but it cannot be resolved, you can refuse the task with an explanation.

### Handling Errors in Third-Party Libraries
* For `sonic`, `frugal`, `choleraehyq/pid`, try to update them to latest version.
  - Since you don't know the exact version, just use `latest`. If `latest` doesn't work, try `master`, especially for `overpass/x`.
  - When go get fails, it may be due to a replace directive in `go.mod` file that blocks the intended update or installation. Remove the problematic replace directive using `go mod edit -dropreplace=<module_path>` as needed.
* For `thrift`, `dynamicgo` related 3rd library, try to add `replace github.com/apache/thrift => github.com/apache/thrift v0.13.0` in go.mod.
* For `code.byted.org/gdp/` related 3rd library, try to update them to `latest` version (for `code.byted.org/gdp/*` use `latest` rather than `master`).
* For `tiktok/apimodels/commons`: unknown revision commons/v0.0.0
  - Run `go mod edit -droprequire code.byted.org/tiktok/dtoconv && go get code.byted.org/tiktok/dtoconv` if `tiktok/dtoconv` is included.
  - Run `go mod edit -droprequire code.byted.org/tiktok/apimodels/commons && go get code.byted.org/tiktok/apimodels/commons`


## Supplementary information from the repository maintainers

{{ if gt (len .human_supplementary_info) 0 -}}
{{ .human_supplementary_info }}
{{- else -}}
No supplementary information provided.
{{- end }}