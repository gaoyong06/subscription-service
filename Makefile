# Subscription Service Makefile
# 使用 devops-tools 的通用 Makefile

SERVICE_NAME=subscription-service
API_PROTO_PATH=api/subscription/v1/subscription.proto
API_PROTO_DIR=api/subscription/v1

# 服务特定配置
SERVICE_DISPLAY_NAME=Subscription Service
HTTP_PORT=8102
TEST_CONFIG=test/api/api-test-config.yaml
WIRE_DIRS=cmd/server cmd/cron

# 引入通用 Makefile
DEVOPS_TOOLS_DIR := $(shell cd .. && pwd)/devops-tools
include $(DEVOPS_TOOLS_DIR)/Makefile.common

# 服务特定的目标

.PHONY: build-cron
# 构建 cron 服务
build-cron:
	mkdir -p bin/
	go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/cron ./cmd/cron

.PHONY: build-all
# 构建所有服务
build-all: build build-cron

.PHONY: run-cron
# 运行 cron 服务
run-cron:
	./bin/cron -conf ./configs/config.yaml

.PHONY: run-all
# 同时运行所有服务（cron 后台，server 前台）
run-all:
	@echo "启动 cron 服务（后台）..."
	@mkdir -p logs
	@nohup ./bin/cron -conf ./configs/config.yaml > logs/cron.log 2>&1 & echo $$! > logs/cron.pid
	@sleep 1
	@if [ -f logs/cron.pid ]; then \
		CRON_PID=$$(cat logs/cron.pid); \
		if ps -p $$CRON_PID > /dev/null; then \
			echo "cron 服务已启动，PID: $$CRON_PID"; \
		else \
			echo "cron 服务启动失败!"; \
		fi \
	fi
	@echo "启动主服务（前台）..."
	@echo "========================================="
	@./bin/server -conf ./configs/config.yaml; \
	if [ -f logs/cron.pid ]; then \
		CRON_PID=$$(cat logs/cron.pid); \
		if ps -p $$CRON_PID > /dev/null; then \
			echo "停止 cron 服务..."; \
			kill $$CRON_PID; \
		fi; \
		rm -f logs/cron.pid; \
	fi

.PHONY: stop-all
# 停止所有服务
stop-all:
	@echo "停止所有服务..."
	@-pkill -f "bin/server" || true
	@-pkill -f "bin/cron" || true
	@-rm -f logs/cron.pid
	@echo "所有服务已停止"

.PHONY: all
# 生成所有代码并构建（覆盖通用版本）
all: api wire build-all

# 覆盖 help 目标（添加服务特定的目标）
help:
	@echo "$(SERVICE_DISPLAY_NAME) Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make init         - 安装所需工具"
	@echo "  make api          - 生成 API 代码"
	@echo "  make swagger      - 生成 Swagger 文档"
	@echo "  make wire         - 生成依赖注入代码（server + cron）"
	@echo "  make build        - 编译主服务"
	@echo "  make build-cron   - 编译 cron 服务"
	@echo "  make build-all    - 编译所有服务"
	@echo "  make run          - 运行主服务（前台）"
	@echo "  make run-cron     - 运行 cron 服务（前台）"
	@echo "  make run-all      - 运行所有服务（cron 后台 + server 前台）"
	@echo "  make restart      - 重启服务（使用 devops-tools）"
	@echo "  make stop-all     - 停止所有服务"
	@echo "  make test         - 运行 API 测试"
	@echo "  make clean        - 清理生成的文件"
	@echo "  make docker-build - 构建 Docker 镜像"
	@echo "  make docker-run   - 运行 Docker 容器（需要设置 DOCKER_PORTS）"
	@echo "  make all          - 生成代码并构建所有服务"
