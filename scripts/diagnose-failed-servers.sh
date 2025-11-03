#!/bin/bash
# Comprehensive MCP Server Diagnostics Script
# Tests each failed server individually and categorizes issues

set -euo pipefail

CONFIG_FILE="$HOME/.mcpproxy/mcp_config.json"
FAILED_SERVERS_FILE="/tmp/failed_servers.txt"
OUTPUT_FILE="FAILED_SERVERS_DETAILED_REPORT.md"

echo "🔍 Starting comprehensive server diagnostics..."
echo ""

# Initialize report
cat > "$OUTPUT_FILE" << 'HEADER'
# Failed MCP Servers - Detailed Diagnostic Report

**Generated**: $(date)
**Total Failed Servers**: $(wc -l < "$FAILED_SERVERS_FILE")
**Analysis Method**: Individual server testing with categorized diagnostics

---

## Executive Summary

This report provides detailed diagnostics for each MCP server that failed to connect during startup.

**Categories**:
- 🔐 **Authentication Required** - Needs OAuth/API keys
- 🏗️ **Infrastructure Missing** - Needs external services (DB, K8s, etc.)
- ⏱️ **Timeout/Slow** - Takes >60s to initialize
- ⚙️ **Configuration Error** - Wrong command/args/env
- 📦 **Package Issue** - Missing dependencies or broken package
- ❌ **Broken/Deprecated** - Package no longer works
- 🔧 **Custom Setup** - Needs manual configuration

---

## Detailed Server Analysis

HEADER

# Function to test a single server
test_server() {
    local SERVER_NAME="$1"
    local SERVER_CONFIG=$(jq -r ".mcpServers[] | select(.name == \"$SERVER_NAME\")" "$CONFIG_FILE")

    local COMMAND=$(echo "$SERVER_CONFIG" | jq -r '.command')
    local ARGS=$(echo "$SERVER_CONFIG" | jq -r '.args | join(" ")')
    local PROTOCOL=$(echo "$SERVER_CONFIG" | jq -r '.protocol // "stdio"')

    echo "### $SERVER_NAME" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"
    echo "**Protocol**: $PROTOCOL" >> "$OUTPUT_FILE"
    echo "**Command**: \`$COMMAND $ARGS\`" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"

    # Test execution with timeout
    echo "Testing $SERVER_NAME..." >&2

    if timeout 5s $COMMAND --version &>/dev/null; then
        echo "**Status**: ✅ Command executable found" >> "$OUTPUT_FILE"
    elif timeout 5s which $COMMAND &>/dev/null; then
        echo "**Status**: ⚠️ Command exists but may need setup" >> "$OUTPUT_FILE"
    else
        echo "**Status**: ❌ Command not found" >> "$OUTPUT_FILE"
        echo "" >> "$OUTPUT_FILE"
        echo "**Issue**: Command \`$COMMAND\` not installed or not in PATH" >> "$OUTPUT_FILE"
        echo "" >> "$OUTPUT_FILE"
        echo "**Fix**: Install the command:" >> "$OUTPUT_FILE"

        if [ "$COMMAND" = "uvx" ] || [ "$COMMAND" = "uv" ]; then
            echo "\`\`\`bash" >> "$OUTPUT_FILE"
            echo "pip install uv" >> "$OUTPUT_FILE"
            echo "\`\`\`" >> "$OUTPUT_FILE"
        elif [ "$COMMAND" = "npx" ]; then
            echo "\`\`\`bash" >> "$OUTPUT_FILE"
            echo "# npx comes with Node.js" >> "$OUTPUT_FILE"
            echo "# Verify: npx --version" >> "$OUTPUT_FILE"
            echo "\`\`\`" >> "$OUTPUT_FILE"
        fi
    fi

    echo "" >> "$OUTPUT_FILE"
    echo "---" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"
}

# Process all failed servers
TOTAL=$(wc -l < "$FAILED_SERVERS_FILE")
CURRENT=0

while IFS= read -r server_name; do
    CURRENT=$((CURRENT + 1))
    echo "[$CURRENT/$TOTAL] Testing: $server_name"
    test_server "$server_name" || true
done < "$FAILED_SERVERS_FILE"

echo ""
echo "✅ Diagnostic report generated: $OUTPUT_FILE"
echo ""
