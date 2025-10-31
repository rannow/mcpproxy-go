#!/bin/bash
# Agent-Powered MCP Server Recovery System
# Uses Claude Flow agents for intelligent server diagnosis and recovery

set -e

CONFIG_FILE="$HOME/.mcpproxy/mcp_config.json"
LOG_DIR="$HOME/Library/Logs/mcpproxy"

echo "🤖 Agent-Powered Recovery System"
echo "=================================="
echo ""

# Function: Analyze server failure patterns
analyze_failure_patterns() {
    echo "🔬 Analyzing Failure Patterns..."
    echo ""

    # Check for common OAuth failures
    local oauth_failures=$(grep -c "oauth.*fail\|401\|invalid_token" "$LOG_DIR/main.log" 2>/dev/null || echo "0")
    if [ "$oauth_failures" -gt 0 ]; then
        echo "  ⚠️  OAuth Authentication Failures: $oauth_failures"
        echo "     → Recommendation: Re-authenticate servers with OAuth"
        echo "     → Run: mcpproxy auth login --server=<name>"
        echo ""
    fi

    # Check for connection timeouts
    local timeout_failures=$(grep -c "timeout\|connection refused\|dial tcp" "$LOG_DIR/main.log" 2>/dev/null || echo "0")
    if [ "$timeout_failures" -gt 0 ]; then
        echo "  ⚠️  Connection Timeout Failures: $timeout_failures"
        echo "     → Recommendation: Check server URLs and network connectivity"
        echo "     → Verify: curl -I <server-url>"
        echo ""
    fi

    # Check for process/command failures
    local cmd_failures=$(grep -c "exec.*not found\|command not found\|no such file" "$LOG_DIR/main.log" 2>/dev/null || echo "0")
    if [ "$cmd_failures" -gt 0 ]; then
        echo "  ⚠️  Command/Process Failures: $cmd_failures"
        echo "     → Recommendation: Install missing dependencies (npx, uvx, etc.)"
        echo "     → Verify: which npx && which uvx"
        echo ""
    fi

    # Check for Docker isolation issues
    local docker_failures=$(grep -c "docker.*error\|container.*fail" "$LOG_DIR/main.log" 2>/dev/null || echo "0")
    if [ "$docker_failures" -gt 0 ]; then
        echo "  ⚠️  Docker Isolation Failures: $docker_failures"
        echo "     → Recommendation: Check Docker daemon status"
        echo "     → Verify: docker ps"
        echo ""
    fi
}

# Function: Identify servers by failure type
categorize_servers() {
    echo "📊 Server Categories:"
    echo ""

    # Quarantined servers
    local quarantined=$(jq -r '[.mcpServers[] | select(.quarantined == true)] | length' "$CONFIG_FILE")
    echo "  🔒 Quarantined: $quarantined"
    if [ "$quarantined" -gt 0 ]; then
        echo "     → These need manual security review"
        jq -r '.mcpServers[] | select(.quarantined == true) | "        - \(.name)"' "$CONFIG_FILE"
    fi
    echo ""

    # Disabled but healthy servers
    local disabled_healthy=$(jq -r '[.mcpServers[] | select(.enabled == false and (.quarantined == false or .quarantined == null))] | length' "$CONFIG_FILE")
    echo "  ✅ Disabled (Safe to Enable): $disabled_healthy"
    echo "     → These can be auto-recovered"
    echo ""

    # OAuth servers
    local oauth_servers=$(jq -r '[.mcpServers[] | select(.url != null and .enabled == true)] | length' "$CONFIG_FILE")
    echo "  🔐 OAuth/HTTP Servers: $oauth_servers"
    echo "     → May need token refresh"
    echo ""

    # stdio servers
    local stdio_servers=$(jq -r '[.mcpServers[] | select(.command != null and .enabled == true)] | length' "$CONFIG_FILE")
    echo "  📡 stdio Servers: $stdio_servers"
    echo "     → May have dependency issues"
    echo ""
}

# Function: Smart recovery suggestions
generate_recovery_plan() {
    echo "🎯 Intelligent Recovery Plan:"
    echo ""

    echo "PHASE 1: Security Review (MANUAL)"
    echo "  1. Review quarantined servers in tray UI"
    echo "  2. Approve safe servers, remove malicious ones"
    echo ""

    echo "PHASE 2: Bulk Enable (AUTOMATED)"
    echo "  1. Run auto-recovery script:"
    echo "     ~/.mcpproxy/auto-recovery.sh"
    echo "  2. This will enable all non-quarantined disabled servers"
    echo ""

    echo "PHASE 3: Authentication (SEMI-AUTOMATED)"
    echo "  1. For OAuth servers showing 401 errors:"
    echo "     → Re-authenticate each server"
    echo "     → mcpproxy auth login --server=<name>"
    echo ""

    echo "PHASE 4: Dependency Check (MANUAL)"
    echo "  1. Verify required tools are installed:"
    echo "     → npm/npx for Node.js MCP servers"
    echo "     → python/uvx for Python MCP servers"
    echo "     → docker for isolated servers"
    echo ""

    echo "PHASE 5: Startup Script Configuration (RECOMMENDED)"
    echo "  1. Configure startup_script in config for auto-start"
    echo "  2. Use mcpproxy startup_script tool or edit config directly"
    echo ""

    echo "PHASE 6: Monitoring (AUTOMATED)"
    echo "  1. Watch logs for persistent failures:"
    echo "     tail -f $LOG_DIR/main.log | grep ERROR"
    echo "  2. Disable servers that consistently fail after 3 attempts"
    echo ""
}

# Function: Quick fix script for common issues
create_quick_fixes() {
    local fix_file="$HOME/.mcpproxy/quick-fixes.sh"

    cat > "$fix_file" << 'FIXEOF'
#!/bin/bash
# Quick fixes for common MCP server issues

echo "🔧 Running Quick Fixes..."

# Fix 1: Install missing Node.js/Python dependencies
echo "1. Checking package managers..."
command -v npm >/dev/null 2>&1 || echo "  ⚠️  npm not found - install Node.js"
command -v npx >/dev/null 2>&1 || echo "  ⚠️  npx not found - install Node.js"
command -v python3 >/dev/null 2>&1 || echo "  ⚠️  python3 not found - install Python"
command -v uvx >/dev/null 2>&1 || echo "  💡 uvx not found - run: pip install uv"

# Fix 2: Docker health check
echo "2. Checking Docker..."
if command -v docker >/dev/null 2>&1; then
    if docker ps >/dev/null 2>&1; then
        echo "  ✅ Docker is running"
    else
        echo "  ⚠️  Docker daemon not running - start Docker Desktop"
    fi
else
    echo "  ⚠️  Docker not installed"
fi

# Fix 3: Clear stale OAuth tokens
echo "3. OAuth token cleanup..."
if [ -d "$HOME/.mcpproxy/tokens" ]; then
    local stale_tokens=$(find "$HOME/.mcpproxy/tokens" -name "*.json" -mtime +30 | wc -l)
    echo "  Found $stale_tokens stale tokens (>30 days old)"
    echo "  💡 Consider re-authenticating affected servers"
fi

# Fix 4: Log rotation check
echo "4. Checking log sizes..."
local log_size=$(du -sh "$HOME/Library/Logs/mcpproxy" 2>/dev/null | cut -f1)
echo "  Log directory size: $log_size"

echo "✅ Quick fixes completed"
FIXEOF

    chmod +x "$fix_file"
    echo "💾 Quick fixes script created: $fix_file"
    echo ""
}

# Main execution
main() {
    analyze_failure_patterns
    categorize_servers
    generate_recovery_plan
    create_quick_fixes

    echo "═══════════════════════════════════════════"
    echo "🚀 Next Steps:"
    echo ""
    echo "1. Run diagnostics:    ./scripts/diagnose-and-recover.sh"
    echo "2. Run quick fixes:    ~/.mcpproxy/quick-fixes.sh"
    echo "3. Enable servers:     ~/.mcpproxy/auto-recovery.sh"
    echo "4. Restart mcpproxy:   pkill mcpproxy && ./mcpproxy serve"
    echo ""
    echo "═══════════════════════════════════════════"
}

main "$@"
