# 星图资金结算领域业务术语

## 一、核心术语定义

### 收款

- **简明定义**：**客户在星图平台创建任务后，向平台入账的资金**操作。代表了一次具体的资金行为

- **业务概念**：

    1. **支付订单 **: 对应系统中的star\_settlement\_order实体\. 客户侧所有资金行为的前提都是先有一笔支付订单创建

    2. **支付流水**:  对应系统中的star\_settlement\_flow实体\. 对于一笔支付订单，可以有多次、多种操作流水，包括首次支付、追加预算、解冻资金、扣款等。

    3. **冻结**:  客户支付订单时，账户上的钱会被冻结在订单上。首次支付和追加预算，都属于冻结

    4. **解冻**:  客户取消订单时，订单上的钱会解冻回到账户上。部分退款和全额退款场景，都属于解冻

    5. **扣款**:  客户确认验收订单时，冻结的金额会被扣除。钱从账户余额转移到星图收入户上。扣款发生后线上无逆向流程，如果有发生错扣等情况，需要走调差流程。

- **相关模型**：

    - ad\_imce库star\_settlement\_order表

    - ad\_imce库star\_settlement\_flow表

#### 典型场景短例

- **普通指派任务**: 客户创建一个 10,000 元的指派任务，系统会生成一个支付单（BizId=10），并冻结客户账户中的 10,000 元。

- **追投加热**: 任务进行中，客户决定追加 5,000 元用于加热，系统会再生成一个支付单（BizId=6），并冻结 5,000 元。

- **任务取消**: 客户取消任务，系统会操作对应的支付单，将已冻结但未扣款的资金解冻并退还至客户账户。



### 出款

- **简明定义**：是达人/服务商等角色在星图平台完成任务后，平台向达人/服务商等角色结算，包含了入账、资金变动流水明细，提现等。它是个人或机构视角下的“银行账本”。

- **检索标签**: 结算记录, 结算单, 对账, `star\_user\_trade\_record`

- **业务概念**：

    1. **入账 **: 任务完成后，平台根据任务金额扣除平台服务费后向达人等角色打款

    2. **提现**:  对应系统中的star\_settlement\_flow实体\. 对于一笔支付订单，可以有多次、多种操作流水，包括首次支付、追加预算、解冻资金、扣款等。

    3. **对公结算**:  客户支付订单时，账户上的钱会被冻结在订单上。首次支付和追加预算，都属于冻结

    4. **对私结算**:  客户取消订单时，订单上的钱会解冻回到账户上。部分退款和全额退款场景，都属于解冻

    5. **结算单**:  客户确认验收订单时，冻结的金额会被扣除。钱从账户余额转移到星图收入户上。扣款发生后线上无逆向流程，如果有发生错扣等情况，需要走调差流程。

- **相关数据模型**：`star\_user\_trade\_record`, star\_settlement\_flow

#### 关键属性

- **结算记录 \(****`star\_user\_trade\_record`****\)**:

    - `amount`: 金额为正数；收入/支出由 `op\_type` 判定。        \- `status`: 结算状态（如待结算、已结算、已提现）。

    - `op\_type`: 操作类型（如任务收入、提现）。

    - `business\_type`: 业务类型。

#### 领域边界 \(Scope Boundary\)

- **归属资金域**: 结算记录的生成、结算单的创建与状态流转、以及与财务系统的对接均在**资金域**完成。

- **常见误解**:

    - **误解**: 结算记录就是订单。

    - **反例**: 订单是单次打款行为的凭证，而结算记录是该笔打款在收款方账户上的体现。结算记录还包含了提现等其他非订单类资金变动。

#### 上下游与触发

- **上游触发**:

    - **订单 \(Order\)**: 订单支付成功后，会生成一笔op\_type=2的入账结算记录。

    - **提现 \(Withdrawal\)**: 用户发起提现操作，会生成一笔op\_type=1的提现结算记录。

    - **对公结算周期**: 到达结算周期（如每月初），系统会自动或由运营手动圈选待结算的记录，生成结算单。

- **下游依赖**:

    - **提现/打款**: 结算单的最终确认会触发对公账户的实际打款操作。

    - **发票**: 对公结算流程中，结算单需要关联有效的发票。

#### 数据表/服务映射

- **核心模型表**:

    - `star\_user\_trade\_record`: 达人结算记录表。

    - `imce\_user\_trade\_record\_xingtu`: 新版结算记录表。

    - `star\_corporate\_settlement`: 对公结算单记录表。

#### 典型场景短例

- **MCN 对公结算**: 某 MCN 在 5 月份有多笔总计 100,000 元的结算记录。6 月初，系统将这些记录打包生成一个 `star\_corporate\_settlement` 结算单，MCN 完成在线签章并上传发票后，财务进行打款。

- **个人达人收入**: 达人完成任务获得一笔 5,000 元的订单付款，其账户的 `star\_user\_trade\_record` 会增加一条金额为 5,000 元的记录，op\_type=收入（订单入账）。

#### JSON 元数据

```JSON
{
  "term": "结算单 / 结算记录 (Corporate Settlement / Trade Record)",
  "aliases": ["交易记录", "结算明细", "对公结算单"],
  "primary_keys": ["结算记录ID", "结算单ID"],
  "domain": "funds",
  "scope_boundary": "结算记录与结算单的生成和管理属资金域，依赖于订单和提现操作。",
  "tables": ["star_user_trade_record", "imce_user_trade_record_xingtu", "star_corporate_settlement"],
  "psm": [],
  "events": ["订单支付成功", "发起提现", "对公结算周期到达"],
  "tags": ["结算记录", "结算单", "对账"]
}

```

### 提现 \(Withdrawal\)

- **Canonical 名称**: 提现 \(Withdrawal\)

- **同义词/别名**: 打款 \(对私\), 个人结算

#### 简明定义

- **检索标签**: 提现, 打款, 对私结算, 收款人信息, `star\_user\_trade\_record`

提现是指个人收款方（如野生达人）将其在星图平台账户中的可结算余额，转移到其个人银行账户的过程。这是**对私打款**的主要形式。**结算单 \(Corporate Settlement\)** 是针对**对公结算**场景（如与 MCN 机构、企业达人结算）的聚合凭证。它将一个结算周期内的多笔结算记录（Trade Records）打包在一起，形成一个正式的结算请求，并关联电子签章、发票等对公流程所需材料。

#### 关键属性

- **提现金额 \(Amount\)**: 用户申请提现的数额。

- **收款人信息 \(Auth Info\)**: 提现操作依赖于用户预先维护的收款账户信息，包括银行卡号、开户行、身份证信息等。

- **状态 \(Status\)**: 提现流程的状态，如审核中、打款中、已到账、失败等。

- **结算单 \(Corporate Settlement\)**:

    - `trade\_ids`: 包含的结算记录 ID 列表。

    - `status`: 结算单状态（如待签章、待打款、已完成）。

    - `statement\_id`: 关联的 MMM（财务系统）结算 ID。

    - `sign\_url`: 电子签章的链接。

#### 领域边界 \(Scope Boundary\)

- **归属资金域**: 提现申请的受理、风控审核、与支付渠道对接完成打款等全流程均在**资金域**。

- **依赖结算记录**: 提现的可用额度取决于“结算记录 \(Trade Record\)”中可结算的余额。

#### 上下游与触发

- **上游触发**:

    - **用户操作**: 个人用户在平台前端页面主动发起提现申请。

- **下游依赖**:

    - **结算记录 \(Trade Record\)**: 提现成功后，会生成一笔负向的结算记录，减少账户余额。

    - **银行/支付渠道**: 资金域会调用第三方支付或银行接口完成最终的资金划转。

#### 数据表/服务映射

#### 典型场景短例

#### JSON 元数据

### 发票 \(Invoice\)

- **达人提现**: 一位个人达人账户中有 8,000 元可结算余额，他在星图 App 中发起一笔 5,000 元的提现到自己的储蓄卡。资金域系统审核通过后，调用银行接口将 5,000 元打入其卡中，并生成一条金额为 5,000 元的结算记录，op\_type=提现。

- **Canonical 名称**: 发票 \(Invoice\)

```JSON
{
  "term": "提现 (Withdrawal)",
  "aliases": ["打款 (对私)", "个人结算"],
  "primary_keys": ["提现申请ID"],
  "domain": "funds",
  "scope_boundary": "提现全流程属资金域，依赖于结算记录中的可结算余额。",
  "tables": ["star_user_trade_record"],
  "psm": [],
  "events": ["用户发起提现申请"],
  "tags": ["提现", "打款", "对私结算", "收款人信息"]
}

```

- **同义词/别名**: 开票, `star\_invoice`

- **检索标签**: 发票, 开票, 可开票任务, `star\_invoice\_task`, `star\_invoice`

#### 简明定义

发票是客户在星图平台消费后，用于报销和税务目的的正式凭证。资金域管理着哪些任务可以开票、发票的申请、开具与状态跟踪的全流程。

#### 关键属性

- **发票抬头**: 开具发票的对象名称，即公司名。

- **发票金额**: 发票上记录的金额。

- **关联任务 \(Invoice Task List\)**: 一张发票可以关联一个或多个“可开票任务”。

- **发票状态**: 如待开具、已开具、已邮寄等。

#### 领域边界 \(Scope Boundary\)

- **归属资金域**: 发票相关的逻辑，包括可开票任务的生成、发票任务的管理、发票的开具与邮寄状态跟踪，均属于**资金域**。

- **依赖支付与计费**: 只有经过客户支付并成功扣款的任务，才有可能成为可开票任务。

#### 上下游与触发

- **上游触发**:

    - **扣款成功**: 客户支付的订单被成功扣款后，系统会生成对应的“可开票任务”记录。

    - **客户申请**: 客户在平台前端选择可开票任务，提交开票申请。

- **下游依赖**:

    - **财务开票系统**: 资金域会将开票请求传递给集团的统一财务系统来完成电子或纸质发票的实际开具。

#### 数据表/服务映射

- **核心模型表**:

    - `star\_invoice\_task`: 可开票任务表，记录了每一笔可以用于开票的已消耗任务。

    - `star\_invoice`: 发票主表，记录了每一张发票的抬头、金额、关联任务等信息。

    - `imce\_settlement\_invoice`: 结算电子发票表，用于对公结算场景。

#### 典型场景短例

- **客户合并开票**: 客户在一个月内完成了 3 个任务，总计消费 50,000 元。月底，客户在开票中心看到了这 3 个可开票任务，他选择将它们合并到一张发票中，提交了开票申请。

#### JSON 元数据

```JSON
{
  "term": "发票 (Invoice)",
  "aliases": ["开票", "star_invoice"],
  "primary_keys": ["发票ID", "发票任务ID"],
  "domain": "funds",
  "scope_boundary": "发票相关逻辑均属资金域，依赖于已成功扣款的任务。",
  "tables": ["star_invoice_task", "star_invoice", "imce_settlement_invoice"],
  "psm": [],
  "events": ["扣款成功", "客户提交开票申请"],
  "tags": ["发票", "开票", "可开票任务"]
}

```

### 资金分配 \(Allocation\)

- **Canonical 名称**: 资金分配 \(Allocation\)

- **同义词/别名**: 预算分配

- **检索标签**: 资金分配, 预算分配, `star\_payment\_order\_allocation`

#### 简明定义

资金分配是指将一笔入账资金（通常来自一个更高层级的支付单，如项目或需求层级）拆分并关联到多个具体的下游营销服务（任务/增值/附加）上的过程。它解决了“一笔钱花在多个地方”的账目对应问题。

#### 关键属性

- **源支付单 ID**: 用于分配的资金来源支付单。

- **目标实体 ID**: 资金被分配到的具体任务或服务实例。

- **分配金额**: 分配给该目标实体的具体金额。

#### 领域边界 \(Scope Boundary\)

- **归属资金域**: 资金分配的记录和管理属于**资金域**的范畴。

- **由交易域驱动**: **交易域**的业务逻辑（如一个需求下包含多个任务）决定了资金需要如何被分配。

#### 上下游与触发

- **上游触发**: 在一个聚合层级（如项目或需求）上创建支付单，并需要将这笔资金用于其下的多个子任务时，会触发资金分配。

- **下游依赖**: 具体的任务或服务实例，它们的预算和可消费金额来源于资金分配的结果。

#### 数据表/服务映射

- **核心模型表**:

    - `star\_payment\_order\_allocation`: 资金分配记录表。

#### 典型场景短例

- **一个需求下多个任务**: 客户在一个需求下创建了 2 个指派任务和 1 个附加费服务，并为此支付了一笔总金额。资金域会将这笔总支付金额，按各个服务的价格，通过资金分配记录分别关联到这 3 个服务上。

#### JSON 元数据

```JSON
{
  "term": "资金分配 (Allocation)",
  "aliases": ["预算分配"],
  "primary_keys": ["分配记录ID"],
  "domain": "funds",
  "scope_boundary": "资金分配的记录和管理属资金域，由交易域的业务逻辑驱动。",
  "tables": ["star_payment_order_allocation"],
  "psm": [],
  "events": ["聚合层级支付单创建"],
  "tags": ["资金分配", "预算分配"]
}

```

### 支付流水 \(Settlement Flow\)

- **Canonical 名称**: 支付流水 \(Settlement Flow\)

- **同义词/别名**: 资金流水, `star\_settlement\_flow`

- **检索标签**: 支付流水, 资金流水, 操作记录

#### 简明定义

支付流水是**支付单 \(Payment Order\)** 下属的、更细粒度的资金操作记录。它详细记载了围绕一笔支付单发生的所有具体动作，如首次支付、追加预算、解冻资金、部分扣款等。每一条流水都是对支付单状态和金额变动的一次快照。

#### 关键属性

- **支付单 ID**: 关联的支付单。

- **操作类型 \(Operation Type\)**: 如 `支付 \(Pay\)`、`退款 \(Refund\)`、`扣款 \(Charge\)`、`冻结 \(Freeze\)`、`解冻 \(Unfreeze\)`。

- **操作金额**: 本次流水操作的金额。

- **成本项 \(Cost Item\)**: 标记流水的成本归属，如 OA 成本、电商成本等，用于更精细的预算和 quota 控制。

#### 领域边界 \(Scope Boundary\)

- **归属资金域**: 支付流水的生成和记录是**资金域**内部的核心功能，是对支付单操作的原子化记录。

#### 上下游与触发

- **上游触发**: 任何对“支付单”进行的操作（如调用支付、退款、扣款接口）都会生成一条对应的支付流水。

- **下游依赖**: 支付流水是生成客户对账单、进行财务审计和问题排查的最终数据依据。

#### 数据表/服务映射

- **核心模型表**:

    - `star\_settlement\_flow`: 支付流水主表。

#### 典型场景短例

- **一笔任务的完整生命周期**: 一个任务支付单创建后，`star\_settlement\_flow` 表中可能会依次记录下：

    1. 一条 `冻结` 流水（客户支付）。

    2. 一条 `扣款` 流水（任务完成，平台扣费）。

    3. 一条 `解冻` 流水（任务有余款，退还给客户）。

#### JSON 元数据

```JSON
{
  "term": "支付流水 (Settlement Flow)",
  "aliases": ["资金流水", "star_settlement_flow"],
  "primary_keys": ["流水ID"],
  "domain": "funds",
  "scope_boundary": "支付流水的生成和记录是资金域的核心功能。",
  "tables": ["star_settlement_flow"],
  "psm": [],
  "events": ["支付单操作"],
  "tags": ["支付流水", "资金流水", "操作记录"]
}

```

### 消费记录 \(Customer Bill\)

- **Canonical 名称**: 消费记录 \(Customer Bill\)

- **同义词/别名**: 客户账单, `star\_customer\_bill\_info`

- **检索标签**: 消费记录, 客户账单

#### 简明定义

消费记录是从**客户视角**出发的账单明细，用于告知客户其资金在星图平台的具体花费去向。它通常直接关联到具体的业务任务名称，让客户能够清晰地将消费金额与业务活动对应起来。

#### 关键属性

- **类型**: 消费的业务分类，如广告投放、任务费用等。

- **任务名称**: 关联的具体业务任务名称。

- **金额**: 该笔消费的资金数值。

#### 领域边界 \(Scope Boundary\)

- **归属资金域**: 消费记录的生成和管理属于**资金域**。

- **与支付流水的关系**: 消费记录可以看作是支付流水中“扣款”类流水的一个面向客户的、更业务化的友好展示。它隐藏了冻结、解冻等中间过程，直接展示最终的实际花费。

#### 上下游与触发

- **上游触发**: 通常在“扣款”事件发生后，系统会生成一笔客户消费记录。

- **下游依赖**: 是客户中心、财务报表等展示客户实际消费情况的数据来源。

#### 数据表/服务映射

- **核心模型表**:

    - `star\_customer\_bill\_info`: 消费记录主表。

#### 典型场景短例

- **客户查看账单**: 客户在自己的账户后台查看本月账单，会看到一条条类似“【视频任务】与达人A合作，\-5000元”、“【直播任务】B达人专场，\-20000元”的记录，这些就是消费记录。

#### JSON 元数据

```JSON
{
  "term": "消费记录 (Customer Bill)",
  "aliases": ["客户账单", "star_customer_bill_info"],
  "primary_keys": ["消费记录ID"],
  "domain": "funds",
  "scope_boundary": "消费记录的生成和管理属资金域，是扣款流水的业务化展示。",
  "tables": ["star_customer_bill_info"],
  "psm": [],
  "events": ["扣款成功"],
  "tags": ["消费记录", "客户账单"]
}

```


# 星图资金结算域业务流程知识

## 域职责与范围

星图作为撮合平台，其资金结算域的核心职责是管理从客户到平台，再到业务参与者（如达人、服务商）的全链路资金流转。

#### 资金动线总览

平台整体的资金动线遵循以下核心步骤：

1. **充值**：客户将资金充值至星图平台账户。

2. **冻结**：客户发布任务或下单时，平台会根据任务预算冻结相应金额。

3. **扣款**：任务完成后，平台根据实际消耗从冻结金额中扣除相应款项。

4. **解冻**：如任务取消或未完全消耗预算，剩余的冻结资金将解冻，退回至客户的可用余额。

5. **打款**：平台将任务款项结算并支付给参与任务的达人、MCN 机构或服务商。

6. **发票**：收钱方需出具发票给出款方，即平台需提供电子发票给客户，达人或机构等需提供电子发票给平台

---

## 核心中心能力

资金结算域通过以下五大中心对外提供服务，构成了其核心能力矩阵。

### 订单中心

订单中心是连接上层业务与底层资金、票据的桥梁，是完成资金扣费的唯一业务维度。

<table><tbody>
<tr>
<td>

能力要点

</td>
<td>

关键字段或维度

</td>
<td>

关键校验与幂等提示

</td>
<td>

典型状态或状态机

</td>
</tr>
<tr>
<td>

**创建订单**：为每次业务交易（如任务创建）生成一个唯一的结算订单。

**冻结/解冻**：响应任务状态，锁定或释放客户账户资金。

**扣款**：在任务确认完成后，执行实际的资金扣减。

**完结订单**：标记业务流程结束，并将实收消耗推送至下游系统。

**同步消耗/支付信息**：为不直接涉及资金实时流转的业务（如推数结算）提供记账凭证。

</td>
<td>

**订单类型 \(biz\_id\)**：区分不同业务模式，如 `预充值结算订单`（资金随信息流转）和 `推数结算订单`（仅记账，资金由财务处理）。新增业务模式时，为保证财务核算清晰，需新增订单类型。

**业务方 \(****`biz\_type`****\)**：资金类型，常用的如PAYMENT\_TO\_PLATFORM = 1  *// 支付给平台*
REMIT\_TO\_USER = 4  *// 打款给用户*

**订单 ID \(****`biz\_order\_id`****\)**：全局唯一。上游业务订单id

</td>
<td>

**幂等性**：所有资金操作接口（冻结、解冻、扣款）均需保证幂等，防止重复操作。

**前置校验**：扣款前必须校验订单是否已有足够冻结金额。

</td>
<td>

**订单状态机：10：待创建；20：进行中；30：已完成；40：已取消**

</td>
</tr>
</tbody></table>

### 资金中心

资金中心负责管理平台内所有资金的类型、流入、流出和状态。

<table><tbody>
<tr>
<td>

能力要点

</td>
<td>

关键字段或维度

</td>
<td>

关键校验与幂等提示

</td>
<td>

典型状态或状态机

</td>
</tr>
<tr>
<td>

**资金流入**：支持客户多种方式入金，包括 `充值`、`授信下发`、`赠款发放`、`补贴金下发`。

**资金流出**：支持客户 `退款`（仅限预充值资金）。

**资金失效**：支持对特定资金类型（如补贴金）设置有效期，到期自动失效。

</td>
<td>

**资金类型 **：

- `预充值`: 客户主动存入，无有效期。

- `授信`: \&\#34;先消费后付款\&\#34;，一种特殊的预充值。

- `赠款`: 平台营销活动发放，可设有效期，不计入业绩。

- `补贴`: 面向非商业化部门的赠款解决方案，可设有效期。

- `贷款`: 用于特定场景（如星立方）。

**资金池**：用于区分不同来源或用途的资金，例如“普通赠款”和“星广联投赠款”使用不同的 `payment\_source`。

</td>
<td>

**退款校验**：申请退款时，系统需校验资金类型，仅 `预充值` 类型可退。

**补贴金发放**：需指定有效期。

</td>
<td>

**资金状态：10：待创建；20：进行中；30：已完成；40：已取消**

</td>
</tr>
</tbody></table>

### 成本中心

成本中心用于管理与订单关联的预算或成本来源，确保业务支出的合规性。

<table><tbody>
<tr>
<td>

能力要点

</td>
<td>

关键字段或维度

</td>
<td>

关键校验与幂等提示

</td>
<td>

典型状态或状态机

</td>
</tr>
<tr>
<td>

**成本关联**：客户下单时，将订单与一个具体的成本项进行绑定。

**成本校验**：下单时校验关联的成本是否充足。

**成本占用/扣减**：订单冻结资金时同步占用成本额度，扣款时同步扣减。

</td>
<td>

**成本类型**：

- `OA成本`: 集团 OA 立项的成本，如采购、报销。

- `招商成本`: 售卖给外部商家的资源包。

- `电商成本`: 费用承担部门为电商的成本。

**下单方式**：

- `媒介下单`: 销售或运营通过内部系统操作。

- `自助下单`: 客户自行在平台操作。

</td>
<td>

**成本充足性校验**：下单及资金冻结时，必须校验成本余额是否足够覆盖订单金额。

</td>
<td>



</td>
</tr>
</tbody></table>

### 支出中心

支出中心负责向平台各类收益角色（达人、机构、服务商）进行打款。

<table><tbody>
<tr>
<td>

能力要点

</td>
<td>

关键字段或维度

</td>
<td>

关键校验与幂等提示

</td>
<td>

典型状态或状态机

</td>
</tr>
<tr>
<td>

**对私支出**：向野生达人打款，资金进入其抖音钱包。支持税筹方案，平台代扣代缴个税。

**对公支出**：向企业达人、MCN、服务商打款。流程涉及生成分成明细、聚合成结算单、签章、上传发票等。

</td>
<td>

**结算方式 \(****`settlement\_type`****\)**：`1` \(对私\) 或 `2` \(对公\)。

**MMM 业务类型 \(****`biz\_type`****\)**：用于下游 3M 系统区分业务，如 `星图PGC`、`星图MCN` 等。

**费率 \(****`tax\_fee`****\)**：如对私打款的服务费（原文为 5%）、对公不同角色的服务费（PGC 5%，MCN 3%）。

</td>
<td>

**对私打款前置校验**：黑名单、用户账号状态、打款金额、单日限额、接口幂等。

**对公打款前置校验**：黑名单、用户资质、合同、统一社会信用代码、金额、是否重复打款。

**幂等提示**：所有打款相关接口必须实现幂等，防止重复支付。

</td>
<td>

**对公结算状态 \(****`SettlementStatus`****\)**

**结算单状态 \(****`CorporateSettlementStatus`****\)**

（详见业务流程部分）

</td>
</tr>
</tbody></table>

### 票据中心

票据中心管理客户侧的发票开具和支出侧的发票上传与审核。

<table><tbody>
<tr>
<td>

能力要点

</td>
<td>

关键字段或维度

</td>
<td>

关键校验与幂等提示

</td>
<td>

典型状态或状态机

</td>
</tr>
<tr>
<td>

**客户开票**：客户消费后，平台向其提供增值税发票作为做账凭证。

**支出侧上传发票**：对公结算角色（达人/机构/服务商）在结算单签章后，上传发票作为打款的必要条件。

</td>
<td>

**开票项目**：由财税根据服务性质决定，如 `信息技术服务\*信息服务`、`广告服务\*推广`。

**发票类型**：目前线上只支持 `数电票`（专票/普票）。

**开票平台**：`直客` 在星图平台开票，`虚客` 的代理商在方舟平台开票。

</td>
<td>

**发票金额校验**：支出侧上传发票时，需确保发票金额与关联的结算单金额一致。

**发票审核**：财务会审核发票信息与合同信息、金额是否一致。

</td>
<td>

**电子发票状态**：

`processing` \(1\): 审核中

`success` \(2\): 审核通过

`fail` \(3\): 审核失败

</td>
</tr>
</tbody></table>

---

## 业务流程

本章节详述资金结算域的几个核心业务流程，并提供相应的状态机定义。

### 对私打款

适用于向“野生达人”的个人账户支付收益。

<table><tbody>
<tr>
<td>

环节

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

**前置条件**

</td>
<td>

1. 上游业务（如交易侧）已完成计费，确定了达人收益。

2. 达人已完成实名认证，且账户状态正常。

</td>
</tr>
<tr>
<td>

**关键步骤**

</td>
<td>

1. **调用打款接口**：上游业务系统根据结算周期，携带收益金额、达人信息等调用对私打款接口。

2. **前置校验**：结算系统执行一系列校验，包括但不限于：

    - 是否在黑名单内

    - 用户账户状态是否正常

    - 接口调用是否重复（幂等校验）

3. **查询结算配置**：获取达人的结算配置，如税筹身份、费率（5%）等。

4. **请求下游入账**：向下游的抖音钱包服务发起入账请求。

5. **更新结算明细**：在达人的星图账户中生成一条结算明细记录。

</td>
</tr>
<tr>
<td>

**后置状态**

</td>
<td>

1. 达人抖音钱包余额增加。

2. 星图平台生成对应的结算流水。

</td>
</tr>
</tbody></table>

#### 状态机枚举

- **对私打款状态机：**status=3 打款成功，status=1 打款失败

### 对公打款

适用于向企业认证的达人（PGC）、MCN 机构或服务商进行对公支付。

<table><tbody>
<tr>
<td>

环节

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

**前置条件**

</td>
<td>

1. 上游业务完成计费。

2. 收益方（PGC/MCN/服务商）已完成企业资质认证、对公验证、CA 认证并签署了合同。

</td>
</tr>
<tr>
<td>

**关键步骤**

</td>
<td>

1. **调用接口同步分成明细**：上游业务调用接口，将分成明细推送到结算系统。

2. **前置校验**：结算系统执行校验，包括：

    - 黑名单、用户资质、合同、企业统一社会信用代码

    - 打款金额、是否重复打款

    - **常见失败点**：因资质问题导致订单推送 3M 失败。

3. **推送 3M 系统**：校验通过后，查询结算配置（如费率），将分成明细数据推送到下游的 3M 系统。

4. **更新状态**：推送成功后，该笔结算记录在星图侧转为 `un\_settle`（待结算）状态。若推送失败，则记录不会在星图前端展示给用户。

</td>
</tr>
<tr>
<td>

**后置状态**

</td>
<td>

1. 在 3M 系统中生成一条分成明细订单。

2. 在星图系统中，对应的结算记录状态变为 `un\_settle`（待结算）。

</td>
</tr>
</tbody></table>

#### 状态机枚举：对公结算状态 \(`SettlementStatus`\)

```Python
class SettlementStatus(ChoiceIntEnum):
    """对公打款（分成明细）的状态机"""
    init = 0           # 初始值
    un_settle = 1      # 待结算 (已成功推送3M，等待用户操作)
    pushed = 2         # 已经推送 (历史状态，现多用 un_settle)
    settling = 3       # 结算中 (已关联到结算单)
    finish = 4         # 已结算 (结算单已完成打款)
    not_push = 7       # 暂时不推送
    syncing = 8        # 同步中
    push_not_show = 9  # 推送但不展示 (一种特殊策略)

```

### 提现

特指“野生达人”将其在抖音钱包中的星图收益提取至个人银行卡或支付宝。

<table><tbody>
<tr>
<td>

环节

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

**前置条件**

</td>
<td>

1. 达人抖音钱包内有来自星图的可用余额。

2. 达人已绑定有效的提现账户（银行卡或支付宝）。

</td>
</tr>
<tr>
<td>

**关键步骤**

</td>
<td>

1. **发起提现请求**：达人在星图 App 或 PC 端发起提现操作。

2. **提现前置校验**：系统执行一系列安全和合规校验：

    - 短信验证码校验

    - 黑名单校验

    - 税筹身份实名校验

    - 大额提现风控校验

3. **选择提现方式**：校验通过后，用户进入提现页面，选择支付方式（银行卡/支付宝）和确认金额。

4. **调用支付渠道**：系统根据用户的税筹身份和支付方式，通过不同的商户通账户调用相应支付渠道执行出款。

</td>
</tr>
<tr>
<td>

**后置状态**

</td>
<td>

1. 提现成功或失败，用户均可在提现记录中查看明细。

2. 抖音钱包余额相应扣减。

</td>
</tr>
</tbody></table>

#### 状态机枚举

- **提现流程状态机：status=3 提现成功，status=1 提现失败**

### 结算单

对公结算的核心凭证，将多笔“待结算”的分成明细聚合在一起进行处理。

<table><tbody>
<tr>
<td>

环节

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

**前置条件**

</td>
<td>

PGC 达人/机构/服务商账户中存在一或多笔状态为 `un\_settle`（待结算）的结算记录。

</td>
</tr>
<tr>
<td>

**关键步骤**

</td>
<td>

1. **生成结算单**：用户在财务管理页面手动勾选待结算的记录，点击“生成结算单”。系统将这些记录聚合成一个结算单，总金额为所选记录金额之和。关联的结算记录状态变为 `settling`（结算中）。

2. **取消结算单**：在签章前，用户可取消结算单。取消后，结算单状态变为 `canceled`（已取消），关联的结算记录状态回滚至 `un\_settle`（待结算）。

3. **结算单签章**：用户确认结算单金额、收款账户等信息无误后，发起线上电子签章。签章完成后，结算单状态变为 `signed`（已签署）。

</td>
</tr>
<tr>
<td>

**后置状态**

</td>
<td>

结算单进入 `signed`（已签署）状态，等待上传发票。

</td>
</tr>
</tbody></table>

#### 状态机枚举：结算单状态 \(`CorporateSettlementStatus`\)

```Python
class CorporateSettlementStatus(ChoiceIntEnum):
    """结算单的状态机"""
    init = 0        # 初始值
    signing = 1     # 签署中
    canceled = 2    # 已取消
    signed = 3      # 已签署 (等待上传发票)
    auditing = 5    # 审批中 (发票审核或OA打款流程)
    settled = 4     # 已提现/已结算 (打款完成)
    syncing = 6     # 同步中

```

### 电子发票

对公结算流程中的关键一步，是完成打款的必要条件。

<table><tbody>
<tr>
<td>

环节

</td>
<td>

描述

</td>
</tr>
<tr>
<td>

**前置条件**

</td>
<td>

存在一个或多个状态为 `signed`（已签署）的结算单。

</td>
</tr>
<tr>
<td>

**关键步骤**

</td>
<td>

1. **上传发票**：对公用户上传电子发票，并将其与一个或多个已签章的结算单进行关联。

2. **金额校验**：系统需确保关联的结算单总金额与发票总金额完全一致。

3. **发票审核**：上传后，发票进入审核流程。财务人员会核对发票信息、合同信息及金额。审核状态可能为 `processing` \-\&gt; `success` 或 `fail`。

4. **进入打款**：发票审核通过（`success`）后，对应的结算单进入后续的 OA 打款流程。

</td>
</tr>
<tr>
<td>

**后置状态**

</td>
<td>

结算单进入 `auditing`（审批中）或后续打款状态。

</td>
</tr>
</tbody></table>

#### 状态机枚举：电子发票状态 \(`ElectronicInvoiceStatus`\)

```Python
# 电子发票状态
ElectronicInvoiceStatus_processing = 1  # 审核中
ElectronicInvoiceStatus_success = 2     # 审核通过
ElectronicInvoiceStatus_fail = 3        # 审核失败

```

---

## 资金流水与对账

流水是记录客户资金变动的明细，主要用于客户对账。

#### 对账公式

为了确保资金平衡，系统中的各项流水需满足以下对账逻辑：

```Plain Text
(充值流水 + 赠款发放流水 + 补贴发放流水)
- (补贴消耗流水 + 任务扣费流水 + 赠款消耗流水)
- (退款金额 + 补贴失效流水)
= 剩余冻结金额 + 剩余可用资金余额 + 赠款余额 + 补贴余额

```

---

## 用户与角色

资金结算域涉及多种用户角色，其定义和结算场景各不相同。

<table><tbody>
<tr>
<td>

角色大类

</td>
<td>

细分角色

</td>
<td>

定义

</td>
<td>

结算场景说明

</td>
</tr>
<tr>
<td>

**客户**

</td>
<td>

直客

</td>
<td>

由销售直接对接，通过报备链路入驻的客户。

</td>
<td>

- 使用自身创建的合同 ID 进行结算。

- 在星图平台直接操作充值、开票。

</td>
</tr>
<tr>
<td>



</td>
<td>

虚客

</td>
<td>

挂接在星图代理商下的客户，由代理商创建和管理。

</td>
<td>

- 所有资金结算均使用其挂接的代理商信息。

- 充值、开票等操作需由代理商在方舟平台完成。

</td>
</tr>
<tr>
<td>

**代理商**

</td>
<td>

​\-

</td>
<td>

负责拓展客户资源、推广星图服务的合作伙伴。

</td>
<td>

- 在方舟平台完成入驻和结算操作。

</td>
</tr>
<tr>
<td>

**收益方（对私）**

</td>
<td>

野生达人

</td>
<td>

在抖/头/火/西端注册，未进行对公认证的自然人达人。

</td>
<td>

- **对私结算**：收益打入抖音钱包，税筹方案由平台提供。

</td>
</tr>
<tr>
<td>

**收益方（对公）**

</td>
<td>

PGC

</td>
<td>

完成企业认证的企业达人，如明星等公众影响力达人。

</td>
<td>

- **对公结算**：通过结算单、发票流程完成打款。

</td>
</tr>
<tr>
<td>



</td>
<td>

MCN

</td>
<td>

绑定了多个达人账号的机构。

</td>
<td>

- **对公结算**：同上。

</td>
</tr>
<tr>
<td>



</td>
<td>

服务商

</td>
<td>

包括撮合中介和任务服务商，需对公认证并加入白名单。

</td>
<td>

- **对公结算**：同上。

</td>
</tr>
</tbody></table>

---

## 安全与资损防控

为保障资金安全，结算域建立了一套包含技术、流程和监控的资损防控体系。

- **风控拦截**：在达人提现或生成结算单时，若触发套现、虚假交易等风控规则，系统会自动禁止操作。

- **风控处罚**：当发现客户存在黑产行为时，系统会冻结其所有资金，禁止后续任何资金操作。

- **打款/提现管控**：支持通过黑名单或运营配置，禁止向特定用户（如涉及官司的机构/达人、老赖用户）打款或提现。

- **财务/票务审核**：

    - 财务每月对各业务线的消耗与收入进行审计，确保账目匹配。

    - 票务审核团队在打款前对上传的发票信息、合同、金额进行人工核对。

- **离线对账**：通过 T\+1 离线任务，对星图与下游平台的结算记录（订单号、金额、用户信息等）进行对账，不一致则报警。

- **资金追回**：提供在上游业务异常多打款项时的资金追回能力，但前提是资金尚未被用户提现至银行卡或支付宝。

---

## 数据模型与表用途

以下是资金结算域涉及的核心数据表及其用途。

<table><tbody>
<tr>
<td>

库名

</td>
<td>

表名

</td>
<td>

用途说明

</td>
<td>

关键字段说明

</td>
</tr>
<tr>
<td>

`ad\_supplier`

</td>
<td>

`star\_user\_trade\_record`

</td>
<td>

**交易结算记录表**：记录达人/机构的收入和提现明细，是对公结算中“分成明细”的载体。

</td>
<td>

`user\_id`, `order\_id`, `amount`, `op\_type` \(操作类型\), `business\_type` \(业务类型\), `settlement\_status` \(对公结算状态\)。

</td>
</tr>
<tr>
<td>

`ad\_supplier`

</td>
<td>

`star\_corporate\_settlement`

</td>
<td>

**结算单记录表**：记录对公结算单的核心信息，包括金额、状态、关联的 trade\_ids 等。

</td>
<td>

`user\_id`, `statement\_id`, `amount`, `status` \(结算单状态\), `trade\_ids` \(关联的结算记录ID\), `invoice\_status` \(发票状态\)。

</td>
</tr>
<tr>
<td>

`ad\_imce`

</td>
<td>

`imce\_user\_auth\_info\_xingtu`

</td>
<td>

**用户出款信息表**：存储用户的收款账户信息和结算配置。

</td>
<td>

`user\_id`, `settlement\_type` \(1\-对私, 2\-对公\), `account\_type` \(对私是钱包类型, 对公是3M结算配置\), `account\_info` \(对公存储 customer\_id 和合同号\)。

</td>
</tr>
<tr>
<td>

`ad\_imce`

</td>
<td>

`imce\_settlement\_invoice`

</td>
<td>

**支出结算电票表**：存储用户上传的电子发票信息及其审核状态。

</td>
<td>

`user\_id`, `req\_id`, `batch\_id`, `status` \(1\-审核中, 2\-通过, 3\-失败\)。

</td>
</tr>
<tr>
<td>

`ad\_imce`

</td>
<td>

`imce\_user\_trade\_record\_xingtu`

</td>
<td>

**新交易记录表**：需补充（原文描述为“新交易记录表”，但未提供详细用途和字段）。

</td>
<td>

需补充。

</td>
</tr>
</tbody></table>

#### 核心表 SQL 结构

- **`star\_user\_trade\_record`**** \(交易结算记录表\)**

    ```SQL
    CREATE TABLE `star_user_trade_record` (
      `id` bigint(20) unsigned NOT NULL,
      `user_id` bigint(20) unsigned NOT NULL COMMENT '星图ID',
      `order_id` bigint(20) unsigned NOT NULL COMMENT '订单ID',
      `amount` decimal(10,2) unsigned NOT NULL COMMENT '金额',
      `settlement_status` tinyint(4) NOT NULL DEFAULT '0' COMMENT '结算状态, 关联 SettlementStatus 枚举',
      -- ... 其他字段
      PRIMARY KEY (`id`),
      KEY `idx_user_id_op_type_trade_time` (`user_id`,`op_type`,`trade_time`),
      KEY `idx_order_id` (`order_id`),
      KEY `idx_settlement_status` (`settlement_status`)
    ) COMMENT='星图交易记录表';
    
    ```

- **`star\_corporate\_settlement`**** \(结算单记录表\)**

    ```SQL
    CREATE TABLE `star_corporate_settlement` (
      `id` bigint(20) unsigned NOT NULL,
      `user_id` bigint(20) unsigned NOT NULL COMMENT '用户id',
      `statement_id` bigint(20) unsigned NOT NULL COMMENT '结算单id',
      `amount` decimal(10,2) NOT NULL DEFAULT '0.00' COMMENT '结算金额',
      `status` smallint(6) NOT NULL DEFAULT '0' COMMENT '状态, 关联 CorporateSettlementStatus 枚举',
      `trade_ids` mediumtext COMMENT '结算单导出的trade_id',
      `invoice_status` tinyint(4) DEFAULT '1' COMMENT '结算单上传电子发票状态',
      -- ... 其他字段
      PRIMARY KEY (`id`),
      KEY `idx_user_id` (`user_id`)
    ) COMMENT='星图结算单记录表';
    
    ```

# 星图资金结算数据模型知识

## 1\. 领域概览与核心实体

星图资金结算域是星图业务生态的核心后台之一，负责处理所有与资金相关的流入与流出操作。它确保了客户、达人、服务商与平台之间的交易得以准确、及时地结算与支付。本领域的核心目标是保障资金流转的准确性、一致性与安全性，并为财务对账、发票管理、合规审计提供坚实的数据基础。

## 2\. 实体关系图 \(ER Diagram\)

下图展示了资金结算域核心数据表之间的关联关系。

## 3\. 数据模型详解

### 3\.1 `ad\_imce` 库

#### 3\.1\.1 `star\_settlement\_order` \- 支付订单表

- **用途**：记录客户发起的支付意图，是所有资金操作的源头。

- **关联关系**：

    - 主表，其 `id` \(`settlement\_order\_id`\) 被 `star\_settlement\_flow` 表作为外键引用。

- **关键字段含义**：

    - **支付单ID \(biz\_order\_id\)**: 支付单业务ID，由上游透传，bizOrderId\+bizId组成一笔入款

    - **业务类型 \(biz\_id\)**: 标识该笔支付关联的具体业务场景。这是区分不同资金用途的关键字段。定义在ad/star\_idl仓库的idl/settlement/settlement\_origin\.thrift:46

    - **金额 \(Amount\)**: 该次资金操作涉及的金额。单位：分

    - **状态 \(Status\)**: 支付订单的生命周期状态，如 10：待创建；20：进行中；30：已完成；40：已取消

- **关联实体**: 可以关联到支付流水`star\_settlement\_flow`

- **DDL 语句**:

```SQL
CREATE TABLE `star_settlement_order` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '主键',
  `biz_type` int unsigned NOT NULL DEFAULT '0' COMMENT '业务类型',
  `biz_id` int unsigned NOT NULL DEFAULT '0' COMMENT '业务ID',
  `biz_order_id` bigint NOT NULL DEFAULT '0' COMMENT '业务订单id',
  `parent_order_id` bigint NOT NULL DEFAULT '0' COMMENT '父订单id，没有则为0',
  `biz_order_name` varchar(256) COLLATE utf8mb4_general_ci DEFAULT '' COMMENT '业务订单名称',
  `outer_id` varchar(256) COLLATE utf8mb4_general_ci DEFAULT '' COMMENT '外部订单id，如下游3m订单id',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '10：待创建；20：进行中；30：已完成；40：已取消',
  `amount` bigint NOT NULL DEFAULT '0' COMMENT '订单金额，单位分',
  `payer_type` bigint NOT NULL DEFAULT '0' COMMENT '付款人类型',
  `payer_id` bigint NOT NULL DEFAULT '0' COMMENT '付款人的id',
  `payer_info` text COLLATE utf8mb4_general_ci COMMENT '付款人详细信息',
  `payee_type` bigint NOT NULL DEFAULT '0' COMMENT '收款人类型',
  `payee_id` bigint NOT NULL DEFAULT '0' COMMENT '收款人的id',
  `payee_info` text COLLATE utf8mb4_general_ci COMMENT '收款人详细信息',
  `migration_type` int NOT NULL DEFAULT '1' COMMENT '迁移类型。1：新接入；2：存量星图业务迁移',
  `data` text COLLATE utf8mb4_general_ci COMMENT '扩展信息',
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `modify_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted` tinyint NOT NULL DEFAULT '0' COMMENT '是否删除',
  PRIMARY KEY (`id`),
  KEY `idx_biz_order_id_biz_id_biz_type` (`biz_order_id`,`biz_id`,`biz_type`),
  KEY `idx_outer_id` (`outer_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='星图结算订单表'

```

---

#### 3\.1\.2 `star\_settlement\_flow` \- 支付流水表

- **用途**：记录支付订单的每一次资金操作，如支付、退款、扣款等。

- **关键字段含义**：

    - request\_id：请求id，业务幂等id，由上游服务透传

- **关联关系**：

    - 从属于 `star\_settlement\_order`，通过 `settlement\_order\_id` 关联。

- **DDL 语句**:

```SQL
CREATE TABLE `star_settlement_flow` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '主键',
  `settlement_order_id` bigint NOT NULL DEFAULT '0' COMMENT '订单id',
  `operate_type` int NOT NULL DEFAULT '0' COMMENT '操作类型。1：付款；2：扣款; 3:退款',
  `request_id` varchar(256) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '操作id',
  `payer_type` bigint NOT NULL DEFAULT '0' COMMENT '付款人类型',
  `payer_id` bigint NOT NULL DEFAULT '0' COMMENT '付款人的id',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '1：待创建；2：进行中；3：已完成；4：已取消',
  `amount` bigint unsigned NOT NULL DEFAULT '0' COMMENT '金额',
  `display_info` varchar(512) COLLATE utf8mb4_general_ci DEFAULT '' COMMENT '展示信息',
  `callback_url` varchar(512) COLLATE utf8mb4_general_ci DEFAULT '' COMMENT '回调url,仅限付款场景有效',
  `migration_type` int NOT NULL DEFAULT '1' COMMENT '迁移类型。1：新接入；2：存量星图业务迁移',
  `data` text COLLATE utf8mb4_general_ci COMMENT '扩展信息',
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `modify_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted` tinyint NOT NULL DEFAULT '0' COMMENT '是否删除',
  `sync_status` tinyint NOT NULL DEFAULT '1' COMMENT '1：待同步；2：已同步',
  PRIMARY KEY (`id`),
  KEY `idx_settlement_order_id` (`settlement_order_id`),
  KEY `idx_payer_type_payer_id_create_time` (`payer_type`,`payer_id`,`modify_time`),
  KEY `idx_request_id` (`request_id`),
  KEY `idx_status_modify_time` (`status`,`modify_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='星图结算流水表'

```

---

#### 3\.1\.3 `imce\_user\_trade\_record\_xingtu` \- 用户交易记录表

- **用途**：新打款链路中记录用户（达人/MCN/服务商）的每一笔收入与支出，

- **读写频次**：写多次，读多次（用于聚合查询、生成结算单）。

- **关联关系**：`imce\_user\_trade\_record\_xingtu`表中的数据会通过异步消息同步到ad\_supplier库中的star\_user\_trade\_record表中。

- **关键字段含义**：

    - biz\_id: 上游打款的业务类型

    - status：对私打款0\-失败，1\-成功；对公打款0\-未同步，21\-同步成功

    - amount：打款金额，单位分

- **DDL 语句**:

```SQL
CREATE TABLE `imce_user_trade_record_xingtu` (
  `id` bigint unsigned NOT NULL COMMENT '主键ID, 不用',
  `biz_type` int unsigned NOT NULL DEFAULT '0' COMMENT '操作类型，1:打款进钱包,2:提款到用户 & 下游信息',
  `biz_id` int unsigned NOT NULL DEFAULT '0' COMMENT '业务自定义类型',
  `biz_order_id` bigint NOT NULL DEFAULT '0' COMMENT '业务订单ID, 结合biz_type 业务幂等，可重复取消',
  `payee_type` bigint NOT NULL DEFAULT '0' COMMENT '收款人类型，imce用户/star_id/业务方id',
  `payee_id` bigint NOT NULL DEFAULT '0' COMMENT '实名/钱包 收款人账号的id',
  `payee_info` text COLLATE utf8mb4_general_ci COMMENT '收款人详细信息',
  `amount` bigint NOT NULL DEFAULT '0' COMMENT '订单金额，单位分',
  `status` int NOT NULL DEFAULT '0' COMMENT '操作状态，0:初始化，1x:打款子状态,2x:提款子状态，8:取消,9:完成',
  `data` text COLLATE utf8mb4_general_ci COMMENT '业务数据',
  `step_data` text COLLATE utf8mb4_general_ci COMMENT '状态数据,状态变更时记录快照',
  `deleted` tinyint NOT NULL DEFAULT '0' COMMENT '是否删除',
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '新建时间',
  `modify_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  PRIMARY KEY (`id`),
  KEY `idx_biz_order_id_biz_id_biz_type` (`biz_order_id`,`biz_id`,`biz_type`),
  KEY `idx_payee_id_payee_type_biz_id_create_time` (`payee_id`,`payee_type`,`biz_id`,`create_time`),
  KEY `idx_payee_id_payee_type_biz_type_create_time` (`payee_id`,`payee_type`,`biz_type`,`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='交易记录表'

```

---

#### 3\.1\.5 `imce\_settlement\_invoice` \- 电子发票表

- **用途**：记录与财经系统交互的电子发票申请记录。

- **读写频次**：写一次，读多次。

- **关联关系**：

    - 通过 `user\_id` 关联 `imce\_user\_auth\_info\_xingtu`。

    - `req\_id` 作为与外部系统交互的幂等键。

- **DDL 语句**:

```SQL
CREATE TABLE `imce_settlement_invoice` (
  `id` bigint unsigned NOT NULL COMMENT 'id',
  `app_id` int NOT NULL COMMENT 'app_id',
  `user_id` bigint NOT NULL COMMENT '用户ID',
  `req_id` varchar(255) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '请求id',
  `batch_id` varchar(255) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '财经返回批次ID',
  `status` tinyint DEFAULT '1' COMMENT '状态：1，审核中。2，审核通过。3，审核失败',
  `data` text COLLATE utf8mb4_general_ci COMMENT 'data信息',
  `create_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `modify_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted` tinyint DEFAULT '0' COMMENT '删除状态',
  PRIMARY KEY (`id`),
  KEY `idx_user_id_and_app_id` (`user_id`,`app_id`),
  KEY `idx_req_id` (`req_id`),
  KEY `idx_batch_id` (`batch_id`),
  KEY `idx_create_time` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='结算单电子发票发票 options:{"with_DEFAULT": true}'

```

### 3\.2 `ad\_supplier` 库

#### 3\.2\.1 `star\_user\_trade\_record` \- 用户交易记录\&amp;提现表

- **用途**：星图交易记录，主要用于支出侧结算，记录给达人/MCN的打款和野生达人的提现。

- **读写频次**：写多次，读多次。

- **关联关系**：

    - 通过 `user\_id` 关联用户。

    - 通过 `order\_id` 关联上游业务订单。

    - `id` 被 `star\_corporate\_settlement` 的 `trade\_ids` 字段引用。

- **关键字段**：

    - op\_type：操作类型 1\-提现，2\-打款

    - business\_type：业务类型

    ```Python
    *class *TradeRecordBusinessType(ChoiceIntEnum):
        withdraw = 1
        task = 2
        reward = 3
        sync = 4
        plan_task = 5
        mcn_task = 6
        pgc_task = 7
        contract_task = 8
        multi_mcn_task = 10
        provider_task = 11
        prefer_mcn_task = 12
        term_settlement = 13
        trusteeship_plan = 14
        trusteeship_plan_mcn = 15
        component_rewards = 16
        commander_rewards = 17
        activities_rewards = 18
        material_commission = 19
        recruit_provider = 20
        star_ad_union_provider = 21
        match_provider = 22
    
        __choices__ = (
            (withdraw, u'提现'),
            (task, u'任务，对私打款'),
            (reward, u'奖励，已废弃'),
            (sync, u'同步，已废弃'),
            (plan_task, u'繁星计划任务，已废弃'),
            (mcn_task, u'mcn旗下创作者结算任务，mcn机构对公结算'),
            (pgc_task, u'企业创作者结算任务，企业达人对公结算'),
            (contract_task, u'承包投稿任务承包费用，mcn机构对公结算'),
            (multi_mcn_task, u'多mcn分成任务，mcn机构对公结算'),
            (provider_task, u'优选服务商任务，对公结算'),
            (prefer_mcn_task, u'优选mcn任务，对公结算'),
            (term_settlement, u'定期结算'),
            (trusteeship_plan, u'营销托管计划'),
            (trusteeship_plan_mcn, u'营销托管计划'),
            (component_rewards, u'组件奖励'),
            (commander_rewards, u'团长奖励'),
            (activities_rewards, u'活动奖励'),
            (material_commission, u'素材佣金'),
            (recruit_provider, u'招募服务商任务'),
            (star_ad_union_provider, u'星广联投服务商任务'),
            (match_provider, u'撮合中介服务商任务')
        )
    ```

    - source：来源，3\-对公结算，4\-对私结算

    - status：对私结算和提现时使用，3\-成功，1\-失败

    - settlement\_status：对公结算使用，1\-待推送，2\-推送成功，3\-结算中，4\-已结算

- **DDL 语句**:

```SQL
CREATE TABLE `star_user_trade_record` (
  `id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL COMMENT '星图ID',
  `order_id` bigint(20) unsigned NOT NULL COMMENT '订单ID',
  `mcn_id` bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT 'mcn_id',
  `amount` decimal(10,2) unsigned NOT NULL COMMENT '金额',
  `trade_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '交易确认时间',
  `op_type` tinyint(4) NOT NULL DEFAULT '0' COMMENT '操作类型',
  `business_type` tinyint(4) NOT NULL DEFAULT '0' COMMENT '业务类型',
  `source` tinyint(4) NOT NULL DEFAULT '2' COMMENT '来源',
  `status` tinyint(4) NOT NULL DEFAULT '1' COMMENT '状态',
  `settlement_status` tinyint(4) NOT NULL DEFAULT '0' COMMENT '结算状态',
  `data` text COMMENT '数据',
  `push_time` timestamp NOT NULL DEFAULT '1970-01-01 08:01:00' COMMENT '推送时间',
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '新建时间',
  `modify_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted` tinyint(4) NOT NULL DEFAULT '0' COMMENT '是否删除',
  PRIMARY KEY (`id`),
  KEY `idx_user_id_op_type_trade_time` (`user_id`,`op_type`,`trade_time`),
  KEY `idx_order_id` (`order_id`),
  KEY `idx_mcn_id` (`mcn_id`),
  KEY `idx_create_time` (`create_time`) USING BTREE,
  KEY `idx_settlement_status` (`settlement_status`),
  KEY `idx_push_time` (`push_time`),
  KEY `idx_ business_type_source` (`business_type`,`source`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='星图交易记录表'

```

---

#### 3\.2\.2 `star\_corporate\_settlement` \- 对公结算单表

- **用途**：由用户操作，将多笔 `star\_user\_trade\_record` 记录中对公记录聚合为一张对公结算单，用于与企业主体（MCN/服务商）进行定期结算。只适用于对公结算场景，用户会基于对公结算单签署电子签、上传电子发票。

- **读写频次**：写少读多。

- **关联关系**：

    - 通过 `user\_id` 关联收款企业。

    - `trade\_ids` 字段以文本形式存储了所包含的 `star\_user\_trade\_record` 的 `id` 列表。

- **DDL 语句**:

```SQL
CREATE TABLE `star_corporate_settlement` (
  `id` bigint(20) unsigned NOT NULL COMMENT '主键id',
  `user_id` bigint(20) unsigned NOT NULL COMMENT '用户id',
  `statement_id` bigint(20) unsigned NOT NULL COMMENT '结算单id',
  `amount` decimal(10,2) NOT NULL DEFAULT '0.00' COMMENT '结算金额',
  `status` smallint(6) NOT NULL DEFAULT '0' COMMENT '状态',
  `trade_ids` mediumtext COMMENT '结算单导出的trade_id',
  `sign_url` varchar(1024) NOT NULL DEFAULT '' COMMENT '签署链接',
  `sign_url_encrypt` varchar(1024) NOT NULL DEFAULT '' COMMENT '签署链接密文',
  `statement_file_name` varchar(256) NOT NULL DEFAULT '' COMMENT '结算单pdf对应的名称',
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `modify_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted` tinyint(4) NOT NULL DEFAULT '0' COMMENT '0-未删 1-已删',
  `display` int(10) NOT NULL DEFAULT '1' COMMENT '是否星图可见',
  `invoice_status` tinyint(4) DEFAULT '1' COMMENT '结算单上传电子发票状态',
  `biz_type` tinyint(3) unsigned NOT NULL DEFAULT '0' COMMENT 'MMM业务类型 2-mcn,18-pgc,48-招募,49-星广',
  `user_type` tinyint(3) unsigned NOT NULL DEFAULT '0' COMMENT '用户类型 1-pgc,2-mcn,3-优选,4-托管,5-招募,6-星广',
  `data` text COMMENT '数据',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_statement_id` (`statement_id`),
  KEY `idx_create_time` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='星图结算单记录表'

```

---

#### 3\.2\.3 `star\_invoice` \- 发票表

- **用途**：存储客户申请的发票信息，包括抬头、税号、金额、类型等。

- **读写频次**：写一次，读多次。

- **关联关系**：

    - 通过 `user\_id` 关联开票客户。

    - 业务上与 `star\_invoice\_task` 关联，`data` 字段可能存储了任务ID列表。

- **DDL 语句**:

```SQL
CREATE TABLE `star_invoice` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) NOT NULL COMMENT '开票广告主id',
  `serial` varchar(50) NOT NULL DEFAULT '' COMMENT '发票编号（bpm返回）',
  `project_id` tinyint(20) NOT NULL DEFAULT '2' COMMENT '发票项目id，ex：2 广告费，3 广告发布费，8 推广费',
  `invoice_form` tinyint(4) NOT NULL DEFAULT '1' COMMENT '发票形式：1 纸质； 2 电子 ',
  `amount` bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT '发票金额，单位为厘，1/1000元',
  `type` tinyint(4) NOT NULL DEFAULT '1' COMMENT '发票类型：1 增值税普通发票，2 增值税专用发票，3 增值税电子普通发票',
  `title_type` tinyint(4) NOT NULL DEFAULT '1' COMMENT '抬头类型，1 个人， 2 企业',
  `title` varchar(200) NOT NULL DEFAULT '' COMMENT '发票抬头',
  `tax_no` varchar(50) NOT NULL DEFAULT '' COMMENT '税号',
  `customer_address` varchar(50) NOT NULL DEFAULT '' COMMENT '企业地址',
  `customer_phone` varchar(30) NOT NULL COMMENT '联系电话',
  `customer_email` varchar(50) DEFAULT NULL COMMENT '电子邮箱',
  `bank` varchar(200) NOT NULL DEFAULT '' COMMENT '开户银行',
  `bank_account` varchar(30) NOT NULL DEFAULT '' COMMENT '银行账号',
  `status` tinyint(4) NOT NULL COMMENT '状态,  0=作废，1=未提交审批，2=审批中，3=审批通过，4=已开票',
  `data` text COMMENT '订单id列表，包含备注和发票接收方地址信息(如果有的话)',
  `apply_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `modify_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `customer_phone_encrypt` varchar(800) NOT NULL DEFAULT '' COMMENT '联系人电话加密',
  PRIMARY KEY (`id`),
  KEY `idx_user_id_and_apply_time` (`user_id`,`apply_time`),
  KEY `idx_serial` (`serial`)
) ENGINE=InnoDB AUTO_INCREMENT=7574343281690017837 DEFAULT CHARSET=utf8mb4

```

#### 
