package model

// PlanPricing 套餐区域定价模型
type PlanPricing struct {
	ID       uint64  `gorm:"primaryKey;column:id;autoIncrement"`
	PlanID   string  `gorm:"column:plan_id;index"`
	Region   string  `gorm:"column:region;index"` // US, CN, EU, etc.
	Price    float64 `gorm:"column:price"`
	Currency string  `gorm:"column:currency"`
}

func (PlanPricing) TableName() string { return "plan_pricing" }
