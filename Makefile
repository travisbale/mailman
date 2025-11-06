.PHONY: help dev build proto clean test lint fmt

# Build production binary
build:
	@echo "Building production binary..."
	@CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/mailman ./cmd/mailman

# Build development binary (faster, includes debug symbols)
dev:
	@echo "Building development binary..."
	@go build -o bin/mailman ./cmd/mailman

# Run tests
test:
	@echo "Running tests..."
	@go test -race -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -rf internal/pb/*.pb.go

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@go run golang.org/x/tools/cmd/goimports@v0.38.0 -w $(shell \
		find . -type f -name '*.go' \
		-not -path './internal/pb/*' \
		-not -path './internal/db/*' )

# Lint code
lint:
	@echo "Linting code..."
	@docker run -t --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:v2.6.0 golangci-lint run

# Generate sqlc code (uses version from go.mod)
sqlc:
	@echo "Generating sqlc code..."
	@docker run --rm -v $(shell pwd):/src -w /src sqlc/sqlc generate

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

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Run database migrations up
migrate-up:
	@./bin/authsvc migrate up

# Run database migrations down
migrate-down:
	@./bin/authsvc migrate down

# Docker targets
docker-build:
	@echo "Building Docker image..."
	@docker build -t authsvc:dev .


# Default target
help:
	@echo "Available targets:"
	@echo "  make dev       - Build development binary with debug symbols"
	@echo "  make build     - Build production binary (optimized)"
	@echo "  make protoc    - Generate protobuf and gRPC code"
	@echo "  make test      - Run tests with race detector"
	@echo "  make lint      - Run golangci-lint"
	@echo "  make fmt       - Format code with gofmt"
	@echo "  make clean     - Remove build artifacts"

