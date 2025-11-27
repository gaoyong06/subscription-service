package data

import (
	"xinyuan_tech/subscription-service/internal/conf"
	"xinyuan_tech/subscription-service/internal/data/model"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewDB,
	NewPlanRepo,
	NewUserSubscriptionRepo,
	NewSubscriptionOrderRepo,
	NewSubscriptionHistoryRepo,
	NewPaymentClient,
)

// Data .
type Data struct {
	db *gorm.DB
}

// NewData .
func NewData(c *conf.Bootstrap, logger log.Logger, db *gorm.DB) (*Data, func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
	}
	return &Data{db: db}, cleanup, nil
}

// NewDB .
func NewDB(c *conf.Bootstrap) *gorm.DB {
	db, err := gorm.Open(mysql.Open(c.Data.Database.Source), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(&model.Plan{}, &model.UserSubscription{}, &model.SubscriptionOrder{}, &model.SubscriptionHistory{}, &model.PlanPricing{}); err != nil {
		panic(err)
	}
	initPlans(db)
	initPlanPricing(db)
	return db
}

func initPlans(db *gorm.DB) {
	var count int64
	db.Model(&model.Plan{}).Count(&count)
	if count == 0 {
		plans := []model.Plan{
			{ID: "plan_monthly", Name: "Pro Monthly", Description: "Pro features for 1 month", Price: 9.99, Currency: "USD", DurationDays: 30, Type: "pro"},
			{ID: "plan_yearly", Name: "Pro Yearly", Description: "Pro features for 1 year", Price: 99.99, Currency: "USD", DurationDays: 365, Type: "pro"},
			{ID: "plan_quarterly", Name: "Pro Quarterly", Description: "Pro features for 3 months", Price: 25.99, Currency: "USD", DurationDays: 90, Type: "pro"},
		}
		if err := db.Create(&plans).Error; err != nil {
			panic(err)
		}
	}
}

func initPlanPricing(db *gorm.DB) {
	var count int64
	db.Model(&model.PlanPricing{}).Count(&count)
	if count == 0 {
		pricings := []model.PlanPricing{
			// Monthly plan pricing
			{PlanID: "plan_monthly", Region: "US", Price: 9.99, Currency: "USD"},
			{PlanID: "plan_monthly", Region: "CN", Price: 68.00, Currency: "CNY"},
			{PlanID: "plan_monthly", Region: "EU", Price: 8.99, Currency: "EUR"},
			// Yearly plan pricing
			{PlanID: "plan_yearly", Region: "US", Price: 99.99, Currency: "USD"},
			{PlanID: "plan_yearly", Region: "CN", Price: 688.00, Currency: "CNY"},
			{PlanID: "plan_yearly", Region: "EU", Price: 89.99, Currency: "EUR"},
			// Quarterly plan pricing
			{PlanID: "plan_quarterly", Region: "US", Price: 25.99, Currency: "USD"},
			{PlanID: "plan_quarterly", Region: "CN", Price: 178.00, Currency: "CNY"},
			{PlanID: "plan_quarterly", Region: "EU", Price: 23.99, Currency: "EUR"},
		}
		if err := db.Create(&pricings).Error; err != nil {
			panic(err)
		}
	}
}
