package biz

import (
	"context"
	"fmt"
	"time"

	"subscription-service/internal/constants"
	"subscription-service/internal/errors"

	pkgErrors "github.com/gaoyong06/go-pkg/errors"
	"github.com/gaoyong06/go-pkg/middleware/app_id"
)

// SubscriptionOrder 简易订单记录 (用于记录订阅购买请求)
type SubscriptionOrder struct {
	OrderID       string
	PaymentID     string // 支付流水号(payment-service返回的payment_id，用于追溯支付记录)
	UserID        string // 用户ID（字符串 UUID）
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
	ListOrders(ctx context.Context, appID, userID, planID, status string, page, pageSize int) ([]*SubscriptionOrder, int, error)
}

// CreateSubscriptionOrder 创建订阅订单（保持向后兼容）
// region 参数为可选，如果为空则使用默认值
func (uc *SubscriptionUsecase) CreateSubscriptionOrder(ctx context.Context, userId string, planID, method, region string) (*SubscriptionOrder, string, string, string, string, error) {
	return uc.CreateSubscriptionOrderWithContext(ctx, userId, planID, method, region, "", "", "")
}

// CreateSubscriptionOrderWithContext 创建订阅订单（支持自动地区推断）
// region 参数为可选，如果为空则自动推断
// clientIP, acceptLanguage, xLanguage 用于地区推断
func (uc *SubscriptionUsecase) CreateSubscriptionOrderWithContext(ctx context.Context, userId string, planID, method, region, clientIP, acceptLanguage, xLanguage string) (*SubscriptionOrder, string, string, string, string, error) {
	uc.log.Infof("CreateSubscriptionOrder: userId=%s, planID=%s, method=%s, region=%s", userId, planID, method, region)

	// 如果 region 为空，自动推断
	if region == "" {
		if uc.regionDetectionSvc != nil {
			detectedRegion, err := uc.regionDetectionSvc.DetectRegion(ctx, userId, clientIP, acceptLanguage, xLanguage)
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
	orderID := fmt.Sprintf("SUB%d%s", time.Now().UnixNano(), userId[:8]) // 使用 userId 的前8位
	order := &SubscriptionOrder{
		OrderID:       orderID,
		PaymentID:     "", // 初始为空，调用支付服务后更新
		UserID:        userId,
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
	paymentID, payUrl, payCode, payParams, err := uc.paymentClient.CreatePayment(ctx, orderID, userId, pricing.Price, pricing.Currency, method, subject, returnURL)
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

		// 3. 获取套餐周期配置（UTC 自然历 / FOREVER）
		plan, err := uc.planRepo.GetPlan(ctx, order.PlanID)
		if err != nil {
			uc.log.Errorf("Failed to get plan: %v", err)
			return pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodePlanNotFound)
		}
		if err := ValidatePlanPeriod(plan.PeriodType, plan.IntervalCount); err != nil {
			uc.log.Errorf("Invalid plan billing config: %v", err)
			return pkgErrors.NewBizErrorWithLang(ctx, pkgErrors.ErrCodeInvalidArgument)
		}
		periodType := NormalizePeriodType(plan.PeriodType)
		uc.log.Infof("Found plan: %s, periodType=%s intervalCount=%d", plan.Name, periodType, plan.IntervalCount)

		// 4. 更新或创建用户订阅
		sub, err := uc.subRepo.GetSubscription(ctx, order.UserID)
		now := time.Now().UTC()
		prevPlanID := ""
		hadSubscription := false
		if sub != nil {
			hadSubscription = true
			prevPlanID = sub.PlanID
		}

		if sub == nil {
			endTime, err := FirstPeriodEndUTC(now, periodType, plan.IntervalCount)
			if err != nil {
				uc.log.Errorf("FirstPeriodEndUTC: %v", err)
				return pkgErrors.NewBizErrorWithLang(ctx, pkgErrors.ErrCodeInvalidArgument)
			}
			uc.log.Infof("Creating new subscription for user %s", order.UserID)
			sub = &UserSubscription{
				UserID:           order.UserID,
				PlanID:           order.PlanID,
				AppID:            order.AppID,
				BillingAnchorDay: BillingAnchorDayFromPurchase(now, periodType),
				StartTime:        now,
				EndTime:          endTime,
				Status:           constants.StatusActive,
				OrderID:          order.OrderID,
				CreatedAt:        now,
				UpdatedAt:        now,
			}
		} else {
			uc.log.Infof("Renewing subscription for user %s, current end time: %v", order.UserID, sub.EndTime)
			if sub.AppID == "" || sub.AppID != order.AppID {
				sub.AppID = order.AppID
			}
			anchor := sub.BillingAnchorDay
			if anchor == 0 && (periodType == constants.PeriodTypeMonth || periodType == constants.PeriodTypeYear) {
				anchor = BillingAnchorDayFromPurchase(sub.StartTime, periodType)
			}
			if sub.EndTime.Before(now) {
				sub.StartTime = now
				endTime, err := FirstPeriodEndUTC(now, periodType, plan.IntervalCount)
				if err != nil {
					uc.log.Errorf("FirstPeriodEndUTC: %v", err)
					return pkgErrors.NewBizErrorWithLang(ctx, pkgErrors.ErrCodeInvalidArgument)
				}
				sub.EndTime = endTime
				sub.BillingAnchorDay = BillingAnchorDayFromPurchase(now, periodType)
			} else {
				nextEnd, err := NextPeriodEndUTC(sub.EndTime, anchor, periodType, plan.IntervalCount)
				if err != nil {
					uc.log.Errorf("NextPeriodEndUTC: %v", err)
					return pkgErrors.NewBizErrorWithLang(ctx, pkgErrors.ErrCodeInvalidArgument)
				}
				sub.EndTime = nextEnd
			}
			sub.PlanID = order.PlanID
			sub.Status = constants.StatusActive
			sub.OrderID = order.OrderID
			sub.UpdatedAt = now
		}

		if err := uc.subRepo.SaveSubscription(ctx, sub); err != nil {
			uc.log.Errorf("Failed to save subscription: %v", err)
			return err
		}
		uc.log.Infof("Subscription saved successfully, new end time: %v", sub.EndTime)

		// 记录历史：新建 | 同档续费 | 换档升级
		action := constants.ActionCreated
		if hadSubscription {
			if prevPlanID != order.PlanID {
				action = constants.ActionUpgraded
			} else {
				action = constants.ActionRenewed
			}
		}
		history := &SubscriptionHistory{
			UserID:    order.UserID,
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
func (uc *SubscriptionUsecase) withTransaction(ctx context.Context, fn func(context.Context) error) error {
	return uc.tm.Exec(ctx, fn)
}

// ListSubscriptionOrders 获取订阅订单列表（管理员视角）
func (uc *SubscriptionUsecase) ListSubscriptionOrders(ctx context.Context, appID, userID, planID, status string, page, pageSize int) ([]*SubscriptionOrder, int, error) {
	uc.log.Infof("ListSubscriptionOrders: appID=%s, userID=%s, planID=%s, status=%s, page=%d, pageSize=%d", appID, userID, planID, status, page, pageSize)

	// 参数验证
	if appID == "" {
		return nil, 0, pkgErrors.NewBizErrorWithLang(ctx, pkgErrors.ErrCodeInvalidArgument)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	orders, total, err := uc.orderRepo.ListOrders(ctx, appID, userID, planID, status, page, pageSize)
	if err != nil {
		uc.log.Errorf("Failed to list subscription orders: %v", err)
		return nil, 0, err
	}

	uc.log.Infof("Retrieved %d subscription orders", len(orders))
	return orders, total, nil
}

// GetSubscriptionOrder 获取单个订阅订单
func (uc *SubscriptionUsecase) GetSubscriptionOrder(ctx context.Context, orderID string) (*SubscriptionOrder, error) {
	uc.log.Infof("GetSubscriptionOrder: orderID=%s", orderID)

	// 参数验证
	if orderID == "" {
		return nil, pkgErrors.NewBizErrorWithLang(ctx, pkgErrors.ErrCodeInvalidArgument)
	}

	order, err := uc.orderRepo.GetOrder(ctx, orderID)
	if err != nil {
		uc.log.Errorf("Failed to get subscription order: %v", err)
		return nil, err
	}

	return order, nil
}

// ListAppSubscriptions 获取应用的订阅用户列表（管理员视角）
func (uc *SubscriptionUsecase) ListAppSubscriptions(ctx context.Context, appID, status, userID string, page, pageSize int) ([]*UserSubscription, int, error) {
	uc.log.Infof("ListAppSubscriptions: appID=%s, status=%s, userID=%s, page=%d, pageSize=%d", appID, status, userID, page, pageSize)

	// 参数验证
	if appID == "" {
		return nil, 0, pkgErrors.NewBizErrorWithLang(ctx, pkgErrors.ErrCodeInvalidArgument)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	subscriptions, total, err := uc.subRepo.ListAppSubscriptions(ctx, appID, status, userID, page, pageSize)
	if err != nil {
		uc.log.Errorf("Failed to list app subscriptions: %v", err)
		return nil, 0, err
	}

	uc.log.Infof("Retrieved %d app subscriptions", len(subscriptions))
	return subscriptions, total, nil
}

// GetAppSubscriptionHistory 获取应用的订阅历史记录（管理员视角）
func (uc *SubscriptionUsecase) GetAppSubscriptionHistory(ctx context.Context, appID, userID, action string, startTime, endTime *time.Time, page, pageSize int) ([]*SubscriptionHistory, int, error) {
	uc.log.Infof("GetAppSubscriptionHistory: appID=%s, userID=%s, action=%s, page=%d, pageSize=%d", appID, userID, action, page, pageSize)

	// 参数验证
	if appID == "" {
		return nil, 0, pkgErrors.NewBizErrorWithLang(ctx, pkgErrors.ErrCodeInvalidArgument)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	items, total, err := uc.historyRepo.GetAppSubscriptionHistory(ctx, appID, userID, action, startTime, endTime, page, pageSize)
	if err != nil {
		uc.log.Errorf("Failed to get app subscription history: %v", err)
		return nil, 0, err
	}

	uc.log.Infof("Retrieved %d app subscription history items", len(items))
	return items, total, nil
}
