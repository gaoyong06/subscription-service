package constants

import "time"

// 缓存相关常量
const (
	// DefaultCacheExpiration 默认缓存过期时间
	DefaultCacheExpiration = time.Hour
	// NullCacheExpiration 空值缓存过期时间 (防止缓存穿透)
	NullCacheExpiration = 5 * time.Minute
	// CacheRandomMaxSeconds 缓存随机过期时间最大值(秒) - 防止缓存雪崩
	CacheRandomMaxSeconds = 600 // 10分钟
)

// 分页相关常量
const (
	// DefaultPageSize 默认分页大小
	DefaultPageSize = 10
	// MaxPageSize 最大分页大小
	MaxPageSize = 100
)

// 订阅相关常量
const (
	// DefaultExpiryDays 默认过期检查天数
	DefaultExpiryDays = 7
	// MaxExpiryDays 最大过期检查天数
	MaxExpiryDays = 30
	// DefaultAutoRenewDays 默认自动续费提前天数
	DefaultAutoRenewDays = 3
)

// 分布式锁相关常量
const (
	// AutoRenewLockExpiration 自动续费锁过期时间
	AutoRenewLockExpiration = 10 * time.Minute
	// AutoRenewLockRetries 自动续费锁重试次数
	AutoRenewLockRetries = 1
)

// 支持的区域列表
var SupportedRegions = map[string]bool{
	"default": true,
	"CN":      true,
	"US":      true,
	"EU":      true,
}

// 订阅状态
const (
	StatusActive    = "active"
	StatusExpired   = "expired"
	StatusPaused    = "paused"
	StatusCancelled = "cancelled"
)

// 订阅操作
const (
	ActionCreated           = "created"
	ActionRenewed           = "renewed"
	ActionUpgraded          = "upgraded"
	ActionPaused            = "paused"
	ActionResumed           = "resumed"
	ActionCancelled         = "cancelled"
	ActionExpired           = "expired"
	ActionEnabledAutoRenew  = "enabled_auto_renew"
	ActionDisabledAutoRenew = "disabled_auto_renew"
)

// 支付状态(与payment-service保持一致)
const (
	PaymentStatusPending           = "pending"            // 待支付(订单已创建，等待支付)
	PaymentStatusSuccess           = "success"            // 支付成功
	PaymentStatusFailed            = "failed"             // 支付失败
	PaymentStatusClosed            = "closed"             // 订单关闭
	PaymentStatusRefunded          = "refunded"           // 已全额退款
	PaymentStatusPartiallyRefunded = "partially_refunded" // 部分退款
)

// 支付来源常量（用于 payment-service）
const (
	// PaymentSourceBilling 充值来源
	PaymentSourceBilling = "billing"
	// PaymentSourceSubscription 订阅来源
	PaymentSourceSubscription = "subscription"
)
