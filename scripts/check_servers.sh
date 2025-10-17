#!/bin/bash

echo "Checking MCP servers for missing packages..."
echo "============================================"

# Extract server configurations that use npx
grep -A 8 '"command": "npx"' ~/.mcpproxy/mcp_config.json | grep -E '"name"|"args"' | paste - - | while read line; do
    name=$(echo "$line" | grep -o '"name": "[^"]*"' | cut -d'"' -f4)
    package=$(echo "$line" | grep -o '\["[^"]*"' | cut -d'"' -f2 | head -1)
    
    if [[ "$package" == "-y" ]]; then
        package=$(echo "$line" | grep -o '"[^"]*"' | tail -1 | cut -d'"' -f2)
    fi
    
    if [[ -n "$package" && "$package" != "-y" ]]; then
        echo -n "Testing $name ($package): "
        if timeout 5s npm view "$package" version >/dev/null 2>&1; then
            echo "✅ OK"
        else
            echo "❌ NOT FOUND"
        fi
    fi
done
