package data

import (
	"context"
	"time"
	"xinyuan_tech/subscription-service/internal/biz"
	"xinyuan_tech/subscription-service/internal/conf"
	"xinyuan_tech/subscription-service/internal/data/model"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9" // Updated Redis import
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewDB,
	NewRedis, // Added NewRedis
	NewPlanRepo,
	NewUserSubscriptionRepo,
	NewSubscriptionOrderRepo,
	NewSubscriptionHistoryRepo,
	NewPaymentClient,
	wire.Bind(new(biz.Transaction), new(*Data)),
)

// Data .
type Data struct {
	db  *gorm.DB
	rdb *redis.Client
}

type contextTxKey struct{}

// Exec 执行事务
func (d *Data) Exec(ctx context.Context, fn func(ctx context.Context) error) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ctx = context.WithValue(ctx, contextTxKey{}, tx)
		return fn(ctx)
	})
}

// NewData .
func NewData(c *conf.Bootstrap, logger log.Logger, db *gorm.DB, rdb *redis.Client) (*Data, func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
	}
	return &Data{db: db, rdb: rdb}, cleanup, nil
}

// NewDB .
func NewDB(c *conf.Bootstrap) *gorm.DB {
	source := ""
	if c != nil && c.GetData() != nil && c.GetData().GetDatabase() != nil {
		source = c.GetData().GetDatabase().GetSource()
	}
	if source == "" {
		panic("database source is required")
	}

	db, err := gorm.Open(mysql.Open(source), &gorm.Config{})
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

// NewRedis .
func NewRedis(c *conf.Bootstrap) *redis.Client {
	var readTimeout, writeTimeout time.Duration
	var addr, password string
	var db int32

	if c != nil && c.GetData() != nil && c.GetData().GetRedis() != nil {
		redisConf := c.GetData().GetRedis()
		if redisConf.GetReadTimeout() != nil {
			readTimeout = redisConf.GetReadTimeout().AsDuration()
		}
		if redisConf.GetWriteTimeout() != nil {
			writeTimeout = redisConf.GetWriteTimeout().AsDuration()
		}
		addr = redisConf.GetAddr()
		password = redisConf.GetPassword()
		db = redisConf.GetDb()
	}

	if addr == "" {
		addr = "localhost:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           int(db),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	})
	return rdb
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
			// 中国区域定价 - 人民币，价格更亲民
			{PlanID: "plan_monthly", Region: "CN", Price: 38.00, Currency: "CNY"},
			{PlanID: "plan_yearly", Region: "CN", Price: 388.00, Currency: "CNY"},
			{PlanID: "plan_quarterly", Region: "CN", Price: 98.00, Currency: "CNY"},
			// 欧洲区域定价 - 欧元
			{PlanID: "plan_monthly", Region: "EU", Price: 8.99, Currency: "EUR"},
			{PlanID: "plan_yearly", Region: "EU", Price: 89.99, Currency: "EUR"},
			{PlanID: "plan_quarterly", Region: "EU", Price: 23.99, Currency: "EUR"},
			// 注意：US 和其他所有未配置的区域会自动使用 plan 表中的默认 USD 价格
		}
		if err := db.Create(&pricings).Error; err != nil {
			panic(err)
		}
	}
}
