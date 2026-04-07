package biz

import (
	"context"
	"time"

	"subscription-service/internal/constants"
	"subscription-service/internal/errors"

	pkgErrors "github.com/gaoyong06/go-pkg/errors"
)

// EnsureDefaultFreeSubscription 若 user_subscription 尚无记录，则为用户写入默认免费档（FOREVER）并记 history created。
// 已存在任意订阅记录时直接返回幂等成功（不降级、不覆盖）。
func (uc *SubscriptionUsecase) EnsureDefaultFreeSubscription(ctx context.Context, userID, appID string) (created bool, already bool, err error) {
	if userID == "" || appID == "" {
		return false, false, pkgErrors.NewBizErrorWithLang(ctx, pkgErrors.ErrCodeInvalidArgument)
	}

	err = uc.withTransaction(ctx, func(ctx context.Context) error {
		existing, e := uc.subRepo.GetSubscription(ctx, userID)
		if e != nil {
			return e
		}
		if existing != nil {
			already = true
			return nil
		}

		freePlan, e := uc.planRepo.FindFreeForeverPlanByApp(ctx, appID)
		if e != nil {
			return e
		}
		if freePlan == nil {
			uc.log.Errorf("no free FOREVER plan for app_id=%s", appID)
			return pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodeDefaultFreePlanNotFound)
		}
		if NormalizePeriodType(freePlan.PeriodType) != constants.PeriodTypeForever {
			return pkgErrors.NewBizErrorWithLang(ctx, errors.ErrCodeDefaultFreePlanNotFound)
		}

		now := time.Now().UTC()
		endTime := ForeverEndUTC()
		sub := &UserSubscription{
			UserID:           userID,
			PlanID:           freePlan.PlanID,
			AppID:            appID,
			BillingAnchorDay: 0,
			StartTime:        now,
			EndTime:          endTime,
			Status:           constants.StatusActive,
			OrderID:          "",
			IsAutoRenew:      false,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if e := uc.subRepo.SaveSubscription(ctx, sub); e != nil {
			return e
		}

		history := &SubscriptionHistory{
			UserID:    userID,
			PlanID:    freePlan.PlanID,
			PlanName:  freePlan.Name,
			AppID:     appID,
			StartTime: sub.StartTime,
			EndTime:   sub.EndTime,
			Status:    sub.Status,
			Action:    constants.ActionCreated,
			CreatedAt: now,
		}
		if e := uc.historyRepo.AddSubscriptionHistory(ctx, history); e != nil {
			uc.log.Warnf("EnsureDefaultFreeSubscription: add history failed: %v", e)
		}
		created = true
		return nil
	})
	return created, already, err
}
