package model

import "time"

// SubscriptionHistory 订阅历史模型
type SubscriptionHistory struct {
	SubscriptionHistoryID uint64    `gorm:"primaryKey;column:subscription_history_id;autoIncrement"`
	UID                   string    `gorm:"column:uid;type:varchar(36);index"` // 用户ID（字符串 UUID）
	PlanID                string    `gorm:"column:plan_id"`
	PlanName              string    `gorm:"column:plan_name"`
	AppID                 string    `gorm:"column:app_id;not null;index:idx_app_id;index:idx_app_uid"` // 应用ID（冗余字段，便于按app统计和查询）
	StartTime             time.Time `gorm:"column:start_time"`
	EndTime               time.Time `gorm:"column:end_time"`
	Status                string    `gorm:"column:status"`
	Action                string    `gorm:"column:action;type:enum('created','renewed','upgraded','paused','resumed','cancelled','expired','enabled_auto_renew','disabled_auto_renew')"` // 操作类型
	CreatedAt             time.Time `gorm:"column:created_at"`
}

func (SubscriptionHistory) TableName() string { return "subscription_history" }
