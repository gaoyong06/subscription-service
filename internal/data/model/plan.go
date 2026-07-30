package model

import (
	"time"

	"gorm.io/gorm"
)

// Plan 套餐模型（删除为软删除：DeletedAt 非空表示已删除）
type Plan struct {
	PlanID        string         `gorm:"primaryKey;column:plan_id"`
	AppID         string         `gorm:"column:app_id;not null;index:idx_app_id;index:idx_app_user_id"`
	UserID        string         `gorm:"column:user_id;not null;index:idx_user_id;index:idx_app_user_id"` // 开发者ID（用户ID）
	Name          string         `gorm:"column:name"`
	Description   string         `gorm:"column:description"`
	Price         float64        `gorm:"column:price"`                  // 默认价格（用于兜底）
	Currency      string         `gorm:"column:currency;default:'USD'"` // 默认币种（用于兜底）
	PeriodType    string         `gorm:"column:period_type;type:varchar(16);not null;default:'MONTH'"`
	IntervalCount int32          `gorm:"column:interval_count;not null;default:1"`
	Features      string         `gorm:"column:features;type:json"` // JSON 数组，元素为 i18n key
	Type          string         `gorm:"column:type"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;index"` // 软删除时间
	CreatedAt     time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;autoUpdateTime"`
}

func (Plan) TableName() string { return "plan" }
