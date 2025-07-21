.PHONY: all clean install-deps build build-js build-wasm

# Default target
all: build

# Install dependencies
install-deps:
	@echo "Installing dependencies..."
	npm install
	@echo "Checking for Javy..."
	@which javy > /dev/null || (echo "Javy not found. Installing..." && npm install -g @shopify/javy)

# Build JavaScript bundle
build-js: install-deps
	@echo "Building JavaScript bundle..."
	npm run build

# Build WASM from JavaScript
build-wasm: build-js
	@echo "Compiling to WebAssembly with Javy..."
	javy compile dist/plugin.js -o plugin.wasm
	@echo "WASM plugin built successfully!"

# Full build
build: build-wasm

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf dist/
	rm -f plugin.wasm
	@echo "Clean complete!"

# Install plugin to Pocket
install: build
	@echo "Installing plugin..."
	@if command -v pocket >/dev/null 2>&1; then \
		pocket plugins install .; \
	else \
		echo "Pocket CLI not found. Copy plugin.wasm and manifest.yaml to your plugins directory."; \
	fi

# Development build with watch mode
dev:
	@echo "Starting development mode..."
	npm run dev

# Run tests
test:
	@echo "Running tests..."
	npm test

# Check Javy installation
check-javy:
	@echo "Checking Javy installation..."
	@if command -v javy >/dev/null 2>&1; then \
		echo "✓ Javy is installed: $$(javy --version)"; \
	else \
		echo "✗ Javy is not installed"; \
		echo "  Install with: npm install -g @shopify/javy"; \
		echo "  Or download from: https://github.com/Shopify/javy/releases"; \
		exit 1; \
	fi