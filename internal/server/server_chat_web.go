//go:build !nogui && !headless && !linux

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// chatSession stores conversation history for a chat session
type chatSession struct {
	ServerName string
	Messages   []chatMessage
	CreatedAt  time.Time
}

// chatMessage represents a single message in the conversation
type chatMessage struct {
	Role    string `json:"role"`    // "system", "user", or "assistant"
	Content string `json:"content"`
}

// sessionManager manages chat sessions
type sessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*chatSession
}

// Global session manager
var sessions = &sessionManager{
	sessions: make(map[string]*chatSession),
}

// getOrCreateSession retrieves or creates a chat session
func (sm *sessionManager) getOrCreateSession(sessionID, serverName string) *chatSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		return session
	}

	session := &chatSession{
		ServerName: serverName,
		Messages:   make([]chatMessage, 0),
		CreatedAt:  time.Now(),
	}
	sm.sessions[sessionID] = session
	return session
}

// addMessage adds a message to the session
func (sm *sessionManager) addMessage(sessionID string, role, content string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		session.Messages = append(session.Messages, chatMessage{
			Role:    role,
			Content: content,
		})
	}
}

// getMessages retrieves all messages from a session
func (sm *sessionManager) getMessages(sessionID string) []chatMessage {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if session, exists := sm.sessions[sessionID]; exists {
		return session.Messages
	}
	return nil
}

// OpenAI API structures
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []chatMessage   `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

type openAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int         `json:"index"`
		Message chatMessage `json:"message"`
		Finish  string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// handleServerChat serves the server diagnostic chat page
func (s *Server) handleServerChat(w http.ResponseWriter, r *http.Request) {
	serverName := r.URL.Query().Get("server")
	if serverName == "" {
		http.Error(w, "Server name required", http.StatusBadRequest)
		return
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + serverName + ` - Diagnostic Chat</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #f5f7fa;
            height: 100vh;
            display: flex;
            flex-direction: column;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 20px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .header h1 {
            font-size: 1.5em;
            margin-bottom: 5px;
        }
        .header p {
            opacity: 0.9;
            font-size: 0.9em;
        }
        .back-link {
            color: white;
            text-decoration: none;
            display: inline-block;
            margin-bottom: 10px;
            opacity: 0.9;
        }
        .back-link:hover {
            opacity: 1;
        }
        .container {
            flex: 1;
            display: flex;
            max-width: 1400px;
            margin: 0 auto;
            width: 100%;
            gap: 20px;
            padding: 20px;
            overflow: hidden;
        }
        .sidebar {
            width: 350px;
            background: white;
            border-radius: 12px;
            padding: 20px;
            overflow-y: auto;
            box-shadow: 0 2px 10px rgba(0,0,0,0.05);
        }
        .sidebar h3 {
            color: #333;
            margin-bottom: 15px;
            font-size: 1.1em;
        }
        .info-section {
            margin-bottom: 20px;
        }
        .info-section h3 {
            cursor: pointer;
            user-select: none;
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 10px;
            margin: -10px -10px 10px -10px;
            border-radius: 6px;
            transition: background 0.2s;
        }
        .info-section h3:hover {
            background: #f8f9fa;
        }
        .info-section h3::after {
            content: '▼';
            font-size: 0.7em;
            transition: transform 0.2s;
            color: #666;
        }
        .info-section.collapsed h3::after {
            transform: rotate(-90deg);
        }
        .info-section-content {
            transition: max-height 0.3s ease-out, opacity 0.3s ease-out;
            overflow: hidden;
            max-height: 2000px;
            opacity: 1;
        }
        .info-section.collapsed .info-section-content {
            max-height: 0;
            opacity: 0;
        }
        .info-item {
            margin-bottom: 10px;
            padding: 10px;
            background: #f8f9fa;
            border-radius: 6px;
            font-size: 0.9em;
        }
        .info-label {
            font-weight: 600;
            color: #666;
            margin-bottom: 3px;
        }
        .info-value {
            color: #333;
            word-break: break-word;
        }
        .status-badge {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 12px;
            font-size: 0.85em;
            font-weight: 600;
        }
        .status-connected {
            background: #d4edda;
            color: #155724;
        }
        .status-error {
            background: #f8d7da;
            color: #721c24;
        }
        .status-disabled {
            background: #d6d8db;
            color: #383d41;
        }
        .chat-container {
            flex: 1;
            display: flex;
            flex-direction: column;
            background: white;
            border-radius: 12px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.05);
            overflow: hidden;
        }
        .chat-messages {
            flex: 1;
            padding: 20px;
            overflow-y: auto;
        }
        .message {
            margin-bottom: 20px;
            display: flex;
            gap: 10px;
        }
        .message-user {
            justify-content: flex-end;
        }
        .message-avatar {
            width: 36px;
            height: 36px;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: 600;
            flex-shrink: 0;
        }
        .avatar-user {
            background: #667eea;
            color: white;
        }
        .avatar-agent {
            background: #28a745;
            color: white;
        }
        .message-content {
            max-width: 70%;
            padding: 12px 16px;
            border-radius: 12px;
            line-height: 1.5;
        }
        .content-user {
            background: #667eea;
            color: white;
        }
        .content-agent {
            background: #f8f9fa;
            color: #333;
        }
        .content-agent pre {
            background: #2d2d2d;
            color: #f8f8f2;
            padding: 12px;
            border-radius: 6px;
            overflow-x: auto;
            margin: 10px 0;
        }
        .content-agent code {
            background: #2d2d2d;
            color: #f8f8f2;
            padding: 2px 6px;
            border-radius: 3px;
        }
        .chat-input-container {
            padding: 20px;
            border-top: 1px solid #e9ecef;
            background: white;
        }
        .chat-input-wrapper {
            display: flex;
            gap: 10px;
        }
        .chat-input {
            flex: 1;
            padding: 12px 16px;
            border: 2px solid #e9ecef;
            border-radius: 24px;
            font-size: 0.95em;
            outline: none;
            transition: border-color 0.2s;
        }
        .chat-input:focus {
            border-color: #667eea;
        }
        .send-button {
            padding: 12px 24px;
            background: #667eea;
            color: white;
            border: none;
            border-radius: 24px;
            font-weight: 600;
            cursor: pointer;
            transition: background 0.2s;
        }
        .send-button:hover:not(:disabled) {
            background: #5568d3;
        }
        .send-button:disabled {
            opacity: 0.5;
            cursor: not-allowed;
        }
        .loading {
            display: none;
            text-align: center;
            padding: 20px;
            color: #666;
        }
        .loading.active {
            display: block;
        }
        .tools-list {
            max-height: 200px;
            overflow-y: auto;
            margin-top: 10px;
        }
        .tool-item {
            padding: 8px;
            background: #f8f9fa;
            border-radius: 4px;
            margin-bottom: 5px;
            font-size: 0.85em;
        }
        .error-message {
            background: #f8d7da;
            color: #721c24;
            padding: 12px;
            border-radius: 6px;
            margin-bottom: 15px;
        }
        .configure-button, .save-config-button {
            width: 100%;
            padding: 10px;
            margin-top: 10px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 6px;
            font-weight: 600;
            cursor: pointer;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .configure-button:hover, .save-config-button:hover {
            transform: translateY(-1px);
            box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
        }
        .configure-button:active, .save-config-button:active {
            transform: translateY(0);
        }
        .save-config-button {
            background: linear-gradient(135deg, #10b981 0%, #059669 100%);
        }
        .config-field {
            margin-bottom: 8px;
            font-size: 0.85em;
        }
        .config-label {
            font-weight: 600;
            color: #666;
            margin-bottom: 3px;
            display: block;
        }
        .config-input, .config-select {
            width: 100%;
            padding: 6px 8px;
            border: 1px solid #e9ecef;
            border-radius: 4px;
            font-size: 0.85em;
            transition: border-color 0.2s;
        }
        .config-input:focus, .config-select:focus {
            outline: none;
            border-color: #667eea;
        }
        .config-checkbox {
            margin-right: 6px;
        }
        .config-value {
            color: #333;
            word-break: break-word;
            padding: 6px 8px;
            background: #f8f9fa;
            border-radius: 4px;
        }
    </style>
</head>
<body>
    <div class="header">
        <a href="/servers" class="back-link">← Back to Servers</a>
        <h1>🤖 AI Diagnostic Agent</h1>
        <p>Server: <strong>` + serverName + `</strong></p>
    </div>

    <div class="container">
        <div class="sidebar">
            <div class="info-section">
                <h3 onclick="toggleSection(this)">📊 Server Status</h3>
                <div class="info-section-content">
                    <div id="server-info">Loading...</div>
                </div>
            </div>

            <div class="info-section">
                <h3 onclick="toggleSection(this)">⚙️ Configuration</h3>
                <div class="info-section-content">
                    <div id="config-info">Loading...</div>
                    <button class="save-config-button" onclick="saveConfiguration()" style="display:none;">💾 Save Changes</button>
                </div>
            </div>

            <div class="info-section">
                <h3 onclick="toggleSection(this)">🔧 Available Tools</h3>
                <div class="info-section-content">
                    <div id="tools-info">Loading...</div>
                </div>
            </div>

            <div class="info-section">
                <h3 onclick="toggleSection(this)">⚡ Quick Actions</h3>
                <div class="info-section-content">
                    <div id="quick-actions" style="font-size: 0.85em; line-height: 1.8;">
                        <div id="quick-actions-links">Loading...</div>
                    </div>
                </div>
            </div>

            <div class="info-section">
                <h3 onclick="toggleSection(this)">💭 Conversation Context</h3>
                <div class="info-section-content">
                    <div id="contextInfo" style="display: flex; flex-direction: column; gap: 8px; padding: 12px; background: #f8f9fa; border-radius: 6px; font-size: 13px;">
                        <div class="context-item">
                            <span style="font-weight: 500;">📊 Messages:</span>
                            <span id="contextMessages" style="margin-left: 8px; color: #495057;">Loading...</span>
                        </div>
                        <div class="context-item">
                            <span style="font-weight: 500;">🔢 Estimated Tokens:</span>
                            <span id="contextTokens" style="margin-left: 8px; color: #495057;">Loading...</span>
                        </div>
                        <div class="context-item">
                            <span style="font-weight: 500;">✂️ Context Status:</span>
                            <span id="contextStatus" style="margin-left: 8px; color: #495057;">Loading...</span>
                        </div>
                        <div class="context-item" id="pruningInfo" style="display: none;">
                            <span style="font-weight: 500;">📉 Last Pruning:</span>
                            <span id="pruningDetails" style="margin-left: 8px; color: #6c757d; font-size: 12px;">N/A</span>
                        </div>
                    </div>
                </div>
            </div>

            <div class="info-section">
                <h3 onclick="toggleSection(this)">🔌 MCP Communication</h3>
                <div class="info-section-content">
                    <div id="mcpCommunications" style="max-height: 400px; overflow-y: auto;">
                        <div style="color: #666; font-size: 0.9em; padding: 12px;">
                            No MCP communications yet. Start a conversation to see protocol messages.
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <div class="chat-container">
            <div class="chat-messages" id="chat-messages">
                <div class="message">
                    <div class="message-avatar avatar-agent">AI</div>
                    <div class="message-content content-agent">
                        <p>Hello! I'm your AI-powered diagnostic agent. I have access to:</p>
                        <ul style="margin: 10px 0 0 20px;">
                            <li>Server configuration and status</li>
                            <li>Available tools and their documentation</li>
                            <li>Server logs and error messages</li>
                            <li>GitHub repository information</li>
                        </ul>
                        <p style="margin-top: 10px;">How can I help you with <strong>` + serverName + `</strong>?</p>
                    </div>
                </div>
            </div>

            <div class="loading" id="loading">
                <p>AI is thinking...</p>
            </div>

            <div class="chat-input-container">
                <div class="chat-input-wrapper">
                    <input
                        type="text"
                        class="chat-input"
                        id="chat-input"
                        placeholder="Ask me anything about this server..."
                        autocomplete="off"
                    >
                    <button class="send-button" id="send-button">Send</button>
                </div>
            </div>
        </div>
    </div>

    <script>
        const serverName = "` + serverName + `";
        let sessionId = null;

        // Load server info
        async function loadServerInfo() {
            try {
                const response = await fetch('/api/servers');
                if (!response.ok) {
                    throw new Error('HTTP ' + response.status + ': ' + response.statusText);
                }

                const data = await response.json();
                const server = data.servers.find(s => s.name === serverName);

                if (!server) {
                    document.getElementById('server-info').innerHTML =
                        '<div class="error-message">Server "' + serverName + '" not found in configuration.<br><button onclick="loadServerInfo()" style="margin-top:10px;padding:6px 12px;background:#667eea;color:white;border:none;border-radius:4px;cursor:pointer;">Retry</button></div>';
                    return;
                }

                const statusClass = server.enabled ?
                    (server.connection_state === 'Ready' ? 'status-connected' : 'status-error') :
                    'status-disabled';

                let html = '<div class="info-item">';
                html += '<div class="info-label">Status</div>';
                html += '<div class="info-value"><span class="status-badge ' + statusClass + '">' + (server.connection_state || 'Unknown') + '</span></div>';
                html += '</div>';

                html += '<div class="info-item">';
                html += '<div class="info-label">Enabled</div>';
                html += '<div class="info-value">' + (server.enabled ? 'Yes' : 'No') + '</div>';
                html += '</div>';

                if (server.protocol) {
                    html += '<div class="info-item"><div class="info-label">Protocol</div><div class="info-value">' + server.protocol + '</div></div>';
                }

                if (server.url) {
                    html += '<div class="info-item"><div class="info-label">URL</div><div class="info-value" style="word-break:break-all;">' + server.url + '</div></div>';
                }

                if (server.command) {
                    html += '<div class="info-item"><div class="info-label">Command</div><div class="info-value">' + server.command + '</div></div>';
                }

                if (server.working_dir) {
                    html += '<div class="info-item"><div class="info-label">Working Dir</div><div class="info-value" style="word-break:break-all;">' + server.working_dir + '</div></div>';
                }

                if (server.quarantined) {
                    html += '<div class="error-message">⚠️ <strong>Quarantined:</strong><br>Server is quarantined for security review</div>';
                }

                if (server.last_error) {
                    html += '<div class="error-message"><strong>Last Error:</strong><br>' + server.last_error + '</div>';
                }

                document.getElementById('server-info').innerHTML = html;

                // Display quick actions
                displayQuickActions(server);
            } catch (error) {
                console.error('Failed to load server info:', error);
                document.getElementById('server-info').innerHTML =
                    '<div class="error-message"><strong>Failed to load server info</strong><br>' +
                    'Error: ' + error.message +
                    '<br><button onclick="loadServerInfo()" style="margin-top:10px;padding:6px 12px;background:#667eea;color:white;border:none;border-radius:4px;cursor:pointer;">Retry</button></div>';
            }
        }

        // Display quick action links
        function displayQuickActions(server) {
            let html = '';

            // Log File link
            const logFile = '/Users/hrannow/Library/Logs/mcpproxy/server-' + encodeURIComponent(server.name) + '.log';
            html += '<div style="margin-bottom: 8px;">';
            html += '📄 <a href="#" onclick="openPath(\'' + logFile + '\'); return false;" style="color:#667eea;text-decoration:none;">Log File</a>';
            html += '</div>';

            // Configuration file link
            html += '<div style="margin-bottom: 8px;">';
            html += '⚙️ <a href="#" onclick="openPath(\'/Users/hrannow/.mcpproxy/mcp_config.json\'); return false;" style="color:#667eea;text-decoration:none;">Configuration</a>';
            html += '</div>';

            // Repository link (if available)
            if (server.repository_url) {
                html += '<div style="margin-bottom: 8px;">';
                html += '🔗 <a href="' + server.repository_url + '" target="_blank" style="color:#667eea;text-decoration:none;">Repository</a>';
                html += '</div>';
            }

            document.getElementById('quick-actions-links').innerHTML = html;
        }

        // Open path using API
        async function openPath(path) {
            try {
                const response = await fetch('/api/open-path', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ path: path })
                });

                if (!response.ok) {
                    const data = await response.json();
                    alert('Failed to open path: ' + (data.error || 'Unknown error'));
                }
            } catch (error) {
                console.error('Failed to open path:', error);
                alert('Failed to open path: ' + error.message);
            }
        }

        // Toggle section collapse state
        function toggleSection(headerElement) {
            const section = headerElement.parentElement;
            section.classList.toggle('collapsed');
        }

        // Current server configuration
        let currentConfig = null;
        let configModified = false;

        // Load server configuration
        async function loadConfiguration() {
            try {
                const response = await fetch('/api/servers');
                if (!response.ok) {
                    throw new Error('HTTP ' + response.status + ': ' + response.statusText);
                }

                const data = await response.json();
                const server = data.servers.find(s => s.name === serverName);

                if (!server) {
                    document.getElementById('config-info').innerHTML =
                        '<div class="error-message">Server configuration not found</div>';
                    return;
                }

                currentConfig = server;
                displayConfiguration(server);
            } catch (error) {
                console.error('Failed to load configuration:', error);
                document.getElementById('config-info').innerHTML =
                    '<div class="error-message">Failed to load configuration<br>' + error.message + '</div>';
            }
        }

        // Display configuration fields
        function displayConfiguration(server) {
            let html = '';

            // Enabled
            html += '<div class="config-field">';
            html += '<label class="config-label"><input type="checkbox" class="config-checkbox" id="cfg-enabled" ' +
                    (server.enabled ? 'checked' : '') + ' onchange="markConfigModified()"> Enabled</label>';
            html += '</div>';

            // Description
            html += '<div class="config-field">';
            html += '<span class="config-label">Description</span>';
            html += '<input type="text" class="config-input" id="cfg-description" value="' + (server.description || '') + '" onchange="markConfigModified()" placeholder="Optional description">';
            html += '</div>';

            // Protocol
            html += '<div class="config-field">';
            html += '<span class="config-label">Protocol</span>';
            html += '<select class="config-select" id="cfg-protocol" onchange="markConfigModified();toggleProtocolFields()">';
            ['stdio', 'http', 'sse', 'streamable-http'].forEach(function(proto) {
                html += '<option value="' + proto + '"' + (server.protocol === proto ? ' selected' : '') + '>' + proto + '</option>';
            });
            html += '</select>';
            html += '</div>';

            // stdio fields
            html += '<div id="stdio-fields" style="' + (server.protocol === 'stdio' ? '' : 'display:none;') + '">';

            html += '<div class="config-field">';
            html += '<span class="config-label">Command</span>';
            html += '<input type="text" class="config-input" id="cfg-command" value="' + (server.command || '') + '" onchange="markConfigModified()">';
            html += '</div>';

            // Arguments
            html += '<div class="config-field">';
            html += '<span class="config-label">Arguments</span>';
            html += '<div id="args-list" style="display: flex; flex-direction: column; gap: 8px;">';
            if (server.args && server.args.length > 0) {
                server.args.forEach(function(arg, idx) {
                    html += '<div class="arg-row" style="display: flex; gap: 8px;">';
                    html += '<input type="text" class="config-input arg-input" data-index="' + idx + '" value="' + arg + '" onchange="markConfigModified()" placeholder="Argument">';
                    html += '<button type="button" class="remove-btn" onclick="removeArg(' + idx + ')" style="padding: 6px 12px; background: #dc3545; color: white; border: none; border-radius: 4px; cursor: pointer;">Remove</button>';
                    html += '</div>';
                });
            }
            html += '</div>';
            html += '<button type="button" onclick="addArg()" style="margin-top: 8px; padding: 6px 12px; background: #28a745; color: white; border: none; border-radius: 4px; cursor: pointer;">Add Argument</button>';
            html += '</div>';

            html += '<div class="config-field">';
            html += '<span class="config-label">Working Directory</span>';
            html += '<input type="text" class="config-input" id="cfg-working-dir" value="' + (server.working_dir || '') + '" onchange="markConfigModified()" placeholder="Optional">';
            html += '</div>';

            // Environment Variables
            html += '<div class="config-field">';
            html += '<span class="config-label">Environment Variables</span>';
            html += '<div id="env-list" style="display: flex; flex-direction: column; gap: 8px;">';
            if (server.env && Object.keys(server.env).length > 0) {
                Object.keys(server.env).forEach(function(key) {
                    html += '<div class="env-row" style="display: grid; grid-template-columns: 1fr 1fr auto; gap: 8px;">';
                    html += '<input type="text" class="config-input env-key" value="' + key + '" onchange="markConfigModified()" placeholder="Variable name">';
                    html += '<input type="text" class="config-input env-value" value="' + (server.env[key] || '') + '" onchange="markConfigModified()" placeholder="Value">';
                    html += '<button type="button" class="remove-btn" onclick="removeEnv(this)" style="padding: 6px 12px; background: #dc3545; color: white; border: none; border-radius: 4px; cursor: pointer;">Remove</button>';
                    html += '</div>';
                });
            }
            html += '</div>';
            html += '<button type="button" onclick="addEnv()" style="margin-top: 8px; padding: 6px 12px; background: #28a745; color: white; border: none; border-radius: 4px; cursor: pointer;">Add Variable</button>';
            html += '</div>';

            html += '</div>';

            // HTTP fields
            html += '<div id="http-fields" style="' + (server.protocol !== 'stdio' ? '' : 'display:none;') + '">';

            html += '<div class="config-field">';
            html += '<span class="config-label">URL</span>';
            html += '<input type="text" class="config-input" id="cfg-url" value="' + (server.url || '') + '" onchange="markConfigModified()">';
            html += '</div>';

            html += '</div>';

            // Repository URL
            html += '<div class="config-field">';
            html += '<span class="config-label">Repository URL</span>';
            html += '<input type="text" class="config-input" id="cfg-repo-url" value="' + (server.repository_url || '') + '" onchange="markConfigModified()" placeholder="Optional">';
            html += '</div>';

            // Quarantined
            if (server.quarantined !== undefined) {
                html += '<div class="config-field">';
                html += '<label class="config-label"><input type="checkbox" class="config-checkbox" id="cfg-quarantined" ' +
                        (server.quarantined ? 'checked' : '') + ' onchange="markConfigModified()"> Quarantined</label>';
                html += '</div>';
            }

            // Start on Boot
            html += '<div class="config-field">';
            html += '<label class="config-label"><input type="checkbox" class="config-checkbox" id="cfg-start-on-boot" ' +
                    (server.start_on_boot ? 'checked' : '') + ' onchange="markConfigModified()"> Start on Boot</label>';
            html += '</div>';

            // Health Check
            html += '<div class="config-field">';
            html += '<label class="config-label"><input type="checkbox" class="config-checkbox" id="cfg-health-check" ' +
                    (server.health_check ? 'checked' : '') + ' onchange="markConfigModified()"> Health Check</label>';
            html += '</div>';

            document.getElementById('config-info').innerHTML = html;
        }

        // Toggle protocol-specific fields
        function toggleProtocolFields() {
            const protocol = document.getElementById('cfg-protocol').value;
            const isStdio = protocol === 'stdio';

            const stdioFields = document.getElementById('stdio-fields');
            const httpFields = document.getElementById('http-fields');

            if (stdioFields) stdioFields.style.display = isStdio ? '' : 'none';
            if (httpFields) httpFields.style.display = isStdio ? 'none' : '';
        }

        // Add argument field
        function addArg() {
            const argsList = document.getElementById('args-list');
            const idx = argsList.children.length;
            const div = document.createElement('div');
            div.className = 'arg-row';
            div.style.cssText = 'display: flex; gap: 8px;';
            div.innerHTML = '<input type="text" class="config-input arg-input" data-index="' + idx + '" onchange="markConfigModified()" placeholder="Argument">' +
                           '<button type="button" class="remove-btn" onclick="removeArg(' + idx + ')" style="padding: 6px 12px; background: #dc3545; color: white; border: none; border-radius: 4px; cursor: pointer;">Remove</button>';
            argsList.appendChild(div);
            markConfigModified();
        }

        // Remove argument field
        function removeArg(idx) {
            const rows = document.querySelectorAll('.arg-row');
            if (rows[idx]) {
                rows[idx].remove();
                markConfigModified();
            }
        }

        // Add environment variable field
        function addEnv() {
            const envList = document.getElementById('env-list');
            const div = document.createElement('div');
            div.className = 'env-row';
            div.style.cssText = 'display: grid; grid-template-columns: 1fr 1fr auto; gap: 8px;';
            div.innerHTML = '<input type="text" class="config-input env-key" onchange="markConfigModified()" placeholder="Variable name">' +
                           '<input type="text" class="config-input env-value" onchange="markConfigModified()" placeholder="Value">' +
                           '<button type="button" class="remove-btn" onclick="removeEnv(this)" style="padding: 6px 12px; background: #dc3545; color: white; border: none; border-radius: 4px; cursor: pointer;">Remove</button>';
            envList.appendChild(div);
            markConfigModified();
        }

        // Remove environment variable field
        function removeEnv(btn) {
            btn.parentElement.remove();
            markConfigModified();
        }

        // Mark configuration as modified
        function markConfigModified() {
            configModified = true;
            document.querySelector('.save-config-button').style.display = 'block';
        }

        // Save configuration
        async function saveConfiguration() {
            if (!currentConfig) {
                alert('No configuration loaded');
                return;
            }

            try {
                // Collect arguments
                const args = [];
                document.querySelectorAll('.arg-input').forEach(input => {
                    if (input.value.trim()) {
                        args.push(input.value.trim());
                    }
                });

                // Collect environment variables
                const env = {};
                document.querySelectorAll('.env-row').forEach(row => {
                    const key = row.querySelector('.env-key').value.trim();
                    const value = row.querySelector('.env-value').value.trim();
                    if (key) {
                        env[key] = value;
                    }
                });

                // Build updated config from form fields
                const updatedConfig = {
                    name: serverName,
                    description: document.getElementById('cfg-description')?.value || '',
                    enabled: document.getElementById('cfg-enabled').checked,
                    protocol: document.getElementById('cfg-protocol').value,
                    command: document.getElementById('cfg-command')?.value || '',
                    args: args,
                    working_dir: document.getElementById('cfg-working-dir')?.value || '',
                    env: env,
                    url: document.getElementById('cfg-url')?.value || '',
                    repository_url: document.getElementById('cfg-repo-url')?.value || '',
                    quarantined: document.getElementById('cfg-quarantined')?.checked || false,
                    start_on_boot: document.getElementById('cfg-start-on-boot')?.checked || false,
                    health_check: document.getElementById('cfg-health-check')?.checked || false
                };

                // Send update request
                const response = await fetch('/api/servers/' + encodeURIComponent(serverName) + '/config', {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify(updatedConfig)
                });

                if (!response.ok) {
                    throw new Error('HTTP ' + response.status + ': ' + response.statusText);
                }

                // Reload everything
                configModified = false;
                document.querySelector('.save-config-button').style.display = 'none';

                await Promise.all([
                    loadServerInfo(),
                    loadConfiguration()
                ]);

                alert('Configuration saved successfully!');
            } catch (error) {
                console.error('Failed to save configuration:', error);
                alert('Failed to save configuration: ' + error.message);
            }
        }

        // Auto-refresh configuration when agent makes changes
        function startConfigAutoRefresh() {
            setInterval(async function() {
                if (!configModified) {
                    // Only refresh if user hasn't made manual changes
                    await loadConfiguration();
                    await loadServerInfo();
                }
            }, 5000); // Check every 5 seconds
        }

        // Load tools
        async function loadTools() {
            try {
                const response = await fetch('/api/servers/' + encodeURIComponent(serverName) + '/tools');
                const data = await response.json();

                if (data.tools && data.tools.length > 0) {
                    let html = '<div class="tools-list" style="max-height: 400px; overflow-y: auto;">';
                    html += '<div style="padding: 8px; color: #666; font-size: 0.85em; border-bottom: 1px solid #e9ecef;">Total: ' + data.tools.length + ' tools</div>';
                    data.tools.forEach(tool => {
                        html += '<div class="tool-item">' + tool.name;
                        if (tool.description) {
                            html += '<br><span style="color: #666; font-size: 0.9em;">' + tool.description.substring(0, 100) + (tool.description.length > 100 ? '...' : '') + '</span>';
                        }
                        html += '</div>';
                    });
                    html += '</div>';
                    document.getElementById('tools-info').innerHTML = html;
                } else {
                    document.getElementById('tools-info').innerHTML = '<div style="color: #666; font-size: 0.9em;">No tools available</div>';
                }
            } catch (error) {
                document.getElementById('tools-info').innerHTML = '<div class="error-message">Failed to load tools</div>';
            }
        }

        // Send message
        async function sendMessage() {
            const input = document.getElementById('chat-input');
            const message = input.value.trim();
            if (!message) return;

            // Add user message to chat
            addMessage('user', message);
            input.value = '';

            // Show loading
            document.getElementById('loading').classList.add('active');
            document.getElementById('send-button').disabled = true;

            try {
                // Create session if needed
                if (!sessionId) {
                    const sessionResponse = await fetch('/api/chat/sessions', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ server_name: serverName })
                    });
                    const sessionData = await sessionResponse.json();
                    sessionId = sessionData.session_id;
                }

                // Send message
                const response = await fetch('/api/chat/sessions/' + sessionId + '/messages', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        message: message,
                        agent_type: 'llm'
                    })
                });

                const data = await response.json();

                if (data.error) {
                    addMessage('agent', 'Error: ' + data.error, true);
                } else {
                    addMessage('agent', data.response);
                }

                // Display MCP communications if available
                if (data.mcp_communications && data.mcp_communications.length > 0) {
                    displayMCPCommunications(data.mcp_communications);
                }
            } catch (error) {
                addMessage('agent', 'Error: Failed to communicate with AI agent. Make sure OPENAI_API_KEY is set.', true);
            } finally {
                document.getElementById('loading').classList.remove('active');
                document.getElementById('send-button').disabled = false;
                // Update context info after message is processed
                updateContextInfo();
            }
        }

        // Display MCP communications in sidebar
        function displayMCPCommunications(communications) {
            const mcpDiv = document.getElementById('mcpCommunications');

            // Clear placeholder if it's the first communication
            if (mcpDiv.querySelector('[style*="No MCP communications"]')) {
                mcpDiv.innerHTML = '';
            }

            communications.forEach(comm => {
                const commDiv = document.createElement('div');
                commDiv.style.cssText = 'padding: 10px; margin-bottom: 8px; background: #f8f9fa; border-left: 3px solid ' +
                    (comm.error ? '#dc3545' : '#667eea') + '; border-radius: 4px; font-size: 12px;';

                let html = '<div style="margin-bottom: 6px;">';
                html += '<strong style="color: #495057;">' + comm.server + ' : ' + comm.tool + '</strong>';
                html += '<span style="float: right; color: #6c757d; font-size: 11px;">' + new Date(comm.timestamp).toLocaleTimeString() + '</span>';
                html += '</div>';

                if (comm.error) {
                    html += '<div style="color: #dc3545; margin-top: 4px;">❌ Error: ' + comm.error + '</div>';
                }

                // Add collapsible request section
                html += '<details style="margin-top: 4px;">';
                html += '<summary style="cursor: pointer; color: #667eea; font-weight: 500;">📤 Request</summary>';
                html += '<pre style="background: white; padding: 8px; margin-top: 4px; border-radius: 4px; overflow-x: auto; font-size: 11px;">' +
                    JSON.stringify(comm.request, null, 2) + '</pre>';
                html += '</details>';

                // Add collapsible response section if available
                if (comm.response) {
                    html += '<details style="margin-top: 4px;">';
                    html += '<summary style="cursor: pointer; color: #10b981; font-weight: 500;">📥 Response</summary>';
                    html += '<pre style="background: white; padding: 8px; margin-top: 4px; border-radius: 4px; overflow-x: auto; font-size: 11px;">' +
                        JSON.stringify(comm.response, null, 2) + '</pre>';
                    html += '</details>';
                }

                commDiv.innerHTML = html;
                mcpDiv.appendChild(commDiv);
            });

            // Scroll to bottom of MCP communications
            mcpDiv.scrollTop = mcpDiv.scrollHeight;
        }

        // Add message to chat
        function addMessage(role, content, isError = false) {
            const messagesDiv = document.getElementById('chat-messages');
            const messageDiv = document.createElement('div');
            messageDiv.className = 'message' + (role === 'user' ? ' message-user' : '');

            const avatar = document.createElement('div');
            avatar.className = 'message-avatar avatar-' + role;
            avatar.textContent = role === 'user' ? 'U' : 'AI';

            const contentDiv = document.createElement('div');
            contentDiv.className = 'message-content content-' + role;

            // Format content (convert markdown-style code blocks)
            let formattedContent = content;
            // Use triple-tilde instead of triple-backtick to avoid string delimiter conflicts
            formattedContent = formattedContent.replace(/~~~(\w+)?\n([\s\S]*?)~~~/g, '<pre><code>$2</code></pre>');
            formattedContent = formattedContent.replace(/~([^~]+)~/g, '<code>$1</code>');
            formattedContent = formattedContent.replace(/\n/g, '<br>');

            contentDiv.innerHTML = formattedContent;

            if (role === 'user') {
                messageDiv.appendChild(contentDiv);
                messageDiv.appendChild(avatar);
            } else {
                messageDiv.appendChild(avatar);
                messageDiv.appendChild(contentDiv);
            }

            messagesDiv.appendChild(messageDiv);
            messagesDiv.scrollTop = messagesDiv.scrollHeight;
        }

        // Update context information
        function updateContextInfo() {
            if (!sessionId) {
                return;
            }

            fetch('/chat/context?session_id=' + sessionId)
            .then(response => response.json())
            .then(data => {
                // Update message count
                const messagesEl = document.getElementById('contextMessages');
                messagesEl.textContent = data.total_messages + ' messages';
                messagesEl.style.color = data.total_messages > 50 ? '#dc3545' : '#495057';

                // Update token count
                const tokensEl = document.getElementById('contextTokens');
                tokensEl.textContent = data.estimated_tokens.toLocaleString() + ' tokens';

                // Color code based on token usage
                const tokenPercentage = data.estimated_tokens / data.max_tokens;
                if (tokenPercentage > 0.8) {
                    tokensEl.style.color = '#dc3545'; // Red
                } else if (tokenPercentage > 0.6) {
                    tokensEl.style.color = '#ffc107'; // Yellow
                } else {
                    tokensEl.style.color = '#28a745'; // Green
                }

                // Update context status
                const statusEl = document.getElementById('contextStatus');
                if (data.pruning_active) {
                    statusEl.textContent = '✂️ Pruning Active';
                    statusEl.style.color = '#ffc107';
                    statusEl.style.fontWeight = '600';
                } else {
                    statusEl.textContent = '✅ Normal';
                    statusEl.style.color = '#28a745';
                }

                // Show/hide pruning info
                const pruningInfoEl = document.getElementById('pruningInfo');
                const pruningDetailsEl = document.getElementById('pruningDetails');

                if (data.last_pruning) {
                    pruningInfoEl.style.display = 'block';
                    const saved = data.last_pruning.tokens_saved || 0;
                    const original = data.last_pruning.original_messages || 0;
                    const pruned = data.last_pruning.pruned_messages || 0;

                    pruningDetailsEl.textContent =
                        saved.toLocaleString() + ' tokens saved (' +
                        original + ' → ' + pruned + ' msgs)';
                } else {
                    pruningInfoEl.style.display = 'none';
                }
            })
            .catch(error => {
                console.log('Context update error:', error);
                // Don't show errors to user - just log them
            });
        }

        // Event listeners
        document.getElementById('send-button').addEventListener('click', sendMessage);
        document.getElementById('chat-input').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') sendMessage();
        });

        // Load initial data
        loadServerInfo();
        loadConfiguration();
        loadTools();

        // Start auto-refresh
        startConfigAutoRefresh();

        // Initialize context monitoring
        updateContextInfo();
        setInterval(updateContextInfo, 5000); // Update every 5 seconds
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

// handleChatSession creates or retrieves a chat session
func (s *Server) handleChatSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ServerName string `json:"server_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// For now, create a simple session ID
	// In a real implementation, this would use the ChatSystem
	sessionID := fmt.Sprintf("session_%s_%d", req.ServerName, 1)

	response := map[string]interface{}{
		"session_id":  sessionID,
		"server_name": req.ServerName,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleChatMessage sends a message to the LLM agent using OpenAI directly
func (s *Server) handleChatMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract session ID from URL path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}
	sessionID := parts[4]

	var req struct {
		Message   string `json:"message"`
		AgentType string `json:"agent_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Extract server name from session ID
	serverName := strings.TrimPrefix(sessionID, "session_")
	serverName = strings.Split(serverName, "_")[0]

	// Check for OpenAI API key
	// Priority: Config file (.env or mcp_config.json) > Environment variables
	apiKey := ""
	if s.config.LLM != nil && s.config.LLM.OpenAIKey != "" {
		apiKey = s.config.LLM.OpenAIKey
	} else {
		// Fallback to environment variables
		apiKey = os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_KEY")
		}
	}

	if apiKey == "" {
		responseData := map[string]interface{}{
			"error": "OpenAI API key not configured. Please set OPENAI_API_KEY in .env file or environment variable.",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseData)
		return
	}

	// Get or create session
	session := sessions.getOrCreateSession(sessionID, serverName)

	// Add system message on first interaction
	if len(session.Messages) == 0 {
		serverContext := s.buildServerContext(serverName)
		systemPrompt := fmt.Sprintf(`You are an expert diagnostic agent for MCP (Model Context Protocol) servers.

IMPORTANT: You have direct access to the following tools that you can call autonomously:

Built-in Diagnostic Tools:
- read_config: Read the mcp_config.json configuration file to analyze current server setup
- write_config: Write/update the mcp_config.json file (automatically creates backups and reloads configuration)
- read_log: Read server log files to view MCP communication, errors, and diagnostic information
- read_github: Fetch documentation or README from GitHub repository URLs

Server-Specific Tools:
All tools from the %s server are also available and can be called directly.
These server-specific tools are prefixed with the server name (e.g., %s:tool_name).

You can and should use these tools directly whenever needed. DO NOT suggest that the user should manually execute tools or commands.

Your capabilities include:
1. Configuration Analysis and Updates - Analyze and fix server configurations using read_config and write_config
2. Log Analysis - Diagnose issues from logs using read_log
3. Documentation Analysis - Fetch and analyze GitHub docs using read_github
4. Installation Help - Guide users through server setup and installation
5. Testing Assistance - Help test server functionality and troubleshoot issues
6. Server Management - Start, stop, enable, disable servers
7. Tool Testing - Test individual tools with appropriate parameters
8. General Troubleshooting - Solve any MCP server-related problems

%s

=== Instructions ===
Always use tools proactively - don't ask the user to do things you can do yourself.

EFFICIENCY GUIDELINES:
- You have a maximum of 10 tool call iterations per response
- Use tools efficiently - call only what's necessary to answer the question
- When creating and testing tools, do it in ONE iteration: create test → execute test → return results
- If a task requires many tool calls, break it into smaller steps and complete the most critical parts first
- After tool calls, provide a complete answer - don't make additional tool calls unless absolutely necessary

Example: When asked "Can you read the config?", immediately call the read_config tool instead of explaining how the user could read it manually.

Please provide clear, actionable responses that:
- Directly address the user's questions
- Use read_config tool to analyze server configuration and status
- Use read_log tool to investigate errors and diagnostic information
- Use write_config tool to fix configuration issues (automatic backup and reload)
- Use read_github tool to fetch repository documentation
- Identify potential issues from error messages or status
- Provide specific steps or solutions when applicable
- Include relevant code examples or configuration snippets when helpful
- Suggest tool tests with example parameters if testing is needed
- Recommend next steps or follow-up actions

If the server has errors:
1. Explain what the error means
2. Use read_config and read_log to identify the root cause
3. Provide step-by-step fix instructions
4. Use write_config to apply the fix if needed
5. Suggest verification steps after fix

If testing tools:
1. Explain what the tool does
2. Suggest appropriate test parameters
3. Explain expected results
4. Help interpret actual results`, serverName, serverName, serverContext)

		sessions.addMessage(sessionID, "system", systemPrompt)
	}

	// Add user message to history
	sessions.addMessage(sessionID, "user", req.Message)

	// Get all messages for OpenAI
	messages := sessions.getMessages(sessionID)

	// Prune messages if context is too large (keep system prompt + recent messages)
	// More aggressive pruning: target 40K tokens to account for OpenAI's tokenization
	// and leave room for function definitions (~151 tokens) + response (~2000 tokens)
	messages = pruneMessages(messages, 40000)

	// Call OpenAI API with full conversation history and tools support
	response, mcpCommunications, err := s.callOpenAIWithTools(apiKey, messages, serverName)
	if err != nil {
		responseData := map[string]interface{}{
			"error":               fmt.Sprintf("AI request failed: %v", err),
			"mcp_communications":  mcpCommunications,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseData)
		return
	}

	// Add assistant response to history
	sessions.addMessage(sessionID, "assistant", response)

	responseData := map[string]interface{}{
		"response":           response,
		"mcp_communications": mcpCommunications,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseData)
}

// pruneMessages removes old messages to keep total tokens under target
// Always preserves system prompt (first message)
func pruneMessages(messages []chatMessage, targetTokens int) []chatMessage {
	if len(messages) == 0 {
		return messages
	}

	// Calculate total tokens (conservative estimate: 1 token ≈ 3 characters)
	// OpenAI's actual tokenization is more aggressive than character count suggests
	totalTokens := 0
	for _, msg := range messages {
		totalTokens += len(msg.Content) / 3
	}

	// If under target, no pruning needed
	if totalTokens <= targetTokens {
		return messages
	}

	// Start with system prompt (always first message)
	pruned := []chatMessage{}
	systemPromptTokens := 0

	if len(messages) > 0 && messages[0].Role == "system" {
		pruned = append(pruned, messages[0])
		systemPromptTokens = len(messages[0].Content) / 3
	}

	// Add messages from most recent backwards until we hit token limit
	// Leave 50% buffer for response, tool definitions, and safety margin
	effectiveTarget := int(float64(targetTokens) * 0.5)
	currentTokens := systemPromptTokens

	// Collect messages in reverse order (excluding system prompt)
	startIdx := 1
	if len(messages) > 0 && messages[0].Role == "system" {
		startIdx = 1
	} else {
		startIdx = 0
	}

	// Build from end backwards
	recentMessages := []chatMessage{}
	for i := len(messages) - 1; i >= startIdx; i-- {
		msgTokens := len(messages[i].Content) / 3
		if currentTokens+msgTokens > effectiveTarget {
			break
		}
		recentMessages = append([]chatMessage{messages[i]}, recentMessages...)
		currentTokens += msgTokens
	}

	// Combine system prompt + recent messages
	pruned = append(pruned, recentMessages...)

	// Log pruning statistics
	originalCount := len(messages)
	prunedCount := len(pruned)
	tokensSaved := totalTokens - currentTokens

	if prunedCount < originalCount {
		fmt.Printf("Context pruned: %d → %d messages, %d → %d tokens (saved %d tokens)\n",
			originalCount, prunedCount, totalTokens, currentTokens, tokensSaved)
	}

	return pruned
}

// buildServerContext creates comprehensive context about a server
func (s *Server) buildServerContext(serverName string) string {
	servers, err := s.GetAllServers()
	if err != nil {
		return fmt.Sprintf("=== Context ===\nCurrent Server: %s\n⚠️ Could not retrieve server information", serverName)
	}

	var serverInfo string
	for _, srv := range servers {
		if name, ok := srv["name"].(string); ok && name == serverName {
			// Basic server information
			serverInfo = fmt.Sprintf("\n\n=== Server Information ===\n- Name: %s\n- Enabled: %v\n- Protocol: %v",
				name,
				srv["enabled"],
				srv["protocol"])

			// Connection details
			if url, ok := srv["url"].(string); ok && url != "" {
				serverInfo += fmt.Sprintf("\n- URL: %s", url)
			}
			if cmd, ok := srv["command"].(string); ok && cmd != "" {
				serverInfo += fmt.Sprintf("\n- Command: %s", cmd)
			}
			if args, ok := srv["args"].([]interface{}); ok && len(args) > 0 {
				argsStr := make([]string, len(args))
				for i, arg := range args {
					argsStr[i] = fmt.Sprintf("%v", arg)
				}
				serverInfo += fmt.Sprintf("\n- Arguments: %s", strings.Join(argsStr, " "))
			}

			// Working directory
			if workingDir, ok := srv["working_dir"].(string); ok && workingDir != "" {
				serverInfo += fmt.Sprintf("\n- Working Directory: %s", workingDir)
			}

			// Status and state
			if status, ok := srv["status"].(string); ok {
				serverInfo += fmt.Sprintf("\n- Status: %s", status)
			}
			if state, ok := srv["state"].(string); ok {
				serverInfo += fmt.Sprintf("\n- Connection State: %s", state)
			}

			// Error information
			if lastError, ok := srv["last_error"].(string); ok && lastError != "" {
				serverInfo += fmt.Sprintf("\n- Last Error: %s", lastError)
			}

			// Configuration path
			configPath := s.GetConfigPath()
			if configPath != "" {
				serverInfo += fmt.Sprintf("\n- Config File: %s", configPath)
			}

			// Repository URL if available
			if repoURL, ok := srv["repository_url"].(string); ok && repoURL != "" {
				serverInfo += fmt.Sprintf("\n- Repository: %s", repoURL)
			} else if homepage, ok := srv["homepage"].(string); ok && homepage != "" {
				serverInfo += fmt.Sprintf("\n- Homepage: %s", homepage)
			}

			// Docker isolation info
			if isolated, ok := srv["docker_isolated"].(bool); ok && isolated {
				serverInfo += "\n- Docker Isolation: Enabled"
				if image, ok := srv["docker_image"].(string); ok && image != "" {
					serverInfo += fmt.Sprintf("\n- Docker Image: %s", image)
				}
			}

			// Quarantine status
			if quarantined, ok := srv["quarantined"].(bool); ok && quarantined {
				serverInfo += "\n- ⚠️  QUARANTINED: Server is quarantined for security review"
			}

			break
		}
	}

	// Get available tools for this server
	tools, err := s.GetServerTools(serverName)
	if err == nil && len(tools) > 0 {
		serverInfo += fmt.Sprintf("\n\n=== Available Tools (%d) ===", len(tools))
		for i, tool := range tools {
			if i >= 10 {
				serverInfo += fmt.Sprintf("\n... and %d more tools", len(tools)-10)
				break
			}
			if name, ok := tool["name"].(string); ok {
				toolDesc := ""
				if desc, ok := tool["description"].(string); ok && desc != "" {
					// Truncate description to 100 chars
					if len(desc) > 100 {
						desc = desc[:100] + "..."
					}
					toolDesc = fmt.Sprintf(" - %s", desc)
				}
				serverInfo += fmt.Sprintf("\n  %d. %s%s", i+1, name, toolDesc)
			}
		}
	} else if err != nil {
		serverInfo += fmt.Sprintf("\n\n⚠️  Could not retrieve tools: %v", err)
	}

	return fmt.Sprintf("=== Context ===\nCurrent Server: %s%s", serverName, serverInfo)
}

// callOpenAI makes a request to OpenAI API with conversation history
func (s *Server) callOpenAI(apiKey string, messages []chatMessage) (string, error) {
	// OpenAI API endpoint
	apiURL := "https://api.openai.com/v1/chat/completions"

	// Prepare request payload
	reqBody := openAIRequest{
		Model:       "gpt-4o-mini", // Use gpt-4o-mini for cost efficiency, can be changed to "gpt-4" for better quality
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	// Marshal request to JSON
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	// Send request
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var openAIResp openAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API errors
	if openAIResp.Error != nil {
		return "", fmt.Errorf("OpenAI API error: %s (type: %s, code: %s)",
			openAIResp.Error.Message,
			openAIResp.Error.Type,
			openAIResp.Error.Code)
	}

	// Extract response text
	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response choices from OpenAI")
	}

	response := openAIResp.Choices[0].Message.Content

	// Log token usage
	s.logger.Debug("OpenAI API call completed",
		zap.String("model", openAIResp.Model),
		zap.Int("prompt_tokens", openAIResp.Usage.PromptTokens),
		zap.Int("completion_tokens", openAIResp.Usage.CompletionTokens),
		zap.Int("total_tokens", openAIResp.Usage.TotalTokens))

	return response, nil
}
