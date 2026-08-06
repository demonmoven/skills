---
description: Initialize OpenSpec for a team repository by analyzing its architecture, standards, and functionalities.
argument-hint: none
---

<!-- OPENSPEC:START -->

# Prompt: Initialize OpenSpec for Team Repository

You are an expert software architect. Your goal is to initialize the **OpenSpec** documentation framework for this repository to ensure all future changes follow team standards and are well-documented.

## 1. Analysis Phase
First, explore the repository to understand its nature:
- **Tech Stack**: Identify primary languages (Go, Python, etc.), frameworks (KiteX, Gin, etc.), and infrastructure (MySQL, Redis, Thrift/IDL).
- **Project Structure**: Locate source code, configuration, and API definitions.
- **Standards**: Look for existing documentation or patterns that indicate team-specific conventions.

## 2. Initialization Steps
Create the following structure under the root directory:

### Step 0: Do Bash `openspec init`
- Pay attention to choices natively supported provider: `coco`
- If the user does not have openspec installed, terminate and prompt the user to install openspec.

### Step 1:[Important]`openspec/project.md` init
仔细分析代码仓库，遵循openspec规范补充openspec/project.md
并在Important Constraints中添加以下内容
```
### Design约束[重要]
#### 开发分支确定
- 如果用户没有指定分支，则用合适的名称创建新分支，并记录在design.md中
#### idl design
- 确定当前需求是否需要修改RPC接口，如有修改则[必须]将完整的idl变更在design.md中列出
- IDL设计必须要严格遵循[IDL规范](https://bytedance.larkoffice.com/wiki/wikcn83lmkgMrPzPejvIQGcX5Fb?from=from_parent_docx)
#### storage design
- mysql 变动[必须]将完整DDL变更在design.md中列出，db表名需要以`star_`开头，必须包含create_time，modify_time，deleted字段
- redis 变动需要包含对应key和value结构
- es 变动需要包含index信息，字段变更等
#### RPC client设计
- [必须]使用overpass client，禁止使用star_kitex_gen client

### 其他约束[重要]
- 禁止运行go build
- 禁止运行go test
- 必须根据skill进行编译和构建操作
```

### Step 2:[Important] 优化`openspec/project.md` Architecture Patterns
```
### Architecture Patterns
- 分层结构：
  - `handler/`：Kitex Server 实现（IDL 接口落地），负责入参校验、上下文组装与调用应用层。
  - `application/`：用例编排（面向场景/接口），聚合领域与基础设施能力。
  - `service/`：同 `application/`。
  - `business/`：可复用业务逻辑抽象，供service/application调用。
  - `domain/`：领域实体/仓库接口/策略（Repository/Policy/Strategy），尽量保持无基础设施依赖。
  - `infrastructure/`：外部系统适配与实现（RPC、Repo、MQ、Redis、LocalCache 等）。
    - `mysql/`：mysql数据访问。
      - `base_dal/`：**DAL层**，表名作为目录名称，每个目录下包含一个`dal.generated.go`文件，供所有上层调用
      - `db/`：**DAO层**，表名作为目录名称，每个目录下包含三个文件`dao.common.go`、`dao.generated.go`、`dao.go`，[仅base_dal可调用]
      - `model/`：表名作为文件名，包含mysql模型struct结构
      - `sql/`：表名作为文件名，mysql的DDL
    - `abase/`：abase数据访问。
      - `model/`：Abase数据模型struct结构
      - `dal/`：Abase数据存取的简单封装
    - `bytedoc/`：bytedoc数据访问。
      - `model/`：bytedoc数据模型struct结构
      - `dal/`：bytedoc数据存取的简单封装
    - `local_cache/`：本地缓存访问。
      - `model/`：本地缓存数据模型struct结构
      - `dal/`：本地缓存数据存取的简单封装
    - `repository/`：`domain/`中的interface实现，对infrastructure内逻辑的封装。对mysql的操作[必须]使用base_dal/
    - `metrics/`：监控打点能力。
    - `mq/`：消息中间件相关，主要是消息的发送。
    - `rpc/`：调用外部RPC的client封装，使用overpass依赖。
    - `tcc/`：配置读取能力。
    - `es/`：ES读写能力。
  - `script/`：定时脚本逻辑。
```
根据仓库内容和以上格式，产出适配本仓库的Architecture Patterns
### Step 3: [Important] `openspec/project.md` Task Generation & Guideline生成
```
## Task Generation & Guideline

为确保 AI 生成的 `tasks.md` 具备足够的工程深度，能够直接指导编码，所有任务拆解必须遵循以下框架、质量门禁和示例。

### 1. 任务拆解框架 (Task Decomposition Framework)

任何后端变更都应首先识别其**变更类型**，然后按照下述模板展开为具体的子任务清单。


#### **类型一：新增/修改 RPC (Thrift) 接口**
- **[IDL]** 设计接口：在`code.byted.org/ad/star_idl/idl/go_author/` 目录下创建/修改 `.thrift` 文件。严格遵循**[星图IDL规范](https://bytedance.larkoffice.com/wiki/wikcn83lmkgMrPzPejvIQGcX5Fb)**
- **[Handler]** 实现接口入口：在 `handler/handler.go` 中新增/修改 `AdStarGoAuthorServiceImpl` 的方法。职责仅限于：参数校验、日志打印、上下文构建、调用 `application` 层。
- **[Application]** 编排业务逻辑：在 `application/{sub_domain}/` 目录下实现核心用例。负责协调领域服务、聚合数据、处理业务流程。
- **[Domain]** 定义领域能力：若涉及新的业务实体或复杂规则，在 `domain/{sub_domain}_domain/` 中定义 `Entity`, `Repository` 接口, `Policy` 或 `Strategy`。
- **[Infrastructure]** 实现外部依赖：
  - 若调用新 RPC，在 `infrastructure/rpc/` 下实现其 client，以目标服务名为文件名（eg.ad_star_control.go）。
  - 若涉及 DB 操作，在 `infrastructure/repositor/` 下实现 `Repository` 接口，对mysql的操作[必须]使用base_dal
- **[工程化]** 完善非功能性需求：
  - **幂等**: 对于写操作，明确幂等键来源（如 `req.idempotent_key` 或业务唯一标识），并在 `application` 层处理。
  - **鉴权**: 在 `handler` 层或 `application` 层调用公共鉴权能力，校验调用方角色与权限。
  - **错误码**: 在 `domain` 或 `application` 层定义清晰的业务错误码，并在 `handler` 层返回。


#### **类型二：MySQL 表/索引变更**
- **[DB]** 设计并提交 DDL：在 `infrastructure/mysql/sql/` 目录下创建/修改 `{{表名}}.sql` 文件。表名以 `star_` 开头，必须包含 `create_time`, `modify_time`, `deleted` 字段。
- **[CONF]** 在**conf/database.yml**中将对应数据表注册到合适的库
- **[Model]** 生成/修改数据模型：在 `infrastructure/mysql/model/` 目录下创建/修改 `{{表名}}.go` 文件，**禁止使用make model**。
- **[DAO/DAL]** 生成/修改数据访问层：[必须]按照现有代码全套风格，在 `infrastructure/mysql/db/{{表名}}/` 目录生成DAO代码，在 `infrastructure/mysql/base_dal/{{表名}}/` 目录生成DAL代码,**禁止使用make db**。
- **[Repository]** 实现仓库接口：在 `infrastructure/repositor/{{表名}}/` 目录下，实现 `domain` 层定义的 `Repository` 接口，封装对 `DAL` 的调用。
- **[事务]** 划定事务边界：写操作的事务**必须**在 `application` 层通过 `transaction.Exec` 或类似方式包裹。**严禁**在 `handler` 或 `infrastructure` 层开启事务。
- **[数据迁移]** 若涉及存量数据处理，需额外提供数据迁移和回滚方案。


#### **类型三：Redis 结构与并发控制**
- **[设计]** 定义 Key-Value 结构：明确 Redis Key 的命名规范（如 `star:goauthor:{biz}:{field}:{id}`）、Value 的数据结构（String, Hash, ZSet等）以及序列化方式（JSON, Protobuf）。
- **[实现]** 封装 Redis 操作：在 `infrastructure/redis/` 目录下封装具体的读写逻辑，实现 `domain` 层定义的缓存接口。
- **[原子性]** 并发控制：对于计数、状态推进等场景，**必须**使用 `Lua` 脚本或 Redis 原子命令（如 `INCRBY`, `SETNX`）来保证操作的原子性。
- **[生命周期]** 策略制定：明确 Key 的过期时间（TTL）和清理策略，避免热点数据无限增长。


#### **类型四：ES 索引读写**
- **[设计]** 定义索引与查询口径：明确 ES index 名称、mapping 结构、以及核心查询字段（分页、排序、过滤等）。
- **[Client]** 封装客户端：在 `infrastructure/es/` 目录下封装对 ES client 的调用，提供结构化的查询和写入方法。
- **[Repository]** 实现仓库接口：实现 `domain` 层定义的搜索 `Repository`，供 `application` 层调用。

#### **类型五：观测与风控**
- **[日志]** 结构化日志：在关键业务节点（`handler` 入口、`application` 流程、`infrastructure` 调用前后）使用 `logs.CtxInfo/CtxError` 打印结构化日志。
- **[告警]** 预留占位：在 `tasks.md` 中为关键指标预留告警阈值配置项。

#### **类型六：单测/集成测试**
- **[覆盖面]** 明确测试重点：核心业务逻辑（`application`）、领域规则（`domain`）、复杂的数据处理（`infrastructure/repository`）**必须**有单测覆盖。
- **[断言]** 编写典型断言：覆盖成功场景、失败场景（如参数错误、依赖失败）、边界条件（如空列表、nil 指针）。
- **[设计模式]** 推荐表驱动测试：对于多 case 场景，使用表格法组织测试用例。
- **[复杂场景]** 验证并发与事务：对于涉及并发控制和数据库事务的逻辑，需设计专门的测试用例进行验证。

---

### 2. 质量门禁 (Quality Gates for Tasks)

在 AI 生成或人工编写 `tasks.md` 时，**必须**遵循以下质量门禁，否则视为不合格：

- **禁止宽泛描述**: 严禁出现如“实现XXX功能”、“修复XXX bug”这类没有明确范围和交付物的任务。
- **明确文件/函数落点**: 每个实现类的任务，都必须明确指出其主要代码落点，粒度至少到**文件路径**，最好到**函数或方法名**。
- **分层职责清晰**: 每个任务都应能清晰地归属到 `Handler`, `Application`, `Domain`, `Infrastructure` 中的某一层。
- **显式选择策略**: 涉及写操作的任务，必须显式声明其**事务策略**（`use_db_commit` 或 `drop_db_commit`）、**幂等键**的来源和设计。
- **指标命名具体**: 涉及观测的任务，必须给出具体的**指标名称**（Metric Name），而不是“增加监控”这种模糊描述。

---

### 3. 高质量任务示例 (Few-shot Task Example)

**场景**: 为星图达人新增一个“设置专属折扣”的功能。该功能允许 MCN 为其名下达人设置一个全局的专属折扣率（如 95%），该设置会影响后续所有报价的计算。

**高质量 `tasks.md` 示例**:

```markdown
## 1. 接口与协议 (IDL)
- [ ] **1.1 [IDL]** 设计 `SetAuthorExclusiveDiscount` 接口:
  - **文件**: `code.byted.org/ad/star_idl/idl/go_author/author_biz.thrift`
  - **接口**:
    ```thrift
    
    SetAuthorExclusiveDiscountResp SetAuthorExclusiveDiscount(1: SetAuthorExclusiveDiscountReq req)(api.post='/gw/api/gauthor/set_author_discount');// http_post use_db_commit 设置达人专属折扣
    ```
  - **结构体**:
    ```thrift
    struct SetAuthorExclusiveDiscountReq {
        1: required i64 s_mcn_id (api.header="s_mcn_id"); // MCN 登录态
        2: required i64 author_id;
        3: required i32 discount_rate; // 万分位, e.g., 9500 for 95%
        255: required base.Base Base;
    }

    struct SetAuthorExclusiveDiscountResp {
        255: required base.BaseResp BaseResp;
    }
    ```

## 2. 存储与数据 (DB)
- [ ] **2.1 [DB]** 新增 `star_author_discount` 表:
  - **文件**: `infrastructure/mysql/sql/star_author_discount.sql`
  - **DDL**:
    ```sql
    CREATE TABLE `star_author_discount` (
      `id` bigint unsigned NOT NULL AUTO_INCREMENT,
      `author_id` bigint NOT NULL COMMENT '达人ID',
      `discount_rate` int NOT NULL COMMENT '折扣率（万分位）',
      `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
      `modify_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
      `deleted` tinyint NOT NULL DEFAULT '0',
      PRIMARY KEY (`id`),
      UNIQUE KEY `uk_author_id` (`author_id`)
    ) ENGINE=InnoDB COMMENT='达人专属折扣表';
    ```
- [ ] **2.2 [Model/DAO]** 生成新表对应的 Model 和 DAO/DAL 文件。
- [ ] **2.3 [Cache]** 设计缓存结构:
    - **Key**: `star:goauthor:author_discount:{author_id}`
    - **Value**: `{"discount_rate": 9500}` (JSON String)
    - **TTL**: 24 hours with random jitter.
    - **策略**: 更新 DB 后，采用 `Cache-Aside` 模式，主动删除缓存。

## 3. 业务实现 (Coding)
- [ ] **3.1 [Handler]** 实现 `handler.SetAuthorExclusiveDiscount`:
  - **文件**: `handler/handler.go`
  - **职责**: 校验 `discount_rate` 在 (0, 10000] 之间, 调用 `application.AuthorDiscountApp.SetDiscount`。
- [ ] **3.2 [Application]** 实现 `application.author_discount.SetDiscount`:
  - **文件**: `application/author_discount/discount_app.go`
  - **职责**:
    1.  调用风控服务校验 MCN 与达人的绑定关系。
    2.  开启事务 (`transaction.Exec`)。
    3.  调用 `domain.AuthorDiscountDomain.Save` 保存到数据库。
    4.  调用 `domain.CacheDomain.DeleteAuthorDiscount` 删除缓存。
    5.  发出领域事件（可选）。
- [ ] **3.3 [Domain]** 定义 `AuthorDiscountDomain`:
  - **文件**: `domain/author_discount_domain/discount.go`
  - **职责**: 定义 `Repository` 接口 (`Save`, `Get`)。
- [ ] **3.4 [Infrastructure]** 实现 `Repository` 和 `Cache`:
  - **文件**: `infrastructure/repository/author_discount_repo.go`, `infrastructure/redis/author_discount_cache.go`
  - **职责**: 调用`infrastructure/mysql/base_dal/star_author_discount` 中能力实现 DB 的 `Upsert` 逻辑和 Redis 的 `Delete` 逻辑。

## 4. 可观测性 (Observability)
- [ ] **4.1 [Log]** 在 `handler` 和 `application` 层添加入参、出参和关键决策的结构化日志。

## 5. 测试 (Testing)
- [ ] **5.1 [Unit Test]** 编写 `application.SetDiscount` 的单测:
  - **文件**: `application/author_discount/discount_app_test.go`
  - **场景**:
    - 成功设置。
    - `discount_rate` 超出范围的错误场景。
    - MCN 与达人无绑定关系的权限错误场景。
    - 依赖的 Repository 返回错误的场景。
```
根据仓库内容和以上格式，产出适配本仓库的Task Generation & Guideline并放在External Dependencies之前

### Step 4: Create `openspec/specs/` Directory
仔细分析代码仓库中的功能，并进行合适的归类，遵循openspec规范补充openspec/specs，完整描述每个功能的主要逻辑，不能遗漏任何功能且内容需要尽可能详细。请使用简体中文描述。

$ARGUMENTS

<!-- OPENSPEC:END -->
