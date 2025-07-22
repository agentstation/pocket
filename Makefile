.PHONY: all test lint fmt clean coverage bench install-tools generate help install-devbox devbox-update devbox build build-all install

# Build variables
VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION := $(shell go version | cut -d ' ' -f 3)

# Build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE) -X main.goVersion=$(GO_VERSION)"

# Default target
all: test lint generate build

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	@go test -race -v ./...

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	@go test -tags=integration -v ./cmd/pocket -run "Integration"

# Run end-to-end tests
test-e2e: build
	@echo "Running end-to-end tests..."
	@go test -tags=e2e -v ./cmd/pocket -run "E2E"

# Run all tests (unit, integration, e2e)
test-all: test test-integration test-e2e
	@echo "All tests completed!"

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

# Comprehensive formatting with all tools
fmt-all:
	@echo "Running comprehensive formatting..."
	@echo "  → Running gofmt..."
	@gofmt -w .
	@echo "  → Running goimports..."
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w -local "github.com/agentstation/pocket" .; \
	else \
		echo "    goimports not installed, skipping..."; \
	fi
	@echo "  → Running godot..."
	@if command -v godot >/dev/null 2>&1; then \
		godot -w .; \
	else \
		echo "    godot not installed, skipping..."; \
	fi
	@echo "  → Running golangci-lint with auto-fix..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --fix; \
	else \
		echo "    golangci-lint not installed, skipping..."; \
	fi
	@echo "  → Running go mod tidy..."
	@go mod tidy
	@echo "Formatting complete!"

# Check formatting without making changes
fmt-check:
	@echo "Checking code formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "The following files need formatting:"; \
		gofmt -l .; \
		exit 1; \
	fi
	@echo "All files are properly formatted."

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@go clean -cache -testcache
	@rm -rf bin/

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
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@go install github.com/tetafro/godot/cmd/godot@latest
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

# Build the pocket binary
build:
	@echo "Building pocket binary..."
	@mkdir -p bin
	@go build $(LDFLAGS) -o bin/pocket ./cmd/pocket
	@echo "Binary built: bin/pocket"

# Note: pocket-plugins has been integrated into the main pocket binary
# Use 'pocket plugins' command instead

# Build for all platforms
build-all:
	@echo "Building for all platforms..."
	@mkdir -p bin
	@echo "  → Darwin AMD64"
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/pocket-darwin-amd64 ./cmd/pocket
	@echo "  → Darwin ARM64"
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/pocket-darwin-arm64 ./cmd/pocket
	@echo "  → Linux AMD64"
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/pocket-linux-amd64 ./cmd/pocket
	@echo "  → Windows AMD64"
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/pocket-windows-amd64.exe ./cmd/pocket
	@echo "All binaries built in bin/"

# Install the binary to GOPATH/bin
install: build
	@echo "Installing pocket to $(GOPATH)/bin..."
	@cp bin/pocket $(GOPATH)/bin/pocket
	@echo "Installed successfully!"

# Create a new release tag
release-tag:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make release-tag VERSION=v1.0.0"; \
		exit 1; \
	fi
	@echo "Creating release tag $(VERSION)..."
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@echo "Tag created. Push with: git push origin $(VERSION)"

# Test release locally with GoReleaser
release-test:
	@echo "Testing release with GoReleaser..."
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "Error: goreleaser not installed. Install with:"; \
		echo "  go install github.com/goreleaser/goreleaser@latest"; \
		exit 1; \
	fi
	@goreleaser release --snapshot --clean --skip=publish

# Create a full release (requires appropriate permissions)
release:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make release VERSION=v1.0.0"; \
		exit 1; \
	fi
	@echo "Creating release $(VERSION)..."
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@git push origin $(VERSION)
	@echo "Tag pushed. GitHub Actions will handle the release."

# Show help
help:
	@echo "Available targets:"
	@echo "  make test          - Run tests"
	@echo "  make test-race     - Run tests with race detector"
	@echo "  make test-integration - Run integration tests"
	@echo "  make test-e2e      - Run end-to-end tests (builds binary first)"
	@echo "  make test-all      - Run all tests (unit, integration, e2e)"
	@echo "  make lint          - Run linters"
	@echo "  make fmt           - Format code (basic)"
	@echo "  make fmt-all       - Comprehensive formatting with all tools"
	@echo "  make fmt-check     - Check formatting without changes"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make coverage      - Generate coverage report"
	@echo "  make bench         - Run benchmarks"
	@echo "  make generate      - Generate documentation"
	@echo "  make build         - Build pocket binary"
	@echo "  make build-all     - Build for all platforms"
	@echo "  make install       - Install pocket binary to GOPATH/bin"
	@echo "  make install-tools - Install development tools"
	@echo "  make install-devbox - Install devbox"
	@echo "  make devbox-update - Update devbox packages"
	@echo "  make devbox        - Run devbox shell"
	@echo "  make release-tag VERSION=v1.0.0 - Create a release tag"
	@echo "  make release-test  - Test release locally with GoReleaser"
	@echo "  make release VERSION=v1.0.0 - Create and push release tag"
	@echo "  make help          - Show this help message"