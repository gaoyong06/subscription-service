package model

import "time"

// UserSubscription 用户订阅模型
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

