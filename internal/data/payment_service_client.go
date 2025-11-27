package data

import (
	"context"
	"xinyuan_tech/subscription-service/internal/biz"
	"xinyuan_tech/subscription-service/internal/conf"

	paymentv1 "xinyuan_tech/payment-service/api/payment/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type paymentServiceClient struct {
	client paymentv1.PaymentClient
}

func NewPaymentClient(c *conf.Bootstrap) (biz.PaymentClient, error) {
	conn, err := grpc.Dial(c.Client.PaymentService.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &paymentServiceClient{
		client: paymentv1.NewPaymentClient(conn),
	}, nil
}

func (c *paymentServiceClient) CreatePayment(ctx context.Context, orderID string, userID uint64, amount float64, method, subject, returnURL string) (string, string, string, string, error) {
	req := &paymentv1.CreatePaymentRequest{
		OrderId:   orderID,
		UserId:    userID,
		Amount:    amount,
		Currency:  "CNY",
		Method:    method,
		Subject:   subject,
		ReturnUrl: returnURL,
	}

	resp, err := c.client.CreatePayment(ctx, req)
	if err != nil {
		return "", "", "", "", err
	}

	return resp.PaymentId, resp.PayUrl, resp.PayCode, resp.PayParams, nil
}
