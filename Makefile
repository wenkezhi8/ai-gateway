.PHONY: build run test test-safe clean docker-build docker-up docker-down \
        deps tidy fmt lint lint-fix security-scan ci-local \
        frontend-install frontend-build frontend-lint frontend-test \
        help

# ============================================
# Variables
# ============================================
APP_NAME := ai-gateway
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt
GOVET := $(GOCMD) vet

# Directories
WEB_DIR := web
BIN_DIR := bin
COVERAGE_DIR := coverage

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
NC := \033[0m

# ============================================
# Default target
# ============================================
.DEFAULT_GOAL := help

# ============================================
# Build
# ============================================

build: ## Build the application
	@echo "$(BLUE)Building $(APP_NAME) v$(VERSION)...$(NC)"
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME) ./cmd/gateway
	@echo "$(GREEN)Build complete: $(BIN_DIR)/$(APP_NAME)$(NC)"

build-all: build frontend-build ## Build backend and frontend

run: ## Run the application
	$(GOCMD) run ./cmd/gateway

dev: ## Run in development mode with hot reload (requires air)
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	air

# ============================================
# Testing
# ============================================

test: ## Run all tests
	@echo "$(BLUE)Running tests...$(NC)"
	$(GOTEST) -v -race ./...

test-safe: ## Run tests safe for sandbox/limited-port environments
	@echo "$(BLUE)Running sandbox-safe tests...$(NC)"
	$(GOTEST) -v ./scripts -count=1
	$(GOTEST) -v ./internal/docs ./internal/bootstrap -count=1

test-coverage: ## Run tests with coverage
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "$(GREEN)Coverage report: $(COVERAGE_DIR)/coverage.html$(NC)"
	$(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage.out | tail -1

test-unit: ## Run unit tests only
	$(GOTEST) -v -race ./internal/...

test-integration: ## Run integration tests
	$(GOTEST) -v -race ./tests/integration/...

test-benchmark: ## Run benchmark tests
	./scripts/benchmark/benchmark.sh all

test-security: ## Run security tests
	$(GOTEST) -v -race -run Security ./tests/integration/...

# ============================================
# Code Quality
# ============================================

fmt: ## Format code
	@echo "$(BLUE)Formatting Go code...$(NC)"
	$(GOFMT) ./...
	@echo "$(GREEN)Formatting complete$(NC)"

lint: ## Run linters
	@echo "$(BLUE)Running golangci-lint...$(NC)"
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run ./... --timeout=5m

lint-fix: ## Run linters and fix issues
	@echo "$(BLUE)Running golangci-lint with auto-fix...$(NC)"
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run ./... --fix --timeout=5m

vet: ## Run go vet
	$(GOVET) ./...

security-scan: ## Run security scanners
	@echo "$(BLUE)Running security scan...$(NC)"
	@which gosec > /dev/null || go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec ./...
	@echo "$(GREEN)Security scan complete$(NC)"

check: fmt vet lint test ## Run all checks (fmt, vet, lint, test)

# ============================================
# Dependencies
# ============================================

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) verify

tidy: ## Tidy dependencies
	$(GOMOD) tidy

update-deps: ## Update dependencies
	$(GOMOD) get -u
	$(GOMOD) tidy

# ============================================
# Frontend
# ============================================

frontend-install: ## Install frontend dependencies
	@echo "$(BLUE)Installing frontend dependencies...$(NC)"
	cd $(WEB_DIR) && npm ci
	@echo "$(GREEN)Frontend dependencies installed$(NC)"

frontend-build: frontend-install ## Build frontend
	@echo "$(BLUE)Building frontend...$(NC)"
	cd $(WEB_DIR) && npm run build
	@echo "$(GREEN)Frontend build complete$(NC)"

frontend-dev: ## Run frontend development server
	cd $(WEB_DIR) && npm run dev

frontend-lint: ## Lint frontend code
	@echo "$(BLUE)Linting frontend...$(NC)"
	cd $(WEB_DIR) && npm run lint

frontend-lint-fix: ## Fix frontend lint issues
	cd $(WEB_DIR) && npm run lint:fix

frontend-format: ## Format frontend code
	cd $(WEB_DIR) && npm run format

frontend-test: ## Run frontend tests
	cd $(WEB_DIR) && npm run test

frontend-typecheck: ## Run frontend type check
	cd $(WEB_DIR) && npm run typecheck

# ============================================
# Docker
# ============================================

docker-build: ## Build Docker image
	@echo "$(BLUE)Building Docker image $(APP_NAME):$(VERSION)...$(NC)"
	docker build -t $(APP_NAME):$(VERSION) -t $(APP_NAME):latest .

docker-build-no-cache: ## Build Docker image without cache
	docker build --no-cache -t $(APP_NAME):$(VERSION) -t $(APP_NAME):latest .

docker-up: ## Start Docker containers
	docker-compose up -d

docker-down: ## Stop Docker containers
	docker-compose down

docker-logs: ## Show Docker logs
	docker-compose logs -f

docker-restart: docker-down docker-up ## Restart Docker containers

# ============================================
# CI/CD
# ============================================

ci-local: ## Run CI checks locally
	@echo "$(BLUE)Running CI checks locally...$(NC)"
	@$(MAKE) fmt
	@$(MAKE) lint
	@$(MAKE) test
	@$(MAKE) frontend-lint
	@$(MAKE) frontend-typecheck
	@echo "$(GREEN)All CI checks passed!$(NC)"

ci-coverage: test-coverage frontend-typecheck ## Generate coverage reports

# ============================================
# Development Setup
# ============================================

setup: ## Initial project setup
	@echo "$(BLUE)Setting up project...$(NC)"
	@mkdir -p data configs $(BIN_DIR) $(COVERAGE_DIR)
	@if [ ! -f configs/config.json ]; then \
		cp configs/config.example.json configs/config.json; \
		echo "$(YELLOW)Created configs/config.json from example$(NC)"; \
	fi
	$(MAKE) deps
	$(MAKE) frontend-install
	@echo "$(GREEN)Setup complete!$(NC)"

install-tools: ## Install development tools
	@echo "$(BLUE)Installing development tools...$(NC)"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	@echo "$(GREEN)Tools installed$(NC)"

# ============================================
# Clean
# ============================================

clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning...$(NC)"
	$(GOCLEAN)
	rm -rf $(BIN_DIR)/
	rm -rf $(COVERAGE_DIR)/
	rm -f coverage.out coverage.html
	@echo "$(GREEN)Clean complete$(NC)"

clean-all: clean ## Clean all including node_modules
	rm -rf $(WEB_DIR)/node_modules/
	rm -rf $(WEB_DIR)/dist/

# ============================================
# Help
# ============================================

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "$(YELLOW)AI Gateway - Available targets:$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Examples:$(NC)"
	@echo "  make build            # Build the application"
	@echo "  make test-coverage    # Run tests with coverage report"
	@echo "  make ci-local         # Run all CI checks locally"
	@echo "  make docker-up        # Start with Docker Compose"
