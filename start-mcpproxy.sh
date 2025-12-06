#!/bin/bash
# MCPProxy Startup Script
# This script must be run manually in a Terminal window to enable system tray
#
# Usage: ./start-mcpproxy.sh
# Auto Start on Login
# Create a plist file in ~/Library/LaunchAgents/com.smartmcpproxy.mcpproxy.plist
cd "$(dirname "$0")"

echo "🚀 Starting MCPProxy with system tray..."
echo ""
echo "✅ HTTP server will be available at: http://localhost:8080"
echo "✅ System tray icon will appear in your menu bar"
echo ""
echo "📋 161 upstream servers configured"
echo ""
echo "Press Ctrl+C to stop the server"
echo ""
echo "─────────────────────────────────────────────────────────"
echo ""

# Start mcpproxy with tray enabled
./mcpproxy serve --tray
