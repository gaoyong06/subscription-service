# 订阅服务测试指南

## 快速开始

### 1. 启动服务

```bash
# 编译所有服务
make build-all

# 运行主服务（在一个终端）
make run

# 或者同时运行主服务和 cron 服务
make run-all
```

### 2. 运行 API 测试

```bash
make test
```

这将使用 `api-tester` 运行 `api-test-config.yaml` 中定义的所有测试场景。

## 测试场景

### 场景 1: 基础功能测试
- 获取套餐列表

### 场景 2: 多区域定价测试
测试不同区域的定价策略：
- **默认区域** (default/US): $9.99 USD
- **中国区域** (CN): ¥38.00 CNY
- **欧洲区域** (EU): €8.99 EUR
- **未知区域** (XX): 回退到 $9.99 USD

### 场景 3: 订阅生命周期测试
完整的订阅流程：
1. 创建订单（中国区域）
2. 模拟支付成功
3. 获取订阅状态（验证激活）
4. 取消订阅
5. 验证取消后状态

## 测试报告

测试完成后，报告会生成在 `./reports` 目录。

## 手动测试

如果需要手动测试特定接口：

### 获取套餐列表
```bash
curl http://localhost:8000/v1/subscription/plans
```

### 创建订单（中国区域）
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

### 创建订单（欧洲区域）
```bash
curl -X POST http://localhost:8000/v1/subscription/order \
  -H "Content-Type: application/json" \
  -d '{
    "uid": 1002,
    "plan_id": "plan_monthly",
    "payment_method": "alipay",
    "region": "EU"
  }'
```

### 创建订单（默认区域）
```bash
curl -X POST http://localhost:8000/v1/subscription/order \
  -H "Content-Type: application/json" \
  -d '{
    "uid": 1003,
    "plan_id": "plan_monthly",
    "payment_method": "alipay",
    "region": "US"
  }'
```

## 验证数据库

### 查看套餐
```sql
SELECT * FROM plans;
```

### 查看区域定价
```sql
SELECT * FROM plan_pricings ORDER BY plan_id, region;
```

预期结果：只有 CN 和 EU 两个区域的配置。

## 故障排查

### 服务无法启动
1. 检查 MySQL 是否运行
2. 检查 Redis 是否运行（如果配置了）
3. 检查配置文件 `configs/config.yaml`

### 测试失败
1. 确保服务已启动：`curl http://localhost:8000/v1/subscription/plans`
2. 检查数据库是否已初始化（包含 plan 和 plan_pricing 数据）
3. 查看服务日志

### 价格不正确
1. 检查数据库中的 `plan_pricings` 表
2. 确认请求中的 `region` 参数正确
3. 查看服务日志中的 `GetPlanPricing` 调用

## 相关文档

- [多区域定价测试指南](./MULTI_REGION_PRICING_TEST.md) - 详细的定价策略说明
- [重构总结](./REFACTORING_SUMMARY.md) - 项目重构历史
- [错误处理文档](./ERROR_HANDLING.md) - 错误码和 i18n 说明
