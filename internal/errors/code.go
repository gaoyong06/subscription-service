package errors

import (
	"net/http"

	pkgErrors "github.com/gaoyong06/go-pkg/errors"
	i18nPkg "github.com/gaoyong06/go-pkg/middleware/i18n"
)

func init() {
	// 初始化全局错误管理器（使用项目特定的配置）
	pkgErrors.InitGlobalErrorManager("i18n", i18nPkg.Language)

	// 业务错误码和http status错误码映射
	pkgErrors.RegisterHTTPStatusMapping("subscription-service", map[int]int{
		ErrCodePlanNotFound:            http.StatusNotFound,
		ErrCodeDefaultFreePlanNotFound: http.StatusNotFound,
		ErrCodeSubscriptionNotFound:    http.StatusNotFound,
		ErrCodeInvalidStatus:           http.StatusBadRequest,
		ErrCodeOrderNotFound:           http.StatusNotFound,
		ErrCodeOrderAlreadyPaid:        http.StatusConflict,
		ErrCodeCannotCancelStatus:      http.StatusConflict,
		ErrCodeCannotPauseStatus:       http.StatusConflict,
		ErrCodeCannotResumeStatus:      http.StatusConflict,
		ErrCodeCannotSetAutoRenew:      http.StatusConflict,
		ErrCodeOrderCreateFailed:       http.StatusInternalServerError,
		ErrCodePaymentFailed:           http.StatusBadGateway,
	})
}

// 订阅服务错误码定义
// 错误码格式：SSMMEE (6位数字)，其中 SS=13 表示 subscription-service
// 模块划分：
//   01: 套餐模块
//   02: 订阅生命周期
//   03: 订单模块
//   04: 支付模块

// 套餐模块 (130100-130199)
const (
	// ErrCodePlanNotFound 套餐不存在错误
	ErrCodePlanNotFound = 130101
	// ErrCodeDefaultFreePlanNotFound 当前应用未配置免费 FOREVER 套餐，无法开通默认免费档
	ErrCodeDefaultFreePlanNotFound = 130102
)

// 订阅生命周期模块 (130200-130299)
const (
	// ErrCodeSubscriptionNotFound 订阅不存在错误
	ErrCodeSubscriptionNotFound = 130201
	// ErrCodeInvalidStatus 无效的订阅状态错误
	ErrCodeInvalidStatus = 130204
	// ErrCodeCannotCancelStatus 当前状态无法取消订阅错误
	ErrCodeCannotCancelStatus = 130206
	// ErrCodeCannotPauseStatus 当前状态无法暂停订阅错误
	ErrCodeCannotPauseStatus = 130207
	// ErrCodeCannotResumeStatus 当前状态无法恢复订阅错误
	ErrCodeCannotResumeStatus = 130208
	// ErrCodeCannotSetAutoRenew 当前状态无法设置自动续费错误
	ErrCodeCannotSetAutoRenew = 130209
)

// 订单模块 (130300-130399)
const (
	// ErrCodeOrderNotFound 订单不存在错误
	ErrCodeOrderNotFound = 130301
	// ErrCodeOrderAlreadyPaid 订单已支付错误
	ErrCodeOrderAlreadyPaid = 130302
	// ErrCodeOrderCreateFailed 订单创建失败错误
	ErrCodeOrderCreateFailed = 130303
)

// 支付模块 (130400-130499)
const (
	// ErrCodePaymentFailed 支付服务错误
	ErrCodePaymentFailed = 130401
)
