package server

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"mcpproxy-go/internal/config"
)

// handleDocs serves the API documentation page showing all servers and tools
func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Get all servers from upstream manager
	servers := s.upstreamManager.ListServers()

	// Create a sorted list of server names for consistent display
	var serverNames []string
	for name := range servers {
		serverNames = append(serverNames, name)
	}
	sort.Strings(serverNames)

	// Build HTML content
	var htmlBuilder strings.Builder
	htmlBuilder.WriteString(s.getDocsHeader())

	// Add server sections
	for _, serverName := range serverNames {
		serverConfig := servers[serverName]
		client, exists := s.upstreamManager.GetClient(serverName)

		if !exists {
			continue
		}

		// Get server status
		isConnected := client.IsConnected()
		state := client.GetState()

		// Get tools for this server
		var tools []toolInfo
		if isConnected {
			serverTools, err := client.ListTools(ctx)
			if err != nil {
				s.logger.Warn("Failed to list tools for documentation",
					zap.String("server", serverName),
					zap.Error(err))
			} else {
				tools = s.parseToolsForDocs(serverTools)
			}
		}

		// Render server section
		htmlBuilder.WriteString(s.renderServerSection(serverConfig, state.String(), isConnected, tools))
	}

	htmlBuilder.WriteString(s.getDocsFooter())

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, htmlBuilder.String())
}

// toolInfo represents parsed tool information for documentation
type toolInfo struct {
	Name        string
	Description string
	Parameters  []paramInfo
	Required    []string
}

// paramInfo represents a parameter in tool documentation
type paramInfo struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

// parseToolsForDocs parses tool metadata into a documentation-friendly format
func (s *Server) parseToolsForDocs(tools []*config.ToolMetadata) []toolInfo {
	var result []toolInfo

	for _, tool := range tools {
		ti := toolInfo{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  []paramInfo{},
			Required:    []string{},
		}

		// Parse ParamsJSON to extract parameter information
		if tool.ParamsJSON != "" {
			var schema map[string]interface{}
			if err := json.Unmarshal([]byte(tool.ParamsJSON), &schema); err == nil {
				// Extract properties
				if props, ok := schema["properties"].(map[string]interface{}); ok {
					for paramName, paramValue := range props {
						if paramSchema, ok := paramValue.(map[string]interface{}); ok {
							param := paramInfo{
								Name: paramName,
								Type: "any",
							}

							// Extract type
							if paramType, ok := paramSchema["type"].(string); ok {
								param.Type = paramType
							}

							// Extract description
							if desc, ok := paramSchema["description"].(string); ok {
								param.Description = desc
							}

							ti.Parameters = append(ti.Parameters, param)
						}
					}
				}

				// Extract required fields
				if required, ok := schema["required"].([]interface{}); ok {
					for _, req := range required {
						if reqStr, ok := req.(string); ok {
							ti.Required = append(ti.Required, reqStr)
						}
					}
				}

				// Mark parameters as required
				for i := range ti.Parameters {
					for _, req := range ti.Required {
						if ti.Parameters[i].Name == req {
							ti.Parameters[i].Required = true
							break
						}
					}
				}

				// Sort parameters: required first, then alphabetically
				sort.Slice(ti.Parameters, func(i, j int) bool {
					if ti.Parameters[i].Required != ti.Parameters[j].Required {
						return ti.Parameters[i].Required
					}
					return ti.Parameters[i].Name < ti.Parameters[j].Name
				})
			}
		}

		result = append(result, ti)
	}

	// Sort tools alphabetically
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// renderServerSection renders a server section with its tools
func (s *Server) renderServerSection(serverConfig *config.ServerConfig, state string, isConnected bool, tools []toolInfo) string {
	var sb strings.Builder

	// Server section header
	statusClass := "disconnected"
	statusText := state
	if isConnected {
		statusClass = "connected"
		statusText = "Connected"
	}

	sb.WriteString(fmt.Sprintf(`
	<div class="server-section">
		<div class="server-header" onclick="toggleServer('%s')">
			<div class="server-title">
				<h2>%s</h2>
				<span class="status-badge %s">%s</span>
			</div>
			<div class="server-meta">
				<span class="tool-count">%d tools</span>
				<span class="toggle-icon" id="toggle-%s">▼</span>
			</div>
		</div>
		<div class="server-content" id="content-%s">
`,
		html.EscapeString(serverConfig.Name),
		html.EscapeString(serverConfig.Name),
		statusClass,
		statusText,
		len(tools),
		html.EscapeString(serverConfig.Name),
		html.EscapeString(serverConfig.Name),
	))

	// Server details
	sb.WriteString(`<div class="server-details">`)

	if serverConfig.URL != "" {
		sb.WriteString(fmt.Sprintf(`<p><strong>URL:</strong> <code>%s</code></p>`, html.EscapeString(serverConfig.URL)))
	}

	if serverConfig.Command != "" {
		sb.WriteString(fmt.Sprintf(`<p><strong>Command:</strong> <code>%s</code></p>`, html.EscapeString(serverConfig.Command)))
	}

	sb.WriteString(fmt.Sprintf(`<p><strong>Protocol:</strong> <code>%s</code></p>`, html.EscapeString(serverConfig.Protocol)))
	sb.WriteString(`</div>`)

	// Tools section
	if !isConnected {
		sb.WriteString(`<div class="no-tools"><p>Server is not connected. Connect the server to view available tools.</p></div>`)
	} else if len(tools) == 0 {
		sb.WriteString(`<div class="no-tools"><p>No tools available from this server.</p></div>`)
	} else {
		sb.WriteString(`<div class="tools-grid">`)
		for _, tool := range tools {
			sb.WriteString(s.renderToolCard(serverConfig.Name, tool))
		}
		sb.WriteString(`</div>`)
	}

	sb.WriteString(`</div></div>`)
	return sb.String()
}

// renderToolCard renders a tool card with parameters
func (s *Server) renderToolCard(serverName string, tool toolInfo) string {
	var sb strings.Builder

	fullToolName := fmt.Sprintf("%s:%s", serverName, tool.Name)

	sb.WriteString(fmt.Sprintf(`
		<div class="tool-card">
			<div class="tool-header">
				<h3>%s</h3>
				<code class="tool-name">%s</code>
			</div>
`,
		html.EscapeString(tool.Name),
		html.EscapeString(fullToolName),
	))

	// Tool description
	if tool.Description != "" {
		sb.WriteString(fmt.Sprintf(`<p class="tool-description">%s</p>`, html.EscapeString(tool.Description)))
	}

	// Parameters table
	if len(tool.Parameters) > 0 {
		sb.WriteString(`
			<div class="params-section">
				<h4>Parameters</h4>
				<table class="params-table">
					<thead>
						<tr>
							<th>Name</th>
							<th>Type</th>
							<th>Required</th>
							<th>Description</th>
						</tr>
					</thead>
					<tbody>
`)

		for _, param := range tool.Parameters {
			requiredBadge := ""
			if param.Required {
				requiredBadge = `<span class="required-badge">Required</span>`
			} else {
				requiredBadge = `<span class="optional-badge">Optional</span>`
			}

			description := param.Description
			if description == "" {
				description = "-"
			}

			sb.WriteString(fmt.Sprintf(`
						<tr>
							<td><code>%s</code></td>
							<td><code>%s</code></td>
							<td>%s</td>
							<td>%s</td>
						</tr>
`,
				html.EscapeString(param.Name),
				html.EscapeString(param.Type),
				requiredBadge,
				html.EscapeString(description),
			))
		}

		sb.WriteString(`
					</tbody>
				</table>
			</div>
`)
	} else {
		sb.WriteString(`<p class="no-params">No parameters required</p>`)
	}

	sb.WriteString(`</div>`)
	return sb.String()
}

// getDocsHeader returns the HTML header for the docs page
func (s *Server) getDocsHeader() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MCPProxy - API Documentation</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            max-width: 1400px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 8px 32px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .nav-header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 20px 40px;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .nav-header h1 {
            font-size: 2em;
            font-weight: 600;
        }
        .nav-link {
            background: rgba(255, 255, 255, 0.2);
            color: white;
            padding: 10px 20px;
            border-radius: 6px;
            text-decoration: none;
            transition: background 0.2s;
        }
        .nav-link:hover {
            background: rgba(255, 255, 255, 0.3);
        }
        .content {
            padding: 40px;
        }
        .intro {
            margin-bottom: 40px;
            padding: 20px;
            background: #f8f9fa;
            border-left: 4px solid #007bff;
            border-radius: 4px;
        }
        .intro h2 {
            color: #333;
            margin-bottom: 10px;
        }
        .intro p {
            color: #666;
            line-height: 1.6;
        }
        .server-section {
            margin-bottom: 30px;
            border: 1px solid #e0e0e0;
            border-radius: 8px;
            overflow: hidden;
        }
        .server-header {
            background: #f8f9fa;
            padding: 20px;
            cursor: pointer;
            display: flex;
            justify-content: space-between;
            align-items: center;
            transition: background 0.2s;
        }
        .server-header:hover {
            background: #e9ecef;
        }
        .server-title {
            display: flex;
            align-items: center;
            gap: 15px;
        }
        .server-title h2 {
            color: #333;
            font-size: 1.5em;
            font-weight: 600;
        }
        .server-meta {
            display: flex;
            align-items: center;
            gap: 15px;
        }
        .status-badge {
            padding: 4px 12px;
            border-radius: 12px;
            font-size: 0.85em;
            font-weight: 600;
        }
        .status-badge.connected {
            background: #d4edda;
            color: #155724;
        }
        .status-badge.disconnected {
            background: #f8d7da;
            color: #721c24;
        }
        .tool-count {
            color: #666;
            font-size: 0.9em;
        }
        .toggle-icon {
            font-size: 1.2em;
            transition: transform 0.3s;
        }
        .toggle-icon.rotated {
            transform: rotate(-180deg);
        }
        .server-content {
            padding: 20px;
            border-top: 1px solid #e0e0e0;
        }
        .server-content.collapsed {
            display: none;
        }
        .server-details {
            margin-bottom: 20px;
            padding: 15px;
            background: #f8f9fa;
            border-radius: 6px;
        }
        .server-details p {
            margin: 8px 0;
            color: #666;
        }
        .server-details code {
            background: white;
            padding: 2px 6px;
            border-radius: 3px;
            color: #e83e8c;
        }
        .no-tools {
            padding: 20px;
            text-align: center;
            color: #666;
            background: #f8f9fa;
            border-radius: 6px;
        }
        .tools-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
            gap: 20px;
        }
        .tool-card {
            background: white;
            border: 1px solid #e0e0e0;
            border-radius: 8px;
            padding: 20px;
            transition: box-shadow 0.2s, transform 0.2s;
        }
        .tool-card:hover {
            box-shadow: 0 4px 12px rgba(0,0,0,0.1);
            transform: translateY(-2px);
        }
        .tool-header {
            margin-bottom: 15px;
        }
        .tool-header h3 {
            color: #333;
            font-size: 1.2em;
            margin-bottom: 5px;
        }
        .tool-name {
            display: inline-block;
            background: #e7f3ff;
            color: #007bff;
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 0.85em;
            font-family: 'Courier New', monospace;
        }
        .tool-description {
            color: #666;
            line-height: 1.5;
            margin-bottom: 15px;
        }
        .params-section {
            margin-top: 15px;
        }
        .params-section h4 {
            color: #333;
            font-size: 1em;
            margin-bottom: 10px;
        }
        .params-table {
            width: 100%;
            border-collapse: collapse;
            font-size: 0.9em;
        }
        .params-table th {
            background: #f8f9fa;
            padding: 8px;
            text-align: left;
            color: #666;
            font-weight: 600;
            border-bottom: 2px solid #dee2e6;
        }
        .params-table td {
            padding: 8px;
            border-bottom: 1px solid #e0e0e0;
            color: #333;
        }
        .params-table code {
            background: #f8f9fa;
            padding: 2px 6px;
            border-radius: 3px;
            color: #e83e8c;
            font-size: 0.9em;
        }
        .required-badge {
            background: #ffc107;
            color: #856404;
            padding: 2px 8px;
            border-radius: 4px;
            font-size: 0.8em;
            font-weight: 600;
        }
        .optional-badge {
            background: #e7f3ff;
            color: #004085;
            padding: 2px 8px;
            border-radius: 4px;
            font-size: 0.8em;
            font-weight: 600;
        }
        .no-params {
            color: #999;
            font-style: italic;
        }
        .footer {
            text-align: center;
            padding: 20px;
            color: #666;
            font-size: 0.9em;
            border-top: 1px solid #e0e0e0;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="nav-header">
            <h1>📚 API Documentation</h1>
            <a href="/" class="nav-link">← Back to Dashboard</a>
        </div>
        <div class="content">
            <div class="intro">
                <h2>MCP Server Tools Reference</h2>
                <p>This page provides comprehensive documentation for all available MCP server tools. Each server section can be expanded to view its tools, parameters, and usage information.</p>
            </div>
`
}

// getDocsFooter returns the HTML footer for the docs page
func (s *Server) getDocsFooter() string {
	return `
        </div>
        <div class="footer">
            <p>MCPProxy API Documentation | Generated on ` + time.Now().Format("2006-01-02 15:04:05") + `</p>
        </div>
    </div>
    <script>
        function toggleServer(serverName) {
            const content = document.getElementById('content-' + serverName);
            const toggle = document.getElementById('toggle-' + serverName);

            if (content.classList.contains('collapsed')) {
                content.classList.remove('collapsed');
                toggle.classList.remove('rotated');
            } else {
                content.classList.add('collapsed');
                toggle.classList.add('rotated');
            }
        }
    </script>
</body>
</html>
`
}
