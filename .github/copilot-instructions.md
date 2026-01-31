# GitHub Copilot Instructions for Kopilot

This document provides context and guidelines for GitHub Copilot when working with the Kopilot codebase.

## Project Overview

**Kopilot** is a Kubernetes cluster management agent built with the official GitHub Copilot SDK for Go. It provides an interactive natural language interface for monitoring, querying, and managing Kubernetes clusters.

### Core Purpose
- Monitor Kubernetes clusters from kubeconfig
- Execute kubectl commands through natural language
- Provide health monitoring with parallel execution
- Intelligent cost optimization via model selection

### Technology Stack
- **Language**: Go 1.23+
- **Framework**: GitHub Copilot SDK (`github.com/github/copilot-sdk/go@v0.1.18`)
- **Kubernetes**: `k8s.io/client-go@v0.31.4`
- **Target Platforms**: Linux, macOS, Windows (amd64/arm64)

## Project Structure

```
kopilot/
├── main.go              # Entry point, CLI flags, version management
├── pkg/
│   ├── agent/          # Copilot agent implementation
│   │   ├── agent.go    # Core agent logic, model selection
│   │   ├── tools.go    # Copilot SDK tools (functions)
│   │   └── *_test.go   # Unit tests
│   └── k8s/            # Kubernetes client wrapper
│       ├── provider.go # K8s cluster management
│       ├── collectors.go # Data collection from clusters
│       └── types.go    # K8s data structures
├── docs/               # Detailed documentation
└── website/            # Jekyll website (builds to website/_site/)
```

## Coding Standards

### Go Conventions
1. **Package Comments**: Every package must have a doc comment (see `pkg/agent/agent.go`)
2. **Exported Symbols**: All exported functions, types, and constants need doc comments
3. **Error Handling**: Always wrap errors with context using `fmt.Errorf`
4. **Formatting**: Use `go fmt` - enforced by CI and git hooks
5. **Idioms**: Follow standard Go idioms and effective Go practices

### Project-Specific Patterns

#### 1. Copilot SDK Tool Definition
Use the SDK's `DefineTool` pattern for type-safe tool registration:

```go
// Define parameter and result structs
type MyToolParams struct {
    FieldName string `json:"field_name" jsonschema:"description=Field description"`
}

type MyToolResult struct {
    Output string
}

// Register with SDK
myTool := copilot.DefineTool("tool_name", "Tool description", 
    func(params MyToolParams) (MyToolResult, error) {
        // Implementation
    })
```

**Key points:**
- Use JSON struct tags for field names
- Use `jsonschema` tag for parameter descriptions
- Return typed result structs, not raw strings

#### 2. Execution Modes
The agent has two execution modes for kubectl operations:
- `ModeReadOnly`: Blocks write operations (default, safe)
- `ModeInteractive`: Prompts for confirmation on write operations

**Always check mode before executing write operations:**
```go
if strings.Contains(args, "apply|create|delete|patch|edit") {
    if state.mode == ModeReadOnly {
        return result, fmt.Errorf("write operation blocked in read-only mode")
    }
    // Handle interactive confirmation...
}
```

#### 3. Model Selection Strategy
Kopilot uses intelligent model selection for cost optimization:

**Cost-effective model (`gpt-4o-mini`)** - for simple queries:
- Listing clusters
- Getting status
- Health checks
- Comparing clusters

**Premium model (`gpt-4o`)** - for complex tasks:
- Troubleshooting
- kubectl execution
- Problem diagnosis
- Complex queries

Use `selectModelForPrompt()` function to determine the appropriate model based on prompt analysis.

#### 4. Parallel Execution
When checking multiple clusters, use goroutines with sync.WaitGroup:

```go
var wg sync.WaitGroup
for _, cluster := range clusters {
    wg.Add(1)
    go func(c string) {
        defer wg.Done()
        // Check cluster...
    }(cluster)
}
wg.Wait()
```

#### 5. Output Formatting
- Use **Markdown tables** for structured data presentation
- Support both `OutputText` and `OutputJSON` formats
- Include ANSI colors for terminal output (via `color*` constants)
- Show quota information with color-coded indicators

#### 6. Version Management
- Version is set via git tags, not hardcoded
- Build uses ldflags: `-X main.version=$(VERSION)`
- Default fallback is `"dev"` for development builds
- Makefile handles version extraction: `VERSION := $(shell git describe --tags --always --dirty)`

## Testing Guidelines

### Test Structure
- Place tests in `*_test.go` files alongside source
- Use table-driven tests where appropriate
- Mock external dependencies (Kubernetes API)

### Test Commands
```bash
make test              # Unit tests
make test-integration  # Integration tests (requires kubeconfig)
make test-all         # Both unit and integration
make coverage         # Generate coverage report
```

### CI/CD
- Tests run on Ubuntu (linux/amd64) and macOS (darwin/arm64)
- All 6 platform combinations are cross-compiled
- Pre-commit hooks run tests automatically (optional setup)

## Common Tasks

### Adding a New Tool
1. Define parameter and result structs in `pkg/agent/tools.go`
2. Implement the tool function
3. Register with `copilot.DefineTool`
4. Add to tools slice in `Run()` function
5. Write tests in `pkg/agent/agent_test.go`
6. Update documentation if needed

### Adding Kubernetes Functionality
1. Add data collection logic to `pkg/k8s/collectors.go`
2. Update types in `pkg/k8s/types.go` if needed
3. Expose via `Provider` interface in `pkg/k8s/provider.go`
4. Use in agent tools via `state.k8sProvider`

### Documentation Updates
- Update `README.md` for user-facing changes
- Update `docs/` files for detailed documentation
- Follow existing documentation style and structure
- Keep website content in `website/` directory (Jekyll builds to `website/_site/`)

## Important Notes

### DO
✅ Follow conventional commit messages (enables auto-changelog)
✅ Add tests for new functionality
✅ Check execution mode before write operations
✅ Use type-safe SDK tool definitions
✅ Consider model selection for cost optimization
✅ Handle errors with context
✅ Format code with `go fmt`
✅ Update documentation

### DON'T
❌ Hardcode version numbers (use git tags)
❌ Execute write operations without mode checks
❌ Use raw strings for tool responses (use typed structs)
❌ Skip error handling or return bare errors
❌ Commit without running tests (use git hooks)
❌ Build Jekyll website at project root (use `website/` directory only)
❌ Forget to update docs for user-facing changes

## Keywords for Context

When suggesting code, consider these key concepts:
- **Copilot SDK**: Tools, agents, model selection
- **Kubernetes**: kubeconfig, contexts, clusters, nodes, pods
- **Execution modes**: read-only, interactive, confirmations
- **Cost optimization**: model selection, token usage
- **Parallel execution**: goroutines, sync patterns
- **Type safety**: structs, interfaces, error handling
- **CLI**: flags, commands, version info

## Resources

- [Copilot SDK Documentation](https://github.com/github/copilot-sdk)
- [Project Documentation](../docs/)
- [Contributing Guide](../CONTRIBUTING.md)
- [Model Selection](../docs/MODEL_SELECTION.md)
- [Testing Guide](../docs/TESTING.md)

---

**Note**: This project was created as an AI-powered experiment. Contributions to improve quality and production-readiness are welcome!
