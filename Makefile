# Configurator Library Makefile
# Following design principles: "Do more with less" and "Test, test, test"

.PHONY: help test lint fmt clean install

# Default target
help: ## Show this help message
	@echo "Configurator Library - Available targets:"
	@echo
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo

test: ## Run tests with coverage
	@echo "Running configurator tests..."
	@go test -v -cover ./...
	@echo "Tests completed ✅"

lint: ## Run comprehensive linting
	@echo "Running linters..."
	@golangci-lint run --fix
	@echo "Cleaning caches..."
	@golangci-lint cache clean
	@go clean -cache
	@echo "Linting completed ✅"

fmt: ## Format code
	@echo "Formatting Go code..."
	@go fmt ./...
	@gofmt -w .
	@echo "Formatting completed ✅"

clean: ## Clean build cache
	@echo "Cleaning cache..."
	@go clean -cache -testcache
	@echo "Cleanup completed ✅"

build: ## Build configurator binary to ~/bin
	@echo "Building configurator binary..."
	@CGO_ENABLED=0 go build -o ~/bin/configurator ./cmd/configurator
	@echo "Build completed ✅"
	@echo "Binary installed: ~/bin/configurator"

install: build ## Build and install configurator binary
	@echo "Configurator installed ✅"
	@echo "Usage: configurator --help"

# Development workflow
dev: fmt test lint ## Developer workflow: format, test, lint
	@echo "Development workflow completed ✅"
