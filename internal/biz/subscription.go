package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// Plan 订阅套餐
type Plan struct {
	ID           string
	Name         string
	Description  string
	Price        float64
	Currency     string
	DurationDays int
	Type         string
}

// UserSubscription 用户订阅记录
type UserSubscription struct {
	ID        uint64
	UserID    uint64
	PlanID    string
	StartTime time.Time
	EndTime   time.Time
	Status    string // active, expired, paused, cancelled
	AutoRenew bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SubscriptionHistory 订阅历史记录
type SubscriptionHistory struct {
	ID        uint64
	UserID    uint64
	PlanID    string
	PlanName  string
	StartTime time.Time
	EndTime   time.Time
	Status    string
	Action    string // created, renewed, upgraded, paused, resumed, cancelled
	CreatedAt time.Time
}

// Order 简易订单记录 (用于记录订阅购买请求)
type Order struct {
	ID            string
	UserID        uint64
	PlanID        string
	Amount        float64
	PaymentStatus string // pending, paid
	CreatedAt     time.Time
}

// AutoRenewResult 自动续费结果
type AutoRenewResult struct {
	UID          uint64
	PlanID       string
	Success      bool
	OrderID      string
	PaymentID    string
	ErrorMessage string
}

// SubscriptionRepo 数据层接口
type SubscriptionRepo interface {
	ListPlans(ctx context.Context) ([]*Plan, error)
	GetPlan(ctx context.Context, id string) (*Plan, error)
	GetSubscription(ctx context.Context, userID uint64) (*UserSubscription, error)
	SaveSubscription(ctx context.Context, sub *UserSubscription) error
	CreateOrder(ctx context.Context, order *Order) error
	GetOrder(ctx context.Context, orderID string) (*Order, error)
	UpdateOrder(ctx context.Context, order *Order) error
	// 订阅历史
	AddSubscriptionHistory(ctx context.Context, history *SubscriptionHistory) error
	GetSubscriptionHistory(ctx context.Context, userID uint64, page, pageSize int) ([]*SubscriptionHistory, int, error)
	// 批量操作（用于定时任务）
	GetExpiringSubscriptions(ctx context.Context, daysBeforeExpiry, page, pageSize int) ([]*UserSubscription, int, error)
	UpdateExpiredSubscriptions(ctx context.Context) (int, []uint64, error)
	GetAutoRenewSubscriptions(ctx context.Context, daysBeforeExpiry int) ([]*UserSubscription, error)
}

// PaymentClient 支付服务客户端接口 (防腐层)
type PaymentClient interface {
	CreatePayment(ctx context.Context, orderID string, userID uint64, amount float64, method, subject, returnURL string) (paymentID, payUrl, payCode, payParams string, err error)
}

// SubscriptionUsecase 订阅业务逻辑
type SubscriptionUsecase struct {
	repo          SubscriptionRepo
	paymentClient PaymentClient
	log           *log.Helper
}

func NewSubscriptionUsecase(repo SubscriptionRepo, paymentClient PaymentClient, logger log.Logger) *SubscriptionUsecase {
	return &SubscriptionUsecase{
		repo:          repo,
		paymentClient: paymentClient,
		log:           log.NewHelper(logger),
	}
}

func (uc *SubscriptionUsecase) ListPlans(ctx context.Context) ([]*Plan, error) {
	return uc.repo.ListPlans(ctx)
}

func (uc *SubscriptionUsecase) GetMySubscription(ctx context.Context, userID uint64) (*UserSubscription, error) {
	sub, err := uc.repo.GetSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 检查是否过期
	if sub != nil && sub.EndTime.Before(time.Now().UTC()) {
		sub.Status = "expired"
		// 可以在这里异步更新数据库状态
	}

	return sub, nil
}

func (uc *SubscriptionUsecase) CreateSubscriptionOrder(ctx context.Context, userID uint64, planID, method string) (*Order, string, string, string, string, error) {
	uc.log.Infof("CreateSubscriptionOrder: userID=%d, planID=%s, method=%s", userID, planID, method)
	
	// 1. 获取套餐信息
	plan, err := uc.repo.GetPlan(ctx, planID)
	if err != nil {
		uc.log.Errorf("Failed to get plan: %v", err)
		return nil, "", "", "", "", err
	}
	if plan == nil {
		uc.log.Errorf("Plan not found: %s", planID)
		return nil, "", "", "", "", fmt.Errorf("plan not found")
	}
	uc.log.Infof("Found plan: %s, price: %.2f", plan.Name, plan.Price)

	// 2. 创建本地订单
	orderID := fmt.Sprintf("SUB%d%d", time.Now().UnixNano(), userID)
	order := &Order{
		ID:            orderID,
		UserID:        userID,
		PlanID:        planID,
		Amount:        plan.Price,
		PaymentStatus: "pending",
		CreatedAt:     time.Now().UTC(),
	}
	if err := uc.repo.CreateOrder(ctx, order); err != nil {
		uc.log.Errorf("Failed to create order: %v", err)
		return nil, "", "", "", "", err
	}
	uc.log.Infof("Created order: %s", orderID)

	// 3. 调用支付服务
	// TODO: ReturnURL 应该从配置中获取
	returnURL := "http://localhost:8080/subscription/success"
	uc.log.Infof("Calling payment service: orderID=%s, amount=%.2f, method=%s", orderID, plan.Price, method)
	paymentID, payUrl, payCode, payParams, err := uc.paymentClient.CreatePayment(ctx, orderID, userID, plan.Price, method, "Subscription: "+plan.Name, returnURL)
	if err != nil {
		uc.log.Errorf("Failed to create payment: %v", err)
		return nil, "", "", "", "", err
	}
	uc.log.Infof("Payment created: paymentID=%s", paymentID)

	return order, paymentID, payUrl, payCode, payParams, nil
}

func (uc *SubscriptionUsecase) HandlePaymentSuccess(ctx context.Context, orderID string, amount float64) error {
	uc.log.Infof("HandlePaymentSuccess: orderID=%s, amount=%.2f", orderID, amount)
	
	// 1. 获取订单
	order, err := uc.repo.GetOrder(ctx, orderID)
	if err != nil {
		uc.log.Errorf("Failed to get order: %v", err)
		return err
	}
	if order.PaymentStatus == "paid" {
		uc.log.Infof("Order already paid, skipping (idempotent)")
		return nil // 幂等
	}

	// 2. 更新订单状态
	order.PaymentStatus = "paid"
	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		uc.log.Errorf("Failed to update order: %v", err)
		return err
	}
	uc.log.Infof("Order updated to paid status")

	// 3. 获取套餐时长
	plan, err := uc.repo.GetPlan(ctx, order.PlanID)
	if err != nil {
		uc.log.Errorf("Failed to get plan: %v", err)
		return err
	}
	uc.log.Infof("Found plan: %s, duration: %d days", plan.Name, plan.DurationDays)

	// 4. 更新或创建用户订阅
	sub, err := uc.repo.GetSubscription(ctx, order.UserID)
	now := time.Now().UTC()

	if sub == nil {
		// 新订阅
		uc.log.Infof("Creating new subscription for user %d", order.UserID)
		sub = &UserSubscription{
			UserID:    order.UserID,
			PlanID:    order.PlanID,
			StartTime: now,
			EndTime:   now.AddDate(0, 0, plan.DurationDays),
			Status:    "active",
			CreatedAt: now,
			UpdatedAt: now,
		}
	} else {
		// 续费
		uc.log.Infof("Renewing subscription for user %d, current end time: %v", order.UserID, sub.EndTime)
		if sub.EndTime.Before(now) {
			sub.StartTime = now
			sub.EndTime = now.AddDate(0, 0, plan.DurationDays)
		} else {
			sub.EndTime = sub.EndTime.AddDate(0, 0, plan.DurationDays)
		}
		sub.PlanID = order.PlanID // 更新为最新购买的套餐
		sub.Status = "active"
		sub.UpdatedAt = now
	}

	if err := uc.repo.SaveSubscription(ctx, sub); err != nil {
		uc.log.Errorf("Failed to save subscription: %v", err)
		return err
	}
	uc.log.Infof("Subscription saved successfully, new end time: %v", sub.EndTime)

	// 记录历史
	action := "created"
	if sub.ID > 0 {
		action = "renewed"
	}
	history := &SubscriptionHistory{
		UserID:    order.UserID,
		PlanID:    plan.ID,
		PlanName:  plan.Name,
		StartTime: sub.StartTime,
		EndTime:   sub.EndTime,
		Status:    sub.Status,
		Action:    action,
		CreatedAt: now,
	}
	if err := uc.repo.AddSubscriptionHistory(ctx, history); err != nil {
		uc.log.Errorf("Failed to add subscription history: %v", err)
		// 不影响主流程，只记录日志
	}

	return nil
}

// CancelSubscription 取消订阅
func (uc *SubscriptionUsecase) CancelSubscription(ctx context.Context, userID uint64, reason string) error {
	uc.log.Infof("CancelSubscription: userID=%d, reason=%s", userID, reason)

	// 获取当前订阅
	sub, err := uc.repo.GetSubscription(ctx, userID)
	if err != nil {
		uc.log.Errorf("Failed to get subscription: %v", err)
		return err
	}
	if sub == nil {
		return fmt.Errorf("no active subscription found")
	}

	// 只能取消 active 或 paused 状态的订阅
	if sub.Status != "active" && sub.Status != "paused" {
		return fmt.Errorf("cannot cancel subscription with status: %s", sub.Status)
	}

	now := time.Now().UTC()
	sub.Status = "cancelled"
	sub.AutoRenew = false // 取消时关闭自动续费
	sub.UpdatedAt = now

	if err := uc.repo.SaveSubscription(ctx, sub); err != nil {
		uc.log.Errorf("Failed to save subscription: %v", err)
		return err
	}

	// 记录历史
	history := &SubscriptionHistory{
		UserID:    userID,
		PlanID:    sub.PlanID,
		StartTime: sub.StartTime,
		EndTime:   sub.EndTime,
		Status:    sub.Status,
		Action:    "cancelled",
		CreatedAt: now,
	}
	if err := uc.repo.AddSubscriptionHistory(ctx, history); err != nil {
		uc.log.Errorf("Failed to add subscription history: %v", err)
	}

	uc.log.Infof("Subscription cancelled successfully for user %d", userID)
	return nil
}

// PauseSubscription 暂停订阅
func (uc *SubscriptionUsecase) PauseSubscription(ctx context.Context, userID uint64, reason string) error {
	uc.log.Infof("PauseSubscription: userID=%d, reason=%s", userID, reason)

	// 获取当前订阅
	sub, err := uc.repo.GetSubscription(ctx, userID)
	if err != nil {
		uc.log.Errorf("Failed to get subscription: %v", err)
		return err
	}
	if sub == nil {
		return fmt.Errorf("no active subscription found")
	}

	// 只能暂停 active 状态的订阅
	if sub.Status != "active" {
		return fmt.Errorf("can only pause active subscription, current status: %s", sub.Status)
	}

	now := time.Now().UTC()
	sub.Status = "paused"
	sub.UpdatedAt = now

	if err := uc.repo.SaveSubscription(ctx, sub); err != nil {
		uc.log.Errorf("Failed to save subscription: %v", err)
		return err
	}

	// 记录历史
	history := &SubscriptionHistory{
		UserID:    userID,
		PlanID:    sub.PlanID,
		StartTime: sub.StartTime,
		EndTime:   sub.EndTime,
		Status:    sub.Status,
		Action:    "paused",
		CreatedAt: now,
	}
	if err := uc.repo.AddSubscriptionHistory(ctx, history); err != nil {
		uc.log.Errorf("Failed to add subscription history: %v", err)
	}

	uc.log.Infof("Subscription paused successfully for user %d", userID)
	return nil
}

// ResumeSubscription 恢复订阅
func (uc *SubscriptionUsecase) ResumeSubscription(ctx context.Context, userID uint64) error {
	uc.log.Infof("ResumeSubscription: userID=%d", userID)

	// 获取当前订阅
	sub, err := uc.repo.GetSubscription(ctx, userID)
	if err != nil {
		uc.log.Errorf("Failed to get subscription: %v", err)
		return err
	}
	if sub == nil {
		return fmt.Errorf("no subscription found")
	}

	// 只能恢复 paused 状态的订阅
	if sub.Status != "paused" {
		return fmt.Errorf("can only resume paused subscription, current status: %s", sub.Status)
	}

	now := time.Now().UTC()
	sub.Status = "active"
	sub.UpdatedAt = now

	if err := uc.repo.SaveSubscription(ctx, sub); err != nil {
		uc.log.Errorf("Failed to save subscription: %v", err)
		return err
	}

	// 记录历史
	history := &SubscriptionHistory{
		UserID:    userID,
		PlanID:    sub.PlanID,
		StartTime: sub.StartTime,
		EndTime:   sub.EndTime,
		Status:    sub.Status,
		Action:    "resumed",
		CreatedAt: now,
	}
	if err := uc.repo.AddSubscriptionHistory(ctx, history); err != nil {
		uc.log.Errorf("Failed to add subscription history: %v", err)
	}

	uc.log.Infof("Subscription resumed successfully for user %d", userID)
	return nil
}

// GetSubscriptionHistory 获取订阅历史记录
func (uc *SubscriptionUsecase) GetSubscriptionHistory(ctx context.Context, userID uint64, page, pageSize int) ([]*SubscriptionHistory, int, error) {
	uc.log.Infof("GetSubscriptionHistory: userID=%d, page=%d, pageSize=%d", userID, page, pageSize)

	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	items, total, err := uc.repo.GetSubscriptionHistory(ctx, userID, page, pageSize)
	if err != nil {
		uc.log.Errorf("Failed to get subscription history: %v", err)
		return nil, 0, err
	}

	uc.log.Infof("Retrieved %d history items for user %d", len(items), userID)
	return items, total, nil
}

// SetAutoRenew 设置自动续费
func (uc *SubscriptionUsecase) SetAutoRenew(ctx context.Context, userID uint64, autoRenew bool) error {
	uc.log.Infof("SetAutoRenew: userID=%d, autoRenew=%v", userID, autoRenew)

	// 获取当前订阅
	sub, err := uc.repo.GetSubscription(ctx, userID)
	if err != nil {
		uc.log.Errorf("Failed to get subscription: %v", err)
		return err
	}
	if sub == nil {
		return fmt.Errorf("no subscription found")
	}

	// 只有 active 状态的订阅才能设置自动续费
	if sub.Status != "active" {
		return fmt.Errorf("can only set auto-renew for active subscription, current status: %s", sub.Status)
	}

	now := time.Now().UTC()
	sub.AutoRenew = autoRenew
	sub.UpdatedAt = now

	if err := uc.repo.SaveSubscription(ctx, sub); err != nil {
		uc.log.Errorf("Failed to save subscription: %v", err)
		return err
	}

	action := "disabled_auto_renew"
	if autoRenew {
		action = "enabled_auto_renew"
	}
	uc.log.Infof("Auto-renew %s successfully for user %d", action, userID)
	return nil
}

// GetPlan 获取套餐信息
func (uc *SubscriptionUsecase) GetPlan(ctx context.Context, planID string) (*Plan, error) {
	return uc.repo.GetPlan(ctx, planID)
}

// GetExpiringSubscriptions 获取即将过期的订阅
func (uc *SubscriptionUsecase) GetExpiringSubscriptions(ctx context.Context, daysBeforeExpiry, page, pageSize int) ([]*UserSubscription, int, error) {
	uc.log.Infof("GetExpiringSubscriptions: daysBeforeExpiry=%d, page=%d, pageSize=%d", daysBeforeExpiry, page, pageSize)

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

		// 获取套餐名称
		plan, _ := uc.repo.GetPlan(ctx, sub.PlanID)
		planName := sub.PlanID
		if plan != nil {
			planName = plan.Name
		}

		// 添加历史记录
		history := &SubscriptionHistory{
			UserID:    uid,
			PlanID:    sub.PlanID,
			PlanName:  planName,
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
