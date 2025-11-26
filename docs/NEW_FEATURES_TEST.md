# 新功能测试文档

## 测试环境
- 服务地址: http://localhost:8102
- 数据库: subscription_service
- 测试用户: 1001, 1002

## 功能测试

### 1. 取消订阅 (CancelSubscription)

**API**: `POST /v1/subscription/cancel`

**请求示例**:
```bash
curl -X POST http://localhost:8102/v1/subscription/cancel \
  -H "Content-Type: application/json" \
  -d '{
    "uid": 1001,
    "reason": "不再需要此服务"
  }'
```

**成功响应**:
```json
{
  "success": true,
  "data": {
    "success": true,
    "message": "Subscription cancelled successfully"
  }
}
```

**错误场景**:
- 用户没有订阅: `no active subscription found`
- 订阅状态不允许取消: `cannot cancel subscription with status: xxx`

### 2. 暂停订阅 (PauseSubscription)

**API**: `POST /v1/subscription/pause`

**请求示例**:
```bash
curl -X POST http://localhost:8102/v1/subscription/pause \
  -H "Content-Type: application/json" \
  -d '{
    "uid": 1001,
    "reason": "临时不使用"
  }'
```

**成功响应**:
```json
{
  "success": true,
  "data": {
    "success": true,
    "message": "Subscription paused successfully"
  }
}
```

**错误场景**:
- 只能暂停 active 状态的订阅
- 用户没有订阅: `no active subscription found`

### 3. 恢复订阅 (ResumeSubscription)

**API**: `POST /v1/subscription/resume`

**请求示例**:
```bash
curl -X POST http://localhost:8102/v1/subscription/resume \
  -H "Content-Type: application/json" \
  -d '{
    "uid": 1001
  }'
```

**成功响应**:
```json
{
  "success": true,
  "data": {
    "success": true,
    "message": "Subscription resumed successfully"
  }
}
```

**错误场景**:
- 只能恢复 paused 状态的订阅
- 用户没有订阅: `no subscription found`

### 4. 获取订阅历史 (GetSubscriptionHistory)

**API**: `GET /v1/subscription/history/{uid}`

**请求示例**:
```bash
curl -X GET "http://localhost:8102/v1/subscription/history/1001?page=1&page_size=10"
```

**成功响应**:
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 1,
        "planId": "plan_monthly",
        "planName": "Pro Monthly",
        "startTime": 1732608000,
        "endTime": 1735200000,
        "status": "active",
        "action": "created",
        "createdAt": 1732608000
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 10
  }
}
```

**参数说明**:
- `page`: 页码，从1开始，默认1
- `page_size`: 每页数量，默认10，最大100

### 5. 设置自动续费 (SetAutoRenew)

**API**: `POST /v1/subscription/auto-renew`

**开启自动续费**:
```bash
curl -X POST http://localhost:8102/v1/subscription/auto-renew \
  -H "Content-Type: application/json" \
  -d '{
    "uid": 1001,
    "auto_renew": true
  }'
```

**关闭自动续费**:
```bash
curl -X POST http://localhost:8102/v1/subscription/auto-renew \
  -H "Content-Type: application/json" \
  -d '{
    "uid": 1001,
    "auto_renew": false
  }'
```

**成功响应**:
```json
{
  "success": true,
  "data": {
    "success": true,
    "message": "Auto-renew enabled successfully"
  }
}
```

**错误场景**:
- 只有 active 状态的订阅才能设置自动续费
- 用户没有订阅: `no subscription found`

### 6. 查询订阅状态（已更新）

**API**: `GET /v1/subscription/my/{uid}`

**响应示例**（包含新增的 auto_renew 字段）:
```json
{
  "success": true,
  "data": {
    "isActive": true,
    "planId": "plan_monthly",
    "startTime": 1732608000,
    "endTime": 1735200000,
    "status": "active",
    "autoRenew": true
  }
}
```

**状态说明**:
- `active`: 活跃订阅
- `expired`: 已过期
- `paused`: 已暂停
- `cancelled`: 已取消

## 完整测试流程

### 流程1: 订阅生命周期测试

```bash
# 1. 创建订阅
curl -X POST http://localhost:8102/v1/subscription/order \
  -H "Content-Type: application/json" \
  -d '{"uid": 1001, "plan_id": "plan_monthly", "payment_method": "alipay"}'

# 2. 模拟支付成功
curl -X POST http://localhost:8102/v1/subscription/payment/success \
  -H "Content-Type: application/json" \
  -d '{"order_id": "SUB...", "payment_id": "PAY...", "amount": 9.99}'

# 3. 查询订阅状态
curl http://localhost:8102/v1/subscription/my/1001

# 4. 开启自动续费
curl -X POST http://localhost:8102/v1/subscription/auto-renew \
  -H "Content-Type: application/json" \
  -d '{"uid": 1001, "auto_renew": true}'

# 5. 暂停订阅
curl -X POST http://localhost:8102/v1/subscription/pause \
  -H "Content-Type: application/json" \
  -d '{"uid": 1001, "reason": "暂时不用"}'

# 6. 恢复订阅
curl -X POST http://localhost:8102/v1/subscription/resume \
  -H "Content-Type: application/json" \
  -d '{"uid": 1001}'

# 7. 查看历史记录
curl "http://localhost:8102/v1/subscription/history/1001?page=1&page_size=10"

# 8. 取消订阅
curl -X POST http://localhost:8102/v1/subscription/cancel \
  -H "Content-Type: application/json" \
  -d '{"uid": 1001, "reason": "不再需要"}'
```

### 流程2: 异常场景测试

```bash
# 1. 取消不存在的订阅
curl -X POST http://localhost:8102/v1/subscription/cancel \
  -H "Content-Type: application/json" \
  -d '{"uid": 9999, "reason": "测试"}'

# 2. 暂停已取消的订阅
curl -X POST http://localhost:8102/v1/subscription/pause \
  -H "Content-Type: application/json" \
  -d '{"uid": 1001, "reason": "测试"}'

# 3. 恢复未暂停的订阅
curl -X POST http://localhost:8102/v1/subscription/resume \
  -H "Content-Type: application/json" \
  -d '{"uid": 1001}'

# 4. 为已取消的订阅设置自动续费
curl -X POST http://localhost:8102/v1/subscription/auto-renew \
  -H "Content-Type: application/json" \
  -d '{"uid": 1001, "auto_renew": true}'
```

## 数据库验证

### 检查订阅状态
```sql
SELECT * FROM user_subscription WHERE user_id = 1001;
```

### 检查历史记录
```sql
SELECT * FROM subscription_history WHERE user_id = 1001 ORDER BY created_at DESC;
```

### 检查自动续费设置
```sql
SELECT user_id, plan_id, status, auto_renew FROM user_subscription WHERE user_id = 1001;
```

## 注意事项

1. **状态转换规则**:
   - active → paused (暂停)
   - active → cancelled (取消)
   - paused → active (恢复)
   - paused → cancelled (取消)

2. **自动续费规则**:
   - 只有 active 状态的订阅可以设置自动续费
   - 取消订阅时会自动关闭自动续费

3. **历史记录**:
   - 所有订阅状态变更都会记录到历史表
   - 历史记录按创建时间倒序排列
   - 支持分页查询

4. **幂等性**:
   - 支付成功回调是幂等的
   - 重复的状态变更操作会返回相应错误信息

