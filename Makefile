## Build and Development commands for QUIC Reverse Proxy

.PHONY: build run test clean deps docker help dev

# Variables
BINARY_NAME=quic-proxy
MAIN_PATH=./cmd/proxy
BUILD_DIR=./build
GO_VERSION=1.21
DOCKER_IMAGE=quic-reverse-proxy
VERSION?=latest

help: ## Display this help message
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<command>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

deps: ## Download and tidy Go modules
	@echo "📦 Installing dependencies..."
	go mod download
	go mod tidy
	go mod verify

build: deps ## Build the binary
	@echo "🔨 Building $(BINARY_NAME)..."
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s -X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-windows: deps ## Build Windows binary
	@echo "🔨 Building $(BINARY_NAME) for Windows..."
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-w -s -X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME).exe $(MAIN_PATH)
	@echo "✅ Windows build complete: $(BUILD_DIR)/$(BINARY_NAME).exe"

build-all: ## Build for all platforms
	@echo "🔨 Building for all platforms..."
	mkdir -p $(BUILD_DIR)
	# Linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s -X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	# Windows
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-w -s -X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	# macOS
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-w -s -X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	@echo "✅ All builds complete"

run: ## Run the application locally
	@echo "🚀 Running $(BINARY_NAME)..."
	go run $(MAIN_PATH) -config configs/proxy.yaml -debug

dev: ## Run in development mode with hot reload
	@echo "🔥 Starting development mode..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "⚠️  Air not found. Installing..."; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

##@ Testing

test: ## Run tests
	@echo "🧪 Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests with coverage report
	@echo "📊 Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

benchmark: ## Run benchmarks
	@echo "⚡ Running benchmarks..."
	go test -bench=. -benchmem ./...

##@ Code Quality

lint: ## Run linting
	@echo "🔍 Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint not found. Installing..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2; \
		golangci-lint run; \
	fi

format: ## Format code
	@echo "🎨 Formatting code..."
	go fmt ./...
	goimports -w .

vet: ## Run go vet
	@echo "🔍 Running go vet..."
	go vet ./...

##@ Certificates

generate-certs: ## Generate self-signed certificates for testing
	@echo "🔐 Generating self-signed certificates..."
	mkdir -p certs
	openssl req -x509 -newkey rsa:2048 -keyout certs/server.key -out certs/server.crt -days 365 -nodes \
		-subj "/C=US/ST=CA/L=San Francisco/O=QUIC Proxy/OU=Development/CN=localhost" \
		-addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
	@echo "✅ Certificates generated in ./certs/"

##@ Docker

docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(VERSION) -f deployments/docker/Dockerfile .
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest
	@echo "✅ Docker image built: $(DOCKER_IMAGE):$(VERSION)"

docker-run: ## Run Docker container
	@echo "🐳 Running Docker container..."
	docker run -p 443:443 -p 9090:9090 \
		-v $$(pwd)/configs:/app/configs \
		-v $$(pwd)/certs:/app/certs \
		$(DOCKER_IMAGE):$(VERSION)

docker-compose-up: ## Start services with docker-compose
	@echo "🐳 Starting services with docker-compose..."
	docker-compose up -d

docker-compose-down: ## Stop services with docker-compose
	@echo "🐳 Stopping services with docker-compose..."
	docker-compose down

##@ Utilities

clean: ## Clean build artifacts
	@echo "🧹 Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	go clean -testcache
	@echo "✅ Cleanup complete"

install: build ## Install binary to system
	@echo "📦 Installing $(BINARY_NAME) to system..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "✅ $(BINARY_NAME) installed to /usr/local/bin/"

uninstall: ## Uninstall binary from system
	@echo "🗑️  Uninstalling $(BINARY_NAME)..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "✅ $(BINARY_NAME) uninstalled"

version: ## Show version information
	@echo "QUIC Reverse Proxy v$(VERSION)"
	@echo "Go version: $$(go version)"

init-project: deps generate-certs ## Initialize project (install deps, generate certs)
	@echo "🎉 Project initialized successfully!"
	@echo "📖 Next steps:"
	@echo "   1. Review configs/proxy.yaml"
	@echo "   2. Start backend services on ports 8080, 8081, 3000, 3001"
	@echo "   3. Run 'make run' to start the proxy"

##@ CI/CD

ci-test: deps lint test ## Run CI pipeline tests
	@echo "✅ CI pipeline completed"

release: clean build-all ## Create release artifacts
	@echo "📦 Creating release artifacts..."
	cd $(BUILD_DIR) && tar -czf $(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	cd $(BUILD_DIR) && zip $(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	cd $(BUILD_DIR) && tar -czf $(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	@echo "✅ Release artifacts created in $(BUILD_DIR)/"