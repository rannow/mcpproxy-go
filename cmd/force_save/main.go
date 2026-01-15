package main

import (
	"fmt"
	"os"
	"path/filepath"

	"mcpproxy-go/internal/config"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get home directory: %v\n", err)
		os.Exit(1)
	}

	configPath := filepath.Join(homeDir, ".mcpproxy", "mcp_config.json")

	fmt.Printf("Loading config from: %s\n", configPath)

	// Load existing config
	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %d servers\n", len(cfg.Servers))

	// Re-save the config - this will use the new JSON tags with json:"-"
	fmt.Printf("Re-saving config to apply new JSON serialization rules...\n")
	if err := config.SaveConfig(cfg, configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Config re-saved successfully!\n")
	fmt.Printf("Deprecated fields (enabled, quarantined, start_on_boot, auto_disabled) should now be removed from JSON.\n")
}
