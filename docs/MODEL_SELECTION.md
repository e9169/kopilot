# Intelligent Model Selection Strategy

## Overview

Kopilot now uses intelligent model selection to optimize costs while maintaining high quality for complex operations. The system automatically chooses the appropriate AI model based on the user's query complexity and intent.

## Model Strategy

### Cost-Effective Model: `gpt-4o-mini`

**Use Cases:**

- Initial health checks and status monitoring
- Simple queries (list, show, get, describe, status, health)
- Basic information retrieval ("what", "how many", "check")
- Routine cluster operations

**Benefits:**

- Lower cost per request (often free tier)
- Fast response times
- Sufficient for straightforward tasks

### Premium Model: `gpt-4o`

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

1. **Initial Session**: Starts with `gpt-4o-mini` for health checks
2. **Query Analysis**: Each user query is analyzed for complexity
3. **Session Recreation**: When a different model is needed, the session is recreated with the optimal model
4. **Transparent Switching**: Model switches are logged but don't interrupt the conversation

### Code Flow

```go
// Determine optimal model based on query
optimalModel := selectModelForQuery(userInput)

// Create new session if model needs to change
if optimalModel != currentModel {
    newSession := createSessionWithModel(client, k8sProvider, optimalModel)
    currentSession = newSession
    currentModel = optimalModel
}
```

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
2024/01/15 10:30:45 Switching from gpt-4o-mini to gpt-4o for query complexity
2024/01/15 10:30:45 Session created with model: gpt-4o
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
func selectModelForQuery(query string) string {
    // Add or modify keyword lists
    // Adjust model selection logic
    // Return appropriate model name
}
```
