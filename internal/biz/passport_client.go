package biz

import "context"

// PassportClient 用户服务客户端接口（防腐层）
type PassportClient interface {
	// GetUserCountryCode 获取用户的国家代码（ISO 3166-1 alpha-2）
	// 从用户注册信息中的 iso_code 字段获取
	GetUserCountryCode(ctx context.Context, uid string) (string, error)
	// GetCountryCodeByIP 根据 IP 地址获取国家代码（ISO 3166-1 alpha-2）
	// 如果 passport-service 提供了 GeoIP API，则调用该 API
	// 否则返回空字符串，由调用方处理
	GetCountryCodeByIP(ctx context.Context, ip string) (string, error)
}
