package model

// Plan 套餐模型
type Plan struct {
	ID           string  `gorm:"primaryKey;column:plan_id"`
	Name         string  `gorm:"column:name"`
	Description  string  `gorm:"column:description"`
	Price        float64 `gorm:"column:price"`
	Currency     string  `gorm:"column:currency"`
	DurationDays int     `gorm:"column:duration_days"`
	Type         string  `gorm:"column:type"`
}

func (Plan) TableName() string { return "plan" }
