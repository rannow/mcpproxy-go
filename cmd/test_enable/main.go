package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/smart-mcp-proxy/mcpproxy-go/internal/storage"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".mcpproxy")
	sm, err := storage.NewManager(dataDir, sugar)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer sm.Close()

	serverName := "Container User"
	
	fmt.Println("=== BEFORE ENABLE ===")
	server, _ := sm.GetUpstreamServer(serverName)
	fmt.Printf("Enabled: %v, AutoDisabled: %v, Reason: %s\n", server.Enabled, server.AutoDisabled, server.AutoDisableReason)

	fmt.Println("\n=== ENABLING SERVER ===")
	sm.EnableUpstreamServer(serverName, true)

	fmt.Println("\n=== AFTER ENABLE ===")
	server, _ = sm.GetUpstreamServer(serverName)
	fmt.Printf("Enabled: %v, AutoDisabled: %v, Reason: %s\n", server.Enabled, server.AutoDisabled, server.AutoDisableReason)

	if server.Enabled && !server.AutoDisabled && server.AutoDisableReason == "" {
		fmt.Println("\n✅ FIX VERIFIED: All states properly cleared!")
	} else {
		fmt.Println("\n❌ FIX FAILED!")
		os.Exit(1)
	}
}
