package server

import (
	v1 "subscription-service/api/subscription/v1"
	"subscription-service/internal/conf"
	"subscription-service/internal/service"

	"github.com/gaoyong06/go-pkg/middleware/app_id"
	"github.com/gaoyong06/go-pkg/middleware/developer_id"
	"github.com/gaoyong06/go-pkg/middleware/i18n"
	"github.com/gaoyong06/go-pkg/middleware/user_id"

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
			app_id.Middleware(),
			developer_id.Middleware(),
			// 添加参数验证中间件
			validate.Validator(),
			// 添加 user_id 中间件（提取终端用户 ID，支持 gRPC metadata: x-end-user-id）
			user_id.Middleware(),
			// 添加 i18n 中间件
			i18n.Middleware(),
		),
	}
	if c != nil && c.GetServer() != nil && c.GetServer().GetGrpc() != nil {
		if addr := c.GetServer().GetGrpc().GetAddr(); addr != "" {
			opts = append(opts, grpc.Address(addr))
		}
		if timeout := c.GetServer().GetGrpc().GetTimeout(); timeout != nil {
			opts = append(opts, grpc.Timeout(timeout.AsDuration()))
		}
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterSubscriptionServer(srv, sub)
	return srv
}
