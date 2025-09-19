package server

import (
	"net/http"
)

// handleAssignmentWeb serves the server assignment web interface
func (s *Server) handleAssignmentWeb(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Server Group Assignment - MCPProxy</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 800px;
            margin: 0 auto;
            background: white;
            padding: 30px;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            margin-bottom: 30px;
            text-align: center;
        }
        .form-group {
            margin-bottom: 20px;
        }
        label {
            display: block;
            margin-bottom: 5px;
            font-weight: 600;
            color: #555;
        }
        select, input {
            width: 100%;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 5px;
            font-size: 16px;
        }
        .btn {
            background-color: #007bff;
            color: white;
            padding: 12px 24px;
            border: none;
            border-radius: 5px;
            cursor: pointer;
            font-size: 16px;
            margin-right: 10px;
        }
        .btn:hover {
            background-color: #0056b3;
        }
        .btn-secondary {
            background-color: #6c757d;
        }
        .btn-secondary:hover {
            background-color: #545b62;
        }
        .assignments {
            margin-top: 30px;
        }
        .assignment-item {
            background: #f8f9fa;
            padding: 15px;
            margin-bottom: 10px;
            border-radius: 5px;
            border-left: 4px solid #007bff;
        }
        .message {
            padding: 10px;
            margin: 10px 0;
            border-radius: 5px;
        }
        .success {
            background-color: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
        }
        .error {
            background-color: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🏷️ Server Group Assignment</h1>
        
        <div id="message"></div>
        
        <form id="assignmentForm">
            <div class="form-group">
                <label for="serverName">Server Name:</label>
                <input type="text" id="serverName" name="serverName" placeholder="Enter server name" required>
            </div>
            
            <div class="form-group">
                <label for="groupName">Group:</label>
                <select id="groupName" name="groupName" required>
                    <option value="">Select a group...</option>
                </select>
            </div>
            
            <button type="submit" class="btn">Assign Server</button>
            <button type="button" class="btn btn-secondary" onclick="loadAssignments()">Refresh</button>
        </form>
        
        <div class="assignments">
            <h2>Current Assignments</h2>
            <div id="assignmentsList">Loading...</div>
        </div>
    </div>

    <script>
        // Load groups on page load
        document.addEventListener('DOMContentLoaded', function() {
            loadGroups();
            loadAssignments();
        });

        // Load available groups
        async function loadGroups() {
            try {
                const response = await fetch('/api/groups');
                const data = await response.json();
                
                const select = document.getElementById('groupName');
                select.innerHTML = '<option value="">Select a group...</option>';
                
                if (data.success && data.groups) {
                    data.groups.forEach(group => {
                        const option = document.createElement('option');
                        option.value = group.name;
                        option.textContent = group.name;
                        select.appendChild(option);
                    });
                }
            } catch (error) {
                showMessage('Error loading groups: ' + error.message, 'error');
            }
        }

        // Load current assignments
        async function loadAssignments() {
            try {
                const response = await fetch('/api/assignments');
                const data = await response.json();
                
                const container = document.getElementById('assignmentsList');
                
                if (data.success && data.assignments && data.assignments.length > 0) {
                    container.innerHTML = data.assignments.map(assignment => 
                        '<div class="assignment-item">' +
                        '<strong>' + assignment.server_name + '</strong> → ' + assignment.group_name +
                        '</div>'
                    ).join('');
                } else {
                    container.innerHTML = '<p>No server assignments found.</p>';
                }
            } catch (error) {
                document.getElementById('assignmentsList').innerHTML = '<p>Error loading assignments.</p>';
            }
        }

        // Handle form submission
        document.getElementById('assignmentForm').addEventListener('submit', async function(e) {
            e.preventDefault();
            
            const serverName = document.getElementById('serverName').value;
            const groupName = document.getElementById('groupName').value;
            
            if (!serverName || !groupName) {
                showMessage('Please fill in all fields', 'error');
                return;
            }
            
            try {
                const response = await fetch('/api/assign-server', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        server_name: serverName,
                        group_name: groupName
                    })
                });
                
                const data = await response.json();
                
                if (data.success) {
                    showMessage(data.message, 'success');
                    document.getElementById('serverName').value = '';
                    document.getElementById('groupName').value = '';
                    loadAssignments();
                } else {
                    showMessage(data.error || 'Assignment failed', 'error');
                }
            } catch (error) {
                showMessage('Error: ' + error.message, 'error');
            }
        });

        // Show message
        function showMessage(text, type) {
            const messageDiv = document.getElementById('message');
            messageDiv.innerHTML = '<div class="message ' + type + '">' + text + '</div>';
            setTimeout(() => {
                messageDiv.innerHTML = '';
            }, 5000);
        }
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(tmpl))
}
