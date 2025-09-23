package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func main() {
	// Read config
	data, err := os.ReadFile("/Users/hrannow/.mcpproxy/mcp_config.json")
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		return
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}

	// Check groups
	if groups, ok := config["groups"].([]interface{}); ok {
		fmt.Printf("Found %d groups:\n", len(groups))
		for _, groupInterface := range groups {
			if group, ok := groupInterface.(map[string]interface{}); ok {
				name := group["name"].(string)
				color := group["color"].(string)
				emoji := getColorEmoji(color)
				
				fmt.Printf("  %s %s (color: %s)\n", emoji, name, color)
			}
		}
	} else {
		fmt.Println("No groups found in config")
	}
}

func getColorEmoji(hex string) string {
	switch strings.ToLower(hex) {
	case "#e83e8c": return "🩷" // Pink
	case "#ffc107": return "🟡" // Yellow
	case "#fd7e14": return "🟠" // Orange
	case "#6610f2": return "🟣" // Purple
	case "#28a745": return "🟢" // Green
	default: return "🔵" // Blue
	}
}
