package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type ServerGroup struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Color       string   `json:"color"`
	ColorEmoji  string   `json:"color_emoji"`
	Description string   `json:"description"`
	ServerNames []string `json:"server_names"`
	Enabled     bool     `json:"enabled"`
}

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

	serverGroups := make(map[string]*ServerGroup)

	if groups, ok := configData["groups"].([]interface{}); ok {
		fmt.Printf("Loading %d groups from config:\n", len(groups))
		for _, groupInterface := range groups {
			if group, ok := groupInterface.(map[string]interface{}); ok {
				name, _ := group["name"].(string)
				id, _ := group["id"].(float64)
				description, _ := group["description"].(string)
				color, _ := group["color"].(string)
				colorEmoji, _ := group["color_emoji"].(string)
				enabled, _ := group["enabled"].(bool)
				
				fmt.Printf("  Group: %s\n", name)
				fmt.Printf("    ID: %d\n", int(id))
				fmt.Printf("    Color: %s\n", color)
				fmt.Printf("    ColorEmoji: '%s'\n", colorEmoji)
				fmt.Printf("    Enabled: %v\n", enabled)
				
				serverGroups[name] = &ServerGroup{
					ID:          int(id),
					Name:        name,
					Description: description,
					Color:       color,
					ColorEmoji:  colorEmoji,
					ServerNames: make([]string, 0),
					Enabled:     enabled,
				}
				fmt.Printf("    Stored ColorEmoji: '%s'\n\n", serverGroups[name].ColorEmoji)
			}
		}
	}

	fmt.Println("=== Testing Menu Title Generation ===")
	for groupName, group := range serverGroups {
		menuTitle := fmt.Sprintf("%s %s (%d servers)", group.ColorEmoji, groupName, len(group.ServerNames))
		fmt.Printf("Group: %s -> Menu: '%s'\n", groupName, menuTitle)
	}
}
