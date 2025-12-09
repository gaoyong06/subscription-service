package biz

import (
	"context"
	"strings"

	"github.com/gaoyong06/go-pkg/utils"
	"github.com/go-kratos/kratos/v2/log"
)

// RegionDetectionService 地区推断服务接口
// 根据多维度信息推断用户所在国家代码（ISO 3166-1 alpha-2）
type RegionDetectionService interface {
	// DetectRegion 推断用户所在地区
	// 优先级：
	// 1. 用户注册信息（从 passport-service 查询，如果用户已注册）
	// 2. GeoIP 数据库（根据 IP 地址，如果配置了 GeoIP）
	// 3. HTTP 头 Accept-Language（从语言推断）
	// 4. HTTP 头 X-Language（从语言推断）
	// 5. 默认值 "default"
	DetectRegion(ctx context.Context, userID uint64, clientIP, acceptLanguage, xLanguage string) (string, error)
}

// regionDetectionService 地区推断服务实现
type regionDetectionService struct {
	log            *log.Helper
	passportClient PassportClient // passport-service 客户端，用于查询用户信息和 GeoIP
}

// NewRegionDetectionService 创建地区推断服务
func NewRegionDetectionService(passportClient PassportClient, logger log.Logger) RegionDetectionService {
	return &regionDetectionService{
		log:            log.NewHelper(logger),
		passportClient: passportClient,
	}
}

// DetectRegion 推断用户所在地区
func (s *regionDetectionService) DetectRegion(ctx context.Context, userID uint64, clientIP, acceptLanguage, xLanguage string) (string, error) {
	// 优先级 1: 从 passport-service 查询用户注册信息中的国家代码（如果用户已注册）
	if userID > 0 && s.passportClient != nil {
		countryCode, err := s.passportClient.GetUserCountryCode(ctx, userID)
		if err == nil && countryCode != "" {
			s.log.WithContext(ctx).Infof("Detected region from user profile: %s (userID: %d)", countryCode, userID)
			return countryCode, nil
		}
		if err != nil {
			s.log.WithContext(ctx).Warnf("Failed to get user country code from passport service: %v", err)
		}
	}

	// 优先级 2: 从 GeoIP 数据库查询（根据 IP 地址）
	// 如果 passport-service 提供了 GeoIP API，则调用该 API
	if clientIP != "" && s.passportClient != nil && utils.IsValidPublicIP(clientIP) {
		countryCode, err := s.passportClient.GetCountryCodeByIP(ctx, clientIP)
		if err == nil && countryCode != "" {
			s.log.WithContext(ctx).Infof("Detected region from GeoIP: %s (IP: %s)", countryCode, clientIP)
			return countryCode, nil
		}
		if err != nil {
			s.log.WithContext(ctx).Warnf("Failed to get country code from IP: %v", err)
		}
		// 如果返回空字符串，可能是 API 未实现，继续使用其他方式推断
	}

	// 优先级 3: 从 Accept-Language 头推断
	if acceptLanguage != "" {
		countryCode := s.extractCountryFromLanguage(acceptLanguage)
		if countryCode != "" {
			s.log.WithContext(ctx).Infof("Detected region from Accept-Language: %s", countryCode)
			return countryCode, nil
		}
	}

	// 优先级 4: 从 X-Language 头推断
	if xLanguage != "" {
		countryCode := s.extractCountryFromLanguage(xLanguage)
		if countryCode != "" {
			s.log.WithContext(ctx).Infof("Detected region from X-Language: %s", countryCode)
			return countryCode, nil
		}
	}

	// 优先级 5: 默认值
	s.log.WithContext(ctx).Infof("Using default region: default")
	return "default", nil
}

// extractCountryFromLanguage 从语言字符串中提取国家代码
// 支持格式：zh-CN, zh, en-US, en 等
func (s *regionDetectionService) extractCountryFromLanguage(langStr string) string {
	if langStr == "" {
		return ""
	}

	// 语言到国家代码的映射（常见语言）
	langToCountry := map[string]string{
		"zh":    "CN", // 中文 -> 中国
		"zh-CN": "CN", // 简体中文 -> 中国
		"zh-TW": "TW", // 繁体中文（台湾）-> 台湾
		"zh-HK": "HK", // 繁体中文（香港）-> 香港
		"en":    "US", // 英语 -> 美国（默认）
		"en-US": "US", // 美式英语 -> 美国
		"en-GB": "GB", // 英式英语 -> 英国
		"ja":    "JP", // 日语 -> 日本
		"ko":    "KR", // 韩语 -> 韩国
		"de":    "DE", // 德语 -> 德国
		"fr":    "FR", // 法语 -> 法国
		"es":    "ES", // 西班牙语 -> 西班牙
		"it":    "IT", // 意大利语 -> 意大利
		"pt":    "BR", // 葡萄牙语 -> 巴西
		"ru":    "RU", // 俄语 -> 俄罗斯
		"ar":    "SA", // 阿拉伯语 -> 沙特阿拉伯
		"hi":    "IN", // 印地语 -> 印度
		"th":    "TH", // 泰语 -> 泰国
		"vi":    "VN", // 越南语 -> 越南
		"id":    "ID", // 印尼语 -> 印度尼西亚
		"ms":    "MY", // 马来语 -> 马来西亚
		"tr":    "TR", // 土耳其语 -> 土耳其
		"pl":    "PL", // 波兰语 -> 波兰
		"nl":    "NL", // 荷兰语 -> 荷兰
		"sv":    "SE", // 瑞典语 -> 瑞典
		"da":    "DK", // 丹麦语 -> 丹麦
		"no":    "NO", // 挪威语 -> 挪威
		"fi":    "FI", // 芬兰语 -> 芬兰
		"cs":    "CZ", // 捷克语 -> 捷克
		"hu":    "HU", // 匈牙利语 -> 匈牙利
		"ro":    "RO", // 罗马尼亚语 -> 罗马尼亚
		"el":    "GR", // 希腊语 -> 希腊
		"he":    "IL", // 希伯来语 -> 以色列
		"is":    "IS", // 冰岛语 -> 冰岛
	}

	// 处理 Accept-Language 格式：可能包含多个语言，如 "zh-CN,zh;q=0.9,en;q=0.8"
	langs := strings.Split(langStr, ",")
	if len(langs) > 0 {
		// 取第一个语言（优先级最高）
		firstLang := strings.TrimSpace(langs[0])
		// 移除权重信息，如 "zh-CN;q=0.9" -> "zh-CN"
		if idx := strings.Index(firstLang, ";"); idx > 0 {
			firstLang = firstLang[:idx]
		}
		firstLang = strings.TrimSpace(firstLang)

		// 精确匹配
		if country, exists := langToCountry[firstLang]; exists {
			return country
		}

		// 提取基础语言代码（如 "zh-CN" -> "zh"）
		if idx := strings.Index(firstLang, "-"); idx > 0 {
			baseLang := firstLang[:idx]
			if country, exists := langToCountry[baseLang]; exists {
				return country
			}
		}
	}

	return ""
}
