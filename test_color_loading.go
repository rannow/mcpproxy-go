package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	configPath := "/Users/hrannow/.mcpproxy/mcp_config.json"
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		return
	}

	var configData map[string]interface{}
	if err := json.Unmarshal(data, &configData); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}

	if groups, ok := configData["groups"].([]interface{}); ok {
		fmt.Printf("Found %d groups in config:\n", len(groups))
		for _, groupInterface := range groups {
			if group, ok := groupInterface.(map[string]interface{}); ok {
				name, _ := group["name"].(string)
				color, _ := group["color"].(string)
				colorEmoji, _ := group["color_emoji"].(string)
				enabled, _ := group["enabled"].(bool)
				
				fmt.Printf("  %s %s (color: %s, emoji: %s, enabled: %v)\n", 
					colorEmoji, name, color, colorEmoji, enabled)
			}
		}
	} else {
		fmt.Println("No groups found in config")
	}
}
