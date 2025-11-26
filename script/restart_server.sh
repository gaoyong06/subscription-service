#!/bin/bash

# 设置颜色
GREEN="\033[0;32m"
YELLOW="\033[1;33m"
RED="\033[0;31m"
NC="\033[0m" # 恢复默认颜色

echo -e "${GREEN}============================================${NC}"
echo -e "${GREEN}   Subscription Service Restart (All)      ${NC}"
echo -e "${GREEN}============================================${NC}"

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

# 停止可能运行的 cron 服务
echo -e "${YELLOW}检查并停止 cron 服务...${NC}"
CRON_PID=$(pgrep -f "bin/cron")
if [ -n "$CRON_PID" ]; then
    echo -e "${YELLOW}停止 cron 服务 (PID: $CRON_PID)...${NC}"
    kill -9 $CRON_PID
    sleep 1
    echo -e "${GREEN}cron 服务已停止${NC}"
else
    echo -e "${GREEN}cron 服务未运行${NC}"
fi

# 切换到项目根目录
cd "$(dirname "$0")/../" || exit

# 生成proto文件
echo -e "${YELLOW}正在生成proto文件...${NC}"
make api

# 生成swagger文档
echo -e "${YELLOW}正在生成swagger文档...${NC}"
make swagger

# 编译所有服务
echo -e "${YELLOW}正在编译所有服务...${NC}"
make build-all

# 启动 cron 服务（后台运行）
echo -e "${YELLOW}正在启动 cron 服务...${NC}"
nohup ./bin/cron -conf ./configs/config.yaml > logs/cron.log 2>&1 &
CRON_PID=$!
sleep 1

# 检查 cron 服务是否启动成功
if ps -p $CRON_PID > /dev/null; then
    echo -e "${GREEN}cron 服务已启动，PID: $CRON_PID${NC}"
else
    echo -e "${RED}cron 服务启动失败!${NC}"
fi

# 启动主服务器（前台运行）
echo -e "${YELLOW}正在启动主服务器...${NC}"
echo -e "${GREEN}============================================${NC}"
./bin/server -conf ./configs/config.yaml

# 如果主服务器退出，也停止 cron 服务
echo -e "${YELLOW}主服务器已停止，正在停止 cron 服务...${NC}"
if ps -p $CRON_PID > /dev/null; then
    kill $CRON_PID
    echo -e "${GREEN}cron 服务已停止${NC}"
fi
