package biz

import (
	"context"
	"fmt"
	"time"

	"xinyuan_tech/subscription-service/internal/constants"
	"xinyuan_tech/subscription-service/internal/errors"

	pkgErrors "github.com/gaoyong06/go-pkg/errors"
	"github.com/gaoyong06/go-pkg/middleware/app_id"
)

// SubscriptionOrder 简易订单记录 (用于记录订阅购买请求)
type SubscriptionOrder struct {
	OrderID       string
	PaymentID     string // 支付流水号(payment-service返回的payment_id，用于追溯支付记录)
	UID           uint64
	PlanID        string
	AppID         string // 应用ID
	Amount        float64
	PaymentStatus string // pending, success, failed, closed, refunded, partially_refunded (与payment-service保持一致)
	CreatedAt     time.Time
}

// SubscriptionOrderRepo 订阅订单仓库接口
type SubscriptionOrderRepo interface {
	CreateOrder(ctx context.Context, order *SubscriptionOrder) error
	GetOrder(ctx context.Context, orderID string) (*SubscriptionOrder, error)
	UpdateOrder(ctx context.Context, order *SubscriptionOrder) error
}

// CreateSubscriptionOrder 创建订阅订单（保持向后兼容）
// region 参数为可选，如果为空则使用默认值
func (uc *SubscriptionUsecase) CreateSubscriptionOrder(ctx context.Context, userID uint64, planID, method, region string) (*SubscriptionOrder, string, string, string, string, error) {
	return uc.CreateSubscriptionOrderWithContext(ctx, userID, planID, method, region, "", "", "")
}

// CreateSubscriptionOrderWithContext 创建订阅订单（支持自动地区推断）
// region 参数为可选，如果为空则自动推断
// clientIP, acceptLanguage, xLanguage 用于地区推断
func (uc *SubscriptionUsecase) CreateSubscriptionOrderWithContext(ctx context.Context, userID uint64, planID, method, region, clientIP, acceptLanguage, xLanguage string) (*SubscriptionOrder, string, string, string, string, error) {
	uc.log.Infof("CreateSubscriptionOrder: userID=%d, planID=%s, method=%s, region=%s", userID, planID, method, region)

	// 如果 region 为空，自动推断
	if region == "" {
		if uc.regionDetectionSvc != nil {
			detectedRegion, err := uc.regionDetectionSvc.DetectRegion(ctx, userID, clientIP, acceptLanguage, xLanguage)
			if err != nil {
				uc.log.Warnf("Failed to detect region, using default: %v", err)
				region = "default"
			} else {
				region = detectedRegion
				uc.log.Infof("Auto-detected region: %s", region)
			}
		} else {
			// 如果没有配置地区推断服务，使用默认值
			region = "default"
			uc.log.Infof("Region detection service not configured, using default region")
		}
	} else {
		// 如果提供了 region，验证是否支持
		if !constants.SupportedRegions[region] {
			uc.log.Warnf("Unsupported region: %s, using default", region)
			region = "default"
		}
	}

	// 1. 获取套餐区域定价（从数据库查询，所有价格都在数据库中配置）
	// region 是国家代码（ISO 3166-1 alpha-2），如 CN, US, DE 等
	pricing, err := uc.GetPlanPricing(ctx, planID, region)
	if err != nil {
		uc.log.Errorf("Failed to get plan pricing: %v", err)
		return nil, "", "", "", "", pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodePlanNotFound)
	}
	if pricing == nil {
		uc.log.Errorf("Plan pricing not found: %s", planID)
		return nil, "", "", "", "", pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodePlanNotFound)
	}
	uc.log.Infof("Found plan pricing: countryCode=%s, price=%.2f %s", pricing.CountryCode, pricing.Price, pricing.Currency)

	// 2. 获取 app_id（优先从 Context，由中间件从 Header 提取）
	appID := app_id.GetAppIDFromContext(ctx)
	if appID == "" {
		uc.log.Errorf("app_id is required, please provide X-App-Id header")
		return nil, "", "", "", "", pkgErrors.NewBizErrorWithLang(ctx, pkgErrors.ErrCodeInvalidArgument)
	}

	// 3. 获取套餐信息（用于获取名称等信息，并验证 app_id 是否匹配）
	plan, err := uc.planRepo.GetPlan(ctx, planID)
	if err != nil {
		uc.log.Errorf("Failed to get plan: %v", err)
		return nil, "", "", "", "", pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodePlanNotFound)
	}
	if plan == nil {
		uc.log.Errorf("Plan not found: %s", planID)
		return nil, "", "", "", "", pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodePlanNotFound)
	}

	// 4. 验证 app_id 是否与 plan 的 app_id 匹配（数据一致性校验）
	if plan.AppID != "" && plan.AppID != appID {
		uc.log.Errorf("app_id mismatch: plan %s belongs to app %s, but request app_id is %s", planID, plan.AppID, appID)
		return nil, "", "", "", "", pkgErrors.NewBizErrorWithLang(ctx, pkgErrors.ErrCodeInvalidArgument)
	}

	// 5. 创建本地订单
	orderID := fmt.Sprintf("SUB%d%d", time.Now().UnixNano(), userID)
	order := &SubscriptionOrder{
		OrderID:       orderID,
		PaymentID:     "", // 初始为空，调用支付服务后更新
		UID:           userID,
		PlanID:        planID,
		AppID:         appID, // 使用从 Context 获取的 app_id
		Amount:        pricing.Price,
		PaymentStatus: constants.PaymentStatusPending,
		CreatedAt:     time.Now().UTC(),
	}
	if err := uc.orderRepo.CreateOrder(ctx, order); err != nil {
		uc.log.Errorf("Failed to create order: %v", err)
		return nil, "", "", "", "", pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodeOrderCreateFailed)
	}
	uc.log.Infof("Created order: %s", orderID)

	// 6. 调用支付服务
	// 从配置中获取 ReturnURL
	returnURL := ""
	if uc.config != nil && uc.config.GetSubscription() != nil {
		returnURL = uc.config.GetSubscription().GetReturnUrl()
	}
	if returnURL == "" {
		uc.log.Errorf("ReturnURL is not configured")
		return nil, "", "", "", "", pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodeOrderCreateFailed)
	}

	subject := "Subscription"
	if plan.Name != "" {
		subject = "Subscription: " + plan.Name
	}

	uc.log.Infof("Calling payment service: orderID=%s, appID=%s, amount=%.2f %s, method=%s", orderID, appID, pricing.Price, pricing.Currency, method)
	// 注意：appId 现在只从 Context 获取（由中间件从 Header/metadata 提取），不再作为参数传递
	paymentID, payUrl, payCode, payParams, err := uc.paymentClient.CreatePayment(ctx, orderID, userID, pricing.Price, pricing.Currency, method, subject, returnURL)
	if err != nil {
		uc.log.Errorf("Failed to create payment: %v", err)
		return nil, "", "", "", "", pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodePaymentFailed)
	}
	uc.log.Infof("Payment created: paymentID=%s", paymentID)

	// 7. 更新订单，保存 payment_id
	order.PaymentID = paymentID
	if err := uc.orderRepo.UpdateOrder(ctx, order); err != nil {
		uc.log.Errorf("Failed to update order with payment_id: %v", err)
		// 不影响主流程，只记录日志
	}

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
		if order.PaymentStatus == constants.PaymentStatusSuccess {
			uc.log.Infof("Order already paid, skipping (idempotent)")
			return nil // 幂等
		}

		// 2. 更新订单状态
		order.PaymentStatus = constants.PaymentStatusSuccess
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
		sub, err := uc.subRepo.GetSubscription(ctx, order.UID)
		now := time.Now().UTC()

		if sub == nil {
			// 新订阅
			uc.log.Infof("Creating new subscription for user %d", order.UID)
			sub = &UserSubscription{
				UserID:    order.UID,
				PlanID:    order.PlanID,
				AppID:     order.AppID, // 从订单中获取 app_id
				StartTime: now,
				EndTime:   now.AddDate(0, 0, plan.DurationDays),
				Status:    constants.StatusActive,
				OrderID:   order.OrderID,
				CreatedAt: now,
				UpdatedAt: now,
			}
		} else {
			// 续费
			uc.log.Infof("Renewing subscription for user %d, current end time: %v", order.UID, sub.EndTime)
			// 更新 app_id（如果为空或需要更新）
			if sub.AppID == "" || sub.AppID != order.AppID {
				sub.AppID = order.AppID
			}
			if sub.EndTime.Before(now) {
				sub.StartTime = now
				sub.EndTime = now.AddDate(0, 0, plan.DurationDays)
			} else {
				sub.EndTime = sub.EndTime.AddDate(0, 0, plan.DurationDays)
			}
			sub.PlanID = order.PlanID // 更新为最新购买的套餐
			sub.Status = constants.StatusActive
			sub.OrderID = order.OrderID // 更新为最新订单ID
			sub.UpdatedAt = now
		}

		if err := uc.subRepo.SaveSubscription(ctx, sub); err != nil {
			uc.log.Errorf("Failed to save subscription: %v", err)
			return err
		}
		uc.log.Infof("Subscription saved successfully, new end time: %v", sub.EndTime)

		// 记录历史
		action := constants.ActionCreated
		if sub.SubscriptionID > 0 {
			action = constants.ActionRenewed
		}
		history := &SubscriptionHistory{
			UID:       order.UID,
			PlanID:    plan.PlanID,
			PlanName:  plan.Name,
			AppID:     plan.AppID,
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
