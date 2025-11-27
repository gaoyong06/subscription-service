package model

import "time"

// SubscriptionHistory 订阅历史模型
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
