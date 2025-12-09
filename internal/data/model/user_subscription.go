package model

import "time"

// UserSubscription 用户订阅模型
type UserSubscription struct {
	SubscriptionID uint64    `gorm:"primaryKey;column:subscription_id;autoIncrement"`
	UID            uint64    `gorm:"column:uid;uniqueIndex;not null"`
	PlanID         string    `gorm:"column:plan_id;not null"`                                   // 套餐ID
	AppID          string    `gorm:"column:app_id;not null;index:idx_app_id;index:idx_app_uid"` // 应用ID（冗余字段，便于按app统计和查询）
	StartTime      time.Time `gorm:"column:start_time;not null"`
	EndTime        time.Time `gorm:"column:end_time;not null"`
	Status         string    `gorm:"column:status;type:enum('active','expired','paused','cancelled');not null;default:'active'"` // 订阅状态: active-活跃(订阅有效中), expired-过期(订阅已过期), paused-暂停(用户主动暂停), cancelled-已取消(用户主动取消)
	OrderID        string    `gorm:"column:order_id;not null;index"`
	IsAutoRenew    bool      `gorm:"column:is_auto_renew;default:false"` // 是否自动续费
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (UserSubscription) TableName() string { return "user_subscription" }
