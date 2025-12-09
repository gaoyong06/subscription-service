package model

import "time"

// Plan 套餐模型
type Plan struct {
	PlanID       string    `gorm:"primaryKey;column:plan_id"`
	AppID        string    `gorm:"column:app_id;not null;index:idx_app_id;index:idx_app_uid"`
	UID          string    `gorm:"column:uid;not null;index:idx_uid;index:idx_app_uid"` // 开发者ID（用户ID）
	Name         string    `gorm:"column:name"`
	Description  string    `gorm:"column:description"`
	Price        float64   `gorm:"column:price"` // 默认价格（用于兜底）
	Currency     string    `gorm:"column:currency;default:'USD'"` // 默认币种（用于兜底）
	DurationDays int       `gorm:"column:duration_days"`
	Type         string    `gorm:"column:type"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Plan) TableName() string { return "plan" }
