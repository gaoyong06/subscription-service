package server

import (
	v1 "xinyuan_tech/subscription-service/api/subscription/v1"
	"xinyuan_tech/subscription-service/internal/conf"
	"xinyuan_tech/subscription-service/internal/service"

	"github.com/gaoyong06/go-pkg/middleware/i18n"
	"github.com/gaoyong06/go-pkg/middleware/response"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Bootstrap, sub *service.SubscriptionService, logger log.Logger) *http.Server {
	// 响应中间件配置
	responseConfig := &response.Config{
		EnableUnifiedResponse: true,
		IncludeDetailedError:  true, // 开发环境可以为 true
		IncludeHost:           true,
		IncludeTraceId:        true,
	}

	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			// 添加 i18n 中间件
			i18n.Middleware(),
		),
		// 使用自定义响应编码器统一响应格式
		http.ResponseEncoder(response.NewResponseEncoder(response.NewDefaultErrorHandler(), responseConfig)),
		// 使用自定义错误编码器统一错误响应格式
		http.ErrorEncoder(response.NewErrorEncoder(response.NewDefaultErrorHandler())),
	}
	if c.Server.Http.Addr != "" {
		opts = append(opts, http.Address(c.Server.Http.Addr))
	}
	srv := http.NewServer(opts...)

	// 注册业务路由
	v1.RegisterSubscriptionHTTPServer(srv, sub)

	// 注册健康检查端点
	srv.Route("/").GET("/health", func(ctx http.Context) error {
		return ctx.Result(200, map[string]interface{}{
			"status":  "UP",
			"service": "subscription-service",
		})
	})

	return srv
}
