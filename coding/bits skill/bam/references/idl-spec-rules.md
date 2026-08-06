# IDL 规范检测规则速查表

> 本文档提供所有规范检测规则的快速参考，按规范层和 IDL 类型分类。

## 规则标识说明

规则 ID 格式：`{规范层}-{IDL类型}{序号}`

- `OS` = Open Source (开源规范)
- `BD` = ByteDance (字节体系规范)
- `BZ` = Business (业务规范)
- `T` = Thrift
- `P` = Protobuf

---

## Layer 1: 开源规范

### Thrift 规则

| 规则 ID | 检测内容 | 级别 | 自动修复 | 检测方法 |
|---------|---------|------|---------|---------|
| OS-T001 | 字段编号重复 | Error | ❌ | 同一 struct 内字段编号唯一性检查 |
| OS-T002 | 字段编号非正整数 | Error | ❌ | 字段编号值范围检查 |
| OS-T003 | Include 文件不存在 | Error | ❌ | 文件系统检查 |
| OS-T004 | 类型引用不存在 | Error | ❌ | struct/enum/typedef 定义查找 |
| OS-T005 | 方法参数数量错误 | Error | ❌ | 正则匹配方法签名 |
| OS-T006 | 方法参数类型错误 | Error | ❌ | 检查参数是否为 struct |
| OS-T007 | 方法返回类型错误 | Error | ❌ | 检查返回值是否为 struct |
| OS-T008 | Enum 值重复 | Error | ❌ | 同一 enum 内值唯一性检查 |

#### OS-T001: 字段编号重复

**描述**: 同一 struct/exception 内字段编号必须唯一。

**示例 (错误)**:
```thrift
struct User {
  1: optional string Name;
  1: optional i64 ID;    // ❌ 字段编号 1 重复
}
```

**修复**:
```thrift
struct User {
  1: optional string Name;
  2: optional i64 ID;    // ✅ 使用唯一编号
}
```

#### OS-T003: Include 文件不存在

**描述**: include 引用的文件必须存在。

**示例 (错误)**:
```thrift
include "non_existent.thrift"  // ❌ 文件不存在
```

**修复**: 确保 include 路径正确，或删除未使用的 include。

#### OS-T005: 方法参数数量错误

**描述**: 每个 service 方法只能有一个参数。

**示例 (错误)**:
```thrift
service UserService {
  GetUserResponse GetUser(1: GetUserRequest req, 2: string extra);  // ❌ 两个参数
}
```

**修复**:
```thrift
struct GetUserRequest {
  1: optional i64 UserID;
  2: optional string Extra;
}

service UserService {
  GetUserResponse GetUser(1: GetUserRequest req);  // ✅ 单参数
}
```

### Protobuf 规则

| 规则 ID | 检测内容 | 级别 | 自动修复 | 检测方法 |
|---------|---------|------|---------|---------|
| OS-P001 | syntax 声明缺失 | Error | ✅ | 检查文件开头 |
| OS-P002 | 字段编号重复 | Error | ❌ | 同一 message 内字段编号唯一性 |
| OS-P003 | Import 文件不存在 | Error | ❌ | 文件系统检查 |
| OS-P004 | 字段编号范围错误 | Error | ❌ | 值范围 1-536870911 |
| OS-P005 | 使用 reserved 字段号 | Error | ❌ | 与 reserved 对比 |
| OS-P006 | RPC 参数/返回类型错误 | Error | ❌ | 检查是否为 message |
| OS-P007 | go_package 缺失 | Error | ✅ | 检查 option |
| OS-P008 | Enum 零值缺失 | Warning | ✅ | 检查是否有 0 值 |

#### OS-P001: syntax 声明缺失

**描述**: 必须声明 `syntax = "proto3"`。

**示例 (错误)**:
```protobuf
package example;

message User {
  string name = 1;
}
```

**修复**:
```protobuf
syntax = "proto3";

package example;

message User {
  string name = 1;
}
```

#### OS-P008: Enum 零值缺失

**描述**: 枚举应包含零值默认项 (proto3 规范)。

**示例 (警告)**:
```protobuf
enum Status {
  ACTIVE = 1;    // ⚠️ 缺少零值
  INACTIVE = 2;
}
```

**修复**:
```protobuf
enum Status {
  STATUS_UNSPECIFIED = 0;  // ✅ 添加零值默认项
  ACTIVE = 1;
  INACTIVE = 2;
}
```

---

## Layer 2: 字节体系规范

### Thrift 规则

| 规则 ID | 检测内容 | 级别 | 自动修复 | 适用场景 |
|---------|---------|------|---------|---------|
| BD-T001 | Response 缺少 BaseResp | Error | ✅ | 全部 |
| BD-T002 | BaseResp 字段编号非 255 | Error | ✅ | 全部 |
| BD-T003 | Request 缺少 Base | Warning | ✅ | 全部 |
| BD-T004 | Base 字段编号非 255 | Warning | ✅ | 全部 |
| BD-T005 | HTTP 注解大小写错误 | Error | ✅ | HTTP |
| BD-T006 | HTTP Method 缺少路由注解 | Error | ❌ | HTTP |
| BD-T007 | GET 请求使用 api.body | Error | ❌ | HTTP |
| BD-T008 | api.serializer 值非法 | Error | ✅ | HTTP |
| BD-T009 | api.path 字段类型非法 | Error | ❌ | HTTP |
| BD-T010 | api.query/list 类型非法 | Warning | ❌ | HTTP |
| BD-T011 | api.js_conv 用于非 i64 | Error | ✅ | HTTP |
| BD-T012 | go.tag 与 api.body key 冲突 | Warning | ❌ | HTTP |
| BD-T013 | namespace go 格式不规范 | Warning | ✅ | 全部 |
| BD-T014 | include 未使用 | Warning | ✅ | 全部 |

#### BD-T001: Response 缺少 BaseResp

**描述**: 每个 Response 必须包含 `base.BaseResp` 字段，字段编号 255。

**示例 (错误)**:
```thrift
struct GetUserResponse {
  1: optional string UserName;
  // ❌ 缺少 BaseResp
}
```

**自动修复**:
```thrift
struct GetUserResponse {
  1: optional string UserName;
  255: optional base.BaseResp BaseResp;  // ✅ 自动添加
}
```

#### BD-T005: HTTP 注解大小写错误

**描述**: 注解必须使用小写。

**示例 (错误)**:
```thrift
service UserService {
  GetUserResponse GetUser(1: GetUserRequest req) (api.GET = "/v1/user/:id");  // ❌ GET 应为小写
}
```

**自动修复**:
```thrift
service UserService {
  GetUserResponse GetUser(1: GetUserRequest req) (api.get = "/v1/user/:id");  // ✅
}
```

#### BD-T006: HTTP Method 缺少路由注解

**描述**: HTTP 服务的每个方法必须有路由注解。

**示例 (错误)**:
```thrift
service UserService {
  GetUserResponse GetUser(1: GetUserRequest req);  // ❌ 缺少路由注解
}
```

**修复**:
```thrift
service UserService {
  GetUserResponse GetUser(1: GetUserRequest req) (api.get = "/v1/user/:id");  // ✅
}
```

#### BD-T007: GET 请求使用 api.body

**描述**: GET 请求不应使用 api.body 注解。

**示例 (错误)**:
```thrift
struct GetUserRequest {
  1: optional string Filter;  // ❌ GET 请求不应使用 api.body
}

service UserService {
  GetUserResponse GetUser(1: GetUserRequest req) (api.get = "/v1/user");
}
```

**修复**:
```thrift
struct GetUserRequest {
  1: optional string Filter (api.query = "filter");  // ✅ 使用 api.query
}
```

#### BD-T008: api.serializer 值非法

**描述**: api.serializer 只能是 `json`, `form`, `pb`, `muti-form` 之一。

**示例 (错误)**:
```thrift
service UserService {
  UpdateUserResponse UpdateUser(1: UpdateUserRequest req) (api.post = "/v1/user", api.serializer = "proto");  // ❌ 应为 "pb"
}
```

**自动修复**:
```thrift
service UserService {
  UpdateUserResponse UpdateUser(1: UpdateUserRequest req) (api.post = "/v1/user", api.serializer = "pb");  // ✅
}
```

#### BD-T011: api.js_conv 用于非 i64 字段

**描述**: api.js_conv 只能用于 i64 类型字段。

**示例 (错误)**:
```thrift
struct User {
  1: optional string UserID (api.js_conv = "true");  // ❌ 非 i64 字段
}
```

**修复**:
```thrift
struct User {
  1: optional i64 UserID (api.js_conv = "true");  // ✅ i64 字段
}
```

### Protobuf 规则

| 规则 ID | 检测内容 | 级别 | 自动修复 | 适用场景 |
|---------|---------|------|---------|---------|
| BD-P001 | Response 缺少统一错误承载 | Warning | ❌ | RPC |
| BD-P002 | package 与 PSM 不对齐 | Warning | ✅ | 全部 |
| BD-P003 | google.api.http GET 配置 body | Error | ❌ | HTTP |
| BD-P004 | google.api.http path 字段缺失 | Error | ❌ | HTTP |
| BD-P005 | 建议避免 streaming RPC | Warning | ❌ | RPC |

#### BD-P003: google.api.http GET 配置 body

**描述**: GET 请求不应配置 body。

**示例 (错误)**:
```protobuf
rpc GetUser(GetUserRequest) returns (GetUserResponse) {
  option (google.api.http) = {
    get: "/v1/user/{user_id}"
    body: "*"  // ❌ GET 不应有 body
  };
}
```

**修复**:
```protobuf
rpc GetUser(GetUserRequest) returns (GetUserResponse) {
  option (google.api.http) = {
    get: "/v1/user/{user_id}"
  };
}
```

#### BD-P004: google.api.http path 字段缺失

**描述**: path 模板中的字段必须在 request message 中存在。

**示例 (错误)**:
```protobuf
message GetUserRequest {
  string name = 1;  // ❌ 缺少 user_id 字段
}

rpc GetUser(GetUserRequest) returns (GetUserResponse) {
  option (google.api.http) = {
    get: "/v1/user/{user_id}"
  };
}
```

**修复**:
```protobuf
message GetUserRequest {
  string name = 1;
  string user_id = 2;  // ✅ 添加 user_id 字段
}
```

---

## Layer 3: 业务规范

| 规则 ID | 检测内容 | 级别 | 自动修复 | 说明 |
|---------|---------|------|---------|------|
| BZ-001 | 方法命名不规范 | Warning | ❌ | CRUD 命名规范 |
| BZ-002 | Request/Response 命名不规范 | Warning | ❌ | {Method}Request/Response |
| BZ-003 | Enum 缺少 Undefined = 0 | Warning | ✅ | 默认值规范 |
| BZ-004 | Enum 分隔符不统一 | Warning | ✅ | 统一使用 `;` |
| BZ-005 | 使用 set 类型 | Warning | ❌ | 建议 map 或 list |
| BZ-006 | 字段设置 default value | Warning | ❌ | 不建议默认值 |
| BZ-007 | 核心字段使用 map<string, string> | Warning | ❌ | 应显式定义 |
| BZ-008 | 字段名以 New 开头 | Warning | ❌ | 禁止 |
| BZ-009 | 字段名以 Result/Args 结尾 | Warning | ❌ | 禁止 |
| BZ-010 | 时间字段命名不规范 | Warning | ❌ | 应以 Time 结尾 |
| BZ-011 | 时间类型不规范 | Warning | ❌ | 应使用 i64 (ms) |
| BZ-012 | 注释风格不统一 | Info | ✅ | 使用 `//` |
| BZ-013 | 孤立 struct/enum | Info | ❌ | 未被引用 |
| BZ-014 | ID 字段精度问题 | Info | ❌ | i64 前端需 js_conv |

#### BZ-001: 方法命名不规范

**描述**: 方法应遵循 CRUD 命名规范。

**规范命名**:
- `CreateXxx` - 创建资源
- `DeleteXxx` - 删除资源
- `UpdateXxx` - 更新资源
- `GetXxx` - 获取单个资源
- `MGetXxx` - 批量获取
- `ListXxx` - 列表查询

**示例 (警告)**:
```thrift
service UserService {
  GetUserResponse QueryUser(1: GetUserRequest req);  // ⚠️ 应为 GetUser
  GetUserResponse SearchUsers(1: ListUserRequest req);  // ⚠️ 应为 ListUser
}
```

#### BZ-003: Enum 缺少 Undefined = 0

**描述**: 枚举应包含默认值 `Undefined = 0`。

**示例 (警告)**:
```thrift
enum UserStatus {
  Active = 1;    // ⚠️ 缺少零值
  Inactive = 2;
}
```

**自动修复**:
```thrift
enum UserStatus {
  Undefined = 0;  // ✅ 自动添加
  Active = 1;
  Inactive = 2;
}
```

#### BZ-005: 使用 set 类型

**描述**: 不建议使用 set 类型，建议用 map 或 list 替代。

**示例 (警告)**:
```thrift
struct User {
  1: optional set<string> Tags;  // ⚠️ 不建议使用 set
}
```

**修复建议**:
```thrift
struct User {
  1: optional list<string> Tags;  // ✅ 使用 list
  // 或
  1: optional map<string, bool> TagMap;  // ✅ 使用 map
}
```

#### BZ-006: 字段设置 default value

**描述**: 不建议为字段设置默认值，避免 client/server 语义不一致。

**示例 (警告)**:
```thrift
struct User {
  1: optional string Name = "unknown";  // ⚠️ 不建议默认值
}
```

**修复**:
```thrift
struct User {
  1: optional string Name;  // ✅ 不设默认值
}
```

#### BZ-007: 核心字段使用 map<string, string>

**描述**: 核心业务字段应显式定义 struct，不应使用模糊的 map 结构。

**示例 (警告)**:
```thrift
struct GetUserRequest {
  1: optional map<string, string> Filters;  // ⚠️ 核心字段不应使用 map
}
```

**修复**:
```thrift
struct UserFilters {
  1: optional string Status;
  2: optional string Category;
  3: optional i64 CreateTimeStart;
  4: optional i64 CreateTimeEnd;
}

struct GetUserRequest {
  1: optional UserFilters Filters;  // ✅ 显式定义
}
```

---

## 兼容性检测规则

| 规则 ID | 检测内容 | 级别 | 说明 |
|---------|---------|------|------|
| CP-001 | 字段编号被复用 | Error | 已发布的编号不能复用 |
| CP-002 | 字段类型变更 | Error | 修改类型为不兼容变更 |
| CP-003 | required 字段新增 | Error | 影响老客户端 |
| CP-004 | 字段删除未保留编号 | Warning | 应 reserved |
| CP-005 | Enum 值删除 | Error | 可能解析失败 |
| CP-006 | Enum 值编号复用 | Error | 不兼容 |
| CP-007 | 方法删除 | Error | 客户端调用失败 |
| CP-008 | 方法签名变更 | Error | 不兼容 |
| CP-009 | 路由变更 | Warning | 影响存量调用方 |
| CP-010 | 新增 required 字段 | Warning | 影响老客户端 |

### 兼容性检测示例

#### CP-002: 字段类型变更

**场景**: 字段类型从 `i64` 变更为 `string`

**检测方法**: 对比 git diff 前后的字段类型

**输出**:
```
❌ CP-002 字段类型变更
   └─ GetUserRequest.UserID: i64 → string
   └─ 影响: 不兼容变更，存量客户端可能解析失败
   └─ 建议:
       方案 A: 新增字段 UserIDStr，保留 UserID
       方案 B: 版本化方法 GetUserV2
```

#### CP-004: 字段删除未保留编号

**场景**: 删除字段后未使用 reserved

**检测方法**: 分析 git diff 中的字段删除

**输出**:
```
⚠️ CP-004 字段删除未保留编号
   └─ UpdateUserRequest.OldField (字段 5 已删除)
   └─ 建议: 添加 reserved 5; reserved "OldField";
```

---

## 规则配置

可在项目根目录创建 `.idl-lint.yaml` 配置文件：

```yaml
# IDL 规范检测配置
version: 1

# 规则禁用
rules:
  disable:
    - BZ-001  # 禁用方法命名规范检查
    - BZ-013  # 禁用孤立 struct 检查

# 规则严重级别调整
severity:
  BD-T003: info     # Request 缺少 Base 降级为 info
  BZ-003: error     # Enum 缺少默认值升级为 error

# 兼容性检测配置
compatibility:
  # 基准版本 (用于对比)
  baseline: main
  # 是否检测未发布的变更
  check_uncommitted: true

# 自动修复配置
autofix:
  # 是否自动修复
  enabled: true
  # 排除的规则
  exclude:
    - BZ-001  # 不自动修复命名问题
```

---

## 检测输出格式

### 控制台输出

```
═══════════════════════════════════════════════════════════════
                    IDL 规范检测报告
═══════════════════════════════════════════════════════════════
项目: example.service.api_server | 类型: HTTP (Hertz) | IDL: Thrift

【开源规范 Layer 1】 ─────────────────────────────────────────
✅ OS-T001 字段编号重复检查        通过
❌ OS-T005 Service 方法参数数量
   └─ service.thrift:45 - GetUserV2 有 2 个参数

【字节体系规范 Layer 2】 ────────────────────────────────────
❌ BD-T001 Response 缺少 BaseResp
   └─ types.thrift:78 - GetUserResponse
   └─ 💡 自动修复: bytedcli bam idl fix --issue BD-T001

【业务规范 Layer 3】 ─────────────────────────────────────────
⚠️  BZ-001 方法命名不规范
   └─ service.thrift:38 - QueryUser

═══════════════════════════════════════════════════════════════
检测结果: 2 Errors, 1 Warning
阻断问题: 2 个 (必须修复)
═══════════════════════════════════════════════════════════════
```

### JSON 输出

```json
{
  "summary": {
    "total_errors": 2,
    "total_warnings": 1,
    "total_info": 0,
    "blocking_issues": 2
  },
  "issues": [
    {
      "rule_id": "OS-T005",
      "level": "error",
      "layer": "opensource",
      "file": "service.thrift",
      "line": 45,
      "message": "GetUserV2 有 2 个参数，应为 1 个",
      "fix_available": false
    },
    {
      "rule_id": "BD-T001",
      "level": "error",
      "layer": "bytedance",
      "file": "types.thrift",
      "line": 78,
      "message": "GetUserResponse 缺少 BaseResp 字段",
      "fix_available": true,
      "fix_command": "bytedcli bam idl fix --issue BD-T001 --file types.thrift:78"
    }
  ]
}
```
