package server

import (
	"encoding/json"
	stdhttp "net/http"

	"github.com/gaoyong06/go-pkg/health"
	"github.com/gaoyong06/go-pkg/middleware/i18n"

	v1 "xinyuan_tech/subscription-service/api/subscription/v1"
	"xinyuan_tech/subscription-service/internal/conf"
	"xinyuan_tech/subscription-service/internal/service"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Bootstrap, sub *service.SubscriptionService, logger log.Logger) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			// 添加参数验证中间件
			validate.Validator(),
			// 添加 i18n 中间件
			i18n.Middleware(),
		),
		http.ErrorEncoder(customErrorEncoder),
	}
	if c.Server.Http.Addr != "" {
		opts = append(opts, http.Address(c.Server.Http.Addr))
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

func customErrorEncoder(w stdhttp.ResponseWriter, r *stdhttp.Request, err error) {
	se := kerrors.FromError(err)
	status := stdhttp.StatusInternalServerError
	response := map[string]interface{}{
		"code":    status,
		"message": "internal server error",
	}

	if se != nil {
		status = mapErrorStatus(int(se.Code))
		response["code"] = se.Code
		response["reason"] = se.Reason
		response["message"] = se.Message
		if len(se.Metadata) > 0 {
			response["metadata"] = se.Metadata
		}
	} else if err != nil {
		response["message"] = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func mapErrorStatus(code int) int {
	if code >= 100 && code < 600 {
		return code
	}
	if code >= 130000 && code < 140000 {
		return stdhttp.StatusBadRequest
	}
	return stdhttp.StatusInternalServerError
}
