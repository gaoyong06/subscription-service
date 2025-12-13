package data

import (
	"context"
	"fmt"
	"xinyuan_tech/subscription-service/internal/biz"
	"xinyuan_tech/subscription-service/internal/conf"
	"xinyuan_tech/subscription-service/internal/constants"

	paymentv1 "xinyuan_tech/payment-service/api/payment/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &paymentServiceClient{
		client: paymentv1.NewPaymentClient(conn),
	}, nil
}

func (c *paymentServiceClient) CreatePayment(ctx context.Context, orderID string, uid string, amount float64, currency, method, subject, returnURL string) (string, string, string, string, error) {
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
		Uid:     uid, // 用户ID（字符串 UUID）
		// 注意：appId 现在只从 Context 获取（由中间件从 Header/metadata 提取），不再从请求体传递
		Source:    constants.PaymentSourceSubscription, // 标记来源为订阅
		Amount:    int64(amount),
		Currency:  currency,
		Method:    paymentMethod,
		Subject:   subject,
		ReturnUrl: returnURL,
	}

	resp, err := c.client.CreatePayment(ctx, req)
	if err != nil {
		return "", "", "", "", err
	}

	return resp.PaymentId, resp.PayUrl, resp.PayCode, resp.PayParams, nil
}
