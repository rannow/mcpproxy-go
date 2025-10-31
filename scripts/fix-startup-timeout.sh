#!/bin/bash
# Quick Fix: Reduce Concurrency to Prevent Startup Timeouts

set -e

CONFIG_FILE="$HOME/.mcpproxy/mcp_config.json"
BACKUP_FILE="$HOME/.mcpproxy/backups/config-before-concurrency-fix-$(date +%Y%m%d-%H%M%S).json"

echo "🔧 MCPProxy Startup Timeout Fix"
echo "================================"
echo ""

# Backup config
mkdir -p "$(dirname "$BACKUP_FILE")"
cp "$CONFIG_FILE" "$BACKUP_FILE"
echo "✅ Config backed up to: $BACKUP_FILE"
echo ""

# Check current setting
CURRENT=$(jq -r '.max_concurrent_connections // "NOT_SET (default: 20)"' "$CONFIG_FILE")
echo "Current concurrency: $CURRENT"
echo ""

# Apply fix
echo "Applying fix: max_concurrent_connections = 5"
jq '.max_concurrent_connections = 5' "$CONFIG_FILE" > "$CONFIG_FILE.tmp"
mv "$CONFIG_FILE.tmp" "$CONFIG_FILE"
echo "✅ Configuration updated"
echo ""

# Verify
NEW=$(jq -r '.max_concurrent_connections' "$CONFIG_FILE")
echo "New concurrency: $NEW"
echo ""

echo "═══════════════════════════════════════════"
echo "✅ Fix Applied Successfully!"
echo ""
echo "Next steps:"
echo "1. Restart mcpproxy:"
echo "   pkill mcpproxy"
echo "   ./mcpproxy serve"
echo ""
echo "2. Monitor startup:"
echo "   tail -f ~/Library/Logs/mcpproxy/main.log | grep 'ConnectAll\\|Successfully connected'"
echo ""
echo "Expected behavior:"
echo "  - Servers connect in waves of 5"
echo "  - Each wave takes 30-60 seconds"
echo "  - Total startup time: 2-3 minutes"
echo "  - Success rate: >90%"
echo "═══════════════════════════════════════════"
