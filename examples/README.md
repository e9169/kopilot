# Kopilot Examples

This directory contains example configurations and usage patterns for Kopilot.

## Contents

- [kubeconfig-samples/](kubeconfig-samples/) - Example kubeconfig configurations
- [scripts/](scripts/) - Helper scripts for common tasks
- [prompts/](prompts/) - Example prompts for interacting with Kopilot

## Quick Start Examples

### Basic Usage

```bash
# Start Kopilot in read-only mode (default)
kopilot

# Start with verbose logging
kopilot -v

# Start in interactive mode (allows write operations with confirmation)
kopilot --interactive

# Use a custom kubeconfig file
KUBECONFIG=./examples/kubeconfig-samples/development.yaml kopilot
```

### Example Prompts

Once Kopilot is running, try these natural language commands:

```
# List all clusters
> show me all available clusters

# Check cluster health
> check the health of all clusters

# Get detailed status
> what's the status of my production cluster?

# Compare clusters
> compare development and staging clusters

# kubectl operations (requires interactive mode)
> scale the frontend deployment to 3 replicas in the production cluster
> show me all pods in the kube-system namespace
> describe the failing pod in default namespace
```

### Runtime Commands

While Kopilot is running, you can use these special commands:

```
/readonly    - Switch to read-only mode
/interactive - Switch to interactive mode  
/mode        - Show current execution mode
/help        - Show available commands
/quit or /exit - Exit Kopilot
```

## Configuration Examples

### Environment Variables

```bash
# Custom kubeconfig location
export KUBECONFIG=/path/to/custom/kubeconfig

# Custom model selection (advanced)
export KOPILOT_MODEL_COST_EFFECTIVE=gpt-4o-mini
export KOPILOT_MODEL_PREMIUM=gpt-4o

# Debug logging
export KOPILOT_DEBUG=true
```

### Output Formats

```bash
# Default text output with colors and formatting
kopilot --output text

# JSON output for scripting/parsing
kopilot --output json
```

## Common Workflows

### Development Workflow

```bash
# 1. Check all cluster statuses
kopilot
> check all clusters

# 2. Review any issues
> show me unhealthy pods across all clusters

# 3. Investigate specific cluster
> get detailed status for development cluster

# 4. Take action (interactive mode)
kopilot --interactive
> restart the crashlooping pod in development
```

### Production Monitoring

```bash
# Safe monitoring (read-only mode)
kopilot -v
> check health of production cluster
> compare production with staging
> show me resource usage in production
```

### Multi-Cluster Management

```bash
# View all clusters side-by-side
> compare all clusters

# Check specific clusters
> compare cluster1, cluster2, cluster3

# Focus on production environments
> show me all clusters with 'prod' in the name
```

## Tips and Best Practices

1. **Start in Read-Only Mode**: Default mode prevents accidental changes
2. **Use Interactive Mode for Changes**: Explicit confirmation for write operations
3. **Leverage Natural Language**: Describe what you want in plain English
4. **Check Status First**: Always review cluster status before making changes
5. **Use Verbose Mode**: `-v` flag for troubleshooting and detailed logging

## See Also

- [Main README](../README.md)
- [Execution Modes Documentation](../docs/EXECUTION_MODES.md)
- [Interactive Features](../docs/INTERACTIVE_FEATURES.md)
- [Testing Guide](../docs/TESTING.md)
