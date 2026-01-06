//go:build wireinject
// +build wireinject

package main

import (
	"os"
	"xinyuan_tech/subscription-service/internal/biz"
	"xinyuan_tech/subscription-service/internal/conf"
	"xinyuan_tech/subscription-service/internal/data"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// SchedulerApp Scheduler 应用结构
type SchedulerApp struct {
	subscriptionUsecase *biz.SubscriptionUsecase
}

// wireApp 初始化应用
func wireApp(*conf.Bootstrap) (*SchedulerApp, func(), error) {
	panic(wire.Build(
		// Logger
		wire.FieldsOf(new(*conf.Bootstrap), "Log"),
		newLogger,
		
		// Data 层
		data.ProviderSet,
		
		// Biz 层
		biz.ProviderSet,
		
		// App 结构
		wire.Struct(new(SchedulerApp), "*"),
	))
}

// newLogger 创建 logger
func newLogger(c *conf.Log) log.Logger {
	return log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.name", "subscription-scheduler",
	)
}

