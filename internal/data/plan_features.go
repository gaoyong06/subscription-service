package data

import (
	"encoding/json"
	"strings"
)

// encodePlanFeaturesJSON 将 i18n key 列表序列化为 JSON 数组存入数据库
func encodePlanFeaturesJSON(keys []string) string {
	if len(keys) == 0 {
		return "[]"
	}
	b, err := json.Marshal(keys)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// decodePlanFeaturesJSON 从数据库 JSON 列解析为 key 列表
func decodePlanFeaturesJSON(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var keys []string
	if err := json.Unmarshal([]byte(s), &keys); err != nil {
		return nil
	}
	return keys
}
