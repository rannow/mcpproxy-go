#!/bin/bash

# Permanent color fix for MCPProxy
CONFIG_FILE="/Users/hrannow/.mcpproxy/mcp_config.json"

# Add color_emoji fields using jq
jq '
.groups |= map(
  if .color == "#e83e8c" then . + {"color_emoji": "🩷"}
  elif .color == "#ffc107" then . + {"color_emoji": "🟡"}
  elif .color == "#fd7e14" then . + {"color_emoji": "🟠"}
  elif .color == "#6610f2" then . + {"color_emoji": "🟣"}
  elif .color == "#28a745" then . + {"color_emoji": "🟢"}
  else . + {"color_emoji": "🔵"}
  end
)
' "$CONFIG_FILE" > "$CONFIG_FILE.tmp" && mv "$CONFIG_FILE.tmp" "$CONFIG_FILE"

echo "Color emojis restored permanently"
