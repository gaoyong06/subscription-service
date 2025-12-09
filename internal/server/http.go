package server

import (
	"github.com/gaoyong06/go-pkg/health"
	"github.com/gaoyong06/go-pkg/middleware/app_id"
	"github.com/gaoyong06/go-pkg/middleware/i18n"
	"github.com/gaoyong06/go-pkg/middleware/response"

	v1 "xinyuan_tech/subscription-service/api/subscription/v1"
	"xinyuan_tech/subscription-service/internal/conf"
	"xinyuan_tech/subscription-service/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/validate"
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

	// 使用默认错误处理器（已支持 Kratos errors 的 HTTP 状态码映射）
	errorHandler := response.NewDefaultErrorHandler()

	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			// 添加 app_id 中间件（优先于其他中间件，确保 app_id 在 Context 中可用）
			app_id.Middleware(),
			// 添加参数验证中间件
			validate.Validator(),
			// 添加 i18n 中间件
			i18n.Middleware(),
		),
		// 使用统一响应格式编码器（已支持 OPTIONS 请求处理）
		http.ResponseEncoder(response.NewResponseEncoder(errorHandler, responseConfig)),
		// 使用统一错误编码器
		http.ErrorEncoder(response.NewErrorEncoder(errorHandler)),
	}
	if c != nil && c.GetServer() != nil && c.GetServer().GetHttp() != nil {
		if addr := c.GetServer().GetHttp().GetAddr(); addr != "" {
			opts = append(opts, http.Address(addr))
		}
		if timeout := c.GetServer().GetHttp().GetTimeout(); timeout != nil {
			opts = append(opts, http.Timeout(timeout.AsDuration()))
		}
	}
	srv := http.NewServer(opts...)

	// 注册业务路由
	v1.RegisterSubscriptionHTTPServer(srv, sub)

	// 注册健康检查端点
	srv.Route("/").GET("/health", func(ctx http.Context) error {
		return ctx.Result(200, health.NewResponse("subscription-service"))
	})

	return srv
}
