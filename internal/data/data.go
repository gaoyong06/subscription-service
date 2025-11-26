package data

import (
	"xinyuan_tech/subscription-service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewDB, NewSubscriptionRepo, NewPaymentClient)

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
	if err := db.AutoMigrate(&Plan{}, &UserSubscription{}, &Order{}); err != nil {
		panic(err)
	}
	initPlans(db)
	return db
}

func initPlans(db *gorm.DB) {
	var count int64
	db.Model(&Plan{}).Count(&count)
	if count == 0 {
		plans := []Plan{
			{ID: "plan_monthly", Name: "Pro Monthly", Description: "Pro features for 1 month", Price: 9.99, Currency: "CNY", DurationDays: 30, Type: "pro"},
			{ID: "plan_yearly", Name: "Pro Yearly", Description: "Pro features for 1 year", Price: 99.99, Currency: "CNY", DurationDays: 365, Type: "pro"},
			{ID: "plan_quarterly", Name: "Pro Quarterly", Description: "Pro features for 3 months", Price: 25.99, Currency: "CNY", DurationDays: 90, Type: "pro"},
		}
		if err := db.Create(&plans).Error; err != nil {
			panic(err)
		}
	}
}
