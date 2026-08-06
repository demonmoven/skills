# Thrift IDL 规范（HTTP / RPC）

本文档用于描述在字节体系内使用 Thrift 进行接口定义时的统一规范，覆盖 **RPC（Kitex）** 与 **HTTP（Hertz/网关协议转换）** 两类场景。

## 0. 适用范围与术语

- **IDL**：接口定义语言文件（`.thrift`）。
- **PSM**：服务唯一标识（通常形如 `product.subsys.module`）。
- **RPC**：以 Kitex/Thrift 为主的二进制 RPC 调用。
- **HTTP**：通过 Thrift IDL + 注解描述 HTTP 路由与参数映射（ByteAPI 注解体系）。
- **规范分层**：
  - **开源规范**：Apache Thrift 语法与兼容性通用原则。
  - **字节体系规范**：在字节基础设施（BAM/ByteAPI/Kitex/Hertz）中可解析、可生成代码、可稳定演进的约束。
  - **业务规范**：跨业务的可维护性、可读性与长期演进最佳实践。

> 建议在后续 IDL 检测中按层级输出问题来源：`开源` / `字节` / `业务`，并区分严重程度：`强制(阻断)` / `建议(提醒)`。

---

## 1. 开源规范（Apache Thrift 基线）

### 1.1 文件结构与语法

1. **文件基础结构建议顺序**：`namespace` → `include`/`cpp_include` → `typedef/const/enum` → `struct/union/exception` → `service`。
2. **注释**：统一使用 `//` 单行注释，避免使用 `#`、`/* ... */`（不同工具链解析存在差异）。
3. **字段分隔符**：Thrift 允许使用 `,` 或 `;` 分隔字段/枚举项；**建议统一使用 `;`**（便于一致性检查与格式化）。
4. **字段编号（Field ID）**：
   - 必须是正整数。
   - 同一 `struct/exception` 内必须唯一。
   - 兼容性演进中字段编号一旦发布后不得复用（见兼容性章节）。

### 1.2 兼容性与演进（通用）

1. **新增字段**：推荐新增为 `optional`，并选择新的字段编号。
2. **删除字段**：不建议直接删除（避免编号复用风险）；可通过注释、标记废弃等方式保留。
3. **变更字段类型**：属于不兼容变更，原则上禁止（除非明确评估所有上下游并按灰度策略升级）。
4. **枚举**：建议为枚举预留默认值（常见为 `0`），并避免重排或复用历史值。

---

## 2. 字节体系规范（BAM / ByteAPI / Kitex / Hertz）

### 2.1 命名与工程组织（PSM / 文件名 / namespace）

#### 2.1.1 namespace

- **Go**：使用 PSM 形态的 package（示例：`namespace go suite.search.example`）。
- **Java**：使用 `com.bytedance.{Product}.{Subsys}.{Module}`（示例：`namespace java com.bytedance.suite.search.example`）。

#### 2.1.2 文件命名

- 主文件（包含 service 定义的文件）推荐与 PSM 对齐，示例：
  - 当按 `Product/Subsys/` 组织目录时，文件可简化为 `{Module}.thrift`。
  - 需要拆分时，使用同前缀：`{Module}.{SubModule}.thrift`。

#### 2.1.3 include

- include 路径必须可解析；不要保留未使用 include。
- 建议通过 **主文件集中 service 定义**、子文件承载通用 struct/enum 的方式拆分。

### 2.2 Service 规范（通用）

1. **Service 命名**：使用大驼峰，通常为 `{Module}Service`。
2. **Service 数量**：一个 `.thrift` 文件建议只有一个 service；如确需组合，使用 `extends`，但需确认客户端/生成工具链是否支持。
3. **方法签名**：
   - **强制**：每个方法只允许 **一个参数** 与 **一个返回值**。
   - 参数与返回值类型必须是自定义 `struct`，不要直接使用基础类型作为参数/返回值。
   - 参数变量名统一为 `request` 或 `req`。

### 2.3 Request / Response 规范（字节通用基线）

1. **命名**：`{Method}Request` / `{Method}Response`。
2. **Base 字段（推荐）**：每个 Request 建议包含 `base.Base`，字段编号 **255**，`optional`。
3. **BaseResp 字段（强制）**：每个 Response 必须包含 `base.BaseResp`，字段编号 **255**，`optional`。
4. **业务错误模型**：
   - 业务错误通过 `BaseResp` 传递，RPC transport error（如超时/网络/序列化）通过 RPC 框架错误处理。
   - 不建议再定义大量 `exception` 来表达业务错误（兼容性与多语言成本高）。

示例：

```thrift
namespace go suite.search.example
namespace java com.bytedance.suite.search.example

include "../../base.thrift"

struct GetUserRequest {
  1: optional i64 UserID;
  255: optional base.Base Base;
}

struct GetUserResponse {
  1: optional string UserName;
  255: optional base.BaseResp BaseResp;
}

service ExampleService {
  GetUserResponse GetUser(1: GetUserRequest request);
}
```

### 2.4 字节 HTTP（ByteAPI 注解）规范（详细版）

> 适用于 Hertz/网关协议转换等以 HTTP 为入口的服务：通过 **method 注解** 描述路由与行为，通过 **field 注解** 描述参数来源与序列化。

#### 2.4.0 基础约定（强制）

1. **注解大小写敏感**：统一使用小写（如 `api.get`、`api.header`），不要写成 `api.GET`/`api.Header`。
2. **注解表达式形式**：通常为 `api.{key} = {value}`。
   - 注解 value 在 Thrift 中本质是字符串，推荐使用双引号（兼容性更好，必要时做转义）。
3. **标识符字符集约束**：IDL 中变量名仅允许使用 `A-Za-z0-9_`，禁止出现特殊符号（如 `~!@#$%^&*()+='` 等）。

#### 2.4.1 Method：路由与行为注解

**强制：每个 URI 对应一个 method；method 上必须存在路由注解，且不可为空。**

支持的 HTTP 动词注解：

- `api.get` / `api.post` / `api.put` / `api.delete` / `api.patch`

路由语法：

- 与 gin/httprouter 风格兼容；path 变量使用 `:` 标记（示例：`/api/v:version/user/:id/profile`）。
- 一个 method 支持绑定多个 URI：用英文逗号 `,` 分隔（示例：`api.get = "/path1,/path2"`）。

> 备注：`api.patch` 在部分网关/链路中支持不一致，若业务依赖 patch 需确认目标网关与 SDK 支持情况。

示例：

```thrift
service ExampleHttpService {
  GetUserResponse GetUser(1: GetUserRequest req) (api.get = "/v1/user/:id");
  UpdateUserResponse UpdateUser(1: UpdateUserRequest req) (api.post = "/v1/user/update");
}
```

#### 2.4.2 Request：参数位置注解

字段可通过 `api.{position} = "name"` 映射到 HTTP：

- `api.query`：URL query
- `api.path`：path 参数
- `api.header`：header
- `api.cookie`：cookie
- `api.body`：JSON body 的一级 key（或默认 body）
- `api.raw_body`：原始 body（少数需要原始字节的场景）

关键约束（强制/建议）：

1. **默认映射规则**：未标注时，GET 通常映射 query，POST 通常映射 body，key 为字段名。
2. **GET 请求**：通常不支持 `api.body`（或表现不一致）；优先使用 `api.query` / `api.path` / `api.header` / `api.cookie`。
3. `api.query` / `api.header`：仅支持 **基本类型**（`string/i32/i64/bool/double` 等）与 **基础类型 list**。
   - list 常见序列化约定为逗号分隔（如 `a=1,2,3`），但不同框架可能存在差异，需要以目标框架为准。
4. `api.path`：只支持基本类型。
5. `form` 序列化（`application/x-www-form-urlencoded`）：request 字段不应包含复杂嵌套结构，否则字段可能无效。

#### 2.4.3 Method：序列化方式（Request/Response body）

可通过 method 注解标明 body 的序列化方式：

- `api.serializer`：Request body 序列化方式
- `api.resp_serializer`：Response body 序列化方式

支持的枚举值（仅允许这些写法）：

- `json` → `application/json`
- `form` → `application/x-www-form-urlencoded`
- `pb` → `application/x-protobuf`
- `muti-form` → `multipart/form-data`

规则：

1. 只允许使用上述枚举值（例如写成 `proto` 会无法识别）。
2. 为空时默认 `json`。
3. GET 请求上 `api.serializer` 无效。
4. 若存在多种可能序列化方式，可用 `,` 分割多个值（仅用于描述，不代表所有工具链都支持多值）。

#### 2.4.4 Response：body/header/status code 注解

Response 默认行为：

- 默认将 response struct JSON 序列化到 body，key 为字段名（注解可为空）。

支持的常用注解：

- `api.body`：设置 response 中某字段作为 body 的一级 key（或在 body 中展开多个字段）
- `api.header`：将字段写入 HTTP response header
- `api.http_code`：将字段作为 HTTP status code
- `api.none`：忽略字段（不参与映射/展示）
- `api.js_conv`：将 i64 表示为 string（见 2.4.5）

补充约束：

1. header 不支持复杂类型（除“逗号分隔 + value 为基本类型的 list”以外）。
2. 若业务未显式指定 http code：常见约定是 `BaseResp.StatusCode == 0 → 200`，否则 `500`；如需更丰富状态码策略，应在业务中间件明确落地。

#### 2.4.5 go.tag / api.js_conv / api.ref / api.vd（工具链相关注解）

- `go.tag`：用于 Go 侧字段 tag 重定义（常用于 JSON key、string tag 等）。
  - 注意：部分工具链仅支持有限语法（如字段名/类型/omitempty）。
- `api.js_conv`：兼容 JS int64 精度问题，将 int64 表示为 string。
  - 注解 value 通常为 `"true"` 才生效。
  - **只能应用在 i64 类型字段**，不能用于 `list<i64>` / `set<i64>`，否则可能解析失败或不生效。
- `api.ref`：用于 string 字段标识其实际承载的 JSON 结构（例如 string 内是一个 JSON），便于文档展开展示。
- `api.vd`：用于参数校验表达式（validator 语法由 go-tagexpr validator 定义）。
  - 仅用于入参约束，不要混入业务逻辑。

#### 2.4.6 通用参数（TTNet 公参 / 业务公参 / AGW Loader 参数）

> 该部分仅适用于业务确实接入了对应链路能力（TTNet 注入、AGW loader 等）。

1. **TTNetCommonParam**
   - 部分链路支持在 request 中携带 TTNet 公参结构。
   - 公参字段通常只支持基础数据类型；复杂结构可能无法被统一注入。
2. **业务公参（BizCommonParam 等）**
   - 业务公参 struct 建议以 `CommonParam` 作为后缀（例如 `BizCommonParam`）。
   - 公参字段通常仅允许使用 `api.query` / `api.header`（部分链路也支持 `api.cookie`），并且字段类型仅允许基础类型（`string/int/double/bool`）。
   - 业务公参常被视为“虚拟结构”，不对应 body 中某个实际 key；使用时应以目标框架/网关对公参的注入与解析行为为准。
3. **AGW Loader 通用参数（AgwCommonParam）**
   - 仅在接入 AGW 的场景使用；类型与数据来源通常由 AGW loader 提供。

#### 2.4.7 Service 级注解（AGW 相关）

- `agw.to_snake = "true"`：将字段名转换为 snake 风格（提供给端上接口）。
  - 若使用“多 service 合并”模式，建议在所有 service 上填写相同注解，避免合并时因 map 取值不稳定导致注解不稳定。
- `agw.preserve_base = "true"`：保留 BaseResp 的展示/传递（可写在 method 或 service 粒度）。

#### 2.4.8 Method 元信息注解（可选但推荐）

以下注解用于描述接口元信息、客户端生成与路由选路策略，常用于平台侧展示/代码生成：

- `api.param`：客户端请求是否需要携带公共参数，取值通常为 `"true"` / `"false"`。
- `api.baseurl`：TTNet 选路使用的 baseurl（如 `ib.snssdk.com`）。
- `api.gen_path`：客户端生成代码使用的 path（用于固化 `:version` 等变量）；优先级高于 `api.version`。
- `api.version`：替换 path 中的 `:version` 变量；优先级低于 `api.gen_path`。
- `api.tag`：客户端 RPC 增加标签，支持多个逗号分隔（部分端上/平台能力可能有限）。
- `api.api_level`：接口等级，取值范围通常为 `0~4`。
- `api.category`：接口分类目录，支持用 `/` 分隔多级目录（语义上是“一条目录路径”，不是多个分类）。
- `api.data_policy`：数据原则/安全策略等元信息，支持多个枚举值逗号分隔。

---

## 3. 业务规范（跨业务通用最佳实践）

> 本节在「字节体系规范」基线之上，补充更偏工程落地与长期可维护性的业务侧约束。其目标是让 IDL 更易读、更易演进，并降低跨端/跨语言/跨团队协作成本。

### 3.1 文件拆分与组织

1. **推荐**：按职责适当拆分 IDL 文件，例如：`service.thrift`（仅承载 service/method）、`*.types.thrift`（struct/enum）、`common.thrift`（通用结构）。
2. **强制**：拆分后的文件前缀保持一致，避免目录中出现难以追踪的“散装定义”。
3. **强制**：主文件中 service 定义要能通过 include 递归解析到所依赖的所有 struct/enum。

### 3.2 方法命名与语义

1. **强制**：CRUD 标准命名：`CreateXxx`、`DeleteXxx`、`UpdateXxx`、`GetXxx`。
2. **强制**：根据 ID 批量获取统一使用 `MGetXxx`。
   - **建议**：入参使用 `list`，返回使用 `map`；并对 batch size 设置上限（避免放大效应）。
3. **强制**：复杂条件列表查询（分页/筛选/排序）统一使用 `ListXxx`。
   - **建议**：分页优先使用 token/cursor 而不是 offset。
4. **强制**：方法入参/出参的 struct 命名遵循 `{Method}Request` / `{Method}Response`。

### 3.3 时间、ID 与数值

1. **强制**：时间统一使用 `i64`，精度为毫秒（ms）。
2. **强制**：时间字段命名以 `Time` 结尾，推荐：`{Verb|Noun}{Time}` 或 `{VerbPast}{Time}`（动词过去式表达历史时间戳），如 `CreateTime`、`UpdateTime`、`ExpiredTime`。
3. **建议**：ID 类字段优先使用 `i64`；对外（或前端/JS）场景需明确精度策略（例如 `api.js_conv` 或直接定义为 `string`）。

### 3.4 Enum

1. **强制**：枚举项分隔符统一使用且仅使用 `;`（避免风格混用影响格式化与解析差异）。
2. **强制**：枚举必须包含默认值，统一为 `Undefined = 0`（若历史包袱导致命名不同，至少确保“0 值兜底”存在且语义为未定义）。
3. **强制**：不要删除/复用历史枚举值；新增枚举值只能追加。

### 3.5 结构体字段设计

1. **强制**：禁止设置 default value（避免 client/server 对“未传值”的语义不一致）。
2. **强制**：不要使用 `set`（在 Go 等语言生成结果不符合预期、且存在性能/语义问题）；建议用 `map<T, bool>` 或 `list<T>` + 去重。
3. **强制**：字段/类型命名中，缩略词与专有名词应全大写（如 `ID`、`URL`）。
4. **强制**：struct 字段名禁止以 `New` 作为前缀，禁止以 `Result` / `Args` 作为后缀（避免与工具链/生成代码隐式结构冲突）。
5. **建议**：列表字段使用复数命名；map 字段以 `Map` 结尾。
6. **强制**：req/resp 信息传递禁止使用 `map<string, string>` / `map<string, any>` 这类不清晰结构作为核心业务字段；应使用 `struct` 显式定义内部结构。
   - **例外**：当服务端对结构完全不理解、仅做透传时，才可用 `string` 承载 JSON 或使用 map，并必须在文档/注释中说明其序列化格式。

### 3.6 注释与可读性

1. **强制**：注释统一使用 `//`，不使用 `#` 或 `/* ... */`；`//` 后需留空格。
2. **强制**：struct/method 的注释放在定义上一行；字段/枚举项注释优先放在行尾，过长可放上一行。
3. **建议**：避免中式英语或易引起误解的注释；对外接口优先英文。

### 3.7 状态码与错误码声明

1. **强制**：在 IDL 中声明业务状态码/错误码的定义（enum 或 const），并保证外部接入方可从 IDL 中直接理解错误码语义。
2. **建议**：错误码与错误信息在 Response 的 `BaseResp`（或业务统一错误结构）中稳定承载；避免把业务错误混入 transport error。

### 3.8 参数校验（推荐）

1. **推荐**：对关键入参添加可机器执行的校验约束（如 ByteAPI 的 `api.vd`，或通过 `go.tag` 让后端框架接入校验器）。
2. **建议**：校验表达式仅描述入参合法性，不要混入业务逻辑。

---

## 4. 兼容性与版本演进（字节落地要点）

### 4.1 字段

- **禁止** 修改已发布字段的编号与类型。
- 新增字段：
  - 建议选择新的、未使用过的编号。
  - 推荐 `optional`。
- 废弃字段：
  - 建议注释保留或加 `Obsolete` 前缀（不要删除后复用编号）。

### 4.2 方法

- **禁止** 变更方法名（会影响客户端生成与路由/handler 对齐）。
- 新增方法：确保同步更新路由/handler 与生成代码。

---

## 5. 可自动化检查清单（为后续 IDL 检测准备）

以下规则建议可直接落地为 lint/静态检查（括号内给出建议严重度）：

- 通用：字段编号重复（Error）；include 不存在（Error）；未被任何 service 引用的孤立 struct（Warn）。
- RPC：方法入参/出参非 struct（Error）；Response 缺少 `BaseResp(255)`（Error）。
- HTTP：method 缺少路由注解（Error）；GET 方法的 request 出现 `api.body`（Error）；`api.*` key 大小写不规范（Error）；`api.serializer`/`api.resp_serializer` 值不在枚举集合内（Error）；`go.tag` 与 `api.body` key 冲突（Warn）；`api.js_conv` 用在非 i64 字段（Error）。
- 业务：enum 缺少 `Undefined = 0`（Warn→可视业务强制）；enum 分隔符未统一为 `;`（Warn）；出现 `set`（Warn/Policy）；出现字段 default value（Warn/Policy）；req/resp 中出现 `map<string, string>` 作为核心字段（Warn/Policy）；字段名出现 `New*` 或 `*Result/*Args`（Warn）。
