package main

import (
	"fmt"
	"path/filepath"
	"os"
	"./internal/config"
)

func main() {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".mcpproxy", "mcp_config.json")
	fmt.Printf("Loading config from: %s\n", configPath)
	
	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}
	
	fmt.Printf("Total servers in config: %d\n", len(cfg.Servers))
	
	enabled := 0
	disabled := 0
	for _, server := range cfg.Servers {
		if server.Enabled {
			enabled++
		} else {
			disabled++
		}
	}
	fmt.Printf("Enabled servers: %d\n", enabled)
	fmt.Printf("Disabled servers: %d\n", disabled)
}
