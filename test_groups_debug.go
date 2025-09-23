package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"mcpproxy-go/internal/config"
)

func main() {
	fmt.Println("=== Testing Group Loading ===")
	
	// Get config path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		return
	}
	
	configPath := filepath.Join(homeDir, ".mcpproxy", "mcp_config.json")
	fmt.Printf("Config path: %s\n", configPath)
	
	// Load config using the config package
	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}
	
	fmt.Printf("Found %d groups in config:\n", len(cfg.Groups))
	for i, group := range cfg.Groups {
		fmt.Printf("  [%d] %s (ID: %d, Color: %s, Enabled: %v)\n", 
			i, group.Name, group.ID, group.Color, group.Enabled)
	}
	
	fmt.Println("\n=== Testing Raw Config Reading ===")
	
	// Read raw config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		return
	}

	var configData map[string]interface{}
	if err := json.Unmarshal(data, &configData); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}

	if groups, ok := configData["groups"].([]interface{}); ok {
		fmt.Printf("Found %d groups in raw config:\n", len(groups))
		for i, groupInterface := range groups {
			if group, ok := groupInterface.(map[string]interface{}); ok {
				name, _ := group["name"].(string)
				color, _ := group["color"].(string)
				colorEmoji, _ := group["color_emoji"].(string)
				id, _ := group["id"].(float64)
				enabled, _ := group["enabled"].(bool)
				
				fmt.Printf("  [%d] %s (ID: %.0f, Color: %s, Emoji: %s, Enabled: %v)\n", 
					i, name, id, color, colorEmoji, enabled)
			}
		}
	} else {
		fmt.Println("No groups found in raw config")
	}
	
	fmt.Println("\n=== Testing Server Assignments ===")
	
	// Check server group assignments
	if servers, ok := configData["mcpServers"].([]interface{}); ok {
		fmt.Printf("Found %d servers in config:\n", len(servers))
		for i, serverInterface := range servers {
			if server, ok := serverInterface.(map[string]interface{}); ok {
				name, _ := server["name"].(string)
				groupID, _ := server["group_id"].(float64)
				groupName, _ := server["group_name"].(string)
				
				fmt.Printf("  [%d] %s (GroupID: %.0f, GroupName: %s)\n", 
					i, name, groupID, groupName)
			}
		}
	} else {
		fmt.Println("No servers found in config")
	}
}
