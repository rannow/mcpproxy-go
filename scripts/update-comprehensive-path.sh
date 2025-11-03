#!/bin/bash

# update-comprehensive-path.sh - Update PATH with comprehensive directory list
# Includes all paths from user's shell configuration

set -e

CONFIG_FILE="$HOME/.mcpproxy/mcp_config.json"
BACKUP_FILE="$CONFIG_FILE.backup-before-comprehensive-path-$(date +%Y%m%d-%H%M%S)"

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🔧 Comprehensive PATH Update Script"
echo "============================================================"
echo ""

# Backup configuration
echo "📦 Creating backup..."
cp "$CONFIG_FILE" "$BACKUP_FILE"
echo "✅ Backup created: $BACKUP_FILE"
echo ""

# Comprehensive PATH including all user directories
COMPREHENSIVE_PATH="/Users/hrannow/.local/bin:/Users/hrannow/.amplify/bin:/Users/hrannow/.pyenv/bin:/Users/hrannow/.codeium/windsurf/bin:/opt/homebrew/bin:/opt/homebrew/sbin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin:/Users/hrannow/.rvm/bin"

echo "🔍 Finding uvx servers..."
echo ""

# Get list of servers using uvx command
UVX_SERVERS=$(jq -r '.mcpServers[] | select(.command == "uvx") | .name' "$CONFIG_FILE")

if [ -z "$UVX_SERVERS" ]; then
    echo "❌ No uvx servers found in configuration"
    exit 0
fi

SERVER_COUNT=$(echo "$UVX_SERVERS" | wc -l | tr -d ' ')
echo "Found $SERVER_COUNT servers using uvx command"
echo ""

echo "🔨 Updating PATH to comprehensive version..."
echo ""
echo "New PATH includes:"
echo "  - ~/.local/bin (uvx, uv)"
echo "  - ~/.amplify/bin (AWS Amplify)"
echo "  - ~/.pyenv/bin (Python version management)"
echo "  - ~/.codeium/windsurf/bin (Windsurf)"
echo "  - /opt/homebrew/bin (Homebrew)"
echo "  - ~/.rvm/bin (Ruby version management)"
echo "  - Standard system paths"
echo ""

# Use jq to update PATH for each uvx server
jq --arg path "$COMPREHENSIVE_PATH" '
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
    echo "✅ PATH updated for $SERVER_COUNT uvx servers"
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
jq --arg name "$FIRST_SERVER" '.mcpServers[] | select(.name == $name) | {name, command, "PATH": .env.PATH}' "$CONFIG_FILE"

echo ""
echo "✅ Comprehensive PATH update complete!"
echo ""
echo "📝 Next steps:"
echo "   1. Restart mcpproxy to apply changes:"
echo "      ps aux | grep '[m]cpproxy serve' | awk '{print \$2}' | xargs kill -9"
echo "      cd ~/Library/CloudStorage/OneDrive-Persönlich/workspace/mcp-server/mcpproxy-go"
echo "      ./mcpproxy serve"
echo ""
echo "📂 Backup location: $BACKUP_FILE"
