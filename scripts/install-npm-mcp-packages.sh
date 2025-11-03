#!/bin/bash

# MCPProxy NPM Package Pre-Installation Script
# This script pre-installs commonly used MCP server packages to avoid timeout issues

set -e

echo "🚀 Starting NPM MCP Package Installation..."
echo "📦 This will install ~20 packages and may take 5-10 minutes"
echo ""

# Color output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track installation progress
INSTALLED=0
FAILED=0

install_package() {
    local package=$1
    echo -e "${YELLOW}Installing: ${package}${NC}"
    if npm install -g "$package" 2>&1 | grep -v "npm WARN"; then
        echo -e "${GREEN}✅ Installed: ${package}${NC}"
        ((INSTALLED++))
    else
        echo -e "❌ Failed: ${package}"
        ((FAILED++))
    fi
    echo ""
}

# Priority 1 - Most commonly used packages
echo "📍 Priority 1: Core MCP Servers"
install_package "@modelcontextprotocol/server-brave-search"
install_package "@modelcontextprotocol/server-filesystem"
install_package "@modelcontextprotocol/server-github"
install_package "@modelcontextprotocol/server-sqlite"
install_package "@modelcontextprotocol/server-postgres"

# Priority 2 - AWS and Cloud Services
echo "📍 Priority 2: AWS and Cloud Services"
install_package "@aws/athena-mcp-server"
install_package "@aws/s3-mcp-server"

# Priority 3 - Integration Services
echo "📍 Priority 3: Integration Services"
install_package "@airtable/airtable-mcp-server"
install_package "mcp-discord"
install_package "mcp-graphql"
install_package "mcp-postman"

# Priority 4 - Development Tools
echo "📍 Priority 4: Development Tools"
install_package "browserless-mcp-server"
install_package "browsermcp"
install_package "e2b-mcp-server"
install_package "enhanced-memory-mcp"

# Priority 5 - Database Tools
echo "📍 Priority 5: Database Tools"
install_package "dbhub-universal"
install_package "elasticsearch-mcp-server"

# Priority 6 - Additional Utilities
echo "📍 Priority 6: Additional Utilities"
install_package "auto-mcp"
install_package "mcp-obsidian"

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
npm list -g --depth=0 2>/dev/null | grep -E "(mcp-|@modelcontextprotocol|@aws.*-mcp|@airtable)" || echo "No MCP packages found"
echo ""
echo "✅ NPM package installation complete!"
echo ""
echo "Next steps:"
echo "1. Install PIPX packages: ./scripts/install-pipx-packages.sh"
echo "2. Restart mcpproxy: pkill mcpproxy && ./mcpproxy serve"
