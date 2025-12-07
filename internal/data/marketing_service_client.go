package data

import (
	"context"
	"fmt"
	"xinyuan_tech/subscription-service/internal/biz"
	"xinyuan_tech/subscription-service/internal/conf"

	marketingv1 "marketing-service/api/marketing_service/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type marketingServiceClient struct {
	client marketingv1.MarketingClient
}

func NewMarketingClient(c *conf.Bootstrap) (biz.MarketingClient, error) {
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

// ValidateCoupon 验证优惠券
func (c *marketingServiceClient) ValidateCoupon(ctx context.Context, code, appID string, amount int64) (*biz.CouponValidation, error) {
	req := &marketingv1.ValidateCouponRequest{
		Code:   code,
		AppId:  appID,
		Amount: amount,
	}

	resp, err := c.client.ValidateCoupon(ctx, req)
	if err != nil {
		return nil, err
	}

	if !resp.Valid {
		return &biz.CouponValidation{
			Valid:   false,
			Message: resp.Message,
		}, nil
	}

	return &biz.CouponValidation{
		Valid:          true,
		Message:        resp.Message,
		DiscountAmount: resp.DiscountAmount,
		FinalAmount:    resp.FinalAmount,
		CouponCode:     code,
	}, nil
}

// UseCoupon 使用优惠券
func (c *marketingServiceClient) UseCoupon(ctx context.Context, code string, userID uint64, orderID, paymentID string, originalAmount, discountAmount, finalAmount int64) error {
	req := &marketingv1.UseCouponRequest{
		Code:           code,
		UserId:         userID,
		OrderId:        orderID,
		PaymentId:      paymentID,
		OriginalAmount: originalAmount,
		DiscountAmount: discountAmount,
		FinalAmount:    finalAmount,
	}

	resp, err := c.client.UseCoupon(ctx, req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("failed to use coupon: %s", resp.Message)
	}

	return nil
}

