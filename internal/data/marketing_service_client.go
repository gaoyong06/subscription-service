package data

import (
	"fmt"
	"xinyuan_tech/subscription-service/internal/conf"

	marketingv1 "marketing-service/api/marketing_service/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type marketingServiceClient struct {
	client marketingv1.MarketingClient
}

// MarketingClient 营销服务客户端接口（防腐层）
// 注意：当前 subscription-service 可能不需要调用 marketing service
// 但为了保持 wire 依赖注入的一致性，提供此接口
type MarketingClient interface {
	// 如果需要调用 marketing service，可以在这里添加方法
	// ValidateCoupon(ctx context.Context, couponCode, appID string, amount int64) (bool, int64, error)
}

func NewMarketingClient(c *conf.Bootstrap) (MarketingClient, error) {
	addr := ""
	if c != nil && c.GetClient() != nil && c.GetClient().GetMarketingService() != nil {
		addr = c.GetClient().GetMarketingService().GetAddr()
	}
	if addr == "" {
		return nil, fmt.Errorf("marketing service address is required")
	}

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &marketingServiceClient{
		client: marketingv1.NewMarketingClient(conn),
	}, nil
}

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
