.PHONY: build test clean run-terminal run-server deploy-contracts generate-bindings

# Go build settings
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
TERMINAL_BINARY=bin/terminal
SERVER_BINARY=bin/server
ADMIN_BINARY=bin/admin

# Build all applications
build: clean
	@echo "🔨 Building applications..."
	@mkdir -p bin
	$(GOBUILD) -o $(TERMINAL_BINARY) cmd/terminal/main.go
	$(GOBUILD) -o $(SERVER_BINARY) cmd/server/main.go
	$(GOBUILD) -o $(ADMIN_BINARY) cmd/admin/main.go
	@echo "✅ Build completed"

# Build for Linux (useful for deployment)
build-linux:
	@echo "🔨 Building for Linux..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(TERMINAL_BINARY)-linux cmd/terminal/main.go
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(SERVER_BINARY)-linux cmd/server/main.go
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(ADMIN_BINARY)-linux cmd/admin/main.go
	@echo "✅ Linux build completed"

# Run tests
test:
	@echo "🧪 Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "🧪 Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf bin/
	rm -rf build/
	rm -f coverage.out coverage.html
	rm -f CONTRACT_INFO.md

# Run terminal application
run-terminal:
	@echo "🖥️  Starting voting terminal..."
	$(GOCMD) run cmd/terminal/main.go

# Run server application
run-server:
	@echo "🖥️  Starting central server..."
	$(GOCMD) run cmd/server/main.go

# Run admin application
run-admin:
	@echo "🖥️  Starting admin dashboard..."
	$(GOCMD) run cmd/admin/main.go

# Blockchain commands

# Start local blockchain (Ganache)
start-blockchain:
	@echo "⛓️  Starting local blockchain..."
	@pkill -f ganache-cli || true
	@sleep 2
	ganache-cli \
		--deterministic \
		--accounts 10 \
		--host 0.0.0.0 \
		--port 8545 \
		--gasLimit 10000000 \
		--gasPrice 20000000000 \
		--mnemonic "candy maple cake sugar pudding cream honey rich smooth crumble sweet treat" \
		--networkId 1337 \
		--verbose

# Stop local blockchain
stop-blockchain:
	@echo "⛓️  Stopping local blockchain..."
	@pkill -f ganache-cli || true

# Compile smart contracts
compile-contracts:
	@echo "📜 Compiling smart contracts..."
	truffle compile
	@echo "✅ Contracts compiled successfully"

# Deploy smart contracts
deploy-contracts: compile-contracts
	@echo "🚀 Deploying smart contracts..."
	truffle migrate --reset --network development
	@echo "✅ Contracts deployed successfully"

# Generate Go bindings from smart contracts
generate-bindings: compile-contracts
	@echo "🔗 Generating Go bindings..."
	@chmod +x scripts/generate-bindings.sh
	./scripts/generate-bindings.sh

# Full blockchain setup (compile, deploy, generate bindings)
setup-blockchain: compile-contracts deploy-contracts generate-bindings
	@echo "✅ Blockchain setup completed"

# Verify contract deployment
verify-contracts:
	@echo "🔍 Verifying contract deployment..."
	truffle exec scripts/verify-deployment.js

# Reset blockchain (clean deploy)
reset-blockchain: stop-blockchain
	@echo "🔄 Resetting blockchain..."
	@sleep 2
	@make start-blockchain &
	@sleep 5
	@make deploy-contracts

# Development commands

# Setup development environment
setup-dev:
	@echo "🛠️  Setting up development environment..."
	$(GOMOD) tidy
	@mkdir -p logs bin contracts/bindings
	@chmod +x scripts/*.sh
	@echo "✅ Development environment ready"

# Install development tools
install-tools:
	@echo "🔧 Installing development tools..."
	@npm install -g truffle ganache-cli
	@go install github.com/ethereum/go-ethereum/cmd/abigen@latest
	@echo "✅ Development tools installed"

# Lint Go code
lint:
	@echo "🔍 Linting Go code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Format Go code
fmt:
	@echo "💅 Formatting Go code..."
	$(GOCMD) fmt ./...

# Update dependencies
update-deps:
	@echo "📦 Updating dependencies..."
	$(GOMOD) tidy
	$(GOMOD) download

# Database commands

# Initialize local database
init-db:
	@echo "🗄️  Initializing local database..."
	@if [ -f "scripts/migrate.go" ]; then \
		$(GOCMD) run scripts/migrate.go; \
	else \
		echo "Migration script not found"; \
	fi

# Reset local database
reset-db:
	@echo "🗄️  Resetting local database..."
	rm -f terminal.db
	@make init-db

# Docker commands

# Build Docker images
docker-build:
	@echo "🐳 Building Docker images..."
	docker build -f docker/Dockerfile.terminal -t voting-terminal .
	docker build -f docker/Dockerfile.server -t voting-server .

# Run with Docker Compose
docker-up:
	@echo "🐳 Starting services with Docker Compose..."
	docker-compose -f docker/docker-compose.yml up -d

# Stop Docker services
docker-down:
	@echo "🐳 Stopping Docker services..."
	docker-compose -f docker/docker-compose.yml down

# Monitoring commands

# Check system status
status:
	@echo "📊 System Status:"
	@echo "==================="
	@echo "Go version: $(shell go version)"
	@echo "Node version: $(shell node --version 2>/dev/null || echo 'Not installed')"
	@echo "Truffle version: $(shell truffle version 2>/dev/null || echo 'Not installed')"
	@echo ""
	@echo "📁 Project structure:"
	@ls -la
	@echo ""
	@echo "🔗 Blockchain status:"
	@curl -s http://localhost:8545 >/dev/null && echo "✅ Ganache running" || echo "❌ Ganache not running"

# Show logs
logs-terminal:
	@echo "📋 Terminal logs:"
	@tail -f logs/terminal.log 2>/dev/null || echo "No terminal logs found"

logs-server:
	@echo "📋 Server logs:"
	@tail -f logs/server.log 2>/dev/null || echo "No server logs found"

# Development workflow commands

# Full development setup
dev-setup: install-tools setup-dev start-blockchain setup-blockchain
	@echo "🎉 Development environment is ready!"
	@echo ""
	@echo "Next steps:"
	@echo "1. Run terminal: make run-terminal"
	@echo "2. Run server: make run-server"
	@echo "3. Check status: make status"

# Quick development start
dev: 
	@echo "🚀 Starting development environment..."
	@make start-blockchain &
	@sleep 5
	@make setup-blockchain
	@echo "✅ Ready for development"

# Production deployment preparation
prod-prep: test lint build-linux
	@echo "🏭 Production deployment package prepared"
	@echo "Binaries available in bin/ directory"

# Help command
help:
	@echo "📚 Available commands:"
	@echo ""
	@echo "🔨 Build commands:"
	@echo "  build              - Build all applications"
	@echo "  build-linux        - Build for Linux deployment"
	@echo "  clean              - Clean build artifacts"
	@echo ""
	@echo "🧪 Test commands:"
	@echo "  test               - Run tests"
	@echo "  test-coverage      - Run tests with coverage"
	@echo "  lint               - Lint Go code"
	@echo "  fmt                - Format Go code"
	@echo ""
	@echo "🖥️  Run commands:"
	@echo "  run-terminal       - Run voting terminal"
	@echo "  run-server         - Run central server"
	@echo "  run-admin          - Run admin dashboard"
	@echo ""
	@echo "⛓️  Blockchain commands:"
	@echo "  start-blockchain   - Start local blockchain"
	@echo "  stop-blockchain    - Stop local blockchain"
	@echo "  compile-contracts  - Compile smart contracts"
	@echo "  deploy-contracts   - Deploy smart contracts"
	@echo "  generate-bindings  - Generate Go bindings"
	@echo "  setup-blockchain   - Full blockchain setup"
	@echo "  reset-blockchain   - Reset and redeploy"
	@echo ""
	@echo "🛠️  Development commands:"
	@echo "  setup-dev          - Setup development environment"
	@echo "  dev-setup          - Full development setup"
	@echo "  dev                - Quick development start"
	@echo "  status             - Check system status"
	@echo ""
	@echo "🐳 Docker commands:"
	@echo "  docker-build       - Build Docker images"
	@echo "  docker-up          - Start with Docker Compose"
	@echo "  docker-down        - Stop Docker services"

# Default target
.DEFAULT_GOAL := help