#!/bin/bash

# Test script to verify group persistence to config file

echo "Testing group persistence to config file..."

# Create a temporary test directory
TEST_DIR="/tmp/mcpproxy-test-$$"
mkdir -p "$TEST_DIR"

echo "Test directory: $TEST_DIR"

# Use a different port to avoid conflicts
TEST_PORT="8081"

# Start MCPProxy in background with test config
./mcpproxy-test serve --data-dir="$TEST_DIR" --listen=":$TEST_PORT" --tray=false --log-level=info &
MCPPROXY_PID=$!

echo "Started MCPProxy with PID: $MCPPROXY_PID on port $TEST_PORT"

# Wait for server to start
sleep 5

# Check if config file exists
CONFIG_FILE="$TEST_DIR/mcp_config.json"
if [ ! -f "$CONFIG_FILE" ]; then
    echo "ERROR: Config file not created at $CONFIG_FILE"
    kill $MCPPROXY_PID 2>/dev/null
    rm -rf "$TEST_DIR"
    exit 1
fi

echo "Config file created successfully"

# Test creating a group via API
echo "Creating test group via API..."
curl -s -X POST http://localhost:$TEST_PORT/api/groups \
  -H "Content-Type: application/json" \
  -d '{"name": "Test Group", "color": "#123456"}' > /dev/null

# Wait a moment for the save to complete
sleep 3

# Check if the group was saved to config file
echo "Checking if group was saved to config file..."
if grep -q "Test Group" "$CONFIG_FILE"; then
    echo "SUCCESS: Group was saved to config file!"
    grep -A 3 -B 1 "Test Group" "$CONFIG_FILE"
else
    echo "ERROR: Group was not saved to config file"
    echo "Config file contents:"
    cat "$CONFIG_FILE"
fi

# Clean up
echo "Cleaning up..."
kill $MCPPROXY_PID 2>/dev/null
sleep 1
rm -rf "$TEST_DIR"

echo "Test completed."
