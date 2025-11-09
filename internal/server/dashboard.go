package server

import (
	"fmt"
	"net/http"
)

// handleDashboard serves the main dashboard page with links to all features
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Only handle root path
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MCPProxy - Dashboard</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
        }
        .container {
            max-width: 1000px;
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
        .dashboard-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 24px;
            margin-top: 40px;
        }
        .card {
            background: #f8f9fa;
            border-radius: 12px;
            padding: 30px;
            transition: transform 0.2s, box-shadow 0.2s;
            border-left: 4px solid #007bff;
        }
        .card:hover {
            transform: translateY(-4px);
            box-shadow: 0 8px 24px rgba(0,0,0,0.12);
        }
        .card h3 {
            color: #007bff;
            margin-top: 0;
            margin-bottom: 12px;
            font-size: 1.5em;
        }
        .card p {
            color: #666;
            margin-bottom: 20px;
            line-height: 1.5;
        }
        .card-button {
            display: inline-block;
            background: #007bff;
            color: white;
            padding: 10px 20px;
            border-radius: 6px;
            text-decoration: none;
            transition: background 0.2s;
        }
        .card-button:hover {
            background: #0056b3;
        }
        .metrics-card {
            border-left-color: #28a745;
        }
        .metrics-card h3 {
            color: #28a745;
        }
        .metrics-card .card-button {
            background: #28a745;
        }
        .metrics-card .card-button:hover {
            background: #1e7e34;
        }
        .footer {
            text-align: center;
            margin-top: 40px;
            color: #666;
            font-size: 0.9em;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔌 MCPProxy</h1>
            <p>Smart Model Context Protocol Proxy</p>
        </div>

        <div class="dashboard-grid">
            <div class="card">
                <h3>📊 Server Groups</h3>
                <p>Organize and manage your MCP servers into logical groups</p>
                <a href="/groups" class="card-button">Manage Groups</a>
            </div>

            <div class="card">
                <h3>🔧 Server Overview</h3>
                <p>Monitor server connections, status, and performance metrics</p>
                <a href="/servers" class="card-button">View Servers</a>
            </div>

            <div class="card" style="border-left-color: #6f42c1;">
                <h3 style="color: #6f42c1;">🤖 AI Diagnostic Agent</h3>
                <p>Chat with AI to diagnose and troubleshoot server issues</p>
                <a href="/servers" class="card-button" style="background: #6f42c1;">Access Chat</a>
            </div>

            <div class="card" style="border-left-color: #dc3545;">
                <h3 style="color: #dc3545;">❌ Failed Servers</h3>
                <p>View servers with connection or initialization failures</p>
                <a href="/failed-servers" class="card-button" style="background: #dc3545;">View Failures</a>
            </div>

            <div class="card">
                <h3>🔗 Server Assignments</h3>
                <p>Assign servers to groups for better organization</p>
                <a href="/assignments" class="card-button">Manage Assignments</a>
            </div>

            <div class="card metrics-card">
                <h3>📈 Resource Metrics</h3>
                <p>Monitor system performance and resource usage in real-time</p>
                <a href="/metrics" class="card-button">View Metrics</a>
            </div>

            <div class="card metrics-card">
                <h3>💻 Resource Monitor</h3>
                <p>Advanced resource monitoring and system diagnostics</p>
                <a href="/resources" class="card-button">Open Monitor</a>
            </div>
        </div>

        <div class="footer">
            <p>MCPProxy %s | Running on %s</p>
        </div>
    </div>
</body>
</html>`

	// Get version and listen address
	version := "v1.0.0"
	listenAddr := s.config.Listen

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, html, version, listenAddr)
}
