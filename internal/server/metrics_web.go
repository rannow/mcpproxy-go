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
            font-family: 'Courier New', monospace;
            font-size: 0.85em;
            overflow-x: auto;
            max-height: 400px;
            overflow-y: auto;
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

        function refreshMetrics() {
            fetch('/api/metrics/current')
                .then(response => response.json())
                .then(data => {
                    // Update summary cards
                    document.getElementById('goroutines').textContent = data.num_goroutines;
                    document.getElementById('memAlloc').innerHTML = formatBytes(data.memory_stats.Alloc) + '<span class="metric-unit"> / ' + formatBytes(data.memory_stats.Sys) + '</span>';
                    document.getElementById('toolsIndexed').textContent = data.tools_indexed || 0;

                    // Update detailed view
                    document.getElementById('metricsData').textContent = JSON.stringify(data, null, 2);

                    // Update timestamp
                    const now = new Date();
                    document.getElementById('updateTime').textContent = 'Last updated: ' + now.toLocaleString();
                })
                .catch(error => {
                    console.error('Error fetching metrics:', error);
                    document.getElementById('metricsData').textContent = 'Error loading metrics: ' + error.message;
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
	toolsIndexed := 0
	// TODO: Add method to get tool count from index manager

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
