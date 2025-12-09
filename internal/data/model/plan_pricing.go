package model

import "time"

// PlanPricing 套餐区域定价模型（所有价格都在数据库中配置）
type PlanPricing struct {
	PlanPricingID uint64    `gorm:"primaryKey;column:plan_pricing_id;autoIncrement;type:bigint unsigned"`
	PlanID        string    `gorm:"column:plan_id;type:varchar(50);not null;index:idx_plan_id;uniqueIndex:uk_plan_country"`
	AppID         string    `gorm:"column:app_id;type:varchar(50);not null;index:idx_app_id;index:idx_app_plan_country"`              // 应用ID（冗余字段，便于按app查询）
	CountryCode   string    `gorm:"column:country_code;type:varchar(10);not null;index:idx_country_code;uniqueIndex:uk_plan_country"` // ISO 3166-1 alpha-2
	Price         float64   `gorm:"column:price;type:decimal(10,2);not null"`
	Currency      string    `gorm:"column:currency;type:varchar(10);not null"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (PlanPricing) TableName() string { return "plan_pricing" }
