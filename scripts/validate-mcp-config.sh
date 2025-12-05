#!/bin/bash
# MCP Configuration Validation Script
# Validates the consolidated MCP configuration and tests server availability

set -e

echo "🔍 MCP Configuration Validation"
echo "================================"
echo ""

# Configuration file paths
MCP_CONFIG="/Users/hrannow/Library/CloudStorage/OneDrive-Persönlich/workspace/mcp-server/mcpproxy-go/.mcp.json"
SETTINGS_CONFIG="/Users/hrannow/Library/CloudStorage/OneDrive-Persönlich/workspace/mcp-server/mcpproxy-go/.claude/settings.json"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Validate JSON syntax
echo "1️⃣ Validating JSON syntax..."
if jq empty "$MCP_CONFIG" 2>/dev/null; then
    echo -e "${GREEN}✓${NC} .mcp.json is valid JSON"
else
    echo -e "${RED}✗${NC} .mcp.json has invalid JSON syntax"
    exit 1
fi

if jq empty "$SETTINGS_CONFIG" 2>/dev/null; then
    echo -e "${GREEN}✓${NC} .claude/settings.json is valid JSON"
else
    echo -e "${RED}✗${NC} .claude/settings.json has invalid JSON syntax"
    exit 1
fi

echo ""

# Count servers
echo "2️⃣ Analyzing server configuration..."
TOTAL_SERVERS=$(jq '.mcpServers | length' "$MCP_CONFIG")
echo -e "${GREEN}✓${NC} Found $TOTAL_SERVERS configured servers"

# List servers
echo ""
echo "Configured servers:"
jq -r '.mcpServers | keys[]' "$MCP_CONFIG" | while read -r server; do
    TYPE=$(jq -r ".mcpServers[\"$server\"].type" "$MCP_CONFIG")
    PRIORITY=$(jq -r ".mcpServers[\"$server\"].priority" "$MCP_CONFIG")
    DESCRIPTION=$(jq -r ".mcpServers[\"$server\"].description" "$MCP_CONFIG")
    echo "  • $server (Priority $PRIORITY, Type: $TYPE)"
    echo "    $DESCRIPTION"
done

echo ""

# Validate server types
echo "3️⃣ Validating server types..."
VALID_TYPES=("stdio" "streamable-http")
ALL_VALID=true

jq -r '.mcpServers | keys[]' "$MCP_CONFIG" | while read -r server; do
    TYPE=$(jq -r ".mcpServers[\"$server\"].type" "$MCP_CONFIG")
    if [[ " ${VALID_TYPES[@]} " =~ " ${TYPE} " ]]; then
        echo -e "${GREEN}✓${NC} $server: Valid type '$TYPE'"
    else
        echo -e "${RED}✗${NC} $server: Invalid type '$TYPE'"
        ALL_VALID=false
    fi
done

echo ""

# Check for required fields
echo "4️⃣ Checking required fields..."
jq -r '.mcpServers | keys[]' "$MCP_CONFIG" | while read -r server; do
    TYPE=$(jq -r ".mcpServers[\"$server\"].type" "$MCP_CONFIG")

    if [ "$TYPE" = "stdio" ]; then
        if jq -e ".mcpServers[\"$server\"] | has(\"command\") and has(\"args\")" "$MCP_CONFIG" > /dev/null; then
            echo -e "${GREEN}✓${NC} $server: Has required 'command' and 'args' fields"
        else
            echo -e "${RED}✗${NC} $server: Missing required 'command' or 'args' fields"
        fi
    elif [ "$TYPE" = "streamable-http" ]; then
        if jq -e ".mcpServers[\"$server\"] | has(\"url\")" "$MCP_CONFIG" > /dev/null; then
            echo -e "${GREEN}✓${NC} $server: Has required 'url' field"
        else
            echo -e "${RED}✗${NC} $server: Missing required 'url' field"
        fi
    fi
done

echo ""

# Test server availability
echo "5️⃣ Testing server availability..."

# Test npx commands
if command_exists npx; then
    echo -e "${GREEN}✓${NC} npx is available"

    # Test claude-flow@alpha
    if npx -y claude-flow@alpha --version >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} claude-flow@alpha is available"
    else
        echo -e "${YELLOW}⚠${NC} claude-flow@alpha may need installation"
    fi

    # Test flow-nexus
    if npx -y flow-nexus@latest --version >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} flow-nexus is available"
    else
        echo -e "${YELLOW}⚠${NC} flow-nexus may need installation"
    fi

    # Test ruv-swarm
    if npx -y ruv-swarm --version >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} ruv-swarm is available"
    else
        echo -e "${YELLOW}⚠${NC} ruv-swarm may need installation"
    fi
else
    echo -e "${RED}✗${NC} npx is not available"
fi

echo ""

# Test mcpproxy local server
echo "6️⃣ Testing mcpproxy local server..."
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/mcp | grep -q "200\|404"; then
    echo -e "${GREEN}✓${NC} mcpproxy local server is responding"
else
    echo -e "${YELLOW}⚠${NC} mcpproxy local server is not running (start with: go run cmd/mcpproxy/main.go)"
fi

echo ""

# Check hooks configuration
echo "7️⃣ Validating hooks configuration..."
HOOKS_ENABLED=$(jq -r '.env.CLAUDE_FLOW_HOOKS_ENABLED' "$SETTINGS_CONFIG")
if [ "$HOOKS_ENABLED" = "true" ]; then
    echo -e "${GREEN}✓${NC} Claude Flow hooks are enabled"
else
    echo -e "${YELLOW}⚠${NC} Claude Flow hooks are disabled"
fi

# Check enabled MCP servers in settings
ENABLED_SERVERS=$(jq -r '.enabledMcpjsonServers[]' "$SETTINGS_CONFIG" 2>/dev/null || echo "")
if [ -n "$ENABLED_SERVERS" ]; then
    echo -e "${GREEN}✓${NC} Enabled MCP servers in settings: $ENABLED_SERVERS"
else
    echo -e "${YELLOW}⚠${NC} No enabled MCP servers defined in settings"
fi

echo ""

# Summary
echo "8️⃣ Configuration Summary"
echo "========================"
jq -r '.metadata | to_entries[] | "\(.key): \(.value)"' "$MCP_CONFIG" | grep -v "^\[" | grep -v "^{" || true

echo ""

# Optimization recommendations
echo "9️⃣ Optimization Recommendations"
echo "================================"
jq -r '.metadata.optimizations[]' "$MCP_CONFIG" | while read -r opt; do
    echo -e "${GREEN}✓${NC} $opt"
done

echo ""

# Final status
echo "🎉 Validation Complete!"
echo ""
echo "Next steps:"
echo "1. Restart Claude Code to load new configuration"
echo "2. Test MCP server connections with: npx claude-flow@alpha mcp test"
echo "3. Review documentation: docs/MCP_CONFIGURATION.md"
echo "4. Start mcpproxy local server if needed: go run cmd/mcpproxy/main.go"
echo ""
echo "Configuration file: $MCP_CONFIG"
echo "Backup files: .mcp.json.backup.* and mcp.json.backup.*"
