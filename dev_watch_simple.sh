#!/bin/bash

# MCPProxy Development Auto-Restart (No fswatch required)

APP_PID=""
LAST_CHANGE=0

cleanup() {
    if [ ! -z "$APP_PID" ]; then
        kill $APP_PID 2>/dev/null
    fi
    exit 0
}

trap cleanup SIGINT SIGTERM

start_app() {
    if [ ! -z "$APP_PID" ]; then
        kill $APP_PID 2>/dev/null
        wait $APP_PID 2>/dev/null
    fi
    
    echo "Starting mcpproxy..."
    go run ./cmd/mcpproxy serve &
    APP_PID=$!
}

# Initial start
start_app

# Watch for changes (polling)
while true; do
    CURRENT_CHANGE=$(find . -name "*.go" -newer /tmp/mcpproxy_watch 2>/dev/null | wc -l)
    
    if [ $CURRENT_CHANGE -gt 0 ]; then
        touch /tmp/mcpproxy_watch
        echo "Go files changed, restarting..."
        start_app
    fi
    
    sleep 2
done
