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
                <h3>📊 Server Status</h3>
                <div id="server-info">Loading...</div>
            </div>

            <div class="info-section">
                <h3>🔧 Available Tools</h3>
                <div id="tools-info">Loading...</div>
            </div>

            <div class="info-section">
                <h3>ℹ️ Quick Actions</h3>
                <div style="font-size: 0.85em; color: #666; line-height: 1.6;">
                    Ask the AI agent to:
                    <ul style="margin: 10px 0 0 20px;">
                        <li>Analyze server configuration</li>
                        <li>Diagnose connection errors</li>
                        <li>Test specific tools</li>
                        <li>Review server logs</li>
                        <li>Suggest fixes</li>
                    </ul>
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
                const data = await response.json();
                const server = data.servers.find(s => s.name === serverName);

                if (server) {
                    const statusClass = server.enabled ?
                        (server.state === 'Ready' ? 'status-connected' : 'status-error') :
                        'status-disabled';

                    let html = '<div class="info-item">';
                    html += '<div class="info-label">Status</div>';
                    html += '<div class="info-value"><span class="status-badge ' + statusClass + '">' + (server.state || 'Unknown') + '</span></div>';
                    html += '</div>';

                    if (server.protocol) {
                        html += '<div class="info-item"><div class="info-label">Protocol</div><div class="info-value">' + server.protocol + '</div></div>';
                    }

                    if (server.url) {
                        html += '<div class="info-item"><div class="info-label">URL</div><div class="info-value">' + server.url + '</div></div>';
                    }

                    if (server.command) {
                        html += '<div class="info-item"><div class="info-label">Command</div><div class="info-value">' + server.command + '</div></div>';
                    }

                    if (server.last_error) {
                        html += '<div class="error-message"><strong>Last Error:</strong><br>' + server.last_error + '</div>';
                    }

                    document.getElementById('server-info').innerHTML = html;
                }
            } catch (error) {
                document.getElementById('server-info').innerHTML = '<div class="error-message">Failed to load server info</div>';
            }
        }

        // Load tools
        async function loadTools() {
            try {
                const response = await fetch('/api/servers/' + encodeURIComponent(serverName) + '/tools');
                const data = await response.json();

                if (data.tools && data.tools.length > 0) {
                    let html = '<div class="tools-list">';
                    data.tools.slice(0, 10).forEach(tool => {
                        html += '<div class="tool-item">' + tool.name;
                        if (tool.description) {
                            html += '<br><span style="color: #666; font-size: 0.9em;">' + tool.description.substring(0, 100) + '</span>';
                        }
                        html += '</div>';
                    });
                    if (data.tools.length > 10) {
                        html += '<div style="text-align: center; padding: 8px; color: #666; font-size: 0.85em;">... and ' + (data.tools.length - 10) + ' more</div>';
                    }
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
            } catch (error) {
                addMessage('agent', 'Error: Failed to communicate with AI agent. Make sure OPENAI_API_KEY is set.', true);
            } finally {
                document.getElementById('loading').classList.remove('active');
                document.getElementById('send-button').disabled = false;
            }
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

        // Event listeners
        document.getElementById('send-button').addEventListener('click', sendMessage);
        document.getElementById('chat-input').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') sendMessage();
        });

        // Load initial data
        loadServerInfo();
        loadTools();
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
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_KEY")
	}

	if apiKey == "" {
		responseData := map[string]interface{}{
			"error": "OpenAI API key not configured. Please set OPENAI_API_KEY environment variable.",
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

Your capabilities include:
1. Configuration Analysis and Updates - Analyze and fix server configurations
2. Log Analysis - Diagnose issues from logs and error messages
3. Documentation Analysis - Help understand and implement server features from GitHub
4. Installation Help - Guide users through server setup and installation
5. Testing Assistance - Help test server functionality and troubleshoot issues
6. Server Management - Start, stop, enable, disable servers
7. Tool Testing - Test individual tools with appropriate parameters
8. General Troubleshooting - Solve any MCP server-related problems

%s

=== Instructions ===
Please provide clear, actionable responses that:
- Directly address the user's questions
- Analyze server configuration and status when relevant
- Identify potential issues from error messages or status
- Provide specific steps or solutions when applicable
- Include relevant code examples or configuration snippets when helpful
- Suggest tool tests with example parameters if testing is needed
- Reference GitHub repository documentation when available
- Recommend next steps or follow-up actions

If the server has errors:
1. Explain what the error means
2. Identify the root cause based on configuration and status
3. Provide step-by-step fix instructions
4. Suggest verification steps after fix

If testing tools:
1. Explain what the tool does
2. Suggest appropriate test parameters
3. Explain expected results
4. Help interpret actual results`, serverContext)

		sessions.addMessage(sessionID, "system", systemPrompt)
	}

	// Add user message to history
	sessions.addMessage(sessionID, "user", req.Message)

	// Get all messages for OpenAI
	messages := sessions.getMessages(sessionID)

	// Call OpenAI API with full conversation history
	response, err := s.callOpenAI(apiKey, messages)
	if err != nil {
		responseData := map[string]interface{}{
			"error": fmt.Sprintf("AI request failed: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseData)
		return
	}

	// Add assistant response to history
	sessions.addMessage(sessionID, "assistant", response)

	responseData := map[string]interface{}{
		"response": response,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseData)
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
