//go:build !nogui && !headless && !linux

package tray

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"go.uber.org/zap"
	"mcpproxy-go/internal/config"
)

// ServerConfigDialog represents a server configuration dialog
type ServerConfigDialog struct {
	logger     *zap.SugaredLogger
	server     *config.ServerConfig
	serverName string

	// HTTP server for dialog
	httpServer *http.Server
	listener   net.Listener
	dialogPort int

	// Dialog state
	onSave   func(*config.ServerConfig) error
	onCancel func()
	
	// Diagnostic agent
	diagnosticAgent *DiagnosticAgent
	
	// Server manager for tools fetching
	serverManager interface {
		GetServerTools(serverName string) ([]map[string]interface{}, error)
	}
}

// ConfigDialogData contains data passed to the HTML template
type ConfigDialogData struct {
	Server      *config.ServerConfig `json:"server"`
	ServerName  string               `json:"serverName"`
	DialogTitle string               `json:"dialogTitle"`
	Port        int                  `json:"port"`
}

// ConfigDialogResult contains the result from the dialog
type ConfigDialogResult struct {
	Action string               `json:"action"`
	Server *config.ServerConfig `json:"server"`
}

// HTML template for the configuration dialog
const configDialogTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.DialogTitle}}</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }

        .dialog-container {
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 40px rgba(0, 0, 0, 0.1);
            max-width: 600px;
            width: 100%;
            max-height: 90vh;
            overflow-y: auto;
        }

        .dialog-header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 20px 24px;
            border-radius: 12px 12px 0 0;
            text-align: center;
        }

        .dialog-header h1 {
            font-size: 1.5rem;
            font-weight: 600;
        }

        .dialog-body {
            padding: 24px;
        }

        .form-group {
            margin-bottom: 20px;
        }

        .form-group label {
            display: block;
            margin-bottom: 6px;
            font-weight: 500;
            color: #374151;
        }

        .form-group input,
        .form-group select,
        .form-group textarea {
            width: 100%;
            padding: 10px 12px;
            border: 2px solid #e5e7eb;
            border-radius: 6px;
            font-size: 14px;
            transition: border-color 0.2s, box-shadow 0.2s;
        }

        .form-group input:focus,
        .form-group select:focus,
        .form-group textarea:focus {
            outline: none;
            border-color: #667eea;
            box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
        }

        .form-row {
            display: flex;
            gap: 16px;
        }

        .form-row .form-group {
            flex: 1;
        }

        .checkbox-group {
            display: flex;
            align-items: center;
            gap: 8px;
            margin-bottom: 20px;
        }

        .checkbox-group input[type="checkbox"] {
            width: auto;
        }

        .section-title {
            font-size: 1.1rem;
            font-weight: 600;
            margin: 24px 0 16px 0;
            color: #374151;
            border-bottom: 2px solid #e5e7eb;
            padding-bottom: 8px;
        }

        .section-title:first-child {
            margin-top: 0;
        }

        .help-text {
            font-size: 12px;
            color: #6b7280;
            margin-top: 4px;
        }

        .dialog-footer {
            padding: 20px 24px;
            border-top: 1px solid #e5e7eb;
            background: #f9fafb;
            border-radius: 0 0 12px 12px;
            display: flex;
            gap: 12px;
            justify-content: flex-end;
        }

        .btn {
            padding: 10px 20px;
            border-radius: 6px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.2s;
            border: 2px solid transparent;
            font-size: 14px;
        }

        .btn-primary {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }

        .btn-primary:hover {
            transform: translateY(-1px);
            box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
        }

        .btn-secondary {
            background: #f3f4f6;
            color: #374151;
            border-color: #d1d5db;
        }

        .btn-secondary:hover {
            background: #e5e7eb;
        }

        .btn-info {
            background-color: #17a2b8;
            border-color: #17a2b8;
            color: white;
        }
        .btn-info:hover {
            background-color: #138496;
            border-color: #117a8b;
        }
        .section {
            background: #f8f9fa;
            border: 1px solid #dee2e6;
            border-radius: 5px;
            padding: 15px;
            margin: 10px 0;
        }
        .tools-list {
            max-height: 300px;
            overflow-y: auto;
        }
        .tool-item {
            background: white;
            border: 1px solid #dee2e6;
            border-radius: 3px;
            padding: 10px;
            margin: 5px 0;
        }
        .tool-name {
            font-weight: bold;
            color: #007bff;
        }
        .tool-description {
            color: #6c757d;
            margin-top: 5px;
        }
        .diagnostic-report {
            background: white;
            border: 1px solid #dee2e6;
            border-radius: 3px;
            padding: 15px;
            font-family: monospace;
            white-space: pre-wrap;
            max-height: 400px;
            overflow-y: auto;
        }
        .connection-status {
            padding: 10px;
            background: #f8f9fa;
            border-radius: 3px;
            font-weight: bold;
        }

        .list-item {
            display: flex;
            gap: 8px;
            margin-bottom: 8px;
            align-items: center;
        }

        .list-item input {
            flex: 1;
            margin-bottom: 0;
        }

        .env-item .env-key {
            flex: 0 0 40%;
        }

        .env-item .env-value {
            flex: 1;
        }

        .btn-remove {
            background: #ef4444;
            color: white;
            border: none;
            border-radius: 4px;
            width: 28px;
            height: 28px;
            cursor: pointer;
            font-size: 16px;
            line-height: 1;
            display: flex;
            align-items: center;
            justify-content: center;
        }

        .btn-remove:hover {
            background: #dc2626;
        }

        .btn-add {
            background: #10b981;
            color: white;
            border: none;
            border-radius: 4px;
            padding: 8px 12px;
            cursor: pointer;
            font-size: 12px;
            margin-top: 4px;
        }

        .btn-add:hover {
            background: #059669;
        }
    </style>
</head>
<body>
    <div class="dialog-container">
        <div class="dialog-header">
            <h1>{{.DialogTitle}}</h1>
        </div>

        <div class="dialog-body">
            <form id="configForm">
                <div class="section-title">Basic Configuration</div>

                <div class="form-group">
                    <label for="name">Server Name</label>
                    <input type="text" id="name" name="name" value="{{.Server.Name}}" required>
                    <div class="help-text">Unique identifier for this server</div>
                </div>

                <div class="form-group">
                    <label for="description">Description</label>
                    <input type="text" id="description" name="description" value="{{.Server.Description}}" placeholder="Optional description of what this server does">
                    <div class="help-text">Brief description of the server's purpose or functionality</div>
                </div>

                <div class="form-row">
                    <div class="form-group">
                        <label for="protocol">Protocol</label>
                        <select id="protocol" name="protocol" onchange="toggleProtocolFields()">
                            <option value="stdio" {{if eq .Server.Protocol "stdio"}}selected{{end}}>stdio</option>
                            <option value="http" {{if eq .Server.Protocol "http"}}selected{{end}}>http</option>
                            <option value="sse" {{if eq .Server.Protocol "sse"}}selected{{end}}>sse</option>
                            <option value="streamable-http" {{if eq .Server.Protocol "streamable-http"}}selected{{end}}>streamable-http</option>
                        </select>
                        <div class="help-text">Communication protocol with the server</div>
                    </div>

                    <div class="checkbox-group">
                        <input type="checkbox" id="enabled" name="enabled" {{if .Server.Enabled}}checked{{end}}>
                        <label for="enabled">Enabled</label>
                    </div>
                </div>

                <div id="stdio-fields" style="display: none;">
                    <div class="section-title">stdio Configuration</div>

                    <div class="form-group">
                        <label for="command">Command</label>
                        <input type="text" id="command" name="command" value="{{.Server.Command}}">
                        <div class="help-text">Executable command (e.g., npx, python, uvx)</div>
                    </div>

                    <div class="form-group">
                        <label>Arguments</label>
                        <div id="args-list">
                            {{range .Server.Args}}
                            <div class="list-item">
                                <input type="text" class="arg-input" value="{{.}}" placeholder="Argument">
                                <button type="button" class="btn-remove" onclick="removeListItem(this)">&times;</button>
                            </div>
                            {{end}}
                        </div>
                        <button type="button" class="btn-add" onclick="addArg()">+ Add Argument</button>
                        <div class="help-text">Command line arguments</div>
                    </div>

                    <div class="form-group">
                        <label for="working_dir">Working Directory</label>
                        <input type="text" id="working_dir" name="working_dir" value="{{.Server.WorkingDir}}">
                        <div class="help-text">Working directory for the command (optional)</div>
                    </div>

                    <div class="form-group">
                        <label>Environment Variables</label>
                        <div id="env-list">
                            {{range $key, $value := .Server.Env}}
                            <div class="list-item env-item">
                                <input type="text" class="env-key" value="{{$key}}" placeholder="Variable name">
                                <input type="text" class="env-value" value="{{$value}}" placeholder="Value">
                                <button type="button" class="btn-remove" onclick="removeListItem(this)">&times;</button>
                            </div>
                            {{end}}
                        </div>
                        <button type="button" class="btn-add" onclick="addEnv()">+ Add Environment Variable</button>
                        <div class="help-text">Environment variables for the command</div>
                    </div>
                </div>

                <div id="http-fields" style="display: none;">
                    <div class="section-title">HTTP Configuration</div>

                    <div class="form-group">
                        <label for="url">URL</label>
                        <input type="url" id="url" name="url" value="{{.Server.URL}}">
                        <div class="help-text">HTTP endpoint URL</div>
                    </div>
                </div>

                <div class="section-title">Additional Settings</div>

                <div class="form-group">
                    <label for="repository_url">Repository URL</label>
                    <input type="url" id="repository_url" name="repository_url" value="{{.Server.RepositoryURL}}">
                    <div class="help-text">GitHub or repository URL for this MCP server (optional)</div>
                </div>

                <div class="checkbox-group">
                    <input type="checkbox" id="quarantined" name="quarantined" {{if .Server.Quarantined}}checked{{end}}>
                    <label for="quarantined">Quarantined</label>
                    <div class="help-text">Security quarantine status</div>
                </div>
            </form>
        </div>

        <div class="dialog-footer">
            <button type="button" class="btn btn-secondary" onclick="cancel()">Cancel</button>
            <button type="button" class="btn btn-info" onclick="runDiagnostic()">🔍 Diagnostic</button>
            <button type="button" class="btn btn-primary" onclick="save()">Save</button>
        </div>

        <!-- Connection Status Section -->
        <div class="section" style="margin-top: 20px;">
            <h3>🔗 Connection Status</h3>
            <div id="connectionStatus" class="connection-status">
                <span id="statusIndicator">⚪</span>
                <span id="statusText">Checking connection...</span>
            </div>
        </div>

        <!-- Tools Section -->
        <div class="section" style="margin-top: 20px;">
            <h3>🛠️ Available Tools</h3>
            <div id="toolsList" class="tools-list"></div>
        </div>

        <!-- Diagnostic Section -->
        <div class="section" style="margin-top: 20px;">
            <h3>🔍 Diagnostic Report</h3>
            <div id="diagnosticReport" class="diagnostic-report"></div>
        </div>
    </div>

    <script>
        const configData = {{.}};

        document.addEventListener('DOMContentLoaded', function() {
            toggleProtocolFields();
        });

        function toggleProtocolFields() {
            const protocol = document.getElementById('protocol').value;
            const stdioFields = document.getElementById('stdio-fields');
            const httpFields = document.getElementById('http-fields');

            if (protocol === 'stdio') {
                stdioFields.style.display = 'block';
                httpFields.style.display = 'none';
            } else {
                stdioFields.style.display = 'none';
                httpFields.style.display = 'block';
            }
        }

        function addArg() {
            const argsList = document.getElementById('args-list');
            const div = document.createElement('div');
            div.className = 'list-item';
            div.innerHTML = ` + "`" + `
                <input type="text" class="arg-input" placeholder="Argument">
                <button type="button" class="btn-remove" onclick="removeListItem(this)">&times;</button>
            ` + "`" + `;
            argsList.appendChild(div);
        }

        function addEnv() {
            const envList = document.getElementById('env-list');
            const div = document.createElement('div');
            div.className = 'list-item env-item';
            div.innerHTML = ` + "`" + `
                <input type="text" class="env-key" placeholder="Variable name">
                <input type="text" class="env-value" placeholder="Value">
                <button type="button" class="btn-remove" onclick="removeListItem(this)">&times;</button>
            ` + "`" + `;
            envList.appendChild(div);
        }

        function removeListItem(button) {
            button.parentElement.remove();
        }

        function collectFormData() {
            const form = document.getElementById('configForm');
            const formData = new FormData(form);

            const server = {
                name: formData.get('name'),
                description: formData.get('description') || '',
                protocol: formData.get('protocol'),
                enabled: formData.get('enabled') === 'on',
                quarantined: formData.get('quarantined') === 'on',
                repository_url: formData.get('repository_url') || '',
                created: (configData && configData.Server && configData.Server.created) ? configData.Server.created : new Date().toISOString(),
                updated: new Date().toISOString()
            };

            if (server.protocol === 'stdio') {
                server.command = formData.get('command') || '';
                server.working_dir = formData.get('working_dir') || '';
                
                // Collect args
                server.args = [];
                const argInputs = document.querySelectorAll('.arg-input');
                argInputs.forEach(input => {
                    if (input.value.trim()) {
                        server.args.push(input.value.trim());
                    }
                });

                // Collect env
                server.env = {};
                const envItems = document.querySelectorAll('.env-item');
                envItems.forEach(item => {
                    const key = item.querySelector('.env-key').value.trim();
                    const value = item.querySelector('.env-value').value.trim();
                    if (key) {
                        server.env[key] = value;
                    }
                });
            } else {
                server.url = formData.get('url') || '';
                server.args = [];
                server.env = {};
            }

            return server;
        }

        function save() {
            try {
                const server = collectFormData();

                if (!server.name.trim()) {
                    alert('Server name is required');
                    return;
                }

                if (server.protocol === 'stdio' && !server.command.trim()) {
                    alert('Command is required for stdio protocol');
                    return;
                }

                if (server.protocol !== 'stdio' && !server.url.trim()) {
                    alert('URL is required for HTTP-based protocols');
                    return;
                }

                // Show immediate feedback
                const saveBtn = document.querySelector('.btn-primary');
                if (saveBtn) {
                    saveBtn.textContent = 'Saving...';
                    saveBtn.disabled = true;
                }

                // Close window immediately - don't wait for server response
                setTimeout(() => window.close(), 100);

                // Send save request in background (fire and forget)
                fetch('/save', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        action: 'save',
                        server: server
                    })
                }).catch(() => {
                    // Ignore errors since window is already closing
                });

            } catch (error) {
                alert('Error collecting form data: ' + error.message);
                window.close();
            }
        }

        function cancel() {
            fetch('/cancel', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({action: 'cancel'})
            })
            .then(() => {
                window.close();
            })
            .catch(() => {
                window.close();
            });
        }

        function runDiagnostic() {
            const diagnosticBtn = document.querySelector('.btn-info');
            const diagnosticReport = document.getElementById('diagnosticReport');
            
            // Show loading state
            diagnosticBtn.textContent = '🔄 Running...';
            diagnosticBtn.disabled = true;
            
            diagnosticReport.textContent = 'Running diagnostic analysis...';
            
            fetch('/diagnostic', { method: 'POST' })
            .then(response => response.json())
            .then(report => {
                // Format diagnostic report
                let reportText = '🔍 MCP Server Diagnostic Report for: ' + report.server_name + '\n';
                reportText += 'Generated: ' + new Date(report.timestamp).toLocaleString() + '\n\n';
                
                // Configuration Analysis
                reportText += '📋 Configuration Analysis:\n';
                if (report.config_analysis.valid) {
                    reportText += '  ✅ Configuration is valid\n';
                } else {
                    reportText += '  ❌ Configuration has issues:\n';
                    if (report.config_analysis.missing_fields) {
                        report.config_analysis.missing_fields.forEach(function(field) {
                            reportText += '    - Missing field: ' + field + '\n';
                        });
                    }
                }
                
                if (report.config_analysis.suggestions && report.config_analysis.suggestions.length > 0) {
                    reportText += '  💡 Suggestions:\n';
                    report.config_analysis.suggestions.forEach(function(suggestion) {
                        reportText += '    - ' + suggestion + '\n';
                    });
                }
                
                // Log Analysis
                reportText += '\n📊 Log Analysis:\n';
                reportText += '  - Error count: ' + report.log_analysis.error_count + '\n';
                reportText += '  - Connection attempts: ' + report.log_analysis.connection_attempts + '\n';
                
                if (report.log_analysis.last_error) {
                    reportText += '  - Last error: ' + report.log_analysis.last_error + '\n';
                }
                
                if (report.log_analysis.common_errors && report.log_analysis.common_errors.length > 0) {
                    reportText += '  - Common errors:\n';
                    report.log_analysis.common_errors.forEach(function(error) {
                        reportText += '    - ' + error + '\n';
                    });
                }
                
                // Recommendations
                if (report.recommendations && report.recommendations.length > 0) {
                    reportText += '\n💡 Recommendations:\n';
                    report.recommendations.forEach(function(rec, i) {
                        reportText += '  ' + (i + 1) + '. ' + rec + '\n';
                    });
                }
                
                diagnosticReport.textContent = reportText;
            })
            .catch(function(error) {
                diagnosticReport.textContent = 'Error running diagnostic: ' + error.message;
            })
            .finally(function() {
                diagnosticBtn.textContent = '🔍 Diagnostic';
                diagnosticBtn.disabled = false;
            });
            
            // Also load tools if server is connected
            loadTools();
        }

        function loadConnectionStatus() {
            const statusIndicator = document.getElementById('statusIndicator');
            const statusText = document.getElementById('statusText');
            
            // Check if server is connected by trying to load tools
            fetch(window.location.origin + '/tools')
            .then(function(response) { return response.json(); })
            .then(function(data) {
                if (data.tools && data.tools.length > 0) {
                    statusIndicator.textContent = '🟢';
                    statusText.textContent = 'Connected (' + data.tools.length + ' tools available)';
                } else if (data.error) {
                    statusIndicator.textContent = '🔴';
                    statusText.textContent = 'Disconnected - ' + data.error;
                } else {
                    statusIndicator.textContent = '🟡';
                    statusText.textContent = 'Connected (no tools available)';
                }
            })
            .catch(function(error) {
                statusIndicator.textContent = '🔴';
                statusText.textContent = 'Connection failed - ' + error.message;
            });
        }

        function loadTools() {
            const toolsList = document.getElementById('toolsList');
            
            fetch(window.location.origin + '/tools')
            .then(function(response) { return response.json(); })
            .then(function(data) {
                if (data.tools && data.tools.length > 0) {
                    let toolsHtml = '';
                    data.tools.forEach(function(tool) {
                        toolsHtml += '<div class="tool-item">';
                        toolsHtml += '<div class="tool-name">' + (tool.name || 'Unnamed Tool') + '</div>';
                        toolsHtml += '<div class="tool-description">' + (tool.description || 'No description available') + '</div>';
                        toolsHtml += '</div>';
                    });
                    
                    toolsList.innerHTML = toolsHtml;
                } else if (data.error) {
                    toolsList.innerHTML = '<p>Error loading tools: ' + data.error + '</p>';
                } else {
                    toolsList.innerHTML = '<p>No tools available</p>';
                }
            })
            .catch(function(error) {
                toolsList.innerHTML = '<p>Failed to load tools: ' + error.message + '</p>';
            });
        }

        // Load initial data on page load
        window.addEventListener('load', function() {
            loadConnectionStatus();
            loadTools();
        });
    </script>
</body>
</html>`

// NewServerConfigDialog creates a new server configuration dialog
func NewServerConfigDialog(logger *zap.SugaredLogger, server *config.ServerConfig, serverName string) *ServerConfigDialog {
	return &ServerConfigDialog{
		logger:     logger,
		server:     server,
		serverName: serverName,
	}
}

// Show displays the configuration dialog
func (d *ServerConfigDialog) Show(ctx context.Context, onSave func(*config.ServerConfig) error, onCancel func()) error {
	d.onSave = onSave
	d.onCancel = onCancel

	// Find available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to find available port: %w", err)
	}

	d.listener = listener
	d.dialogPort = listener.Addr().(*net.TCPAddr).Port

	// Create HTTP server for dialog
	mux := http.NewServeMux()
	mux.HandleFunc("/", d.handleDialog)
	mux.HandleFunc("/save", d.handleSave)
	mux.HandleFunc("/cancel", d.handleCancel)
	mux.HandleFunc("/diagnostic", d.handleDiagnostic)
	mux.HandleFunc("/tools", d.handleTools)

	d.httpServer = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start HTTP server
	go func() {
		if err := d.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			d.logger.Error("Dialog HTTP server error", zap.Error(err))
		}
	}()

	// Open dialog in browser
	dialogURL := fmt.Sprintf("http://127.0.0.1:%d", d.dialogPort)
	if err := d.openBrowser(dialogURL); err != nil {
		d.logger.Warn("Failed to open browser for dialog", zap.Error(err))
		return fmt.Errorf("failed to open dialog: %w", err)
	}

	d.logger.Info("Opened server configuration dialog",
		zap.String("server", d.serverName),
		zap.String("url", dialogURL))

	return nil
}

// Close closes the dialog
func (d *ServerConfigDialog) Close() error {
	if d.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := d.httpServer.Shutdown(ctx); err != nil {
			d.logger.Warn("Failed to shutdown dialog HTTP server", zap.Error(err))
		}
	}

	if d.listener != nil {
		d.listener.Close()
	}

	return nil
}

// handleDialog serves the main dialog HTML
func (d *ServerConfigDialog) handleDialog(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("dialog").Parse(configDialogTemplate)
	if err != nil {
		d.logger.Error("Failed to parse dialog template", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Ensure Args and Env are initialized
	if d.server.Args == nil {
		d.server.Args = []string{}
	}
	if d.server.Env == nil {
		d.server.Env = make(map[string]string)
	}

	data := ConfigDialogData{
		Server:      d.server,
		ServerName:  d.serverName,
		DialogTitle: fmt.Sprintf("Configure Server: %s", d.serverName),
		Port:        d.dialogPort,
	}

	d.logger.Debug("Serving dialog", 
		zap.String("server_name", d.serverName),
		zap.Int("args_count", len(d.server.Args)),
		zap.Int("env_count", len(d.server.Env)))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		d.logger.Error("Failed to execute dialog template", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// handleSave handles save requests from the dialog
func (d *ServerConfigDialog) handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var result ConfigDialogResult
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		d.logger.Error("Failed to decode save request", zap.Error(err))
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if result.Action == "save" && d.onSave != nil {
		if err := d.onSave(result.Server); err != nil {
			d.logger.Error("Failed to save server configuration", zap.Error(err))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		d.logger.Info("Server configuration saved successfully", zap.String("server", result.Server.Name))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})

	// Close dialog after successful save
	go func() {
		time.Sleep(500 * time.Millisecond)
		d.Close()
	}()
}

// handleCancel handles cancel requests from the dialog
func (d *ServerConfigDialog) handleCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if d.onCancel != nil {
		d.onCancel()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})

	// Close dialog
	go func() {
		time.Sleep(500 * time.Millisecond)
		d.Close()
	}()
}

// handleDiagnostic handles diagnostic requests from the dialog
func (d *ServerConfigDialog) handleDiagnostic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if d.diagnosticAgent == nil {
		http.Error(w, "Diagnostic agent not available", http.StatusServiceUnavailable)
		return
	}

	report, err := d.diagnosticAgent.DiagnoseServer(d.serverName)
	if err != nil {
		d.logger.Error("Diagnostic analysis failed", zap.Error(err))
		http.Error(w, "Diagnostic failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// handleTools handles tools list requests from the dialog
func (d *ServerConfigDialog) handleTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if d.serverManager == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tools": []interface{}{},
			"error": "Server manager not available",
		})
		return
	}

	tools, err := d.serverManager.GetServerTools(d.serverName)
	if err != nil {
		d.logger.Error("Failed to fetch tools", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tools": []interface{}{},
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tools": tools,
	})
}

// openBrowser opens the given URL in the default browser
func (d *ServerConfigDialog) openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}