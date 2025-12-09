package biz

import (
	"context"
	"time"
)

// SubscriptionHistory 订阅历史记录
type SubscriptionHistory struct {
	SubscriptionHistoryID uint64
	UID                   uint64
	PlanID                string
	PlanName              string
	AppID                 string // 应用ID（冗余字段，便于按app统计和查询）
	StartTime             time.Time
	EndTime               time.Time
	Status                string
	Action                string // created, renewed, upgraded, paused, resumed, cancelled
	CreatedAt             time.Time
}

// SubscriptionHistoryRepo 订阅历史记录仓库接口
type SubscriptionHistoryRepo interface {
	AddSubscriptionHistory(ctx context.Context, history *SubscriptionHistory) error
	GetSubscriptionHistory(ctx context.Context, userID uint64, page, pageSize int) ([]*SubscriptionHistory, int, error)
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

	items, total, err := uc.historyRepo.GetSubscriptionHistory(ctx, userID, page, pageSize)
	if err != nil {
		uc.log.Errorf("Failed to get subscription history: %v", err)
		return nil, 0, err
	}

	uc.log.Infof("Retrieved %d history items for user %d", len(items), userID)
	return items, total, nil
}
