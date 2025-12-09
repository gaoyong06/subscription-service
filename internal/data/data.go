package data

import (
	"context"
	"time"
	"xinyuan_tech/subscription-service/internal/biz"
	"xinyuan_tech/subscription-service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9" // Updated Redis import
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewDB,
	NewRedis,
	NewRedsync, // 添加 redsync
	NewPlanRepo,
	NewUserSubscriptionRepo,
	NewSubscriptionOrderRepo,
	NewSubscriptionHistoryRepo,
	NewPaymentClient,
	NewPassportClient,
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

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}

	if c != nil && c.GetData() != nil && c.GetData().GetDatabase() != nil {
		dbConf := c.GetData().GetDatabase()
		if dbConf.GetMaxIdleConns() > 0 {
			sqlDB.SetMaxIdleConns(int(dbConf.GetMaxIdleConns()))
		}
		if dbConf.GetMaxOpenConns() > 0 {
			sqlDB.SetMaxOpenConns(int(dbConf.GetMaxOpenConns()))
		}
		if dbConf.GetConnMaxLifetime() != nil {
			sqlDB.SetConnMaxLifetime(dbConf.GetConnMaxLifetime().AsDuration())
		}
	}
	return db
}

// NewRedis .
func NewRedis(c *conf.Bootstrap) *redis.Client {
	var readTimeout, writeTimeout, dialTimeout time.Duration
	var addr, password string
	var db, poolSize, minIdleConns int32

	if c != nil && c.GetData() != nil && c.GetData().GetRedis() != nil {
		redisConf := c.GetData().GetRedis()
		if redisConf.GetReadTimeout() != nil {
			readTimeout = redisConf.GetReadTimeout().AsDuration()
		}
		if redisConf.GetWriteTimeout() != nil {
			writeTimeout = redisConf.GetWriteTimeout().AsDuration()
		}
		if redisConf.GetDialTimeout() != nil {
			dialTimeout = redisConf.GetDialTimeout().AsDuration()
		}
		addr = redisConf.GetAddr()
		password = redisConf.GetPassword()
		db = redisConf.GetDb()
		poolSize = redisConf.GetPoolSize()
		minIdleConns = redisConf.GetMinIdleConns()
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
		DialTimeout:  dialTimeout,
		PoolSize:     int(poolSize),
		MinIdleConns: int(minIdleConns),
	})
	return rdb
}

// NewRedsync 创建 redsync 实例
func NewRedsync(rdb *redis.Client) *redsync.Redsync {
	pool := goredis.NewPool(rdb)
	return redsync.New(pool)
}
