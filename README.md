# Kopilot - Kubernetes Cluster Status Agent

[![CI](https://github.com/e9169/kopilot/actions/workflows/ci.yml/badge.svg)](https://github.com/e9169/kopilot/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/e9169/kopilot/branch/main/graph/badge.svg)](https://codecov.io/gh/e9169/kopilot)
[![Security](https://github.com/e9169/kopilot/actions/workflows/security.yml/badge.svg)](https://github.com/e9169/kopilot/actions/workflows/security.yml)
[![CodeQL](https://github.com/e9169/kopilot/actions/workflows/codeql.yml/badge.svg)](https://github.com/e9169/kopilot/actions/workflows/codeql.yml)
[![Release](https://github.com/e9169/kopilot/actions/workflows/release.yml/badge.svg)](https://github.com/e9169/kopilot/actions/workflows/release.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev/doc/devel/release)
[![Go Report Card](https://goreportcard.com/badge/github.com/e9169/kopilot)](https://goreportcard.com/report/github.com/e9169/kopilot)
[![Author](https://img.shields.io/badge/Author-@e9169-blue?logo=github)](https://github.com/e9169)
[![Made in Sweden](https://img.shields.io/badge/Made%20in-Sweden-blue?logo=sweden)](https://github.com/e9169)

An interactive agent built with the **official GitHub Copilot SDK** in Go that provides real-time status information, management, and troubleshooting about Kubernetes clusters from your kubeconfig file.

> **🤖 AI-Generated Project Notice**  
> This entire project was created during a vibe coding session using **GitHub Copilot** (Claude Sonnet 4.5 model) with the sole purpose of having fun and exploring what's possible with AI-assisted development. While it works, it started as an experiment in AI-powered software creation.  
> 
> **Contributions welcome!** If you'd like to help turn this fun experiment into a serious, production-ready tool, pull requests are greatly appreciated.

## Features

- 🔍 **List All Clusters**: View all Kubernetes clusters configured in your kubeconfig
- 📊 **Detailed Status**: Get comprehensive status information for specific clusters
- ⚖️ **Compare Clusters**: Side-by-side comparison of multiple clusters
- 🏥 **Health Monitoring**: Real-time node and **pod health** tracking across all clusters
- ⚡ **Parallel Execution**: Check all clusters simultaneously for 5-10x faster results
- 🤖 **Interactive Agent**: Natural language interface powered by GitHub Copilot
- 🛠️ **kubectl Integration**: Execute kubectl commands through natural language
- � **Safe by Default**: Read-only mode protects against accidental changes
- 🔓 **Interactive Mode**: Confirmation prompts for write operations with clear visibility
- �🚀 **Proactive Monitoring**: Automatic cluster health check on startup with pod status
- 📋 **Pretty Output**: Markdown tables and structured formatting for easy reading
- 💎 **Quota Tracking**: Real-time Copilot Premium request quota display with color indicators
- 🎯 **Type-Safe Tools**: Uses Copilot SDK's DefineTool for automatic schema generation
- 💰 **Smart Cost Optimization**: Intelligent model selection based on task complexity (see [docs/MODEL_SELECTION.md](docs/MODEL_SELECTION.md))
  - Uses `gpt-4o-mini` for simple queries (list, status, health checks)
  - Automatically upgrades to `gpt-4o` for troubleshooting and complex operations
  - 50-70% cost reduction while maintaining quality for critical tasks

## Architecture

The agent uses the **official GitHub Copilot SDK** to create an interactive assistant that can query your Kubernetes clusters. The SDK handles model invocation, tool selection, and response generation, while our custom tools provide the actual cluster information.

## Supported Platforms

Kopilot is compiled and released for:

| OS | Architecture | Tested in CI | Binary Available |
|----|--------------|--------------|------------------|
| Linux | amd64 | ✅ | ✅ |
| Linux | arm64 | ❌ | ✅ |
| macOS | amd64 (Intel) | ❌ | ✅ |
| macOS | arm64 (Apple Silicon) | ✅ | ✅ |
| Windows | amd64 | ❌ | ✅ |
| Windows | arm64 | ❌ | ✅ |

**Note:** All platforms are verified to compile successfully in CI. Full test suite runs on Ubuntu (linux/amd64) and macOS (darwin/arm64).

## Quick Start

```bash
# Install dependencies
make deps

# Build the binary
make build

# Run kopilot
./bin/kopilot

# Or install and run from anywhere
make install
kopilot
```

## Prerequisites

- **Go 1.25 or later** - For building from source
- **kubectl** - Kubernetes command-line tool
- **GitHub Copilot CLI** - Version 0.0.350 or later (tested with 0.0.394)
- **GitHub Copilot subscription** - Required for AI features
- Access to Kubernetes clusters via kubeconfig
- Valid kubeconfig file at `~/.kube/config` or set via `KUBECONFIG`

### Dependencies

This project uses the following key dependencies:

- **GitHub Copilot SDK**: `github.com/github/copilot-sdk/go@v0.1.20`
- **Kubernetes Client**: `k8s.io/client-go@v0.35.0`
- **Kubernetes API**: `k8s.io/api@v0.35.0`

Run `go mod verify` to ensure dependency integrity.

### Compatibility Matrix

| Component | Minimum Version | Tested Version | Notes |
|-----------|----------------|----------------|-------|
| Go | 1.25.0 | 1.25.6 | Required for building |
| Copilot CLI | 0.0.350 | 0.0.394 | Auto-detected from PATH or VS Code |
| Copilot SDK | v0.1.20 | v0.1.20 | Current version |
| kubectl | Any | Latest | Must be in PATH |
| Kubernetes | 1.28+ | 1.35.0 | API compatibility |

## Installation

### 1. Install Copilot CLI

Choose one of the following methods:

**Option A: Using npm (recommended)**
```bash
npm install -g @githubnext/github-copilot-cli

# Verify installation
copilot --version

# Authenticate
copilot auth login
```

**Option B: Using Homebrew**
```bash
brew install github/gh-copilot/gh-copilot
```

**Option C: Using GitHub CLI extension**
```bash
gh extension install github/gh-copilot
```

**Option D: VS Code Extension**
```bash
# Install GitHub Copilot extension in VS Code
# The CLI will be automatically bundled
```

For more details, see: https://docs.github.com/en/copilot/github-copilot-in-the-cli

### 2. Install Kopilot

**Build from source**
```bash
git clone https://github.com/e9169/kopilot.git
cd kopilot
make deps
make build
```

### 3. Verify Installation

```bash
kopilot --version
```

## Usage

```bash
# Run in read-only mode (default - safest)
./bin/kopilot

# Run in interactive mode (asks before write operations)
./bin/kopilot --interactive

# Show version information
./bin/kopilot --version

# Run with verbose logging
./bin/kopilot -v

# Show help
./bin/kopilot --help

# Use custom kubeconfig
KUBECONFIG=/path/to/kubeconfig ./bin/kopilot

# Use custom kubeconfig and context
./bin/kopilot --kubeconfig /path/to/kubeconfig --context my-context

# JSON output for tool responses
./bin/kopilot --output json
```

### Execution Modes

**🔒 Read-Only Mode (Default)**
- Blocks all write operations (scale, delete, apply, etc.)
- Safe for production environments and exploratory use
- Perfect for monitoring, troubleshooting, and viewing cluster state
- Use `--interactive` flag to enable writes with confirmation

**🔓 Interactive Mode**
- Asks for confirmation before executing write operations
- Shows exactly what command will run
- Allows cancellation of dangerous operations
- Can be enabled at startup with `--interactive` flag

**Runtime Mode Switching**
- `/readonly` - Switch to read-only mode
- `/interactive` - Switch to interactive mode
- `/mode` - Show current execution mode

### Command-Line Flags

- `--version` - Display version information
- `--interactive` - Enable interactive mode (asks before write operations)
- `--kubeconfig` - Path to kubeconfig file (default: `$KUBECONFIG` or `~/.kube/config`)
- `--context` - Override kubeconfig context
- `--output` - Output format: `text` or `json`
- `-v, --verbose` - Enable verbose logging with timestamps
- `--help` - Show usage information

### Environment Variables

**Optional:**
- `KUBECONFIG` - Path to kubeconfig file (default: `~/.kube/config`)

**Optional - Model Configuration:**
- `KOPILOT_MODEL_COST_EFFECTIVE` - Override AI model for simple queries (default: `gpt-4o-mini`)
- `KOPILOT_MODEL_PREMIUM` - Override AI model for complex operations (default: `gpt-4o`)

**Example:**
```bash
# Use different AI models
export KOPILOT_MODEL_COST_EFFECTIVE="gpt-3.5-turbo"
export KOPILOT_MODEL_PREMIUM="gpt-4"
./bin/kopilot

# Use custom kubeconfig
export KUBECONFIG="/path/to/custom/kubeconfig"
./bin/kopilot --interactive
```

### Interactive Session

When you start kopilot, it automatically:

1. Checks the health of ALL your clusters
2. Reports any issues or confirms everything is healthy
3. Asks "What would you like me to do?"

Then you can interact naturally:

```bash
> Show me pods in the default namespace on dev-mgmt-01
> Scale the nginx deployment to 3 replicas in prod-wrk-01
> Check the logs of the api pod in namespace apps
> What's the status of nodes in the SEML region clusters?
> exit
```

The agent will execute kubectl commands on your behalf and explain the results.

## Available Tools

1. **list_clusters** - Lists all clusters from kubeconfig
2. **get_cluster_status** - Gets detailed status for a specific cluster
3. **compare_clusters** - Compares multiple clusters side by side
4. **check_all_clusters** - Fast parallel health check of all clusters (🚀 5-10x faster)
5. **kubectl_exec** - Execute kubectl commands against any cluster

## References

- [GitHub Copilot SDK](https://github.com/github/copilot-sdk)
- [Copilot CLI Docs](https://docs.github.com/en/copilot/using-github-copilot/using-github-copilot-in-the-command-line)


## Author

**Eneko P** ([@e9169](https://github.com/e9169))  
📍 Based in Sweden

## License

MIT License - Copyright © 2026 e9169
