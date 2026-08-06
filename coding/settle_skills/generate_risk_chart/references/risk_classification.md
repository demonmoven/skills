# 结算计费域风险分类规则

## 风险点类别

| 风险点类别 | 风险点说明 | 适用场景 |
|-----------|-----------|---------|
| 配置类风险 | 配置延迟、配置错误、配置过期 | 规则合约相关 |
| 时效性风险 | 数据延迟、业务处理延迟 | 业务卡单相关 |
| 一致性风险 | 上下游一致、内外部幂等、在离线一致 | 端内外核对 |
| 业务正确性风险 | 参与对象/资金流/余额/金额正确性 | 资金逻辑相关 |

## 分类优先级

当一个任务名称同时匹配多个风险类别时，按以下优先级取第一个匹配：

**配置类风险 > 时效性风险 > 一致性风险 > 业务正确性风险**

## 关键词匹配规则

### 配置类风险关键词
```
规则, 配置, 合约, 有效期, 过期,
fee_activity, template_aggregation, settle_rule,
charge_rule, settle_code, factor_match, settle_template,
收费规则, 计费规则, 商户结算信息
```

### 时效性风险关键词
```
卡单, 延迟, 超时, 未终态, 处理中, 时效,
recovery, 首笔发现, 首笔监控, 首笔命中
```

### 一致性风险关键词
```
幂等, 一致性, vs, VS, entry汇总, 核对, 重复
```

### 业务正确性风险关键词
```
金额, 退费, 资金流, 资金平衡, 不大于, 不能超过,
超限, 约束, 正确性, 合法性, 唯一性, 参数校验,
不能有, 资金明细, 必须, 只能
```

## 依赖表提取映射

任务名称中包含以下关键词时，映射到对应数据表：

| 关键词 | 对应表名 |
|--------|---------|
| settle_order | bytepay_settle_order |
| settle_voucher | bytepay_settle_voucher |
| settle_entry | bytepay_settle_entry |
| settle_request | settle_request |
| settle_center | settle_center_order |
| charge_order | bytepay_charge_order |
| charge_detail_entry | bytepay_charge_detail_entry |
| charge_sharding_voucher | bytepay_charge_sharding_voucher |
| charge_voucher | bytepay_charge_sharding_voucher |
| settle_rule | bytepay_settle_rule |
| settle_code | bytepay_settle_code |
| settle_template | bytepay_settle_template |
| merchant_settle_info | merchant_settle_info |
| factor_match | factor_match |
| fee_activity_rule | fee_activity_rule |
| fee_activity_config | fee_activity_config |
| template_aggregation | template_aggregation |
| charge_rule_info | charge_rule_info |
| cycle_task | cycle_task |
| cycle_result | cycle_result |
| fund_clause | fund_clause |
| pay_order | pay_order |
| settle_sharding_set | bytepay_settle_sharding_set |

## 业务分类说明（参考）

以下业务分类为人工标注字段的参考值：

**结算关联业务**：
- 端内普通收单
- 端内普通担保
- 端外普通收单
- 端外间联业务
- 端外分账
- 重点商户保障
- 归档幂等保障
- 端内月付贴息
- 端外月付贴息

**计费关联业务**：
- 交易明细计费
- 周期收费
- 退款退费

## 数据来源

- 飞书文档：《结算计费域规则分类》
- 链接：https://bytedance.larkoffice.com/wiki/JF0nwpzD4ihPPdkq4OUc9agtn7b
- 白皮书：《资损风险分析白皮书》
