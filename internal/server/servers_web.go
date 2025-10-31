package server

import (
	"context"
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
            cursor: pointer;
            user-select: none;
            position: relative;
        }
        th:hover {
            background: rgba(255, 255, 255, 0.1);
        }
        th.sortable::after {
            content: ' ⇅';
            opacity: 0.3;
            font-size: 0.9em;
        }
        th.sort-asc::after {
            content: ' ▲';
            opacity: 1;
        }
        th.sort-desc::after {
            content: ' ▼';
            opacity: 1;
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
                        <div class="summary-label">Total in Config</div>
                        <div class="summary-value" id="config-total-servers">0</div>
                    </div>
                    <div class="summary-card">
                        <div class="summary-label">Displayed</div>
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
                                <th class="sortable" data-sort="name">Server</th>
                                <th class="sortable" data-sort="status">Status</th>
                                <th class="sortable" data-sort="protocol">Protocol</th>
                                <th class="sortable" data-sort="retry_count">Retry Count</th>
                                <th class="sortable" data-sort="last_retry">Last Attempt</th>
                                <th class="sortable" data-sort="time_to_connect">Time to Connect</th>
                                <th class="sortable" data-sort="tool_count">Tools</th>
                                <th>Error</th>
                                <th>Actions</th>
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
        let currentServers = [];
        let currentSort = loadSortPreference();

        // Load sort preference from localStorage
        function loadSortPreference() {
            try {
                const saved = localStorage.getItem('serversSortPreference');
                if (saved) {
                    return JSON.parse(saved);
                }
            } catch (e) {
                console.error('Error loading sort preference:', e);
            }
            return { column: null, direction: 'asc' };
        }

        // Save sort preference to localStorage
        function saveSortPreference() {
            try {
                localStorage.setItem('serversSortPreference', JSON.stringify(currentSort));
            } catch (e) {
                console.error('Error saving sort preference:', e);
            }
        }

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

            // Check if date is invalid or too old (before year 2000)
            if (isNaN(date.getTime()) || date.getFullYear() < 2000) {
                return '-';
            }

            const now = new Date();
            const diff = Math.floor((now - date) / 1000); // seconds

            // If negative (future) or too large (> 1 year), show '-'
            if (diff < 0 || diff > 31536000) {
                return '-';
            }

            if (diff < 60) return diff + 's ago';
            if (diff < 3600) return Math.floor(diff / 60) + 'm ago';
            if (diff < 86400) return Math.floor(diff / 3600) + 'h ago';
            return Math.floor(diff / 86400) + 'd ago';
        }

        function sortServers(column) {
            // Toggle direction if clicking same column
            if (currentSort.column === column) {
                currentSort.direction = currentSort.direction === 'asc' ? 'desc' : 'asc';
            } else {
                currentSort.column = column;
                currentSort.direction = 'asc';
            }

            // Save preference
            saveSortPreference();

            // Sort the servers array
            currentServers.sort((a, b) => {
                let valA, valB;

                switch(column) {
                    case 'name':
                        valA = (a.name || '').toLowerCase();
                        valB = (b.name || '').toLowerCase();
                        break;
                    case 'status':
                        valA = (a.status || '').toLowerCase();
                        valB = (b.status || '').toLowerCase();
                        break;
                    case 'protocol':
                        valA = (a.protocol || '').toLowerCase();
                        valB = (b.protocol || '').toLowerCase();
                        break;
                    case 'retry_count':
                        valA = a.retry_count || 0;
                        valB = b.retry_count || 0;
                        break;
                    case 'last_retry':
                        valA = a.last_retry_time ? new Date(a.last_retry_time).getTime() : 0;
                        valB = b.last_retry_time ? new Date(b.last_retry_time).getTime() : 0;
                        break;
                    case 'time_to_connect':
                        valA = a.time_to_connection || '';
                        valB = b.time_to_connection || '';
                        break;
                    case 'tool_count':
                        valA = a.tool_count || 0;
                        valB = b.tool_count || 0;
                        break;
                    default:
                        return 0;
                }

                if (valA < valB) return currentSort.direction === 'asc' ? -1 : 1;
                if (valA > valB) return currentSort.direction === 'asc' ? 1 : -1;
                return 0;
            });

            // Update UI
            updateTableWithCurrentData();
            updateSortIndicators();
        }

        function updateSortIndicators() {
            // Remove all sort classes
            document.querySelectorAll('th.sortable').forEach(th => {
                th.classList.remove('sort-asc', 'sort-desc');
            });

            // Add current sort class
            if (currentSort.column) {
                const th = document.querySelector('th[data-sort="' + currentSort.column + '"]');
                if (th) {
                    th.classList.add('sort-' + currentSort.direction);
                }
            }
        }

        function updateTableWithCurrentData() {
            const tbody = document.getElementById('servers-table');
            tbody.innerHTML = '';

            currentServers.forEach(server => {
                const row = document.createElement('tr');
                const timeSince = formatTimeSince(server.last_retry_time);
                const errorText = server.last_error || '-';
                const toolCount = server.tool_count || 0;

                row.innerHTML =
                    '<td><span class="server-name">' + server.name + '</span><br><small>' + (server.url || server.command || '-') + '</small></td>' +
                    '<td><span class="status-badge ' + getStatusClass(server.status) + '">' + server.status + '</span></td>' +
                    '<td>' + (server.protocol || '-') + '</td>' +
                    '<td>' + (server.retry_count || 0) + '</td>' +
                    '<td>' + timeSince + '</td>' +
                    '<td>' + (server.time_to_connection || '-') + '</td>' +
                    '<td><strong>' + toolCount + '</strong></td>' +
                    '<td style="max-width: 300px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="' + errorText + '">' + errorText + '</td>' +
                    '<td><a href="/server/chat?server=' + encodeURIComponent(server.name) + '" style="display: inline-block; padding: 6px 12px; background: #667eea; color: white; text-decoration: none; border-radius: 4px; font-size: 0.85em;">🤖 Chat</a></td>';

                tbody.appendChild(row);
            });
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
                    document.getElementById('config-total-servers').textContent = data.summary.config_total || 0;
                    document.getElementById('total-servers').textContent = data.summary.total;
                    document.getElementById('connected-servers').textContent = data.summary.connected;
                    document.getElementById('connecting-servers').textContent = data.summary.connecting;
                    document.getElementById('error-servers').textContent = data.summary.errors;

                    // Store servers and update table
                    currentServers = data.servers;

                    // Apply current sort if any
                    if (currentSort.column) {
                        // Apply sort without triggering save (already loaded from localStorage)
                        currentServers.sort((a, b) => {
                            let valA, valB;

                            switch(currentSort.column) {
                                case 'name':
                                    valA = (a.name || '').toLowerCase();
                                    valB = (b.name || '').toLowerCase();
                                    break;
                                case 'status':
                                    valA = (a.status || '').toLowerCase();
                                    valB = (b.status || '').toLowerCase();
                                    break;
                                case 'protocol':
                                    valA = (a.protocol || '').toLowerCase();
                                    valB = (b.protocol || '').toLowerCase();
                                    break;
                                case 'retry_count':
                                    valA = a.retry_count || 0;
                                    valB = b.retry_count || 0;
                                    break;
                                case 'last_retry':
                                    valA = a.last_retry_time ? new Date(a.last_retry_time).getTime() : 0;
                                    valB = b.last_retry_time ? new Date(b.last_retry_time).getTime() : 0;
                                    break;
                                case 'time_to_connect':
                                    valA = a.time_to_connection || '';
                                    valB = b.time_to_connection || '';
                                    break;
                                case 'tool_count':
                                    valA = a.tool_count || 0;
                                    valB = b.tool_count || 0;
                                    break;
                                default:
                                    return 0;
                            }

                            if (valA < valB) return currentSort.direction === 'asc' ? -1 : 1;
                            if (valA > valB) return currentSort.direction === 'asc' ? 1 : -1;
                            return 0;
                        });
                        updateTableWithCurrentData();
                        updateSortIndicators();
                    } else {
                        updateTableWithCurrentData();
                    }

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

        // Add click handlers to sortable headers (add once after initial load)
        document.querySelectorAll('th.sortable').forEach(th => {
            th.addEventListener('click', () => {
                const sortColumn = th.getAttribute('data-sort');
                sortServers(sortColumn);
            });
        });

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
	// Add timeout to prevent hanging
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Get all servers from storage
	allServers, err := s.storageManager.ListUpstreamServers()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get servers: %v", err), http.StatusInternalServerError)
		return
	}

	// Get total from config (source of truth)
	configTotal := len(s.config.Servers)

	var activeServers []ServerStatusData
	summary := struct {
		Total       int  `json:"total"`
		ConfigTotal int  `json:"config_total"` // Total servers in configuration
		Connected   int  `json:"connected"`
		Connecting  int  `json:"connecting"`
		Errors      int  `json:"errors"`
		Timeout     bool `json:"timeout"` // Indicates if response was incomplete due to timeout
	}{
		ConfigTotal: configTotal, // Set total from config (source of truth)
	}

	startTime := time.Now()

	for _, server := range allServers {
		// Check for timeout
		select {
		case <-ctx.Done():
			s.logger.Warn("Server status API timeout reached",
				zap.Int("processed", len(activeServers)),
				zap.Int("total", len(allServers)))
			summary.Timeout = true
			goto respond
		default:
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

		// Show special status for disabled and quarantined servers
		if !server.Enabled {
			serverData.Status = "Disabled"
		} else if server.Quarantined {
			serverData.Status = "Quarantined"
		}

		// Get connection status from upstream manager
		// Only update status for enabled, non-quarantined servers
		if s.upstreamManager != nil && server.Enabled && !server.Quarantined {
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
					// Only show LastRetryTime for servers that are not Ready
					// Ready servers should show "-" for Last Attempt
					if serverData.Status != "Ready" {
						serverData.LastRetryTime = lastRetryTime
						if !lastRetryTime.IsZero() {
							timeSince := time.Since(lastRetryTime)
							serverData.TimeSinceLastTry = formatDuration(timeSince)
						}
					}
					// For Ready servers, LastRetryTime stays zero (will display as "-")
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

respond:
	s.logger.Debug("Server status API request completed",
		zap.Duration("duration", time.Since(startTime)),
		zap.Int("active_servers", len(activeServers)),
		zap.Bool("timeout", summary.Timeout))

	response := map[string]interface{}{
		"servers":   activeServers,
		"summary":   summary,
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
