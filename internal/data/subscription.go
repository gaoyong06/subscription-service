package data

import (
	"context"
	"errors"
	"time"
	"xinyuan_tech/subscription-service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

// Gorm Models
type Plan struct {
	ID           string  `gorm:"primaryKey;column:plan_id"`
	Name         string  `gorm:"column:name"`
	Description  string  `gorm:"column:description"`
	Price        float64 `gorm:"column:price"`
	Currency     string  `gorm:"column:currency"`
	DurationDays int     `gorm:"column:duration_days"`
	Type         string  `gorm:"column:type"`
}

func (Plan) TableName() string { return "plan" }

type UserSubscription struct {
	ID        uint64    `gorm:"primaryKey;column:user_subscription_id"`
	UserID    uint64    `gorm:"column:user_id;uniqueIndex"`
	PlanID    string    `gorm:"column:plan_id"`
	StartTime time.Time `gorm:"column:start_time"`
	EndTime   time.Time `gorm:"column:end_time"`
	Status    string    `gorm:"column:status"` // active, expired, paused, cancelled
	AutoRenew bool      `gorm:"column:auto_renew;default:false"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (UserSubscription) TableName() string { return "user_subscription" }

type SubscriptionHistory struct {
	ID        uint64    `gorm:"primaryKey;column:subscription_history_id;autoIncrement"`
	UserID    uint64    `gorm:"column:user_id;index"`
	PlanID    string    `gorm:"column:plan_id"`
	PlanName  string    `gorm:"column:plan_name"`
	StartTime time.Time `gorm:"column:start_time"`
	EndTime   time.Time `gorm:"column:end_time"`
	Status    string    `gorm:"column:status"`
	Action    string    `gorm:"column:action"` // created, renewed, upgraded, paused, resumed, cancelled
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (SubscriptionHistory) TableName() string { return "subscription_history" }

type Order struct {
	ID            string    `gorm:"primaryKey;column:subscription_order_id"`
	UserID        uint64    `gorm:"column:user_id"`
	PlanID        string    `gorm:"column:plan_id"`
	Amount        float64   `gorm:"column:amount"`
	PaymentStatus string    `gorm:"column:payment_status"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

func (Order) TableName() string { return "subscription_order" }

// Repo Implementation
type subscriptionRepo struct {
	data *Data
	log  *log.Helper
}

func NewSubscriptionRepo(data *Data, logger log.Logger) biz.SubscriptionRepo {
	return &subscriptionRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *subscriptionRepo) ListPlans(ctx context.Context) ([]*biz.Plan, error) {
	var models []Plan
	if err := r.data.db.WithContext(ctx).Find(&models).Error; err != nil {
		return nil, err
	}

	plans := make([]*biz.Plan, len(models))
	for i, m := range models {
		plans[i] = &biz.Plan{
			ID:           m.ID,
			Name:         m.Name,
			Description:  m.Description,
			Price:        m.Price,
			Currency:     m.Currency,
			DurationDays: m.DurationDays,
			Type:         m.Type,
		}
	}
	return plans, nil
}

func (r *subscriptionRepo) GetPlan(ctx context.Context, id string) (*biz.Plan, error) {
	var m Plan
	if err := r.data.db.WithContext(ctx).First(&m, "plan_id = ?", id).Error; err != nil {
		return nil, err
	}
	return &biz.Plan{
		ID:           m.ID,
		Name:         m.Name,
		Description:  m.Description,
		Price:        m.Price,
		Currency:     m.Currency,
		DurationDays: m.DurationDays,
		Type:         m.Type,
	}, nil
}

func (r *subscriptionRepo) GetSubscription(ctx context.Context, userID uint64) (*biz.UserSubscription, error) {
	var m UserSubscription
	err := r.data.db.WithContext(ctx).Where("user_id = ?", userID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
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

func (r *subscriptionRepo) SaveSubscription(ctx context.Context, sub *biz.UserSubscription) error {
	model := &UserSubscription{
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
	return r.data.db.WithContext(ctx).Save(model).Error
}

func (r *subscriptionRepo) CreateOrder(ctx context.Context, order *biz.Order) error {
	model := &Order{
		ID:            order.ID,
		UserID:        order.UserID,
		PlanID:        order.PlanID,
		Amount:        order.Amount,
		PaymentStatus: order.PaymentStatus,
		CreatedAt:     order.CreatedAt,
	}
	return r.data.db.WithContext(ctx).Create(model).Error
}

func (r *subscriptionRepo) GetOrder(ctx context.Context, orderID string) (*biz.Order, error) {
	var m Order
	if err := r.data.db.WithContext(ctx).First(&m, "id = ?", orderID).Error; err != nil {
		return nil, err
	}
	return &biz.Order{
		ID:            m.ID,
		UserID:        m.UserID,
		PlanID:        m.PlanID,
		Amount:        m.Amount,
		PaymentStatus: m.PaymentStatus,
		CreatedAt:     m.CreatedAt,
	}, nil
}

func (r *subscriptionRepo) UpdateOrder(ctx context.Context, order *biz.Order) error {
	model := &Order{
		ID:            order.ID,
		UserID:        order.UserID,
		PlanID:        order.PlanID,
		Amount:        order.Amount,
		PaymentStatus: order.PaymentStatus,
		CreatedAt:     order.CreatedAt,
	}
	return r.data.db.WithContext(ctx).Save(model).Error
}

func (r *subscriptionRepo) AddSubscriptionHistory(ctx context.Context, history *biz.SubscriptionHistory) error {
	model := &SubscriptionHistory{
		UserID:    history.UserID,
		PlanID:    history.PlanID,
		PlanName:  history.PlanName,
		StartTime: history.StartTime,
		EndTime:   history.EndTime,
		Status:    history.Status,
		Action:    history.Action,
		CreatedAt: history.CreatedAt,
	}
	return r.data.db.WithContext(ctx).Create(model).Error
}

func (r *subscriptionRepo) GetSubscriptionHistory(ctx context.Context, userID uint64, page, pageSize int) ([]*biz.SubscriptionHistory, int, error) {
	var models []SubscriptionHistory
	var total int64

	// 获取总数
	if err := r.data.db.WithContext(ctx).Model(&SubscriptionHistory{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
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

// GetExpiringSubscriptions 获取即将过期的订阅
func (r *subscriptionRepo) GetExpiringSubscriptions(ctx context.Context, daysBeforeExpiry, page, pageSize int) ([]*biz.UserSubscription, int, error) {
	var models []UserSubscription
	var total int64

	now := time.Now().UTC()
	expiryDate := now.AddDate(0, 0, daysBeforeExpiry)

	// 获取总数
	if err := r.data.db.WithContext(ctx).Model(&UserSubscription{}).
		Where("end_time BETWEEN ? AND ? AND status = ?", now, expiryDate, "active").
		Count(&total).Error; err != nil {
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
	var subscriptions []UserSubscription
	if err := r.data.db.WithContext(ctx).
		Where("end_time < ? AND status = ?", now, "active").
		Find(&subscriptions).Error; err != nil {
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
	result := r.data.db.WithContext(ctx).Model(&UserSubscription{}).
		Where("end_time < ? AND status = ?", now, "active").
		Update("status", "expired")

	if result.Error != nil {
		return 0, nil, result.Error
	}

	return int(result.RowsAffected), uids, nil
}

// GetAutoRenewSubscriptions 获取需要自动续费的订阅
func (r *subscriptionRepo) GetAutoRenewSubscriptions(ctx context.Context, daysBeforeExpiry int) ([]*biz.UserSubscription, error) {
	var models []UserSubscription

	now := time.Now().UTC()
	expiryDate := now.AddDate(0, 0, daysBeforeExpiry)

	// 查询即将过期且开启了自动续费的订阅
	if err := r.data.db.WithContext(ctx).
		Where("end_time BETWEEN ? AND ? AND status = ? AND auto_renew = ?",
			now, expiryDate, "active", true).
		Order("end_time ASC").
		Find(&models).Error; err != nil {
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
