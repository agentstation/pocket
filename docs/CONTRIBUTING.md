# Contributing to Pocket

Thank you for your interest in contributing to Pocket! We welcome contributions from the community.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/your-username/pocket.git`
3. Create a feature branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Run tests: `go test ./...`
6. Run linters: `golangci-lint run`
7. Commit your changes
8. Push to your fork
9. Create a Pull Request

## Development Setup

### Prerequisites

- Go 1.21 or higher
- golangci-lint (optional but recommended)

Install development tools:

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install other tools
go install golang.org/x/vuln/cmd/govulncheck@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with race detector
go test -race ./...

# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Running Linters

```bash
# Run golangci-lint
golangci-lint run

# Run specific linters
go vet ./...
gofmt -l .
staticcheck ./...
govulncheck ./...
```

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Write clear, self-documenting code
- Add comments for exported types and functions
- Keep functions small and focused
- Prefer composition over inheritance

## Testing

- Write tests for all new functionality
- Maintain or improve code coverage
- Use table-driven tests where appropriate
- Test edge cases and error conditions
- Run tests with race detector

## Pull Request Process

1. Ensure all tests pass
2. Ensure code passes all linters
3. Update documentation as needed
4. Add examples if introducing new features
5. Update README.md if necessary
6. Ensure your PR has a clear description

## Commit Messages

Follow conventional commit format:

```
type(scope): subject

body

footer
```

Types:
- feat: New feature
- fix: Bug fix
- docs: Documentation changes
- style: Code style changes (formatting, etc)
- refactor: Code refactoring
- test: Test changes
- chore: Build process or auxiliary tool changes

## Release Process (Maintainers)

Pocket follows semantic versioning (semver). To create a new release:

1. Ensure all tests pass on master branch
2. Update version references if needed
3. Create and push a version tag:

```bash
# For patch release (bug fixes)
git tag v0.1.1
git push origin v0.1.1

# For minor release (new features, backward compatible)
git tag v0.2.0
git push origin v0.2.0

# For major release (breaking changes)
git tag v1.0.0
git push origin v1.0.0
```

The release workflow will automatically:
- Run all tests
- Run linters
- Create a GitHub release
- Generate changelog from commits

## Getting Help

- Open an issue for bugs or feature requests
- Join discussions in issues and pull requests
- Check existing issues before creating new ones

## Code of Conduct

Please be respectful and inclusive in all interactions. We strive to maintain a welcoming community for all contributors.