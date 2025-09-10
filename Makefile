BINARY_NAME=nclip
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS = -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"

.PHONY: all build clean test run dev install check fmt-check help

all: build

help: ## Show this help message
	@echo "nclip - Modern netcat-to-clipboard service"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	@echo "🔨 Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/server
	@echo "✅ Build complete"

clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	rm -f $(BINARY_NAME) coverage.out coverage.html
	go clean
	@echo "✅ Clean complete"

test: ## Run tests
	@echo "🧪 Running tests..."
	go test -v ./...
	@echo "✅ Tests complete"

test-coverage: ## Run tests with coverage
	@echo "🧪 Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

run: build ## Build and run the application
	@echo "🚀 Starting nclip..."
	./$(BINARY_NAME)

dev: ## Run in development mode
	@echo "🚀 Starting nclip in development mode..."
	go run $(LDFLAGS) ./cmd/server -log-level debug

install: build ## Install to /usr/local/bin
	@echo "📦 Installing $(BINARY_NAME)..."
	sudo cp $(BINARY_NAME) /usr/local/bin/
	@echo "✅ Installed to /usr/local/bin/$(BINARY_NAME)"

# Code quality targets
format: ## Format code
	@echo "📝 Formatting code..."
	go fmt ./...
	@echo "✅ Code formatted"

fmt-check: ## Check if code is formatted
	@echo "🔍 Checking code formatting..."
	@if [ "$$(gofmt -s -l . | wc -l)" -gt 0 ]; then \
		echo "❌ Code is not formatted. Run 'make format'"; \
		gofmt -s -l .; \
		exit 1; \
	fi
	@echo "✅ Code is properly formatted"

vet: ## Run go vet
	@echo "🔍 Running go vet..."
	go vet ./...
	@echo "✅ go vet passed"

lint: ## Run golangci-lint
	@echo "🔍 Running golangci-lint..."
	golangci-lint run
	@echo "✅ Linting passed"

check: fmt-check vet lint test ## Run all code quality checks
	@echo "✅ All checks passed!"

# Dependency management
mod-tidy: ## Tidy go modules
	go mod tidy

mod-download: ## Download dependencies
	go mod download

# Build for multiple platforms
build-all: ## Build for all supported platforms
	@echo "🔨 Building for all platforms..."
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/server
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/server
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/server
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/server
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/server
	@echo "✅ Multi-platform build complete"

# Docker targets
docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t nclip:$(VERSION) \
		-t nclip:latest .
	@echo "✅ Docker image built"

docker-run: docker-build ## Build and run Docker container
	@echo "🐳 Running Docker container..."
	docker run --rm -p 8080:8080 -p 9999:9999 nclip:latest

# Release preparation
release-check: clean check build-all ## Run all checks for release
	@echo "🎯 Release preparation complete!"

version: ## Show version information
	@echo "nclip version: $(VERSION)"
	@echo "Build time: $(BUILD_TIME)"
	@echo "Git commit: $(GIT_COMMIT)"
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 ./cmd/server
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 ./cmd/server
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 ./cmd/server
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe ./cmd/server

# Docker targets
docker-build:
	docker build -t nclip:$(VERSION) .

docker-run:
	docker run -p 9999:9999 -p 8080:8080 -v $(PWD)/data:/data nclip:$(VERSION)

# Development setup
dev-setup:
	go mod download
	@echo "Development environment ready!"
	@echo "Run 'make dev' to start the server"
	@echo "Run 'make test' to run tests"
