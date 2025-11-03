#!/bin/bash

# fix-pipx-path.sh - Add PATH environment variable to all pipx servers
# This fixes the issue where pipx command is not found by mcpproxy child processes

set -e

CONFIG_FILE="$HOME/.mcpproxy/mcp_config.json"
BACKUP_FILE="$CONFIG_FILE.backup-before-pipx-path-fix-$(date +%Y%m%d-%H%M%S)"

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🔧 PIPX PATH Fix Script"
echo "============================================================"
echo ""

# Backup configuration
echo "📦 Creating backup..."
cp "$CONFIG_FILE" "$BACKUP_FILE"
echo "✅ Backup created: $BACKUP_FILE"
echo ""

# Comprehensive PATH including all user directories
COMPREHENSIVE_PATH="/Users/hrannow/.local/bin:/Users/hrannow/.amplify/bin:/Users/hrannow/.pyenv/bin:/Users/hrannow/.codeium/windsurf/bin:/opt/homebrew/bin:/opt/homebrew/sbin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin:/Users/hrannow/.rvm/bin"

echo "🔍 Finding pipx servers..."
echo ""

# Get list of servers using pipx command
PIPX_SERVERS=$(jq -r '.mcpServers[] | select(.command == "pipx") | .name' "$CONFIG_FILE")

if [ -z "$PIPX_SERVERS" ]; then
    echo "❌ No pipx servers found in configuration"
    exit 0
fi

SERVER_COUNT=$(echo "$PIPX_SERVERS" | wc -l | tr -d ' ')
echo "Found $SERVER_COUNT servers using pipx command:"
echo "$PIPX_SERVERS" | sed 's/^/  - /'
echo ""

echo "🔨 Adding PATH environment variable to each server..."
echo ""
echo "PATH includes pipx location: /opt/homebrew/bin/pipx"
echo ""

# Use jq to add PATH to env for each pipx server
jq --arg path "$COMPREHENSIVE_PATH" '
  .mcpServers |= map(
    if .command == "pipx" then
      .env = ((.env // {}) + {"PATH": $path})
    else
      .
    end
  )
' "$CONFIG_FILE" > "$CONFIG_FILE.tmp"

# Verify the modification worked
if jq empty "$CONFIG_FILE.tmp" 2>/dev/null; then
    mv "$CONFIG_FILE.tmp" "$CONFIG_FILE"
    echo "✅ PATH added to $SERVER_COUNT pipx servers"
else
    echo "❌ Error: JSON validation failed"
    rm "$CONFIG_FILE.tmp"
    exit 1
fi

echo ""
echo "📋 Verification:"
echo ""

# Show one example of the change
FIRST_SERVER=$(echo "$PIPX_SERVERS" | head -1)
echo "Example server configuration ($FIRST_SERVER):"
jq --arg name "$FIRST_SERVER" '.mcpServers[] | select(.name == $name) | {name, command, "PATH": .env.PATH}' "$CONFIG_FILE"

echo ""
echo "✅ PIPX PATH fix applied successfully!"
echo ""
echo "📝 Next steps:"
echo "   1. Restart mcpproxy:"
echo "      ps aux | grep '[m]cpproxy serve' | awk '{print \$2}' | xargs kill -9"
echo "      cd ~/Library/CloudStorage/OneDrive-Persönlich/workspace/mcp-server/mcpproxy-go"
echo "      ./mcpproxy serve"
echo ""
echo "   2. Wait 2 minutes for startup to complete"
echo ""
echo "   3. Check pipx server logs:"
echo "      tail -50 ~/Library/Logs/mcpproxy/server-bigquery-lucashild.log"
echo "      tail -50 ~/Library/Logs/mcpproxy/server-duckdb-ktanaka.log"
echo ""
echo "📂 Backup location: $BACKUP_FILE"
