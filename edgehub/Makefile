.PHONY: help build test lint clean deploy docker-build docker-push

REGISTRY ?= registry.edgehub.io
VERSION ?= $(shell git describe --tags --always --dirty)
IMAGE := $(REGISTRY)/edgehub
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

clean: ## Clean build artifacts
	rm -rf build/
	rm -rf bin/

build: clean ## Build all binaries
	@mkdir -p build
	CGO_ENABLED=0 go build $(LDFLAGS) -o build/api-server ./cmd/api-server
	CGO_ENABLED=0 go build $(LDFLAGS) -o build/scheduler ./cmd/scheduler
	CGO_ENABLED=0 go build $(LDFLAGS) -o build/node-agent ./cmd/node-agent
	CGO_ENABLED=0 go build $(LDFLAGS) -o build/edge-cli ./cmd/cli

test: ## Run unit tests
	go test -v -race -coverprofile=coverage.out ./...

test-integration: ## Run integration tests
	go test -v -tags=integration ./...

lint: ## Run linters
	golangci-lint run ./...

fmt: ## Format code
	go fmt ./...
	gofmt -s -w .

mod: ## Tidies go modules
	go mod tidy
	go mod verify

docker-build: ## Build Docker images
	docker build --target api-server -t $(IMAGE)/api-server:$(VERSION) .
	docker build --target scheduler -t $(IMAGE)/scheduler:$(VERSION) .
	docker build --target node-agent -t $(IMAGE)/node-agent:$(VERSION) .
	docker build --target cli -t $(IMAGE)/cli:$(VERSION) .
	docker tag $(IMAGE)/api-server:$(VERSION) $(IMAGE)/api-server:latest
	docker tag $(IMAGE)/scheduler:$(VERSION) $(IMAGE)/scheduler:latest
	docker tag $(IMAGE)/node-agent:$(VERSION) $(IMAGE)/node-agent:latest
	docker tag $(IMAGE)/cli:$(VERSION) $(IMAGE)/cli:latest

docker-push: docker-build ## Push Docker images
	docker push $(IMAGE)/api-server:$(VERSION)
	docker push $(IMAGE)/scheduler:$(VERSION)
	docker push $(IMAGE)/node-agent:$(VERSION)
	docker push $(IMAGE)/cli:$(VERSION)
	docker push $(IMAGE)/api-server:latest
	docker push $(IMAGE)/scheduler:latest
	docker push $(IMAGE)/node-agent:latest
	docker push $(IMAGE)/cli:latest

deploy: ## Deploy to Kubernetes
	kubectl apply -f config/manifests/00-namespace.yaml
	kubectl apply -f config/manifests/01-config.yaml
	kubectl apply -f config/manifests/02-deployment.yaml
	kubectl apply -f config/manifests/03-storage.yaml
	kubectl apply -f config/manifests/04-ingress.yaml

deploy-clean: ## Clean deployment from Kubernetes
	kubectl delete -f config/manifests/

logs: ## Tail logs from Kubernetes deployment
	kubectl logs -n edgehub-system -l app=edgehub-api -f

rollout-restart: ## Restart all deployments
	kubectl rollout restart deployment/edgehub-api -n edgehub-system
	kubectl rollout restart deployment/edgehub-scheduler -n edgehub-system

rollout-status: ## Check rollout status
	kubectl rollout status deployment/edgehub-api -n edgehub-system
	kubectl rollout status deployment/edgehub-scheduler -n edgehub-system

gen: ## Generate code (protobuf, etc.)
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative pkg/api/*.proto

coverage: test ## Generate coverage report
	go tool cover -html=coverage.out -o coverage.html

cross-build: ## Build for multiple platforms
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o build/api-server-linux-amd64 ./cmd/api-server
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o build/api-server-linux-arm64 ./cmd/api-server
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o build/api-server-darwin-amd64 ./cmd/api-server
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o build/api-server-darwin-arm64 ./cmd/api-server

install-cli: build ## Install CLI to /usr/local/bin
	cp build/edge-cli /usr/local/bin/edge

dev: ## Run in development mode
	go run ./cmd/api-server

api-docs: ## Generate API documentation
	swagger generate spec -o ./docs/swagger.json
	swagger serve ./docs/swagger.json

benchmark: ## Run benchmarks
	go test -bench=. -benchmem ./...

stress: ## Run stress tests
	go test -v -run=^$$ -bench=. -benchtime=10s ./...

version: ## Show version info
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
