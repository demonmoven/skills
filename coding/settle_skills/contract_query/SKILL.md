---
name: contract-query
description: 查询端外商户合约信息。当用户需要查询商户合约信息时触发。触发词：查询合约、查询合约信息。
---

## 执行步骤

### 步骤 1：获取 JWT
使用 bytedance-jwt 技能获取 JWT Token。

### 步骤2: 查询商户合约信息

```bash
python3 <SKILL路径>/scripts/PageQueryCommercialProductRecords.py --merchant_id <MERCHANT_ID> --jwt <JWT>
```
参数说明：

- `merchant_id`：商户 ID，必填。
- `jwt`：JWT Token cookie value，必填。 使用 bytedance-jwt 技能获取。

## 完整示例

### 场景：查询6020107510750703的合约信息

**用户：** "帮我查询6020241010750703的合约信息"

**执行流程：**

1. 使用 bytedance-jwt 技能获取 JWT Token。
2. 确定 merchant_id为6020241010750703。
3. 调用 contract-query 技能，查询合约信息。
