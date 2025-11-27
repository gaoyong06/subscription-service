# Subscription Service

基于 Kratos 框架的订阅管理微服务

## 服务说明

Subscription Service 是一个独立的**订阅管理微服务**，负责管理用户的订阅套餐、会员权益和订阅生命周期。

### 核心能力

#### 订阅管理
- ✅ **套餐管理**: 管理各种订阅套餐（月卡、年卡等）
- ✅ **订阅查询**: 查询用户当前订阅状态
- ✅ **订阅购买**: 创建订阅订单并调用支付服务
- ✅ **订阅延长**: 处理支付成功后的订阅延长
- ✅ **订阅续费**: 支持订阅到期后续费
- ✅ **订阅升级**: 支持从低级套餐升级到高级套餐
- ✅ **订阅取消**: 支持用户主动取消订阅
- ✅ **订阅暂停/恢复**: 支持临时暂停和恢复订阅
- ✅ **历史记录**: 记录所有订阅状态变更历史
- ✅ **自动续费**: 支持开启/关闭自动续费功能

#### 定时任务（Cron 服务）
- ✅ **过期检查**: 每天自动更新过期订阅状态
- ✅ **续费提醒**: 每天检查即将过期的订阅
- ✅ **自动续费**: 每天自动处理开启自动续费的订阅
- ✅ **批量查询**: 支持批量查询即将过期的订阅
- ✅ **批量更新**: 支持批量更新过期订阅状态

#### 技术特性
- ✅ **统一响应**: 标准化的 API 响应格式
- ✅ **国际化**: 支持多语言错误消息
- ✅ **参数验证**: 使用 protobuf validate 进行参数校验

### 服务边界

**负责**:
- 订阅套餐的定义和管理
- 用户订阅状态的管理和查询
- 订阅订单的创建和管理
- 与 Payment Service 的集成
- 订阅生命周期管理（激活、延长、过期、取消、暂停、恢复）
- 订阅业务逻辑（首购、续费、升级）
- 订阅历史记录的管理
- 自动续费功能的管理

**不负责**:
- 支付处理（由 Payment Service 处理）
- 用户身份验证（由 Passport Service 处理）
- 具体业务功能的权限控制（由业务服务处理）
- 短信/邮件通知（由 Notification Service 处理）

## 技术规格

### 服务信息

| 项目 | 值 |
|------|-----|  
| 服务名称 | subscription-service |
| 框架 | Kratos v2.9.1 |
| Go 版本 | 1.21+ |
| gRPC 端口 | 9102 |
| HTTP 端口 | 8102 |
| 数据库 | MySQL 8.0+ |
| 数据库名 | subscription_service |
| 依赖服务 | Payment Service (9101) |

### 数据表

| 表名 | 说明 | 主键 |
|------|------|------|
| `plan` | 订阅套餐表 | plan_id |
| `user_subscription` | 用户订阅表 | user_subscription_id |
| `subscription_order` | 订阅订单表 | order_id |

### 技术栈

| 组件 | 技术 | 说明 |
|------|------|------|
| 框架 | Kratos v2 | Go 微服务框架 |
| 协议 | gRPC + HTTP | 双协议支持 |
| 数据库 | MySQL 8.0+ | 主数据存储 |
| ORM | GORM | 数据库操作 |
| 日志 | go-pkg/logger | 结构化日志，支持轮转 |
| 国际化 | go-pkg/i18n | 多语言支持 |

### 订阅状态

| 状态 | 说明 |
|------|------|
| active | 激活中 |
| expired | 已过期 |

## Cron 定时任务服务

Subscription Service 包含一个独立的 Cron 服务，用于执行定时任务。

### 定时任务列表

| 任务名称 | 执行时间 | Cron 表达式 | 功能描述 |
|---------|---------|------------|---------|
| 订阅过期检查 | 每天凌晨 2:00 | `0 0 2 * * *` | 批量更新过期订阅状态 |
| 续费提醒 | 每天上午 10:00 | `0 0 10 * * *` | 获取7天内过期的订阅并发送提醒 |
| 自动续费处理 | 每天凌晨 3:00 | `0 0 3 * * *` | 处理3天内过期且开启自动续费的订阅 |

### Cron 服务启动

```bash
# 编译 Cron 服务
make build-cron

# 启动 Cron 服务
./bin/cron -conf ./configs/config.yaml

# 或使用 Makefile
make run-cron

# 使用 Supervisor（生产环境）
supervisorctl start subscription-cron
```

### Supervisor 配置

配置文件位于 `deploy/supervisor/subscription-cron.conf`：

```ini
[program:subscription-cron]
directory=/path/to/subscription-service
command=/path/to/subscription-service/bin/cron -conf /path/to/configs/config.yaml
autostart=true
autorestart=true
user=www-data
stdout_logfile=/path/to/logs/cron.log
stderr_logfile=/path/to/logs/cron_error.log
```

### 日志查看

```bash
# 查看 Cron 服务日志
tail -f logs/cron.log

# 查看错误日志
tail -f logs/cron_error.log
```

详细文档请参考：[Cron 服务实现总结](docs/CRON_SERVICE_SUMMARY.md)

## API 文档

### gRPC 接口

#### 1. 获取套餐列表 (ListPlans)

```protobuf
rpc ListPlans (ListPlansRequest) returns (ListPlansReply);

message ListPlansRequest {}

message ListPlansReply {
  repeated Plan plans = 1;
}

message Plan {
  string id = 1;             // plan_monthly, plan_yearly
  string name = 2;           // Pro Monthly
  string description = 3;    // Pro features for 1 month
  double price = 4;          // 9.99
  string currency = 5;       // CNY
  int32 duration_days = 6;   // 30
  string type = 7;           // free, pro, enterprise
}
```

**示例**:
```go
conn, _ := grpc.Dial("localhost:9102", grpc.WithInsecure())
client := subscriptionv1.NewSubscriptionClient(conn)

resp, err := client.ListPlans(context.Background(), &subscriptionv1.ListPlansRequest{})

for _, plan := range resp.Plans {
    fmt.Printf("套餐: %s, 价格: %.2f, 时长: %d天\n", 
        plan.Name, plan.Price, plan.DurationDays)
}
```

#### 2. 获取我的订阅 (GetMySubscription)

```protobuf
rpc GetMySubscription (GetMySubscriptionRequest) returns (GetMySubscriptionReply);

message GetMySubscriptionRequest {
  uint64 uid = 1;
}

message GetMySubscriptionReply {
  bool is_active = 1;        // 是否激活
  string plan_id = 2;        // 当前套餐ID
  int64 start_time = 3;      // 开始时间（Unix时间戳）
  int64 end_time = 4;        // 结束时间（Unix时间戳）
  string status = 5;         // active, expired, paused, cancelled
  bool auto_renew = 6;       // 是否自动续费
}
```

**示例**:
```go
resp, err := client.GetMySubscription(context.Background(), &subscriptionv1.GetMySubscriptionRequest{
    Uid: 1001,
})

if resp.IsActive {
    fmt.Printf("会员有效期至: %s\n", time.Unix(resp.EndTime, 0))
} else {
    fmt.Println("当前不是会员")
}
```

#### 3. 创建订阅订单 (CreateSubscriptionOrder)

```protobuf
rpc CreateSubscriptionOrder (CreateSubscriptionOrderRequest) returns (CreateSubscriptionOrderReply);

message CreateSubscriptionOrderRequest {
  uint64 uid = 1;
  string plan_id = 2;        // plan_monthly, plan_yearly
  string payment_method = 3; // alipay, wechatpay
}

message CreateSubscriptionOrderReply {
  string order_id = 1;       // 订单号
  string payment_id = 2;     // 支付流水号
  string pay_url = 3;        // 支付链接
  string pay_code = 4;       // 支付二维码
  string pay_params = 5;     // 支付参数
}
```

**示例**:
```go
resp, err := client.CreateSubscriptionOrder(context.Background(), &subscriptionv1.CreateSubscriptionOrderRequest{
    Uid:           1001,
    PlanId:        "plan_monthly",
    PaymentMethod: "alipay",
})

// 引导用户到支付页面
payUrl := resp.PayUrl
```

#### 4. 处理支付成功 (HandlePaymentSuccess)

```protobuf
rpc HandlePaymentSuccess (HandlePaymentSuccessRequest) returns (HandlePaymentSuccessReply);

message HandlePaymentSuccessRequest {
  string order_id = 1;
  string payment_id = 2;
  double amount = 3;
}

message HandlePaymentSuccessReply {
  bool success = 1;
}
```

**示例**:
```go
// 通常由 Payment Service 的回调触发
resp, err := client.HandlePaymentSuccess(context.Background(), &subscriptionv1.HandlePaymentSuccessRequest{
    OrderId:   "SUB20231123001",
    PaymentId: "PAY20231123001",
    Amount:    9.99,
})
```

#### 5. 取消订阅 (CancelSubscription)

```protobuf
rpc CancelSubscription (CancelSubscriptionRequest) returns (CancelSubscriptionReply);

message CancelSubscriptionRequest {
  uint64 uid = 1;
  string reason = 2;  // 取消原因（可选）
}

message CancelSubscriptionReply {
  bool success = 1;
  string message = 2;
}
```

**示例**:
```go
resp, err := client.CancelSubscription(context.Background(), &subscriptionv1.CancelSubscriptionRequest{
    Uid:    1001,
    Reason: "不再需要此服务",
})
```

#### 6. 暂停订阅 (PauseSubscription)

```protobuf
rpc PauseSubscription (PauseSubscriptionRequest) returns (PauseSubscriptionReply);

message PauseSubscriptionRequest {
  uint64 uid = 1;
  string reason = 2;  // 暂停原因（可选）
}

message PauseSubscriptionReply {
  bool success = 1;
  string message = 2;
}
```

**示例**:
```go
resp, err := client.PauseSubscription(context.Background(), &subscriptionv1.PauseSubscriptionRequest{
    Uid:    1001,
    Reason: "临时不使用",
})
```

#### 7. 恢复订阅 (ResumeSubscription)

```protobuf
rpc ResumeSubscription (ResumeSubscriptionRequest) returns (ResumeSubscriptionReply);

message ResumeSubscriptionRequest {
  uint64 uid = 1;
}

message ResumeSubscriptionReply {
  bool success = 1;
  string message = 2;
}
```

**示例**:
```go
resp, err := client.ResumeSubscription(context.Background(), &subscriptionv1.ResumeSubscriptionRequest{
    Uid: 1001,
})
```

#### 8. 获取订阅历史 (GetSubscriptionHistory)

```protobuf
rpc GetSubscriptionHistory (GetSubscriptionHistoryRequest) returns (GetSubscriptionHistoryReply);

message GetSubscriptionHistoryRequest {
  uint64 uid = 1;
  int32 page = 2;       // 页码，从1开始
  int32 page_size = 3;  // 每页数量，默认10
}

message SubscriptionHistoryItem {
  uint64 id = 1;
  string plan_id = 2;
  string plan_name = 3;
  int64 start_time = 4;
  int64 end_time = 5;
  string status = 6;
  string action = 7;     // created, renewed, upgraded, paused, resumed, cancelled
  int64 created_at = 8;
}

message GetSubscriptionHistoryReply {
  repeated SubscriptionHistoryItem items = 1;
  int32 total = 2;
  int32 page = 3;
  int32 page_size = 4;
}
```

**示例**:
```go
resp, err := client.GetSubscriptionHistory(context.Background(), &subscriptionv1.GetSubscriptionHistoryRequest{
    Uid:      1001,
    Page:     1,
    PageSize: 10,
})

for _, item := range resp.Items {
    fmt.Printf("操作: %s, 套餐: %s, 时间: %s\n", 
        item.Action, item.PlanName, time.Unix(item.CreatedAt, 0))
}
```

#### 9. 设置自动续费 (SetAutoRenew)

```protobuf
rpc SetAutoRenew (SetAutoRenewRequest) returns (SetAutoRenewReply);

message SetAutoRenewRequest {
  uint64 uid = 1;
  bool auto_renew = 2;  // true: 开启, false: 关闭
}

message SetAutoRenewReply {
  bool success = 1;
  string message = 2;
}
```

**示例**:
```go
// 开启自动续费
resp, err := client.SetAutoRenew(context.Background(), &subscriptionv1.SetAutoRenewRequest{
    Uid:       1001,
    AutoRenew: true,
})

// 关闭自动续费
resp, err := client.SetAutoRenew(context.Background(), &subscriptionv1.SetAutoRenewRequest{
    Uid:       1001,
    AutoRenew: false,
})
```

#### 10. 获取即将过期的订阅 (GetExpiringSubscriptions)

**用途**: 用于定时任务，查询即将过期的订阅

```protobuf
rpc GetExpiringSubscriptions (GetExpiringSubscriptionsRequest) returns (GetExpiringSubscriptionsReply);

message GetExpiringSubscriptionsRequest {
  int32 days_before_expiry = 1;  // 过期前多少天，默认7天
  int32 page = 2;                // 页码，从1开始
  int32 page_size = 3;           // 每页数量，默认10
}

message GetExpiringSubscriptionsReply {
  repeated SubscriptionInfo subscriptions = 1;
  int32 total = 2;
  int32 page = 3;
  int32 page_size = 4;
}
```

#### 11. 批量更新过期订阅 (UpdateExpiredSubscriptions)

**用途**: 用于定时任务，批量更新过期订阅状态

```protobuf
rpc UpdateExpiredSubscriptions (UpdateExpiredSubscriptionsRequest) returns (UpdateExpiredSubscriptionsReply);

message UpdateExpiredSubscriptionsRequest {
  // 空请求，自动处理所有过期订阅
}

message UpdateExpiredSubscriptionsReply {
  int32 updated_count = 1;        // 更新的订阅数量
  repeated uint64 updated_uids = 2;  // 更新的用户ID列表
}
```

#### 12. 处理自动续费 (ProcessAutoRenewals)

**用途**: 用于定时任务，自动处理订阅续费

```protobuf
rpc ProcessAutoRenewals (ProcessAutoRenewalsRequest) returns (ProcessAutoRenewalsReply);

message ProcessAutoRenewalsRequest {
  int32 days_before_expiry = 1;  // 提前多少天续费，默认3天
  bool dry_run = 2;              // 是否为测试运行
}

message ProcessAutoRenewalsReply {
  int32 total_count = 1;      // 总共需要处理的数量
  int32 success_count = 2;    // 成功的数量
  int32 failed_count = 3;     // 失败的数量
  repeated AutoRenewResult results = 4;
}
```

### HTTP 接口

#### 健康检查
```bash
curl http://localhost:8102/health
```

响应示例：
```json
{
  "success": true,
  "data": {
    "status": "UP",
    "service": "subscription-service"
  }
}
```

#### 获取套餐列表
```bash
curl -X GET http://localhost:8102/v1/subscription/plans
```

响应示例：
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
  },
  "errorCode": "",
  "errorMessage": "",
  "showType": 0,
  "traceId": "20251126103134-GGGGGGGG",
  "host": "localhost:8102"
}
```

#### 获取我的订阅
```bash
curl -X GET http://localhost:8102/v1/subscription/my/1001
```

响应示例：
```json
{
  "success": true,
  "data": {
    "is_active": true,
    "plan_id": "plan_monthly",
    "start_time": 1700726400,
    "end_time": 1703318400,
    "status": "active"
  },
  "errorCode": "",
  "errorMessage": "",
  "showType": 0,
  "traceId": "20251126103134-GGGGGGGG",
  "host": "localhost:8102"
}
```

#### 创建订阅订单
```bash
curl -X POST http://localhost:8102/v1/subscription/order \
  -H "Content-Type: application/json" \
  -d '{
    "uid": 1001,
    "plan_id": "plan_monthly",
    "payment_method": "alipay"
  }'
```

响应示例：
```json
{
  "success": true,
  "data": {
    "order_id": "SUB1700726400001001",
    "payment_id": "PAY1700726400001",
    "pay_url": "https://openapi.alipay.com/gateway.do?...",
    "pay_code": "",
    "pay_params": ""
  },
  "errorCode": "",
  "errorMessage": "",
  "showType": 0,
  "traceId": "20251126103134-GGGGGGGG",
  "host": "localhost:8102"
}
```

### 响应格式

所有 HTTP 接口返回统一的响应格式：

```json
{
  "success": true,
  "data": {
    // 实际数据
  },
  "errorCode": "",
  "errorMessage": "",
  "showType": 0,
  "traceId": "20251126103134-GGGGGGGG",
  "host": "localhost:8102"
}
```

错误响应示例：

```json
{
  "success": false,
  "data": null,
  "errorCode": "PLAN_NOT_FOUND",
  "errorMessage": "plan not found",
  "showType": 2,
  "traceId": "20251126103134-GGGGGGGG",
  "host": "localhost:8102"
}
```

## 业务流程

### 订阅购买流程

```
1. 用户选择套餐
   ↓
2. 调用 CreateSubscriptionOrder
   ↓
3. Subscription Service 创建订单
   ↓
4. 调用 Payment Service 创建支付
   ↓
5. 返回支付链接给用户
   ↓
6. 用户完成支付
   ↓
7. Payment Service 回调 Subscription Service
   ↓
8. HandlePaymentSuccess 延长用户订阅
```

### 续费逻辑

- **首次购买**: 从当前时间开始计算有效期
- **续费（未过期）**: 在当前有效期基础上延长
- **续费（已过期）**: 从续费时间开始重新计算

## 快速开始

### 前置要求

- Go 1.21+
- MySQL 8.0+
- Payment Service（依赖服务）

### 本地开发

```bash
# 1. 克隆项目
git clone <repository-url>
cd subscription-service

# 2. 安装依赖
go mod tidy

# 3. 初始化数据库
mysql -u root -p < docs/sql/subscription.sql

# 4. 配置文件
# 编辑 configs/config.yaml，修改数据库连接等配置

# 5. 启动服务

# 方式1: 只启动主服务
make run

# 方式2: 启动所有服务（主服务 + Cron 服务）
bash script/restart_server.sh
# 或
make build-all && make run-all

# 方式3: 分别启动
# 终端1: 启动主服务
make run
# 终端2: 启动 Cron 服务
make run-cron
```

### 开发工具

```bash
# 安装开发工具
make init

# 生成 API 代码（修改 proto 后）
make api

# 生成依赖注入代码（修改 wire.go 后）
make wire

# 编译主服务
make build

# 编译 Cron 服务
make build-cron

# 编译所有服务
make build-all

# 运行测试
make test

# 停止所有服务
make stop-all

# 查看所有命令
make help
```

### Docker 部署

```bash
# 构建镜像
docker build -t subscription-service:latest .

# 运行容器
docker run -d \
  -p 9102:9102 \
  -v $(pwd)/configs:/app/configs \
  -v $(pwd)/logs:/app/logs \
  --name subscription-service \
  subscription-service:latest
```

### Docker Compose

```bash
# 在项目根目录
docker-compose up -d subscription-service
```

## 配置说明

配置文件位于 `configs/config.yaml`：

```yaml
server:
  http:
    addr: 0.0.0.0:8102
    timeout: 1s
  grpc:
    addr: 0.0.0.0:9102
    timeout: 1s

data:
  database:
    driver: mysql
    source: root:@tcp(localhost:3306)/subscription_service?charset=utf8mb4&parseTime=True&loc=Local

client:
  payment:
    addr: localhost:9101  # Payment Service 地址

log:
  level: info
  format: json
  output: both  # stdout, file, both
  file_path: logs/subscription-service.log
  max_size: 100  # MB
  max_age: 30    # days
  max_backups: 10
  compress: true
```

### 配置项说明

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `server.http.addr` | HTTP 服务地址 | 0.0.0.0:8102 |
| `server.grpc.addr` | gRPC 服务地址 | 0.0.0.0:9102 |
| `data.database.source` | MySQL 连接字符串 | - |
| `client.payment.addr` | Payment Service 地址 | localhost:9101 |
| `log.level` | 日志级别 | info |
| `log.format` | 日志格式 (json/text) | json |
| `log.output` | 日志输出 (stdout/file/both) | both |

详细配置说明请参考 [docs/CONFIG.md](docs/CONFIG.md)

## 数据库

### MySQL

统一使用 MySQL 数据库（开发和生产环境）：

**数据库名**: `subscription_service`

**配置示例**:
```yaml
data:
  database:
    driver: mysql
    source: root:root@tcp(localhost:3306)/subscription_service?charset=utf8mb4&parseTime=True&loc=Local
```

**Docker 快速启动**:
```bash
# 使用 docker-compose 启动 MySQL
docker-compose up -d mysql

# 数据库会自动初始化（通过 init-db/02-init.sql）
```

建表语句请参考 [docs/sql/subscription.sql](docs/sql/subscription.sql)

## 测试

### 运行测试

项目包含全面的 API 测试，覆盖所有功能场景：

```bash
# 运行所有测试
make test

# 清理数据库后测试
mysql -u root -D subscription_service -e "TRUNCATE TABLE user_subscription; TRUNCATE TABLE subscription_order;" && make test
```

### 测试覆盖

| 测试类别 | 说明 |
|---------|------|
| 健康检查 | 服务健康状态 |
| 套餐管理 | 套餐列表查询、数据验证 |
| 订阅查询 | 用户订阅状态查询 |
| 订阅购买 | 创建订阅订单流程 |
| 支付回调 | 支付成功处理、订阅激活 |
| 订阅续费 | 续费流程、订阅延长 |
| 订阅升级 | 套餐升级流程 |
| 异常场景 | 边界条件、错误处理 |

测试报告会自动生成在 `test-reports/` 目录下。

## 套餐管理

### 添加新套餐

直接在数据库中插入：

```sql
INSERT INTO plan (plan_id, name, description, price, currency, duration_days, type)
VALUES ('plan_quarterly', 'Pro Quarterly', 'Pro features for 3 months', 25.99, 'CNY', 90, 'pro');
```

或在代码中初始化：

```go
func initPlans(db *gorm.DB, log *logrus.Logger) {
    plans := []data.Plan{
        {PlanID: "plan_monthly", Name: "Pro Monthly", Price: 9.99, DurationDays: 30, Type: "pro"},
        {PlanID: "plan_yearly", Name: "Pro Yearly", Price: 99.99, DurationDays: 365, Type: "pro"},
        {PlanID: "plan_quarterly", Name: "Pro Quarterly", Price: 25.99, DurationDays: 90, Type: "pro"},
    }
    db.Create(&plans)
}
```

## 监控和日志

### 日志

日志配置支持：
- **格式**: JSON 或 Text
- **输出**: 控制台、文件或同时输出
- **轮转**: 自动日志轮转（按大小、时间）
- **压缩**: 自动压缩旧日志

日志文件位置：`logs/subscription-service.log`

查看日志：
```bash
# 实时查看
tail -f logs/subscription-service.log

# 查看最近的错误
grep "ERROR" logs/subscription-service.log
```

### 健康检查

```bash
# HTTP 健康检查
curl http://localhost:8102/health

# 预期响应
{
  "success": true,
  "data": {
    "status": "UP",
    "service": "subscription-service"
  }
}
```

### 监控指标

建议监控的关键指标：
- API 响应时间
- 错误率
- 订阅转化率
- 续费率
- 各套餐销售分布
- Payment Service 调用成功率
- 数据库连接池状态

## 故障排查

### 常见问题

1. **无法连接数据库**
   ```bash
   # 检查 MySQL 是否运行
   mysql -u root -p
   
   # 检查数据库是否存在
   SHOW DATABASES LIKE 'subscription_service';
   
   # 检查配置文件
   cat configs/config.yaml | grep source
   ```

2. **无法连接 Payment Service**
   ```bash
   # 检查 Payment Service 是否启动
   curl http://localhost:8101/health
   
   # 检查配置文件中的地址
   cat configs/config.yaml | grep payment
   ```

3. **支付成功但订阅未延长**
   - 检查 HandlePaymentSuccess 日志
   - 检查订单状态
   - 检查数据库事务
   ```bash
   # 查看订单状态
   mysql -u root -D subscription_service -e "SELECT * FROM subscription_order WHERE order_id='订单号';"
   
   # 查看用户订阅状态
   mysql -u root -D subscription_service -e "SELECT * FROM user_subscription WHERE user_id=用户ID;"
   ```

4. **订阅状态不准确**
   - 检查服务器时区设置
   - 检查时间计算逻辑
   ```bash
   # 检查系统时区
   date
   timedatectl
   ```

5. **端口被占用**
   ```bash
   # 查看端口占用
   lsof -i:8102
   lsof -i:9102
   
   # 停止占用端口的进程
   lsof -ti:8102 -ti:9102 | xargs kill -9
   
   # 或修改配置文件中的端口号
   ```

6. **日志文件权限错误**
   ```bash
   # 确保 logs 目录有写权限
   chmod 755 logs/
   ```

### 调试技巧

```bash
# 查看服务日志
tail -f logs/subscription-service.log

# 查看最近的错误
grep "ERROR" logs/subscription-service.log | tail -20

# 查看特定用户的操作
grep "uid=1" logs/subscription-service.log

# 测试数据库连接
mysql -u root -D subscription_service -e "SELECT COUNT(*) FROM plan;"

# 测试 Payment Service 连接
grpcurl -plaintext localhost:9101 list
```

## 项目结构

```
subscription-service/
├── api/                    # API 定义
│   └── subscription/v1/    # Protobuf 定义
├── cmd/                    # 应用入口
│   └── server/             # 服务启动代码
├── configs/                # 配置文件
│   └── config.yaml         # 主配置文件
├── docs/                   # 文档
│   ├── sql/                # SQL 脚本
│   └── CONFIG.md           # 配置文档
├── internal/               # 内部代码
│   ├── biz/                # 业务逻辑层
│   ├── conf/               # 配置结构
│   ├── data/               # 数据访问层
│   ├── logger/             # 日志模块
│   ├── server/             # 服务器配置
│   └── service/            # 服务实现层
├── logs/                   # 日志文件
├── reports/                # 测试报告
├── api-test-config.yaml    # API 测试配置
├── Makefile                # 构建脚本
└── README.md               # 本文档
```

## 相关文档

- [API 定义](api/subscription/v1/subscription.proto) - Protobuf 接口定义
- [SQL 脚本](docs/sql/subscription.sql) - 数据库建表语句
- [配置文档](docs/CONFIG.md) - 详细配置说明
- [API 示例](docs/API_EXAMPLES.md) - API 调用示例

## 开发指南

### 添加新接口

1. 修改 `api/subscription/v1/subscription.proto`
2. 运行 `make api` 生成代码
3. 在 `internal/service/subscription.go` 实现接口
4. 在 `internal/biz/subscription.go` 添加业务逻辑
5. 在 `internal/data/subscription.go` 添加数据访问
6. 添加测试用例到 `api-test-config.yaml`
7. 运行 `make test` 验证

### 代码规范

- 遵循 Go 官方代码规范
- 使用 `gofmt` 格式化代码
- 注释使用中文
- 日志和错误消息使用英文
- 遵循 Kratos 框架最佳实践

## 扩展建议

1. **自动续费**: 实现订阅到期前自动扣款
2. **优惠券**: 支持优惠码和折扣
3. **试用期**: 支持免费试用
4. **家庭套餐**: 支持多用户共享
5. **降级策略**: 订阅到期后的降级处理
6. **订阅暂停**: 支持暂停和恢复订阅
7. **订阅转让**: 支持订阅权益转让

## 贡献指南

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License