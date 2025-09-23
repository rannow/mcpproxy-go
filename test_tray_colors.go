package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	// Take screenshot of system tray
	fmt.Println("Testing tray colors...")
	
	// Use AppleScript to click on system tray and capture
	script := `
	tell application "System Events"
		-- Click on MCPProxy tray icon (this might need adjustment)
		click menu bar item "MCPProxy" of menu bar 1 of application process "MCPProxy"
		delay 1
	end tell
	`
	
	cmd := exec.Command("osascript", "-e", script)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Could not interact with tray: %v\n", err)
	}
	
	time.Sleep(2)
	
	// Take screenshot
	exec.Command("screencapture", "-x", "/tmp/tray_test.png").Run()
	fmt.Println("Screenshot saved to /tmp/tray_test.png")
	fmt.Println("Please check if the colors in the tray menu match:")
	fmt.Println("🟠 To Test (Orange)")
	fmt.Println("🟣 AWS Services (Purple)")  
	fmt.Println("🟢 OK (Green)")
	fmt.Println("🩷 To Update (Pink)")
	fmt.Println("🟡 Neu (Yellow)")
}
