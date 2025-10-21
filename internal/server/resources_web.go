package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SystemResourcesData represents comprehensive system resources
type SystemResourcesData struct {
	Timestamp        time.Time              `json:"timestamp"`
	Process          ProcessInfo            `json:"process"`
	System           SystemInfo             `json:"system"`
	Docker           DockerInfo             `json:"docker"`
	Goroutines       int                    `json:"goroutines"`
	Memory           runtime.MemStats       `json:"memory"`
	UpstreamServers  map[string]interface{} `json:"upstream_servers"`
	ConnectedServers int                    `json:"connected_servers"` // Number of connected upstream servers
	ProcessTree      []ProcessTreeNode      `json:"process_tree"`      // Process tree of mcpproxy and its children
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

type ProcessTreeNode struct {
	PID      int                `json:"pid"`
	PPID     int                `json:"ppid"`
	Command  string             `json:"command"`
	CPU      string             `json:"cpu"`
	Memory   string             `json:"memory"`
	Runtime  string             `json:"runtime"`
	Children []ProcessTreeNode  `json:"children"`
}

// ResourceHistory stores historical resource data in a ring buffer
type ResourceHistory struct {
	mu          sync.RWMutex
	data        []SystemResourcesData
	maxSize     int
	currentIdx  int
}

// Global resource history (max 24 data points)
var resourceHistory = &ResourceHistory{
	data:    make([]SystemResourcesData, 0, 24),
	maxSize: 24,
}

// Add adds a new resource data point to the history
func (rh *ResourceHistory) Add(data SystemResourcesData) {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	if len(rh.data) < rh.maxSize {
		// Still filling up the buffer
		rh.data = append(rh.data, data)
	} else {
		// Buffer is full, overwrite oldest entry (ring buffer behavior)
		rh.data[rh.currentIdx] = data
		rh.currentIdx = (rh.currentIdx + 1) % rh.maxSize
	}
}

// GetAll returns all historical data points in chronological order
func (rh *ResourceHistory) GetAll() []SystemResourcesData {
	rh.mu.RLock()
	defer rh.mu.RUnlock()

	if len(rh.data) == 0 {
		return []SystemResourcesData{}
	}

	// If buffer not full yet, return in order
	if len(rh.data) < rh.maxSize {
		result := make([]SystemResourcesData, len(rh.data))
		copy(result, rh.data)
		return result
	}

	// Buffer is full, reorder from oldest to newest
	result := make([]SystemResourcesData, rh.maxSize)
	for i := 0; i < rh.maxSize; i++ {
		idx := (rh.currentIdx + i) % rh.maxSize
		result[i] = rh.data[idx]
	}
	return result
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

	// Check if Docker is available with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "info")
	if err := cmd.Run(); err != nil {
		return info // Docker not available or timeout
	}

	// Get all mcpproxy-managed containers (by label) AND containers with "mcp" in name
	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// First: Get containers with mcpproxy.server label
	cmd = exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "label=mcpproxy.server", "--format", "{{.ID}}|{{.Names}}|{{.Status}}|{{.Label \"mcpproxy.server\"}}")
	output, err := cmd.Output()

	containerMap := make(map[string]bool) // Track unique containers
	var allLines []string

	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			if line != "" {
				// Extract container ID (first field) for deduplication
				parts := strings.Split(line, "|")
				if len(parts) > 0 {
					containerID := parts[0]
					if !containerMap[containerID] {
						containerMap[containerID] = true
						allLines = append(allLines, line)
					}
				}
			}
		}
	}

	// Second: Get containers with "mcp" in name (for legacy compatibility)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()
	cmd2 := exec.CommandContext(ctx2, "docker", "ps", "-a", "--filter", "name=mcp", "--format", "{{.ID}}|{{.Names}}|{{.Status}}|")
	output2, err2 := cmd2.Output()

	if err2 == nil {
		lines := strings.Split(strings.TrimSpace(string(output2)), "\n")
		for _, line := range lines {
			if line != "" {
				parts := strings.Split(line, "|")
				if len(parts) > 0 {
					containerID := parts[0]
					if !containerMap[containerID] {
						containerMap[containerID] = true
						allLines = append(allLines, line)
					}
				}
			}
		}
	}

	info.Total = len(allLines)

	if len(allLines) > 0 {
		for _, line := range allLines {
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

				// Add server name from label if available (4th field)
				if len(parts) >= 4 && parts[3] != "" {
					container.Name = parts[3] + " (" + parts[1] + ")"
				}

				// Get container stats with timeout (skip if it takes too long)
				statsCtx, statsCancel := context.WithTimeout(context.Background(), 2*time.Second)
				statsCmd := exec.CommandContext(statsCtx, "docker", "stats", "--no-stream", "--format", "{{.CPUPerc}}|{{.MemUsage}}", container.ID)
				statsOutput, err := statsCmd.Output()
				statsCancel()
				if err == nil {
					statsParts := strings.Split(strings.TrimSpace(string(statsOutput)), "|")
					if len(statsParts) >= 2 {
						container.CPU = statsParts[0]
						container.Memory = statsParts[1]
					}
				}

				// Get container mounts with timeout
				mountCtx, mountCancel := context.WithTimeout(context.Background(), 2*time.Second)
				mountCmd := exec.CommandContext(mountCtx, "docker", "inspect", "--format", "{{range .Mounts}}{{.Source}}:{{.Destination}}|{{end}}", container.ID)
				mountOutput, err := mountCmd.Output()
				mountCancel()
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

// getProcessTree builds a process tree starting from mcpproxy main process
func getProcessTree() []ProcessTreeNode {
	pid := os.Getpid()

	// Get all child processes using ps
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Use ps to get process information for mcpproxy and all its descendants
	// -o for custom format, -e for all processes
	cmd := exec.CommandContext(ctx, "ps", "-eo", "pid,ppid,%cpu,%mem,etime,command")
	output, err := cmd.Output()
	if err != nil {
		return []ProcessTreeNode{}
	}

	// Parse process list
	lines := strings.Split(string(output), "\n")
	processMap := make(map[int]*ProcessTreeNode)

	for i, line := range lines {
		if i == 0 || line == "" {
			continue // Skip header and empty lines
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		procPID, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}

		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		cpu := fields[2]
		mem := fields[3]
		runtime := fields[4]
		command := strings.Join(fields[5:], " ")

		// Truncate very long commands
		if len(command) > 150 {
			command = command[:147] + "..."
		}

		node := &ProcessTreeNode{
			PID:      procPID,
			PPID:     ppid,
			Command:  command,
			CPU:      cpu,
			Memory:   mem,
			Runtime:  runtime,
			Children: []ProcessTreeNode{},
		}

		processMap[procPID] = node
	}

	// Build tree structure - find mcpproxy and all its children
	var tree []ProcessTreeNode

	// Find mcpproxy main process
	if mainProcess, exists := processMap[pid]; exists {
		// Build children recursively
		buildChildren(mainProcess, processMap)
		tree = append(tree, *mainProcess)
	}

	return tree
}

// buildChildren recursively builds the process tree
func buildChildren(node *ProcessTreeNode, processMap map[int]*ProcessTreeNode) {
	for _, proc := range processMap {
		if proc.PPID == node.PID {
			buildChildren(proc, processMap)
			node.Children = append(node.Children, *proc)
		}
	}
}

// handleResourcesWeb serves the comprehensive resources web interface
func (s *Server) handleResourcesWeb(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MCPProxy - System Resources</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/chartjs-adapter-date-fns@3.0.0/dist/chartjs-adapter-date-fns.bundle.min.js"></script>
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
        .chart-container {
            background: white;
            border-radius: 12px;
            padding: 24px;
            margin-bottom: 30px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        .chart-wrapper {
            position: relative;
            height: 300px;
        }
        .process-node {
            margin: 8px 0;
            padding: 12px;
            background: white;
            border-left: 3px solid #667eea;
            border-radius: 6px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .process-node.child {
            margin-left: 30px;
            border-left-color: #764ba2;
        }
        .process-node.grandchild {
            margin-left: 60px;
            border-left-color: #10b981;
        }
        .process-info {
            display: flex;
            justify-content: space-between;
            align-items: center;
            flex-wrap: wrap;
            gap: 10px;
        }
        .process-command {
            font-weight: 600;
            color: #333;
            flex: 1;
            min-width: 200px;
        }
        .process-stats {
            display: flex;
            gap: 15px;
            font-size: 0.9em;
        }
        .process-stat {
            display: flex;
            align-items: center;
            gap: 5px;
            color: #666;
        }
        .process-stat-label {
            font-weight: 600;
            color: #333;
        }
        .collapse-btn {
            background: #667eea;
            color: white;
            border: none;
            padding: 4px 12px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 0.85em;
            margin-left: 10px;
        }
        .collapse-btn:hover {
            background: #5568d3;
        }
        .children-container {
            margin-top: 8px;
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

                <!-- Process Tree -->
                <div class="section">
                    <h2 class="section-title">🌳 Process Tree</h2>
                    <div class="table-container" id="process-tree-container">
                        <div id="process-tree-content" style="padding: 20px; font-family: 'Courier New', monospace; background: #f8f9fa;">
                            <!-- Process tree will be rendered here -->
                        </div>
                    </div>
                </div>

                <!-- Historical Charts -->
                <div class="section">
                    <h2 class="section-title">📈 Resource History (Last 24 Points)</h2>

                    <div class="chart-container">
                        <h3 style="margin-bottom: 15px; color: #333;">CPU & Memory Usage</h3>
                        <div class="chart-wrapper">
                            <canvas id="cpuMemoryChart"></canvas>
                        </div>
                    </div>

                    <div class="chart-container">
                        <h3 style="margin-bottom: 15px; color: #333;">Goroutines & Threads</h3>
                        <div class="chart-wrapper">
                            <canvas id="goroutinesChart"></canvas>
                        </div>
                    </div>

                    <div class="chart-container">
                        <h3 style="margin-bottom: 15px; color: #333;">File Descriptors</h3>
                        <div class="chart-wrapper">
                            <canvas id="fdsChart"></canvas>
                        </div>
                    </div>

                    <div class="chart-container">
                        <h3 style="margin-bottom: 15px; color: #333;">Connected Servers</h3>
                        <div class="chart-wrapper">
                            <canvas id="connectedServersChart"></canvas>
                        </div>
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
        // Chart instances
        let cpuMemoryChart = null;
        let goroutinesChart = null;
        let fdsChart = null;
        let connectedServersChart = null;

        // Initialize charts
        function initCharts() {
            const commonOptions = {
                responsive: true,
                maintainAspectRatio: false,
                interaction: {
                    mode: 'index',
                    intersect: false,
                },
                plugins: {
                    legend: {
                        position: 'top',
                    }
                },
                scales: {
                    x: {
                        type: 'time',
                        time: {
                            unit: 'minute',
                            displayFormats: {
                                minute: 'HH:mm'
                            }
                        },
                        title: {
                            display: true,
                            text: 'Time'
                        }
                    }
                }
            };

            // CPU & Memory Chart
            const cpuMemoryCtx = document.getElementById('cpuMemoryChart').getContext('2d');
            cpuMemoryChart = new Chart(cpuMemoryCtx, {
                type: 'line',
                data: {
                    datasets: [{
                        label: 'CPU %',
                        data: [],
                        borderColor: '#667eea',
                        backgroundColor: 'rgba(102, 126, 234, 0.1)',
                        yAxisID: 'y',
                        tension: 0.4
                    }, {
                        label: 'Memory MB',
                        data: [],
                        borderColor: '#764ba2',
                        backgroundColor: 'rgba(118, 75, 162, 0.1)',
                        yAxisID: 'y1',
                        tension: 0.4
                    }]
                },
                options: {
                    ...commonOptions,
                    scales: {
                        ...commonOptions.scales,
                        y: {
                            type: 'linear',
                            display: true,
                            position: 'left',
                            title: {
                                display: true,
                                text: 'CPU %'
                            }
                        },
                        y1: {
                            type: 'linear',
                            display: true,
                            position: 'right',
                            title: {
                                display: true,
                                text: 'Memory (MB)'
                            },
                            grid: {
                                drawOnChartArea: false,
                            }
                        }
                    }
                }
            });

            // Goroutines & Threads Chart
            const goroutinesCtx = document.getElementById('goroutinesChart').getContext('2d');
            goroutinesChart = new Chart(goroutinesCtx, {
                type: 'line',
                data: {
                    datasets: [{
                        label: 'Goroutines',
                        data: [],
                        borderColor: '#10b981',
                        backgroundColor: 'rgba(16, 185, 129, 0.1)',
                        tension: 0.4
                    }, {
                        label: 'Threads',
                        data: [],
                        borderColor: '#f59e0b',
                        backgroundColor: 'rgba(245, 158, 11, 0.1)',
                        tension: 0.4
                    }]
                },
                options: {
                    ...commonOptions,
                    scales: {
                        ...commonOptions.scales,
                        y: {
                            beginAtZero: true,
                            title: {
                                display: true,
                                text: 'Count'
                            }
                        }
                    }
                }
            });

            // File Descriptors Chart
            const fdsCtx = document.getElementById('fdsChart').getContext('2d');
            fdsChart = new Chart(fdsCtx, {
                type: 'line',
                data: {
                    datasets: [{
                        label: 'File Descriptors',
                        data: [],
                        borderColor: '#ef4444',
                        backgroundColor: 'rgba(239, 68, 68, 0.1)',
                        fill: true,
                        tension: 0.4
                    }]
                },
                options: {
                    ...commonOptions,
                    scales: {
                        ...commonOptions.scales,
                        y: {
                            beginAtZero: true,
                            title: {
                                display: true,
                                text: 'Count'
                            }
                        }
                    }
                }
            });

            // Connected Servers Chart
            const connectedServersCtx = document.getElementById('connectedServersChart').getContext('2d');
            connectedServersChart = new Chart(connectedServersCtx, {
                type: 'line',
                data: {
                    datasets: [{
                        label: 'Connected Servers',
                        data: [],
                        borderColor: '#8b5cf6',
                        backgroundColor: 'rgba(139, 92, 246, 0.1)',
                        fill: true,
                        tension: 0.4
                    }]
                },
                options: {
                    ...commonOptions,
                    scales: {
                        ...commonOptions.scales,
                        y: {
                            beginAtZero: true,
                            title: {
                                display: true,
                                text: 'Server Count'
                            }
                        }
                    }
                }
            });
        }

        // Update charts with historical data
        function updateCharts() {
            fetch('/api/resources/history')
                .then(response => response.json())
                .then(data => {
                    if (!data.history || data.history.length === 0) {
                        return;
                    }

                    const history = data.history;

                    // Prepare data for charts
                    const cpuData = [];
                    const memoryData = [];
                    const goroutinesData = [];
                    const threadsData = [];
                    const fdsData = [];
                    const connectedServersData = [];

                    history.forEach(point => {
                        const timestamp = new Date(point.timestamp);

                        cpuData.push({ x: timestamp, y: point.process.cpu_percent });
                        memoryData.push({ x: timestamp, y: (point.process.rss_bytes / 1024 / 1024).toFixed(2) });
                        goroutinesData.push({ x: timestamp, y: point.goroutines });
                        threadsData.push({ x: timestamp, y: point.process.threads });
                        fdsData.push({ x: timestamp, y: point.process.file_descriptors });
                        connectedServersData.push({ x: timestamp, y: point.connected_servers || 0 });
                    });

                    // Update CPU & Memory Chart
                    cpuMemoryChart.data.datasets[0].data = cpuData;
                    cpuMemoryChart.data.datasets[1].data = memoryData;
                    cpuMemoryChart.update('none'); // 'none' for no animation on update

                    // Update Goroutines Chart
                    goroutinesChart.data.datasets[0].data = goroutinesData;
                    goroutinesChart.data.datasets[1].data = threadsData;
                    goroutinesChart.update('none');

                    // Update FDs Chart
                    fdsChart.data.datasets[0].data = fdsData;
                    fdsChart.update('none');

                    // Update Connected Servers Chart
                    connectedServersChart.data.datasets[0].data = connectedServersData;
                    connectedServersChart.update('none');
                })
                .catch(error => {
                    console.error('Error loading chart data:', error);
                });
        }

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

        function renderProcessNode(node, depth) {
            const nodeDiv = document.createElement('div');
            nodeDiv.className = 'process-node';

            // Add depth-specific classes
            if (depth === 1) nodeDiv.classList.add('child');
            if (depth >= 2) nodeDiv.classList.add('grandchild');

            // Create process info container
            const infoDiv = document.createElement('div');
            infoDiv.className = 'process-info';

            // Command and collapse button
            const commandDiv = document.createElement('div');
            commandDiv.className = 'process-command';

            const icon = depth === 0 ? '🔷' : depth === 1 ? '└─' : '  └─';
            commandDiv.innerHTML = icon + ' PID ' + node.pid + ': ' + node.command;

            if (node.children && node.children.length > 0) {
                const collapseBtn = document.createElement('button');
                collapseBtn.className = 'collapse-btn';
                collapseBtn.textContent = '▼ ' + node.children.length;
                collapseBtn.onclick = function() {
                    const childrenContainer = nodeDiv.querySelector('.children-container');
                    if (childrenContainer.style.display === 'none') {
                        childrenContainer.style.display = 'block';
                        collapseBtn.textContent = '▼ ' + node.children.length;
                    } else {
                        childrenContainer.style.display = 'none';
                        collapseBtn.textContent = '▶ ' + node.children.length;
                    }
                };
                commandDiv.appendChild(collapseBtn);
            }

            // Stats
            const statsDiv = document.createElement('div');
            statsDiv.className = 'process-stats';
            statsDiv.innerHTML =
                '<div class="process-stat"><span class="process-stat-label">CPU:</span> ' + node.cpu + '%</div>' +
                '<div class="process-stat"><span class="process-stat-label">MEM:</span> ' + node.memory + '%</div>' +
                '<div class="process-stat"><span class="process-stat-label">Runtime:</span> ' + node.runtime + '</div>';

            infoDiv.appendChild(commandDiv);
            infoDiv.appendChild(statsDiv);
            nodeDiv.appendChild(infoDiv);

            // Render children
            if (node.children && node.children.length > 0) {
                const childrenContainer = document.createElement('div');
                childrenContainer.className = 'children-container';

                node.children.forEach(child => {
                    childrenContainer.appendChild(renderProcessNode(child, depth + 1));
                });

                nodeDiv.appendChild(childrenContainer);
            }

            return nodeDiv;
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

                    // Process Tree
                    const processTreeContent = document.getElementById('process-tree-content');
                    processTreeContent.innerHTML = '';

                    if (data.process_tree && data.process_tree.length > 0) {
                        data.process_tree.forEach(node => {
                            processTreeContent.appendChild(renderProcessNode(node, 0));
                        });
                    } else {
                        processTreeContent.innerHTML = '<p style="color: #666; text-align: center; padding: 20px;">No process tree data available</p>';
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

        // Initialize charts on page load
        document.addEventListener('DOMContentLoaded', function() {
            initCharts();
            updateCharts();
        });

        // Initial load
        refreshResources();

        // Auto-refresh every 5 seconds
        setInterval(refreshResources, 5000);

        // Auto-update charts every 5 seconds
        setInterval(updateCharts, 5000);
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
	connectedCount := 0
	if s.upstreamManager != nil {
		servers := s.upstreamManager.ListServers()
		upstreamStats["total"] = len(servers)
		upstreamStats["servers"] = servers

		// Count connected servers by checking client connection status
		clients := s.upstreamManager.GetAllClients()
		for _, client := range clients {
			if client.IsConnected() {
				connectedCount++
			}
		}
	}

	resources := SystemResourcesData{
		Timestamp:        time.Now(),
		Process:          getProcessInfo(),
		System:           getSystemInfo(),
		Docker:           getDockerInfo(),
		Goroutines:       runtime.NumGoroutine(),
		Memory:           memStats,
		UpstreamServers:  upstreamStats,
		ConnectedServers: connectedCount,
		ProcessTree:      getProcessTree(),
	}

	// Add to history
	resourceHistory.Add(resources)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resources)
}

// handleResourcesHistoryAPI returns historical resource data as JSON
func (s *Server) handleResourcesHistoryAPI(w http.ResponseWriter, r *http.Request) {
	history := resourceHistory.GetAll()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"history": history,
		"count":   len(history),
		"max_size": 24,
	})
}
