#!/bin/bash
# Interactive Agent for MCPProxy Diagnostics
# Provides menu-driven access to diagnostic agents

CONFIG_FILE="$HOME/.mcpproxy/mcp_config.json"
LOG_DIR="$HOME/Library/Logs/mcpproxy"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔══════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  MCPProxy Interactive Diagnostic Agent  ║${NC}"
echo -e "${BLUE}╔══════════════════════════════════════════╗${NC}"
echo ""

# Function: Display main menu
show_menu() {
    echo -e "${GREEN}Available Operations:${NC}"
    echo ""
    echo "  ${YELLOW}[1]${NC} Server Status Summary"
    echo "  ${YELLOW}[2]${NC} Analyze Failure Patterns"
    echo "  ${YELLOW}[3]${NC} Check Quarantined Servers"
    echo "  ${YELLOW}[4]${NC} Verify Dependencies"
    echo "  ${YELLOW}[5]${NC} OAuth Server Status"
    echo "  ${YELLOW}[6]${NC} Enable All Safe Servers (AUTO-RECOVERY)"
    echo "  ${YELLOW}[7]${NC} Configure Startup Script"
    echo "  ${YELLOW}[8]${NC} View Recent Errors"
    echo "  ${YELLOW}[9]${NC} Generate Full Report"
    echo "  ${YELLOW}[0]${NC} Exit"
    echo ""
    echo -n "Select operation: "
}

# [1] Server Status Summary
server_status() {
    echo -e "\n${BLUE}📊 Server Status Summary${NC}"
    echo "══════════════════════════════════"

    local total=$(jq -r '.mcpServers | length' "$CONFIG_FILE")
    local enabled=$(jq -r '[.mcpServers[] | select(.enabled == true)] | length' "$CONFIG_FILE")
    local disabled=$(jq -r '[.mcpServers[] | select(.enabled == false)] | length' "$CONFIG_FILE")
    local quarantined=$(jq -r '[.mcpServers[] | select(.quarantined == true)] | length' "$CONFIG_FILE")

    echo "Total Servers:       $total"
    echo -e "${GREEN}Enabled:${NC}             $enabled ($(echo "scale=1; $enabled*100/$total" | bc)%)"
    echo -e "${RED}Disabled:${NC}            $disabled ($(echo "scale=1; $disabled*100/$total" | bc)%)"
    echo -e "${YELLOW}Quarantined:${NC}         $quarantined ($(echo "scale=1; $quarantined*100/$total" | bc)%)"
    echo ""

    read -p "Press Enter to continue..."
}

# [2] Analyze Failure Patterns
analyze_failures() {
    echo -e "\n${BLUE}🔬 Analyzing Failure Patterns${NC}"
    echo "══════════════════════════════════"

    if [ ! -f "$LOG_DIR/main.log" ]; then
        echo -e "${RED}❌ Log file not found${NC}"
        read -p "Press Enter to continue..."
        return
    fi

    echo "Analyzing last 1000 log entries..."
    echo ""

    # OAuth failures
    local oauth=$(tail -1000 "$LOG_DIR/main.log" | grep -c "oauth.*fail\|401\|invalid_token" || echo "0")
    echo -e "${YELLOW}OAuth Failures:${NC}      $oauth"

    # Connection failures
    local conn=$(tail -1000 "$LOG_DIR/main.log" | grep -c "timeout\|connection refused" || echo "0")
    echo -e "${YELLOW}Connection Fails:${NC}    $conn"

    # Command not found
    local cmd=$(tail -1000 "$LOG_DIR/main.log" | grep -c "not found\|no such file" || echo "0")
    echo -e "${YELLOW}Command Failures:${NC}    $cmd"

    # Docker issues
    local docker=$(tail -1000 "$LOG_DIR/main.log" | grep -c "docker.*error" || echo "0")
    echo -e "${YELLOW}Docker Issues:${NC}       $docker"

    echo ""
    read -p "Press Enter to continue..."
}

# [3] Check Quarantined Servers
check_quarantined() {
    echo -e "\n${BLUE}🔒 Quarantined Servers${NC}"
    echo "══════════════════════════════════"

    local quarantined=$(jq -r '[.mcpServers[] | select(.quarantined == true)] | length' "$CONFIG_FILE")

    if [ "$quarantined" -eq 0 ]; then
        echo -e "${GREEN}✅ No quarantined servers${NC}"
    else
        echo -e "${YELLOW}Found $quarantined quarantined server(s):${NC}"
        echo ""
        jq -r '.mcpServers[] | select(.quarantined == true) | "  - \(.name)"' "$CONFIG_FILE"
        echo ""
        echo "Review these servers in the tray UI before enabling."
    fi

    echo ""
    read -p "Press Enter to continue..."
}

# [4] Verify Dependencies
verify_deps() {
    echo -e "\n${BLUE}🔧 Dependency Verification${NC}"
    echo "══════════════════════════════════"

    echo "Checking required tools..."
    echo ""

    # Node.js
    if command -v node >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Node.js:${NC}    $(node --version)"
    else
        echo -e "${RED}❌ Node.js:${NC}    NOT FOUND"
    fi

    if command -v npm >/dev/null 2>&1; then
        echo -e "${GREEN}✅ npm:${NC}        $(npm --version)"
    else
        echo -e "${RED}❌ npm:${NC}        NOT FOUND"
    fi

    if command -v npx >/dev/null 2>&1; then
        echo -e "${GREEN}✅ npx:${NC}        Available"
    else
        echo -e "${RED}❌ npx:${NC}        NOT FOUND"
    fi

    # Python
    if command -v python3 >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Python:${NC}     $(python3 --version | cut -d' ' -f2)"
    else
        echo -e "${RED}❌ Python:${NC}     NOT FOUND"
    fi

    if command -v uvx >/dev/null 2>&1; then
        echo -e "${GREEN}✅ uvx:${NC}        Available"
    else
        echo -e "${YELLOW}⚠️  uvx:${NC}        NOT FOUND (install: pip install uv)"
    fi

    # Docker
    if command -v docker >/dev/null 2>&1; then
        if docker ps >/dev/null 2>&1; then
            echo -e "${GREEN}✅ Docker:${NC}     Running"
        else
            echo -e "${YELLOW}⚠️  Docker:${NC}     Installed but not running"
        fi
    else
        echo -e "${RED}❌ Docker:${NC}     NOT FOUND"
    fi

    echo ""
    read -p "Press Enter to continue..."
}

# [5] OAuth Server Status
oauth_status() {
    echo -e "\n${BLUE}🔐 OAuth Server Status${NC}"
    echo "══════════════════════════════════"

    local oauth_count=$(jq -r '[.mcpServers[] | select(.url != null)] | length' "$CONFIG_FILE")
    echo "Total OAuth/HTTP Servers: $oauth_count"
    echo ""

    echo "Enabled OAuth Servers:"
    jq -r '.mcpServers[] | select(.url != null and .enabled == true) | "  ✅ \(.name)"' "$CONFIG_FILE"
    echo ""

    echo "Disabled OAuth Servers:"
    jq -r '.mcpServers[] | select(.url != null and .enabled == false) | "  ❌ \(.name)"' "$CONFIG_FILE"
    echo ""

    echo "💡 To re-authenticate:"
    echo "   mcpproxy auth login --server=<name>"
    echo ""

    read -p "Press Enter to continue..."
}

# [6] Enable All Safe Servers
auto_recovery() {
    echo -e "\n${BLUE}🚀 Auto-Recovery (Enable All Safe Servers)${NC}"
    echo "══════════════════════════════════"

    local safe_count=$(jq -r '[.mcpServers[] | select(.enabled == false and (.quarantined == false or .quarantined == null))] | length' "$CONFIG_FILE")

    echo "This will enable $safe_count disabled servers"
    echo "(excluding quarantined servers)"
    echo ""
    read -p "Continue? [y/N] " -n 1 -r
    echo

    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if [ -f "$HOME/.mcpproxy/auto-recovery.sh" ]; then
            echo "Running auto-recovery script..."
            bash "$HOME/.mcpproxy/auto-recovery.sh"
            echo ""
            echo -e "${GREEN}✅ Recovery complete${NC}"
            echo "Restart mcpproxy for changes to take effect:"
            echo "  pkill mcpproxy && ./mcpproxy serve"
        else
            echo -e "${RED}❌ Auto-recovery script not found${NC}"
            echo "Run: ./scripts/diagnose-and-recover.sh first"
        fi
    else
        echo "Cancelled."
    fi

    echo ""
    read -p "Press Enter to continue..."
}

# [7] Configure Startup Script
configure_startup() {
    echo -e "\n${BLUE}⚙️  Startup Script Configuration${NC}"
    echo "══════════════════════════════════"

    if grep -q '"startup_script"' "$CONFIG_FILE" 2>/dev/null; then
        local enabled=$(jq -r '.startup_script.enabled // false' "$CONFIG_FILE")
        local path=$(jq -r '.startup_script.path // "NOT_SET"' "$CONFIG_FILE")
        echo "Current Status:"
        echo "  Configured: YES"
        echo "  Enabled:    $enabled"
        echo "  Path:       $path"
    else
        echo -e "${YELLOW}⚠️  Startup script NOT configured${NC}"
        echo ""
        echo "This means MCP servers won't auto-start at launch."
        echo ""
        echo "To configure, use:"
        echo "  mcpproxy startup_script update_config ..."
        echo ""
        echo "Or manually edit: $CONFIG_FILE"
    fi

    echo ""
    read -p "Press Enter to continue..."
}

# [8] View Recent Errors
view_errors() {
    echo -e "\n${BLUE}🔴 Recent Errors (Last 20)${NC}"
    echo "══════════════════════════════════"

    if [ ! -f "$LOG_DIR/main.log" ]; then
        echo -e "${RED}❌ Log file not found${NC}"
        read -p "Press Enter to continue..."
        return
    fi

    grep "ERROR" "$LOG_DIR/main.log" | tail -20 || echo "No errors found"

    echo ""
    read -p "Press Enter to continue..."
}

# [9] Generate Full Report
generate_report() {
    echo -e "\n${BLUE}📄 Generating Full Diagnostic Report${NC}"
    echo "══════════════════════════════════"

    local report_file="$HOME/.mcpproxy/diagnostic-report-$(date +%Y%m%d-%H%M%S).txt"

    {
        echo "MCPProxy Diagnostic Report"
        echo "Generated: $(date)"
        echo "══════════════════════════════════"
        echo ""

        echo "SERVER STATUS:"
        jq -r '.mcpServers | length' "$CONFIG_FILE" | xargs echo "Total:"
        jq -r '[.mcpServers[] | select(.enabled == true)] | length' "$CONFIG_FILE" | xargs echo "Enabled:"
        jq -r '[.mcpServers[] | select(.enabled == false)] | length' "$CONFIG_FILE" | xargs echo "Disabled:"
        echo ""

        echo "QUARANTINED SERVERS:"
        jq -r '.mcpServers[] | select(.quarantined == true) | .name' "$CONFIG_FILE"
        echo ""

        echo "RECENT ERRORS:"
        grep "ERROR" "$LOG_DIR/main.log" 2>/dev/null | tail -50 || echo "No errors"

    } > "$report_file"

    echo -e "${GREEN}✅ Report saved to:${NC}"
    echo "   $report_file"
    echo ""

    read -p "Press Enter to continue..."
}

# Main loop
main() {
    while true; do
        clear
        echo -e "${BLUE}╔══════════════════════════════════════════╗${NC}"
        echo -e "${BLUE}║  MCPProxy Interactive Diagnostic Agent  ║${NC}"
        echo -e "${BLUE}╚══════════════════════════════════════════╝${NC}"
        echo ""

        show_menu
        read -r choice

        case $choice in
            1) server_status ;;
            2) analyze_failures ;;
            3) check_quarantined ;;
            4) verify_deps ;;
            5) oauth_status ;;
            6) auto_recovery ;;
            7) configure_startup ;;
            8) view_errors ;;
            9) generate_report ;;
            0) echo "Goodbye!"; exit 0 ;;
            *) echo "Invalid option"; sleep 1 ;;
        esac
    done
}

main "$@"
