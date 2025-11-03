#!/bin/bash

# fix-uvx-path.sh - Add PATH environment variable to all uvx servers
# This fixes the issue where uvx command is not found by mcpproxy child processes

set -e

CONFIG_FILE="$HOME/.mcpproxy/mcp_config.json"
BACKUP_FILE="$CONFIG_FILE.backup-before-uvx-path-fix-$(date +%Y%m%d-%H%M%S)"

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🔧 UVX PATH Fix Script"
echo "============================================================"
echo ""

# Backup configuration
echo "📦 Creating backup..."
cp "$CONFIG_FILE" "$BACKUP_FILE"
echo "✅ Backup created: $BACKUP_FILE"
echo ""

# The PATH to add (includes ~/.local/bin where uvx is located)
UVX_PATH="/Users/hrannow/.local/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"

echo "🔍 Finding uvx servers..."
echo ""

# Get list of servers using uvx command
UVX_SERVERS=$(jq -r '.mcpServers[] | select(.command == "uvx") | .name' "$CONFIG_FILE")

if [ -z "$UVX_SERVERS" ]; then
    echo "❌ No uvx servers found in configuration"
    exit 0
fi

SERVER_COUNT=$(echo "$UVX_SERVERS" | wc -l | tr -d ' ')
echo "Found $SERVER_COUNT servers using uvx command:"
echo "$UVX_SERVERS" | sed 's/^/  - /'
echo ""

echo "🔨 Adding PATH environment variable to each server..."
echo ""

# Use jq to add PATH to env for each uvx server
jq --arg path "$UVX_PATH" '
  .mcpServers |= map(
    if .command == "uvx" then
      .env = ((.env // {}) + {"PATH": $path})
    else
      .
    end
  )
' "$CONFIG_FILE" > "$CONFIG_FILE.tmp"

# Verify the modification worked
if jq empty "$CONFIG_FILE.tmp" 2>/dev/null; then
    mv "$CONFIG_FILE.tmp" "$CONFIG_FILE"
    echo "✅ PATH added to $SERVER_COUNT uvx servers"
else
    echo "❌ Error: JSON validation failed"
    rm "$CONFIG_FILE.tmp"
    exit 1
fi

echo ""
echo "📋 Verification:"
echo ""

# Show one example of the change
FIRST_SERVER=$(echo "$UVX_SERVERS" | head -1)
echo "Example server configuration ($FIRST_SERVER):"
jq --arg name "$FIRST_SERVER" '.mcpServers[] | select(.name == $name) | {name, command, env}' "$CONFIG_FILE"

echo ""
echo "✅ Fix applied successfully!"
echo ""
echo "📝 Next steps:"
echo "   1. Restart mcpproxy:"
echo "      ps aux | grep '[m]cpproxy serve' | awk '{print \$2}' | xargs kill -9"
echo "      cd ~/Library/CloudStorage/OneDrive-Persönlich/workspace/mcp-server/mcpproxy-go"
echo "      ./mcpproxy serve"
echo ""
echo "   2. Wait 2 minutes for startup to complete"
echo ""
echo "   3. Check connection count (expect +18 servers):"
echo "      tail -1000 ~/Library/Logs/mcpproxy/main.log | grep -c 'Successfully retrieved tools'"
echo ""
echo "📂 Backup location: $BACKUP_FILE"
