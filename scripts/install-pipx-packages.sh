#!/bin/bash

# MCPProxy PIPX Package Pre-Installation Script
# This script pre-installs PIPX-based MCP server packages

set -e

echo "🚀 Starting PIPX MCP Package Installation..."
echo "📦 This will install 4 packages and may take 2-3 minutes"
echo ""

# Color output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if pipx is installed
if ! command -v pipx &> /dev/null; then
    echo "❌ pipx is not installed!"
    echo "Install with: python3 -m pip install --user pipx"
    echo "Or: brew install pipx"
    exit 1
fi

echo "✅ pipx found at: $(which pipx)"
echo ""

# Track installation progress
INSTALLED=0
FAILED=0

install_package() {
    local package=$1
    echo -e "${YELLOW}Installing: ${package}${NC}"
    if pipx install "$package" 2>&1; then
        echo -e "${GREEN}✅ Installed: ${package}${NC}"
        ((INSTALLED++))
    else
        echo -e "❌ Failed: ${package}"
        ((FAILED++))
    fi
    echo ""
}

# Install all PIPX packages
echo "📍 Installing PIPX MCP Servers"
install_package "mcp-server-bigquery"
install_package "mcp-server-duckdb"
install_package "mcp-server-motherduck"
install_package "toolfront"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 Installation Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}✅ Installed: ${INSTALLED} packages${NC}"
if [ $FAILED -gt 0 ]; then
    echo -e "❌ Failed: ${FAILED} packages"
fi
echo ""
echo "🔍 Verifying installations..."
pipx list
echo ""
echo "✅ PIPX package installation complete!"
echo ""
echo "Expected impact: +4 connected servers"
echo ""
echo "Next step:"
echo "Restart mcpproxy: pkill mcpproxy && ./mcpproxy serve"
