#!/bin/bash

# MCPProxy Auto-Update Script

# Get current version
CURRENT_VERSION=$(mcpproxy --version 2>/dev/null | grep -o 'v[0-9]\+\.[0-9]\+\.[0-9]\+' || echo "v0.0.0")

# Get latest version from GitHub
LATEST_VERSION=$(curl -s https://api.github.com/repos/smart-mcp-proxy/mcpproxy-go/releases/latest | grep '"tag_name"' | cut -d'"' -f4)

echo "Current: $CURRENT_VERSION"
echo "Latest: $LATEST_VERSION"

if [ "$CURRENT_VERSION" != "$LATEST_VERSION" ]; then
    echo "New version available. Updating..."
    
    # Stop MCPProxy
    pkill -f mcpproxy
    
    # Update via Homebrew (if installed that way)
    if command -v brew &> /dev/null && brew list mcpproxy &> /dev/null; then
        brew upgrade mcpproxy
    else
        # Update via go install
        go install github.com/smart-mcp-proxy/mcpproxy-go/cmd/mcpproxy@latest
    fi
    
    # Restart MCPProxy
    nohup mcpproxy serve > /dev/null 2>&1 &
    
    echo "Updated to $LATEST_VERSION and restarted."
else
    echo "Already up to date."
fi
