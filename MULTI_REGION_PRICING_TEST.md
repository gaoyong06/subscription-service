# 多区域定价测试指南

## 概述

本文档说明如何测试 `subscription-service` 的多区域定价功能。

## 定价策略

我们采用**简化的多区域定价策略**：

| 区域 | 货币 | 月度价格 | 季度价格 | 年度价格 | 说明 |
|------|------|----------|----------|----------|------|
| **CN (中国)** | CNY | ¥38 | ¥98 | ¥388 | 本地化定价，约为美元价格的 40% 折扣 |
| **EU (欧洲)** | EUR | €8.99 | €23.99 | €89.99 | 欧元定价 |
| **Default (其他)** | USD | $9.99 | $25.99 | $99.99 | 默认美元价格，覆盖 US 及所有未配置区域 |

### 设计理念

1. **只配置关键市场**：仅为中国和欧洲配置特殊定价
2. **默认回退机制**：未配置的区域自动使用 USD 默认价格
3. **本地化体验**：中国用户看到人民币，欧洲用户看到欧元，心理门槛更低

## 测试场景

### 场景 1：中国区域定价

**请求**：
```bash
curl -X POST http://localhost:8000/v1/subscription/order \
  -H "Content-Type: application/json" \
  -d '{
    "uid": 1001,
    "plan_id": "plan_monthly",
    "payment_method": "wechatpay",
    "region": "CN"
  }'
```

**预期结果**：
- 订单金额：¥38.00 CNY
- 货币：CNY
- 支付方式：wechatpay

### 场景 2：欧洲区域定价

**请求**：
```bash
curl -X POST http://localhost:8000/v1/subscription/order \
  -H "Content-Type: application/json" \
  -d '{
    "uid": 1002,
    "plan_id": "plan_yearly",
    "payment_method": "alipay",
    "region": "EU"
  }'
```

**预期结果**：
- 订单金额：€89.99 EUR
- 货币：EUR

### 场景 3：美国区域（默认定价）

**请求**：
```bash
curl -X POST http://localhost:8000/v1/subscription/order \
  -H "Content-Type: application/json" \
  -d '{
    "uid": 1003,
    "plan_id": "plan_quarterly",
    "payment_method": "alipay",
    "region": "US"
  }'
```

**预期结果**：
- 订单金额：$25.99 USD
- 货币：USD
- 说明：US 区域没有特殊配置，使用 plan 表的默认价格

### 场景 4：未知区域（回退到默认）

**请求**：
```bash
curl -X POST http://localhost:8000/v1/subscription/order \
  -H "Content-Type: application/json" \
  -d '{
    "uid": 1004,
    "plan_id": "plan_monthly",
    "payment_method": "alipay",
    "region": "JP"
  }'
```

**预期结果**：
- 订单金额：$9.99 USD
- 货币：USD
- 说明：JP (日本) 没有配置，自动回退到默认 USD 价格

### 场景 5：空区域参数（默认）

**请求**：
```bash
curl -X POST http://localhost:8000/v1/subscription/order \
  -H "Content-Type: application/json" \
  -d '{
    "uid": 1005,
    "plan_id": "plan_monthly",
    "payment_method": "alipay"
  }'
```

**预期结果**：
- 订单金额：$9.99 USD
- 货币：USD
- 说明：未传 region 参数，Service 层默认使用 "default"

## 使用 API-Tester 运行测试

### 前置条件

1. 安装 `api-tester`（如果尚未安装）
2. 确保服务运行在 `http://localhost:8000`
3. 数据库已初始化（包含 plan 和 plan_pricing 数据）

### 运行测试

```bash
cd /Users/gaoyong/Documents/work/xinyuan_tech/subscription-service
api-tester run -c api-test-config.yaml
```

### 查看测试报告

测试完成后，报告会生成在 `./reports` 目录。

## 验证数据库

### 查看套餐表

```sql
SELECT * FROM plans;
```

预期输出：
```
+----------------+--------------+---------------------------+-------+----------+---------------+------+
| id             | name         | description               | price | currency | duration_days | type |
+----------------+--------------+---------------------------+-------+----------+---------------+------+
| plan_monthly   | Pro Monthly  | Pro features for 1 month  | 9.99  | USD      | 30            | pro  |
| plan_yearly    | Pro Yearly   | Pro features for 1 year   | 99.99 | USD      | 365           | pro  |
| plan_quarterly | Pro Quarterly| Pro features for 3 months | 25.99 | USD      | 90            | pro  |
+----------------+--------------+---------------------------+-------+----------+---------------+------+
```

### 查看区域定价表

```sql
SELECT * FROM plan_pricings ORDER BY plan_id, region;
```

预期输出：
```
+----+----------------+--------+--------+----------+
| id | plan_id        | region | price  | currency |
+----+----------------+--------+--------+----------+
| 1  | plan_monthly   | CN     | 38.00  | CNY      |
| 2  | plan_monthly   | EU     | 8.99   | EUR      |
| 3  | plan_quarterly | CN     | 98.00  | CNY      |
| 4  | plan_quarterly | EU     | 23.99  | EUR      |
| 5  | plan_yearly    | CN     | 388.00 | CNY      |
| 6  | plan_yearly    | EU     | 89.99  | EUR      |
+----+----------------+--------+--------+----------+
```

**注意**：没有 US 或其他区域的记录，这是正确的！

## 代码逻辑验证

### GetPlanPricing 函数

位置：`internal/biz/plan.go`

```go
func (uc *SubscriptionUsecase) GetPlanPricing(ctx context.Context, planID, region string) (*PlanPricing, error) {
    pricing, err := uc.planRepo.GetPlanPricing(ctx, planID, region)
    if err != nil || pricing == nil {
        // 如果没有找到区域定价，返回默认价格
        plan, err := uc.planRepo.GetPlan(ctx, planID)
        if err != nil {
            return nil, err
        }
        return &PlanPricing{
            PlanID:   plan.ID,
            Region:   "default",
            Price:    plan.Price,
            Currency: plan.Currency,
        }, nil
    }
    return pricing, nil
}
```

**关键点**：
1. 先尝试查找 `plan_pricings` 表中的区域定价
2. 如果找不到（`err != nil || pricing == nil`），回退到 `plans` 表的默认价格
3. 这保证了任何区域都能获得价格，不会返回错误

## 常见问题

### Q1: 为什么不为美国单独配置？

**A**: 美国是我们的默认市场，使用 USD 作为基准货币。在 `plans` 表中已经有 USD 价格，无需在 `plan_pricings` 中重复配置。这样可以减少数据冗余。

### Q2: 如果要添加新的区域怎么办？

**A**: 只需在 `plan_pricings` 表中插入新记录即可：

```sql
INSERT INTO plan_pricings (plan_id, region, price, currency) VALUES
('plan_monthly', 'IN', 299.00, 'INR'),
('plan_yearly', 'IN', 2999.00, 'INR'),
('plan_quarterly', 'IN', 799.00, 'INR');
```

### Q3: 中国价格为什么这么低？

**A**: 
1. **购买力平价 (PPP)**：中国的人均收入低于美国，需要调整价格
2. **市场策略**：中国是我们的核心市场，低价有助于快速获客
3. **竞争环境**：中国 SaaS 市场价格普遍较低

### Q4: 如何动态调整价格？

**A**: 直接更新数据库即可，无需重启服务：

```sql
UPDATE plan_pricings 
SET price = 48.00 
WHERE plan_id = 'plan_monthly' AND region = 'CN';
```

## 性能考虑

- **Redis 缓存**：我们已经为 `GetSubscription` 实现了 Redis 缓存
- **未来优化**：可以考虑为 `GetPlanPricing` 也添加缓存，因为价格变动频率很低

## 总结

我们的多区域定价系统：
- ✅ **简洁**：只配置关键市场（CN, EU）
- ✅ **灵活**：可以随时添加新区域
- ✅ **健壮**：未配置区域自动回退到默认价格
- ✅ **本地化**：支持本地货币，提升转化率
- ✅ **可维护**：数据驱动，无需修改代码
