#!/bin/bash
# Unquarantine Safe Servers - Conservative Recovery
#
# This script safely unquarantines servers that were quarantined due to
# the Oct 17, 2025 concurrent startup overload (not security issues).
#
# Actions:
# 1. Remove duplicate gdrive-server configuration
# 2. Unquarantine and enable gdrive (official Anthropic package)
# 3. Keep bigquery-ergut and test-weather-server quarantined (low priority)

set -euo pipefail

CONFIG_FILE="$HOME/.mcpproxy/mcp_config.json"
BACKUP_DIR="$HOME/.mcpproxy/backups"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

echo "🔍 Quarantined Servers Recovery Script"
echo "========================================"
echo ""

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Check if config exists
if [ ! -f "$CONFIG_FILE" ]; then
    echo "❌ Error: Config file not found: $CONFIG_FILE"
    exit 1
fi

# Backup current config
BACKUP_FILE="$BACKUP_DIR/config-before-unquarantine-$TIMESTAMP.json"
cp "$CONFIG_FILE" "$BACKUP_FILE"
echo "✅ Backed up config to: $BACKUP_FILE"
echo ""

# Count quarantined servers before
QUARANTINED_BEFORE=$(jq '[.mcpServers[] | select(.quarantined == true)] | length' "$CONFIG_FILE")
echo "📊 Quarantined servers before: $QUARANTINED_BEFORE"

# Remove gdrive-server duplicate
echo "🗑️  Removing duplicate gdrive-server configuration..."
jq 'del(.mcpServers[] | select(.name == "gdrive-server"))' "$CONFIG_FILE" > "$CONFIG_FILE.tmp"
mv "$CONFIG_FILE.tmp" "$CONFIG_FILE"

# Unquarantine and enable gdrive
echo "🔓 Unquarantining and enabling gdrive..."
jq '.mcpServers |= map(
  if .name == "gdrive"
  then .quarantined = false | .enabled = true
  else .
  end
)' "$CONFIG_FILE" > "$CONFIG_FILE.tmp"
mv "$CONFIG_FILE.tmp" "$CONFIG_FILE"

# Count quarantined servers after
QUARANTINED_AFTER=$(jq '[.mcpServers[] | select(.quarantined == true)] | length' "$CONFIG_FILE")

echo ""
echo "✅ Recovery Complete!"
echo "===================="
echo ""
echo "📊 Results:"
echo "  - Quarantined before: $QUARANTINED_BEFORE"
echo "  - Quarantined after:  $QUARANTINED_AFTER"
echo "  - Servers unquarantined: $((QUARANTINED_BEFORE - QUARANTINED_AFTER))"
echo ""
echo "✅ Enabled servers:"
echo "  - gdrive (Google Drive integration)"
echo ""
echo "🗑️  Removed duplicates:"
echo "  - gdrive-server (duplicate of gdrive)"
echo ""
echo "⚠️  Still quarantined (low priority):"
echo "  - bigquery-ergut (alternative bigquery-lucashild already enabled)"
echo "  - test-weather-server (demo/test server)"
echo ""
echo "📁 Backup location: $BACKUP_FILE"
echo ""
echo "🔄 Next step: Restart mcpproxy to apply changes"
echo "   pkill mcpproxy && ./mcpproxy serve"
echo ""
