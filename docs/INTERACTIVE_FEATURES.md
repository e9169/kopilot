# Interactive Features Implementation

## Overview

Kopilot has been enhanced with interactive and proactive capabilities, making it a conversational Kubernetes operations assistant that can execute kubectl commands on behalf of the user.

## Key Features Implemented

### 1. Proactive Cluster Health Monitoring

When kopilot starts, it automatically:

- Checks the status of ALL clusters in your kubeconfig
- Identifies and reports any issues (unreachable clusters, unhealthy nodes, connection problems)
- Provides a clear summary:
  - If issues found: Lists them with details
  - If all healthy: Confirms "All clusters are healthy ✓"
- Asks "What would you like me to do?" to prompt user interaction

**Example Output:**

```text
🚀 Kopilot - Kubernetes Cluster Assistant

## 📊 Cluster Health Summary

**10 of 11 clusters are healthy ✅**

### ⚠️  Issues Found:

**1. rpi cluster - UNREACHABLE**
   - Connection timeout to 192.168.1.158:6443
   - Cluster may be offline or network unreachable

### ✅ Healthy Clusters:

All 10 enterprise clusters are operational:
- SEML Region: dev-mgmt-01 (2 nodes), prod-wrk-01 (6 nodes), dev-wrk-01 (6 nodes), prod-mgmt-01 (6 nodes), prod-test-01 (6 nodes)
- US Region: dev-mgmt-01 (2 nodes), prod-test-01 (6 nodes), prod-wrk-01 (6 nodes), dev-wrk-01 (6 nodes), prod-mgmt-01 (6 nodes)

All nodes are in Ready status, all clusters running Kubernetes v1.31.4.

**What would you like me to do?**

> 
```

### 2. kubectl Command Execution

Kopilot can now execute kubectl commands through natural language requests using the new `kubectl_exec` tool.

**How it works:**

- User makes a request in natural language
- Kopilot determines the appropriate kubectl command
- Executes the command with the specified cluster context
- Shows the command and output to the user
- Interprets results

**Example Interactions:**

```text
> Show me pods in the default namespace on dev-mgmt-01
> Scale the nginx deployment to 3 replicas in prod-wrk-01
> Check the logs of the api pod in namespace apps on US prod-test-01
> What's the status of nodes in k8s-os-seml-dev-mgmt-01?
> Describe the service named frontend in prod-wrk-01
```

### 3. Interactive Conversation Loop

**Features:**

- Continuous conversation mode
- Natural language understanding
- Context awareness across requests
- Intelligent tool selection
- Streaming responses
- Type "exit" to quit

**User Experience:**

1. Start kopilot: `./bin/kopilot`
2. Wait for initial health check to complete
3. Interact naturally with requests
4. Type `exit` or press Ctrl+C to quit

### 4. Enhanced System Message

The AI assistant is configured with a comprehensive system message that:

- Defines its role as Kopilot, a Kubernetes operations assistant
- Lists its capabilities (monitoring, kubectl execution, health checks, etc.)
- Specifies proactive behavior (check clusters on startup)
- Provides guidance on tool usage
- Encourages conversational and helpful interactions

## Tools Available

| Tool | Description | Parameters |
| ---- | ----------- | ---------- |
| `list_clusters` | Lists all clusters from kubeconfig | None |
| `get_cluster_status` | Gets detailed status for a cluster | `context` (string) |
| `compare_clusters` | Compares multiple clusters side by side | `contexts` (array) |
| `kubectl_exec` | Executes kubectl commands | `context` (string), `args` (array) |

## Technical Implementation

### Agent Architecture

```go
// Key components in pkg/agent/agent.go

1. Run() function:
   - Creates Copilot client
   - Defines tools (including kubectl_exec)
   - Sets custom system message
   - Sends initial cluster health check prompt
   - Enters interactive loop

2. Event Handling:
   - assistant.message_delta: Stream response chunks
   - assistant.message: Complete message with fallback to full content
   - session.idle: Track when ready for next input
   - tool.execution_start: Log tool usage

3. Interactive Loop:
   - Read user input from stdin
   - Handle exit commands
   - Send messages to session
   - Wait for completion before next input
```

### kubectl_exec Tool

```go
type KubectlExecParams struct {
    Context string   `json:"context"`  // Cluster context name
    Args    []string `json:"args"`     // kubectl arguments
}

// Executes: kubectl --context <context> <args...>
// Returns: Command output and any errors
```

### Benefits

1. **Proactive Monitoring**: Users immediately know the state of their infrastructure
2. **Natural Language Operations**: No need to remember kubectl syntax
3. **Multi-Cluster Management**: Easy switching between clusters through conversation
4. **Safe Execution**: Commands are executed with explicit context specification
5. **Transparent Operations**: Shows exactly what commands are being run
6. **Interactive Experience**: Real-time streaming responses feel natural

## Testing

- ✅ 42 unit tests
- ✅ 3 integration tests
- ✅ 3 benchmarks
- ✅ All tests passing
- ✅ Race detector enabled

## Usage Example Session

```bash
$ ./bin/kopilot

🚀 Kopilot - Kubernetes Cluster Assistant

[Automatic health check runs...]

**All clusters are healthy ✅**

**What would you like me to do?**

> show me all pods in the kube-system namespace on dev-mgmt-01

[Executes: kubectl --context k8s-os-seml-dev-mgmt-01 get pods -n kube-system]

> are there any failed pods in prod-wrk-01?

[Executes: kubectl --context k8s-os-seml-prod-wrk-01 get pods --all-namespaces --field-selector=status.phase=Failed]

> scale the api deployment in prod-wrk-01 to 5 replicas

[Executes: kubectl --context k8s-os-seml-prod-wrk-01 scale deployment api --replicas=5]

> exit

Goodbye! 👋
```

## Future Enhancements (Ideas)

- Add history/undo functionality
- Support for multi-step operations (e.g., "restart all pods in namespace X")
- Cluster resource usage graphs/charts
- Alert/notification integration
- Configuration file for custom greetings/behavior
- Support for custom kubectl plugins
- Dry-run mode for destructive operations
- Auto-completion for cluster names and namespaces

## Files Modified

1. `pkg/agent/agent.go` - Added kubectl_exec tool, interactive loop, system message
2. `pkg/agent/agent_test.go` - Updated tests for 4 tools
3. `README.md` - Updated features and usage documentation
4. `INTERACTIVE_FEATURES.md` - This document

## Dependencies Added

- `os/exec` - For kubectl command execution
- `bufio` - For reading user input
- `strings` - For string processing

All existing functionality preserved, no breaking changes.
