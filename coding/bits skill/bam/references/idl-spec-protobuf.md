# Protobuf IDL 规范（HTTP / RPC）

本文档用于描述在字节体系内使用 Protocol Buffers（建议以 **proto3** 为主）进行接口定义时的统一规范，覆盖 **RPC（Kitex/GRPC 风格）** 与 **HTTP（转码/网关）** 两类场景。

## 0. 适用范围与规范分层

- **开源规范**：Google Protocol Buffers / gRPC / API Design Guide 的通用原则。
- **字节体系规范**：在字节基础设施（BAM/Kitex/网关/代码生成）中可解析、可生成代码、可稳定演进的约束。
- **业务规范**：跨业务通用最佳实践。

> Protobuf 在“语义兼容性”与“字段号管理”上比 Thrift 更敏感，建议将字段号治理作为强制流程的一部分。

---

## 1. 开源规范（proto3 基线）

### 1.1 基础语法

1. **统一使用 proto3**：`syntax = "proto3";`。
2. **package 与 go_package**：
   - `package` 表达逻辑命名空间；
   - `option go_package` 必须配置为可编译导入路径（建议与仓库 module path 对齐）。
3. **命名风格（推荐）**：
   - `message` / `service` / `rpc`：大驼峰。
   - 字段名：`lower_snake_case`。
   - `enum`：枚举类型名大驼峰；枚举值名建议 `UPPER_SNAKE_CASE`。
4. **默认值**：proto3 不支持自定义字段默认值（除语言默认值外），避免依赖“未传值==默认值”的业务语义。

### 1.2 兼容性演进（通用）

1. **字段号（Field Number）**：
   - 一旦发布后不可复用。
   - 删除字段必须使用 `reserved` 保留字段号与字段名。
2. **修改字段类型/语义**：原则上禁止（尤其是标量类型与 repeated/map/oneof 之间的切换）。
3. **新增字段**：只能新增新的字段号，且确保老客户端可忽略。
4. **enum**：必须保留历史值；新增只能追加；不要复用历史值。

---

## 2. 字节体系规范（BAM / Kitex / 网关）

### 2.1 包结构与工程组织

1. `package` 建议与 PSM/领域保持稳定映射（例如 `suite.search.example`）。
2. `option go_package`：
   - 必须明确；
   - 建议形如 `"code.byted.org/<repo>/<module>/idl/gen/suite/search/example;example"`（实际以仓库约定为准）。
3. 文件拆分：
   - service 定义建议集中在主文件；
   - 通用 message/enum 可拆到 `*.types.proto`、`common.proto` 等。

### 2.2 Service 与 RPC 方法

1. **方法形态**：
   - **强制**：每个 `rpc` 只接受一个 request message、返回一个 response message。
   - 不建议使用 streaming（除非明确有需求并评估所有网关/SDK/监控链路支持）。
2. **命名**：与业务语义一致，遵循 `Create/Delete/Update/Get/MGet/List` 约定（见业务规范）。
3. **错误模型**：
   - transport error（超时/网络/鉴权失败等）走 RPC 框架错误；
   - 业务错误需要在 response 中有稳定承载（建议统一 `BaseResp`/`Status` message）。

> 不同语言的 error 语义差异较大，统一用 response 内的结构化字段承载业务错误通常更稳定。

### 2.3 HTTP 场景（转码/网关）

Protobuf 描述 HTTP 的开源主流方式是 `google.api.http` 注解（gRPC Transcoding）。若业务使用字节内置网关/框架，也建议优先保持与该模式可映射。

#### 2.3.1 路由注解（建议）

```proto
import "google/api/annotations.proto";

service ExampleHttpService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse) {
    option (google.api.http) = {
      get: "/v1/user/{user_id}"
    };
  }

  rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse) {
    option (google.api.http) = {
      post: "/v1/user/update"
      body: "*"
    };
  }
}
```

关键约束：

1. `get` 路由不要设置 `body`。
2. path 参数必须在 request message 中存在同名字段（如 `{user_id}` 需要有 `user_id` 字段）。
3. query 参数默认从 request message 的其余字段映射；避免在 query 中放置复杂嵌套结构。
4. 若需要 form/multipart/file upload：
   - 尽量避免以“通用网关转码”实现；
   - 需要时应明确 body 语义（例如使用 bytes/string 承载原始内容），并在文档中给出示例报文。

---

## 3. 业务规范（跨业务通用最佳实践）

### 3.1 方法命名与语义

- CRUD：`CreateXxx`、`DeleteXxx`、`UpdateXxx`、`GetXxx`。
- 批量获取：`MGetXxx`（入参 `repeated`，返回 `map` 或 `repeated`，并对 batch size 做限制）。
- 列表查询：`ListXxx`，分页优先使用 token/cursor 模式。

### 3.2 字段设计

1. **强制**：禁止复用字段号；删除字段必须 `reserved`。
2. **建议**：尽量让字段语义“可选”，避免依赖语言默认值表示业务含义；必要时使用 proto3 `optional` 或 wrapper 类型表达“是否存在”。
3. **建议**：避免 `map<string, string>` 作为核心业务结构；动态结构建议显式 message 或使用 `google.protobuf.Struct`/`Any` 并配套文档。

### 3.3 ID、时间与前端精度

1. ID 通常用 `int64`，但对外/前端（JS）场景建议：
   - 要么定义为 `string`；
   - 要么明确在网关/SDK 层做 string 化转换并写入规范。
2. 时间建议统一：
   - 方案 A：使用 `int64` 表示毫秒（ms），命名以 `_time` 结尾；
   - 方案 B：使用 `google.protobuf.Timestamp`（更标准，但跨语言/跨网关需要确认兼容）。

> 建议在同一业务域内固定一种时间表达方式，避免“部分接口用 Timestamp，部分接口用 int64”。

### 3.4 Enum

1. **强制**：每个 enum 必须有 0 值默认项，建议命名为 `*_UNSPECIFIED = 0` 或 `UNSPECIFIED = 0`。
2. **强制**：不要修改历史 enum 值；新增只能追加。

---

## 4. 兼容性与版本演进（Protobuf 要点）

### 4.1 字段删除/重命名

- 删除字段：必须 `reserved` 字段号与字段名。

```proto
message Example {
  reserved 2, 15;
  reserved "old_field", "deprecated_field";

  string new_field = 1;
}
```

- 重命名字段：建议保留旧字段并新增新字段号，通过服务端兼容一段时间后再删除旧字段（并 `reserved`）。

### 4.2 oneof

- `oneof` 适合表达互斥字段，但演进成本高。
- 若使用：
  - 不要删除 oneof 的某个字段后复用其字段号；
  - 不要在已发布的 oneof 内随意变更字段语义。

---

## 5. 可自动化检查清单（为后续 IDL 检测准备）

- 语法：缺少 `syntax = "proto3"`（Error）；字段号重复（Error）；import 不存在（Error）。
- 兼容性：出现未 reserved 的字段删除（Warn/Policy）；enum 缺少 0 值（Warn→可业务强制）。
- RPC：rpc 入参/出参不是 message（Error）；response 缺少统一错误承载（Warn/Policy）。
- HTTP：`google.api.http` 的 `get` 配置了 `body`（Error）；path 模板字段在 request 中缺失（Error）；query 中出现复杂嵌套结构（Warn）。
