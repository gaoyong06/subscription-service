#!/bin/bash

# 设置颜色
GREEN="\033[0;32m"
YELLOW="\033[1;33m"
RED="\033[0;31m"
NC="\033[0m" # 恢复默认颜色

echo -e "${GREEN}====================================${NC}"
echo -e "${GREEN}   Subscription Service Restart    ${NC}"
echo -e "${GREEN}====================================${NC}"

# 要检查的端口列表
PORTS=(8102 9102)

# 检查并释放端口
for PORT in "${PORTS[@]}"; do
    echo -e "${YELLOW}检查端口 $PORT 是否被占用...${NC}"
    
    # 查找占用端口的进程
    PID=$(lsof -ti :$PORT)
    
    if [ -n "$PID" ]; then
        echo -e "${YELLOW}端口 $PORT 被进程 $PID 占用，正在终止...${NC}"
        kill -9 $PID
        sleep 1
        echo -e "${GREEN}端口 $PORT 已释放${NC}"
    else
        echo -e "${GREEN}端口 $PORT 未被占用${NC}"
    fi
done

# 切换到项目根目录
cd "$(dirname "$0")/../" || exit

# 生成proto文件
echo -e "${YELLOW}正在生成proto文件...${NC}"
make api

# 启动主服务器（前台运行）
echo -e "${YELLOW}正在启动主服务器...${NC}"
make run

# 检查服务器是否成功启动
if [ $? -eq 0 ]; then
    echo -e "${GREEN}服务器启动成功!${NC}"
else
    echo -e "${RED}服务器启动失败!${NC}"
    exit 1
fi
