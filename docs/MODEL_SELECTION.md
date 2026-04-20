# Intelligent Model Selection Strategy

## Overview

Kopilot now uses intelligent model selection to optimize costs while maintaining high quality for complex operations. The system automatically chooses the appropriate AI model based on the user's query complexity and intent.

## Model Strategy

### Cost-Effective Model: `gpt-4.1`

**Use Cases:**

- Initial health checks and status monitoring
- Simple queries (list, show, get, describe, status, health)
- Basic information retrieval ("what", "how many", "check")
- Routine cluster operations

**Benefits:**

- Lower cost per request (often free tier)
- Fast response times
- Sufficient for straightforward tasks

### Premium Model: `claude-sonnet-4.6`

**Use Cases:**

- Troubleshooting and debugging
- Investigation and analysis
- Complex kubectl operations (scale, restart, delete, apply, patch, edit, rollback, drain, cordon, taint)
- Error diagnosis and problem solving

**Trigger Keywords:**

- Troubleshooting: "why", "troubleshoot", "debug", "investigate", "error", "fail", "crash", "not working", "broken", "issue", "problem", "wrong"
- Analysis: "fix", "solve", "diagnose", "analyze", "explain", "understand"
- Complex operations: "scale", "restart", "delete", "apply", "patch", "edit", "rollback", "drain", "cordon", "taint"

**Benefits:**

- Superior reasoning capabilities
- Better context understanding
- More accurate troubleshooting
- Higher success rate for complex operations

## Implementation

The model selection is dynamic and happens automatically:

1. **Initial Session**: Starts with `gpt-4.1` for health checks
2. **Agent Check**: If a specialist agent is active (`debugger`, `security`, `optimizer`, `gitops`), always use the premium model regardless of query text
3. **Query Analysis**: For the `default` agent, each user query is analyzed for complexity
4. **Session Recreation**: When a different model is needed, the session is recreated with the optimal model
5. **Transparent Switching**: Model switches are logged but don't interrupt the conversation

### Code Flow

```go
// Determine optimal model based on query and active agent
optimalModel := selectModelForQuery(userInput, currentAgent)

// Create new session if model needs to change
if optimalModel != currentModel {
    newSession := createSessionWithModel(client, k8sProvider, optimalModel)
    currentSession = newSession
    currentModel = optimalModel
}
```

## Agent-Aware Model Selection

Specialist agents (`debugger`, `security`, `optimizer`, `gitops`) always use the premium model, even for queries that would normally be considered simple (e.g., "list pods"). This is intentional: specialist reasoning benefits from the higher capacity of the premium model — even a routine "show events" query issued through the Debugger persona may require deep contextual analysis.

The `default` agent uses the keyword-based heuristic described above.

| Agent | Model strategy |
| ------- | --------------- |
| `default` | Dynamic — cost-effective or premium based on query keywords |
| `debugger` | Always premium |
| `security` | Always premium |
| `optimizer` | Always premium |
| `gitops` | Always premium |

## Cost Optimization Results

By using this strategy:

- **Simple operations** (≈70% of queries): Use free/cheaper model
- **Complex operations** (≈30% of queries): Use premium model only when needed
- **Estimated cost reduction**: 50-70% compared to always using premium model
- **Quality maintained**: Premium model automatically engaged for critical tasks

## Examples

### Cheap Model Queries

```text
> show me all clusters
> check the health status
> list pods in default namespace
> what clusters are available?
> get node status for production
```

### Premium Model Queries

```text
> why is my pod crashlooping?
> investigate the failed deployment
> troubleshoot the connection error
> explain why the service is unreachable
> help me debug this issue
> scale the deployment safely
```

## Monitoring

Model switches are logged:

```text
2024/01/15 10:30:45 Switching from gpt-4.1 to claude-sonnet-4.6 for query complexity
2024/01/15 10:30:45 Session created with model: claude-sonnet-4.6
```

## Future Enhancements

Potential improvements to the model selection strategy:

1. **Usage Analytics**: Track model usage patterns and optimize keywords
2. **User Preferences**: Allow users to override model selection
3. **Context Awareness**: Consider conversation history in model selection
4. **Cost Tracking**: Display cost estimates per query
5. **Adaptive Learning**: Adjust keyword weights based on actual complexity

## Configuration

Currently, model selection is automatic. To modify the strategy, edit the `selectModelForQuery()` function in [pkg/agent/agent.go](pkg/agent/agent.go):

```go
// selectModelForQuery determines the best model based on query complexity,
// intent, and the active agent type.
func selectModelForQuery(query string, agentType AgentType) string {
    // Specialist agents always use the premium model
    if def, ok := agentDefinitions[agentType]; ok && def.preferPremium {
        return modelPremium
    }
    // Add or modify keyword lists for the default agent
    // Adjust model selection logic
    // Return appropriate model name
}
```
