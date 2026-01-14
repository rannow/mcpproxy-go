package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
)

type ToolMetadataRecord struct {
	ServerID     string                 `json:"server_id"`
	ToolName     string                 `json:"tool_name"`
	PrefixedName string                 `json:"prefixed_name"`
	Description  string                 `json:"description"`
	InputSchema  map[string]interface{} `json:"input_schema,omitempty"`
	Created      time.Time              `json:"created"`
	Updated      time.Time              `json:"updated"`
}

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	dbPath := filepath.Join(homeDir, ".mcpproxy", "config.db")

	db, err := bbolt.Open(dbPath, 0644, &bbolt.Options{
		ReadOnly: true,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	toolsByServer := make(map[string][]ToolMetadataRecord)
	totalTools := 0

	err = db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("tool_metadata"))
		if bucket == nil {
			return fmt.Errorf("tool_metadata bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			var record ToolMetadataRecord
			if err := json.Unmarshal(v, &record); err != nil {
				log.Printf("Failed to unmarshal tool: %v\n", err)
				return nil
			}

			toolsByServer[record.ServerID] = append(toolsByServer[record.ServerID], record)
			totalTools++
			return nil
		})
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("                    TOOLS IN DATABASE\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("Total Tools: %d\n", totalTools)
	fmt.Printf("Total Servers: %d\n", len(toolsByServer))
	fmt.Printf("═══════════════════════════════════════════════════════════════\n\n")

	for serverID, tools := range toolsByServer {
		fmt.Printf("📦 Server: %s (%d tools)\n", serverID, len(tools))
		fmt.Printf("───────────────────────────────────────────────────────────────\n")

		for i, tool := range tools {
			fmt.Printf("  %d. %s\n", i+1, tool.PrefixedName)
			if tool.Description != "" {
				desc := tool.Description
				if len(desc) > 80 {
					desc = desc[:77] + "..."
				}
				fmt.Printf("     └─ %s\n", desc)
			}
		}
		fmt.Printf("\n")
	}
}
