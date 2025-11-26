#!/bin/bash

# 测试 Cron 相关的 API

GREEN="\033[0;32m"
YELLOW="\033[1;33m"
RED="\033[0;31m"
NC="\033[0m"

BASE_URL="http://localhost:8102"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}   测试 Cron 相关 API                  ${NC}"
echo -e "${GREEN}========================================${NC}"

# 1. 测试获取即将过期的订阅
echo -e "\n${YELLOW}1. 测试获取即将过期的订阅${NC}"
echo -e "${YELLOW}GET $BASE_URL/v1/subscription/expiring?days_before_expiry=7&page=1&page_size=10${NC}"
curl -s -X GET "$BASE_URL/v1/subscription/expiring?days_before_expiry=7&page=1&page_size=10" | jq '.'

# 2. 测试批量更新过期订阅
echo -e "\n${YELLOW}2. 测试批量更新过期订阅${NC}"
echo -e "${YELLOW}POST $BASE_URL/v1/subscription/expired/update${NC}"
curl -s -X POST "$BASE_URL/v1/subscription/expired/update" \
  -H "Content-Type: application/json" \
  -d '{}' | jq '.'

# 3. 测试自动续费处理（dry run）
echo -e "\n${YELLOW}3. 测试自动续费处理（dry run）${NC}"
echo -e "${YELLOW}POST $BASE_URL/v1/subscription/auto-renew/process${NC}"
curl -s -X POST "$BASE_URL/v1/subscription/auto-renew/process" \
  -H "Content-Type: application/json" \
  -d '{
    "days_before_expiry": 3,
    "dry_run": true
  }' | jq '.'

# 4. 测试获取即将过期的订阅（30天内）
echo -e "\n${YELLOW}4. 测试获取即将过期的订阅（30天内）${NC}"
echo -e "${YELLOW}GET $BASE_URL/v1/subscription/expiring?days_before_expiry=30&page=1&page_size=10${NC}"
curl -s -X GET "$BASE_URL/v1/subscription/expiring?days_before_expiry=30&page=1&page_size=10" | jq '.'

echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}   测试完成                            ${NC}"
echo -e "${GREEN}========================================${NC}"

