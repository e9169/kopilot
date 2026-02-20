# Installation Guide

This guide covers all available methods for installing Kopilot.

## Quick Install (Recommended)

The fastest way to get started is using the one-liner installer:

```bash
curl -fsSL https://raw.githubusercontent.com/e9169/kopilot/main/install.sh | bash
```

### What it does

The install script:
1. Auto-detects your operating system and architecture
2. Downloads the latest release binary from GitHub
3. Verifies the download
4. Installs to `/usr/local/bin` (with sudo prompt if needed) or `~/.local/bin`
5. Makes the binary executable
6. Verifies the installation

### Supported Platforms

The installer supports:
- **Linux**: amd64, arm64
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)
- **Windows**: amd64, arm64 (via Git Bash, WSL, or similar)

## Manual Installation

### Pre-built Binaries

1. Visit the [releases page](https://github.com/e9169/kopilot/releases)
2. Download the archive for your platform
3. Extract the archive:
   ```bash
   # Linux/macOS
   tar -xzf kopilot_VERSION_OS_ARCH.tar.gz
   
   # Windows
   unzip kopilot_VERSION_windows_ARCH.zip
   ```
4. Move the binary to a directory in your PATH:
   ```bash
   # Linux/macOS
   sudo mv kopilot /usr/local/bin/
   
   # Or to user directory
   mkdir -p ~/.local/bin
   mv kopilot ~/.local/bin/
   ```
5. Make it executable (Linux/macOS):
   ```bash
   chmod +x /usr/local/bin/kopilot
   ```

### Build from Source

Requirements:
- Go 1.26 or later
- Git
- Make

```bash
# Clone the repository
git clone https://github.com/e9169/kopilot.git
cd kopilot

# Install dependencies
make deps

# Build the binary
make build

# The binary will be at bin/kopilot
./bin/kopilot --version

# Optionally install to $GOPATH/bin
make install
```

### Development Build

For contributing or testing unreleased features:

```bash
# Clone and build
git clone https://github.com/e9169/kopilot.git
cd kopilot

# Run directly without building
make run

# Or build and install
make build install
```

## Package Managers

### Homebrew (Planned)

In the future, we plan to support Homebrew installation:

```bash
# Not yet available
# brew install e9169/tap/kopilot
```

Currently disabled in GoReleaser config, as it requires a personal access token for the tap repository.

### Chocolatey (Planned)

Windows users may eventually install via Chocolatey:

```bash
# Not yet available
# choco install kopilot
```

### Snap (Planned)

Linux users may install via Snap:

```bash
# Not yet available
# snap install kopilot
```

## Verifying Installation

After installation, verify kopilot is working:

```bash
# Check version
kopilot --version

# Ensure it's in your PATH
which kopilot

# Test with a simple query (requires GitHub Copilot subscription)
kopilot
# Then type: "list clusters"
```

## Updating

### Using the Installer

Simply re-run the install script:

```bash
curl -fsSL https://raw.githubusercontent.com/e9169/kopilot/main/install.sh | bash
```

The script will detect the latest version and update automatically.

### Manual Update

1. Download the latest release
2. Replace the old binary with the new one
3. Verify the new version:
   ```bash
   kopilot --version
   ```

### From Source

```bash
cd kopilot
git pull
make clean build
make install
```

## Uninstalling

Remove the binary from your system:

```bash
# If installed to /usr/local/bin
sudo rm /usr/local/bin/kopilot

# If installed to ~/.local/bin
rm ~/.local/bin/kopilot

# If installed via 'make install'
rm $(go env GOPATH)/bin/kopilot
```

## Troubleshooting

### "command not found: kopilot"

The binary is not in your PATH. Either:
1. Add the installation directory to your PATH
2. Run the binary using its full path

Add to PATH (add to `~/.bashrc`, `~/.zshrc`, or equivalent):
```bash
export PATH="$PATH:$HOME/.local/bin"
```

### "Permission denied"

The binary is not executable:
```bash
chmod +x /path/to/kopilot
```

Or the installation directory requires sudo:
```bash
sudo mv kopilot /usr/local/bin/
```

### Download fails

Check your internet connection and verify the release exists:
- Visit https://github.com/e9169/kopilot/releases
- Ensure your OS and architecture are supported

### Wrong version downloaded

The script detects OS and architecture automatically using `uname`. If detection fails:
1. Download the binary manually
2. Verify your platform with:
   ```bash
   echo "OS: $(uname -s)"
   echo "Arch: $(uname -m)"
   ```

## Next Steps

After installing Kopilot:

1. Install and authenticate GitHub Copilot CLI (required)
   ```bash
   npm install -g @github/copilot@0.0.410
   copilot auth login
   ```

2. Verify your kubeconfig
   ```bash
   kubectl config view
   ```

3. Run kopilot
   ```bash
   kopilot
   ```

For more information, see the [main README](../README.md) and [usage documentation](README.md).
