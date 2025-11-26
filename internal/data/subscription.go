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
	Status    string    `gorm:"column:status"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (UserSubscription) TableName() string { return "user_subscription" }

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
	if err := r.data.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
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
