#!/bin/bash
# MCPProxy Server Diagnostic and Recovery Script
# Created by Claude Flow diagnostic agents

set -e

CONFIG_FILE="$HOME/.mcpproxy/mcp_config.json"
LOG_DIR="$HOME/Library/Logs/mcpproxy"
BACKUP_DIR="$HOME/.mcpproxy/backups"

echo "🔍 MCPProxy Server Diagnostic & Recovery Tool"
echo "=============================================="
echo ""

# Function: Count server status
count_servers() {
    local disabled=$(grep -c '"enabled": false' "$CONFIG_FILE" 2>/dev/null || echo "0")
    local enabled=$(grep -c '"enabled": true' "$CONFIG_FILE" 2>/dev/null || echo "0")
    local total=$((disabled + enabled))

    echo "📊 Server Status:"
    echo "   Total:    $total"
    echo "   Enabled:  $enabled"
    echo "   Disabled: $disabled ($(echo "scale=1; $disabled*100/$total" | bc)%)"
    echo ""
}

# Function: Check startup script configuration
check_startup_script() {
    echo "🚀 Startup Script Configuration:"
    if grep -q '"startup_script"' "$CONFIG_FILE" 2>/dev/null; then
        local enabled=$(jq -r '.startup_script.enabled // false' "$CONFIG_FILE")
        local path=$(jq -r '.startup_script.path // "NOT_SET"' "$CONFIG_FILE")
        echo "   Configured: YES"
        echo "   Enabled:    $enabled"
        echo "   Path:       $path"
    else
        echo "   ❌ NOT CONFIGURED"
        echo "   💡 This means servers are NOT auto-started at launch!"
    fi
    echo ""
}

# Function: List disabled servers
list_disabled() {
    echo "📋 Disabled Servers (first 10):"
    jq -r '.mcpServers[] | select(.enabled == false) | "   - \(.name)"' "$CONFIG_FILE" 2>/dev/null | head -10
    echo ""
}

# Function: Check quarantined servers
check_quarantined() {
    echo "🔒 Quarantined Servers:"
    local quarantined=$(jq -r '.mcpServers[] | select(.quarantined == true) | .name' "$CONFIG_FILE" 2>/dev/null | wc -l)
    echo "   Count: $quarantined"
    if [ "$quarantined" -gt 0 ]; then
        echo "   ⚠️  These servers are blocked for security reasons"
        jq -r '.mcpServers[] | select(.quarantined == true) | "   - \(.name)"' "$CONFIG_FILE" 2>/dev/null
    fi
    echo ""
}

# Function: Analyze recent errors
analyze_errors() {
    echo "🔴 Recent Server Errors (last 24h):"
    if [ -f "$LOG_DIR/main.log" ]; then
        local error_count=$(grep -c "ERROR.*server" "$LOG_DIR/main.log" 2>/dev/null || echo "0")
        echo "   Error count: $error_count"
        if [ "$error_count" -gt 0 ]; then
            echo "   Recent errors:"
            grep "ERROR.*server" "$LOG_DIR/main.log" 2>/dev/null | tail -5 | sed 's/^/   /'
        fi
    else
        echo "   ⚠️  Log file not found"
    fi
    echo ""
}

# Function: Generate recovery script
generate_recovery_script() {
    local recovery_file="$HOME/.mcpproxy/auto-recovery.sh"

    cat > "$recovery_file" << 'EOF'
#!/bin/bash
# Auto-generated recovery script
# Re-enables all non-quarantined disabled servers

CONFIG_FILE="$HOME/.mcpproxy/mcp_config.json"
BACKUP_FILE="$HOME/.mcpproxy/backups/config-before-recovery-$(date +%Y%m%d-%H%M%S).json"

# Backup current config
mkdir -p "$(dirname "$BACKUP_FILE")"
cp "$CONFIG_FILE" "$BACKUP_FILE"
echo "✅ Config backed up to: $BACKUP_FILE"

# Enable all non-quarantined servers
jq '
  .mcpServers = (.mcpServers | map(
    if .enabled == false and (.quarantined == false or .quarantined == null)
    then .enabled = true
    else .
    end
  ))
' "$CONFIG_FILE" > "$CONFIG_FILE.tmp"

mv "$CONFIG_FILE.tmp" "$CONFIG_FILE"
echo "✅ All non-quarantined servers enabled"
echo "🔄 Restart mcpproxy for changes to take effect"
EOF

    chmod +x "$recovery_file"
    echo "💾 Recovery script generated: $recovery_file"
    echo ""
}

# Function: Configure startup script
configure_startup() {
    echo "⚙️  Startup Script Configuration Wizard"
    echo "   Would you like to configure auto-start for servers?"
    echo "   (This allows servers to start automatically when mcpproxy launches)"
    echo ""
    echo "   Example startup script content:"
    echo "     #!/bin/bash"
    echo "     echo 'Initializing MCP servers...'"
    echo "     # Add your startup logic here"
    echo ""
}

# Main execution
main() {
    count_servers
    check_startup_script
    check_quarantined
    list_disabled
    analyze_errors
    generate_recovery_script
    configure_startup

    echo "═══════════════════════════════════════════"
    echo "🎯 Recommended Actions:"
    echo ""
    echo "1. Review quarantined servers and approve safe ones"
    echo "2. Run auto-recovery script to enable disabled servers:"
    echo "   ~/.mcpproxy/auto-recovery.sh"
    echo ""
    echo "3. Configure startup script for auto-initialization"
    echo ""
    echo "4. Check server logs for specific connection issues:"
    echo "   ls -lh $LOG_DIR/server-*.log"
    echo ""
    echo "═══════════════════════════════════════════"
}

main "$@"
