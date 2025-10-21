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
    <title>MCPProxy - Server Group Assignment</title>
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
        .assignment-form {
            background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
            border-radius: 12px;
            padding: 30px;
            margin-bottom: 30px;
        }
        .form-group {
            margin-bottom: 20px;
        }
        label {
            display: block;
            margin-bottom: 8px;
            font-weight: 600;
            color: #333;
            font-size: 1.1em;
        }
        select {
            width: 100%;
            padding: 12px;
            border: 2px solid #ddd;
            border-radius: 8px;
            font-size: 16px;
            background: white;
            transition: all 0.3s;
        }
        select:focus {
            outline: none;
            border-color: #667eea;
            box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
        }
        .table-container {
            background: white;
            border-radius: 12px;
            overflow: hidden;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
            margin-bottom: 20px;
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
        }
        td {
            padding: 16px 12px;
            border-bottom: 1px solid #e9ecef;
        }
        tbody tr:hover {
            background: #f8f9fa;
        }
        .checkbox-cell {
            width: 50px;
            text-align: center;
        }
        input[type="checkbox"] {
            width: 20px;
            height: 20px;
            cursor: pointer;
            accent-color: #667eea;
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
        .server-name {
            font-weight: 600;
            color: #333;
        }
        .group-badge {
            display: inline-block;
            background: #e7f3ff;
            color: #0056b3;
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 0.8em;
            font-weight: 500;
            margin-right: 4px;
            margin-bottom: 4px;
        }
        .btn {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            padding: 14px 32px;
            border-radius: 8px;
            font-size: 1em;
            cursor: pointer;
            transition: all 0.3s;
            font-weight: 600;
            margin-right: 10px;
        }
        .btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 16px rgba(102, 126, 234, 0.4);
        }
        .btn:disabled {
            background: #ccc;
            cursor: not-allowed;
            transform: none;
        }
        .btn-secondary {
            background: linear-gradient(135deg, #6c757d 0%, #5a6268 100%);
        }
        .btn-danger {
            background: linear-gradient(135deg, #dc3545 0%, #c82333 100%);
        }
        .btn-container {
            display: flex;
            gap: 10px;
            margin-top: 20px;
        }
        .message {
            padding: 15px;
            margin: 15px 0;
            border-radius: 8px;
            font-weight: 500;
        }
        .success {
            background-color: #d4edda;
            color: #155724;
            border: 2px solid #c3e6cb;
        }
        .error {
            background-color: #f8d7da;
            color: #721c24;
            border: 2px solid #f5c6cb;
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
        .select-all-container {
            background: #f8f9fa;
            padding: 12px;
            border-radius: 8px;
            margin-bottom: 10px;
            display: flex;
            align-items: center;
            gap: 10px;
        }
        .select-all-container label {
            margin: 0;
            font-size: 1em;
            cursor: pointer;
        }
        .stats {
            color: #666;
            font-size: 0.9em;
            margin-left: auto;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🏷️ Server Group Assignment</h1>
            <a href="/" class="back-btn">← Dashboard</a>
        </div>

        <div class="content">
            <div id="message"></div>

            <div id="loading" class="loading">
                <div class="spinner"></div>
                <p>Loading data...</p>
            </div>

            <div id="content" style="display: none;">
                <div class="assignment-form">
                    <div class="form-group">
                        <label for="groupName">Select Group:</label>
                        <select id="groupName" name="groupName" onchange="updateGroupSelection()">
                            <option value="">-- Select a group to assign servers --</option>
                        </select>
                    </div>
                </div>

                <div class="select-all-container">
                    <input type="checkbox" id="selectAll" onchange="toggleSelectAll()">
                    <label for="selectAll">Select All Servers</label>
                    <span class="stats" id="selectionStats">0 servers selected</span>
                </div>

                <div class="table-container">
                    <table>
                        <thead>
                            <tr>
                                <th class="checkbox-cell">Select</th>
                                <th>Server Name</th>
                                <th>Status</th>
                                <th>Protocol</th>
                                <th>Current Groups</th>
                            </tr>
                        </thead>
                        <tbody id="servers-table"></tbody>
                    </table>
                </div>

                <div class="btn-container">
                    <button class="btn" onclick="assignServers()" id="assignBtn" disabled>
                        ✓ Assign Selected Servers to Group
                    </button>
                    <button class="btn btn-danger" onclick="removeFromGroup()" id="removeBtn" disabled>
                        ✗ Remove Selected from Group
                    </button>
                    <button class="btn btn-secondary" onclick="refreshAll()">
                        🔄 Refresh
                    </button>
                </div>
            </div>
        </div>
    </div>

    <script>
        let allServers = [];
        let allGroups = [];
        let currentAssignments = {};
        let selectedGroup = '';

        // Load on page load
        document.addEventListener('DOMContentLoaded', function() {
            refreshAll();
        });

        // Refresh all data
        async function refreshAll() {
            document.getElementById('loading').style.display = 'block';
            document.getElementById('content').style.display = 'none';

            try {
                await Promise.all([
                    loadServers(),
                    loadGroups(),
                    loadAssignments()
                ]);

                renderServersTable();
                updateButtonStates();

                document.getElementById('loading').style.display = 'none';
                document.getElementById('content').style.display = 'block';
            } catch (error) {
                showMessage('Error loading data: ' + error.message, 'error');
            }
        }

        // Load servers
        async function loadServers() {
            const response = await fetch('/api/servers/status');
            const data = await response.json();

            if (data.servers && Array.isArray(data.servers)) {
                allServers = data.servers.sort((a, b) => a.name.localeCompare(b.name));
            }
        }

        // Load groups
        async function loadGroups() {
            const response = await fetch('/api/groups');
            const data = await response.json();

            if (data.success && data.groups) {
                allGroups = data.groups;

                const select = document.getElementById('groupName');
                select.innerHTML = '<option value="">-- Select a group to assign servers --</option>';

                data.groups.forEach(group => {
                    const option = document.createElement('option');
                    option.value = group.name;
                    option.textContent = group.name;
                    select.appendChild(option);
                });
            }
        }

        // Load assignments
        async function loadAssignments() {
            const response = await fetch('/api/assignments');
            const data = await response.json();

            currentAssignments = {};

            if (data.success && data.assignments) {
                data.assignments.forEach(assignment => {
                    if (!currentAssignments[assignment.server_name]) {
                        currentAssignments[assignment.server_name] = [];
                    }
                    currentAssignments[assignment.server_name].push(assignment.group_name);
                });
            }
        }

        // Render servers table
        function renderServersTable() {
            const tbody = document.getElementById('servers-table');
            tbody.innerHTML = '';

            allServers.forEach(server => {
                const row = document.createElement('tr');

                // Checkbox cell
                const checkboxCell = document.createElement('td');
                checkboxCell.className = 'checkbox-cell';
                const checkbox = document.createElement('input');
                checkbox.type = 'checkbox';
                checkbox.id = 'server-' + server.name;
                checkbox.value = server.name;
                checkbox.onchange = updateButtonStates;
                checkboxCell.appendChild(checkbox);

                // Server name cell
                const nameCell = document.createElement('td');
                nameCell.innerHTML = '<span class="server-name">' + server.name + '</span><br>' +
                                   '<small>' + (server.url || server.command || '-') + '</small>';

                // Status cell
                const statusCell = document.createElement('td');
                const statusClass = getStatusClass(server.status);
                statusCell.innerHTML = '<span class="status-badge ' + statusClass + '">' + server.status + '</span>';

                // Protocol cell
                const protocolCell = document.createElement('td');
                protocolCell.textContent = server.protocol || '-';

                // Groups cell
                const groupsCell = document.createElement('td');
                const serverGroups = currentAssignments[server.name] || [];
                if (serverGroups.length > 0) {
                    groupsCell.innerHTML = serverGroups.map(g =>
                        '<span class="group-badge">' + g + '</span>'
                    ).join('');
                } else {
                    groupsCell.innerHTML = '<em style="color: #999;">No groups</em>';
                }

                row.appendChild(checkboxCell);
                row.appendChild(nameCell);
                row.appendChild(statusCell);
                row.appendChild(protocolCell);
                row.appendChild(groupsCell);

                tbody.appendChild(row);
            });
        }

        function getStatusClass(status) {
            const statusLower = (status || '').toLowerCase();
            if (statusLower === 'ready') return 'status-ready';
            if (statusLower === 'connecting' || statusLower === 'authenticating') return 'status-connecting';
            if (statusLower === 'error') return 'status-error';
            return 'status-disconnected';
        }

        // Update group selection
        function updateGroupSelection() {
            selectedGroup = document.getElementById('groupName').value;
            updateButtonStates();
        }

        // Toggle select all
        function toggleSelectAll() {
            const selectAll = document.getElementById('selectAll').checked;
            allServers.forEach(server => {
                const checkbox = document.getElementById('server-' + server.name);
                if (checkbox) {
                    checkbox.checked = selectAll;
                }
            });
            updateButtonStates();
        }

        // Update button states
        function updateButtonStates() {
            const selectedServers = getSelectedServers();
            const hasSelection = selectedServers.length > 0;
            const hasGroup = selectedGroup !== '';

            document.getElementById('assignBtn').disabled = !hasSelection || !hasGroup;
            document.getElementById('removeBtn').disabled = !hasSelection || !hasGroup;

            document.getElementById('selectionStats').textContent =
                selectedServers.length + ' server' + (selectedServers.length !== 1 ? 's' : '') + ' selected';
        }

        // Get selected servers
        function getSelectedServers() {
            return allServers
                .filter(server => {
                    const checkbox = document.getElementById('server-' + server.name);
                    return checkbox && checkbox.checked;
                })
                .map(server => server.name);
        }

        // Assign servers to group
        async function assignServers() {
            const selectedServers = getSelectedServers();

            if (selectedServers.length === 0) {
                showMessage('Please select at least one server', 'error');
                return;
            }

            if (!selectedGroup) {
                showMessage('Please select a group', 'error');
                return;
            }

            try {
                const results = await Promise.all(
                    selectedServers.map(serverName =>
                        fetch('/api/assign-server', {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({
                                server_name: serverName,
                                group_name: selectedGroup
                            })
                        }).then(r => r.json())
                    )
                );

                const failures = results.filter(r => !r.success);

                if (failures.length === 0) {
                    showMessage(
                        '✓ Successfully assigned ' + selectedServers.length +
                        ' server(s) to group "' + selectedGroup + '"',
                        'success'
                    );

                    // Clear selections
                    selectedServers.forEach(serverName => {
                        const checkbox = document.getElementById('server-' + serverName);
                        if (checkbox) checkbox.checked = false;
                    });
                    document.getElementById('selectAll').checked = false;

                    // Reload data
                    await loadAssignments();
                    renderServersTable();
                    updateButtonStates();
                } else {
                    showMessage('⚠ Some assignments failed: ' + failures.length + ' errors', 'error');
                }
            } catch (error) {
                showMessage('Error: ' + error.message, 'error');
            }
        }

        // Remove servers from group
        async function removeFromGroup() {
            const selectedServers = getSelectedServers();

            if (selectedServers.length === 0) {
                showMessage('Please select at least one server', 'error');
                return;
            }

            if (!selectedGroup) {
                showMessage('Please select a group', 'error');
                return;
            }

            if (!confirm('Remove ' + selectedServers.length + ' server(s) from group "' + selectedGroup + '"?')) {
                return;
            }

            try {
                const results = await Promise.all(
                    selectedServers.map(serverName =>
                        fetch('/api/remove-assignment', {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({
                                server_name: serverName,
                                group_name: selectedGroup
                            })
                        }).then(r => r.json())
                    )
                );

                const failures = results.filter(r => !r.success);

                if (failures.length === 0) {
                    showMessage(
                        '✓ Successfully removed ' + selectedServers.length +
                        ' server(s) from group "' + selectedGroup + '"',
                        'success'
                    );

                    // Clear selections
                    selectedServers.forEach(serverName => {
                        const checkbox = document.getElementById('server-' + serverName);
                        if (checkbox) checkbox.checked = false;
                    });
                    document.getElementById('selectAll').checked = false;

                    // Reload data
                    await loadAssignments();
                    renderServersTable();
                    updateButtonStates();
                } else {
                    showMessage('⚠ Some removals failed: ' + failures.length + ' errors', 'error');
                }
            } catch (error) {
                showMessage('Error: ' + error.message, 'error');
            }
        }

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
