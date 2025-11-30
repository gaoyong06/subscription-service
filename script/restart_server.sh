#!/bin/bash

# Subscription Service 重启脚本
# 调用 devops-tools 统一管理

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_DIR="$(cd "$SCRIPT_DIR/../" && pwd)"
PROJECT_ROOT="$(cd "$SERVICE_DIR/../" && pwd)"

# devops-tools 目录
DEVOPS_TOOLS_DIR="$PROJECT_ROOT/devops-tools"

# 检查 devops-tools 是否存在
if [ ! -d "$DEVOPS_TOOLS_DIR" ] || [ ! -f "$DEVOPS_TOOLS_DIR/Makefile" ]; then
    echo "错误: 找不到 devops-tools，请先安装:"
    echo "  git clone https://github.com/gaoyong06/devops-tools.git $DEVOPS_TOOLS_DIR"
    exit 1
fi

# 调用 devops-tools 统一管理
cd "$DEVOPS_TOOLS_DIR"
make restart SERVICE=subscription-service PROJECT_ROOT="$PROJECT_ROOT"
