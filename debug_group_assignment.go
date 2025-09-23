package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	fmt.Println("=== Debug Group Assignment ===")
	
	// Start mcpproxy in background
	fmt.Println("Starting mcpproxy...")
	cmd := exec.Command("./mcpproxy", "--tray=false")
	cmd.Start()
	
	// Wait for server to start
	fmt.Println("Waiting for server to start...")
	time.Sleep(5 * time.Second)
	
	// Check if server is running
	fmt.Println("Checking server status...")
	statusCmd := exec.Command("curl", "-s", "http://localhost:8080/api/groups")
	output, err := statusCmd.Output()
	if err != nil {
		fmt.Printf("Error checking server: %v\n", err)
		cmd.Process.Kill()
		return
	}
	
	fmt.Printf("Server response: %s\n", string(output))
	
	// Stop server
	fmt.Println("Stopping server...")
	cmd.Process.Kill()
}
