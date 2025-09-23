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
		fmt.Printf("Error: %v\n", err)
		return
	}

	var configData map[string]interface{}
	json.Unmarshal(data, &configData)

	fmt.Println("=== Expected Tray Menu Titles ===")
	if groups, ok := configData["groups"].([]interface{}); ok {
		for _, groupInterface := range groups {
			if group, ok := groupInterface.(map[string]interface{}); ok {
				name, _ := group["name"].(string)
				colorEmoji, _ := group["color_emoji"].(string)
				
				// Simulate menu title creation (like in tray.go lines 1938, 2041, 2075, 3015)
				menuTitle := fmt.Sprintf("%s %s (%d servers)", colorEmoji, name, 0)
				fmt.Printf("'%s'\n", menuTitle)
			}
		}
	}
}
