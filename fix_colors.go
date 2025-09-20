package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func main() {
	configPath := "/Users/hrannow/.mcpproxy/mcp_config.json"
	
	// Read config
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		return
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}

	// Fix groups - ensure color_emoji is set
	if groups, ok := config["groups"].([]interface{}); ok {
		for _, groupInterface := range groups {
			if group, ok := groupInterface.(map[string]interface{}); ok {
				color, _ := group["color"].(string)
				if color != "" {
					group["color_emoji"] = getColorEmoji(color)
					fmt.Printf("Fixed group %s with color %s -> %s\n", 
						group["name"], color, group["color_emoji"])
				}
			}
		}
	}

	// Write back
	output, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}

	if err := os.WriteFile(configPath, output, 0644); err != nil {
		fmt.Printf("Error writing config: %v\n", err)
		return
	}

	fmt.Println("Config fixed successfully!")
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
