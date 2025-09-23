#!/bin/bash

# Monitor and auto-fix color emojis
CONFIG_FILE="/Users/hrannow/.mcpproxy/mcp_config.json"

while true; do
    # Check if color_emoji fields exist
    if ! grep -q "color_emoji" "$CONFIG_FILE"; then
        echo "$(date): Color emojis missing, restoring..."
        /Users/hrannow/Library/CloudStorage/OneDrive-Persönlich/workspace/mcp-server/mcpproxy-go/permanent_color_fix.sh
        
        # Restart MCPProxy to reload
        pkill -f mcpproxy-fixed
        sleep 1
        cd /Users/hrannow/Library/CloudStorage/OneDrive-Persönlich/workspace/mcp-server/mcpproxy-go
        nohup ./mcpproxy-fixed serve > /dev/null 2>&1 &
    fi
    
    sleep 30
done
