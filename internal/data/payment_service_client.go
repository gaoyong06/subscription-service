package data

import (
	"context"
	"fmt"
	"subscription-service/internal/biz"
	"subscription-service/internal/conf"
	"subscription-service/internal/constants"

	paymentv1 "payment-service/api/payment/v1"

	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

type paymentServiceClient struct {
	client paymentv1.PaymentClient
}

func NewPaymentClient(c *conf.Bootstrap) (biz.PaymentClient, error) {
	addr := ""
	if c != nil && c.GetClient() != nil && c.GetClient().GetPaymentService() != nil {
		addr = c.GetClient().GetPaymentService().GetAddr()
	}
	if addr == "" {
		return nil, fmt.Errorf("payment service address is required")
	}

	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(addr),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		return nil, err
	}
	return &paymentServiceClient{
		client: paymentv1.NewPaymentClient(conn),
	}, nil
}

func (c *paymentServiceClient) CreatePayment(ctx context.Context, orderID string, userId string, amount float64, currency, method, subject, returnURL, campaignID, clickID string) (string, string, string, string, error) {
	// 验证必填参数
	if currency == "" {
		return "", "", "", "", fmt.Errorf("currency is required")
	}

	// 将字符串转换为 PaymentMethod 枚举
	var paymentMethod paymentv1.PaymentMethod
	switch method {
	case "alipay":
		paymentMethod = paymentv1.PaymentMethod_PAYMENT_METHOD_ALIPAY
	case "wechatpay":
		paymentMethod = paymentv1.PaymentMethod_PAYMENT_METHOD_WECHATPAY
	default:
		paymentMethod = paymentv1.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED
	}

	req := &paymentv1.CreatePaymentRequest{
		OrderId: orderID,
		UserId:  userId, // 用户ID（字符串 UUID）
		// 注意：appId 现在只从 Context 获取（由中间件从 Header/metadata 提取），不再从请求体传递
		Source:     constants.PaymentSourceSubscription, // 标记来源为订阅
		Amount:     int64(amount),
		Currency:   currency,
		Method:     paymentMethod,
		Subject:    subject,
		ReturnUrl:  returnURL,
		CampaignId: campaignID,
		ClickId:    clickID,
	}

	resp, err := c.client.CreatePayment(ctx, req)
	if err != nil {
		return "", "", "", "", err
	}

	return resp.PaymentId, resp.PayUrl, resp.PayCode, resp.PayParams, nil
}
