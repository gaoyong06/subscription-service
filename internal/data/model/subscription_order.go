package model

import "time"

// SubscriptionOrder 订单模型
type SubscriptionOrder struct {
	ID            string    `gorm:"primaryKey;column:order_id"`
	UserID        uint64    `gorm:"column:user_id"`
	PlanID        string    `gorm:"column:plan_id"`
	Amount        float64   `gorm:"column:amount"`
	PaymentStatus string    `gorm:"column:payment_status"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

func (SubscriptionOrder) TableName() string { return "subscription_order" }
