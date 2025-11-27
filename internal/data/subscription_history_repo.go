package data

import (
	"context"
	"xinyuan_tech/subscription-service/internal/biz"
	"xinyuan_tech/subscription-service/internal/data/model"

	"github.com/go-kratos/kratos/v2/log"
)

// historyRepo 历史记录仓库实现
type historyRepo struct {
	data *Data
	log  *log.Helper
}

// NewSubscriptionHistoryRepo 创建订阅历史记录仓库
func NewSubscriptionHistoryRepo(data *Data, logger log.Logger) biz.SubscriptionHistoryRepo {
	return &historyRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// AddSubscriptionHistory 添加订阅历史记录
func (r *historyRepo) AddSubscriptionHistory(ctx context.Context, history *biz.SubscriptionHistory) error {
	m := &model.SubscriptionHistory{
		UserID:    history.UserID,
		PlanID:    history.PlanID,
		PlanName:  history.PlanName,
		StartTime: history.StartTime,
		EndTime:   history.EndTime,
		Status:    history.Status,
		Action:    history.Action,
		CreatedAt: history.CreatedAt,
	}
	if err := r.data.db.WithContext(ctx).Create(m).Error; err != nil {
		r.log.Errorf("Failed to add subscription history for user %d: %v", history.UserID, err)
		return err
	}
	return nil
}

// GetSubscriptionHistory 获取用户订阅历史
func (r *historyRepo) GetSubscriptionHistory(ctx context.Context, userID uint64, page, pageSize int) ([]*biz.SubscriptionHistory, int, error) {
	var models []model.SubscriptionHistory
	var total int64

	// 获取总数
	if err := r.data.db.WithContext(ctx).Model(&model.SubscriptionHistory{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		r.log.Errorf("Failed to count subscription history for user %d: %v", userID, err)
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := r.data.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&models).Error; err != nil {
		r.log.Errorf("Failed to get subscription history for user %d: %v", userID, err)
		return nil, 0, err
	}

	// 转换为业务对象
	items := make([]*biz.SubscriptionHistory, len(models))
	for i, m := range models {
		items[i] = &biz.SubscriptionHistory{
			ID:        m.ID,
			UserID:    m.UserID,
			PlanID:    m.PlanID,
			PlanName:  m.PlanName,
			StartTime: m.StartTime,
			EndTime:   m.EndTime,
			Status:    m.Status,
			Action:    m.Action,
			CreatedAt: m.CreatedAt,
		}
	}

	return items, int(total), nil
}
