package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// handleMemoryPage serves the memory editor page
func (s *Server) handleMemoryPage(w http.ResponseWriter, r *http.Request) {
	memoryPath := filepath.Join(s.config.DataDir, "memory.md")

	// Read current memory content
	content, err := os.ReadFile(memoryPath)
	if err != nil {
		// If file doesn't exist, create it with default content
		if os.IsNotExist(err) {
			defaultContent := `# Diagnostic Agent Memory

This file stores common problems, findings, and solutions discovered by the AI Diagnostic Agent.

## Common Problems and Solutions

### Example Entry Template
` + "```" + `
Problem: [description]
Cause: [root cause]
Solution: [fix applied]
Timestamp: [YYYY-MM-DD HH:MM:SS]
` + "```" + `

## Memory Entry Log

<!-- AI Agent will append new findings below this line -->
`
			content = []byte(defaultContent)
			if err := os.WriteFile(memoryPath, content, 0644); err != nil {
				http.Error(w, "Failed to create memory file: "+err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "Failed to read memory file: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Diagnostic Memory - MCPProxy</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            min-height: 100vh;
            padding: 20px;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 8px 32px rgba(0,0,0,0.1);
            overflow: hidden;
        }

        .header {
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
            padding: 30px;
            text-align: center;
        }

        .header h1 {
            font-size: 2em;
            margin-bottom: 10px;
        }

        .header p {
            opacity: 0.9;
            font-size: 1.1em;
        }

        .toolbar {
            display: flex;
            gap: 10px;
            padding: 20px;
            background: #f8f9fa;
            border-bottom: 1px solid #e0e0e0;
            flex-wrap: wrap;
        }

        .toolbar button {
            padding: 10px 20px;
            border: none;
            border-radius: 6px;
            cursor: pointer;
            font-size: 14px;
            font-weight: 500;
            transition: all 0.2s;
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .btn-primary {
            background: #667eea;
            color: white;
        }

        .btn-primary:hover {
            background: #5568d3;
            transform: translateY(-1px);
        }

        .btn-secondary {
            background: #6c757d;
            color: white;
        }

        .btn-secondary:hover {
            background: #5a6268;
        }

        .btn-success {
            background: #28a745;
            color: white;
        }

        .editor-container {
            padding: 20px;
        }

        #memory-editor {
            width: 100%%;
            min-height: 600px;
            font-family: 'Menlo', 'Monaco', 'Courier New', monospace;
            font-size: 14px;
            line-height: 1.6;
            padding: 15px;
            border: 1px solid #ddd;
            border-radius: 6px;
            resize: vertical;
            background: #f8f9fa;
        }

        #memory-editor:focus {
            outline: none;
            border-color: #667eea;
            box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
        }

        .preview-container {
            padding: 20px;
            display: none;
        }

        .preview-content {
            padding: 20px;
            border: 1px solid #ddd;
            border-radius: 6px;
            background: white;
            min-height: 600px;
        }

        .preview-content h1 {
            color: #333;
            margin-top: 20px;
            margin-bottom: 10px;
            padding-bottom: 10px;
            border-bottom: 2px solid #667eea;
        }

        .preview-content h2 {
            color: #555;
            margin-top: 20px;
            margin-bottom: 10px;
        }

        .preview-content h3 {
            color: #666;
            margin-top: 15px;
            margin-bottom: 8px;
        }

        .preview-content pre {
            background: #f8f9fa;
            padding: 15px;
            border-radius: 6px;
            overflow-x: auto;
            border: 1px solid #e0e0e0;
        }

        .preview-content code {
            background: #f8f9fa;
            padding: 2px 6px;
            border-radius: 3px;
            font-family: 'Menlo', 'Monaco', 'Courier New', monospace;
        }

        .preview-content ul, .preview-content ol {
            margin-left: 30px;
            margin-top: 10px;
            margin-bottom: 10px;
        }

        .preview-content li {
            margin: 5px 0;
        }

        .status-message {
            padding: 15px;
            margin: 20px;
            border-radius: 6px;
            display: none;
            align-items: center;
            gap: 10px;
        }

        .status-message.success {
            background: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
        }

        .status-message.error {
            background: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
        }

        .back-link {
            color: white;
            text-decoration: none;
            opacity: 0.9;
            display: inline-flex;
            align-items: center;
            gap: 5px;
            margin-bottom: 10px;
        }

        .back-link:hover {
            opacity: 1;
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <a href="/" class="back-link">← Back to Dashboard</a>
            <h1>🧠 Diagnostic Memory Editor</h1>
            <p>Persistent memory for AI Diagnostic Agent findings and solutions</p>
        </div>

        <div class="status-message" id="status-message"></div>

        <div class="toolbar">
            <button class="btn-primary" onclick="saveMemory()" id="save-btn">
                💾 Save Changes
            </button>
            <button class="btn-secondary" onclick="togglePreview()" id="preview-btn">
                👁️ Preview
            </button>
            <button class="btn-secondary" onclick="reloadMemory()">
                🔄 Reload
            </button>
            <button class="btn-secondary" onclick="window.location.href='/chat'">
                🤖 Open AI Diagnostic Agent
            </button>
        </div>

        <div class="editor-container" id="editor-container">
            <textarea id="memory-editor">%s</textarea>
        </div>

        <div class="preview-container" id="preview-container">
            <div class="preview-content" id="preview-content"></div>
        </div>
    </div>

    <script src="https://cdn.jsdelivr.net/npm/marked/marked.min.js"></script>
    <script>
        let isPreviewMode = false;
        let originalContent = document.getElementById('memory-editor').value;

        function showStatus(message, isError = false) {
            const statusEl = document.getElementById('status-message');
            statusEl.textContent = message;
            statusEl.className = 'status-message ' + (isError ? 'error' : 'success');
            statusEl.style.display = 'flex';

            setTimeout(() => {
                statusEl.style.display = 'none';
            }, 3000);
        }

        async function saveMemory() {
            const content = document.getElementById('memory-editor').value;
            const saveBtn = document.getElementById('save-btn');

            saveBtn.disabled = true;
            saveBtn.textContent = '⏳ Saving...';

            try {
                const response = await fetch('/api/memory', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ content })
                });

                const data = await response.json();

                if (data.success) {
                    showStatus('✅ Memory saved successfully!');
                    originalContent = content;
                } else {
                    throw new Error(data.message || 'Failed to save memory');
                }
            } catch (error) {
                showStatus('❌ Failed to save: ' + error.message, true);
            } finally {
                saveBtn.disabled = false;
                saveBtn.textContent = '💾 Save Changes';
            }
        }

        function togglePreview() {
            isPreviewMode = !isPreviewMode;
            const editorContainer = document.getElementById('editor-container');
            const previewContainer = document.getElementById('preview-container');
            const previewBtn = document.getElementById('preview-btn');

            if (isPreviewMode) {
                // Show preview
                const content = document.getElementById('memory-editor').value;
                const html = marked.parse(content);
                document.getElementById('preview-content').innerHTML = html;

                editorContainer.style.display = 'none';
                previewContainer.style.display = 'block';
                previewBtn.textContent = '✏️ Edit';
            } else {
                // Show editor
                editorContainer.style.display = 'block';
                previewContainer.style.display = 'none';
                previewBtn.textContent = '👁️ Preview';
            }
        }

        async function reloadMemory() {
            if (document.getElementById('memory-editor').value !== originalContent) {
                if (!confirm('You have unsaved changes. Are you sure you want to reload?')) {
                    return;
                }
            }

            try {
                const response = await fetch('/api/memory');
                const data = await response.json();

                if (data.success) {
                    document.getElementById('memory-editor').value = data.content;
                    originalContent = data.content;
                    showStatus('✅ Memory reloaded');
                } else {
                    throw new Error(data.message || 'Failed to reload memory');
                }
            } catch (error) {
                showStatus('❌ Failed to reload: ' + error.message, true);
            }
        }

        // Auto-save on Ctrl+S
        document.addEventListener('keydown', (e) => {
            if ((e.ctrlKey || e.metaKey) && e.key === 's') {
                e.preventDefault();
                saveMemory();
            }
        });

        // Warn before leaving with unsaved changes
        window.addEventListener('beforeunload', (e) => {
            if (document.getElementById('memory-editor').value !== originalContent) {
                e.preventDefault();
                e.returnValue = '';
            }
        });
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, html, string(content))
}

// handleMemoryAPI handles GET and POST requests for memory content
func (s *Server) handleMemoryAPI(w http.ResponseWriter, r *http.Request) {
	memoryPath := filepath.Join(s.config.DataDir, "memory.md")

	switch r.Method {
	case http.MethodGet:
		// Read memory content
		content, err := os.ReadFile(memoryPath)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Failed to read memory file: " + err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"content": string(content),
		})

	case http.MethodPost:
		// Save memory content
		var req struct {
			Content string `json:"content"`
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Failed to read request body: " + err.Error(),
			})
			return
		}

		if err := json.Unmarshal(body, &req); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Invalid JSON: " + err.Error(),
			})
			return
		}

		if err := os.WriteFile(memoryPath, []byte(req.Content), 0644); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Failed to write memory file: " + err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Memory saved successfully",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
