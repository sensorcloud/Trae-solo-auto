.PHONY: all build run test lint clean deploy

# 项目名称
APP_NAME := edgeagent-hub

# Go编译参数
GOFLAGS := -v

# 主入口文件
MAIN_FILE := ./cmd/edgeagent/main.go

# 构建目标
BUILD_DIR := ./build

all: build

build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(GOFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)

run:
	@echo "Running $(APP_NAME)..."
	go run $(MAIN_FILE)

test:
	@echo "Running tests..."
	go test ./internal/... ./pkg/...

test-unit:
	@echo "Running unit tests..."
	go test ./test/unit/...

test-integration:
	@echo "Running integration tests..."
	go test ./test/integration/...

lint:
	@echo "Running linter..."
	golangci-lint run ./...

clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)

deploy:
	@echo "Deploying to Kubernetes..."
	helm upgrade --install $(APP_NAME) ./deploy/helm/$(APP_NAME)

deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):latest -f ./deploy/docker/Dockerfile .

docker-push:
	@echo "Pushing Docker image..."
	docker push $(APP_NAME):latest

help:
	@echo "Available targets:"
	@echo "  build          - Build the application"
	@echo "  run            - Run the application"
	@echo "  test           - Run all tests"
	@echo "  test-unit      - Run unit tests"
	@echo "  test-integration - Run integration tests"
	@echo "  lint           - Run linter"
	@echo "  clean          - Clean build artifacts"
	@echo "  deploy         - Deploy to Kubernetes"
	@echo "  deps           - Install dependencies"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-push    - Push Docker image"