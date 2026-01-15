package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

// MetricsData represents system metrics
type MetricsData struct {
	Timestamp       time.Time              `json:"timestamp"`
	Uptime          string                 `json:"uptime"`
	GoVersion       string                 `json:"go_version"`
	NumGoroutines   int                    `json:"num_goroutines"`
	NumCPU          int                    `json:"num_cpu"`
	MemoryStats     runtime.MemStats       `json:"memory_stats"`
	UpstreamServers map[string]interface{} `json:"upstream_servers"`
	ToolsIndexed    int                    `json:"tools_indexed"`
}

// handleMetricsWeb serves the metrics web interface
func (s *Server) handleMetricsWeb(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MCPProxy - Resource Metrics</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background: linear-gradient(135deg, #28a745 0%, #20c997 100%);
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
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 30px;
        }
        .header h1 {
            color: #28a745;
            margin: 0;
        }
        .back-button {
            background: #6c757d;
            color: white;
            padding: 10px 20px;
            border-radius: 6px;
            text-decoration: none;
            transition: background 0.2s;
        }
        .back-button:hover {
            background: #5a6268;
        }
        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .metric-card {
            background: #f8f9fa;
            border-radius: 8px;
            padding: 20px;
            border-left: 4px solid #28a745;
        }
        .metric-label {
            color: #666;
            font-size: 0.9em;
            margin-bottom: 8px;
        }
        .metric-value {
            color: #333;
            font-size: 1.8em;
            font-weight: bold;
        }
        .metric-unit {
            color: #999;
            font-size: 0.7em;
            font-weight: normal;
        }
        .status-badge {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 12px;
            font-size: 0.85em;
            font-weight: 500;
        }
        .status-running {
            background: #d4edda;
            color: #155724;
        }
        .chart-section {
            background: #f8f9fa;
            border-radius: 8px;
            padding: 20px;
            margin-top: 20px;
        }
        .chart-section h3 {
            margin-top: 0;
            color: #333;
        }
        #metricsData {
            background: #fff;
            border: 1px solid #dee2e6;
            border-radius: 6px;
            padding: 15px;
            overflow-x: auto;
            max-height: 600px;
            overflow-y: auto;
        }
        .metric-section {
            margin-bottom: 25px;
        }
        .metric-section h4 {
            color: #28a745;
            margin: 0 0 15px 0;
            padding-bottom: 8px;
            border-bottom: 2px solid #e9ecef;
            font-size: 1.1em;
        }
        .metric-row {
            display: grid;
            grid-template-columns: 200px 1fr;
            padding: 8px 0;
            border-bottom: 1px solid #f1f3f5;
        }
        .metric-row:last-child {
            border-bottom: none;
        }
        .metric-key {
            color: #666;
            font-weight: 500;
        }
        .metric-val {
            color: #333;
            font-family: 'Courier New', monospace;
        }
        .server-list {
            display: grid;
            gap: 10px;
            margin-top: 10px;
        }
        .server-item {
            background: #f8f9fa;
            padding: 10px;
            border-radius: 4px;
            border-left: 3px solid #28a745;
        }
        .server-name {
            font-weight: 600;
            color: #333;
            margin-bottom: 5px;
        }
        .server-details {
            font-size: 0.85em;
            color: #666;
        }
        .update-time {
            text-align: center;
            color: #666;
            font-size: 0.9em;
            margin-top: 20px;
        }
        .auto-refresh {
            text-align: center;
            margin-top: 15px;
        }
        .refresh-button {
            background: #28a745;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 6px;
            cursor: pointer;
            font-size: 1em;
            transition: background 0.2s;
        }
        .refresh-button:hover {
            background: #218838;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📈 Resource Metrics</h1>
            <a href="/" class="back-button">← Back to Dashboard</a>
        </div>

        <div class="metrics-grid">
            <div class="metric-card">
                <div class="metric-label">Status</div>
                <div class="metric-value">
                    <span class="status-badge status-running">Running</span>
                </div>
            </div>
            <div class="metric-card">
                <div class="metric-label">Goroutines</div>
                <div class="metric-value" id="goroutines">-</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">Memory Allocated</div>
                <div class="metric-value" id="memAlloc">-</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">Tools Indexed</div>
                <div class="metric-value" id="toolsIndexed">-</div>
            </div>
        </div>

        <div class="chart-section">
            <h3>Detailed Metrics</h3>
            <div id="metricsData">Loading...</div>
        </div>

        <div class="auto-refresh">
            <button class="refresh-button" onclick="refreshMetrics()">Refresh Now</button>
        </div>

        <div class="update-time" id="updateTime">Last updated: -</div>
    </div>

    <script>
        function formatBytes(bytes) {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }

        function formatDetailedMetrics(data) {
            const mem = data.memory_stats;
            let html = '';

            // System Information Section
            html += '<div class="metric-section">';
            html += '<h4>🖥️ System Information</h4>';
            html += '<div class="metric-row"><div class="metric-key">Go Version</div><div class="metric-val">' + data.go_version + '</div></div>';
            html += '<div class="metric-row"><div class="metric-key">CPU Cores</div><div class="metric-val">' + data.num_cpu + '</div></div>';
            html += '<div class="metric-row"><div class="metric-key">Goroutines</div><div class="metric-val">' + data.num_goroutines + '</div></div>';
            html += '<div class="metric-row"><div class="metric-key">Uptime</div><div class="metric-val">' + data.uptime + '</div></div>';
            html += '</div>';

            // Memory Statistics Section
            html += '<div class="metric-section">';
            html += '<h4>💾 Memory Statistics</h4>';
            html += '<div class="metric-row"><div class="metric-key">Allocated Memory</div><div class="metric-val">' + formatBytes(mem.Alloc) + '</div></div>';
            html += '<div class="metric-row"><div class="metric-key">Total Allocated</div><div class="metric-val">' + formatBytes(mem.TotalAlloc) + '</div></div>';
            html += '<div class="metric-row"><div class="metric-key">System Memory</div><div class="metric-val">' + formatBytes(mem.Sys) + '</div></div>';
            html += '<div class="metric-row"><div class="metric-key">Heap Allocated</div><div class="metric-val">' + formatBytes(mem.HeapAlloc) + '</div></div>';
            html += '<div class="metric-row"><div class="metric-key">Heap System</div><div class="metric-val">' + formatBytes(mem.HeapSys) + '</div></div>';
            html += '<div class="metric-row"><div class="metric-key">Heap In Use</div><div class="metric-val">' + formatBytes(mem.HeapInuse) + '</div></div>';
            html += '<div class="metric-row"><div class="metric-key">Stack In Use</div><div class="metric-val">' + formatBytes(mem.StackInuse) + '</div></div>';
            html += '</div>';

            // Garbage Collection Section
            html += '<div class="metric-section">';
            html += '<h4>🗑️ Garbage Collection</h4>';
            html += '<div class="metric-row"><div class="metric-key">GC Runs</div><div class="metric-val">' + mem.NumGC + '</div></div>';
            html += '<div class="metric-row"><div class="metric-key">Last GC Pause</div><div class="metric-val">' + (mem.PauseNs[(mem.NumGC+255)%256] / 1000000).toFixed(2) + ' ms</div></div>';
            html += '<div class="metric-row"><div class="metric-key">Next GC Target</div><div class="metric-val">' + formatBytes(mem.NextGC) + '</div></div>';
            html += '</div>';

            // Upstream Servers Section
            if (data.upstream_servers && data.upstream_servers.servers) {
                html += '<div class="metric-section">';
                html += '<h4>🔌 Upstream Servers</h4>';
                html += '<div class="metric-row"><div class="metric-key">Total Servers</div><div class="metric-val">' + (data.upstream_servers.total || 0) + '</div></div>';

                if (data.upstream_servers.servers.length > 0) {
                    html += '<div class="server-list">';
                    data.upstream_servers.servers.forEach(server => {
                        const statusClass = server.enabled ? 'status-running' : 'status-badge';
                        const statusText = server.enabled ? '✓ Enabled' : '✗ Disabled';
                        html += '<div class="server-item">';
                        html += '<div class="server-name">' + server.name + ' <span class="status-badge ' + statusClass + '">' + statusText + '</span></div>';
                        html += '<div class="server-details">Protocol: ' + (server.protocol || 'unknown') + '</div>';
                        if (server.url) {
                            html += '<div class="server-details">URL: ' + server.url + '</div>';
                        }
                        if (server.quarantined) {
                            html += '<div class="server-details" style="color: #dc3545;">⚠️ Quarantined</div>';
                        }
                        html += '</div>';
                    });
                    html += '</div>';
                }
                html += '</div>';
            }

            // Tools Section
            html += '<div class="metric-section">';
            html += '<h4>🔧 Tool Index</h4>';
            html += '<div class="metric-row"><div class="metric-key">Tools Indexed</div><div class="metric-val">' + (data.tools_indexed || 0) + '</div></div>';
            html += '</div>';

            return html;
        }

        function refreshMetrics() {
            fetch('/api/metrics/current')
                .then(response => response.json())
                .then(data => {
                    // Update summary cards
                    document.getElementById('goroutines').textContent = data.num_goroutines;
                    document.getElementById('memAlloc').innerHTML = formatBytes(data.memory_stats.Alloc) + '<span class="metric-unit"> / ' + formatBytes(data.memory_stats.Sys) + '</span>';
                    document.getElementById('toolsIndexed').textContent = data.tools_indexed || 0;

                    // Update detailed view with formatted HTML
                    document.getElementById('metricsData').innerHTML = formatDetailedMetrics(data);

                    // Update timestamp
                    const now = new Date();
                    document.getElementById('updateTime').textContent = 'Last updated: ' + now.toLocaleString();
                })
                .catch(error => {
                    console.error('Error fetching metrics:', error);
                    document.getElementById('metricsData').innerHTML = '<div style="color: #dc3545; padding: 20px;">Error loading metrics: ' + error.message + '</div>';
                });
        }

        // Initial load
        refreshMetrics();

        // Auto-refresh every 5 seconds
        setInterval(refreshMetrics, 5000);
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

// handleMetricsAPI returns current metrics as JSON
func (s *Server) handleMetricsAPI(w http.ResponseWriter, r *http.Request) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Get upstream server stats
	upstreamStats := make(map[string]interface{})
	if s.upstreamManager != nil {
		servers := s.upstreamManager.ListServers()
		upstreamStats["total"] = len(servers)
		upstreamStats["servers"] = servers
	}

	// Get tools indexed count
	// NOTE: Feature Backlog - Add IndexManager.GetToolCount() method
	toolsIndexed := 0

	metrics := MetricsData{
		Timestamp:       time.Now(),
		Uptime:          time.Since(time.Now().Add(-1 * time.Hour)).String(), // Placeholder
		GoVersion:       runtime.Version(),
		NumGoroutines:   runtime.NumGoroutine(),
		NumCPU:          runtime.NumCPU(),
		MemoryStats:     memStats,
		UpstreamServers: upstreamStats,
		ToolsIndexed:    toolsIndexed,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}
