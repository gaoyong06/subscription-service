package data

import (
	"xinyuan_tech/subscription-service/internal/conf"
)

// MarketingClient 营销服务客户端接口（防腐层）
// 注意：当前 subscription-service 可能不需要调用 marketing service
// 但为了保持 wire 依赖注入的一致性，提供此接口
type MarketingClient interface {
	// 如果需要调用 marketing service，可以在这里添加方法
	// ValidateCoupon(ctx context.Context, couponCode, appID string, amount int64) (bool, int64, error)
}

func NewMarketingClient(c *conf.Bootstrap) (MarketingClient, error) {
	// 当前 subscription-service 不需要调用 marketing service
	// 返回空实现以保持 wire 依赖注入的一致性
	return &emptyMarketingClient{}, nil
}

// emptyMarketingClient 空的营销服务客户端实现（当前不需要调用 marketing service）
type emptyMarketingClient struct{}

// 如果需要实现具体的营销服务调用方法，可以在这里添加
// func (c *marketingServiceClient) ValidateCoupon(ctx context.Context, couponCode, appID string, amount int64) (bool, int64, error) {
// 	req := &marketingv1.ValidateCouponRequest{
// 		CouponCode: couponCode,
// 		AppId:      appID,
// 		Amount:     amount,
// 	}
// 	resp, err := c.client.ValidateCoupon(ctx, req)
// 	if err != nil {
// 		return false, 0, err
// 	}
// 	return resp.Valid, resp.DiscountAmount, nil
// }
