# Installing Pocket CLI

This guide covers all installation methods for the Pocket CLI.

## Quick Install

The fastest way to install Pocket:

```bash
# macOS and Linux via Homebrew
brew install agentstation/tap/pocket
```

## Installation Methods

### Method 1: Homebrew (Recommended for macOS and Linux)

```bash
# Tap and install
brew tap agentstation/tap
brew install pocket

# Or install directly
brew install agentstation/tap/pocket

# Verify installation
pocket version
```

### Method 2: Using Go Install

```bash
# Install the latest version
go install github.com/agentstation/pocket/cmd/pocket@latest

# Install a specific version
go install github.com/agentstation/pocket/cmd/pocket@v0.1.0

# Verify installation
pocket version
```

Note: This requires Go 1.21 or later installed on your system.

### Method 3: Install Script

Our automated install script detects your platform and downloads the appropriate binary:

```bash
# Install latest version
curl -sSL https://raw.githubusercontent.com/agentstation/pocket/master/install.sh | bash

# Install specific version
curl -sSL https://raw.githubusercontent.com/agentstation/pocket/master/install.sh | bash -s -- --version v1.0.0

# Install to custom directory
curl -sSL https://raw.githubusercontent.com/agentstation/pocket/master/install.sh | bash -s -- --install-dir ~/.local/bin
```

### Method 4: Pre-built Binaries

Download binaries directly from our [releases page](https://github.com/agentstation/pocket/releases/latest):

```bash
# Linux x64
curl -L https://github.com/agentstation/pocket/releases/latest/download/pocket-linux-x86_64.tar.gz -o pocket.tar.gz
tar -xzf pocket.tar.gz
sudo mv pocket-linux-x86_64/pocket /usr/local/bin/

# macOS Intel
curl -L https://github.com/agentstation/pocket/releases/latest/download/pocket-darwin-x86_64.tar.gz -o pocket.tar.gz
tar -xzf pocket.tar.gz
sudo mv pocket-darwin-x86_64/pocket /usr/local/bin/

# macOS Apple Silicon
curl -L https://github.com/agentstation/pocket/releases/latest/download/pocket-darwin-arm64.tar.gz -o pocket.tar.gz
tar -xzf pocket.tar.gz
sudo mv pocket-darwin-arm64/pocket /usr/local/bin/

# Windows
# Download pocket-windows-x86_64.zip from releases page
# Extract and add to PATH
```

Available platforms:
- **macOS**: darwin-x86_64 (Intel), darwin-arm64 (Apple Silicon)
- **Linux**: linux-x86_64, linux-arm64, linux-i386
- **Windows**: windows-x86_64, windows-i386

### Method 5: Building from Source

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