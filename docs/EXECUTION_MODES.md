# Execution Modes

Kopilot provides flexible execution modes to balance convenience with safety when executing kubectl commands.

## Overview

By default, Kopilot runs in **read-only mode** to prevent accidental modifications to your Kubernetes clusters. This is especially important when using AI to execute commands, as misinterpreted natural language could lead to unintended changes.

## Modes

### 🔒 Read-Only Mode (Default)

**Purpose**: Maximum safety for production environments and exploratory use.

**Behavior**:
- Blocks all write operations
- Allows read-only kubectl commands (get, describe, logs, etc.)
- Returns clear error message when write operation is attempted
- No confirmation prompts needed

**Allowed Commands**:
- `get` - List resources
- `describe` - Show detailed information
- `logs` - View container logs
- `top` - Display resource usage
- `explain` - Documentation for resource types
- `api-resources` - List available API resources
- `api-versions` - List available API versions
- `cluster-info` - Display cluster information
- `version` - Show client and server versions
- `config` - View kubeconfig settings
- `diff` - Show differences
- `auth` - Authentication commands

**Blocked Commands**:
- `scale`, `delete`, `apply`, `patch`, `edit`
- `create`, `replace`, `rollout`
- `drain`, `cordon`, `uncordon`, `taint`
- Any other command that modifies cluster state

**Example**:
```bash
# Start in read-only mode (default)
./bin/kopilot

> show me all pods in production
✅ Executes: kubectl get pods -A

> delete the failing pod
❌ Blocked: Write operation not allowed in read-only mode
Use /interactive to enable write operations
```

### 🔓 Interactive Mode

**Purpose**: Safe execution of write operations with explicit user confirmation.

**Behavior**:
- Allows all kubectl commands
- Shows the exact command before execution
- Requires user confirmation (yes/no) for write operations
- Read-only commands execute immediately without confirmation
- Clear distinction between read and write operations

**Example**:
```bash
# Start in interactive mode
./bin/kopilot --interactive

> scale nginx deployment to 5 replicas

⚠️  Write Operation: kubectl --context prod-cluster scale deployment nginx --replicas=5
This will modify the cluster state.
Do you want to proceed? (yes/no): yes

⚡ Executing: kubectl --context prod-cluster scale deployment nginx --replicas=5
deployment.apps/nginx scaled
```

**Cancellation**:
```bash
> delete old pods in namespace test

⚠️  Write Operation: kubectl --context prod-cluster delete pods -l app=old -n test
This will modify the cluster state.
Do you want to proceed? (yes/no): no

❌ Operation cancelled by user
```

## Runtime Mode Switching

You can switch execution modes during a session without restarting kopilot:

### Commands

- `/readonly` - Switch to read-only mode
- `/interactive` - Switch to interactive mode  
- `/mode` or `/status` - Display current mode

### Example Session

```bash
./bin/kopilot --interactive

🔓 Interactive mode: Write operations require confirmation

> /readonly
🔒 Switched to read-only mode
Write operations are now blocked. Use /interactive to enable writes.

> scale deployment nginx to 3
❌ Blocked: Write operation not allowed in read-only mode

> /interactive
🔓 Switched to interactive mode
Write operations now require confirmation.

> scale deployment nginx to 3
⚠️  Write Operation: kubectl scale deployment nginx --replicas=3
Do you want to proceed? (yes/no): yes
✓ Executed successfully
```

## Startup Options

### Read-Only (Default)

```bash
# These are equivalent
./bin/kopilot
```

### Interactive

```bash
./bin/kopilot --interactive
```

## Use Cases

### Read-Only Mode

**Best for**:
- Production environment monitoring
- Cluster health checks
- Troubleshooting and investigation
- Learning and exploration
- CI/CD pipelines (read-only checks)
- Shared team access with restricted permissions
- Demo and presentation environments

**Example Scenarios**:
- "Check if all pods are running"
- "Show me logs for the api service"
- "What's the status of nodes in prod?"
- "Describe the failing deployment"

### Interactive Mode

**Best for**:
- Operational tasks requiring modifications
- Scaling applications
- Restarting failed pods
- Applying configuration changes
- Debugging with rollbacks
- Controlled maintenance windows

**Example Scenarios**:
- "Scale the web service to handle more traffic"
- "Restart the crashed pods"
- "Update the ConfigMap and rollout restart"
- "Drain this node for maintenance"

## Safety Features

### Explicit Command Display

Before executing any write operation in interactive mode, kopilot shows:
1. The exact kubectl command that will run
2. The cluster context being targeted
3. A warning that cluster state will be modified

### Clear Blocking Messages

In read-only mode, blocked operations show:
1. The attempted command
2. Why it was blocked
3. How to enable write operations if needed

### Mode Indicators

Visual feedback shows current mode at all times:
- 🔒 icon for read-only mode
- 🔓 icon for interactive mode
- Mode name displayed in startup banner
- Mode-specific instructions on launch

## Best Practices

1. **Start with Read-Only**: Unless you specifically need to make changes, use read-only mode
2. **Review Before Confirming**: Always read the displayed command carefully before typing "yes"
3. **Use Specific Contexts**: Be explicit about which cluster you're targeting
4. **Switch Modes Intentionally**: Use runtime commands to switch modes only when needed
5. **Exit After Operations**: Close the session after completing write operations in interactive mode

## Implementation Details

The execution mode system:
- Validates commands against a whitelist of read-only kubectl commands
- Requires user input (buffered reader) for confirmation in interactive mode
- Returns descriptive errors with guidance when blocking operations
- Maintains state throughout the session for consistent behavior
- Can be changed at runtime without recreating the AI session

## Testing

Execution modes are fully tested with:
- Unit tests for command classification
- Mode switching behavior validation
- State management verification
- See `pkg/agent/agent_test.go` for test implementation
