#!/bin/bash
# Health check script - monitors all clusters and reports issues

set -euo pipefail

echo "🔍 Kopilot Health Check Script"
echo "================================"
echo ""

# Check if kopilot is installed
if ! command -v kopilot &> /dev/null; then
    echo "❌ Error: kopilot not found in PATH"
    echo "Install it with: make install"
    exit 1
fi

# Check if kubeconfig exists
KUBECONFIG_PATH="${KUBECONFIG:-$HOME/.kube/config}"
if [ ! -f "$KUBECONFIG_PATH" ]; then
    echo "❌ Error: kubeconfig not found at $KUBECONFIG_PATH"
    exit 1
fi

echo "📊 Running health check on all clusters..."
echo ""

# Run kopilot with JSON output for parsing
OUTPUT=$(kopilot --output json <<EOF
check all clusters
/quit
EOF
)

# Parse and display results
echo "$OUTPUT" | jq -r '.clusters[] | "Cluster: \(.name) - Status: \(.status) - Health: \(.healthy_nodes)/\(.total_nodes) nodes"'

echo ""
echo "✅ Health check complete"
