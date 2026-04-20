---
layout: default
title: Documentation
permalink: /docs/
---

## Getting Started

Kopilot is an AI-powered assistant for Kubernetes cluster management. This guide will help you get up and running in minutes.

### Installation

#### Quick Install (Recommended)

Install with a single command:

```bash
curl -fsSL https://raw.githubusercontent.com/e9169/kopilot/main/install.sh | bash
```

This will:

- Auto-detect your OS and architecture (Linux, macOS, Windows)
- Download the latest release from GitHub
- Install to `/usr/local/bin` or `~/.local/bin`
- Make the binary executable and ready to use

**Supported platforms:** Linux (amd64, arm64), macOS (Intel, Apple Silicon), Windows (amd64, arm64)

#### Pre-built Binaries

Download from the [releases page](https://github.com/e9169/kopilot/releases).

#### From Source

```bash
git clone https://github.com/e9169/kopilot.git
cd kopilot
make deps
make build
make install
```

---

## Configuration

### Prerequisites

Kopilot uses the **GitHub Copilot SDK**, which requires:

1. **GitHub Copilot subscription** (Individual, Business, or Enterprise)
2. **GitHub Copilot CLI** installed and authenticated

```bash
# Install Copilot CLI (choose one method)
npm install -g @githubnext/github-copilot-cli
# or
gh extension install github/gh-copilot
# or
brew install github/gh-copilot/gh-copilot

# Authenticate
copilot auth login
```

### AI Models

Kopilot works with **any model available through your GitHub Copilot subscription**.

**Default configuration:**

- **Cost-effective model** (default: gpt-4.1) - Used for simple queries and status checks
- **Premium model** (default: claude-sonnet-4.6) - Automatically selected for troubleshooting and complex operations

**Customization:**

You can override the default models by setting environment variables:

```bash
# Example: Use different models
export KOPILOT_MODEL_COST_EFFECTIVE="claude-3.5-sonnet"
export KOPILOT_MODEL_PREMIUM="o1-preview"

# Then run kopilot
./bin/kopilot
```

Kopilot supports any model available in your GitHub Copilot plan, including: `gpt-4.1`, `claude-sonnet-4.6`, `claude-haiku-4.5`, `gpt-5-mini`, and others.

No API key configuration needed - authentication is handled through GitHub Copilot CLI.

### CLI Flags

```text
kopilot [flags]

Flags:
  --interactive       Enable interactive mode (prompts before write operations)
  --agent <name>      Start with a specialist agent persona (default, debugger, security, optimizer, gitops, sanitizer)
  --context <name>    Override the active kubeconfig context
  --kubeconfig <path> Path to kubeconfig file (default: $KUBECONFIG or ~/.kube/config)
  --output <format>   Output format: text (default) or json
  --mcp-config <path> Path to MCP server config file (default: ~/.kopilot/mcp.json)
  --version           Print version information
  -v, --verbose       Enable verbose logging
```

---

## Basic Usage

### Starting Kopilot

Kopilot offers two execution modes to balance safety and functionality:

#### 🔒 Read-Only Mode (Default — Recommended)

```bash
# Start in read-only mode (safest, default)
kopilot
```

- Blocks all write operations (scale, delete, apply, etc.)
- Perfect for exploration and monitoring
- Prevents accidental cluster modifications
- No confirmation prompts needed

#### 🔓 Interactive Mode

```bash
# Start in interactive mode
kopilot --interactive
```

- Allows write operations with confirmation
- Shows exact kubectl command before execution
- Requires explicit yes/no approval for changes
- Read-only commands execute immediately

#### Runtime Mode Switching

You can switch modes during a session without restarting:

```text
❯ /readonly          # Switch to read-only mode
❯ /interactive       # Switch to interactive mode
❯ /mode              # Show current mode
```

### Runtime Command Reference

All commands available at the `❯` prompt:

| Command | Description |
| ------- | ----------- |
| `/help` | Show all available commands |
| `/clear`, `/new` | Start a fresh conversation |
| `/usage` | Show session duration, turns, and quota |
| `/compact` | Summarize history to save context window |
| `/last` | Re-show the last full response |
| `/copy` | Copy the last response to clipboard |
| `/mode`, `/status` | Show current execution mode |
| `/readonly` | Switch to read-only mode |
| `/interactive` | Switch to interactive mode |
| `/model` | Show current model or routing mode |
| `/model <name>` | Force a specific model for this session |
| `/model reset` | Re-enable automatic model routing |
| `/streamer [on\|off]` | Hide quota badge (useful for screen-sharing) |
| `/context list` | List all kubeconfig contexts |
| `/context use <name>` | Switch active context |
| `/agent` | Show active agent and available roster |
| `/agent list` | Same as `/agent` |
| `/agent <name>` | Switch specialist agent |
| `/mcp list` | List configured MCP servers |
| `/mcp add <name> <url>` | Add or update an MCP server |
| `/mcp delete <name>` | Remove an MCP server |

**Shortcuts:**

| Shortcut | Description |
| -------- | ----------- |
| `@<filepath>` | Attach a file to the next message |
| `!<command>` | Run a shell command without AI |
| `Ctrl+C` | Cancel current input or abort AI response |
| `Ctrl+D` | Exit Kopilot |

### Example Session

```bash
kopilot
```

Example queries:

- "Show me all pods in the default namespace"
- "What's the status of my deployments?"
- "Why is my nginx pod crashing?"
- "Compare resource usage across namespaces"

For write operations, switch to interactive mode:

```text
❯ /interactive
❯ scale nginx deployment to 5 replicas
⚠️  Write Operation: kubectl scale deployment nginx --replicas=5
Do you want to proceed? (yes/no): yes
```

---

## Features

### 🚀 Smart Deployments

Kopilot can:

- Create deployments from natural language descriptions
- Apply best practices automatically
- Suggest resource limits and requests
- Set up health checks and readiness probes

### 🔍 Intelligent Debugging

Get AI-powered help with:

- Pod crash loops
- Scheduling issues
- Network connectivity problems
- Resource constraints

### 🛡️ Execution Modes

Kopilot provides two safety modes:

#### 🔒 Read-Only Mode

- Blocks all write operations for maximum safety
- Allows monitoring, querying, and troubleshooting
- Best for production environments
- No risk of accidental changes

#### 🔓 Interactive Mode (Write Operations)

- Enables write operations with confirmation
- Shows exact command before execution
- Requires explicit approval (yes/no)
- Can be started with `--interactive` flag or switched at runtime with `/interactive`

See the [Execution Modes documentation](https://github.com/e9169/kopilot/blob/main/docs/EXECUTION_MODES.md) for detailed information

### 🎭 Specialist Agent Personas

Kopilot ships five domain-focused AI personas that sharpen the assistant for specific operational areas. Start with `--agent <name>` or switch mid-session with `/agent <name>`.

**🔍 Debugger** (`--agent debugger`)
Root cause analysis, log correlation, and pod failure diagnosis. Starts with events and recent changes, correlates pod status and logs, traces failure chains.

- *Try: "Why is my pod in CrashLoopBackOff?" / "Diagnose why my service returns 503s"*

**🛡️ Security** (`--agent security`)
RBAC auditing, privilege escalation detection, and network policy review. Reports findings with CRITICAL/HIGH/MEDIUM/LOW severity and remediation steps.

- *Try: "Audit RBAC roles for overprivileged accounts" / "Find pods running as root"*

**⚡ Optimizer** (`--agent optimizer`)
Resource right-sizing, HPA/VPA recommendations, and cost optimization. Identifies over-provisioned workloads, missing limits, and idle deployments.

- *Try: "Which pods have no resource limits?" / "Find idle or low-traffic services"*

**🔄 GitOps** (`--agent gitops`)
Flux and ArgoCD sync status, drift detection, and reconciliation diagnostics. Always distinguishes desired state (Git) from actual state (cluster).

- *Try: "Are all Flux Kustomizations synced?" / "Find resources modified outside of GitOps"*

**🧹 Sanitizer** (`--agent sanitizer`)
Workload linting, best-practice scoring, and cluster health grading. Scores workloads against rules covering probes, resource limits, image tags, replica counts, and container hygiene. Produces an A–F grade with prioritised remediation steps.

- *Try: "Sanitize my cluster and give me a grade" / "Which workloads are missing health probes?"*

All specialist agents always use the premium model for the best reasoning quality. Agents and execution modes are independent — read-only protection is always enforced regardless of active agent.

See the [Agents documentation](https://github.com/e9169/kopilot/blob/main/docs/AGENTS.md) for full details.

### 🔌 MCP Servers

Kopilot supports connecting to external [Model Context Protocol (MCP)](https://modelcontextprotocol.io) servers, allowing it to call tools from any MCP-compatible service.

**Configure at startup:**

```bash
kopilot --mcp-config ~/.kopilot/mcp.json
```

The config file is a JSON object with a `servers` array. Each server entry must include `name`, `type`, and `url`:

```json
{
  "servers": [
    {
      "name": "my-server",
      "type": "http",
      "url": "http://localhost:8080/mcp"
    }
  ]
}
```

**Manage servers at runtime:**

```text
❯ /mcp list                          # list configured servers
❯ /mcp add my-server http://host/mcp # add or update a server
❯ /mcp delete my-server              # remove a server
```

Changes take effect immediately — no restart required.

### 📎 File Attachments and Shell Shortcuts

Attach files directly to a message using the `@` prefix:

```text
❯ @deployment.yaml what's wrong with this deployment?
❯ @logs.txt summarize these error logs
```

Run shell commands without involving AI using `!`:

```text
❯ !kubectl get pods -A
❯ !helm list
```

### 🔐 Security

- All credentials stay local
- Audit logs for all operations
- Role-based access control integration
- Queries processed through GitHub Copilot's secure infrastructure

---

## Advanced Topics

### Intelligent Model Selection

Kopilot automatically selects the optimal model based on your query:

- **Simple queries** (list, status, health) → Cost-effective model (default: gpt-4.1)
- **Complex operations** (troubleshooting, scaling, debugging) → Premium model (default: claude-sonnet-4.6)

This provides 50-70% cost reduction while maintaining quality for critical tasks. You can customize which models are used via environment variables.

See the [Model Selection documentation](https://github.com/e9169/kopilot/blob/main/docs/MODEL_SELECTION.md) for more details.

### User Interface

Kopilot features a modern GitHub Copilot CLI-inspired interface:

- **ASCII Logo** - Stylized Kopilot branding on startup
- **Dynamic Suggestions** - Random example prompts to help you get started
- **Smart Prompts** - Clean ❯ prompt for intuitive interaction
- **Color-Coded Status** - Visual indicators for health, quota, and modes

The interface automatically displays your connection status, current execution mode, and helpful examples each time you start a session.

---

## Examples

### Deploy a Simple Application

```bash
kopilot "deploy a redis instance with persistence"
```

### Debug a Failing Pod

```bash
kopilot "why is my nginx pod failing?"
```

### Optimize Resources

```bash
kopilot "analyze resource usage and suggest improvements"
```

### Backup Configuration

```bash
kopilot "export all configmaps in the production namespace"
```

---

## Troubleshooting

### Common Issues

#### "Cannot connect to cluster"

- Ensure your kubeconfig is properly configured
- Check that `kubectl` works: `kubectl get nodes`

#### "GitHub Copilot not authenticated"

- Run: `copilot auth login`
- Ensure you have an active GitHub Copilot subscription

#### "Model not responding"

- Check your internet connection
- Ensure GitHub Copilot CLI is authenticated: run `copilot auth status` (or re-run `copilot auth login`)
- Try a different model

### Getting Help

- [GitHub Issues](https://github.com/e9169/kopilot/issues)
- [Documentation](https://e9169.github.io/kopilot)

---

## Contributing

We welcome contributions! See [CONTRIBUTING.md](https://github.com/e9169/kopilot/blob/main/CONTRIBUTING.md) for guidelines.

## License

Kopilot is open source software licensed under the MIT License.
