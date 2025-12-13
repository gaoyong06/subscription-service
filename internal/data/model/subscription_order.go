package model

import "time"

// SubscriptionOrder 订单模型
type SubscriptionOrder struct {
	OrderID       string    `gorm:"primaryKey;column:order_id"`
	PaymentID     string    `gorm:"column:payment_id;index"`                   // 支付流水号(payment-service返回的payment_id，用于追溯支付记录)
	UID           string    `gorm:"column:uid;type:varchar(36);index:idx_uid"` // 用户ID（字符串 UUID）
	PlanID        string    `gorm:"column:plan_id"`
	AppID         string    `gorm:"column:app_id;type:varchar(50);index"`
	Amount        float64   `gorm:"column:amount"`
	PaymentStatus string    `gorm:"column:payment_status;type:enum('pending','success','failed','closed','refunded','partially_refunded');not null;default:'pending'"` // 支付状态(与payment-service保持一致): pending-待支付(订单已创建，等待支付), success-支付成功, failed-支付失败, closed-订单关闭, refunded-已全额退款, partially_refunded-部分退款
	CreatedAt     time.Time `gorm:"column:created_at"`
}

func (SubscriptionOrder) TableName() string { return "subscription_order" }
