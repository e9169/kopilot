#!/bin/bash
# Compare environments - compares dev/staging/prod clusters

set -euo pipefail

echo "🔄 Environment Comparison"
echo "========================="
echo ""

DEV_CONTEXT="${DEV_CONTEXT:-development}"
STAGING_CONTEXT="${STAGING_CONTEXT:-staging}"
PROD_CONTEXT="${PROD_CONTEXT:-production}"

echo "Comparing environments:"
echo "  Development: $DEV_CONTEXT"
echo "  Staging: $STAGING_CONTEXT"
echo "  Production: $PROD_CONTEXT"
echo ""

kopilot <<EOF
compare $DEV_CONTEXT, $STAGING_CONTEXT, $PROD_CONTEXT
/quit
EOF
