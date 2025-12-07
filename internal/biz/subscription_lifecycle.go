package biz

import (
	"context"
	"fmt"
	"time"
	"xinyuan_tech/subscription-service/internal/constants"

	"github.com/go-redsync/redsync/v4"
)

// AutoRenewResult 自动续费结果
type AutoRenewResult struct {
	UID          uint64
	PlanID       string
	Success      bool
	OrderID      string
	PaymentID    string
	ErrorMessage string
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
	subscriptions, total, err := uc.subRepo.GetExpiringSubscriptions(ctx, daysBeforeExpiry, page, pageSize)
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
	count, uids, err := uc.subRepo.UpdateExpiredSubscriptions(ctx)
	if err != nil {
		uc.log.Errorf("Failed to update expired subscriptions: %v", err)
		return 0, nil, err
	}

	// 为每个过期的订阅添加历史记录
	now := time.Now().UTC()
	for _, uid := range uids {
		// 获取订阅信息
		sub, err := uc.subRepo.GetSubscription(ctx, uid)
		if err != nil {
			uc.log.Errorf("Failed to get subscription for user %d: %v", uid, err)
			continue
		}
		if sub == nil {
			continue
		}

		// 获取套餐名称
		plan, _ := uc.planRepo.GetPlan(ctx, sub.PlanID)
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
		if err := uc.historyRepo.AddSubscriptionHistory(ctx, history); err != nil {
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
	subscriptions, err := uc.subRepo.GetAutoRenewSubscriptions(ctx, daysBeforeExpiry)
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

		// 使用分布式锁防止重复续费
		lockKey := fmt.Sprintf("auto_renew_lock:user:%d", sub.UserID)
		mutex := uc.rs.NewMutex(
			lockKey,
			redsync.WithExpiry(constants.AutoRenewLockExpiration),
			redsync.WithTries(constants.AutoRenewLockRetries), // 只尝试一次,如果失败说明正在处理
		)

		// 尝试获取锁
		if err := mutex.LockContext(ctx); err != nil {
			result.Success = false
			result.ErrorMessage = "failed to acquire lock or already processing"
			uc.log.Infof("Skipping auto-renew for user %d: lock busy or already processing", sub.UserID)
			results = append(results, result)
			continue
		}

		// 确保释放锁
		defer func(m *redsync.Mutex) {
			if _, err := m.UnlockContext(ctx); err != nil {
				uc.log.Warnf("Failed to unlock for user %d: %v", sub.UserID, err)
			}
		}(mutex)

		// 再次检查订阅状态,防止重复处理
		currentSub, err := uc.subRepo.GetSubscription(ctx, sub.UserID)
		if err != nil {
			result.Success = false
			result.ErrorMessage = "failed to get current subscription: " + err.Error()
			failedCount++
			results = append(results, result)
			continue
		}
		if currentSub != nil && currentSub.EndTime.After(sub.EndTime) {
			// 已经被续费过了
			result.Success = true
			result.ErrorMessage = "already renewed"
			uc.log.Infof("Subscription for user %d already renewed", sub.UserID)
			results = append(results, result)
			continue
		}

		if dryRun {
			// 测试模式，只记录不执行
			result.Success = true
			result.ErrorMessage = "dry run - not executed"
			uc.log.Infof("[DRY RUN] Would renew subscription for user %d, plan %s", sub.UserID, sub.PlanID)
		} else {
			// 实际执行续费（使用默认区域定价）
			order, paymentID, _, _, _, err := uc.CreateSubscriptionOrder(ctx, sub.UserID, sub.PlanID, "auto", "default", "")
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

				// TODO: 实际生产环境中，这里应该调用支付服务的自动扣款接口
				// 如果是自动续费，直接处理支付成功（模拟自动扣款）
				// 实际生产环境中，这里应该调用支付服务的自动扣款接口
				// 这里简化处理，假设自动扣款成功
				if err := uc.HandlePaymentSuccess(ctx, order.ID, paymentID, order.Amount); err != nil {
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
