package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	fmt.Println("=== Debug Tray Groups ===")
	
	// Get config path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		return
	}
	
	configPath := filepath.Join(homeDir, ".mcpproxy", "mcp_config.json")
	fmt.Printf("Config path: %s\n", configPath)
	
	// Read the current config file
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

	// Load groups from config (simulating the tray logic)
	serverGroups := make(map[string]*ServerGroup)
	
	if groups, ok := configData["groups"].([]interface{}); ok && len(groups) > 0 {
		fmt.Printf("Found %d groups in config\n", len(groups))
		
		for _, groupInterface := range groups {
			if group, ok := groupInterface.(map[string]interface{}); ok {
				name, nameOk := group["name"].(string)
				if !nameOk || name == "" {
					continue
				}

				// Get ID from config, or generate one if missing
				id, idOk := group["id"].(float64) // JSON numbers are float64
				if !idOk {
					id = 1 // Default ID
				}

				description, _ := group["description"].(string)
				color, _ := group["color"].(string)
				colorEmoji, _ := group["color_emoji"].(string)
				enabled, _ := group["enabled"].(bool)
				
				// Set defaults
				if description == "" {
					description = fmt.Sprintf("Custom group: %s", name)
				}
				if color == "" {
					color = "#6c757d"
				}
				if colorEmoji == "" {
					colorEmoji = getColorEmojiForHex(color)
				}
				
				serverGroups[name] = &ServerGroup{
					ID:          int(id),
					Name:        name,
					Description: description,
					Color:       color,
					ColorEmoji:  colorEmoji,
					ServerNames: make([]string, 0),
					Enabled:     enabled,
				}

				fmt.Printf("  Loaded group: %s (ID: %d, Color: %s, Emoji: %s, Enabled: %v)\n", 
					name, int(id), color, colorEmoji, enabled)
			}
		}
	}

	// Now populate server assignments
	if servers, ok := configData["mcpServers"].([]interface{}); ok {
		fmt.Printf("\nProcessing %d servers for group assignments\n", len(servers))
		
		for _, serverInterface := range servers {
			if server, ok := serverInterface.(map[string]interface{}); ok {
				if serverName, ok := server["name"].(string); ok {
					// Check for group_id (new format) or group_name (legacy format)
					var targetGroup *ServerGroup
					
					if groupID, ok := server["group_id"].(float64); ok && groupID != 0 {
						// New format: use group_id
						targetGroup = getGroupByID(serverGroups, int(groupID))
						fmt.Printf("  Server %s -> GroupID %d -> Group: %v\n", serverName, int(groupID), targetGroup != nil)
					} else if groupName, ok := server["group_name"].(string); ok && groupName != "" {
						// Legacy format: use group_name
						targetGroup = getGroupByName(serverGroups, groupName)
						fmt.Printf("  Server %s -> GroupName %s -> Group: %v\n", serverName, groupName, targetGroup != nil)
					}
					
					if targetGroup != nil {
						// Check if server is not already in the group
						found := false
						for _, existingServer := range targetGroup.ServerNames {
							if existingServer == serverName {
								found = true
								break
							}
						}
						if !found {
							targetGroup.ServerNames = append(targetGroup.ServerNames, serverName)
							fmt.Printf("    Added %s to group %s (%s)\n", serverName, targetGroup.Name, targetGroup.ColorEmoji)
						}
					}
				}
			}
		}
	}

	// Show final group assignments
	fmt.Println("\n=== Final Group Assignments ===")
	for groupName, group := range serverGroups {
		if group.Enabled {
			fmt.Printf("Group: %s %s (%d servers)\n", group.ColorEmoji, groupName, len(group.ServerNames))
			for _, serverName := range group.ServerNames {
				fmt.Printf("  - %s\n", serverName)
			}
		}
	}

	// Test the getServerStatusDisplay logic
	fmt.Println("\n=== Testing getServerStatusDisplay Logic ===")
	testServerName := "Browser-Tools-MCP"
	fmt.Printf("Testing server: %s\n", testServerName)
	
	var groupIcon string
	var groupInfo string
	if serverGroups != nil {
		for groupName, group := range serverGroups {
			if group.Enabled {
				for _, groupServerName := range group.ServerNames {
					if groupServerName == testServerName {
						groupIcon = group.ColorEmoji
						groupInfo = fmt.Sprintf(" [%s %s]", groupIcon, groupName)
						fmt.Printf("  Found server in group: %s %s\n", group.ColorEmoji, groupName)
						break
					}
				}
			}
			if groupIcon != "" {
				break
			}
		}
	}
	
	if groupIcon != "" {
		fmt.Printf("  Display text would be: 🟢 %s %s\n", groupIcon, testServerName)
	} else {
		fmt.Printf("  No group found for server %s\n", testServerName)
		fmt.Printf("  Display text would be: 🟢 %s\n", testServerName)
	}
}

type ServerGroup struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Color       string   `json:"color"`       // Color emoji or hex code
	ColorEmoji  string   `json:"color_emoji"` // Color emoji for display
	Description string   `json:"description"`
	ServerNames []string `json:"server_names"`
	Enabled     bool     `json:"enabled"`
}

func getGroupByID(groups map[string]*ServerGroup, id int) *ServerGroup {
	for _, group := range groups {
		if group.ID == id {
			return group
		}
	}
	return nil
}

func getGroupByName(groups map[string]*ServerGroup, name string) *ServerGroup {
	return groups[name]
}

func getColorEmojiForHex(hex string) string {
	switch hex {
	case "#ff9900": return "🟠" // AWS Orange
	case "#28a745": return "🟢" // Green
	case "#dc3545": return "🔴" // Red
	case "#6f42c1": return "🟣" // Purple
	case "#6610f2": return "🟣" // Purple (variant)
	case "#fd7e14": return "🟠" // Orange
	case "#20c997": return "🟢" // Teal (green)
	case "#e83e8c": return "🩷" // Pink
	case "#ffc107": return "🟡" // Yellow
	case "#6c757d": return "⚫" // Gray (black)
	case "#343a40": return "⚫" // Dark
	default: 
		return "⚫" // Gray
	}
}
