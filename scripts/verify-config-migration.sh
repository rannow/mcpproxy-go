#!/bin/bash
# Script to verify config migration is complete and consistent

CONFIG_FILE="$HOME/.mcpproxy/mcp_config.json"

echo "🔍 Verifying config migration status..."
echo "Config file: $CONFIG_FILE"
echo ""

# Check if config exists
if [ ! -f "$CONFIG_FILE" ]; then
    echo "❌ Config file not found"
    exit 1
fi

# Count servers with each startup_mode
echo "📊 Startup Mode Distribution:"
echo "----------------------------"
jq -r '.mcpServers | group_by(.startup_mode) | .[] | "\(.[0].startup_mode // "null"): \(length) servers"' "$CONFIG_FILE" | sort
echo ""

# Find servers with inconsistent state (missing startup_mode or mismatch)
echo "⚠️  Checking for inconsistencies..."
echo "-----------------------------------"

# Servers with enabled=true but startup_mode=disabled
INCONSISTENT=$(jq -r '.mcpServers | .[] | select(.enabled == true and .startup_mode == "disabled") | .name' "$CONFIG_FILE")
if [ -n "$INCONSISTENT" ]; then
    echo "❌ Servers enabled but startup_mode=disabled:"
    echo "$INCONSISTENT"
else
    echo "✅ No enabled servers with startup_mode=disabled"
fi

# Servers with enabled=false but startup_mode=active or lazy_loading
INCONSISTENT2=$(jq -r '.mcpServers | .[] | select(.enabled == false and (.startup_mode == "active" or .startup_mode == "lazy_loading")) | .name' "$CONFIG_FILE")
if [ -n "$INCONSISTENT2" ]; then
    echo "❌ Servers disabled but startup_mode=active/lazy_loading:"
    echo "$INCONSISTENT2"
else
    echo "✅ No disabled servers with startup_mode=active/lazy_loading"
fi

# Servers with auto_disabled=true but startup_mode != auto_disabled
INCONSISTENT3=$(jq -r '.mcpServers | .[] | select(.auto_disabled == true and .startup_mode != "auto_disabled") | .name' "$CONFIG_FILE")
if [ -n "$INCONSISTENT3" ]; then
    echo "❌ Servers with auto_disabled=true but startup_mode != auto_disabled:"
    echo "$INCONSISTENT3"
else
    echo "✅ No auto-disabled servers with incorrect startup_mode"
fi

# Servers missing startup_mode field
MISSING=$(jq -r '.mcpServers | .[] | select(.startup_mode == null) | .name' "$CONFIG_FILE")
if [ -n "$MISSING" ]; then
    echo "❌ Servers missing startup_mode field:"
    echo "$MISSING"
else
    echo "✅ All servers have startup_mode field"
fi

echo ""
echo "✅ Config migration verification complete"
