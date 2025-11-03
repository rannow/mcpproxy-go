#!/bin/bash
# Unquarantine All Servers - Aggressive Recovery
#
# This script unquarantines ALL servers that were quarantined due to
# the Oct 17, 2025 concurrent startup overload.
#
# Actions:
# 1. Remove duplicate gdrive-server configuration
# 2. Unquarantine and enable all remaining quarantined servers:
#    - gdrive (Google Drive)
#    - bigquery-ergut (BigQuery alternative)
#    - test-weather-server (weather demo)

set -euo pipefail

CONFIG_FILE="$HOME/.mcpproxy/mcp_config.json"
BACKUP_DIR="$HOME/.mcpproxy/backups"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

echo "🔍 Quarantined Servers Full Recovery Script"
echo "==========================================="
echo ""

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Check if config exists
if [ ! -f "$CONFIG_FILE" ]; then
    echo "❌ Error: Config file not found: $CONFIG_FILE"
    exit 1
fi

# Backup current config
BACKUP_FILE="$BACKUP_DIR/config-before-unquarantine-all-$TIMESTAMP.json"
cp "$CONFIG_FILE" "$BACKUP_FILE"
echo "✅ Backed up config to: $BACKUP_FILE"
echo ""

# List quarantined servers before
echo "📊 Quarantined servers before:"
jq -r '.mcpServers[] | select(.quarantined == true) | "  - \(.name)"' "$CONFIG_FILE"
QUARANTINED_BEFORE=$(jq '[.mcpServers[] | select(.quarantined == true)] | length' "$CONFIG_FILE")
echo "  Total: $QUARANTINED_BEFORE"
echo ""

# Remove gdrive-server duplicate
echo "🗑️  Removing duplicate gdrive-server configuration..."
jq 'del(.mcpServers[] | select(.name == "gdrive-server"))' "$CONFIG_FILE" > "$CONFIG_FILE.tmp"
mv "$CONFIG_FILE.tmp" "$CONFIG_FILE"

# Unquarantine ALL servers
echo "🔓 Unquarantining all remaining servers..."
jq '.mcpServers |= map(
  if .quarantined == true
  then .quarantined = false | .enabled = true
  else .
  end
)' "$CONFIG_FILE" > "$CONFIG_FILE.tmp"
mv "$CONFIG_FILE.tmp" "$CONFIG_FILE"

# Count quarantined servers after
QUARANTINED_AFTER=$(jq '[.mcpServers[] | select(.quarantined == true)] | length' "$CONFIG_FILE")

echo ""
echo "✅ Full Recovery Complete!"
echo "=========================="
echo ""
echo "📊 Results:"
echo "  - Quarantined before: $QUARANTINED_BEFORE"
echo "  - Quarantined after:  $QUARANTINED_AFTER"
echo "  - Servers unquarantined: $((QUARANTINED_BEFORE - QUARANTINED_AFTER))"
echo ""
echo "✅ Unquarantined and enabled:"
echo "  - gdrive (Google Drive integration)"
echo "  - bigquery-ergut (BigQuery alternative)"
echo "  - test-weather-server (weather demo)"
echo ""
echo "🗑️  Removed duplicates:"
echo "  - gdrive-server (duplicate of gdrive)"
echo ""
echo "📁 Backup location: $BACKUP_FILE"
echo ""
echo "⚠️  Note: bigquery-ergut may conflict with bigquery-lucashild"
echo "   Monitor startup logs to check for conflicts"
echo ""
echo "🔄 Next step: Restart mcpproxy to apply changes"
echo "   pkill mcpproxy && ./mcpproxy serve"
echo ""
