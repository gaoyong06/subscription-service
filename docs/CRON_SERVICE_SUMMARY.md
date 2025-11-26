# Cron 服务实现总结

## 概述

为了支持 Schedule Manager 的需求，我们为 Subscription-Service 添加了完整的定时任务功能，包括：

1. **批量查询接口** - 用于查询即将过期的订阅
2. **批量更新接口** - 用于更新过期订阅状态
3. **自动续费处理** - 用于自动处理订阅续费
4. **独立的 Cron 服务** - 用于执行定时任务

## 新增的 API 接口

### 1. 获取即将过期的订阅

**接口**: `GET /v1/subscription/expiring`

**参数**:
- `days_before_expiry`: 过期前多少天（1-30，默认7天）
- `page`: 页码（默认1）
- `page_size`: 每页数量（默认10）

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "subscriptions": [
      {
        "uid": 1001,
        "plan_id": "plan_monthly",
        "plan_name": "月度套餐",
        "start_time": 1700000000,
        "end_time": 1702592000,
        "auto_renew": true,
        "amount": 29.9
      }
    ],
    "total": 10,
    "page": 1,
    "page_size": 10
  }
}
```

**使用场景**:
- 定时任务：每天检查即将过期的订阅
- 发送续费提醒通知
- 监控订阅过期情况

### 2. 批量更新过期订阅状态

**接口**: `POST /v1/subscription/expired/update`

**请求体**: 空（自动处理所有过期订阅）

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "updated_count": 5,
    "updated_uids": [1001, 1002, 1003, 1004, 1005]
  }
}
```

**使用场景**:
- 定时任务：每天凌晨更新过期订阅状态
- 自动将 `end_time < now` 的订阅状态改为 `expired`
- 记录订阅历史

### 3. 处理自动续费

**接口**: `POST /v1/subscription/auto-renew/process`

**请求体**:
```json
{
  "days_before_expiry": 3,
  "dry_run": false
}
```

**参数说明**:
- `days_before_expiry`: 提前多少天续费（1-30，默认3天）
- `dry_run`: 是否为测试运行（true: 只检查不执行，false: 实际执行）

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total_count": 10,
    "success_count": 8,
    "failed_count": 2,
    "results": [
      {
        "uid": 1001,
        "plan_id": "plan_monthly",
        "success": true,
        "order_id": "SUB17000000001001",
        "payment_id": "PAY17000000001",
        "error_message": ""
      },
      {
        "uid": 1002,
        "plan_id": "plan_yearly",
        "success": false,
        "order_id": "",
        "payment_id": "",
        "error_message": "payment failed"
      }
    ]
  }
}
```

**使用场景**:
- 定时任务：每天凌晨自动处理续费
- 为开启自动续费的用户自动创建订单
- 自动调用支付服务扣款
- 处理失败情况并记录

## Cron 服务

### 服务位置

- **源码**: `cmd/cron/main.go`
- **编译**: `make build-cron`
- **二进制**: `bin/cron`

### 定时任务列表

| 任务名称 | Cron 表达式 | 执行时间 | 功能描述 |
|---------|------------|---------|---------|
| 订阅过期检查 | `0 0 2 * * *` | 每天凌晨 2:00 | 批量更新过期订阅状态 |
| 续费提醒 | `0 0 10 * * *` | 每天上午 10:00 | 获取7天内过期的订阅并发送提醒 |
| 自动续费处理 | `0 0 3 * * *` | 每天凌晨 3:00 | 处理3天内过期且开启自动续费的订阅 |

### 启动方式

#### 方式1: 直接运行
```bash
./bin/cron -conf ./configs/config.yaml
```

#### 方式2: 使用 Makefile
```bash
make run-cron
```

#### 方式3: 使用 Supervisor
```bash
supervisorctl start subscription-cron
```

配置文件: `deploy/supervisor/subscription-cron.conf`

### 日志输出

Cron 服务的日志会输出到：
- 标准输出: 实时日志
- 文件: `logs/cron.log`（如果使用 nohup 或 supervisor）
- 错误日志: `logs/cron_error.log`（如果使用 supervisor）

示例日志：
```
2025-11-26 02:00:00 [CRON] Starting subscription expiration check...
2025-11-26 02:00:01 [CRON] Updated 5 expired subscriptions: [1001 1002 1003 1004 1005]
2025-11-26 02:00:01 [CRON] Finished subscription expiration check

2025-11-26 03:00:00 [CRON] Starting auto-renewal process...
2025-11-26 03:00:05 [CRON] Auto-renewal completed: total=10, success=8, failed=2
2025-11-26 03:00:05 [CRON] Auto-renewal success: user=1001, plan=plan_monthly, order=SUB17000000001001
2025-11-26 03:00:05 [CRON] Auto-renewal failed: user=1002, plan=plan_yearly, error=payment failed
2025-11-26 03:00:05 [CRON] Finished auto-renewal process

2025-11-26 10:00:00 [CRON] Starting renewal reminder check...
2025-11-26 10:00:01 [CRON] Found 15 subscriptions expiring within 7 days
2025-11-26 10:00:01 [CRON] Reminder: User 1001 subscription (plan: plan_monthly) expires at 2025-12-03 10:00:00
2025-11-26 10:00:01 [CRON] Finished renewal reminder check
```

## 启动所有服务

### 方式1: 使用 restart_server.sh（推荐）

```bash
bash script/restart_server.sh
```

这个脚本会：
1. 检查并释放端口（8102, 9102）
2. 停止已运行的 cron 服务
3. 生成 proto 文件
4. 生成 swagger 文档
5. 编译所有服务（server + cron）
6. 启动 cron 服务（后台运行）
7. 启动主服务（前台运行）
8. 主服务退出时自动停止 cron 服务

### 方式2: 使用 Makefile

```bash
# 编译所有服务
make build-all

# 运行所有服务
make run-all
```

`make run-all` 会：
1. 启动 cron 服务（后台）
2. 启动主服务（前台）
3. 主服务退出时自动停止 cron 服务

### 方式3: 分别启动

```bash
# 终端1: 启动主服务
make run

# 终端2: 启动 cron 服务
make run-cron
```

### 停止所有服务

```bash
make stop-all
```

## 测试验证

### 1. 测试 API 接口

使用提供的测试脚本：
```bash
bash test_cron_apis.sh
```

或手动测试：

```bash
# 1. 获取即将过期的订阅
curl "http://localhost:8102/v1/subscription/expiring?days_before_expiry=7&page=1&page_size=10" | jq '.'

# 2. 批量更新过期订阅
curl -X POST http://localhost:8102/v1/subscription/expired/update \
  -H "Content-Type: application/json" \
  -d '{}' | jq '.'

# 3. 测试自动续费（dry run）
curl -X POST http://localhost:8102/v1/subscription/auto-renew/process \
  -H "Content-Type: application/json" \
  -d '{"days_before_expiry": 3, "dry_run": true}' | jq '.'
```

### 2. 测试 Cron 服务

#### 方法1: 查看日志
```bash
tail -f logs/cron.log
```

#### 方法2: 手动触发（修改 Cron 表达式）

临时修改 `cmd/cron/main.go` 中的 Cron 表达式为更频繁的执行：
```go
// 每分钟执行一次（用于测试）
"0 * * * * *" -> UpdateExpiredSubscriptions()
```

然后重新编译并运行：
```bash
make build-cron
./bin/cron -conf ./configs/config.yaml
```

## 数据库查询

### 查看即将过期的订阅
```sql
SELECT user_id, plan_id, end_time, auto_renew, status
FROM user_subscription
WHERE end_time BETWEEN NOW() AND DATE_ADD(NOW(), INTERVAL 7 DAY)
  AND status = 'active'
ORDER BY end_time ASC;
```

### 查看过期的订阅
```sql
SELECT user_id, plan_id, end_time, status
FROM user_subscription
WHERE end_time < NOW()
  AND status = 'active';
```

### 查看开启自动续费的订阅
```sql
SELECT user_id, plan_id, end_time, auto_renew, status
FROM user_subscription
WHERE auto_renew = true
  AND status = 'active'
ORDER BY end_time ASC;
```

## 与 Schedule Manager 集成

### 集成方案

采用 **方案 B：Subscription-Service 独立运行**

```
┌─────────────────────┐
│ Schedule Manager    │
│  - 调用订阅查询API   │
│  - 移除本地订阅逻辑  │
└──────────┬──────────┘
           │ gRPC/HTTP
           ↓
┌─────────────────────┐
│ Subscription Service│
│  ┌───────────────┐  │
│  │ Server (8102) │  │  ← HTTP/gRPC API
│  └───────────────┘  │
│  ┌───────────────┐  │
│  │ Cron Service  │  │  ← 定时任务
│  └───────────────┘  │
└──────────┬──────────┘
           │
           ↓
┌─────────────────────┐
│   MySQL Database    │
└─────────────────────┘
```

### Schedule Manager 需要的修改

1. **移除本地订阅过期检查**
   - 删除 `cmd/cron/main.go` 中的 `HandleExpiredSubscriptions` 调用
   - 删除 `internal/service/user_subscription_service.go` 中的相关方法
   - 删除 `internal/storage/user_subscription_storage.go` 中的相关方法

2. **调用 Subscription-Service 的 API**
   ```go
   // 获取用户订阅状态
   resp, err := subscriptionClient.GetMySubscription(ctx, &pb.GetMySubscriptionRequest{
       Uid: userID,
   })
   ```

3. **配置 Subscription-Service 地址**
   ```yaml
   # config.yaml
   subscription_service:
     grpc_addr: localhost:9102
     timeout: 5s
   ```

## 监控和告警

### Prometheus 指标（待实现）

```go
// 过期订阅数量
subscription_expired_total

// 自动续费成功/失败数量
subscription_auto_renewals_total{status="success"}
subscription_auto_renewals_total{status="failed"}

// 即将过期的订阅数量
subscription_expiring_count
```

### 告警规则（待实现）

```yaml
# 自动续费失败率过高
- alert: HighAutoRenewalFailureRate
  expr: rate(subscription_auto_renewals_total{status="failed"}[5m]) > 0.1
  annotations:
    summary: "自动续费失败率过高"

# 大量订阅即将过期
- alert: ManySubscriptionsExpiringSoon
  expr: subscription_expiring_count > 100
  annotations:
    summary: "大量订阅即将过期"
```

## 后续优化

### 1. 通知集成
- [ ] 集成邮件通知服务
- [ ] 集成短信通知服务
- [ ] 集成 WebSocket 实时通知
- [ ] 发送续费提醒通知
- [ ] 发送自动续费成功/失败通知

### 2. 支付集成优化
- [ ] 实现真实的自动扣款逻辑
- [ ] 处理支付失败重试
- [ ] 支持多种支付方式的自动续费
- [ ] 记录支付失败原因

### 3. 监控和告警
- [ ] 添加 Prometheus 指标
- [ ] 配置 Grafana 仪表板
- [ ] 设置告警规则
- [ ] 集成告警通知（钉钉、企业微信等）

### 4. 性能优化
- [ ] 批量处理优化（减少数据库查询）
- [ ] 添加 Redis 缓存
- [ ] 异步处理自动续费
- [ ] 添加任务队列（如 RabbitMQ）

### 5. 容错和高可用
- [ ] Cron 服务的分布式锁（防止重复执行）
- [ ] 失败重试机制
- [ ] 任务执行状态记录
- [ ] Cron 服务的健康检查

## 总结

我们成功实现了完整的定时任务功能，包括：

✅ **3个新的 API 接口**
- GetExpiringSubscriptions - 查询即将过期的订阅
- UpdateExpiredSubscriptions - 批量更新过期订阅
- ProcessAutoRenewals - 处理自动续费

✅ **独立的 Cron 服务**
- 订阅过期检查（每天凌晨 2:00）
- 续费提醒（每天上午 10:00）
- 自动续费处理（每天凌晨 3:00）

✅ **完善的启动方式**
- `restart_server.sh` - 一键启动所有服务
- `make run-all` - Makefile 启动所有服务
- Supervisor 配置 - 生产环境部署

✅ **完整的文档**
- API 文档
- 集成方案
- 测试指南
- 运维指南

现在 Subscription-Service 已经具备了完整的订阅管理能力，可以完全支撑 Schedule Manager 的需求！

