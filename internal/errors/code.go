package errors

import (
	pkgErrors "github.com/gaoyong06/go-pkg/errors"
	i18nPkg "github.com/gaoyong06/go-pkg/middleware/i18n"
)

func init() {
	// 初始化全局错误管理器（使用项目特定的配置）
	pkgErrors.InitGlobalErrorManager("i18n", i18nPkg.Language)
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
	// ErrCodePlanPriceInvalid 套餐价格无效错误
	ErrCodePlanPriceInvalid = 130102
	// ErrCodePlanPricingNotFound 套餐区域定价不存在错误
	ErrCodePlanPricingNotFound = 130103
)

// 订阅生命周期模块 (130200-130299)
const (
	// ErrCodeSubscriptionNotFound 订阅不存在错误
	ErrCodeSubscriptionNotFound = 130201
	// ErrCodeSubscriptionNotActive 订阅未激活错误
	ErrCodeSubscriptionNotActive = 130202
	// ErrCodeSubscriptionExpired 订阅已过期错误
	ErrCodeSubscriptionExpired = 130203
	// ErrCodeInvalidStatus 无效的订阅状态错误
	ErrCodeInvalidStatus = 130204
	// ErrCodeAlreadySubscribed 用户已有活跃订阅错误
	ErrCodeAlreadySubscribed = 130205
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
	// ErrCodePaymentInvalidAmount 支付金额无效错误
	ErrCodePaymentInvalidAmount = 130402
)
