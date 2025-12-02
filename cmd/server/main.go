package main

import (
	"context"
	"flag"
	"os"

	"xinyuan_tech/subscription-service/internal/conf"

	"github.com/gaoyong06/go-pkg/errors"
	"github.com/gaoyong06/go-pkg/logger"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string = "subscription-service"
	// Version is the version of the compiled software.
	Version string
	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "configs/config.yaml", "config path, eg: -conf config.yaml")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
		),
	)
}

func main() {
	flag.Parse()

	// 初始化 Kratos Config
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	// 验证配置
	if bc.GetServer() == nil {
		panic("server configuration is required")
	}
	if bc.GetServer().GetHttp() == nil || bc.GetServer().GetHttp().GetAddr() == "" {
		panic("server.http.addr is required")
	}
	if bc.GetServer().GetGrpc() == nil || bc.GetServer().GetGrpc().GetAddr() == "" {
		panic("server.grpc.addr is required")
	}
	if bc.GetData() == nil || bc.GetData().GetDatabase() == nil || bc.GetData().GetDatabase().GetSource() == "" {
		panic("data.database.source is required")
	}
	if bc.GetClient() == nil || bc.GetClient().GetPaymentService() == nil || bc.GetClient().GetPaymentService().GetAddr() == "" {
		panic("client.payment_service.addr is required")
	}
	if bc.GetClient().GetSubscriptionService() == nil {
		panic("client.subscription_service configuration is required")
	}

	// 初始化日志 (使用 go-pkg/logger)
	logConfig := &logger.Config{
		Level:      "info",
		Format:     "json",
		Output:     "stdout",
		FilePath:   "",
		MaxSize:    100,
		MaxAge:     7,
		MaxBackups: 3,
		Compress:   false,
	}
	if bc.GetLog() != nil {
		logConfig.Level = bc.GetLog().GetLevel()
		logConfig.Format = bc.GetLog().GetFormat()
		logConfig.Output = bc.GetLog().GetOutput()
		logConfig.FilePath = bc.GetLog().GetFilePath()
		logConfig.MaxSize = int(bc.GetLog().GetMaxSize())
		logConfig.MaxAge = int(bc.GetLog().GetMaxAge())
		logConfig.MaxBackups = int(bc.GetLog().GetMaxBackups())
		logConfig.Compress = bc.GetLog().GetCompress()
	}

	loggerInstance := logger.NewLogger(logConfig)

	// 添加基本字段
	loggerInstance = log.With(loggerInstance,
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
	)

	// 初始化全局错误管理器
	// 假设 i18n 配置文件在 configs/i18n 目录下
	errors.InitGlobalErrorManager("configs/i18n", func(ctx context.Context) string {
		// 这里可以从 context 中获取语言，暂时返回默认
		return "zh-CN"
	})

	app, cleanup, err := wireApp(&bc, loggerInstance)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
