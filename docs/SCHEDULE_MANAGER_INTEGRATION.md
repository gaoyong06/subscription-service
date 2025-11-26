# Schedule Manager 集成需求分析

## 当前 Schedule Manager 的订阅管理实现

### 1. 定时任务 (Cron Job)

**文件**: `cmd/cron/main.go`

**定时任务**:
- 每天凌晨 2 点执行订阅过期检查
- Cron 表达式: `"0 0 2 * * *"`

**执行内容**:
```go
container.UserSubscriptionService.HandleExpiredSubscriptions()
```

### 2. 订阅过期处理逻辑

**文件**: `internal/service/user_subscription_service.go`

**HandleExpiredSubscriptions 方法做了什么**:

1. **更新过期订阅状态**
   ```go
   s.storage.UpdateExpiredSubscriptions()
   ```
   - 查找 `end_time < now` 且 `status = active` 的订阅
   - 排除免费订阅 (`subscription_type != free`)
   - 将状态更新为 `expired`

2. **获取即将过期的订阅（7天内）**
   ```go
   s.storage.GetExpiringSubscriptions()
   ```
   - 查找 7 天内即将过期的活跃订阅
   - 排除免费订阅

3. **发送续费提醒**
   ```go
   for _, sub := range subscriptions {
       // TODO: 集成消息通知服务
       log.Printf("Subscription expiring soon for user %d", sub.UID)
   }
   ```

### 3. Storage 层实现

**文件**: `internal/storage/user_subscription_storage.go`

**UpdateExpiredSubscriptions**:
```sql
UPDATE user_subscription 
SET status = 'expired' 
WHERE end_time < NOW() 
  AND status = 'active' 
  AND subscription_type != 'free'
```

**GetExpiringSubscriptions**:
```sql
SELECT * FROM user_subscription 
WHERE end_time BETWEEN NOW() AND (NOW() + INTERVAL 7 DAY)
  AND status = 'active' 
  AND subscription_type != 'free'
```

## Subscription-Service 当前能力评估

### ✅ 已具备的能力

1. **订阅状态管理**
   - ✅ 创建订阅
   - ✅ 查询订阅状态
   - ✅ 更新订阅状态
   - ✅ 取消订阅
   - ✅ 暂停/恢复订阅

2. **订阅历史记录**
   - ✅ 记录所有状态变更
   - ✅ 分页查询历史

3. **自动续费标记**
   - ✅ 设置/查询自动续费状态

### ❌ 缺失的能力

1. **批量查询过期订阅**
   - ❌ 没有批量查询即将过期的订阅的接口
   - ❌ 没有批量更新过期订阅状态的接口

2. **定时任务支持**
   - ❌ 没有独立的 cron 服务
   - ❌ 没有定时检查过期订阅的逻辑

3. **自动续费执行**
   - ❌ 没有自动续费的实际执行逻辑
   - ❌ 没有自动创建续费订单的功能

## 需要补充的功能

### 1. 批量查询接口

#### 1.1 获取即将过期的订阅
```protobuf
rpc GetExpiringSubscriptions (GetExpiringSubscriptionsRequest) returns (GetExpiringSubscriptionsReply);

message GetExpiringSubscriptionsRequest {
  int32 days_before_expiry = 1;  // 过期前多少天，默认7天
  int32 page = 2;
  int32 page_size = 3;
}

message GetExpiringSubscriptionsReply {
  repeated UserSubscriptionInfo subscriptions = 1;
  int32 total = 2;
}

message UserSubscriptionInfo {
  uint64 uid = 1;
  string plan_id = 2;
  int64 end_time = 3;
  bool auto_renew = 4;
}
```

#### 1.2 批量更新过期订阅状态
```protobuf
rpc UpdateExpiredSubscriptions (UpdateExpiredSubscriptionsRequest) returns (UpdateExpiredSubscriptionsReply);

message UpdateExpiredSubscriptionsRequest {
  // 空请求，自动处理所有过期订阅
}

message UpdateExpiredSubscriptionsReply {
  int32 updated_count = 1;  // 更新的订阅数量
}
```

### 2. 自动续费执行

#### 2.1 处理自动续费
```protobuf
rpc ProcessAutoRenewals (ProcessAutoRenewalsRequest) returns (ProcessAutoRenewalsReply);

message ProcessAutoRenewalsRequest {
  int32 days_before_expiry = 1;  // 提前多少天续费，默认3天
}

message ProcessAutoRenewalsReply {
  int32 processed_count = 1;     // 处理的订阅数量
  int32 success_count = 2;       // 成功的数量
  int32 failed_count = 3;        // 失败的数量
  repeated AutoRenewResult results = 4;
}

message AutoRenewResult {
  uint64 uid = 1;
  bool success = 2;
  string order_id = 3;
  string error_message = 4;
}
```

### 3. Cron 服务

创建独立的 cron 服务：`cmd/cron/main.go`

**定时任务列表**:

1. **订阅过期检查** - 每天凌晨 2 点
   ```go
   "0 0 2 * * *" -> UpdateExpiredSubscriptions()
   ```

2. **续费提醒** - 每天上午 10 点
   ```go
   "0 0 10 * * *" -> GetExpiringSubscriptions(7天)
   ```

3. **自动续费处理** - 每天凌晨 3 点
   ```go
   "0 0 3 * * *" -> ProcessAutoRenewals(3天)
   ```

## 实现计划

### Phase 1: 补充批量查询接口 ✓

- [x] 添加 GetExpiringSubscriptions RPC
- [x] 添加 UpdateExpiredSubscriptions RPC
- [x] 实现 Biz 层逻辑
- [x] 实现 Data 层查询
- [x] 添加单元测试

### Phase 2: 实现自动续费逻辑 ✓

- [x] 添加 ProcessAutoRenewals RPC
- [x] 实现自动创建续费订单
- [x] 集成 Payment Service
- [x] 处理支付失败情况
- [x] 添加重试机制

### Phase 3: 创建 Cron 服务 ✓

- [x] 创建 cmd/cron/main.go
- [x] 集成 robfig/cron
- [x] 添加订阅过期检查任务
- [x] 添加续费提醒任务
- [x] 添加自动续费处理任务
- [x] 添加优雅退出

### Phase 4: 集成通知服务

- [ ] 定义通知接口
- [ ] 实现续费提醒通知
- [ ] 实现自动续费成功通知
- [ ] 实现自动续费失败通知

### Phase 5: 监控和告警

- [ ] 添加 Prometheus 指标
- [ ] 监控订阅过期率
- [ ] 监控自动续费成功率
- [ ] 配置告警规则

## Schedule Manager 集成方案

### 方案 A: Schedule Manager 调用 Subscription-Service

**优点**:
- Schedule Manager 只需调用 gRPC 接口
- 订阅逻辑完全由 Subscription-Service 管理
- 解耦清晰

**缺点**:
- 需要 Schedule Manager 修改代码
- 需要配置 Subscription-Service 地址

**实现步骤**:
1. Subscription-Service 提供完整的订阅管理接口
2. Schedule Manager 移除本地订阅逻辑
3. Schedule Manager 通过 gRPC 调用 Subscription-Service
4. Schedule Manager 的 cron 任务调用 Subscription-Service 的批量接口

### 方案 B: Subscription-Service 独立运行（推荐）

**优点**:
- Subscription-Service 完全独立
- 自己的 cron 服务处理定时任务
- Schedule Manager 无需修改 cron 逻辑

**缺点**:
- 需要部署额外的 cron 服务
- 需要确保 cron 服务的高可用

**实现步骤**:
1. Subscription-Service 实现完整的订阅管理和定时任务
2. Schedule Manager 通过 gRPC 调用订阅查询接口
3. Schedule Manager 移除本地的订阅过期检查逻辑
4. Subscription-Service 的 cron 服务独立处理所有定时任务

## 数据迁移方案

### 从 Schedule Manager 迁移到 Subscription-Service

**迁移脚本**:
```sql
-- 1. 导出 Schedule Manager 的订阅数据
SELECT 
  uid,
  subscription_type,
  start_time,
  end_time,
  status,
  order_id
FROM user_subscription
WHERE status = 'active';

-- 2. 映射到 Subscription-Service 的数据结构
-- subscription_type 映射:
-- 'one_month' -> 'plan_monthly'
-- 'one_year' -> 'plan_yearly'
-- 'two_year' -> 'plan_2yearly'

-- 3. 导入到 Subscription-Service
INSERT INTO user_subscription 
  (user_id, plan_id, start_time, end_time, status, auto_renew, created_at, updated_at)
VALUES
  (...);
```

## API 对比

### Schedule Manager API
```go
// 获取用户订阅
GetByUID(uid uint) (*UserSubscription, error)

// 创建免费订阅
CreateFreeSubscription(uid uint) error

// 处理过期订阅
HandleExpiredSubscriptions() error

// 检查活跃订阅
HasActiveSubscription(uid uint) (bool, error)
```

### Subscription-Service API
```protobuf
// 获取用户订阅
GetMySubscription(uid) -> GetMySubscriptionReply

// 创建订阅订单
CreateSubscriptionOrder(uid, plan_id, payment_method) -> CreateSubscriptionOrderReply

// 处理支付成功
HandlePaymentSuccess(order_id, payment_id, amount) -> HandlePaymentSuccessReply

// 批量更新过期订阅
UpdateExpiredSubscriptions() -> UpdateExpiredSubscriptionsReply

// 获取即将过期的订阅
GetExpiringSubscriptions(days_before_expiry) -> GetExpiringSubscriptionsReply

// 处理自动续费
ProcessAutoRenewals(days_before_expiry) -> ProcessAutoRenewalsReply
```

## 配置示例

### Subscription-Service 配置
```yaml
server:
  http:
    addr: 0.0.0.0:8102
  grpc:
    addr: 0.0.0.0:9102

data:
  database:
    driver: mysql
    source: root:@tcp(localhost:3306)/subscription_service?charset=utf8mb4&parseTime=True&loc=Local

client:
  payment:
    addr: localhost:9101

cron:
  enabled: true
  jobs:
    - name: "update_expired_subscriptions"
      schedule: "0 0 2 * * *"
      enabled: true
    - name: "send_renewal_reminders"
      schedule: "0 0 10 * * *"
      enabled: true
      days_before_expiry: 7
    - name: "process_auto_renewals"
      schedule: "0 0 3 * * *"
      enabled: true
      days_before_expiry: 3
```

### Schedule Manager 配置
```yaml
# 订阅服务配置
subscription_service:
  grpc_addr: localhost:9102
  timeout: 5s
```

## 总结

### Subscription-Service 需要补充的核心功能

1. ✅ **批量查询接口**
   - GetExpiringSubscriptions
   - UpdateExpiredSubscriptions

2. ✅ **自动续费执行**
   - ProcessAutoRenewals
   - 自动创建订单
   - 调用支付服务

3. ✅ **Cron 服务**
   - 独立的定时任务服务
   - 订阅过期检查
   - 续费提醒
   - 自动续费处理

4. ⏳ **通知集成**（后续）
   - 续费提醒通知
   - 自动续费结果通知

### 推荐方案

采用 **方案 B**：Subscription-Service 独立运行

**理由**:
1. 职责清晰：Subscription-Service 完全负责订阅管理
2. 解耦彻底：Schedule Manager 只需调用查询接口
3. 可扩展性好：未来可以服务更多业务系统
4. 维护简单：订阅逻辑集中管理

### 下一步行动

1. ✅ 实现批量查询接口
2. ✅ 实现自动续费逻辑
3. ✅ 创建 Cron 服务
4. ⏳ 编写迁移脚本
5. ⏳ 集成测试
6. ⏳ 部署上线

