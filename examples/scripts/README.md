# Example Scripts

Useful shell scripts for common Kopilot workflows.

## Available Scripts

### health-check.sh
Automated health monitoring for all clusters.

```bash
./examples/scripts/health-check.sh
```

**Features:**
- Checks all clusters in kubeconfig
- Reports node health status
- JSON output for parsing
- Exit codes for automation

### batch-status.sh
Batch status check with output saved to file.

```bash
./examples/scripts/batch-status.sh [output-file]
```

**Features:**
- Saves status to timestamped file
- Verbose output for debugging
- Suitable for scheduled jobs

**Example:**
```bash
# Save to default timestamped file
./examples/scripts/batch-status.sh

# Save to specific file
./examples/scripts/batch-status.sh daily-status.txt
```

### compare-environments.sh
Compare development, staging, and production clusters.

```bash
./examples/scripts/compare-environments.sh
```

**Customization:**
```bash
# Override context names
DEV_CONTEXT=dev STAGING_CONTEXT=stage PROD_CONTEXT=prod \\
  ./examples/scripts/compare-environments.sh
```

## Making Scripts Executable

```bash
chmod +x examples/scripts/*.sh
```

## Integration Examples

### Cron Job for Monitoring

```bash
# Add to crontab (crontab -e)
# Run health check every hour
0 * * * * /path/to/kopilot/examples/scripts/health-check.sh >> /var/log/kopilot-health.log 2>&1
```

### CI/CD Integration

```yaml
# GitHub Actions example
- name: Check Kubernetes Clusters
  run: |
    ./examples/scripts/health-check.sh
```

### Slack/Discord Notifications

```bash
#!/bin/bash
# health-notify.sh - Run health check and send to Slack

OUTPUT=$(./examples/scripts/health-check.sh 2>&1)
curl -X POST -H 'Content-type: application/json' \\
  --data "{\"text\":\"Cluster Health:\n\`\`\`$OUTPUT\`\`\`\"}" \\
  YOUR_WEBHOOK_URL
```

## Prerequisites

All scripts require:
- kopilot installed and in PATH
- Valid kubeconfig configured
- Bash 4.0+
- jq (for JSON parsing in some scripts)

## Customization

Feel free to modify these scripts for your needs:
- Add email notifications
- Integrate with monitoring systems
- Add custom health checks
- Filter specific clusters
- Format output differently

## See Also

- [Main README](../../README.md)
- [Examples README](../README.md)
