.PHONY: help dev build proto clean test unit integration lint fmt

# Version is derived from git tags
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@go run golang.org/x/tools/cmd/goimports@v0.38.0 -w $(shell \
		find . -type f -name '*.go' \
		-not -path './internal/pb/*' \
		-not -path './internal/db/*' )

# Build development binary (faster, includes debug symbols)
dev: fmt
	@echo "Building development binary..."
	@go build -ldflags="-X 'main.Version=$(VERSION)'" -o bin/mailman ./cmd/mailman
	@echo "Build complete: bin/mailman"

# Build production binary
build: fmt
	@echo "Building production binary..."
	@CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X 'main.Version=$(VERSION)'" -o bin/mailman ./cmd/mailman
	@echo "Build complete: bin/mailman"

# Run unit tests only
unit:
	@echo "Running unit tests..."
	@go test -race -v $$(go list ./... | grep -v '/test')

# Run integration tests (requires Docker)
integration:
	@echo "Running integration tests..."
	@go test -count=1 -timeout 5m -v ./test/...

# Run all tests (unit + integration, requires Docker)
test: unit integration

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -rf internal/pb/*.pb.go

# Lint code
lint:
	@echo "Linting code..."
	@docker run -t --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:v2.11.4 golangci-lint run

# Generate sqlc code (uses version from go.mod)
sqlc:
	@echo "Generating sqlc code..."
	@docker run --rm --user $(shell id -u):$(shell id -g) -v $(shell pwd):/src -w /src sqlc/sqlc generate

# Generate protobuf code
protoc:
	@echo "Generating protobuf code..."
	@mkdir -p internal/pb
	@docker build -q -t mailman-protoc:latest -f proto/Dockerfile . > /dev/null
	@docker run --rm -v $(shell pwd):/workspace --user $(shell id -u):$(shell id -g) \
		-w /workspace \
		mailman-protoc:latest \
		-I proto \
		--go_out=internal/pb --go_opt=paths=source_relative \
		--go-grpc_out=internal/pb --go-grpc_opt=paths=source_relative \
		proto/mailman.proto
	@echo "Protobuf code generated successfully"

# Docker targets
docker-build:
	@echo "Building Docker image..."
	@docker build -t authsvc:dev .


# Default target
help:
	@echo "Available targets:"
	@echo "  make dev         - Build development binary with debug symbols"
	@echo "  make build       - Build production binary (optimized)"
	@echo "  make protoc      - Generate protobuf and gRPC code"
	@echo "  make test        - Run all tests (unit + integration)"
	@echo "  make unit        - Run unit tests only"
	@echo "  make integration - Run integration tests (requires Docker)"
	@echo "  make lint        - Run golangci-lint"
	@echo "  make fmt         - Format code with gofmt"
	@echo "  make clean       - Remove build artifacts"

