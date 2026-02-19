.PHONY: help build bundle run clean deps test fmt lint install

# Variables
APP_NAME := kopilot
BUILD_DIR := bin
GO := go
GOFLAGS := -v
GOLANGCI_LINT_VERSION ?= v1.64.8

# Version management: Use git tags, fallback to "dev"
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Build information
BUILD_DATE := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -X 'main.version=$(VERSION)' -X 'main.buildDate=$(BUILD_DATE)' -X 'main.gitCommit=$(GIT_COMMIT)'

# Default target
.DEFAULT_GOAL := help

help: ## Show this help message
	@echo "$(APP_NAME) - Kubernetes Cluster Status Agent"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

deps: ## Download and verify dependencies
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod verify
	@echo "Dependencies installed successfully"

bundle: ## Download and embed the Copilot CLI binary for the current platform
	@echo "Bundling Copilot CLI..."
	$(GO) tool bundler
	@echo "Copilot CLI bundled successfully"

build: deps bundle ## Build the application binary
	@echo "Building $(APP_NAME) v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"
	@$(BUILD_DIR)/$(APP_NAME) --version

run: deps ## Run the application without building
	@echo "Running $(APP_NAME)..."
	$(GO) run .

install: build ## Install the binary to $GOPATH/bin
	@echo "Installing $(APP_NAME)..."
	$(GO) install .
	@echo "$(APP_NAME) installed to $$(go env GOPATH)/bin/$(APP_NAME)"

version: ## Show current version
	@echo "Current version: $(VERSION)"
	@echo "Git commit: $(GIT_COMMIT)"
	@echo "Build date: $(BUILD_DATE)"

tag: ## Create a new git tag (usage: make tag VERSION=v1.2.3)
ifndef VERSION_TAG
	@echo "Error: VERSION_TAG is required"
	@echo "Usage: make tag VERSION_TAG=v1.2.3"
	@exit 1
endif
	@echo "Creating tag $(VERSION_TAG)..."
	git tag -a $(VERSION_TAG) -m "Release $(VERSION_TAG)"
	@echo "Tag created. Push with: git push origin $(VERSION_TAG)"
	@echo "To push all tags: git push origin --tags"

release: ## Show release instructions
	@echo "Release Process:"
	@echo "  1. Update CHANGELOG.md with release notes"
	@echo "  2. Commit all changes: git commit -am 'chore: prepare release'"
	@echo "  3. Create tag: make tag VERSION_TAG=v1.2.3"
	@echo "  4. Push changes: git push origin main"
	@echo "  5. Push tag: git push origin v1.2.3"
	@echo "  6. Create GitHub release from tag"
	@echo ""
	@echo "Current version: $(VERSION)"

test: ## Run unit tests
	@echo "Running unit tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	@echo "Tests complete"

test-integration: ## Run integration tests (requires valid kubeconfig)
	@echo "Running integration tests..."
	$(GO) test -v -tags=integration ./...
	@echo "Integration tests complete"

test-all: test test-integration ## Run all tests (unit + integration)
	@echo "All tests complete"

test-short: ## Run tests in short mode
	@echo "Running tests in short mode..."
	$(GO) test -short ./...
	@echo "Short tests complete"

coverage: test ## Generate test coverage report
	@echo "Generating coverage report..."
	$(GO) tool cover -html=coverage.out -o coverage.html
	@$(GO) tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'
	@echo "Coverage report generated: coverage.html"

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...
	@echo "Benchmarks complete"

fmt: ## Format Go code
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Code formatted"

lint: ## Run golangci-lint
	@echo "Running linter..."
	@which golangci-lint > /dev/null 2>&1 || \
		(echo "golangci-lint not found. Installing..." && \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION))
	@golangci-lint run ./...
	@echo "Linting complete"

vet: ## Run go vet
	@echo "Running go vet..."
	$(GO) vet ./...
	@echo "Vet complete"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@rm -f zcopilot_*
	@echo "Clean complete"

tidy: ## Tidy and verify Go modules
	@echo "Tidying modules..."
	$(GO) mod tidy
	$(GO) mod verify
	@echo "Modules tidied"

setup-hooks: ## Install git hooks for development
	@echo "Setting up git hooks..."
	@chmod +x .githooks/pre-commit
	@git config core.hooksPath .githooks
	@echo "Git hooks installed successfully"

check: fmt vet lint test ## Run all checks (format, vet, lint, test)
	@echo "All checks passed ✅"

.PHONY: info
info: ## Show build and version information
	@echo "$(APP_NAME) version $(VERSION)"
	@echo "Git commit: $(GIT_COMMIT)"
	@echo "Build date: $(BUILD_DATE)"
	@echo "Go version: $$($(GO) version)"
