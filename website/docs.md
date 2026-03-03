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

- **Cost-effective model** (default: gpt-4o-mini) - Used for simple queries and status checks
- **Premium model** (default: gpt-4o) - Automatically selected for troubleshooting and complex operations

**Customization:**

You can override the default models by setting environment variables:

```bash
# Example: Use different models
export KOPILOT_MODEL_COST_EFFECTIVE="claude-3.5-sonnet"
export KOPILOT_MODEL_PREMIUM="o1-preview"

# Then run kopilot
./bin/kopilot
```

Kopilot supports any model available in your GitHub Copilot plan, including: `gpt-4o`, `gpt-4o-mini`, `o1-preview`, `o1-mini`, `claude-3.5-sonnet`, and others.

No API key configuration needed - authentication is handled through GitHub Copilot CLI.

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
❯ /mode             # Show current mode
```

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

#### 🔒 Read-Only Mode (Default)

- Blocks all write operations for maximum safety
- Allows monitoring, querying, and troubleshooting
- Best for production environments
- No risk of accidental changes

#### 🔓 Interactive Mode

- Enables write operations with confirmation
- Shows exact command before execution
- Requires explicit approval (yes/no)
- Can be started with `--interactive` flag or switched at runtime with `/interactive`

#### Runtime Commands

- `/help` - Show all available commands
- `/readonly` - Switch to read-only mode
- `/interactive` - Switch to interactive mode
- `/mode` - Show current execution mode
- `/agent` or `/agent list` - Show active agent and roster
- `/agent <name>` - Switch specialist agent

See the [Execution Modes documentation](https://github.com/e9169/kopilot/blob/main/docs/EXECUTION_MODES.md) for detailed information

### 🎭 Specialist Agent Personas

Kopilot ships four domain-focused AI personas that sharpen the assistant for specific operational areas. Start with `--agent <name>` or switch mid-session with `/agent <name>`.

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

All specialist agents always use the premium model for the best reasoning quality. Agents and execution modes are independent — read-only protection is always enforced regardless of active agent.

See the [Agents documentation](https://github.com/e9169/kopilot/blob/main/docs/AGENTS.md) for full details.

### 🔐 Security

- All credentials stay local
- Audit logs for all operations
- Role-based access control integration
- Queries processed through GitHub Copilot's secure infrastructure

---

## Advanced Topics

### Intelligent Model Selection

Kopilot automatically selects the optimal model based on your query:

- **Simple queries** (list, status, health) → Cost-effective model (default: gpt-4o-mini)
- **Complex operations** (troubleshooting, scaling, debugging) → Premium model (default: gpt-4o)

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
