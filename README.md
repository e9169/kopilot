# Kopilot - Kubernetes AI Assistant

[![CI](https://github.com/e9169/kopilot/actions/workflows/ci.yml/badge.svg)](https://github.com/e9169/kopilot/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/e9169/kopilot/branch/main/graph/badge.svg)](https://codecov.io/gh/e9169/kopilot)
[![Security](https://github.com/e9169/kopilot/actions/workflows/security.yml/badge.svg)](https://github.com/e9169/kopilot/actions/workflows/security.yml)
[![CodeQL](https://github.com/e9169/kopilot/actions/workflows/codeql.yml/badge.svg)](https://github.com/e9169/kopilot/actions/workflows/codeql.yml)
[![Release](https://github.com/e9169/kopilot/actions/workflows/release.yml/badge.svg)](https://github.com/e9169/kopilot/actions/workflows/release.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev/doc/devel/release)
[![Go Report Card](https://goreportcard.com/badge/github.com/e9169/kopilot)](https://goreportcard.com/report/github.com/e9169/kopilot)
[![Author](https://img.shields.io/badge/Author-@e9169-blue?logo=github)](https://github.com/e9169)
[![Made in Sweden](https://img.shields.io/badge/Made%20in-Sweden-blue?logo=sweden)](https://github.com/e9169)

An interactive agent built with the **official GitHub Copilot SDK** in Go that provides real-time status information, management, and troubleshooting about Kubernetes clusters from your kubeconfig file.

![Kopilot Interface](docs/images/kopilot-interface.png)

> **🤖 AI-Generated Project Notice**
> This entire project was created during a vibe coding session using **GitHub Copilot** (Claude Sonnet 4.5 model) with the sole purpose of having fun and exploring what's possible with AI-assisted development. While it works, it started as an experiment in AI-powered software creation.
>
> **Contributions welcome!** If you'd like to help turn this fun experiment into a serious, production-ready tool, pull requests are greatly appreciated.

> **⚠️ AI Output Disclaimer**
> Kopilot uses AI models (GPT-4o and GPT-4o-mini by default) to generate responses and interpret your requests. While designed to be helpful, **AI-generated outputs may contain errors, misinterpretations, or incomplete information**.
>
> **Important:**
> - Always verify AI suggestions before applying them to production systems
> - Review kubectl commands before confirming execution (especially in interactive mode)
> - AI models can hallucinate or provide outdated information
> - Use read-only mode when learning or testing to prevent unintended changes
> - This tool is provided "as-is" without warranties - see [LICENSE](LICENSE) for details
>
> **You are responsible for understanding and verifying all operations performed on your Kubernetes clusters.**

## Features

- 🔍 **List All Clusters**: View all Kubernetes clusters configured in your kubeconfig
- 🔍 **Detailed Status**: Get comprehensive status information for specific clusters
- ⚖️ **Compare Clusters**: Side-by-side comparison of multiple clusters
- 🏥 **Health Monitoring**: Real-time node and **pod health** tracking across all clusters
- ⚡ **Parallel Execution**: Check all clusters simultaneously for 5-10x faster results
- 🤖 **Interactive Agent**: Natural language interface powered by GitHub Copilot
- 🛠️ **kubectl Integration**: Execute kubectl commands through natural language
- 🔐 **Safe by Default**: Read-only mode protects against accidental changes
- 🔓 **Interactive Mode**: Confirmation prompts for write operations with clear visibility
- ⚡ **Fast Health Checks**: Check all clusters in parallel with the check_all_clusters tool
- 📋 **Pretty Output**: Markdown tables and structured formatting for easy reading
- 💎 **Persistent Quota Display**: Real-time Copilot Premium request quota with color-coded indicators
- � **Modern ASCII Logo**: Stylized Kopilot branding with Kubernetes-themed colors (cyan/red)
- 💡 **Dynamic Example Suggestions**: Shows 3 random example prompts on each launch to help users get started
- 🎯 **Type-Safe Tools**: Uses Copilot SDK's DefineTool for automatic schema generation
- 💰 **Smart Cost Optimization**: Intelligent model selection based on task complexity (see [docs/MODEL_SELECTION.md](docs/MODEL_SELECTION.md))
  - Uses `gpt-4o-mini` for simple queries (list, status, health checks)
  - Automatically upgrades to `gpt-4o` for troubleshooting and complex operations
  - 50-70% cost reduction while maintaining quality for critical tasks
- 🚀 **GitHub Copilot CLI-inspired UX**: Clean, modern interface with chevron prompt (❯) and streamlined design

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

**Quick install (recommended)**

Install the latest release with a single command:

```bash
curl -fsSL https://raw.githubusercontent.com/e9169/kopilot/main/install.sh | bash
```

**Manual install**

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

- **Go 1.26 or later** - For building from source
- **kubectl** - Kubernetes command-line tool
- **GitHub Copilot CLI** - Version **0.0.410** (compatible with SDK v0.1.23)
- **GitHub Copilot subscription** - Required for AI features
- Access to Kubernetes clusters via kubeconfig
- Valid kubeconfig file at `~/.kube/config` or set via `KUBECONFIG`

> **⚠️ CLI Version Compatibility**
> The GitHub Copilot SDK v0.1.23 is compatible with CLI version **0.0.410** (commonly bundled with VS Code). This version combination has been tested and works reliably.

### Dependencies

This project uses the following key dependencies:

- **GitHub Copilot SDK**: `github.com/github/copilot-sdk/go@v0.1.23`
- **Kubernetes Client**: `k8s.io/client-go@v0.35.1`
- **Kubernetes API**: `k8s.io/api@v0.35.1`

Run `go mod verify` to ensure dependency integrity.

### Compatibility Matrix

| Component | Minimum Version | Recommended Version | Notes |
|-----------|----------------|---------------------|-------|
| Go | 1.26.0 | 1.26.0 | Required for building |
| Copilot CLI | 0.0.350 | **0.0.410** | SDK v0.1.23 protocol compatibility |
| Copilot SDK | v0.1.23 | v0.1.23 | Current version |
| kubectl | Any | Latest | Must be in PATH |
| Kubernetes | 1.28+ | 1.35.1 | API compatibility |

## Installation

### 1. Install Copilot CLI

**✅ Recommended:** Kopilot works with CLI version **0.0.410** (commonly bundled with VS Code):

```bash
# Install via npm
npm install -g @github/copilot@0.0.410

# Or use the VS Code bundled version (usually at):
# ~/.Library/Application Support/Code/User/globalStorage/github.copilot-chat/copilotCli/copilot (macOS)

# Verify installation
copilot --version
# Should output: GitHub Copilot CLI 0.0.410

# Authenticate
copilot auth login
```

**Alternative installation methods:**

If you prefer using other package managers (Homebrew, etc.), they typically install the latest version which works fine with SDK v0.1.23.

<details>
<summary>Other installation options</summary>

**Using Homebrew**
```bash
brew install github/gh-copilot/gh-copilot
# May install v0.0.410 - check version after install
```

**Using GitHub CLI extension**
```bash
gh extension install github/gh-copilot
```

**VS Code Extension**
```bash
# Install GitHub Copilot extension in VS Code
# Note: VS Code bundles v0.0.410 which works perfectly with Kopilot
```

</details>

For more details, see: https://docs.github.com/en/copilot/github-copilot-in-the-cli

### 2. Install Kopilot


**Quick install (recommended)**

Install the latest release with a single command:

```bash
curl -fsSL https://raw.githubusercontent.com/e9169/kopilot/main/install.sh | bash
```

This will automatically:
- Detect your OS and architecture
- Download the latest release
- Install to `/usr/local/bin` (or `~/.local/bin` if you prefer)
- Make the binary executable

**Pre-built binaries**

Download pre-built binaries from the [releases page](https://github.com/e9169/kopilot/releases).

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

**� Read-Only Mode (Default)**
- Blocks all write operations (scale, delete, apply, etc.)
- Safe for production environments and exploratory use
- Perfect for monitoring, troubleshooting, and viewing cluster state
- Use `--interactive` flag to enable writes with confirmation

**⚡ Interactive Mode**
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

When you start kopilot, it displays:
- An ASCII art logo with Kopilot branding
- Connection status and cluster information
- Current execution mode (read-only or interactive)
- **3 random example prompts** to help you get started

You can then interact naturally:

```bash
❯ Show me pods in the default namespace on dev-mgmt-01
❯ Scale the nginx deployment to 3 replicas in prod-wrk-01
❯ Check the logs of the api pod in namespace apps
❯ What's the status of nodes in the SEML region clusters?
❯ exit
```

The agent will execute kubectl commands on your behalf and explain the results. Example prompts are randomized on each launch to help you discover different capabilities.

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

MIT License - Copyright © 2026 Eneko Pérez
