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
// TODO: passport-service 的 GetUserRequest.Uid 仍然是 uint64，需要迁移
// 当前实现：暂时返回空字符串，等待 passport-service 迁移完成
// 迁移后：可以直接使用字符串 uid 调用 passport-service
func (c *passportServiceClient) GetUserCountryCode(ctx context.Context, uid string) (string, error) {
	// TODO: passport-service 迁移后，可以直接使用字符串 uid
	// 当前 passport-service 的 GetUserRequest.Uid 是 uint64，需要转换
	// 但根据黄金法则，不应该在跨服务调用中使用 user_internal_id
	// 建议：修改 passport-service 的 proto 文件，将 GetUserRequest.Uid 改为 string
	//
	// 临时方案：返回空字符串，等待 passport-service 迁移
	// 迁移后的代码：
	// req := &passportv1.GetUserRequest{
	// 	Uid: uid, // 字符串 UUID
	// }
	// resp, err := c.client.GetUser(ctx, req)
	// if err != nil {
	// 	return "", fmt.Errorf("failed to get user from passport service: %w", err)
	// }
	// return resp.IsoCode, nil
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

func (e *emptyPassportClient) GetUserCountryCode(ctx context.Context, uid string) (string, error) {
	return "", nil
}

func (e *emptyPassportClient) GetCountryCodeByIP(ctx context.Context, ip string) (string, error) {
	// 空实现，返回空字符串
	return "", nil
}
