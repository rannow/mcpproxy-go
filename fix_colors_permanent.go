package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	configPath := "/Users/hrannow/.mcpproxy/mcp_config.json"
	
	for {
		// Read config
		data, err := os.ReadFile(configPath)
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}

		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err != nil {
			time.Sleep(10 * time.Second)
			continue
		}

		// Check if color_emoji fields exist
		needsFix := false
		if groups, ok := config["groups"].([]interface{}); ok {
			for _, groupInterface := range groups {
				if group, ok := groupInterface.(map[string]interface{}); ok {
					if _, hasColorEmoji := group["color_emoji"]; !hasColorEmoji {
						needsFix = true
						color, _ := group["color"].(string)
						switch color {
						case "#e83e8c":
							group["color_emoji"] = "🩷"
						case "#ffc107":
							group["color_emoji"] = "🟡"
						case "#fd7e14":
							group["color_emoji"] = "🟠"
						case "#6610f2":
							group["color_emoji"] = "🟣"
						case "#28a745":
							group["color_emoji"] = "🟢"
						default:
							group["color_emoji"] = "🔵"
						}
					}
				}
			}
		}

		if needsFix {
			// Write back
			output, _ := json.MarshalIndent(config, "", "  ")
			os.WriteFile(configPath, output, 0644)
			
			// Restart mcpproxy
			exec.Command("pkill", "-f", "mcpproxy").Run()
			time.Sleep(2 * time.Second)
			exec.Command("nohup", "/Users/hrannow/Library/CloudStorage/OneDrive-Persönlich/workspace/mcp-server/mcpproxy-go/mcpproxy", "serve").Start()
		}

		time.Sleep(30 * time.Second)
	}
}
