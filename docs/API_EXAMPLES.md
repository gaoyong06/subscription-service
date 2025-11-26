# Subscription Service API 使用示例

## 订阅购买流程

### 完整流程示例

```
1. 用户浏览套餐列表
   ↓
2. 选择套餐并创建订单
   ↓
3. 跳转到支付页面
   ↓
4. 用户完成支付
   ↓
5. Payment Service 回调 Subscription Service
   ↓
6. 订阅激活，用户获得会员权益
```

### 1. 获取套餐列表

用户在购买前需要先查看可用的订阅套餐。

**HTTP 请求**:
```bash
curl -X GET http://localhost:8102/v1/subscription/plans \
  -H "Content-Type: application/json"
```

**响应示例**:
```json
{
  "success": true,
  "data": {
    "plans": [
      {
        "id": "plan_monthly",
        "name": "Pro Monthly",
        "description": "Pro features for 1 month",
        "price": 9.99,
        "currency": "CNY",
        "duration_days": 30,
        "type": "pro"
      },
      {
        "id": "plan_yearly",
        "name": "Pro Yearly",
        "description": "Pro features for 1 year",
        "price": 99.99,
        "currency": "CNY",
        "duration_days": 365,
        "type": "pro"
      }
    ]
  }
}
```

**gRPC 调用示例**:
```go
conn, _ := grpc.Dial("localhost:9102", grpc.WithInsecure())
client := subscriptionv1.NewSubscriptionClient(conn)

resp, err := client.ListPlans(context.Background(), &subscriptionv1.ListPlansRequest{})
if err != nil {
    log.Fatalf("failed to list plans: %v", err)
}

for _, plan := range resp.Plans {
    fmt.Printf("套餐: %s\n", plan.Name)
    fmt.Printf("  价格: %.2f %s\n", plan.Price, plan.Currency)
    fmt.Printf("  时长: %d 天\n", plan.DurationDays)
    fmt.Printf("  类型: %s\n\n", plan.Type)
}
```

### 2. 查询用户当前订阅状态

在购买前，可以先查询用户是否已有订阅。

**HTTP 请求**:
```bash
curl -X GET http://localhost:8102/v1/subscription/my/1001 \
  -H "Content-Type: application/json"
```

**响应示例（未订阅）**:
```json
{
  "success": true,
  "data": {
    "is_active": false,
    "plan_id": "",
    "start_time": 0,
    "end_time": 0,
    "status": "expired"
  }
}
```

**响应示例（已订阅）**:
```json
{
  "success": true,
  "data": {
    "is_active": true,
    "plan_id": "plan_monthly",
    "start_time": 1700726400,
    "end_time": 1703318400,
    "status": "active"
  }
}
```

**gRPC 调用示例**:
```go
resp, err := client.GetMySubscription(context.Background(), &subscriptionv1.GetMySubscriptionRequest{
    Uid: 1001,
})
if err != nil {
    log.Fatalf("failed to get subscription: %v", err)
}

if resp.IsActive {
    endTime := time.Unix(resp.EndTime, 0)
    fmt.Printf("会员有效期至: %s\n", endTime.Format("2006-01-02 15:04:05"))
    fmt.Printf("当前套餐: %s\n", resp.PlanId)
} else {
    fmt.Println("当前不是会员")
}
```

### 3. 创建订阅订单

用户选择套餐后，创建订阅订单。

**HTTP 请求**:
```bash
curl -X POST http://localhost:8102/v1/subscription/order \
  -H "Content-Type: application/json" \
  -d '{
    "uid": 1001,
    "plan_id": "plan_monthly",
    "payment_method": "alipay"
  }'
```

**响应示例**:
```json
{
  "success": true,
  "data": {
    "order_id": "SUB1700726400001001",
    "payment_id": "PAY1700726400001",
    "pay_url": "https://openapi.alipay.com/gateway.do?...",
    "pay_code": "",
    "pay_params": ""
  }
}
```

**gRPC 调用示例**:
```go
resp, err := client.CreateSubscriptionOrder(context.Background(), &subscriptionv1.CreateSubscriptionOrderRequest{
    Uid:           1001,
    PlanId:        "plan_monthly",
    PaymentMethod: "alipay",
})
if err != nil {
    log.Fatalf("failed to create order: %v", err)
}

fmt.Printf("订单号: %s\n", resp.OrderId)
fmt.Printf("支付链接: %s\n", resp.PayUrl)

// 引导用户到支付页面
// window.location.href = resp.PayUrl
```

### 4. 处理支付成功回调

支付成功后，Payment Service 会回调 Subscription Service。

**HTTP 请求**:
```bash
curl -X POST http://localhost:8102/v1/subscription/payment/success \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "SUB1700726400001001",
    "payment_id": "PAY1700726400001",
    "amount": 9.99
  }'
```

**响应示例**:
```json
{
  "success": true,
  "data": {
    "success": true
  }
}
```

**gRPC 调用示例**:
```go
// 通常由 Payment Service 或 MQ 消费者调用
resp, err := client.HandlePaymentSuccess(context.Background(), &subscriptionv1.HandlePaymentSuccessRequest{
    OrderId:   "SUB1700726400001001",
    PaymentId: "PAY1700726400001",
    Amount:    9.99,
})
if err != nil {
    log.Fatalf("failed to handle payment success: %v", err)
}

if resp.Success {
    fmt.Println("订阅激活成功")
}
```

### 5. 验证订阅已激活

支付成功后，再次查询用户订阅状态，确认已激活。

**HTTP 请求**:
```bash
curl -X GET http://localhost:8102/v1/subscription/my/1001
```

**响应示例**:
```json
{
  "success": true,
  "data": {
    "is_active": true,
    "plan_id": "plan_monthly",
    "start_time": 1700726400,
    "end_time": 1703318400,
    "status": "active"
  }
}
```

## 订阅续费流程

### 场景说明

用户已有订阅（未过期或已过期），希望续费延长订阅时间。

### 续费逻辑

- **未过期续费**: 在当前有效期基础上延长
- **已过期续费**: 从续费时间开始重新计算

### 续费示例

**1. 创建续费订单**:
```bash
curl -X POST http://localhost:8102/v1/subscription/order \
  -H "Content-Type: application/json" \
  -d '{
    "uid": 1001,
    "plan_id": "plan_monthly",
    "payment_method": "alipay"
  }'
```

**2. 支付成功后**:
```bash
curl -X POST http://localhost:8102/v1/subscription/payment/success \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "SUB1700726400002001",
    "payment_id": "PAY1700726400002",
    "amount": 9.99
  }'
```

**3. 验证订阅已延长**:
```bash
curl -X GET http://localhost:8102/v1/subscription/my/1001
```

**响应示例**:
```json
{
  "success": true,
  "data": {
    "is_active": true,
    "plan_id": "plan_monthly",
    "start_time": 1700726400,
    "end_time": 1705910400,
    "status": "active"
  }
}
```

## 订阅升级流程

### 场景说明

用户当前是月度会员，希望升级到年度会员。

### 升级示例

**1. 创建升级订单**:
```bash
curl -X POST http://localhost:8102/v1/subscription/order \
  -H "Content-Type: application/json" \
  -d '{
    "uid": 1001,
    "plan_id": "plan_yearly",
    "payment_method": "wechatpay"
  }'
```

**2. 支付成功后**:
```bash
curl -X POST http://localhost:8102/v1/subscription/payment/success \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "SUB1700726400003001",
    "payment_id": "PAY1700726400003",
    "amount": 99.99
  }'
```

**3. 验证已升级**:
```bash
curl -X GET http://localhost:8102/v1/subscription/my/1001
```

**响应示例**:
```json
{
  "success": true,
  "data": {
    "is_active": true,
    "plan_id": "plan_yearly",
    "start_time": 1700726400,
    "end_time": 1732262400,
    "status": "active"
  }
}
```

## 前端集成示例

### React/Vue 示例

```javascript
// 订阅服务 API 封装
class SubscriptionService {
  constructor(baseURL = 'http://localhost:8102') {
    this.baseURL = baseURL;
  }

  // 获取套餐列表
  async getPlans() {
    const response = await fetch(`${this.baseURL}/v1/subscription/plans`);
    const data = await response.json();
    return data.data.plans;
  }

  // 获取用户订阅状态
  async getMySubscription(uid) {
    const response = await fetch(`${this.baseURL}/v1/subscription/my/${uid}`);
    const data = await response.json();
    return data.data;
  }

  // 创建订阅订单
  async createOrder(uid, planId, paymentMethod) {
    const response = await fetch(`${this.baseURL}/v1/subscription/order`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        uid,
        plan_id: planId,
        payment_method: paymentMethod
      })
    });
    const data = await response.json();
    return data.data;
  }
}

// 使用示例
const subscriptionService = new SubscriptionService();

// 1. 显示套餐列表
async function showPlans() {
  const plans = await subscriptionService.getPlans();
  plans.forEach(plan => {
    console.log(`${plan.name}: ¥${plan.price}/${plan.duration_days}天`);
  });
}

// 2. 检查订阅状态
async function checkSubscription(uid) {
  const subscription = await subscriptionService.getMySubscription(uid);
  if (subscription.is_active) {
    const endDate = new Date(subscription.end_time * 1000);
    console.log(`会员有效期至: ${endDate.toLocaleDateString()}`);
  } else {
    console.log('当前不是会员');
  }
}

// 3. 购买订阅
async function purchaseSubscription(uid, planId) {
  const order = await subscriptionService.createOrder(uid, planId, 'alipay');
  // 跳转到支付页面
  window.location.href = order.pay_url;
}
```

## 数据库查询示例

### 查询订阅信息

```sql
-- 查询用户订阅状态
SELECT * FROM user_subscription WHERE user_id = 1001;

-- 查询所有激活的订阅
SELECT * FROM user_subscription WHERE status = 'active';

-- 查询即将过期的订阅（7天内）
SELECT * FROM user_subscription 
WHERE status = 'active' 
  AND end_time BETWEEN NOW() AND DATE_ADD(NOW(), INTERVAL 7 DAY);
```

### 查询订单信息

```sql
-- 查询用户的订单历史
SELECT * FROM subscription_order 
WHERE user_id = 1001 
ORDER BY created_at DESC;

-- 查询待支付订单
SELECT * FROM subscription_order 
WHERE payment_status = 'pending';

-- 统计订单金额
SELECT 
  DATE(created_at) as date,
  COUNT(*) as order_count,
  SUM(amount) as total_amount
FROM subscription_order
WHERE payment_status = 'paid'
GROUP BY DATE(created_at)
ORDER BY date DESC;
```

### 统计分析

```sql
-- 按套餐统计销售情况
SELECT 
  plan_id,
  COUNT(*) as order_count,
  SUM(amount) as total_revenue
FROM subscription_order
WHERE payment_status = 'paid'
GROUP BY plan_id;

-- 计算续费率
SELECT 
  COUNT(DISTINCT user_id) as total_users,
  COUNT(CASE WHEN order_count > 1 THEN 1 END) as renewed_users,
  ROUND(COUNT(CASE WHEN order_count > 1 THEN 1 END) * 100.0 / COUNT(DISTINCT user_id), 2) as renewal_rate
FROM (
  SELECT user_id, COUNT(*) as order_count
  FROM subscription_order
  WHERE payment_status = 'paid'
  GROUP BY user_id
) as user_orders;

-- 查询活跃会员数
SELECT COUNT(*) as active_members
FROM user_subscription
WHERE status = 'active';
```

## 测试建议

### 单元测试

```go
func TestListPlans(t *testing.T) {
    // 准备测试数据
    ctx := context.Background()
    
    // 调用方法
    resp, err := subscriptionUsecase.ListPlans(ctx)
    
    // 验证结果
    assert.NoError(t, err)
    assert.NotNil(t, resp)
    assert.GreaterOrEqual(t, len(resp.Plans), 2) // 至少有月度和年度套餐
}

func TestCreateSubscriptionOrder(t *testing.T) {
    // 准备测试数据
    ctx := context.Background()
    uid := uint64(1001)
    planID := "plan_monthly"
    paymentMethod := "alipay"
    
    // 调用方法
    order, err := subscriptionUsecase.CreateSubscriptionOrder(ctx, uid, planID, paymentMethod)
    
    // 验证结果
    assert.NoError(t, err)
    assert.NotEmpty(t, order.OrderID)
    assert.NotEmpty(t, order.PaymentID)
    assert.NotEmpty(t, order.PayURL)
}

func TestHandlePaymentSuccess(t *testing.T) {
    // 先创建订单
    ctx := context.Background()
    order, _ := subscriptionUsecase.CreateSubscriptionOrder(ctx, 1001, "plan_monthly", "alipay")
    
    // 处理支付成功
    err := subscriptionUsecase.HandlePaymentSuccess(ctx, order.OrderID, order.PaymentID, 9.99)
    
    // 验证结果
    assert.NoError(t, err)
    
    // 验证订阅已激活
    subscription, _ := subscriptionUsecase.GetMySubscription(ctx, 1001)
    assert.True(t, subscription.IsActive)
    assert.Equal(t, "plan_monthly", subscription.PlanID)
}
```

### 集成测试

使用 API 测试工具（如 api-tester）测试完整流程：

1. 获取套餐列表
2. 查询用户订阅状态（未订阅）
3. 创建订阅订单
4. 处理支付成功
5. 验证订阅已激活
6. 创建续费订单
7. 处理续费支付成功
8. 验证订阅已延长

## 常见问题

### Q: 如何处理支付失败？
A: 订单状态会保持为 `pending`，用户可以重新发起支付。建议设置订单过期时间（如 30 分钟）。

### Q: 如何处理重复支付回调？
A: 系统应该实现幂等性，同一个订单多次回调只会处理一次。可以通过订单状态判断。

### Q: 订阅过期后会自动降级吗？
A: 当前版本不会自动处理，需要业务服务在调用时检查订阅状态。建议实现定时任务定期检查并处理过期订阅。

### Q: 如何实现订阅暂停？
A: 当前版本不支持暂停功能。可以扩展 `user_subscription` 表添加 `paused` 状态，并实现相应的暂停/恢复接口。

### Q: 如何处理退款？
A: 退款由 Payment Service 处理。Subscription Service 需要实现退款回调接口，将订阅状态设置为已取消。

