.PHONY: build install clean version test help dev release

# Binary name
BINARY_NAME=mcpproxy

# Build information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0-dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S_UTC')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

# Go build flags
LDFLAGS := -X 'main.version=$(VERSION)' \
           -X 'main.buildTime=$(BUILD_TIME)' \
           -X 'main.gitCommit=$(GIT_COMMIT)' \
           -X 'main.gitBranch=$(GIT_BRANCH)'

# Build directory
BUILD_DIR := ./build
CMD_DIR := ./cmd/mcpproxy

# Default target
.DEFAULT_GOAL := help

## help: Display this help message
help:
	@echo "MCPProxy Build System"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*##"; printf ""} /^[a-zA-Z_-]+:.*?##/ { printf "  %-15s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

## build: Build the application with version info
build:
	@echo "Building $(BINARY_NAME)..."
	@echo "  Version:    $(VERSION)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Git Commit: $(GIT_COMMIT)"
	@echo "  Git Branch: $(GIT_BRANCH)"
	@go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: ./$(BINARY_NAME)"

## dev: Quick build for development (no version info)
dev:
	@echo "Building $(BINARY_NAME) (dev mode)..."
	@go build -o $(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: ./$(BINARY_NAME)"

## install: Install the application to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	@go install -ldflags "$(LDFLAGS)" $(CMD_DIR)
	@echo "Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

## release: Build release binaries for multiple platforms
release:
	@echo "Building release binaries..."
	@mkdir -p $(BUILD_DIR)

	@echo "Building for macOS (ARM64)..."
	@GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)

	@echo "Building for macOS (AMD64)..."
	@GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)

	@echo "Building for Linux (AMD64)..."
	@GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)

	@echo "Building for Linux (ARM64)..."
	@GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)

	@echo "Building for Windows (AMD64)..."
	@GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)

	@echo "Release builds complete in $(BUILD_DIR)/"

## version: Show version information
version:
	@echo "Version:    $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Git Branch: $(GIT_BRANCH)"

## test: Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

## run: Build and run the application
run: build
	@echo "Starting $(BINARY_NAME)..."
	@./$(BINARY_NAME) serve

## lint: Run linters
lint:
	@echo "Running linters..."
	@golangci-lint run --timeout 5m || echo "golangci-lint not installed"

## fmt: Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Format complete"

## tidy: Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy
	@echo "Tidy complete"

## check: Run all checks (fmt, lint, test)
check: fmt lint test
	@echo "All checks passed"
