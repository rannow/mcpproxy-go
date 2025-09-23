package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	configPath := "/Users/hrannow/.mcpproxy/mcp_config.json"
	
	// Read config
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Force correct colors
	if groups, ok := config["groups"].([]interface{}); ok {
		for _, groupInterface := range groups {
			if group, ok := groupInterface.(map[string]interface{}); ok {
				name, _ := group["name"].(string)
				color, _ := group["color"].(string)
				
				// Force correct color_emoji based on color
				switch color {
				case "#e83e8c":
					group["color_emoji"] = "🩷"
					fmt.Printf("Fixed %s -> 🩷\n", name)
				case "#ffc107":
					group["color_emoji"] = "🟡"
					fmt.Printf("Fixed %s -> 🟡\n", name)
				case "#fd7e14":
					group["color_emoji"] = "🟠"
					fmt.Printf("Fixed %s -> 🟠\n", name)
				case "#6610f2":
					group["color_emoji"] = "🟣"
					fmt.Printf("Fixed %s -> 🟣\n", name)
				case "#28a745":
					group["color_emoji"] = "🟢"
					fmt.Printf("Fixed %s -> 🟢\n", name)
				}
			}
		}
	}

	// Write back
	output, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(configPath, output, 0644)
	fmt.Println("Colors forced in config!")
}
