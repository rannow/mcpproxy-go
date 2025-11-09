package server

import (
	"bufio"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// handleFailedServers displays all failed servers with links to Diagnostic Agent
func (s *Server) handleFailedServers(w http.ResponseWriter, r *http.Request) {
	dataDir := s.config.DataDir
	if dataDir == "" {
		dataDir = filepath.Join(os.Getenv("HOME"), ".mcpproxy")
	}

	failureLogPath := filepath.Join(dataDir, "failed_servers.log")

	// Read failed servers from log file
	failedServers := []map[string]string{}

	file, err := os.Open(failureLogPath)
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			// Parse log line format: timestamp [LEVEL] message
			parts := strings.SplitN(line, "\t", 3)
			if len(parts) >= 3 {
				timestamp := parts[0]
				message := parts[2]

				// Extract server name from message
				serverName := ""
				errorMsg := message
				if strings.Contains(message, "Server") && strings.Contains(message, "failed") {
					// Try to extract server name
					words := strings.Fields(message)
					for i, word := range words {
						if word == "Server" && i+1 < len(words) {
							serverName = strings.Trim(words[i+1], "\"")
							break
						}
					}
				}

				if serverName != "" {
					failedServers = append(failedServers, map[string]string{
						"name":      serverName,
						"timestamp": timestamp,
						"error":     errorMsg,
					})
				}
			}
		}
	}

	// Build failed server cards HTML
	serverCardsHTML := ""
	if len(failedServers) == 0 {
		serverCardsHTML = `<div class="no-failures">
			<h3>✅ No Failed Servers</h3>
			<p>All servers are running smoothly!</p>
		</div>`
	} else {
		for _, server := range failedServers {
			chatURL := "/server/chat?server=" + url.QueryEscape(server["name"])
			serverCardsHTML += fmt.Sprintf(`
			<div class="server-card">
				<div class="server-header">
					<h3>❌ %s</h3>
					<span class="timestamp">%s</span>
				</div>
				<div class="error-message">
					<strong>Error:</strong> %s
				</div>
				<div class="actions">
					<a href="%s" class="btn-diagnostic">🤖 Open Diagnostic Agent</a>
					<a href="/servers" class="btn-servers">View All Servers</a>
				</div>
			</div>`,
				html.EscapeString(server["name"]),
				html.EscapeString(server["timestamp"]),
				html.EscapeString(server["error"]),
				html.EscapeString(chatURL),
			)
		}
	}

	htmlPage := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Failed Servers - MCPProxy</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 8px 32px rgba(0,0,0,0.1);
            padding: 40px;
        }
        .header {
            text-align: center;
            margin-bottom: 40px;
        }
        .header h1 {
            color: #333;
            font-size: 2.5em;
            margin-bottom: 10px;
        }
        .header p {
            color: #666;
            font-size: 1.2em;
            margin: 0;
        }
        .back-link {
            display: inline-block;
            color: #007bff;
            text-decoration: none;
            margin-bottom: 20px;
            font-size: 1.1em;
        }
        .back-link:hover {
            text-decoration: underline;
        }
        .server-card {
            background: #f8f9fa;
            border-left: 4px solid #dc3545;
            border-radius: 8px;
            padding: 24px;
            margin-bottom: 20px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        .server-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 16px;
        }
        .server-header h3 {
            color: #dc3545;
            margin: 0;
            font-size: 1.5em;
        }
        .timestamp {
            color: #666;
            font-size: 0.9em;
        }
        .error-message {
            background: #fff;
            padding: 16px;
            border-radius: 6px;
            margin-bottom: 16px;
            border-left: 3px solid #ffc107;
            font-family: 'Courier New', monospace;
            font-size: 0.9em;
            line-height: 1.5;
            color: #333;
        }
        .actions {
            display: flex;
            gap: 12px;
            margin-top: 16px;
        }
        .btn-diagnostic {
            display: inline-block;
            background: #6f42c1;
            color: white;
            padding: 12px 24px;
            border-radius: 6px;
            text-decoration: none;
            font-weight: 500;
            transition: background 0.2s;
        }
        .btn-diagnostic:hover {
            background: #5a32a3;
        }
        .btn-servers {
            display: inline-block;
            background: #007bff;
            color: white;
            padding: 12px 24px;
            border-radius: 6px;
            text-decoration: none;
            font-weight: 500;
            transition: background 0.2s;
        }
        .btn-servers:hover {
            background: #0056b3;
        }
        .no-failures {
            text-align: center;
            padding: 60px 20px;
            color: #28a745;
        }
        .no-failures h3 {
            font-size: 2em;
            margin-bottom: 12px;
        }
        .no-failures p {
            font-size: 1.2em;
            color: #666;
        }
    </style>
</head>
<body>
    <div class="container">
        <a href="/" class="back-link">← Back to Dashboard</a>
        <div class="header">
            <h1>❌ Failed Servers Report</h1>
            <p>Servers that have encountered connection or initialization failures</p>
        </div>
        ` + serverCardsHTML + `
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, htmlPage)
}
