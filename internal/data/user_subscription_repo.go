package data

import (
	"context"
	"errors"
	"time"
	"xinyuan_tech/subscription-service/internal/biz"
	"xinyuan_tech/subscription-service/internal/data/model"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

// subscriptionRepo 订阅仓库实现
type subscriptionRepo struct {
	data *Data
	log  *log.Helper
}

// NewUserSubscriptionRepo 创建用户订阅仓库
func NewUserSubscriptionRepo(data *Data, logger log.Logger) biz.UserSubscriptionRepo {
	return &subscriptionRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// GetSubscription 获取用户订阅
func (r *subscriptionRepo) GetSubscription(ctx context.Context, userID uint64) (*biz.UserSubscription, error) {
	var m model.UserSubscription
	err := r.data.db.WithContext(ctx).Where("user_id = ?", userID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		r.log.Errorf("Failed to get subscription for user %d: %v", userID, err)
		return nil, err
	}
	return &biz.UserSubscription{
		ID:        m.ID,
		UserID:    m.UserID,
		PlanID:    m.PlanID,
		StartTime: m.StartTime,
		EndTime:   m.EndTime,
		Status:    m.Status,
		AutoRenew: m.AutoRenew,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}, nil
}

// SaveSubscription 保存订阅
func (r *subscriptionRepo) SaveSubscription(ctx context.Context, sub *biz.UserSubscription) error {
	m := &model.UserSubscription{
		ID:        sub.ID,
		UserID:    sub.UserID,
		PlanID:    sub.PlanID,
		StartTime: sub.StartTime,
		EndTime:   sub.EndTime,
		Status:    sub.Status,
		AutoRenew: sub.AutoRenew,
		CreatedAt: sub.CreatedAt,
		UpdatedAt: sub.UpdatedAt,
	}
	if err := r.data.db.WithContext(ctx).Save(m).Error; err != nil {
		r.log.Errorf("Failed to save subscription for user %d: %v", sub.UserID, err)
		return err
	}
	return nil
}

// GetExpiringSubscriptions 获取即将过期的订阅
func (r *subscriptionRepo) GetExpiringSubscriptions(ctx context.Context, daysBeforeExpiry, page, pageSize int) ([]*biz.UserSubscription, int, error) {
	var models []model.UserSubscription
	var total int64

	now := time.Now().UTC()
	expiryDate := now.AddDate(0, 0, daysBeforeExpiry)

	// 获取总数
	if err := r.data.db.WithContext(ctx).Model(&model.UserSubscription{}).
		Where("end_time BETWEEN ? AND ? AND status = ?", now, expiryDate, "active").
		Count(&total).Error; err != nil {
		r.log.Errorf("Failed to count expiring subscriptions: %v", err)
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := r.data.db.WithContext(ctx).
		Where("end_time BETWEEN ? AND ? AND status = ?", now, expiryDate, "active").
		Order("end_time ASC").
		Limit(pageSize).
		Offset(offset).
		Find(&models).Error; err != nil {
		r.log.Errorf("Failed to get expiring subscriptions: %v", err)
		return nil, 0, err
	}

	// 转换为业务对象
	subscriptions := make([]*biz.UserSubscription, len(models))
	for i, m := range models {
		subscriptions[i] = &biz.UserSubscription{
			ID:        m.ID,
			UserID:    m.UserID,
			PlanID:    m.PlanID,
			StartTime: m.StartTime,
			EndTime:   m.EndTime,
			Status:    m.Status,
			AutoRenew: m.AutoRenew,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		}
	}

	return subscriptions, int(total), nil
}

// UpdateExpiredSubscriptions 批量更新过期订阅状态
func (r *subscriptionRepo) UpdateExpiredSubscriptions(ctx context.Context) (int, []uint64, error) {
	now := time.Now().UTC()

	// 先查询需要更新的订阅
	var subscriptions []model.UserSubscription
	if err := r.data.db.WithContext(ctx).
		Where("end_time < ? AND status = ?", now, "active").
		Find(&subscriptions).Error; err != nil {
		r.log.Errorf("Failed to query expired subscriptions: %v", err)
		return 0, nil, err
	}

	if len(subscriptions) == 0 {
		return 0, []uint64{}, nil
	}

	// 提取 user_id 列表
	uids := make([]uint64, len(subscriptions))
	for i, sub := range subscriptions {
		uids[i] = sub.UserID
	}

	// 批量更新状态
	result := r.data.db.WithContext(ctx).Model(&model.UserSubscription{}).
		Where("end_time < ? AND status = ?", now, "active").
		Update("status", "expired")

	if result.Error != nil {
		r.log.Errorf("Failed to update expired subscriptions: %v", result.Error)
		return 0, nil, result.Error
	}

	r.log.Infof("Updated %d expired subscriptions", result.RowsAffected)
	return int(result.RowsAffected), uids, nil
}

// GetAutoRenewSubscriptions 获取需要自动续费的订阅
func (r *subscriptionRepo) GetAutoRenewSubscriptions(ctx context.Context, daysBeforeExpiry int) ([]*biz.UserSubscription, error) {
	var models []model.UserSubscription

	now := time.Now().UTC()
	expiryDate := now.AddDate(0, 0, daysBeforeExpiry)

	// 查询即将过期且开启了自动续费的订阅
	if err := r.data.db.WithContext(ctx).
		Where("end_time BETWEEN ? AND ? AND status = ? AND auto_renew = ?",
			now, expiryDate, "active", true).
		Order("end_time ASC").
		Find(&models).Error; err != nil {
		r.log.Errorf("Failed to get auto-renew subscriptions: %v", err)
		return nil, err
	}

	// 转换为业务对象
	subscriptions := make([]*biz.UserSubscription, len(models))
	for i, m := range models {
		subscriptions[i] = &biz.UserSubscription{
			ID:        m.ID,
			UserID:    m.UserID,
			PlanID:    m.PlanID,
			StartTime: m.StartTime,
			EndTime:   m.EndTime,
			Status:    m.Status,
			AutoRenew: m.AutoRenew,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		}
	}

	return subscriptions, nil
}

