package biz

import (
	"context"
	"time"

	"xinyuan_tech/subscription-service/internal/conf"
	"xinyuan_tech/subscription-service/internal/errors"

	pkgErrors "github.com/gaoyong06/go-pkg/errors"
	"github.com/go-kratos/kratos/v2/log"
)

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

// UserSubscriptionRepo 用户订阅仓库接口
type UserSubscriptionRepo interface {
	GetSubscription(ctx context.Context, userID uint64) (*UserSubscription, error)
	SaveSubscription(ctx context.Context, sub *UserSubscription) error
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
	planRepo      PlanRepo
	subRepo       UserSubscriptionRepo
	orderRepo     SubscriptionOrderRepo
	historyRepo   SubscriptionHistoryRepo
	paymentClient PaymentClient
	config        *conf.Bootstrap
	log           *log.Helper
}

// NewSubscriptionUsecase 创建订阅业务用例
func NewSubscriptionUsecase(
	planRepo PlanRepo,
	subRepo UserSubscriptionRepo,
	orderRepo SubscriptionOrderRepo,
	historyRepo SubscriptionHistoryRepo,
	paymentClient PaymentClient,
	config *conf.Bootstrap,
	logger log.Logger,
) *SubscriptionUsecase {
	return &SubscriptionUsecase{
		planRepo:      planRepo,
		subRepo:       subRepo,
		orderRepo:     orderRepo,
		historyRepo:   historyRepo,
		paymentClient: paymentClient,
		config:        config,
		log:           log.NewHelper(logger),
	}
}

// GetMySubscription 获取用户当前订阅信息
func (uc *SubscriptionUsecase) GetMySubscription(ctx context.Context, userID uint64) (*UserSubscription, error) {
	sub, err := uc.subRepo.GetSubscription(ctx, userID)
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

// CancelSubscription 取消订阅
func (uc *SubscriptionUsecase) CancelSubscription(ctx context.Context, userID uint64, reason string) error {
	uc.log.Infof("CancelSubscription: userID=%d, reason=%s", userID, reason)

	// 获取当前订阅
	sub, err := uc.subRepo.GetSubscription(ctx, userID)
	if err != nil {
		uc.log.Errorf("Failed to get subscription: %v", err)
		return err
	}
	if sub == nil {
		return pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodeSubscriptionNotFound)
	}

	// 只能取消 active 或 paused 状态的订阅
	if sub.Status != "active" && sub.Status != "paused" {
		return pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodeCannotCancelStatus)
	}

	now := time.Now().UTC()
	sub.Status = "cancelled"
	sub.AutoRenew = false // 取消时关闭自动续费
	sub.UpdatedAt = now

	if err := uc.subRepo.SaveSubscription(ctx, sub); err != nil {
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
	if err := uc.historyRepo.AddSubscriptionHistory(ctx, history); err != nil {
		uc.log.Errorf("Failed to add subscription history: %v", err)
	}

	uc.log.Infof("Subscription cancelled successfully for user %d", userID)
	return nil
}

// PauseSubscription 暂停订阅
func (uc *SubscriptionUsecase) PauseSubscription(ctx context.Context, userID uint64, reason string) error {
	uc.log.Infof("PauseSubscription: userID=%d, reason=%s", userID, reason)

	// 获取当前订阅
	sub, err := uc.subRepo.GetSubscription(ctx, userID)
	if err != nil {
		uc.log.Errorf("Failed to get subscription: %v", err)
		return err
	}
	if sub == nil {
		return pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodeSubscriptionNotFound)
	}

	// 只能暂停 active 状态的订阅
	if sub.Status != "active" {
		return pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodeCannotPauseStatus)
	}

	now := time.Now().UTC()
	sub.Status = "paused"
	sub.UpdatedAt = now

	if err := uc.subRepo.SaveSubscription(ctx, sub); err != nil {
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
	if err := uc.historyRepo.AddSubscriptionHistory(ctx, history); err != nil {
		uc.log.Errorf("Failed to add subscription history: %v", err)
	}

	uc.log.Infof("Subscription paused successfully for user %d", userID)
	return nil
}

// ResumeSubscription 恢复订阅
func (uc *SubscriptionUsecase) ResumeSubscription(ctx context.Context, userID uint64) error {
	uc.log.Infof("ResumeSubscription: userID=%d", userID)

	// 获取当前订阅
	sub, err := uc.subRepo.GetSubscription(ctx, userID)
	if err != nil {
		uc.log.Errorf("Failed to get subscription: %v", err)
		return err
	}
	if sub == nil {
		return pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodeSubscriptionNotFound)
	}

	// 只能恢复 paused 状态的订阅
	if sub.Status != "paused" {
		return pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodeCannotResumeStatus)
	}

	now := time.Now().UTC()
	sub.Status = "active"
	sub.UpdatedAt = now

	if err := uc.subRepo.SaveSubscription(ctx, sub); err != nil {
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
	if err := uc.historyRepo.AddSubscriptionHistory(ctx, history); err != nil {
		uc.log.Errorf("Failed to add subscription history: %v", err)
	}

	uc.log.Infof("Subscription resumed successfully for user %d", userID)
	return nil
}

// SetAutoRenew 设置自动续费
func (uc *SubscriptionUsecase) SetAutoRenew(ctx context.Context, userID uint64, autoRenew bool) error {
	uc.log.Infof("SetAutoRenew: userID=%d, autoRenew=%v", userID, autoRenew)

	// 获取当前订阅
	sub, err := uc.subRepo.GetSubscription(ctx, userID)
	if err != nil {
		uc.log.Errorf("Failed to get subscription: %v", err)
		return err
	}
	if sub == nil {
		return pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodeSubscriptionNotFound)
	}

	// 只有 active 状态的订阅才能设置自动续费
	if sub.Status != "active" {
		return pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodeCannotSetAutoRenew)
	}

	now := time.Now().UTC()
	sub.AutoRenew = autoRenew
	sub.UpdatedAt = now

	if err := uc.subRepo.SaveSubscription(ctx, sub); err != nil {
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
