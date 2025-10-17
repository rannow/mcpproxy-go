package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// ServerStatusData represents comprehensive server connection status
type ServerStatusData struct {
	Name              string    `json:"name"`
	Status            string    `json:"status"` // Ready, Connecting, Error, Disconnected
	Connected         bool      `json:"connected"`
	Connecting        bool      `json:"connecting"`
	RetryCount        int       `json:"retry_count"`
	LastRetryTime     time.Time `json:"last_retry_time,omitempty"`
	LastError         string    `json:"last_error,omitempty"`
	TimeSinceLastTry  string    `json:"time_since_last_try"`
	TimeToConnection  string    `json:"time_to_connection"`
	Protocol          string    `json:"protocol"`
	URL               string    `json:"url"`
	Command           string    `json:"command"`
	ToolCount         int       `json:"tool_count"`
}

// handleServersWeb serves the servers overview page with connection statistics
func (s *Server) handleServersWeb(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MCPProxy - Server Overview</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            max-width: 1600px;
            margin: 0 auto;
            background: white;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px 40px;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .header h1 {
            font-size: 2em;
            display: flex;
            align-items: center;
            gap: 15px;
        }
        .back-btn {
            background: rgba(255,255,255,0.2);
            color: white;
            padding: 12px 24px;
            border-radius: 8px;
            text-decoration: none;
            transition: all 0.3s;
            backdrop-filter: blur(10px);
        }
        .back-btn:hover {
            background: rgba(255,255,255,0.3);
            transform: translateY(-2px);
        }
        .content {
            padding: 40px;
        }
        .summary {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .summary-card {
            background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
            border-radius: 12px;
            padding: 20px;
            text-align: center;
        }
        .summary-label {
            color: #666;
            font-size: 0.9em;
            margin-bottom: 8px;
            text-transform: uppercase;
            letter-spacing: 1px;
        }
        .summary-value {
            color: #333;
            font-size: 2.5em;
            font-weight: bold;
        }
        .table-container {
            background: white;
            border-radius: 12px;
            overflow: hidden;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        table {
            width: 100%;
            border-collapse: collapse;
        }
        thead {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }
        th {
            padding: 16px 12px;
            text-align: left;
            font-weight: 600;
            text-transform: uppercase;
            font-size: 0.85em;
            letter-spacing: 1px;
        }
        td {
            padding: 16px 12px;
            border-bottom: 1px solid #e9ecef;
        }
        tbody tr:hover {
            background: #f8f9fa;
        }
        .status-badge {
            display: inline-block;
            padding: 6px 12px;
            border-radius: 20px;
            font-size: 0.85em;
            font-weight: 600;
        }
        .status-ready { background: #d4edda; color: #155724; }
        .status-connecting { background: #fff3cd; color: #856404; }
        .status-error { background: #f8d7da; color: #721c24; }
        .status-disconnected { background: #e2e3e5; color: #383d41; }
        .error-message {
            color: #dc3545;
            font-size: 0.85em;
            font-style: italic;
            max-width: 300px;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
        }
        .protocol-badge {
            background: #e7f3ff;
            color: #0056b3;
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 0.8em;
            font-weight: 500;
        }
        .server-name {
            font-weight: 600;
            color: #333;
        }
        .update-time {
            text-align: center;
            color: #666;
            margin-top: 30px;
            padding-top: 20px;
            border-top: 1px solid #dee2e6;
        }
        .refresh-btn {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            padding: 12px 32px;
            border-radius: 8px;
            font-size: 1em;
            cursor: pointer;
            transition: all 0.3s;
            margin-top: 20px;
        }
        .refresh-btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 16px rgba(102, 126, 234, 0.4);
        }
        .loading {
            text-align: center;
            padding: 60px;
            color: #666;
        }
        .spinner {
            border: 4px solid #f3f3f3;
            border-top: 4px solid #667eea;
            border-radius: 50%;
            width: 50px;
            height: 50px;
            animation: spin 1s linear infinite;
            margin: 20px auto;
        }
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
        .no-servers {
            text-align: center;
            padding: 60px;
            color: #666;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔌 Server Overview</h1>
            <a href="/" class="back-btn">← Dashboard</a>
        </div>

        <div class="content">
            <div id="loading" class="loading">
                <div class="spinner"></div>
                <p>Loading servers...</p>
            </div>

            <div id="content" style="display: none;">
                <div class="summary">
                    <div class="summary-card">
                        <div class="summary-label">Total Servers</div>
                        <div class="summary-value" id="total-servers">0</div>
                    </div>
                    <div class="summary-card">
                        <div class="summary-label">Connected</div>
                        <div class="summary-value" id="connected-servers">0</div>
                    </div>
                    <div class="summary-card">
                        <div class="summary-label">Connecting</div>
                        <div class="summary-value" id="connecting-servers">0</div>
                    </div>
                    <div class="summary-card">
                        <div class="summary-label">Errors</div>
                        <div class="summary-value" id="error-servers">0</div>
                    </div>
                </div>

                <div class="table-container">
                    <table>
                        <thead>
                            <tr>
                                <th>Server</th>
                                <th>Status</th>
                                <th>Protocol</th>
                                <th>Retry Count</th>
                                <th>Last Attempt</th>
                                <th>Time to Connect</th>
                                <th>Tools</th>
                                <th>Error</th>
                            </tr>
                        </thead>
                        <tbody id="servers-table"></tbody>
                    </table>
                </div>

                <div style="text-align: center;">
                    <button class="refresh-btn" onclick="refreshServers()">🔄 Refresh Now</button>
                </div>

                <div class="update-time" id="update-time">Last updated: -</div>
            </div>

            <div id="no-servers" class="no-servers" style="display: none;">
                <h2>No Active Servers</h2>
                <p>All servers are either disabled or quarantined.</p>
            </div>
        </div>
    </div>

    <script>
        function getStatusClass(status) {
            const statusLower = status.toLowerCase();
            if (statusLower === 'ready') return 'status-ready';
            if (statusLower === 'connecting' || statusLower === 'authenticating') return 'status-connecting';
            if (statusLower === 'error') return 'status-error';
            return 'status-disconnected';
        }

        function formatTimeSince(timeStr) {
            if (!timeStr) return '-';
            const date = new Date(timeStr);
            const now = new Date();
            const diff = Math.floor((now - date) / 1000); // seconds

            if (diff < 60) return diff + 's ago';
            if (diff < 3600) return Math.floor(diff / 60) + 'm ago';
            if (diff < 86400) return Math.floor(diff / 3600) + 'h ago';
            return Math.floor(diff / 86400) + 'd ago';
        }

        function refreshServers() {
            fetch('/api/servers/status')
                .then(response => response.json())
                .then(data => {
                    if (!data.servers || data.servers.length === 0) {
                        document.getElementById('loading').style.display = 'none';
                        document.getElementById('content').style.display = 'none';
                        document.getElementById('no-servers').style.display = 'block';
                        return;
                    }

                    // Update summary
                    document.getElementById('total-servers').textContent = data.summary.total;
                    document.getElementById('connected-servers').textContent = data.summary.connected;
                    document.getElementById('connecting-servers').textContent = data.summary.connecting;
                    document.getElementById('error-servers').textContent = data.summary.errors;

                    // Update table
                    const tbody = document.getElementById('servers-table');
                    tbody.innerHTML = '';

                    data.servers.forEach(server => {
                        const row = document.createElement('tr');

                        const timeSince = formatTimeSince(server.last_retry_time);
                        const errorText = server.last_error || '-';
                        const toolCount = server.tool_count || 0;

                        row.innerHTML =
                            '<td><span class="server-name">' + server.name + '</span><br><small>' + (server.url || server.command || '-') + '</small></td>' +
                            '<td><span class="status-badge ' + getStatusClass(server.status) + '">' + server.status + '</span></td>' +
                            '<td><span class="protocol-badge">' + server.protocol + '</span></td>' +
                            '<td>' + server.retry_count + '</td>' +
                            '<td>' + timeSince + '</td>' +
                            '<td>' + server.time_to_connection + '</td>' +
                            '<td>' + toolCount + '</td>' +
                            '<td><div class="error-message" title="' + errorText + '">' + errorText + '</div></td>';

                        tbody.appendChild(row);
                    });

                    // Show content, hide loading
                    document.getElementById('loading').style.display = 'none';
                    document.getElementById('no-servers').style.display = 'none';
                    document.getElementById('content').style.display = 'block';

                    // Update timestamp
                    document.getElementById('update-time').textContent = 'Last updated: ' + new Date().toLocaleString();
                })
                .catch(error => {
                    console.error('Error:', error);
                    document.getElementById('loading').innerHTML = '<p style="color: red;">Error loading servers: ' + error.message + '</p>';
                });
        }

        // Initial load
        refreshServers();

        // Auto-refresh every 5 seconds
        setInterval(refreshServers, 5000);
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

// handleServersStatusAPI returns comprehensive server status as JSON
func (s *Server) handleServersStatusAPI(w http.ResponseWriter, r *http.Request) {
	// Get all servers from storage
	allServers, err := s.storageManager.ListUpstreamServers()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get servers: %v", err), http.StatusInternalServerError)
		return
	}

	var activeServers []ServerStatusData
	summary := struct {
		Total      int `json:"total"`
		Connected  int `json:"connected"`
		Connecting int `json:"connecting"`
		Errors     int `json:"errors"`
	}{}

	startTime := time.Now()

	for _, server := range allServers {
		// Skip disabled and quarantined servers
		if !server.Enabled || server.Quarantined {
			continue
		}

		serverData := ServerStatusData{
			Name:             server.Name,
			Protocol:         server.Protocol,
			URL:              server.URL,
			Command:          server.Command,
			Status:           "Disconnected",
			Connected:        false,
			Connecting:       false,
			RetryCount:       0,
			TimeSinceLastTry: "-",
			TimeToConnection: "-",
			ToolCount:        0,
		}

		// Get connection status from upstream manager
		if s.upstreamManager != nil {
			if client, exists := s.upstreamManager.GetClient(server.Name); exists {
				connectionStatus := client.GetConnectionStatus()

				// Extract status information
				if state, ok := connectionStatus["state"].(string); ok {
					serverData.Status = state
				}
				if connected, ok := connectionStatus["connected"].(bool); ok {
					serverData.Connected = connected
					if connected {
						summary.Connected++
					}
				}
				if connecting, ok := connectionStatus["connecting"].(bool); ok {
					serverData.Connecting = connecting
					if connecting {
						summary.Connecting++
					}
				}
				if retryCount, ok := connectionStatus["retry_count"].(int); ok {
					serverData.RetryCount = retryCount
				}
				if lastError, ok := connectionStatus["last_error"].(string); ok {
					serverData.LastError = lastError
					if serverData.LastError != "" && serverData.Status == "Error" {
						summary.Errors++
					}
				}
				if lastRetryTime, ok := connectionStatus["last_retry_time"].(time.Time); ok {
					serverData.LastRetryTime = lastRetryTime
					if !lastRetryTime.IsZero() {
						timeSince := time.Since(lastRetryTime)
						serverData.TimeSinceLastTry = formatDuration(timeSince)
					}
				}

				// Calculate time to connection if we have both timing values
				if firstAttemptTime, ok := connectionStatus["first_attempt_time"].(time.Time); ok {
					if connectedAt, ok := connectionStatus["connected_at"].(time.Time); ok {
						if !firstAttemptTime.IsZero() && !connectedAt.IsZero() {
							timeToConnect := connectedAt.Sub(firstAttemptTime)
							serverData.TimeToConnection = formatDuration(timeToConnect)
						}
					}
				}

				// Get tool count if connected
				if serverData.Connected {
					serverData.ToolCount = s.getServerToolCount(server.Name)
				}
			}
		}

		summary.Total++
		activeServers = append(activeServers, serverData)
	}

	s.logger.Debug("Server status API request completed",
		zap.Duration("duration", time.Since(startTime)),
		zap.Int("active_servers", len(activeServers)))

	response := map[string]interface{}{
		"servers": activeServers,
		"summary": summary,
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	return fmt.Sprintf("%.1fd", d.Hours()/24)
}
