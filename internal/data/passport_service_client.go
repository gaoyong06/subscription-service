package data

import (
	"context"
	"fmt"
	"xinyuan_tech/subscription-service/internal/biz"
	"xinyuan_tech/subscription-service/internal/conf"

	passportv1 "passport-service/api/passport/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type passportServiceClient struct {
	client passportv1.PassportClient
}

// NewPassportClient 创建用户服务客户端
func NewPassportClient(c *conf.Bootstrap) (biz.PassportClient, error) {
	addr := ""
	if c != nil && c.GetClient() != nil && c.GetClient().GetPassportService() != nil {
		addr = c.GetClient().GetPassportService().GetAddr()
	}
	if addr == "" {
		// 如果没有配置，返回空实现（优雅降级）
		return &emptyPassportClient{}, nil
	}

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		// 如果连接失败，返回空实现（优雅降级）
		return &emptyPassportClient{}, nil
	}
	return &passportServiceClient{
		client: passportv1.NewPassportClient(conn),
	}, nil
}

// GetUserCountryCode 获取用户的国家代码
func (c *passportServiceClient) GetUserCountryCode(ctx context.Context, userID uint64) (string, error) {
	req := &passportv1.GetUserRequest{
		Uid: userID,
	}

	resp, err := c.client.GetUser(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to get user from passport service: %w", err)
	}

	// 返回用户注册信息中的 iso_code（国家代码）
	if resp.IsoCode != "" {
		return resp.IsoCode, nil
	}

	return "", nil
}

// GetCountryCodeByIP 根据 IP 地址获取国家代码
// 调用 passport-service 的 GetLocationByIP API，返回完整的地理位置信息
// 在业务场景中，只使用 country code（isoCode）
func (c *passportServiceClient) GetCountryCodeByIP(ctx context.Context, ip string) (string, error) {
	req := &passportv1.GetLocationByIPRequest{
		Ip: ip,
	}

	resp, err := c.client.GetLocationByIP(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to get location from IP: %w", err)
	}

	// 返回国家代码（ISO 3166-1 alpha-2）
	return resp.IsoCode, nil
}

// emptyPassportClient 空的用户服务客户端实现（优雅降级）
type emptyPassportClient struct{}

func (e *emptyPassportClient) GetUserCountryCode(ctx context.Context, userID uint64) (string, error) {
	return "", nil
}

func (e *emptyPassportClient) GetCountryCodeByIP(ctx context.Context, ip string) (string, error) {
	// 空实现，返回空字符串
	return "", nil
}
