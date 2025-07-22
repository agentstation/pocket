# Installing Pocket CLI

This guide covers all installation methods for the Pocket CLI.

## Quick Install

The fastest way to install Pocket:

```bash
go install github.com/agentstation/pocket/cmd/pocket@latest
```

This requires Go 1.21 or later installed on your system.

## Installation Methods

### Method 1: Using Go Install (Recommended)

```bash
# Install the latest version
go install github.com/agentstation/pocket/cmd/pocket@latest

# Install a specific version
go install github.com/agentstation/pocket/cmd/pocket@v0.1.0

# Verify installation
pocket version
```

### Method 2: Pre-built Binaries (Coming Soon)

```bash
# Linux
curl -L https://github.com/agentstation/pocket/releases/latest/download/pocket-linux-amd64 -o pocket
chmod +x pocket
sudo mv pocket /usr/local/bin/

# macOS
curl -L https://github.com/agentstation/pocket/releases/latest/download/pocket-darwin-amd64 -o pocket
chmod +x pocket
sudo mv pocket /usr/local/bin/

# Windows
# Download from https://github.com/agentstation/pocket/releases
# Add to PATH
```

### Method 3: Building from Source

```bash
# Clone the repository
git clone https://github.com/agentstation/pocket.git
cd pocket

# Build with make
make build

# Or build directly with go
go build -o pocket ./cmd/pocket

# Install to system
sudo mv pocket /usr/local/bin/
```

### Method 4: Using Docker (Coming Soon)

```bash
# Run directly
docker run --rm -v $(pwd):/workspace agentstation/pocket run workflow.yaml

# Create alias for convenience
alias pocket='docker run --rm -v $(pwd):/workspace agentstation/pocket'
```

## System Requirements

- **Go Version**: 1.21 or later (for go install method)
- **Operating Systems**: Linux, macOS, Windows
- **Architecture**: amd64, arm64

## Verifying Installation

After installation, verify Pocket is working:

```bash
# Check version
pocket version

# View help
pocket --help

# List available nodes
pocket nodes list
```

## Setting Up Your Environment

### Create Pocket Directory

Pocket uses `~/.pocket/` for configuration and plugins:

```bash
# Create directory structure
mkdir -p ~/.pocket/{plugins,scripts,config}
```

### Environment Variables

Optional environment variables:

```bash
# Set custom Pocket home directory
export POCKET_HOME="$HOME/.pocket"

# Set default log level
export POCKET_LOG_LEVEL="info"

# Add to your shell profile
echo 'export POCKET_HOME="$HOME/.pocket"' >> ~/.bashrc
```

## Next Steps

Now that Pocket is installed:

1. [Create your first workflow](getting-started.md)
2. [Explore example workflows](../workflows/)
3. [Learn about built-in nodes](../nodes/built-in/)

## Troubleshooting Installation

### Command Not Found

If `pocket` command is not found after installation:

```bash
# Check if Go bin is in PATH
echo $PATH | grep -q "$(go env GOPATH)/bin" || echo "Add $(go env GOPATH)/bin to PATH"

# Add to PATH (bash)
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc

# Add to PATH (zsh)
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc
```

### Permission Denied

If you get permission errors:

```bash
# Use sudo for system-wide installation
sudo mv pocket /usr/local/bin/

# Or install to user directory
mkdir -p ~/.local/bin
mv pocket ~/.local/bin/
export PATH="$PATH:$HOME/.local/bin"
```

### Go Not Found

If Go is not installed:

```bash
# macOS
brew install go

# Linux (Ubuntu/Debian)
sudo apt update && sudo apt install golang-go

# Or download from https://golang.org/dl/
```

## Updating Pocket

To update to the latest version:

```bash
# Using go install
go install github.com/agentstation/pocket/cmd/pocket@latest

# From source
cd pocket
git pull
make build
sudo mv pocket /usr/local/bin/
```

## Uninstalling

To remove Pocket:

```bash
# Remove binary
sudo rm /usr/local/bin/pocket

# Remove configuration (optional)
rm -rf ~/.pocket

# Remove from Go bin
rm $(go env GOPATH)/bin/pocket
```