.PHONY: all test bench race cover lint clean help

# Default target
all: test

# Run tests
test:
	@echo "Running tests..."
	@go test ./...

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Run tests with race detector
race:
	@echo "Running tests with race detector..."
	@go test -race ./...

# Generate coverage report
cover:
	@echo "Generating coverage report..."
	@go test -cover -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linters
lint:
	@echo "Running linters..."
	@go vet ./...
	@gofmt -l .
	@go mod tidy

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f coverage.out coverage.html
	@go clean

# Run examples
example-chat:
	@echo "Running chat example..."
	@go run examples/chat/main.go

example-agent:
	@echo "Running agent example..."
	@go run examples/agent/main.go

example-parallel:
	@echo "Running parallel example..."
	@go run examples/parallel/main.go

# Help
help:
	@echo "Available targets:"
	@echo "  make test          - Run tests"
	@echo "  make bench         - Run benchmarks"
	@echo "  make race          - Run tests with race detector"
	@echo "  make cover         - Generate coverage report"
	@echo "  make lint          - Run linters"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make example-chat  - Run chat example"
	@echo "  make example-agent - Run agent example"
	@echo "  make example-parallel - Run parallel example"
	@echo "  make help          - Show this help"