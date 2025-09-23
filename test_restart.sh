#!/bin/bash

echo "=== MCPProxy Color Persistence Test ==="

# Kill any existing MCPProxy processes
pkill -f mcpproxy 2>/dev/null
sleep 2

echo "1. Starting MCPProxy..."
./mcpproxy serve --tray=false --log-level=debug > test_output.log 2>&1 &
MCPPROXY_PID=$!

echo "   PID: $MCPPROXY_PID"
sleep 5

echo "2. Checking if groups loaded correctly..."
go run debug_groups.go | grep "ColorEmoji:" | head -5

echo "3. Stopping MCPProxy..."
kill $MCPPROXY_PID 2>/dev/null
sleep 2

echo "4. Restarting MCPProxy..."
./mcpproxy serve --tray=false --log-level=debug > test_output2.log 2>&1 &
MCPPROXY_PID=$!

sleep 5

echo "5. Checking groups after restart..."
go run debug_groups.go | grep "ColorEmoji:" | head -5

echo "6. Cleanup..."
kill $MCPPROXY_PID 2>/dev/null

echo "=== Test Complete ==="
