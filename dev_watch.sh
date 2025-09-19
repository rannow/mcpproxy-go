#!/bin/bash

# MCPProxy Development Auto-Restart Script

APP_PID=""

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

# Watch for changes
fswatch -o . --exclude='\.git' --exclude='\.log' --exclude='dev_watch\.sh' | while read num; do
    echo "Files changed, restarting..."
    start_app
done
