# IDL 一致性检测操作指南

> **触发时机**: 修改 .thrift/.proto 后、提 MR 前、联调前、同步 BAM 时。
>
> IDL 规范参考：
> - `references/idl-spec-thrift.md`（Thrift，覆盖 HTTP/RPC）
> - `references/idl-spec-protobuf.md`（Protobuf，覆盖 HTTP/RPC）

## 总体流程

```
P0 上下文探测 → P1 规范检测 → P2 本地一致性检测 → P3 问题分析 → P4 辅助修复 → P5 远端同步
```

| 用户意图 | 执行范围 | 说明 |
|---------|---------|------|
| 改了 IDL 快速检查 | P0 ~ P4 | 检测规范问题、一致性问题和兼容性风险，提供修复建议 |
| 提交代码 / 提 MR | P0 ~ P4 | 全量检测并修复问题，确保代码质量 |
| 联调前 / 发版前 | P0 ~ P4 | 全量检测并修复问题，确保发布安全 |
| 仅检查 IDL 规范 | P0, P1 | 仅做规范合规检查，不涉及代码一致性 |
| 同步 BAM（需用户明确要求） | P5 | 仅执行远端同步，检测本地与 BAM 差异 |
| 检测并同步 BAM | P0 ~ P5 | 全量检测后同步到 BAM |
| 全量检查 | P0 ~ P4 | 完整检测流程，不含远端同步 |

## ⚠️ 约束

- **不要** 未经确认执行 BAM 版本创建 (P5)
- **不要** 删除用户的 IDL 或业务代码
- P2.1 中执行生成后 **默认回滚**，让用户决定是否保留
- 规范检测中的 Error 级问题必须修复后才能继续后续流程

---

## P0: 上下文探测

### P0.1 项目技术栈探测

首次执行时自动识别项目技术栈：

| 探测项 | 方法 | 示例 |
|-------|------|------|
| IDL 类型 | 扫描 *.thrift / *.proto | `thrift` / `protobuf` |
| 服务类型 | go.mod + hertz_gen/ 或 kitex_gen/ | `http(hertz)` / `rpc(kitex)` |
| 生成命令 | Makefile gen_idl_* → gen.go //go:generate | `make gen_idl_api_server` |
| IDL 目录 | .thrift/.proto 聚集目录 | `api/idl/api_server/` |
| 生成产物目录 | hertz_gen/ 或 kitex_gen/ | `api/idl/hertz_gen/` |
| 服务入口 IDL | 含 service 定义的文件 | `service.thrift` |
| Handler/路由 | **/route.go **/handler*.go | `apps/*/handler/` |
| PSM | 项目配置或询问用户 (BAM 必需) | `xx.yy.api_server` |

### P0.2 IDL 变更 Diff 分析

分析 IDL 文件的 git 变更，提取结构化变更信息：

**触发条件：**
- git diff 中存在 .thrift 或 .proto 文件变更
- 或用户显式指定 commit 范围

**分析维度：**

| 变更类型 | 检测内容 | 影响 |
|---------|---------|------|
| Service 变更 | 新增/删除/修改方法 | 高 - 接口契约变更 |
| Struct 变更 | 新增/删除/修改字段 | 高 - 数据结构变更 |
| Enum 变更 | 新增/删除/修改枚举值 | 中 - 枚举扩展 |
| 注解变更 | HTTP 路由/参数映射变更 | 中 - 路由/行为变更 |
| Include 变更 | 新增/删除引用 | 低 - 依赖变更 |

**Thrift 变更提取规则：**

```
1. Service 方法变更：
   - 正则匹配: service\s+\w+\s*\{([^}]*)\}
   - 提取方法签名: \w+\s+\w+\s*\(\s*\d+:\s*[\w.]+\s+\w+\s*\)
   - 对比变更前后

2. Struct 字段变更：
   - 正则匹配: struct\s+\w+\s*\{([^}]*)\}
   - 提取字段: \d+:\s*(optional|required)?\s*[\w.<>]+\s+\w+
   - 对比字段编号、类型、名称、required/optional

3. Enum 变更：
   - 正则匹配: enum\s+\w+\s*\{([^}]*)\}
   - 提取枚举项: \w+\s*=\s*\d+
   - 对比枚举值与编号

4. HTTP 注解变更：
   - 提取 api.get/post/put/delete 等路由注解
   - 提取 api.query/body/header/path 等参数注解
```

**Protobuf 变更提取规则：**

```
1. Service 方法变更：
   - 正则匹配: service\s+\w+\s*\{([^}]*)\}
   - 提取 rpc 定义: rpc\s+\w+\s*\(\s*[\w.]+\s*\)\s*returns\s*\(\s*[\w.]+\s*\)

2. Message 字段变更：
   - 正则匹配: message\s+\w+\s*\{([^}]*)\}
   - 提取字段: (\w+\s+)?\w+\s+\w+\s*=\s*\d+
   - 对比字段编号、类型、名称、repeated/optional

3. Enum 变更：
   - 同 Thrift

4. HTTP 注解变更：
   - 提取 google.api.http 注解
```

### P0.3 上下文汇总

将探测结果汇总为后续检测的输入：

```yaml
# 项目上下文
project:
  idl_type: thrift
  service_type: http
  psm: example.service.api_server
  gen_command: make gen_idl_api_server
  idl_dir: api/idl/api_server/
  gen_dir: api/idl/hertz_gen/
  service_idl: api/idl/api_server/service.thrift

# 变更上下文
changes:
  commit_range: HEAD~3..HEAD
  idl_files:
    - path: api/idl/api_server/service.thrift
      change_type: modified
  new_methods:
    - name: GetUser
      route: GET /v1/user/:id
  modified_structs:
    - name: GetUserRequest
      field_changes:
        - field_id: 2
          change: added
          type: optional string
          name: NickName
        - field_id: 3
          change: type_modified
          old_type: i64
          new_type: string
          compatibility: breaking
```

---

## P1: 规范检测

### P1.0 规范检测架构

规范检测分为三层，按优先级执行：

```
┌─────────────────────────────────────────────────────────┐
│  Layer 3: 业务规范 (Business Rules)                      │
│  - 方法命名规范、字段命名规范、错误码定义等                 │
│  - 级别: Warning / Info                                 │
├─────────────────────────────────────────────────────────┤
│  Layer 2: 字节体系规范 (ByteDance Standards)            │
│  - BaseResp 255 字段、HTTP 注解规范、PSM 命名等           │
│  - 级别: Error / Warning                                │
├─────────────────────────────────────────────────────────┤
│  Layer 1: 开源规范 (Open Source Standards)              │
│  - 语法正确性、字段编号唯一性、类型定义等                  │
│  - 级别: Error (阻断)                                   │
└─────────────────────────────────────────────────────────┘
```

### P1.1 开源规范检测（Layer 1）

**Thrift 开源规范：**

| 规则 ID | 检测内容 | 级别 | 说明 |
|---------|---------|------|------|
| OS-T001 | 字段编号重复 | Error | 同一 struct/exception 内字段编号必须唯一 |
| OS-T002 | 字段编号非正整数 | Error | 字段编号必须是正整数 |
| OS-T003 | Include 文件不存在 | Error | include 引用的文件必须存在 |
| OS-T004 | 类型引用不存在 | Error | 引用的 struct/enum/typedef 必须定义 |
| OS-T005 | Service 方法参数数量 | Error | 每个方法只能有一个参数 |
| OS-T006 | Service 方法参数类型 | Error | 参数必须是 struct 类型 |
| OS-T007 | Service 方法返回类型 | Error | 返回值必须是 struct 类型 |
| OS-T008 | Enum 值重复 | Error | 枚举值编号不能重复 |

**Protobuf 开源规范：**

| 规则 ID | 检测内容 | 级别 | 说明 |
|---------|---------|------|------|
| OS-P001 | syntax 声明缺失 | Error | 必须声明 `syntax = "proto3"` |
| OS-P002 | 字段编号重复 | Error | 同一 message 内字段编号必须唯一 |
| OS-P003 | Import 文件不存在 | Error | import 引用的文件必须存在 |
| OS-P004 | 字段编号范围 | Error | 字段编号必须在 1-536870911 范围内 |
| OS-P005 | 保留字段号使用 | Error | 不能使用 reserved 的字段编号 |
| OS-P006 | RPC 入参/出参类型 | Error | 必须是 message 类型 |
| OS-P007 | go_package 缺失 | Error | 必须配置 option go_package |
| OS-P008 | Enum 零值缺失 | Warning | 枚举应包含零值默认项 |

### P1.2 字节体系规范检测（Layer 2）

**Thrift 字节体系规范：**

| 规则 ID | 检测内容 | 级别 | 适用场景 |
|---------|---------|------|---------|
| BD-T001 | Response 缺少 BaseResp | Error | 全部 |
| BD-T002 | BaseResp 字段编号非 255 | Error | 全部 |
| BD-T003 | Request 缺少 Base 字段 | Warning | 全部 |
| BD-T004 | Base 字段编号非 255 | Warning | 全部 |
| BD-T005 | HTTP 注解大小写 | Error | HTTP |
| BD-T006 | HTTP Method 缺少路由注解 | Error | HTTP |
| BD-T007 | GET 请求使用 api.body | Error | HTTP |
| BD-T008 | api.serializer 值非法 | Error | HTTP |
| BD-T009 | api.path 字段类型非法 | Error | HTTP |
| BD-T010 | api.query/list 字段类型非法 | Warning | HTTP |
| BD-T011 | api.js_conv 用于非 i64 字段 | Error | HTTP |
| BD-T012 | go.tag 与 api.body key 冲突 | Warning | HTTP |
| BD-T013 | namespace go 格式不规范 | Warning | 全部 |
| BD-T014 | include 未使用 | Warning | 全部 |

**Protobuf 字节体系规范：**

| 规则 ID | 检测内容 | 级别 | 适用场景 |
|---------|---------|------|---------|
| BD-P001 | Response 缺少统一错误承载 | Warning | RPC |
| BD-P002 | package 与 PSM 不对齐 | Warning | 全部 |
| BD-P003 | google.api.http GET 配置 body | Error | HTTP |
| BD-P004 | google.api.http path 字段缺失 | Error | HTTP |
| BD-P005 | 建议避免 streaming RPC | Warning | RPC |

### P1.3 业务规范检测（Layer 3）

| 规则 ID | 检测内容 | 级别 | 说明 |
|---------|---------|------|------|
| BZ-001 | 方法命名不规范 | Warning | 应使用 CreateXxx/DeleteXxx/UpdateXxx/GetXxx/MGetXxx/ListXxx |
| BZ-002 | Request/Response 命名不规范 | Warning | 应为 {Method}Request / {Method}Response |
| BZ-003 | Enum 缺少 Undefined = 0 | Warning | 枚举应包含默认值 |
| BZ-004 | Enum 分隔符不统一 | Warning | 应统一使用 `;` |
| BZ-005 | 使用 set 类型 | Warning | 建议用 map<T, bool> 或 list<T> |
| BZ-006 | 字段设置 default value | Warning | 不建议设置默认值 |
| BZ-007 | 核心字段使用 map<string, string> | Warning | 应显式定义 struct |
| BZ-008 | 字段名以 New 开头 | Warning | 禁止字段名以 New 开头 |
| BZ-009 | 字段名以 Result/Args 结尾 | Warning | 禁止字段名以 Result/Args 结尾 |
| BZ-010 | 时间字段命名不规范 | Warning | 应以 Time 结尾 |
| BZ-011 | 时间类型不规范 | Warning | 应使用 i64 (ms) |
| BZ-012 | 注释风格不统一 | Info | 应使用 `//` 单行注释 |
| BZ-013 | 孤立 struct/enum | Info | 未被任何 service 引用 |
| BZ-014 | ID 字段精度问题 | Info | i64 用于前端需 api.js_conv 或改为 string |

### P1.4 兼容性检测

检测 IDL 变更的向后兼容性：

| 兼容性规则 | 级别 | 说明 |
|-----------|------|------|
| 字段编号被复用 | Error | 已发布的字段编号不能被复用 |
| 字段类型变更 | Error | 修改字段类型为不兼容变更 |
| required 字段新增 | Error | 新增 required 字段会导致老客户端失败 |
| 字段删除未保留编号 | Warning | 删除字段应 reserved 编号 |
| Enum 值删除 | Error | 删除枚举值可能导致解析失败 |
| Enum 值复用 | Error | 复用枚举值编号为不兼容变更 |
| 方法删除 | Error | 删除方法会导致客户端调用失败 |
| 方法签名变更 | Error | 修改方法参数/返回值为不兼容变更 |
| 路由变更 | Warning | 修改 HTTP 路由可能影响存量调用方 |
| 新增 required 字段 | Warning | 新增 required 字段影响老客户端 |

---

## P2: 本地一致性检测

> 检测 IDL 与本地代码的一致性，确保 IDL 变更后相关代码同步更新。
>
> **本阶段在 P1 规范检测完成后执行。**

### P2.1 IDL ↔ 生成代码一致性

> 修改 IDL 后忘记重新生成代码

**检测步骤：**

```
1. 检测 IDL 是否有变更
   - 检查 git diff 中是否存在 .thrift/.proto 文件变更
   - 排除注释、空白字符变更

2. 执行代码生成命令
   - 使用 P0.1 探测到的 gen_command
   - 捕获命令执行结果（成功/失败）

3. 对比生成结果
   - git diff gen_dir
   - 有 diff → 生成代码与 IDL 不一致

4. 处理结果
   - 命令成功 + 有 diff → Error: 需要重新生成
   - 命令失败 + 有 diff → Warning: 生成命令执行失败
   - 无 diff → 通过
```

**关键点：**
- **必须实际执行生成命令**：不能仅依赖文件时间戳判断
- **捕获生成命令执行状态**：若命令失败，应明确提示用户检查
- **展示具体 diff**：让用户清楚看到哪些生成代码需要更新

**修复:**
- 若生成成功：保留生成结果不回滚
- 若生成失败：提示用户检查 gen_command 配置或手动执行

### P2.2 IDL 接口 ↔ 业务代码一致性

> IDL 定义了接口但业务代码未注册/实现

**HTTP (Hertz):**

```
1. service.thrift 路由注解 (api.get/post/put/delete) → {method, path, name}
2. route.go Group()嵌套 + HTTP方法注册 → {method, full_path, handler}
3. 交叉对比 → IDL有&route无=Error | route有&IDL无=Warning | method/path不一致=Error
```

**RPC (Kitex):**

```
1. service.thrift/.proto service方法 → {service_name, method_name}
2. handler目录 grep方法实现 → func.*MethodName(
3. 交叉对比 → IDL有&handler无=Error | handler有&IDL无=Warning
```

**修复:** 生成缺失的路由注册代码或 Handler 方法骨架。

### P2.3 Git 变更一致性

> IDL 相关文件未成对变更

**检测步骤：**

```
步骤 1: 获取变更文件
git diff --name-only HEAD
git diff --name-only --staged
git diff --name-only origin/main...HEAD  # 或用户指定的基准分支

步骤 2: 分类变更文件
idl_files:     匹配 **/*.thrift 或 **/*.proto
gen_files:     匹配 **/hertz_gen/** 或 **/kitex_gen/** 或 **/gen/**
route_files:   匹配 **/route.go 或 **/router.go
handler_files: 匹配 **/handler*.go 或 **/*_handler.go

步骤 3: 判断 IDL 是否有实质变更
- 排除注释变更、空白字符变更
- 检测是否有 struct/service/enum/注解 的实际修改
- 若 IDL 无实质变更，跳过后续检测

步骤 4: 执行生成代码对比
若 idl_files 非空且有实质变更:
  a) 临时执行代码生成命令 (如 make gen_idl_xxx)
  b) git diff gen_dir → 有 diff 则生成代码不一致
  c) 回滚临时生成的代码

步骤 5: 一致性判断
G1. IDL 有实质变更 & 生成代码不一致  → Error: 需要重新生成代码
G2. IDL 有实质变更 & route/handler 未变更 → Warning: 可能需要更新路由/处理器
G3. 生成代码变更 & IDL 无变更      → Error: 手动修改了生成代码
G4. service IDL 新增方法 & route/handler 未变更 → Warning: 可能忘了注册
```

**判断规则详解：**

| 规则 | 触发条件 | 级别 | 修复建议 |
|------|----------|------|----------|
| G1 | IDL 变更 + `git diff gen_dir` 有输出 | Error | 执行生成命令并保留结果 |
| G2 | IDL 变更 + 生成代码无变化（可能是生成命令执行失败） | Warning | 检查生成命令是否正确执行 |
| G3 | 生成代码变更 + IDL 未变更 | Error | `git checkout -- gen_dir` |
| G4 | service 新增方法 + route/handler 未更新 | Warning | 补充路由注册和 handler 实现 |

**关键点：**
- **必须执行生成命令**：仅对比文件时间戳不够准确，需要实际执行生成命令后对比 diff
- **区分实质变更**：注释、空白字符变更不算实质变更，不应触发检测
- **检测生成命令是否成功**：若生成命令执行失败，应作为 Warning 而非跳过检测

**修复:** G1→执行生成命令; G2→检查生成命令; G3→`git checkout -- gen_dir`。

---

## P3: 问题分析

### P3.1 问题分类与优先级

| 优先级 | 类型 | 描述 | 处理方式 |
|-------|------|------|---------|
| P0 | 阻断性错误 | 语法错误、不兼容变更 | 必须立即修复 |
| P1 | 严重问题 | 违反字节规范、接口定义缺失 | 应尽快修复 |
| P2 | 一般问题 | 违反业务规范、命名问题 | 建议修复 |
| P3 | 提示信息 | 代码风格、孤立定义 | 可忽略 |

### P3.2 问题影响范围分析

针对每个问题，分析其影响范围：

**字段变更影响分析：**

```
1. 识别变更字段所属的 struct
2. 找到引用该 struct 的所有方法
3. 找到调用这些方法的所有 handler/client
4. 输出影响链路
```

### P3.3 问题关联分析

分析问题之间的关联关系，识别根因问题。

---

## P4: 辅助修复

### P4.1 自动修复能力

| 问题类型 | 自动修复 | 修复方式 |
|---------|---------|---------|
| BaseResp 缺失 | ✅ | 自动添加 `255: optional base.BaseResp BaseResp` |
| Base 缺失 | ✅ | 自动添加 `255: optional base.Base Base` |
| Enum 缺少默认值 | ✅ | 自动添加 `Undefined = 0` |
| Enum 分隔符不统一 | ✅ | 自动替换为 `;` |
| HTTP 注解大小写 | ✅ | 自动转换为小写 |
| 注释风格不统一 | ✅ | 自动转换为 `//` |
| Include 未使用 | ⚠️ | 需确认后删除 |
| 方法命名不规范 | ❌ | 需人工确认 |
| 字段类型变更 | ❌ | 需人工评估兼容性 |

### P4.2 修复建议生成

对于无法自动修复的问题，生成详细修复建议，包括：
- 问题描述
- 影响评估
- 多个修复方案对比
- 推荐方案及理由

### P4.3 修复后验证

修复后自动执行验证：

```
1. 重新执行 P1 规范检测
2. 验证修复的问题已解决
3. 检查是否引入新问题
4. 输出验证报告
```

---

## P5: 远端同步

> 检测本地 IDL 与 BAM 远端的一致性，并支持同步操作。

**⚠️ 触发条件：**

- **必须** 用户主动、清晰地提到要执行远端同步操作，例如：
  - "同步到 BAM"、"同步 BAM"、"推送到 BAM"
  - "更新远端接口"、"同步远端"
  - "执行 P5"、"执行远端同步"
- **禁止** 在用户未明确要求时自动执行 P5
- **禁止** 将"提 MR 前"、"联调前"等场景理解为需要执行远端同步

**仅在满足触发条件时执行本章节内容。**

### P5.1 本地 IDL ↔ BAM 差异

> 本地与 BAM 远端接口不一致

**前置:** PSM + 可选 BAM 分支 (默认 master)

```
1. 查询 BAM 接口列表:
   bytedcli --json bam method list --psm "xx.yy.api_server"
   RPC 服务加 --ep-type rpc

2. 解析本地 service IDL:
   HTTP: {method, path} 从路由注解
   RPC:  {method_name} 从 service 定义

3. 对比 (HTTP key=method+path, RPC key=method_name):
   本地有 & BAM无 → 忘了同步
   BAM有 & 本地无 → 协作者更新了，需 git pull

4. (可选) 字段级对比:
   bytedcli --json bam method get --psm "xx.yy.api_server" --method "MethodName"
   获取 schema 与本地 IDL 字段对比
```

**修复:** P5.2 同步 BAM，或 git pull 拉取最新。

### P5.2 本地 → BAM 同步

> 将本地 IDL 变更推送到 BAM

```
1. 运行 P5.1 获取差异
2. 展示预览: +N 新增 / ~M 修改 / -K 删除
3. ⚠️ 请求用户确认
4. 执行同步:
   bytedcli bam idl update --psm "xx.yy.api_server" --branch master --next-version
5. 再次 P5.1 验证一致
```

---

## IDL 检测报告

完成 P0-P4 检测后，生成统一的 IDL 检测报告。

**报告模板**: `templates/idl-check-report.md`

### 报告章节

| 章节 | 内容 |
|------|------|
| 概览 | 检测结果统计摘要，包含 Errors/Warnings/Info 汇总 |
| 一、项目上下文 | P0 探测结果，包含技术栈、路径配置等 |
| 二、IDL 变更 Diff | P0.2 分析结果，展示具体的 IDL 变更内容 |
| 三、规范检测结果 | P1 检测结果，按规范层分类展示问题 |
| 四、本地一致性检测结果 | P2 检测结果，P2.1/P2.2/P2.3 一致性检查 |
| 五、问题分析与修复建议 | P3-P4 分析结果，包含影响范围和修复方案 |
| 六、检测总结 | 待处理事项清单和后续步骤 |

---

## 远端能力依赖

| 能力 | 调用方式 | 用于 |
|------|---------|------|
| BAM 接口列表 | `bytedcli --json bam method list --psm "..."` | P5 |
| BAM 接口详情 | `bytedcli --json bam method get --psm "..." --method "..."` | P5 |
| BAM 创建版本 | `bytedcli bam idl update --psm "..." --branch "..." --next-version` | P5 ⚠️ |
| IDL 解析 | Agent 读取 .thrift/.proto | P0~P4 |
| 代码生成 | 执行 gen_command | P2 |
| Git | git diff / git checkout | P0, P2 |

---

## 常见问题

### 找不到生成命令

- 原因：项目未使用标准 Makefile 或 go:generate 模式
- 处理：手动指定生成命令，例如 `make gen`、`hz update` 或 `kitex -module ...`
- 补充：检查 Makefile 中是否有 `gen_idl_*` 目标，或 `gen.go` 中是否有 `//go:generate` 指令

### P2 本地一致性检测异常

- 原因：生成命令执行失败或路径不正确
- 处理：检查 gen_command 是否正确，确认 hz/kitex 工具链可用
- 补充：RPC 服务需添加 `--ep-type rpc` 参数

### P2.1 生成命令执行失败

- 原因：本地缺少 hertz/kitex 工具链或 go 依赖
- 处理：确认 `hz`（Hertz）或 `kitex`（Kitex）命令可用；执行 `go mod tidy` 确保依赖完整
- 补充：生成命令失败不影响其他检查，可跳过 P2.1 继续

### 规范检测误报

- 原因：项目有特殊约定，与通用规范不一致
- 处理：可在项目根目录创建 `.idl-lint.yaml` 配置文件，禁用或调整特定规则
- 示例配置：
  ```yaml
  rules:
    disable:
      - BZ-001  # 禁用方法命名规范检查
    severity:
      BD-T003: info  # 降低 Request 缺少 Base 字段的严重级别
  ```

### 兼容性检测不覆盖所有场景

- 原因：兼容性检测基于语法层面，无法覆盖所有语义兼容场景
- 处理：对于关键接口变更，建议人工评估影响范围
- 补充：可结合 A/B 实验平台做灰度验证
