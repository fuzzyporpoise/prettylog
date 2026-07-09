# Go related variables
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin
GOFILES=$(wildcard *.go)

# Colors for pretty output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[0;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

.PHONY: help test clean install uninstall fmt lint vet tidy dev goreleaser-check

## help: Show this help message
help:
	@echo "$(BLUE)prettylog - Available targets:$(NC)"
	@echo ""
	@echo "$(GREEN)Development:$(NC)"
	@echo "  test        Run tests"
	@echo "  test-v      Run tests with verbose output"
	@echo "  test-cover  Run tests with coverage"
	@echo ""
	@echo "$(GREEN)Code Quality:$(NC)"
	@echo "  fmt         Format Go code"
	@echo "  lint        Run golangci-lint"
	@echo "  vet         Run go vet"
	@echo "  tidy        Tidy Go modules"
	@echo "  check       Run all quality checks (fmt, vet, lint, test)"
	@echo ""
	@echo "$(GREEN)Utilities:$(NC)"
	@echo "  deps        Install development dependencies"

## test: Run tests
test:
	@echo "$(BLUE)Running tests...$(NC)"
	@go test ./...
	@echo "$(GREEN)Tests passed$(NC)"

## test-v: Run tests with verbose output
test-v:
	@echo "$(BLUE)Running tests (verbose)...$(NC)"
	@go test -v ./...

## test-cover: Run tests with coverage
test-cover:
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	@go test -v -cover ./...
	@go test -coverprofile=coverage.out ./
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

## fmt: Format Go code
fmt:
	@echo "$(BLUE)Formatting code...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)Code formatted$(NC)"

## lint: Run golangci-lint
lint:
	@echo "$(BLUE)Running linter...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
		echo "$(GREEN)Linting complete$(NC)"; \
	else \
		echo "$(YELLOW)golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(NC)"; \
	fi

## vet: Run go vet
vet:
	@echo "$(BLUE)Running go vet...$(NC)"
	@go vet ./...
	@echo "$(GREEN)Vet check passed$(NC)"

## tidy: Tidy Go modules
tidy:
	@echo "$(BLUE)Tidying modules...$(NC)"
	@go mod tidy
	@echo "$(GREEN)Modules tidied$(NC)"

## check: Run all quality checks
check: fmt vet lint test
	@echo "$(GREEN)All quality checks passed$(NC)"

## clean: Clean artifacts
clean:
	@echo "$(BLUE)Cleaning...$(NC)"
	@rm -f coverage.out coverage.html
	@echo "$(GREEN)Clean complete$(NC)"

## deps: Install development dependencies
deps:
	@echo "$(BLUE)Installing development dependencies...$(NC)"
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "$(BLUE)Installing golangCI-lint...$(NC)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	fi
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "$(BLUE)Installing GoReleaser...$(NC)"; \
		go install github.com/goreleaser/goreleaser@latest; \
	fi
	@echo "$(GREEN)Dependencies installed$(NC)"

## goreleaser-check: Validate GoReleaser configuration
goreleaser-check:
	@echo "$(BLUE)Validating GoReleaser configuration...$(NC)"
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser check; \
		echo "$(GREEN)GoReleaser configuration is valid$(NC)"; \
	else \
		echo "$(YELLOW)GoReleaser not found. Install with: make deps$(NC)"; \
	fi

# Default target
all: check 
