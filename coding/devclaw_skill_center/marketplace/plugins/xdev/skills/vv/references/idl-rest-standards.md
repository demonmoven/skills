# CozeLoop IDL & REST API 审查要点

---

## 1. IDL 文件与命名

- **Namespace**: `namespace go` 的值遵循 `{Product}.{Subsys}.{Module}` 格式
- **文件名**: 必须是 `lower_snake_case` 格式，如 `coze.loop.prompt.manage.thrift`
- **单一 Service**: 一个 `.thrift` 文件**只应定义一个 Service**（使用 `extends` 聚合除外）

---

## 2. 定义约束

- **命名**: `Service`, `Method`, `Struct`, `Enum` 使用 `PascalCase`
- **Struct 字段**: 字段名**必须**使用 `snake_case`
- **Method 签名**: 方法**只能有一个** request 和**一个** response 结构体
- **公共响应体**: 所有 `Response` **必须**包含 `base.BaseResp` 字段；所有 `Request` **建议**包含 `optional base.Base` 字段

---

## 3. 兼容性与演进

- **严禁修改**已存在的字段 ID 和类型
- 新增字段**必须**定义为 `optional`
- **严禁**将已有字段从 `optional` 修改为 `required`
- 枚举**零值不应被使用**，通常定义为 `UNSPECIFIED = 0`

---

## 4. REST API 设计

### 资源路径
- 路径中的资源名**必须使用复数**形式，例如 `/api/v1/prompts`
- 如果使用了单数（如 `/api/v1/prompt`），功能正常时视为**轻微**或**提示**问题

### 标准方法

| 操作 | HTTP 方法 | 路径 |
|------|-----------|------|
| Create | `POST` | `/resources` |
| Get | `GET` | `/resources/{id}` |
| Update (Partial) | `PATCH` | `/resources/{id}` |
| Update (Full) | `PUT` | `/resources/{id}` |
| Delete | `DELETE` | `/resources/{id}` |
| List | `POST` | `/resources/list` |

### 分页
- 列表查询**优先使用游标分页**（cursor-based pagination）

---

## 5. 注解规范

### Hertz 注解
- 正确使用 `api.*` 注解绑定 HTTP 请求各部分（`api.path`, `api.query`, `api.body`）

### 校验器注解
- 使用 `vt.*` 注解（kitex validator）声明字段校验规则

### JS i64 兼容
- 所有 `i64` 类型字段**必须**同时添加：
  - `api.js_conv='true'`
  - `go.tag='json:"field_name,string"'`
