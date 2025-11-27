package model

import "time"

// UserSubscription 用户订阅模型 (对齐 schedule_manager 的 user_subscriptions 表)
type UserSubscription struct {
	SubscriptionID uint64    `gorm:"primaryKey;column:subscription_id;autoIncrement"`
	UID            uint64    `gorm:"column:uid;uniqueIndex;not null"`
	PlanID         string    `gorm:"column:subscription_type;not null"` // 对应 schedule_manager 的 subscription_type
	StartTime      time.Time `gorm:"column:start_time;not null"`
	EndTime        time.Time `gorm:"column:end_time;not null"`
	Status         string    `gorm:"column:status;not null;default:'active'"` // active, expired, paused, cancelled
	OrderID        string    `gorm:"column:order_id;not null;index"`
	AutoRenew      bool      `gorm:"column:auto_renew;default:false"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (UserSubscription) TableName() string { return "user_subscriptions" }
