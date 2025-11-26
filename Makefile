GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always)
SERVICE_NAME=subscription-service

.PHONY: init
# 初始化项目依赖
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	go install github.com/google/wire/cmd/wire@latest

.PHONY: api
# 生成 API 代码
api:
	kratos proto client api/subscription/v1/subscription.proto
	protoc \
	  --proto_path=api/subscription/v1 \
	  --proto_path=$(shell go env GOPATH)/pkg/mod \
	  --proto_path=$(shell go env GOPATH)/pkg/mod/github.com/go-kratos/kratos/v2@v2.9.1/third_party \
	  --proto_path=$(shell go list -m -f '{{.Dir}}' github.com/grpc-ecosystem/grpc-gateway)/third_party/googleapis \
	  --go_out=paths=source_relative:. \
	  --go-http_out=paths=source_relative:. \
	  --go-grpc_out=paths=source_relative:. \
	  --validate_out=paths=source_relative,lang=go:. \
	  api/subscription/v1/subscription.proto

.PHONY: wire
# 生成依赖注入代码
wire:
	cd cmd/server && wire
	cd cmd/cron && wire

.PHONY: build
# 构建项目
build:
	mkdir -p bin/ && go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/server ./cmd/server

.PHONY: build-cron
# 构建 cron 服务
build-cron:
	mkdir -p bin/ && go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/cron ./cmd/cron

.PHONY: build-all
# 构建所有服务
build-all: build build-cron

.PHONY: run
# 运行服务
run:
	go run cmd/server/main.go cmd/server/wire_gen.go -conf configs/config.yaml

.PHONY: run-cron
# 运行 cron 服务
run-cron:
	./bin/cron -conf ./configs/config.yaml

.PHONY: test
# 运行 API 测试
test:
	@echo "========================================="
	@echo "  Testing Subscription Service"
	@echo "========================================="
	@echo "检查服务状态..."
	@curl -s http://localhost:8102/health || echo "Subscription Service 启动中..."
	@echo "\nRunning API tests..."
	../api-tester/bin/api-tester run --config api-test-config.yaml

.PHONY: clean
# 清理生成的文件
clean:
	rm -rf bin/
	rm -f api/subscription/v1/*.pb.go
	rm -f cmd/server/wire_gen.go
	rm -f cmd/cron/wire_gen.go
	rm -rf test-reports/

.PHONY: docker-build
# 构建 Docker 镜像
docker-build:
	docker build -t $(SERVICE_NAME):$(VERSION) .

.PHONY: docker-run
# 运行 Docker 容器
docker-run:
	docker run -p 8102:8102 -p 9102:9102 $(SERVICE_NAME):$(VERSION)

.PHONY: all
# 生成所有代码并构建
all: api wire build-all

.PHONY: help
# 显示帮助信息
help:
	@echo "Subscription Service Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make init         - 安装所需工具"
	@echo "  make api          - 生成 API 代码"
	@echo "  make wire         - 生成依赖注入代码"
	@echo "  make build        - 编译主服务"
	@echo "  make build-cron   - 编译 cron 服务"
	@echo "  make build-all    - 编译所有服务"
	@echo "  make run          - 运行主服务"
	@echo "  make run-cron     - 运行 cron 服务"
	@echo "  make test         - 运行 API 测试"
	@echo "  make clean        - 清理生成的文件"
	@echo "  make docker-build - 构建 Docker 镜像"
	@echo "  make docker-run   - 运行 Docker 容器"
	@echo "  make all          - 生成代码并构建所有服务"
