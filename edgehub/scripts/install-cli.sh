.PHONY: help build test lint clean install run

help:
	@echo "EdgeHub CLI - Command Line Interface"
	@echo ""
	@echo "Usage:"
	@echo "  edge-cli node list                    List all nodes"
	@echo "  edge-cli node get <node-id>          Get node details"
	@echo "  edge-cli node register               Register a new node"
	@echo "  edge-cli job list                    List all jobs"
	@echo "  edge-cli job submit <file>           Submit a job"
	@echo "  edge-cli job logs <job-id>           Get job logs"
	@echo "  edge-cli market list-offers          List market offers"
	@echo ""
	@echo "Options:"
	@echo "  --server <addr>      API server address (default: http://localhost:8080)"
	@echo "  --api-key <key>      API key for authentication"
	@echo "  --json, -j           Output in JSON format"
	@echo "  --help, -h           Show this help message"
	@echo "  --version, -v        Show version"

version:
	@echo "EdgeHub CLI v1.0.0"

install: build
	cp build/edge-cli /usr/local/bin/edge
	@echo "Installed to /usr/local/bin/edge"

build:
	cd /workspace/edgehub && make build

run:
	cd /workspace/edgehub && go run ./cmd/cli node list --server http://localhost:8080
