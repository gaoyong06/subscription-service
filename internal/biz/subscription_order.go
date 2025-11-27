package biz

import (
	"context"
	"fmt"
	"time"
)

// SubscriptionOrder 简易订单记录 (用于记录订阅购买请求)
type SubscriptionOrder struct {
	ID            string
	UserID        uint64
	PlanID        string
	Amount        float64
	PaymentStatus string // pending, paid
	CreatedAt     time.Time
}

// SubscriptionOrderRepo 订阅订单仓库接口
type SubscriptionOrderRepo interface {
	CreateOrder(ctx context.Context, order *SubscriptionOrder) error
	GetOrder(ctx context.Context, orderID string) (*SubscriptionOrder, error)
	UpdateOrder(ctx context.Context, order *SubscriptionOrder) error
}

// CreateSubscriptionOrder 创建订阅订单
func (uc *SubscriptionUsecase) CreateSubscriptionOrder(ctx context.Context, userID uint64, planID, method string) (*SubscriptionOrder, string, string, string, string, error) {
	uc.log.Infof("CreateSubscriptionOrder: userID=%d, planID=%s, method=%s", userID, planID, method)

	// 1. 获取套餐信息
	plan, err := uc.planRepo.GetPlan(ctx, planID)
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
	order := &SubscriptionOrder{
		ID:            orderID,
		UserID:        userID,
		PlanID:        planID,
		Amount:        plan.Price,
		PaymentStatus: "pending",
		CreatedAt:     time.Now().UTC(),
	}
	if err := uc.orderRepo.CreateOrder(ctx, order); err != nil {
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

// HandlePaymentSuccess 处理支付成功回调
func (uc *SubscriptionUsecase) HandlePaymentSuccess(ctx context.Context, orderID string, amount float64) error {
	uc.log.Infof("HandlePaymentSuccess: orderID=%s, amount=%.2f", orderID, amount)

	// 1. 获取订单
	order, err := uc.orderRepo.GetOrder(ctx, orderID)
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
	if err := uc.orderRepo.UpdateOrder(ctx, order); err != nil {
		uc.log.Errorf("Failed to update order: %v", err)
		return err
	}
	uc.log.Infof("Order updated to paid status")

	// 3. 获取套餐时长
	plan, err := uc.planRepo.GetPlan(ctx, order.PlanID)
	if err != nil {
		uc.log.Errorf("Failed to get plan: %v", err)
		return err
	}
	uc.log.Infof("Found plan: %s, duration: %d days", plan.Name, plan.DurationDays)

	// 4. 更新或创建用户订阅
	sub, err := uc.subRepo.GetSubscription(ctx, order.UserID)
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

	if err := uc.subRepo.SaveSubscription(ctx, sub); err != nil {
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
	if err := uc.historyRepo.AddSubscriptionHistory(ctx, history); err != nil {
		uc.log.Errorf("Failed to add subscription history: %v", err)
		// 不影响主流程，只记录日志
	}

	return nil
}
