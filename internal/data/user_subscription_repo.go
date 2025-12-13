package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"
	"xinyuan_tech/subscription-service/internal/biz"
	"xinyuan_tech/subscription-service/internal/constants"
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
func (r *subscriptionRepo) GetSubscription(ctx context.Context, uid string) (*biz.UserSubscription, error) {
	// 1. 尝试从 Redis 获取
	cacheKey := fmt.Sprintf("subscription:user:%s", uid)
	val, err := r.data.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		// 检查是否是空值缓存
		if val == "null" {
			return nil, nil
		}

		var sub biz.UserSubscription
		if err := json.Unmarshal([]byte(val), &sub); err == nil {
			return &sub, nil
		}
	}

	// 2. 从数据库获取
	var m model.UserSubscription
	err = r.data.db.WithContext(ctx).Where("uid = ?", uid).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 缓存空值,防止缓存穿透
		r.data.rdb.Set(ctx, cacheKey, "null", constants.NullCacheExpiration)
		return nil, nil
	}
	if err != nil {
		r.log.Errorf("Failed to get subscription for user %s: %v", uid, err)
		return nil, err
	}

	sub := &biz.UserSubscription{
		SubscriptionID: m.SubscriptionID,
		UID:            m.UID,
		PlanID:         m.PlanID,
		AppID:          m.AppID,
		StartTime:      m.StartTime,
		EndTime:        m.EndTime,
		Status:         m.Status,
		OrderID:        m.OrderID,
		IsAutoRenew:    m.IsAutoRenew,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}

	// 3. 写入 Redis 缓存 (1小时 + 随机时间,防止缓存雪崩)
	if data, err := json.Marshal(sub); err == nil {
		// 添加随机过期时间
		randomSeconds := time.Duration(rand.Intn(constants.CacheRandomMaxSeconds)) * time.Second
		expiration := constants.DefaultCacheExpiration + randomSeconds
		if err := r.data.rdb.Set(ctx, cacheKey, data, expiration).Err(); err != nil {
			r.log.Warnf("Failed to cache subscription for user %s: %v", uid, err)
		}
	}

	return sub, nil
}

// SaveSubscription 保存订阅
func (r *subscriptionRepo) SaveSubscription(ctx context.Context, sub *biz.UserSubscription) error {
	// app_id 应该由业务层从 Context 获取并传入，不再通过 plan 表查询获取
	appID := sub.AppID
	if appID == "" {
		r.log.Warnf("app_id is empty in UserSubscription, this should be set by business layer from Context")
	}

	m := &model.UserSubscription{
		SubscriptionID: sub.SubscriptionID,
		UID:            sub.UID,
		PlanID:         sub.PlanID,
		AppID:          appID,
		StartTime:      sub.StartTime,
		EndTime:        sub.EndTime,
		Status:         sub.Status,
		OrderID:        sub.OrderID,
		IsAutoRenew:    sub.IsAutoRenew,
		CreatedAt:      sub.CreatedAt,
		UpdatedAt:      sub.UpdatedAt,
	}
	if err := r.data.db.WithContext(ctx).Save(m).Error; err != nil {
		r.log.Errorf("Failed to save subscription for user %s: %v", sub.UID, err)
		return err
	}
	// 更新 biz 对象的 SubscriptionID（如果是新创建的）
	sub.SubscriptionID = m.SubscriptionID

	// 删除缓存
	cacheKey := fmt.Sprintf("subscription:user:%s", sub.UID)
	if err := r.data.rdb.Del(ctx, cacheKey).Err(); err != nil {
		r.log.Warnf("Failed to delete cache for user %s: %v", sub.UID, err)
		// 缓存删除失败不影响主流程,但需要记录
		// 缓存会在过期时间后自动失效
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
		Where("end_time BETWEEN ? AND ? AND status = ?", now, expiryDate, constants.StatusActive).
		Count(&total).Error; err != nil {
		r.log.Errorf("Failed to count expiring subscriptions: %v", err)
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := r.data.db.WithContext(ctx).
		Where("end_time BETWEEN ? AND ? AND status = ?", now, expiryDate, constants.StatusActive).
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
			SubscriptionID: m.SubscriptionID,
			UID:            m.UID,
			PlanID:         m.PlanID,
			StartTime:      m.StartTime,
			EndTime:        m.EndTime,
			Status:         m.Status,
			OrderID:        m.OrderID,
			IsAutoRenew:    m.IsAutoRenew,
			CreatedAt:      m.CreatedAt,
			UpdatedAt:      m.UpdatedAt,
		}
	}

	return subscriptions, int(total), nil
}

// UpdateExpiredSubscriptions 批量更新过期订阅状态
func (r *subscriptionRepo) UpdateExpiredSubscriptions(ctx context.Context) (int, []string, error) {
	now := time.Now().UTC()

	// 先查询需要更新的订阅
	var subscriptions []model.UserSubscription
	if err := r.data.db.WithContext(ctx).
		Where("end_time < ? AND status = ?", now, constants.StatusActive).
		Find(&subscriptions).Error; err != nil {
		r.log.Errorf("Failed to query expired subscriptions: %v", err)
		return 0, nil, err
	}

	if len(subscriptions) == 0 {
		return 0, []string{}, nil
	}

	// 提取 uid 列表
	uids := make([]string, len(subscriptions))
	for i, sub := range subscriptions {
		uids[i] = sub.UID
	}

	// 批量更新状态
	result := r.data.db.WithContext(ctx).Model(&model.UserSubscription{}).
		Where("end_time < ? AND status = ?", now, constants.StatusActive).
		Update("status", constants.StatusExpired)

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
		Where("end_time BETWEEN ? AND ? AND status = ? AND is_auto_renew = ?",
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
			SubscriptionID: m.SubscriptionID,
			UID:            m.UID,
			PlanID:         m.PlanID,
			StartTime:      m.StartTime,
			EndTime:        m.EndTime,
			Status:         m.Status,
			OrderID:        m.OrderID,
			IsAutoRenew:    m.IsAutoRenew,
			CreatedAt:      m.CreatedAt,
			UpdatedAt:      m.UpdatedAt,
		}
	}

	return subscriptions, nil
}
