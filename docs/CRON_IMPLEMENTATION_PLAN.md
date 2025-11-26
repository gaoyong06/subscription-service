# Cron 服务实现计划

## 背景

基于 Schedule Manager 的需求分析，Subscription-Service 需要补充以下核心功能：

1. 批量查询即将过期的订阅
2. 批量更新过期订阅状态
3. 自动续费处理逻辑
4. 独立的 Cron 定时任务服务

## 实现步骤

### Step 1: 更新 Proto 定义

添加新的 RPC 方法到 `api/subscription/v1/subscription.proto`:

```protobuf
// 获取即将过期的订阅
rpc GetExpiringSubscriptions (GetExpiringSubscriptionsRequest) returns (GetExpiringSubscriptionsReply) {
  option (google.api.http) = {
    get: "/v1/subscription/expiring"
  };
}

// 批量更新过期订阅状态
rpc UpdateExpiredSubscriptions (UpdateExpiredSubscriptionsRequest) returns (UpdateExpiredSubscriptionsReply) {
  option (google.api.http) = {
    post: "/v1/subscription/expired/update"
    body: "*"
  };
}

// 处理自动续费
rpc ProcessAutoRenewals (ProcessAutoRenewalsRequest) returns (ProcessAutoRenewalsReply) {
  option (google.api.http) = {
    post: "/v1/subscription/auto-renew/process"
    body: "*"
  };
}
```

**消息定义**:

```protobuf
// 获取即将过期的订阅
message GetExpiringSubscriptionsRequest {
  int32 days_before_expiry = 1 [(validate.rules).int32 = {gte: 1, lte: 30}];  // 默认7天
  int32 page = 2;
  int32 page_size = 3;
}

message SubscriptionInfo {
  uint64 uid = 1;
  string plan_id = 2;
  string plan_name = 3;
  int64 start_time = 4;
  int64 end_time = 5;
  bool auto_renew = 6;
  double amount = 7;
}

message GetExpiringSubscriptionsReply {
  repeated SubscriptionInfo subscriptions = 1;
  int32 total = 2;
  int32 page = 3;
  int32 page_size = 4;
}

// 批量更新过期订阅
message UpdateExpiredSubscriptionsRequest {
  // 空请求，自动处理所有过期订阅
}

message UpdateExpiredSubscriptionsReply {
  int32 updated_count = 1;
  repeated uint64 updated_uids = 2;  // 更新的用户ID列表
}

// 自动续费处理
message ProcessAutoRenewalsRequest {
  int32 days_before_expiry = 1 [(validate.rules).int32 = {gte: 1, lte: 30}];  // 默认3天
  bool dry_run = 2;  // 是否为测试运行
}

message AutoRenewResult {
  uint64 uid = 1;
  string plan_id = 2;
  bool success = 3;
  string order_id = 4;
  string payment_id = 5;
  string error_message = 6;
}

message ProcessAutoRenewalsReply {
  int32 total_count = 1;      // 总共需要处理的数量
  int32 success_count = 2;    // 成功的数量
  int32 failed_count = 3;     // 失败的数量
  repeated AutoRenewResult results = 4;
}
```

### Step 2: 实现 Biz 层逻辑

在 `internal/biz/subscription.go` 添加方法：

```go
// GetExpiringSubscriptions 获取即将过期的订阅
func (uc *SubscriptionUsecase) GetExpiringSubscriptions(ctx context.Context, daysBeforeExpiry, page, pageSize int) ([]*UserSubscription, int, error) {
	// 参数验证
	if daysBeforeExpiry < 1 || daysBeforeExpiry > 30 {
		daysBeforeExpiry = 7
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 调用 repo 查询
	subscriptions, total, err := uc.repo.GetExpiringSubscriptions(ctx, daysBeforeExpiry, page, pageSize)
	if err != nil {
		uc.log.Errorf("Failed to get expiring subscriptions: %v", err)
		return nil, 0, err
	}

	uc.log.Infof("Found %d expiring subscriptions (within %d days)", total, daysBeforeExpiry)
	return subscriptions, total, nil
}

// UpdateExpiredSubscriptions 批量更新过期订阅状态
func (uc *SubscriptionUsecase) UpdateExpiredSubscriptions(ctx context.Context) (int, []uint64, error) {
	uc.log.Infof("Starting to update expired subscriptions")

	// 调用 repo 批量更新
	count, uids, err := uc.repo.UpdateExpiredSubscriptions(ctx)
	if err != nil {
		uc.log.Errorf("Failed to update expired subscriptions: %v", err)
		return 0, nil, err
	}

	// 为每个过期的订阅添加历史记录
	now := time.Now().UTC()
	for _, uid := range uids {
		// 获取订阅信息
		sub, err := uc.repo.GetSubscription(ctx, uid)
		if err != nil {
			uc.log.Errorf("Failed to get subscription for user %d: %v", uid, err)
			continue
		}
		if sub == nil {
			continue
		}

		// 添加历史记录
		history := &SubscriptionHistory{
			UserID:    uid,
			PlanID:    sub.PlanID,
			StartTime: sub.StartTime,
			EndTime:   sub.EndTime,
			Status:    "expired",
			Action:    "expired",
			CreatedAt: now,
		}
		if err := uc.repo.AddSubscriptionHistory(ctx, history); err != nil {
			uc.log.Errorf("Failed to add history for user %d: %v", uid, err)
		}
	}

	uc.log.Infof("Updated %d expired subscriptions", count)
	return count, uids, nil
}

// ProcessAutoRenewals 处理自动续费
func (uc *SubscriptionUsecase) ProcessAutoRenewals(ctx context.Context, daysBeforeExpiry int, dryRun bool) (int, int, int, []*AutoRenewResult, error) {
	uc.log.Infof("Starting auto-renewal process (daysBeforeExpiry=%d, dryRun=%v)", daysBeforeExpiry, dryRun)

	// 参数验证
	if daysBeforeExpiry < 1 || daysBeforeExpiry > 30 {
		daysBeforeExpiry = 3
	}

	// 获取需要自动续费的订阅
	subscriptions, err := uc.repo.GetAutoRenewSubscriptions(ctx, daysBeforeExpiry)
	if err != nil {
		uc.log.Errorf("Failed to get auto-renew subscriptions: %v", err)
		return 0, 0, 0, nil, err
	}

	totalCount := len(subscriptions)
	successCount := 0
	failedCount := 0
	results := make([]*AutoRenewResult, 0, totalCount)

	for _, sub := range subscriptions {
		result := &AutoRenewResult{
			UID:    sub.UserID,
			PlanID: sub.PlanID,
		}

		if dryRun {
			// 测试模式，只记录不执行
			result.Success = true
			result.ErrorMessage = "dry run - not executed"
			uc.log.Infof("[DRY RUN] Would renew subscription for user %d, plan %s", sub.UserID, sub.PlanID)
		} else {
			// 实际执行续费
			order, paymentID, _, _, _, err := uc.CreateSubscriptionOrder(ctx, sub.UserID, sub.PlanID, "auto")
			if err != nil {
				result.Success = false
				result.ErrorMessage = err.Error()
				failedCount++
				uc.log.Errorf("Failed to create renewal order for user %d: %v", sub.UserID, err)
			} else {
				result.Success = true
				result.OrderID = order.ID
				result.PaymentID = paymentID
				successCount++
				uc.log.Infof("Successfully created renewal order for user %d: %s", sub.UserID, order.ID)

				// 如果是自动续费，直接处理支付成功（模拟自动扣款）
				// 实际生产环境中，这里应该调用支付服务的自动扣款接口
				// 这里简化处理，假设自动扣款成功
				if err := uc.HandlePaymentSuccess(ctx, order.ID, order.Amount); err != nil {
					uc.log.Errorf("Failed to handle payment success for order %s: %v", order.ID, err)
					result.ErrorMessage = "order created but payment failed: " + err.Error()
					result.Success = false
					failedCount++
					successCount--
				}
			}
		}

		results = append(results, result)
	}

	uc.log.Infof("Auto-renewal process completed: total=%d, success=%d, failed=%d", totalCount, successCount, failedCount)
	return totalCount, successCount, failedCount, results, nil
}
```

### Step 3: 实现 Data 层方法

在 `internal/data/subscription.go` 添加方法：

```go
// GetExpiringSubscriptions 获取即将过期的订阅
func (r *subscriptionRepo) GetExpiringSubscriptions(ctx context.Context, daysBeforeExpiry, page, pageSize int) ([]*biz.UserSubscription, int, error) {
	var models []UserSubscription
	var total int64

	now := time.Now().UTC()
	expiryDate := now.AddDate(0, 0, daysBeforeExpiry)

	// 获取总数
	if err := r.data.db.WithContext(ctx).Model(&UserSubscription{}).
		Where("end_time BETWEEN ? AND ? AND status = ?", now, expiryDate, "active").
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := r.data.db.WithContext(ctx).
		Where("end_time BETWEEN ? AND ? AND status = ?", now, expiryDate, "active").
		Order("end_time ASC").
		Limit(pageSize).
		Offset(offset).
		Find(&models).Error; err != nil {
		return nil, 0, err
	}

	// 转换为业务对象
	subscriptions := make([]*biz.UserSubscription, len(models))
	for i, m := range models {
		subscriptions[i] = &biz.UserSubscription{
			ID:        m.ID,
			UserID:    m.UserID,
			PlanID:    m.PlanID,
			StartTime: m.StartTime,
			EndTime:   m.EndTime,
			Status:    m.Status,
			AutoRenew: m.AutoRenew,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		}
	}

	return subscriptions, int(total), nil
}

// UpdateExpiredSubscriptions 批量更新过期订阅状态
func (r *subscriptionRepo) UpdateExpiredSubscriptions(ctx context.Context) (int, []uint64, error) {
	now := time.Now().UTC()

	// 先查询需要更新的订阅
	var subscriptions []UserSubscription
	if err := r.data.db.WithContext(ctx).
		Where("end_time < ? AND status = ?", now, "active").
		Find(&subscriptions).Error; err != nil {
		return 0, nil, err
	}

	if len(subscriptions) == 0 {
		return 0, []uint64{}, nil
	}

	// 提取 user_id 列表
	uids := make([]uint64, len(subscriptions))
	for i, sub := range subscriptions {
		uids[i] = sub.UserID
	}

	// 批量更新状态
	result := r.data.db.WithContext(ctx).Model(&UserSubscription{}).
		Where("end_time < ? AND status = ?", now, "active").
		Update("status", "expired")

	if result.Error != nil {
		return 0, nil, result.Error
	}

	return int(result.RowsAffected), uids, nil
}

// GetAutoRenewSubscriptions 获取需要自动续费的订阅
func (r *subscriptionRepo) GetAutoRenewSubscriptions(ctx context.Context, daysBeforeExpiry int) ([]*biz.UserSubscription, error) {
	var models []UserSubscription

	now := time.Now().UTC()
	expiryDate := now.AddDate(0, 0, daysBeforeExpiry)

	// 查询即将过期且开启了自动续费的订阅
	if err := r.data.db.WithContext(ctx).
		Where("end_time BETWEEN ? AND ? AND status = ? AND auto_renew = ?", 
			now, expiryDate, "active", true).
		Order("end_time ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}

	// 转换为业务对象
	subscriptions := make([]*biz.UserSubscription, len(models))
	for i, m := range models {
		subscriptions[i] = &biz.UserSubscription{
			ID:        m.ID,
			UserID:    m.UserID,
			PlanID:    m.PlanID,
			StartTime: m.StartTime,
			EndTime:   m.EndTime,
			Status:    m.Status,
			AutoRenew: m.AutoRenew,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		}
	}

	return subscriptions, nil
}
```

### Step 4: 更新 Biz 接口定义

在 `internal/biz/subscription.go` 的 `SubscriptionRepo` 接口添加：

```go
type SubscriptionRepo interface {
	// ... 现有方法 ...
	
	// 新增方法
	GetExpiringSubscriptions(ctx context.Context, daysBeforeExpiry, page, pageSize int) ([]*UserSubscription, int, error)
	UpdateExpiredSubscriptions(ctx context.Context) (int, []uint64, error)
	GetAutoRenewSubscriptions(ctx context.Context, daysBeforeExpiry int) ([]*UserSubscription, error)
}
```

添加新的结构体：

```go
// AutoRenewResult 自动续费结果
type AutoRenewResult struct {
	UID          uint64
	PlanID       string
	Success      bool
	OrderID      string
	PaymentID    string
	ErrorMessage string
}
```

### Step 5: 实现 Service 层

在 `internal/service/subscription.go` 添加方法：

```go
func (s *SubscriptionService) GetExpiringSubscriptions(ctx context.Context, req *pb.GetExpiringSubscriptionsRequest) (*pb.GetExpiringSubscriptionsReply, error) {
	daysBeforeExpiry := int(req.DaysBeforeExpiry)
	if daysBeforeExpiry == 0 {
		daysBeforeExpiry = 7
	}
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 {
		pageSize = 10
	}

	subscriptions, total, err := s.uc.GetExpiringSubscriptions(ctx, daysBeforeExpiry, page, pageSize)
	if err != nil {
		return nil, err
	}

	pbSubscriptions := make([]*pb.SubscriptionInfo, len(subscriptions))
	for i, sub := range subscriptions {
		// 获取套餐信息
		plan, _ := s.uc.GetPlan(ctx, sub.PlanID)
		planName := sub.PlanID
		amount := 0.0
		if plan != nil {
			planName = plan.Name
			amount = plan.Price
		}

		pbSubscriptions[i] = &pb.SubscriptionInfo{
			Uid:       sub.UserID,
			PlanId:    sub.PlanID,
			PlanName:  planName,
			StartTime: sub.StartTime.Unix(),
			EndTime:   sub.EndTime.Unix(),
			AutoRenew: sub.AutoRenew,
			Amount:    amount,
		}
	}

	return &pb.GetExpiringSubscriptionsReply{
		Subscriptions: pbSubscriptions,
		Total:         int32(total),
		Page:          int32(page),
		PageSize:      int32(pageSize),
	}, nil
}

func (s *SubscriptionService) UpdateExpiredSubscriptions(ctx context.Context, req *pb.UpdateExpiredSubscriptionsRequest) (*pb.UpdateExpiredSubscriptionsReply, error) {
	count, uids, err := s.uc.UpdateExpiredSubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateExpiredSubscriptionsReply{
		UpdatedCount: int32(count),
		UpdatedUids:  uids,
	}, nil
}

func (s *SubscriptionService) ProcessAutoRenewals(ctx context.Context, req *pb.ProcessAutoRenewalsRequest) (*pb.ProcessAutoRenewalsReply, error) {
	daysBeforeExpiry := int(req.DaysBeforeExpiry)
	if daysBeforeExpiry == 0 {
		daysBeforeExpiry = 3
	}

	totalCount, successCount, failedCount, results, err := s.uc.ProcessAutoRenewals(ctx, daysBeforeExpiry, req.DryRun)
	if err != nil {
		return nil, err
	}

	pbResults := make([]*pb.AutoRenewResult, len(results))
	for i, result := range results {
		pbResults[i] = &pb.AutoRenewResult{
			Uid:          result.UID,
			PlanId:       result.PlanID,
			Success:      result.Success,
			OrderId:      result.OrderID,
			PaymentId:    result.PaymentID,
			ErrorMessage: result.ErrorMessage,
		}
	}

	return &pb.ProcessAutoRenewalsReply{
		TotalCount:   int32(totalCount),
		SuccessCount: int32(successCount),
		FailedCount:  int32(failedCount),
		Results:      pbResults,
	}, nil
}
```

### Step 6: 创建 Cron 服务

创建 `cmd/cron/main.go`:

```go
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"xinyuan_tech/subscription-service/internal/conf"
	"xinyuan_tech/subscription-service/internal/data"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/robfig/cron/v3"
	_ "go.uber.org/automaxprocs"
)

var (
	flagconf string
)

func init() {
	flag.StringVar(&flagconf, "conf", "configs/config.yaml", "config path, eg: -conf config.yaml")
}

func main() {
	flag.Parse()

	// 初始化配置
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	// 初始化数据库
	dataData, cleanup, err := data.NewData(&bc, nil)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// 初始化服务
	app, cleanup2, err := wireApp(&bc, nil, dataData)
	if err != nil {
		panic(err)
	}
	defer cleanup2()

	// 创建定时任务调度器
	cronScheduler := cron.New(cron.WithSeconds())

	// 1. 订阅过期检查 - 每天凌晨 2 点执行
	_, err = cronScheduler.AddFunc("0 0 2 * * *", func() {
		log.Println("[CRON] Starting subscription expiration check...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if _, _, err := app.subscriptionUsecase.UpdateExpiredSubscriptions(ctx); err != nil {
			log.Printf("[CRON] Error updating expired subscriptions: %v", err)
		} else {
			log.Println("[CRON] Finished subscription expiration check")
		}
	})
	if err != nil {
		log.Printf("Failed to add expiration check job: %v", err)
	}

	// 2. 续费提醒 - 每天上午 10 点执行
	_, err = cronScheduler.AddFunc("0 0 10 * * *", func() {
		log.Println("[CRON] Starting renewal reminder check...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		subscriptions, total, err := app.subscriptionUsecase.GetExpiringSubscriptions(ctx, 7, 1, 100)
		if err != nil {
			log.Printf("[CRON] Error getting expiring subscriptions: %v", err)
			return
		}

		log.Printf("[CRON] Found %d subscriptions expiring within 7 days", total)
		for _, sub := range subscriptions {
			// TODO: 发送续费提醒通知
			log.Printf("[CRON] Reminder: User %d subscription expires at %s", sub.UserID, sub.EndTime)
		}
	})
	if err != nil {
		log.Printf("Failed to add renewal reminder job: %v", err)
	}

	// 3. 自动续费处理 - 每天凌晨 3 点执行
	_, err = cronScheduler.AddFunc("0 0 3 * * *", func() {
		log.Println("[CRON] Starting auto-renewal process...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		totalCount, successCount, failedCount, _, err := app.subscriptionUsecase.ProcessAutoRenewals(ctx, 3, false)
		if err != nil {
			log.Printf("[CRON] Error processing auto-renewals: %v", err)
		} else {
			log.Printf("[CRON] Auto-renewal completed: total=%d, success=%d, failed=%d", 
				totalCount, successCount, failedCount)
		}
	})
	if err != nil {
		log.Printf("Failed to add auto-renewal job: %v", err)
	}

	// 启动定时任务
	cronScheduler.Start()
	log.Println("Cron jobs started successfully")

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gracefully...")

	// 停止定时任务
	ctx := cronScheduler.Stop()
	select {
	case <-ctx.Done():
		log.Println("Cron jobs stopped gracefully")
	case <-time.After(5 * time.Second):
		log.Println("Cron jobs forced to stop after timeout")
	}
}
```

创建 `cmd/cron/wire.go`:

```go
//go:build wireinject
// +build wireinject

package main

import (
	"xinyuan_tech/subscription-service/internal/biz"
	"xinyuan_tech/subscription-service/internal/conf"
	"xinyuan_tech/subscription-service/internal/data"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

type CronApp struct {
	subscriptionUsecase *biz.SubscriptionUsecase
}

func wireApp(*conf.Bootstrap, log.Logger, *data.Data) (*CronApp, func(), error) {
	panic(wire.Build(
		data.ProviderSet,
		biz.ProviderSet,
		wire.Struct(new(CronApp), "*"),
	))
}
```

### Step 7: 更新 Makefile

添加 cron 服务的编译命令：

```makefile
.PHONY: build-cron
# 编译 cron 服务
build-cron:
	mkdir -p bin/ && go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/cron ./cmd/cron

.PHONY: run-cron
# 运行 cron 服务
run-cron:
	./bin/cron -conf ./configs/config.yaml
```

## 测试计划

### 单元测试

1. 测试 `GetExpiringSubscriptions`
2. 测试 `UpdateExpiredSubscriptions`
3. 测试 `ProcessAutoRenewals`

### 集成测试

1. 创建测试订阅数据
2. 测试过期检查逻辑
3. 测试自动续费流程
4. 测试 Cron 任务执行

### 手动测试

```bash
# 1. 编译服务
make build
make build-cron

# 2. 启动主服务
./bin/server -conf configs/config.yaml

# 3. 测试批量查询接口
curl "http://localhost:8102/v1/subscription/expiring?days_before_expiry=7&page=1&page_size=10"

# 4. 测试批量更新接口
curl -X POST http://localhost:8102/v1/subscription/expired/update

# 5. 测试自动续费接口（dry run）
curl -X POST http://localhost:8102/v1/subscription/auto-renew/process \
  -H "Content-Type: application/json" \
  -d '{"days_before_expiry": 3, "dry_run": true}'

# 6. 启动 Cron 服务
./bin/cron -conf configs/config.yaml
```

## 部署说明

### Supervisor 配置

创建 `deploy/supervisor/subscription-cron.conf`:

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

### Docker 部署

更新 `docker-compose.yml`:

```yaml
services:
  subscription-service:
    # ... 主服务配置 ...

  subscription-cron:
    build: .
    command: /app/cron -conf /app/configs/config.yaml
    volumes:
      - ./configs:/app/configs
      - ./logs:/app/logs
    depends_on:
      - mysql
      - redis
    restart: unless-stopped
```

## 监控指标

添加 Prometheus 指标：

```go
var (
	expiredSubscriptionsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "subscription_expired_total",
		Help: "Total number of expired subscriptions processed",
	})

	autoRenewalsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "subscription_auto_renewals_total",
		Help: "Total number of auto-renewals processed",
	}, []string{"status"}) // status: success, failed

	expiringSubscriptionsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "subscription_expiring_count",
		Help: "Number of subscriptions expiring soon",
	})
)
```

## 下一步

1. ✅ 完成 Proto 定义
2. ✅ 实现 Biz 层逻辑
3. ✅ 实现 Data 层方法
4. ✅ 实现 Service 层
5. ✅ 创建 Cron 服务
6. ⏳ 编写测试用例
7. ⏳ 集成测试
8. ⏳ 部署上线

