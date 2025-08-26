# Chi-Chi-Moni Makefile

# Variables
BINARY_NAME=monies
BIN_DIR=bin
BUILD_DIR=build
VERSION?=dev
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Default target
.PHONY: all
all: clean test build

# Build the binary
.PHONY: build
build:
	mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) .

# Build for multiple platforms
.PHONY: build-all
build-all: clean
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .

# Run tests
.PHONY: test
test:
	$(GOTEST) -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	$(GOTEST) -v -cover ./...

# Generate detailed coverage report
.PHONY: test-coverage-html
test-coverage-html:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detection
.PHONY: test-race
test-race:
	$(GOTEST) -v -race ./...

# Benchmark tests
.PHONY: bench
bench:
	$(GOTEST) -bench=. -benchmem ./...

# Clean build artifacts
.PHONY: clean
clean:
	$(GOCLEAN)
	rm -rf $(BIN_DIR)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Install dependencies
.PHONY: deps
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Verify dependencies
.PHONY: deps-verify
deps-verify:
	$(GOMOD) verify

# Update dependencies
.PHONY: deps-update
deps-update:
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

# Format code
.PHONY: fmt
fmt:
	$(GOCMD) fmt ./...

# Run linter (requires golangci-lint)
.PHONY: lint
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin v1.54.2" && exit 1)
	golangci-lint run

# Run security check (requires gosec)
.PHONY: security
security:
	@which gosec > /dev/null || (echo "gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest" && exit 1)
	gosec ./...

# Install the binary to GOPATH/bin
.PHONY: install
install:
	$(GOCMD) install $(LDFLAGS) .

# Run the application (requires setup token as argument)
.PHONY: run
run:
	@if [ -z "$(TOKEN)" ]; then \
		echo "Usage: make run TOKEN=your-setup-token"; \
		echo "Example: make run TOKEN=aHR0cHM6Ly9iZXRhLWJyaWRnZS5zaW1wbGVmaW4ub3JnL3NpbXBsZWZpbi9jbGFpbS8uLi4="; \
		exit 1; \
	fi
	@if [ ! -f "$(BIN_DIR)/$(BINARY_NAME)" ]; then \
		echo "Binary not found. Building..."; \
		$(MAKE) build; \
	fi
	./$(BIN_DIR)/$(BINARY_NAME) "$(TOKEN)"

# Development workflow - format, test, and build
.PHONY: dev
dev: fmt test build

# CI workflow - comprehensive checks
.PHONY: ci
ci: deps-verify fmt lint security test-race test-coverage build

# Release workflow
.PHONY: release
release: clean ci build-all
	@echo "Release artifacts created in $(BUILD_DIR)/"

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all           - Clean, test, and build (default)"
	@echo "  build         - Build the binary for current platform"
	@echo "  build-all     - Build for multiple platforms (Linux, macOS, Windows)"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  test-coverage-html - Generate HTML coverage report"
	@echo "  test-race     - Run tests with race detection"
	@echo "  bench         - Run benchmark tests"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Install and tidy dependencies"
	@echo "  deps-verify   - Verify dependencies"
	@echo "  deps-update   - Update dependencies"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter (requires golangci-lint)"
	@echo "  security      - Run security check (requires gosec)"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  run           - Run the application (use TOKEN=your-token)"
	@echo "  dev           - Development workflow (fmt + test + build)"
	@echo "  ci            - CI workflow (comprehensive checks)"
	@echo "  release       - Release workflow (build for all platforms)"
	@echo "  help          - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make test-coverage"
	@echo "  make run TOKEN=your-base64-token"
	@echo "  make release VERSION=v1.0.0"
