.PHONY: all test lint fmt clean coverage bench install-tools generate help install-devbox devbox-update devbox

# Default target
all: test lint generate

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	@go test -race -v ./...

# Run linters
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, running basic linters..."; \
		go vet ./...; \
		gofmt -l .; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	@gofmt -w .
	@go mod tidy

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@go clean -cache -testcache

# Generate coverage report
coverage:
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Install development tools
install-tools:
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@echo "Tools installed successfully"

# Generate documentation
generate:
	@echo "Generating documentation..."
	@go generate ./...

# Install devbox
install-devbox:
	@echo "Installing devbox..."
	@curl -fsSL https://get.jetify.dev | bash

# Update devbox packages
devbox-update:
	@echo "Updating devbox packages..."
	@devbox update

# Run devbox shell
devbox:
	@echo "Starting devbox shell..."
	@devbox shell

# Show help
help:
	@echo "Available targets:"
	@echo "  make test          - Run tests"
	@echo "  make test-race     - Run tests with race detector"
	@echo "  make lint          - Run linters"
	@echo "  make fmt           - Format code"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make coverage      - Generate coverage report"
	@echo "  make bench         - Run benchmarks"
	@echo "  make generate      - Generate documentation"
	@echo "  make install-tools - Install development tools"
	@echo "  make install-devbox - Install devbox"
	@echo "  make devbox-update - Update devbox packages"
	@echo "  make devbox        - Run devbox shell"
	@echo "  make help          - Show this help message"