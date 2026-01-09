package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	mcpProxyURL    = "http://localhost:8080/mcp"
	timeoutSeconds = 60
)

type TestCase struct {
	Name      string
	Tool      string
	Arguments map[string]interface{}
}

func main() {
	fmt.Println("======================================================================")
	fmt.Printf("MCP Tools Timeout Tester - %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("Server: %s\n", mcpProxyURL)
	fmt.Printf("Timeout: %ds per request\n", timeoutSeconds)
	fmt.Println("======================================================================")
	fmt.Println()

	ctx := context.Background()

	fmt.Println("[*] Connecting to mcpproxy via Streamable HTTP...")
	httpTransport, err := transport.NewStreamableHTTP(mcpProxyURL)
	if err != nil {
		fmt.Printf("[-] Failed to create transport: %v\n", err)
		os.Exit(1)
	}

	// Create client
	mcpClient := client.NewClient(httpTransport)
	defer mcpClient.Close()

	// Start the client
	if err := mcpClient.Start(ctx); err != nil {
		fmt.Printf("[-] Failed to start client: %v\n", err)
		os.Exit(1)
	}

	// Initialize connection
	fmt.Println("[*] Initializing MCP connection...")
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "mcp-tool-tester",
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	serverInfo, err := mcpClient.Initialize(ctx, initRequest)
	if err != nil {
		fmt.Printf("[-] Failed to initialize: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[+] Connected to %s v%s\n", serverInfo.ServerInfo.Name, serverInfo.ServerInfo.Version)
	fmt.Println()

	// Define test cases
	testCases := []TestCase{
		{Name: "List servers", Tool: "upstream_servers", Arguments: map[string]interface{}{"operation": "list"}},
		{Name: "Add test server", Tool: "upstream_servers", Arguments: map[string]interface{}{
			"operation": "add",
			"name":      "test-timeout-checker",
			"command":   "echo",
			"args_json": `["hello"]`,
			"protocol":  "stdio",
		}},
		{Name: "Remove test server", Tool: "upstream_servers", Arguments: map[string]interface{}{
			"operation": "remove",
			"name":      "test-timeout-checker",
		}},
		{Name: "List quarantined", Tool: "quarantine_security", Arguments: map[string]interface{}{"operation": "list_quarantined"}},
		{Name: "List groups", Tool: "groups", Arguments: map[string]interface{}{"operation": "list_groups"}},
		{Name: "Available groups", Tool: "list_available_groups", Arguments: map[string]interface{}{}},
		{Name: "Search tools", Tool: "retrieve_tools", Arguments: map[string]interface{}{"query": "calculator", "limit": float64(3)}},
		{Name: "List registries", Tool: "list_registries", Arguments: map[string]interface{}{}},
		{Name: "Startup status", Tool: "startup_script", Arguments: map[string]interface{}{"operation": "status"}},
	}

	// Print header
	fmt.Println("----------------------------------------------------------------------")
	fmt.Printf("%-35s %-25s %-12s %-8s\n", "Test Case", "Tool", "Status", "Time")
	fmt.Println("----------------------------------------------------------------------")

	results := map[string]int{
		"OK":      0,
		"TIMEOUT": 0,
		"ERROR":   0,
	}
	var failedTests []string

	for _, tc := range testCases {
		startTime := time.Now()

		// Create timeout context for this specific call
		callCtx, callCancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)

		// Call the tool
		request := mcp.CallToolRequest{}
		request.Params.Name = tc.Tool
		request.Params.Arguments = tc.Arguments

		result, err := mcpClient.CallTool(callCtx, request)
		callCancel()

		elapsed := time.Since(startTime)
		var status string

		if err != nil {
			if callCtx.Err() == context.DeadlineExceeded {
				status = "TIMEOUT"
				results["TIMEOUT"]++
			} else {
				status = "ERROR"
				results["ERROR"]++
			}
			failedTests = append(failedTests, fmt.Sprintf("[%s] %s: %v", status, tc.Name, err))
		} else if result.IsError {
			status = "ERROR"
			results["ERROR"]++
			errMsg := ""
			for _, content := range result.Content {
				if textContent, ok := content.(mcp.TextContent); ok {
					errMsg = textContent.Text
					break
				}
			}
			failedTests = append(failedTests, fmt.Sprintf("[%s] %s: %s", status, tc.Name, errMsg))
		} else {
			status = "OK"
			results["OK"]++
		}

		timeStr := fmt.Sprintf("%.2fs", elapsed.Seconds())
		if elapsed.Seconds() >= float64(timeoutSeconds) {
			timeStr = fmt.Sprintf(">%ds", timeoutSeconds)
		}

		fmt.Printf("%-35s %-25s %-12s %-8s\n", tc.Name, tc.Tool, status, timeStr)
	}

	fmt.Println("----------------------------------------------------------------------")
	fmt.Println()

	// Summary
	fmt.Println("======================================================================")
	fmt.Println("SUMMARY")
	fmt.Println("======================================================================")
	total := results["OK"] + results["TIMEOUT"] + results["ERROR"]
	fmt.Printf("  Total tests:     %d\n", total)
	fmt.Printf("  Success (OK):    %d\n", results["OK"])
	fmt.Printf("  Timeout:         %d\n", results["TIMEOUT"])
	fmt.Printf("  Errors:          %d\n", results["ERROR"])
	fmt.Println()

	if len(failedTests) > 0 {
		fmt.Println("FAILED TESTS (potential bugs):")
		fmt.Println("----------------------------------------------------------------------")
		for _, ft := range failedTests {
			fmt.Printf("  %s\n", ft)
		}
		fmt.Println()
	}

	fmt.Println("[*] Test completed")
}
