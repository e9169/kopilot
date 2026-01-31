#!/bin/bash
# Batch status check - gets status for all clusters and saves to file

set -euo pipefail

OUTPUT_FILE="${1:-cluster-status-$(date +%Y%m%d-%H%M%S).txt}"

echo "Kopilot Batch Status Check"
echo "==========================="
echo ""
echo "Output file: $OUTPUT_FILE"
echo ""

# Run kopilot and capture output
kopilot -v <<EOF | tee "$OUTPUT_FILE"
check the health of all clusters
/quit
EOF

echo ""
echo "Status saved to: $OUTPUT_FILE"
