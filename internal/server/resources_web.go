package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// SystemResourcesData represents comprehensive system resources
type SystemResourcesData struct {
	Timestamp       time.Time              `json:"timestamp"`
	Process         ProcessInfo            `json:"process"`
	System          SystemInfo             `json:"system"`
	Docker          DockerInfo             `json:"docker"`
	Goroutines      int                    `json:"goroutines"`
	Memory          runtime.MemStats       `json:"memory"`
	UpstreamServers map[string]interface{} `json:"upstream_servers"`
}

type ProcessInfo struct {
	PID        int     `json:"pid"`
	CPU        float64 `json:"cpu_percent"`
	Memory     uint64  `json:"memory_bytes"`
	RSS        uint64  `json:"rss_bytes"`
	VSZ        uint64  `json:"vsz_bytes"`
	Threads    int     `json:"threads"`
	FDs        int     `json:"file_descriptors"`
	Uptime     string  `json:"uptime"`
}

type SystemInfo struct {
	CPUCores    int     `json:"cpu_cores"`
	LoadAverage string  `json:"load_average"`
	TotalMemory uint64  `json:"total_memory"`
	UsedMemory  uint64  `json:"used_memory"`
}

type DockerInfo struct {
	Running    int               `json:"running"`
	Total      int               `json:"total"`
	Containers []DockerContainer `json:"containers"`
}

type DockerContainer struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	CPU     string   `json:"cpu"`
	Memory  string   `json:"memory"`
	Mounts  []string `json:"mounts"`
}

// getProcessInfo collects information about the mcpproxy process
func getProcessInfo() ProcessInfo {
	pid := os.Getpid()
	info := ProcessInfo{
		PID: pid,
	}

	// Get process stats using ps command
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "%cpu,%mem,rss,vsz,etime")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) > 1 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 5 {
				info.CPU, _ = strconv.ParseFloat(fields[0], 64)
				info.RSS, _ = strconv.ParseUint(fields[2], 10, 64)
				info.RSS *= 1024 // Convert KB to bytes
				info.VSZ, _ = strconv.ParseUint(fields[3], 10, 64)
				info.VSZ *= 1024 // Convert KB to bytes
				info.Uptime = fields[4]
			}
		}
	}

	// Get thread count
	cmd = exec.Command("ps", "-M", "-p", strconv.Itoa(pid))
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		info.Threads = len(lines) - 2 // Subtract header and empty line
		if info.Threads < 0 {
			info.Threads = 0
		}
	}

	// Get file descriptor count
	cmd = exec.Command("lsof", "-p", strconv.Itoa(pid))
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		info.FDs = len(lines) - 1 // Subtract header
		if info.FDs < 0 {
			info.FDs = 0
		}
	}

	return info
}

// getSystemInfo collects system-wide information
func getSystemInfo() SystemInfo {
	info := SystemInfo{
		CPUCores: runtime.NumCPU(),
	}

	// Get load average (macOS/Linux)
	cmd := exec.Command("uptime")
	output, err := cmd.Output()
	if err == nil {
		parts := strings.Split(string(output), "load average:")
		if len(parts) > 1 {
			info.LoadAverage = strings.TrimSpace(parts[1])
		}
	}

	// Get memory info (macOS)
	cmd = exec.Command("vm_stat")
	output, err = cmd.Output()
	if err == nil {
		// Parse vm_stat output for memory info
		lines := strings.Split(string(output), "\n")
		var pageSize uint64 = 4096 // Default page size
		var pagesActive, pagesInactive, pagesWired, pagesFree uint64

		for _, line := range lines {
			if strings.Contains(line, "page size") {
				fields := strings.Fields(line)
				if len(fields) >= 8 {
					pageSize, _ = strconv.ParseUint(fields[7], 10, 64)
				}
			} else if strings.Contains(line, "Pages active:") {
				fields := strings.Fields(line)
				if len(fields) >= 3 {
					val := strings.TrimSuffix(fields[2], ".")
					pagesActive, _ = strconv.ParseUint(val, 10, 64)
				}
			} else if strings.Contains(line, "Pages inactive:") {
				fields := strings.Fields(line)
				if len(fields) >= 3 {
					val := strings.TrimSuffix(fields[2], ".")
					pagesInactive, _ = strconv.ParseUint(val, 10, 64)
				}
			} else if strings.Contains(line, "Pages wired down:") {
				fields := strings.Fields(line)
				if len(fields) >= 4 {
					val := strings.TrimSuffix(fields[3], ".")
					pagesWired, _ = strconv.ParseUint(val, 10, 64)
				}
			} else if strings.Contains(line, "Pages free:") {
				fields := strings.Fields(line)
				if len(fields) >= 3 {
					val := strings.TrimSuffix(fields[2], ".")
					pagesFree, _ = strconv.ParseUint(val, 10, 64)
				}
			}
		}

		info.UsedMemory = (pagesActive + pagesInactive + pagesWired) * pageSize
		info.TotalMemory = info.UsedMemory + (pagesFree * pageSize)
	}

	return info
}

// getDockerInfo collects Docker container information
func getDockerInfo() DockerInfo {
	info := DockerInfo{
		Containers: []DockerContainer{},
	}

	// Check if Docker is available
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		return info // Docker not available
	}

	// Get MCP containers
	cmd = exec.Command("docker", "ps", "-a", "--filter", "name=mcp", "--format", "{{.ID}}|{{.Names}}|{{.Status}}")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		info.Total = len(lines)

		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.Split(line, "|")
			if len(parts) >= 3 {
				container := DockerContainer{
					ID:     parts[0],
					Name:   parts[1],
					Status: parts[2],
				}

				// Get container stats
				statsCmd := exec.Command("docker", "stats", "--no-stream", "--format", "{{.CPUPerc}}|{{.MemUsage}}", container.ID)
				statsOutput, err := statsCmd.Output()
				if err == nil {
					statsParts := strings.Split(strings.TrimSpace(string(statsOutput)), "|")
					if len(statsParts) >= 2 {
						container.CPU = statsParts[0]
						container.Memory = statsParts[1]
					}
				}

				// Get container mounts
				mountCmd := exec.Command("docker", "inspect", "--format", "{{range .Mounts}}{{.Source}}:{{.Destination}}|{{end}}", container.ID)
				mountOutput, err := mountCmd.Output()
				if err == nil {
					mounts := strings.Split(strings.TrimSpace(string(mountOutput)), "|")
					for _, mount := range mounts {
						if mount != "" {
							container.Mounts = append(container.Mounts, mount)
						}
					}
				}

				if strings.Contains(strings.ToLower(container.Status), "up") {
					info.Running++
				}

				info.Containers = append(info.Containers, container)
			}
		}
	}

	return info
}

// handleResourcesWeb serves the comprehensive resources web interface
func (s *Server) handleResourcesWeb(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MCPProxy - System Resources</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
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
        .section {
            margin-bottom: 40px;
        }
        .section-title {
            font-size: 1.5em;
            color: #333;
            margin-bottom: 20px;
            display: flex;
            align-items: center;
            gap: 10px;
            padding-bottom: 10px;
            border-bottom: 3px solid #667eea;
        }
        .grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .card {
            background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
            border-radius: 12px;
            padding: 24px;
            transition: all 0.3s;
            border: 2px solid transparent;
        }
        .card:hover {
            transform: translateY(-4px);
            box-shadow: 0 8px 24px rgba(0,0,0,0.15);
            border-color: #667eea;
        }
        .card-label {
            color: #666;
            font-size: 0.9em;
            margin-bottom: 8px;
            text-transform: uppercase;
            letter-spacing: 1px;
        }
        .card-value {
            color: #333;
            font-size: 2em;
            font-weight: bold;
        }
        .card-unit {
            color: #999;
            font-size: 0.5em;
            font-weight: normal;
        }
        .badge {
            display: inline-block;
            padding: 6px 12px;
            border-radius: 20px;
            font-size: 0.8em;
            font-weight: 600;
        }
        .badge-success { background: #d4edda; color: #155724; }
        .badge-warning { background: #fff3cd; color: #856404; }
        .badge-danger { background: #f8d7da; color: #721c24; }
        .badge-info { background: #d1ecf1; color: #0c5460; }
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
            padding: 16px;
            text-align: left;
            font-weight: 600;
            text-transform: uppercase;
            font-size: 0.85em;
            letter-spacing: 1px;
        }
        td {
            padding: 16px;
            border-bottom: 1px solid #e9ecef;
        }
        tbody tr:hover {
            background: #f8f9fa;
        }
        .mount-list {
            list-style: none;
            margin: 0;
            padding: 0;
        }
        .mount-item {
            background: #f8f9fa;
            padding: 6px 12px;
            margin: 4px 0;
            border-radius: 6px;
            font-size: 0.85em;
            font-family: 'Courier New', monospace;
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
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📊 System Resources</h1>
            <a href="/" class="back-btn">← Dashboard</a>
        </div>

        <div class="content">
            <div id="loading" class="loading">
                <div class="spinner"></div>
                <p>Loading resources...</p>
            </div>

            <div id="resources" style="display: none;">
                <!-- mcpproxy Process -->
                <div class="section">
                    <h2 class="section-title">💻 mcpproxy Process</h2>
                    <div class="grid">
                        <div class="card">
                            <div class="card-label">CPU Usage</div>
                            <div class="card-value" id="proc-cpu">-<span class="card-unit">%</span></div>
                        </div>
                        <div class="card">
                            <div class="card-label">Memory (RSS)</div>
                            <div class="card-value" id="proc-rss">-</div>
                        </div>
                        <div class="card">
                            <div class="card-label">Threads</div>
                            <div class="card-value" id="proc-threads">-</div>
                        </div>
                        <div class="card">
                            <div class="card-label">File Descriptors</div>
                            <div class="card-value" id="proc-fds">-</div>
                        </div>
                        <div class="card">
                            <div class="card-label">Goroutines</div>
                            <div class="card-value" id="goroutines">-</div>
                        </div>
                        <div class="card">
                            <div class="card-label">Uptime</div>
                            <div class="card-value" id="proc-uptime" style="font-size: 1.2em;">-</div>
                        </div>
                    </div>
                </div>

                <!-- System Overview -->
                <div class="section">
                    <h2 class="section-title">🖥️ System Overview</h2>
                    <div class="grid">
                        <div class="card">
                            <div class="card-label">CPU Cores</div>
                            <div class="card-value" id="sys-cores">-</div>
                        </div>
                        <div class="card">
                            <div class="card-label">Load Average</div>
                            <div class="card-value" id="sys-load" style="font-size: 1.2em;">-</div>
                        </div>
                        <div class="card">
                            <div class="card-label">Total Memory</div>
                            <div class="card-value" id="sys-mem-total">-</div>
                        </div>
                        <div class="card">
                            <div class="card-label">Used Memory</div>
                            <div class="card-value" id="sys-mem-used">-</div>
                        </div>
                    </div>
                </div>

                <!-- Docker Containers -->
                <div class="section">
                    <h2 class="section-title">🐳 Docker Containers</h2>
                    <div class="grid">
                        <div class="card">
                            <div class="card-label">Running</div>
                            <div class="card-value" id="docker-running">-</div>
                        </div>
                        <div class="card">
                            <div class="card-label">Total</div>
                            <div class="card-value" id="docker-total">-</div>
                        </div>
                    </div>
                    <div class="table-container" id="docker-table-container" style="display: none;">
                        <table>
                            <thead>
                                <tr>
                                    <th>Container</th>
                                    <th>Status</th>
                                    <th>CPU</th>
                                    <th>Memory</th>
                                    <th>Mounts</th>
                                </tr>
                            </thead>
                            <tbody id="docker-containers"></tbody>
                        </table>
                    </div>
                </div>

                <div style="text-align: center;">
                    <button class="refresh-btn" onclick="refreshResources()">🔄 Refresh Now</button>
                </div>

                <div class="update-time" id="update-time">Last updated: -</div>
            </div>
        </div>
    </div>

    <script>
        function formatBytes(bytes) {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }

        function getBadgeClass(status) {
            if (status.toLowerCase().includes('up')) return 'badge-success';
            if (status.toLowerCase().includes('exited')) return 'badge-danger';
            return 'badge-warning';
        }

        function refreshResources() {
            fetch('/api/resources/current')
                .then(response => response.json())
                .then(data => {
                    // Process info
                    document.getElementById('proc-cpu').innerHTML = data.process.cpu_percent.toFixed(1) + '<span class="card-unit">%</span>';
                    document.getElementById('proc-rss').textContent = formatBytes(data.process.rss_bytes);
                    document.getElementById('proc-threads').textContent = data.process.threads;
                    document.getElementById('proc-fds').textContent = data.process.file_descriptors;
                    document.getElementById('goroutines').textContent = data.goroutines;
                    document.getElementById('proc-uptime').textContent = data.process.uptime;

                    // System info
                    document.getElementById('sys-cores').textContent = data.system.cpu_cores;
                    document.getElementById('sys-load').textContent = data.system.load_average || '-';
                    document.getElementById('sys-mem-total').textContent = formatBytes(data.system.total_memory);
                    document.getElementById('sys-mem-used').textContent = formatBytes(data.system.used_memory);

                    // Docker info
                    document.getElementById('docker-running').textContent = data.docker.running;
                    document.getElementById('docker-total').textContent = data.docker.total;

                    // Docker containers table
                    const containersBody = document.getElementById('docker-containers');
                    containersBody.innerHTML = '';

                    if (data.docker.containers && data.docker.containers.length > 0) {
                        document.getElementById('docker-table-container').style.display = 'block';
                        data.docker.containers.forEach(container => {
                            const row = document.createElement('tr');

                            // Mounts list
                            let mountsHtml = '<ul class="mount-list">';
                            if (container.mounts && container.mounts.length > 0) {
                                container.mounts.forEach(mount => {
                                    mountsHtml += '<li class="mount-item">' + mount + '</li>';
                                });
                            } else {
                                mountsHtml += '<li class="mount-item">No mounts</li>';
                            }
                            mountsHtml += '</ul>';

                            row.innerHTML =
                                '<td><strong>' + container.name + '</strong><br><small>' + container.id.substring(0, 12) + '</small></td>' +
                                '<td><span class="badge ' + getBadgeClass(container.status) + '">' + container.status + '</span></td>' +
                                '<td>' + (container.cpu || '-') + '</td>' +
                                '<td>' + (container.memory || '-') + '</td>' +
                                '<td>' + mountsHtml + '</td>';
                            containersBody.appendChild(row);
                        });
                    } else {
                        document.getElementById('docker-table-container').style.display = 'none';
                    }

                    // Show content, hide loading
                    document.getElementById('loading').style.display = 'none';
                    document.getElementById('resources').style.display = 'block';

                    // Update timestamp
                    document.getElementById('update-time').textContent = 'Last updated: ' + new Date().toLocaleString();
                })
                .catch(error => {
                    console.error('Error:', error);
                    document.getElementById('loading').innerHTML = '<p style="color: red;">Error loading resources: ' + error.message + '</p>';
                });
        }

        // Initial load
        refreshResources();

        // Auto-refresh every 5 seconds
        setInterval(refreshResources, 5000);
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

// handleResourcesAPI returns comprehensive system resources as JSON
func (s *Server) handleResourcesAPI(w http.ResponseWriter, r *http.Request) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Get upstream server stats
	upstreamStats := make(map[string]interface{})
	if s.upstreamManager != nil {
		servers := s.upstreamManager.ListServers()
		upstreamStats["total"] = len(servers)
		upstreamStats["servers"] = servers
	}

	resources := SystemResourcesData{
		Timestamp:       time.Now(),
		Process:         getProcessInfo(),
		System:          getSystemInfo(),
		Docker:          getDockerInfo(),
		Goroutines:      runtime.NumGoroutine(),
		Memory:          memStats,
		UpstreamServers: upstreamStats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resources)
}
