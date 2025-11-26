package server

import (
	v1 "xinyuan_tech/subscription-service/api/subscription/v1"
	"xinyuan_tech/subscription-service/internal/conf"
	"xinyuan_tech/subscription-service/internal/service"

	"github.com/gaoyong06/go-pkg/middleware/i18n"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Bootstrap, sub *service.SubscriptionService, logger log.Logger) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			// 添加参数验证中间件
			validate.Validator(),
			// 添加 i18n 中间件
			i18n.Middleware(),
		),
	}
	if c.Server.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Server.Grpc.Addr))
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterSubscriptionServer(srv, sub)
	return srv
}
