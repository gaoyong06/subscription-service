package biz

import (
	"context"
	"fmt"
	"time"

	"xinyuan_tech/subscription-service/internal/errors"

	pkgErrors "github.com/gaoyong06/go-pkg/errors"
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
func (uc *SubscriptionUsecase) CreateSubscriptionOrder(ctx context.Context, userID uint64, planID, method, region string) (*SubscriptionOrder, string, string, string, string, error) {
	uc.log.Infof("CreateSubscriptionOrder: userID=%d, planID=%s, method=%s, region=%s", userID, planID, method, region)

	// 1. 获取套餐区域定价
	pricing, err := uc.GetPlanPricing(ctx, planID, region)
	if err != nil {
		uc.log.Errorf("Failed to get plan pricing: %v", err)
		return nil, "", "", "", "", pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodePlanNotFound)
	}
	if pricing == nil {
		uc.log.Errorf("Plan pricing not found: %s", planID)
		return nil, "", "", "", "", pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodePlanNotFound)
	}
	uc.log.Infof("Found plan pricing: region=%s, price=%.2f %s", pricing.Region, pricing.Price, pricing.Currency)

	// 2. 创建本地订单
	orderID := fmt.Sprintf("SUB%d%d", time.Now().UnixNano(), userID)
	order := &SubscriptionOrder{
		ID:            orderID,
		UserID:        userID,
		PlanID:        planID,
		Amount:        pricing.Price,
		PaymentStatus: "pending",
		CreatedAt:     time.Now().UTC(),
	}
	if err := uc.orderRepo.CreateOrder(ctx, order); err != nil {
		uc.log.Errorf("Failed to create order: %v", err)
		return nil, "", "", "", "", pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodeOrderCreateFailed)
	}
	uc.log.Infof("Created order: %s", orderID)

	// 3. 调用支付服务
	// 从配置中获取 ReturnURL
	returnURL := ""
	if uc.config != nil && uc.config.GetClient() != nil && uc.config.GetClient().GetSubscriptionService() != nil {
		returnURL = uc.config.GetClient().GetSubscriptionService().GetReturnUrl()
	}
	if returnURL == "" {
		uc.log.Errorf("ReturnURL is not configured")
		return nil, "", "", "", "", pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodeOrderCreateFailed)
	}

	// 获取套餐信息用于支付主题
	plan, _ := uc.planRepo.GetPlan(ctx, planID)
	subject := "Subscription"
	if plan != nil {
		subject = "Subscription: " + plan.Name
	}

	uc.log.Infof("Calling payment service: orderID=%s, amount=%.2f %s, method=%s", orderID, pricing.Price, pricing.Currency, method)
	paymentID, payUrl, payCode, payParams, err := uc.paymentClient.CreatePayment(ctx, orderID, userID, pricing.Price, pricing.Currency, method, subject, returnURL)
	if err != nil {
		uc.log.Errorf("Failed to create payment: %v", err)
		return nil, "", "", "", "", pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodePaymentFailed)
	}
	uc.log.Infof("Payment created: paymentID=%s", paymentID)

	return order, paymentID, payUrl, payCode, payParams, nil
}

// HandlePaymentSuccess 处理支付成功回调
func (uc *SubscriptionUsecase) HandlePaymentSuccess(ctx context.Context, orderID string, amount float64) error {
	uc.log.Infof("HandlePaymentSuccess: orderID=%s, amount=%.2f", orderID, amount)

	// 使用事务确保数据一致性
	return uc.withTransaction(ctx, func(ctx context.Context) error {
		// 1. 获取订单
		order, err := uc.orderRepo.GetOrder(ctx, orderID)
		if err != nil {
			uc.log.Errorf("Failed to get order: %v", err)
			return pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodeOrderNotFound)
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
			return pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodePlanNotFound)
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
	})
}

// withTransaction 执行事务
// withTransaction 执行事务
func (uc *SubscriptionUsecase) withTransaction(ctx context.Context, fn func(context.Context) error) error {
	return uc.tm.Exec(ctx, fn)
}
