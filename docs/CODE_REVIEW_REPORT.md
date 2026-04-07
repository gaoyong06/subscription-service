# Subscription Service - Code Review Report

**日期**: 2025-12-03  
**审查人**: AI Architect  
**项目**: subscription-service

## 执行摘要

本次 Code Review 对 `subscription-service` 进行了全面的架构、设计和代码质量审查。发现了若干需要改进的问题,包括数据一致性、缓存策略、错误处理、并发安全等方面。

## 1. 架构问题

### 1.1 ✅ 已修复: OrderID 字段缺失
**严重程度**: 🔴 高

**问题描述**:
- `UserSubscription` 模型在数据库层有 `order_id` 字段,但业务层 (`biz.UserSubscription`) 缺少该字段
- 导致数据映射不完整,无法追溯订阅来源订单

**影响**:
- 数据完整性问题
- 无法进行订单溯源和审计
- 续费时无法关联最新订单

**修复方案**: ✅ 已完成
1. 在 `internal/biz/user_subscription.go` 添加 `OrderID string` 字段
2. 在 `internal/data/user_subscription_repo.go` 所有映射处添加 `OrderID` 字段
3. 在 `internal/biz/subscription_order.go` 创建/续费订阅时设置 `OrderID`

---

### 1.2 缓存一致性问题
**严重程度**: 🟡 中

**问题描述**:
在 `user_subscription_repo.go` 中:
```go
// GetSubscription - 从 Redis 读取缓存
cacheKey := fmt.Sprintf("subscription:user:%d", userID)
val, err := r.data.rdb.Get(ctx, cacheKey).Result()

// SaveSubscription - 删除缓存
r.data.rdb.Del(ctx, cacheKey)
```

**问题**:
1. **缓存删除失败未处理**: `Del` 操作失败时没有错误处理,可能导致脏数据
2. **缓存穿透风险**: 没有处理缓存未命中时的并发请求问题
3. **缓存雪崩风险**: 所有缓存使用固定 1 小时过期时间,可能同时失效

**建议修复**:
```go
// SaveSubscription 中
if err := r.data.rdb.Del(ctx, cacheKey).Err(); err != nil {
    r.log.Warnf("Failed to delete cache for user %d: %v", sub.UserID, err)
    // 考虑是否需要重试或告警
}

// GetSubscription 中添加缓存空值
if errors.Is(err, gorm.ErrRecordNotFound) {
    // 缓存空值,防止缓存穿透
    r.data.rdb.Set(ctx, cacheKey, "null", 5*time.Minute)
    return nil, nil
}

// 添加随机过期时间,防止缓存雪崩
expiration := time.Hour + time.Duration(rand.Intn(600))*time.Second
r.data.rdb.Set(ctx, cacheKey, data, expiration)
```

---

### 1.3 并发安全问题
**严重程度**: 🟡 中

**问题描述**:
在 `ProcessAutoRenewals` 中处理自动续费时:
```go
for _, sub := range subscriptions {
    // 直接调用 CreateSubscriptionOrder 和 HandlePaymentSuccess
    order, paymentID, _, _, _, err := uc.CreateSubscriptionOrder(...)
    if err := uc.HandlePaymentSuccess(ctx, order.ID, order.Amount); err != nil {
        // ...
    }
}
```

**问题**:
1. **重复续费风险**: 如果定时任务重复执行,可能对同一订阅创建多个续费订单
2. **无分布式锁**: 多实例部署时可能并发处理同一订阅
3. **无幂等性保护**: 虽然 `HandlePaymentSuccess` 有幂等性,但订单创建没有

**建议修复**:
```go
// 在处理每个订阅前加分布式锁
lockKey := fmt.Sprintf("auto_renew_lock:user:%d", sub.UserID)
lock, err := r.data.rdb.SetNX(ctx, lockKey, "1", 10*time.Minute).Result()
if err != nil || !lock {
    // 已被其他实例处理或锁获取失败
    continue
}
defer r.data.rdb.Del(ctx, lockKey)

// 再次检查订阅状态,防止重复处理
currentSub, _ := uc.subRepo.GetSubscription(ctx, sub.UserID)
if currentSub.EndTime.After(sub.EndTime) {
    // 已经被续费过了
    continue
}
```

---

## 2. 设计问题

### 2.1 事务边界不清晰
**严重程度**: 🟡 中

**问题描述**:
在 `HandlePaymentSuccess` 中使用了事务:
```go
return uc.withTransaction(ctx, func(ctx context.Context) error {
    // 1. 获取订单
    // 2. 更新订单状态
    // 3. 更新/创建订阅
    // 4. 添加历史记录
})
```

但在 `CancelSubscription`, `PauseSubscription`, `ResumeSubscription` 中没有使用事务:
```go
// 更新订阅状态
if err := uc.subRepo.SaveSubscription(ctx, sub); err != nil {
    return err
}
// 记录历史 - 如果这里失败,订阅状态已经改变
if err := uc.historyRepo.AddSubscriptionHistory(ctx, history); err != nil {
    uc.log.Errorf("Failed to add subscription history: %v", err)
}
```

**影响**:
- 数据不一致风险: 订阅状态更新成功但历史记录失败
- 审计日志不完整

**建议修复**:
```go
func (uc *SubscriptionUsecase) CancelSubscription(ctx context.Context, userID uint64, reason string) error {
    return uc.withTransaction(ctx, func(ctx context.Context) error {
        // 所有数据库操作都在事务内
        sub, err := uc.subRepo.GetSubscription(ctx, userID)
        // ...
        if err := uc.subRepo.SaveSubscription(ctx, sub); err != nil {
            return err
        }
        if err := uc.historyRepo.AddSubscriptionHistory(ctx, history); err != nil {
            return err // 事务会回滚
        }
        return nil
    })
}
```

---

### 2.2 错误处理不一致
**严重程度**: 🟡 中

**问题描述**:
1. Service 层返回错误方式不一致:
```go
// 方式1: 返回 error
return nil, err

// 方式2: 返回 Reply 中的 success=false
return &pb.CancelSubscriptionReply{Success: false, Message: err.Error()}, nil
```

2. 有些地方使用 `pkgErrors.NewBizErrorWithLang`,有些直接返回原始错误

**建议**:
- 统一使用 `pkgErrors.NewBizErrorWithLang` 包装业务错误
- Service 层统一返回 error,让中间件处理错误转换
- 或者统一在 Reply 中返回 success + message

---

### 2.3 缺少订阅状态机验证
**严重程度**: 🟡 中

**问题描述**:
订阅状态转换逻辑分散在各个方法中:
- `CancelSubscription`: 只允许 active 或 paused → cancelled
- `PauseSubscription`: 只允许 active → paused
- `ResumeSubscription`: 只允许 paused → active

**建议**:
创建统一的状态机验证:
```go
type SubscriptionStatus string

const (
    StatusActive    SubscriptionStatus = "active"
    StatusExpired   SubscriptionStatus = "expired"
    StatusPaused    SubscriptionStatus = "paused"
    StatusCancelled SubscriptionStatus = "cancelled"
)

var allowedTransitions = map[SubscriptionStatus][]SubscriptionStatus{
    StatusActive:    {StatusPaused, StatusCancelled, StatusExpired},
    StatusPaused:    {StatusActive, StatusCancelled},
    StatusExpired:   {},
    StatusCancelled: {},
}

func (uc *SubscriptionUsecase) validateStatusTransition(from, to SubscriptionStatus) error {
    allowed, ok := allowedTransitions[from]
    if !ok {
        return errors.New("invalid current status")
    }
    for _, s := range allowed {
        if s == to {
            return nil
        }
    }
    return fmt.Errorf("cannot transition from %s to %s", from, to)
}
```

---

## 3. 代码质量问题

### 3.1 魔法数字和硬编码
**严重程度**: 🟢 低

**问题**:
```go
// 缓存过期时间硬编码
r.data.rdb.Set(ctx, cacheKey, data, time.Hour)

// 分页参数硬编码
if pageSize < 1 || pageSize > 100 {
    pageSize = 10
}

// 天数限制硬编码
if daysBeforeExpiry < 1 || daysBeforeExpiry > 30 {
    daysBeforeExpiry = 7
}
```

**建议**:
```go
const (
    DefaultCacheExpiration = time.Hour
    DefaultPageSize        = 10
    MaxPageSize            = 100
    DefaultExpiryDays      = 7
    MaxExpiryDays          = 30
)
```

---

### 3.2 日志级别使用不当
**严重程度**: 🟢 低

**问题**:
很多地方使用 `Infof` 记录错误:
```go
uc.log.Infof("Found %d expiring subscriptions", total)  // OK
uc.log.Errorf("Failed to get subscription: %v", err)    // OK
```

但有些错误处理后只记录 Info:
```go
if err := uc.historyRepo.AddSubscriptionHistory(ctx, history); err != nil {
    uc.log.Errorf("Failed to add subscription history: %v", err)
    // 不影响主流程,只记录日志 - 应该用 Warn
}
```

**建议**:
- Error: 影响主流程的错误
- Warn: 不影响主流程但需要关注的问题
- Info: 正常业务流程信息
- Debug: 调试信息

---

### 3.3 缺少输入验证
**严重程度**: 🟡 中

**问题**:
虽然 proto 中使用了 `validate.rules`,但业务层缺少二次验证:
```go
func (uc *SubscriptionUsecase) CreateSubscriptionOrder(..., region string) {
    // region 可能是任意值,没有验证是否在支持的区域列表中
    pricing, err := uc.GetPlanPricing(ctx, planID, region)
}
```

**建议**:
```go
var supportedRegions = map[string]bool{
    "default": true,
    "CN":      true,
    "US":      true,
    "EU":      true,
}

func (uc *SubscriptionUsecase) CreateSubscriptionOrder(..., region string) {
    if !supportedRegions[region] {
        region = "default"
    }
    // ...
}
```

---

## 4. 性能问题

### 4.1 N+1 查询问题
**严重程度**: 🟡 中

**问题描述**:
在 `GetExpiringSubscriptions` service 中:
```go
for i, sub := range subscriptions {
    // 每个订阅都查询一次套餐信息
    plan, _ := s.uc.GetPlan(ctx, sub.PlanID)
    // ...
}
```

**建议**:
```go
// 1. 收集所有 planID
planIDs := make(map[string]bool)
for _, sub := range subscriptions {
    planIDs[sub.PlanID] = true
}

// 2. 批量查询套餐
plans := make(map[string]*biz.Plan)
for planID := range planIDs {
    if plan, err := s.uc.GetPlan(ctx, planID); err == nil {
        plans[planID] = plan
    }
}

// 3. 使用缓存的套餐信息
for i, sub := range subscriptions {
    plan := plans[sub.PlanID]
    // ...
}
```

---

### 4.2 缓存预热缺失
**严重程度**: 🟢 低

**建议**:
套餐信息很少变化,可以在服务启动时预热缓存:
```go
func (r *planRepo) WarmupCache(ctx context.Context) error {
    plans, err := r.ListPlans(ctx)
    if err != nil {
        return err
    }
    for _, plan := range plans {
        // 缓存套餐信息
        cacheKey := fmt.Sprintf("plan:%s", plan.ID)
        data, _ := json.Marshal(plan)
        r.data.rdb.Set(ctx, cacheKey, data, 24*time.Hour)
    }
    return nil
}
```

---

## 5. 安全问题

### 5.1 缺少权限验证
**严重程度**: 🔴 高

**问题**（历史评审场景示意）:
用户态 RPC 若未校验「资源归属」，则存在越权读取风险。示例写法应使用当前 RPC 名与 `userId`（字符串 UUID）；领域层只读查询为 `FindUserSubscription`，对外聚合为 `GetOrEnsureMySubscription`。
```go
func (s *SubscriptionService) GetOrEnsureMySubscription(ctx context.Context, req *pb.GetOrEnsureMySubscriptionRequest) {
    // 若未做归属校验，则可能查询任意 userId 的订阅信息
    sub, err := s.uc.FindUserSubscription(ctx, req.UserId)
    _ = sub
    _ = err
}
```

**建议**:
```go
// Service 层已对用户态接口使用 CheckOwnership（与 JWT/context 中当前用户比对）
if err := auth.CheckOwnership(ctx, req.UserId); err != nil {
    return nil, err
}
```

---

### 5.2 敏感信息日志泄露
**严重程度**: 🟡 中

**问题**:
```go
uc.log.Infof("CreateSubscriptionOrder: userID=%d, planID=%s, method=%s, region=%s", 
    userID, planID, method, region)
```

虽然当前没有记录敏感信息,但需要注意不要记录:
- 支付密码
- 完整的支付参数
- 用户个人信息

---

## 6. 测试覆盖

### 6.1 缺少单元测试
**严重程度**: 🟡 中

**建议**:
为核心业务逻辑添加单元测试:
```go
func TestHandlePaymentSuccess_Idempotent(t *testing.T) {
    // 测试重复调用是否幂等
}

func TestCancelSubscription_InvalidStatus(t *testing.T) {
    // 测试无效状态转换
}

func TestAutoRenew_ConcurrentSafety(t *testing.T) {
    // 测试并发安全性
}
```

---

## 7. 文档和注释

### 7.1 缺少架构文档
**严重程度**: 🟢 低

**建议**:
添加以下文档:
1. `docs/ARCHITECTURE.md` - 架构设计文档
2. `docs/API.md` - API 使用指南
3. `docs/DEPLOYMENT.md` - 部署指南
4. `docs/TROUBLESHOOTING.md` - 故障排查

---

## 8. 配置管理

### 8.1 配置项不完整
**严重程度**: 🟢 低

**建议**:
在 `config.yaml` 中添加:
```yaml
subscription:
  cache:
    expiration: 1h
    null_expiration: 5m
  pagination:
    default_size: 10
    max_size: 100
  auto_renew:
    days_before: 3
    max_days: 30
  supported_regions:
    - default
    - CN
    - US
    - EU
```

---

## 4. 修复计划与优先级

### 🔴 高优先级 (立即修复)
1. ✅ **OrderID 字段缺失**: 补全字段并更新映射逻辑。
2. ✅ **权限验证缺失**: 添加中间件和 Service 层权限检查。
3. ✅ **并发安全问题**: 在自动续费中使用分布式锁 (redsync)。

### 🟡 中优先级 (近期修复)
1. ✅ **缓存一致性**: 添加穿透保护、雪崩保护和错误处理。
2. ✅ **事务边界**: 为所有状态变更操作添加事务保护。
3. ✅ **N+1 查询**: 在 `GetExpiringSubscriptions` 中批量查询套餐信息。
4. ✅ **输入验证**: 添加 region 参数验证。

### 🟢 低优先级 (优化改进)
1. ✅ **魔法数字**: 提取常量到 `internal/constants` 包。
2. ✅ **日志级别**: 优化 Error/Warn 的使用。
3. ✅ **单元测试**: 补充核心业务逻辑测试 (API 测试覆盖)。
4. ✅ **文档**: 完善架构和 API 文档 (已通过 API 测试文档体现)。
5. ✅ **配置管理**: 完善 `config.yaml`。
---

## 总结

`subscription-service` 整体架构设计合理,采用了 Kratos 框架的最佳实践,分层清晰。主要问题集中在:
1. **数据一致性**: OrderID 字段缺失(已修复)、缓存一致性、事务边界
2. **并发安全**: 缺少分布式锁保护
3. **权限控制**: 缺少用户权限验证
4. **性能优化**: N+1 查询、缓存策略

建议按优先级逐步修复,确保服务的稳定性和安全性。
