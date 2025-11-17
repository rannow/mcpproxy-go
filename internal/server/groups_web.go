package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"os"

	"mcpproxy-go/internal/events"

	"go.uber.org/zap"
)

// handleGroupsWeb serves the group management web interface
func (s *Server) handleGroupsWeb(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MCPProxy - Group Management</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            padding: 30px 20px;
            color: white;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .header-content {
            max-width: 1400px;
            margin: 0 auto;
        }
        h1 {
            margin: 0 0 10px 0;
            font-size: 32px;
            font-weight: 600;
        }
        .subtitle {
            opacity: 0.9;
            font-size: 14px;
        }
        .container {
            max-width: 1400px;
            margin: 20px auto;
            padding: 0 20px;
        }
        .summary-cards {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .summary-card {
            background: white;
            border-radius: 12px;
            padding: 24px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
            transition: transform 0.2s;
        }
        .summary-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        }
        .summary-card h3 {
            margin: 0 0 8px 0;
            font-size: 14px;
            color: #666;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .summary-card .value {
            font-size: 32px;
            font-weight: bold;
            color: #333;
        }
        .table-container {
            background: white;
            border-radius: 12px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
            overflow: hidden;
            margin-bottom: 30px;
        }
        .table-header {
            padding: 20px;
            border-bottom: 2px solid #f0f0f0;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .table-header h2 {
            margin: 0;
            font-size: 20px;
            color: #333;
        }
        table {
            width: 100%;
            border-collapse: collapse;
        }
        thead {
            background: #f8f9fa;
        }
        th {
            padding: 16px;
            text-align: left;
            font-weight: 600;
            color: #333;
            border-bottom: 2px solid #e0e0e0;
            white-space: nowrap;
        }
        th.sortable {
            cursor: pointer;
            user-select: none;
        }
        th.sortable:hover {
            background: #e9ecef;
        }
        .filter-row th {
            padding: 8px 16px;
            background: white;
            border-bottom: 1px solid #e0e0e0;
        }
        .filter-input {
            width: 100%;
            padding: 6px 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 13px;
        }
        .filter-input:focus {
            outline: none;
            border-color: #667eea;
            box-shadow: 0 0 0 2px rgba(102, 126, 234, 0.1);
        }
        td {
            padding: 16px;
            border-bottom: 1px solid #f0f0f0;
        }
        tbody tr:hover {
            background: #f8f9fa;
        }
        .group-icon-cell {
            font-size: 24px;
            text-align: center;
            width: 60px;
        }
        .group-color-badge {
            width: 24px;
            height: 24px;
            border-radius: 50%;
            display: inline-block;
            border: 2px solid #ddd;
            vertical-align: middle;
        }
        .group-name-cell {
            font-weight: 600;
            color: #333;
        }
        .server-count {
            display: inline-block;
            padding: 4px 12px;
            background: #e9ecef;
            border-radius: 12px;
            font-size: 13px;
            font-weight: 500;
            color: #495057;
        }
        .server-count.has-servers {
            background: #d4edda;
            color: #155724;
        }
        .actions-cell {
            white-space: nowrap;
        }
        .btn {
            padding: 8px 16px;
            border: none;
            border-radius: 6px;
            cursor: pointer;
            font-size: 14px;
            font-weight: 500;
            transition: all 0.2s;
            display: inline-block;
        }
        .btn-primary {
            background: #667eea;
            color: white;
        }
        .btn-primary:hover {
            background: #5568d3;
            transform: translateY(-1px);
            box-shadow: 0 2px 8px rgba(102, 126, 234, 0.3);
        }
        .btn-secondary {
            background: #6c757d;
            color: white;
        }
        .btn-secondary:hover {
            background: #545b62;
        }
        .btn-danger {
            background: #dc3545;
            color: white;
        }
        .btn-danger:hover {
            background: #c82333;
        }
        .btn-success {
            background: #28a745;
            color: white;
        }
        .btn-success:hover {
            background: #218838;
            transform: translateY(-1px);
            box-shadow: 0 2px 8px rgba(40, 167, 69, 0.3);
        }
        .btn-warning {
            background: #ffc107;
            color: #212529;
        }
        .btn-warning:hover {
            background: #e0a800;
            transform: translateY(-1px);
            box-shadow: 0 2px 8px rgba(255, 193, 7, 0.3);
        }
        .btn-sm {
            padding: 6px 12px;
            font-size: 13px;
        }
        .btn:disabled {
            opacity: 0.5;
            cursor: not-allowed;
        }
        .btn:disabled:hover {
            transform: none;
            box-shadow: none;
        }
        .btn-create {
            background: #667eea;
            color: white;
            padding: 12px 24px;
            font-size: 15px;
        }
        .btn-create:hover {
            background: #5568d3;
            transform: translateY(-1px);
            box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
        }
        .modal {
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(0,0,0,0.5);
            z-index: 1000;
        }
        .modal-content {
            position: absolute;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            background: white;
            padding: 30px;
            border-radius: 8px;
            width: 90%;
            max-width: 500px;
        }
        .form-group {
            margin-bottom: 20px;
        }
        .form-group label {
            display: block;
            margin-bottom: 5px;
            font-weight: bold;
        }
        .form-group input {
            width: 100%;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 16px;
        }
        .color-picker {
            display: flex;
            gap: 10px;
            flex-wrap: wrap;
        }
        .color-option {
            width: 40px;
            height: 40px;
            border-radius: 50%;
            cursor: pointer;
            border: 3px solid transparent;
            transition: border-color 0.2s;
        }
        .color-option.selected {
            border-color: #333;
        }
        .icon-picker {
            display: flex;
            gap: 10px;
            flex-wrap: wrap;
            max-height: 200px;
            overflow-y: auto;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
        }
        .icon-option {
            width: 40px;
            height: 40px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 24px;
            cursor: pointer;
            border: 2px solid transparent;
            border-radius: 4px;
            transition: all 0.2s;
        }
        .icon-option:hover {
            background-color: #f0f0f0;
        }
        .icon-option.selected {
            border-color: #007bff;
            background-color: #e6f3ff;
        }
        .unused-icons {
            margin-top: 30px;
            padding: 20px;
            background: #f8f9fa;
            border-radius: 8px;
        }
        .unused-icons h3 {
            margin-bottom: 15px;
            color: #333;
        }
        .modal-actions {
            display: flex;
            gap: 10px;
            justify-content: flex-end;
            margin-top: 20px;
        }
    </style>
</head>
<body>
    <div class="header">
        <div class="header-content">
            <div style="margin-bottom: 10px;">
                <a href="/" style="color: rgba(255,255,255,0.9); text-decoration: none; font-size: 14px; display: inline-flex; align-items: center; gap: 5px;">
                    ← Back to Dashboard
                </a>
            </div>
            <h1>🏷️ Server Group Management</h1>
            <div class="subtitle">Organize and manage your MCP server groups</div>
        </div>
    </div>

    <div class="container">
        <div class="summary-cards">
            <div class="summary-card">
                <h3>Total Groups</h3>
                <div class="value" id="totalGroups">0</div>
            </div>
            <div class="summary-card">
                <h3>Groups with Servers</h3>
                <div class="value" id="groupsWithServers">0</div>
            </div>
            <div class="summary-card">
                <h3>Empty Groups</h3>
                <div class="value" id="emptyGroups">0</div>
            </div>
            <div class="summary-card">
                <h3>Available Icons</h3>
                <div class="value" id="availableIcons">40</div>
            </div>
        </div>

        <div class="table-container">
            <div class="table-header">
                <h2>Groups</h2>
                <button class="btn btn-create" onclick="openCreateModal()">➕ Create New Group</button>
            </div>
            <table>
                <thead>
                    <tr>
                        <th class="sortable" data-sort="icon">Icon</th>
                        <th class="sortable" data-sort="name">Name</th>
                        <th class="sortable" data-sort="servers">Servers</th>
                        <th class="sortable" data-sort="description">Description</th>
                        <th>Actions</th>
                    </tr>
                    <tr class="filter-row">
                        <th><input type="text" class="filter-input" id="filter-icon" placeholder="🔍" onkeyup="applyFilters()"></th>
                        <th><input type="text" class="filter-input" id="filter-name" placeholder="Filter..." onkeyup="applyFilters()"></th>
                        <th><input type="text" class="filter-input" id="filter-servers" placeholder="Filter..." onkeyup="applyFilters()"></th>
                        <th><input type="text" class="filter-input" id="filter-description" placeholder="Filter..." onkeyup="applyFilters()"></th>
                        <th></th>
                    </tr>
                </thead>
                <tbody id="groupsTable">
                    <!-- Groups will be loaded here -->
                </tbody>
            </table>
        </div>

        <div class="unused-icons">
            <h3>📦 Available Icons</h3>
            <div class="icon-picker" id="unusedIconsList">
                <!-- Unused icons will be shown here -->
            </div>
        </div>
    </div>

    <!-- Create/Edit Group Modal -->
    <div class="modal" id="groupModal">
        <div class="modal-content">
            <h3 id="modalTitle">Create New Group</h3>
            <form id="groupForm">
                <div class="form-group">
                    <label for="groupName">Group Name:</label>
                    <input type="text" id="groupName" name="name" required>
                </div>
                <div class="form-group">
                    <label>Icon:</label>
                    <div class="icon-picker" id="iconPicker">
                        <!-- Icon options will be generated -->
                    </div>
                </div>
                <div class="form-group">
                    <label>Color:</label>
                    <div class="color-picker" id="colorPicker">
                        <!-- Color options will be generated -->
                    </div>
                </div>
                <div class="modal-actions">
                    <button type="button" class="btn btn-secondary" onclick="closeModal()">Cancel</button>
                    <button type="submit" class="btn btn-primary">Save Group</button>
                </div>
            </form>
        </div>
    </div>

    <script>
        const colors = [
            { name: 'Blue', code: '#007bff' },
            { name: 'Green', code: '#28a745' },
            { name: 'Red', code: '#dc3545' },
            { name: 'Orange', code: '#fd7e14' },
            { name: 'Purple', code: '#6f42c1' },
            { name: 'Pink', code: '#e83e8c' },
            { name: 'Teal', code: '#20c997' },
            { name: 'Yellow', code: '#ffc107' },
            { name: 'Indigo', code: '#6610f2' },
            { name: 'Cyan', code: '#17a2b8' }
        ];

        const icons = [
            "🌐", "🔧", "🧪", "🗄️", "☁️", "🎯", "💼", "🔔", "🏠", "🖥️",
            "📊", "🔒", "⚡", "🎨", "📱", "🌟", "🔍", "💾", "🚀", "📁",
            "🔗", "⚙️", "📝", "🎭", "🌈", "🔐", "📡", "🎮", "🏗️", "🔬",
            "📈", "🌍", "🎪", "🔊", "📸", "🎥", "📚", "💡", "🛠️", "🎁"
        ];

        let groups = [];
        let serverAssignments = {};
        let editingGroup = null;
        let selectedColor = colors[0];
        let selectedIcon = icons[0];
        let sortColumn = 'name';
        let sortDirection = 'asc';

        // Initialize page
        document.addEventListener('DOMContentLoaded', function() {
            loadGroups();
            initIconPicker();
            initColorPicker();
            initSorting();
        });

        // Load groups from server
        function loadGroups() {
            fetch('/api/groups')
                .then(response => response.json())
                .then(data => {
                    groups = data.groups || [];
                    // Build server assignments map
                    serverAssignments = {};
                    (data.assignments || []).forEach(a => {
                        if (!serverAssignments[a.group_name]) {
                            serverAssignments[a.group_name] = [];
                        }
                        serverAssignments[a.group_name].push(a.server_name);
                    });
                    renderGroups();
                    updateSummaryCards();
                })
                .catch(error => {
                    console.error('Failed to load groups:', error);
                    renderGroups(); // Render empty state
                });
        }

        // Update summary cards
        function updateSummaryCards() {
            const totalGroups = groups.length;
            let groupsWithServers = 0;
            let emptyGroups = 0;

            groups.forEach(group => {
                const servers = serverAssignments[group.name] || [];
                if (servers.length > 0) {
                    groupsWithServers++;
                } else {
                    emptyGroups++;
                }
            });

            const usedIcons = new Set(groups.map(g => g.icon_emoji).filter(Boolean));
            const availableIcons = icons.length - usedIcons.size;

            document.getElementById('totalGroups').textContent = totalGroups;
            document.getElementById('groupsWithServers').textContent = groupsWithServers;
            document.getElementById('emptyGroups').textContent = emptyGroups;
            document.getElementById('availableIcons').textContent = availableIcons;
        }

        // Initialize sorting
        function initSorting() {
            document.querySelectorAll('th.sortable').forEach(th => {
                th.addEventListener('click', () => {
                    const column = th.dataset.sort;
                    if (sortColumn === column) {
                        sortDirection = sortDirection === 'asc' ? 'desc' : 'asc';
                    } else {
                        sortColumn = column;
                        sortDirection = 'asc';
                    }
                    renderGroups();
                });
            });
        }

        // Sort groups
        function sortGroups(groupsToSort) {
            return groupsToSort.sort((a, b) => {
                let aVal, bVal;

                switch(sortColumn) {
                    case 'icon':
                        aVal = a.icon_emoji || '';
                        bVal = b.icon_emoji || '';
                        break;
                    case 'name':
                        aVal = a.name || '';
                        bVal = b.name || '';
                        break;
                    case 'servers':
                        aVal = (serverAssignments[a.name] || []).length;
                        bVal = (serverAssignments[b.name] || []).length;
                        break;
                    case 'description':
                        aVal = a.description || '';
                        bVal = b.description || '';
                        break;
                    default:
                        aVal = a.name || '';
                        bVal = b.name || '';
                }

                if (sortColumn === 'servers') {
                    return sortDirection === 'asc' ? aVal - bVal : bVal - aVal;
                }

                const compareResult = aVal.toString().localeCompare(bVal.toString());
                return sortDirection === 'asc' ? compareResult : -compareResult;
            });
        }

        // Apply filters
        function applyFilters() {
            const filters = {
                icon: document.getElementById('filter-icon').value.toLowerCase(),
                name: document.getElementById('filter-name').value.toLowerCase(),
                servers: document.getElementById('filter-servers').value.toLowerCase(),
                description: document.getElementById('filter-description').value.toLowerCase()
            };

            const rows = document.querySelectorAll('#groupsTable tr');
            rows.forEach(row => {
                const cells = row.cells;
                if (!cells) return;

                const icon = cells[0]?.textContent.toLowerCase() || '';
                const name = cells[1]?.textContent.toLowerCase() || '';
                const servers = cells[2]?.textContent.toLowerCase() || '';
                const description = cells[3]?.textContent.toLowerCase() || '';

                const match = (
                    icon.includes(filters.icon) &&
                    name.includes(filters.name) &&
                    servers.includes(filters.servers) &&
                    description.includes(filters.description)
                );

                row.style.display = match ? '' : 'none';
            });
        }

        // Helper function to escape strings for use in HTML attributes
        function escapeHtml(str) {
            const div = document.createElement('div');
            div.textContent = str;
            return div.innerHTML;
        }

        // Helper function to escape strings for use in JavaScript
        function escapeJs(str) {
            return str.replace(/\\/g, '\\\\')
                      .replace(/'/g, "\\'")
                      .replace(/"/g, '\\"')
                      .replace(/\n/g, '\\n')
                      .replace(/\r/g, '\\r');
        }

        // Render groups table
        function renderGroups() {
            const tbody = document.getElementById('groupsTable');
            tbody.innerHTML = '';

            if (groups.length === 0) {
                tbody.innerHTML = '<tr><td colspan="5" style="text-align:center; padding:40px; color:#666;">No groups yet. Click "Create New Group" to get started.</td></tr>';
                return;
            }

            const sortedGroups = sortGroups([...groups]);

            sortedGroups.forEach(group => {
                const icon = group.icon_emoji || '📁';
                const servers = serverAssignments[group.name] || [];
                const serverCount = servers.length;
                const description = group.description || '';

                // Escape group name for safe use in onclick handlers
                const escapedGroupName = escapeJs(group.name);

                const row = document.createElement('tr');

                // Create buttons with proper event listeners instead of inline onclick
                const enableBtn = document.createElement('button');
                enableBtn.className = 'btn btn-success btn-sm';
                enableBtn.textContent = '✓ Enable All';
                enableBtn.title = 'Enable all servers in this group';
                enableBtn.disabled = serverCount === 0;
                if (serverCount > 0) {
                    enableBtn.addEventListener('click', () => {
                        console.log('Enable button clicked for group:', group.name, 'Server count:', serverCount);
                        toggleGroupServers(group.name, true);
                    });
                }

                const disableBtn = document.createElement('button');
                disableBtn.className = 'btn btn-warning btn-sm';
                disableBtn.textContent = '✗ Disable All';
                disableBtn.title = 'Disable all servers in this group';
                disableBtn.disabled = serverCount === 0;
                if (serverCount > 0) {
                    disableBtn.addEventListener('click', () => {
                        console.log('Disable button clicked for group:', group.name, 'Server count:', serverCount);
                        toggleGroupServers(group.name, false);
                    });
                }

                const editBtn = document.createElement('button');
                editBtn.className = 'btn btn-primary btn-sm';
                editBtn.textContent = 'Edit';
                editBtn.addEventListener('click', () => editGroup(group.name));

                const deleteBtn = document.createElement('button');
                deleteBtn.className = 'btn btn-danger btn-sm';
                deleteBtn.textContent = 'Delete';
                deleteBtn.addEventListener('click', () => deleteGroup(group.name));

                // Build the row structure
                row.innerHTML = ` + "`" + `
                    <td class="group-icon-cell">${escapeHtml(icon)}</td>
                    <td class="group-name-cell">${escapeHtml(group.name)}</td>
                    <td><span class="server-count ${serverCount > 0 ? 'has-servers' : ''}">${serverCount}</span></td>
                    <td>${escapeHtml(description)}</td>
                    <td class="actions-cell"></td>
                ` + "`" + `;

                // Append buttons to actions cell
                const actionsCell = row.querySelector('.actions-cell');
                actionsCell.appendChild(enableBtn);
                actionsCell.appendChild(disableBtn);
                actionsCell.appendChild(editBtn);
                actionsCell.appendChild(deleteBtn);

                tbody.appendChild(row);
            });

            // Update unused icons list
            renderUnusedIcons();
        }

        // Initialize icon picker
        function initIconPicker() {
            const picker = document.getElementById('iconPicker');
            picker.innerHTML = '';

            icons.forEach((icon, index) => {
                const option = document.createElement('div');
                option.className = 'icon-option';
                option.textContent = icon;
                option.title = icon;
                option.onclick = () => selectIcon(icon, option);

                if (index === 0) {
                    option.classList.add('selected');
                }

                picker.appendChild(option);
            });
        }

        // Initialize color picker
        function initColorPicker() {
            const picker = document.getElementById('colorPicker');
            picker.innerHTML = '';

            colors.forEach((color, index) => {
                const option = document.createElement('div');
                option.className = 'color-option';
                option.style.backgroundColor = color.code;
                option.title = color.name;
                option.onclick = () => selectColor(color, option);

                if (index === 0) {
                    option.classList.add('selected');
                }

                picker.appendChild(option);
            });
        }

        // Render unused icons
        function renderUnusedIcons() {
            const usedIcons = new Set(groups.map(g => g.icon_emoji).filter(Boolean));
            const unusedIcons = icons.filter(icon => !usedIcons.has(icon));

            const container = document.getElementById('unusedIconsList');
            container.innerHTML = '';

            if (unusedIcons.length === 0) {
                container.innerHTML = '<div style="color: #666; font-style: italic;">All icons are in use</div>';
                return;
            }

            unusedIcons.forEach(icon => {
                const option = document.createElement('div');
                option.className = 'icon-option';
                option.textContent = icon;
                option.title = icon + ' (available)';
                option.style.cursor = 'default';
                option.style.opacity = '0.6';
                container.appendChild(option);
            });
        }

        // Select icon
        function selectIcon(icon, element) {
            document.querySelectorAll('#iconPicker .icon-option').forEach(el => el.classList.remove('selected'));
            element.classList.add('selected');
            selectedIcon = icon;
        }

        // Select color
        function selectColor(color, element) {
            document.querySelectorAll('.color-option').forEach(el => el.classList.remove('selected'));
            element.classList.add('selected');
            selectedColor = color;
        }

        // Open create modal
        function openCreateModal() {
            editingGroup = null;
            document.getElementById('modalTitle').textContent = 'Create New Group';
            document.getElementById('groupName').value = '';
            selectedIcon = icons[0];
            selectedColor = colors[0];

            // Reset icon selection
            document.querySelectorAll('#iconPicker .icon-option').forEach((el, i) => {
                el.classList.toggle('selected', i === 0);
            });

            // Reset color selection
            document.querySelectorAll('.color-option').forEach((el, i) => {
                el.classList.toggle('selected', i === 0);
            });

            document.getElementById('groupModal').style.display = 'block';
        }

        // Edit group
        function editGroup(groupName) {
            const group = groups.find(g => g.name === groupName);
            if (!group) return;

            editingGroup = group;
            document.getElementById('modalTitle').textContent = 'Edit Group';
            document.getElementById('groupName').value = group.name;

            // Select the group's icon
            const iconIndex = icons.indexOf(group.icon_emoji);
            if (iconIndex >= 0) {
                selectedIcon = icons[iconIndex];
                document.querySelectorAll('#iconPicker .icon-option').forEach((el, i) => {
                    el.classList.toggle('selected', i === iconIndex);
                });
            } else {
                // Default to first icon if not found
                selectedIcon = icons[0];
                document.querySelector('#iconPicker .icon-option').classList.add('selected');
            }

            // Select the group's color
            const colorIndex = colors.findIndex(c => c.code === group.color);
            if (colorIndex >= 0) {
                selectedColor = colors[colorIndex];
                document.querySelectorAll('.color-option').forEach((el, i) => {
                    el.classList.toggle('selected', i === colorIndex);
                });
            }

            document.getElementById('groupModal').style.display = 'block';
        }

        // Delete group
        function deleteGroup(groupName) {
            if (!confirm(` + "`" + `Are you sure you want to delete the group "${groupName}"?` + "`" + `)) return;

            fetch(` + "`" + `/api/groups/${encodeURIComponent(groupName)}` + "`" + `, {
                method: 'DELETE'
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    loadGroups();
                } else {
                    alert('Failed to delete group: ' + (data.error || 'Unknown error'));
                }
            })
            .catch(error => {
                console.error('Failed to delete group:', error);
                alert('Failed to delete group');
            });
        }

        // Toggle all servers in a group (enable or disable)
        function toggleGroupServers(groupName, enabled) {
            const action = enabled ? 'enable' : 'disable';
            if (!confirm(` + "`" + `Are you sure you want to ${action} all servers in group "${groupName}"?` + "`" + `)) return;

            fetch('/api/toggle-group-servers', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    group_name: groupName,
                    enabled: enabled
                })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    // Show success message with updated count
                    const message = data.message || ` + "`" + `${data.updated || 0} servers ${action}d successfully` + "`" + `;
                    alert(message);
                    // Reload the page to show updated server states
                    setTimeout(() => window.location.reload(), 100);
                } else {
                    alert('Failed to ' + action + ' servers: ' + (data.error || 'Unknown error'));
                }
            })
            .catch(error => {
                console.error('Failed to toggle group servers:', error);
                alert('Failed to ' + action + ' servers');
            });
        }

        // Close modal
        function closeModal() {
            document.getElementById('groupModal').style.display = 'none';
        }

        // Handle form submission
        document.getElementById('groupForm').addEventListener('submit', function(e) {
            e.preventDefault();

            const name = document.getElementById('groupName').value.trim();
            if (!name) return;

            const groupData = {
                name: name,
                icon_emoji: selectedIcon,
                color: selectedColor.code
            };

            const url = editingGroup ? ` + "`" + `/api/groups/${encodeURIComponent(editingGroup.name)}` + "`" + ` : '/api/groups';
            const method = editingGroup ? 'PUT' : 'POST';

            fetch(url, {
                method: method,
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(groupData)
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    closeModal();
                    loadGroups(); // This will update both table and summary cards
                } else {
                    alert('Failed to save group: ' + (data.error || 'Unknown error'));
                }
            })
            .catch(error => {
                console.error('Failed to save group:', error);
                alert('Failed to save group');
            });
        });

        // Close modal when clicking outside
        document.getElementById('groupModal').addEventListener('click', function(e) {
            if (e.target === this) {
                closeModal();
            }
        });
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// handleGroupsAPI handles the groups API endpoints
func (s *Server) handleGroupsAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.handleGetGroups(w, r)
	case "POST":
		s.handleCreateGroup(w, r)
	case "PUT":
		s.handleUpdateGroup(w, r)
	case "DELETE":
		s.handleDeleteGroup(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetGroups returns all groups and server assignments
func (s *Server) handleGetGroups(w http.ResponseWriter, r *http.Request) {

	allGroups := s.getGroups()
	groupList := make([]*Group, 0, len(allGroups))
	for _, group := range allGroups {
		groupList = append(groupList, group)
	}

	// Get server assignments
	assignmentsMutex.RLock()
	assignments := make([]map[string]interface{}, 0, len(serverGroupAssignments))
	for serverName, groupName := range serverGroupAssignments {
		assignments = append(assignments, map[string]interface{}{
			"server_name": serverName,
			"group_name":  groupName,
		})
	}
	assignmentsMutex.RUnlock()

	response := map[string]interface{}{
		"success":     true,
		"groups":      groupList,
		"assignments": assignments,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCreateGroup creates a new group
func (s *Server) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	var groupData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&groupData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	name, ok := groupData["name"].(string)
	if !ok || strings.TrimSpace(name) == "" {
		http.Error(w, "Group name is required", http.StatusBadRequest)
		return
	}

	color, ok := groupData["color"].(string)
	if !ok || strings.TrimSpace(color) == "" {
		color = "#007bff" // Default color
	}

	icon, _ := groupData["icon_emoji"].(string)
	if strings.TrimSpace(icon) == "" {
		icon = "📁" // Default icon
	}

	allGroups := s.getGroups()

	// Check if group already exists
	if _, exists := allGroups[name]; exists {
		response := map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Group '%s' already exists", name),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Generate next available ID for new group
	nextID := s.getNextGroupID()

	// Create new group with ID
	s.setGroup(name, &Group{
		ID:    nextID,
		Name:  name,
		Color: color,
		Icon:  icon,
	})

	s.logger.Info("Creating group", zap.String("name", name), zap.String("color", color), zap.String("icon", icon))

	// Save configuration to persist groups
	if err := s.SaveConfiguration(); err != nil {
		s.logger.Error("Failed to save configuration after creating group", zap.Error(err))
	}

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Group '%s' created successfully", name),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleUpdateGroup updates an existing group
func (s *Server) handleUpdateGroup(w http.ResponseWriter, r *http.Request) {
	// Extract group name from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/groups/")
	oldName := strings.Split(path, "/")[0]

	var groupData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&groupData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	newName, ok := groupData["name"].(string)
	if !ok || strings.TrimSpace(newName) == "" {
		http.Error(w, "Group name is required", http.StatusBadRequest)
		return
	}

	color, ok := groupData["color"].(string)
	if !ok || strings.TrimSpace(color) == "" {
		color = "#007bff" // Default color
	}

	icon, _ := groupData["icon_emoji"].(string)
	if strings.TrimSpace(icon) == "" {
		icon = "📁" // Default icon
	}

	// Update group with proper mutex handling to avoid deadlock
	var updateResult struct {
		success      bool
		errorMsg     string
		preservedID  int
	}

	func() {
		groupsMutex.Lock()
		defer groupsMutex.Unlock()

		// Check if old group exists and preserve its ID
		oldGroup, exists := groups[oldName]
		if !exists {
			updateResult.success = false
			updateResult.errorMsg = fmt.Sprintf("Group '%s' not found", oldName)
			return
		}

		// Preserve the ID and Description from the old group
		preservedID := oldGroup.ID
		preservedDescription := oldGroup.Description
		if description, ok := groupData["description"].(string); ok && strings.TrimSpace(description) != "" {
			preservedDescription = description
		}

		// If name changed, check if new name already exists
		if oldName != newName {
			if _, exists := groups[newName]; exists {
				updateResult.success = false
				updateResult.errorMsg = fmt.Sprintf("Group '%s' already exists", newName)
				return
			}
			// Remove old entry
			delete(groups, oldName)
		}

		// Update/create group with preserved ID and description
		groups[newName] = &Group{
			ID:          preservedID,
			Name:        newName,
			Description: preservedDescription,
			Color:       color,
			Icon:        icon,
		}
		updateResult.success = true
		updateResult.preservedID = preservedID
	}()

	// Handle error cases
	if !updateResult.success {
		response := map[string]interface{}{
			"success": false,
			"error":   updateResult.errorMsg,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	s.logger.Info("Updating group",
		zap.String("old_name", oldName),
		zap.String("new_name", newName),
		zap.Int("id", updateResult.preservedID),
		zap.String("color", color),
		zap.String("icon", icon))

	// Save configuration to persist groups (mutex is now released)
	if err := s.SaveConfiguration(); err != nil {
		s.logger.Error("Failed to save configuration after updating group", zap.Error(err))
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Group updated successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getNextGroupID returns the next available group ID (thread-safe)
func (s *Server) getNextGroupID() int {
	groupsMutex.RLock()
	defer groupsMutex.RUnlock()

	maxID := 0
	for _, group := range groups {
		if group.ID > maxID {
			maxID = group.ID
		}
	}
	return maxID + 1
}

// handleDeleteGroup deletes a group
func (s *Server) handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	// Extract group name from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/groups/")
	groupName := strings.Split(path, "/")[0]

	if strings.TrimSpace(groupName) == "" {
		http.Error(w, "Group name is required", http.StatusBadRequest)
		return
	}

	// Check if group exists and delete it (with proper mutex handling)
	var groupExists bool
	func() {
		groupsMutex.Lock()
		defer groupsMutex.Unlock()

		// Check if group exists
		if _, exists := groups[groupName]; !exists {
			groupExists = false
			return
		}

		// Delete group
		delete(groups, groupName)
		groupExists = true
	}()

	// Handle group not found case
	if !groupExists {
		response := map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Group '%s' not found", groupName),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	s.logger.Info("Deleting group", zap.String("name", groupName))

	// Save configuration to persist groups (mutex is now released)
	if err := s.SaveConfiguration(); err != nil {
		s.logger.Error("Failed to save configuration after deleting group", zap.Error(err))
	}

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Group '%s' deleted successfully", groupName),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
// handleAssignServer assigns a server to a group via web interface
func (s *Server) handleAssignServer(w http.ResponseWriter, r *http.Request) {
	var assignmentData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&assignmentData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	serverName, ok := assignmentData["server_name"].(string)
	if !ok || strings.TrimSpace(serverName) == "" {
		http.Error(w, "server_name is required", http.StatusBadRequest)
		return
	}

	// Support both group_name (legacy) and group_id (new)
	var groupName string
	var groupID int

	if gID, ok := assignmentData["group_id"].(float64); ok && gID > 0 {
		groupID = int(gID)
		// Find group name by ID
		groupsMutex.RLock()
		for name, group := range groups {
			if group.ID == groupID {
				groupName = name
				break
			}
		}
		groupsMutex.RUnlock()
	} else if gName, ok := assignmentData["group_name"].(string); ok && strings.TrimSpace(gName) != "" {
		// Legacy name path
		groupName = gName
		groupsMutex.RLock()
		if group, exists := groups[groupName]; exists {
			groupID = group.ID
		}
		groupsMutex.RUnlock()
	} else {
		http.Error(w, "group_id or group_name is required", http.StatusBadRequest)
		return
	}

	// Check if group exists
	groupsMutex.RLock()
	_, groupExists := groups[groupName]
	groupsMutex.RUnlock()

	if !groupExists {
		response := map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Group '%s' does not exist", groupName),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Assign server to group
	assignmentsMutex.Lock()
	serverGroupAssignments[serverName] = groupName
	assignmentsMutex.Unlock()

	// Save to configuration file
	if err := s.SaveConfiguration(); err != nil {
		s.logger.Error("Failed to save configuration after assigning server to group", zap.Error(err))
	}

	s.logger.Info("Server assigned to group via web interface", zap.String("server", serverName), zap.String("group", groupName))

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Server '%s' assigned to group '%s'", serverName, groupName),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetAssignments returns all server-to-group assignments
func (s *Server) handleGetAssignments(w http.ResponseWriter, r *http.Request) {
	assignmentsMutex.RLock()
	defer assignmentsMutex.RUnlock()

	assignments := make([]map[string]interface{}, 0, len(serverGroupAssignments))
	for serverName, groupName := range serverGroupAssignments {
		assignments = append(assignments, map[string]interface{}{
			"server_name": serverName,
			"group_name":  groupName,
		})
	}

	response := map[string]interface{}{
		"success":     true,
		"assignments": assignments,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleUnassignServer removes a server's group assignment via web interface (HTTP)
func (s *Server) handleUnassignServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct { ServerName string `json:"server_name"` }
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || strings.TrimSpace(payload.ServerName) == "" {
		http.Error(w, "server_name is required", http.StatusBadRequest)
		return
	}

	// Update in-memory assignments
	assignmentsMutex.Lock()
	delete(serverGroupAssignments, payload.ServerName)
	assignmentsMutex.Unlock()

	// Load config JSON and set group_id=0 for this server, remove group_name
	configPath := s.GetConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		http.Error(w, "failed to read config", http.StatusInternalServerError)
		return
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		http.Error(w, "failed to parse config", http.StatusInternalServerError)
		return
	}
	if servers, ok := cfg["mcpServers"].([]interface{}); ok {
		for i, it := range servers {
			if m, ok := it.(map[string]interface{}); ok {
				if name, ok := m["name"].(string); ok && name == payload.ServerName {
					m["group_id"] = 0
					delete(m, "group_name")
					servers[i] = m
					break
				}
			}
		}
		cfg["mcpServers"] = servers
	}
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		http.Error(w, "failed to serialize config", http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(configPath, out, 0600); err != nil {
		http.Error(w, "failed to write config", http.StatusInternalServerError)
		return
	}

	// Optionally reload internal config
	_ = s.ReloadConfiguration()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Server '%s' unassigned from group", payload.ServerName),
	})
}

// handleToggleGroupServers enables or disables all servers in a group
func (s *Server) handleToggleGroupServers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		GroupName string `json:"group_name"`
		Enabled   bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(payload.GroupName) == "" {
		http.Error(w, "group_name is required", http.StatusBadRequest)
		return
	}

	// Get all servers assigned to this group
	assignmentsMutex.RLock()
	var serversInGroup []string
	for serverName, groupName := range serverGroupAssignments {
		if groupName == payload.GroupName {
			serversInGroup = append(serversInGroup, serverName)
		}
	}
	assignmentsMutex.RUnlock()

	if len(serversInGroup) == 0 {
		response := map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("No servers in group '%s'", payload.GroupName),
			"updated": 0,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Track results for partial failure handling
	var successfulUpdates []string
	var failedUpdates []map[string]string
	updatedCount := 0

	// Process each server in the group
	for _, serverName := range serversInGroup {
		var updateErr error

		if payload.Enabled {
			// Enable server: Use EnableUpstreamServer which clears auto-disable automatically
			updateErr = s.storageManager.EnableUpstreamServer(serverName, true)
			if updateErr != nil {
				s.logger.Error("Failed to enable server via storage API",
					zap.String("server", serverName),
					zap.Error(updateErr))
				failedUpdates = append(failedUpdates, map[string]string{
					"server": serverName,
					"error":  updateErr.Error(),
				})
				continue
			}

			// Update upstream manager to trigger reconnection
			srv, err := s.storageManager.GetUpstreamServer(serverName)
			if err != nil {
				s.logger.Error("Failed to get server config after enabling",
					zap.String("server", serverName),
					zap.Error(err))
				failedUpdates = append(failedUpdates, map[string]string{
					"server": serverName,
					"error":  "failed to get config after enable",
				})
				continue
			}

			// Re-add server to upstream manager to trigger connection
			if err := s.upstreamManager.AddServerConfig(srv.Name, srv); err != nil {
				s.logger.Warn("Failed to add server to upstream manager",
					zap.String("server", srv.Name),
					zap.Error(err))
				// Don't fail the operation, just log warning
			}

			// Update s.config.Servers in memory for SaveConfiguration
			for i := range s.config.Servers {
				if s.config.Servers[i].Name == srv.Name {
					s.config.Servers[i].StartupMode = srv.StartupMode
					break
				}
			}

			// Emit ServerStateChanged event for tray update
			if s.eventBus != nil {
				s.eventBus.Publish(events.Event{
					Type:       events.ServerStateChanged,
					ServerName: serverName,
					Data: map[string]interface{}{
						"enabled":       true,
						"auto_disabled": false,
						"action":        "group_enable",
						"group":         payload.GroupName,
					},
				})
			}

		} else {
			// Disable server: Stop it first, then update storage
			s.upstreamManager.RemoveServer(serverName)

			// Use EnableUpstreamServer(name, false) to disable with two-phase commit
			updateErr = s.storageManager.EnableUpstreamServer(serverName, false)
			if updateErr != nil {
				s.logger.Error("Failed to disable server via storage API",
					zap.String("server", serverName),
					zap.Error(updateErr))
				failedUpdates = append(failedUpdates, map[string]string{
					"server": serverName,
					"error":  updateErr.Error(),
				})
				continue
			}

			// Get updated config for s.config.Servers sync
			srv, err := s.storageManager.GetUpstreamServer(serverName)
			if err != nil {
				s.logger.Error("Failed to get server config after disabling",
					zap.String("server", serverName),
					zap.Error(err))
				failedUpdates = append(failedUpdates, map[string]string{
					"server": serverName,
					"error":  "failed to get config after disable",
				})
				continue
			}

			// Update s.config.Servers in memory for SaveConfiguration
			for i := range s.config.Servers {
				if s.config.Servers[i].Name == srv.Name {
					s.config.Servers[i].StartupMode = srv.StartupMode
					break
				}
			}

			// Emit ServerStateChanged event for tray update
			if s.eventBus != nil {
				s.eventBus.Publish(events.Event{
					Type:       events.ServerStateChanged,
					ServerName: serverName,
					Data: map[string]interface{}{
						"enabled": false,
						"action":  "group_disable",
						"group":   payload.GroupName,
					},
				})
			}
		}

		successfulUpdates = append(successfulUpdates, serverName)
		updatedCount++
	}

	// Save configuration to disk (already done by storage APIs, but ensures s.config is persisted)
	if err := s.SaveConfiguration(); err != nil {
		s.logger.Error("Failed to save configuration after toggling group servers", zap.Error(err))
	}

	// Emit ServerGroupUpdated event to notify tray of group-level change
	if s.eventBus != nil {
		s.eventBus.Publish(events.Event{
			Type: events.ServerGroupUpdated,
			Data: map[string]interface{}{
				"group":              payload.GroupName,
				"action":             map[bool]string{true: "enable", false: "disable"}[payload.Enabled],
				"successful_updates": successfulUpdates,
				"failed_updates":     failedUpdates,
				"total_updated":      updatedCount,
			},
		})
	}

	// Trigger upstream server change event for index rebuild
	s.OnUpstreamServerChange()

	action := "disabled"
	if payload.Enabled {
		action = "enabled"
	}

	s.logger.Info("Toggled servers in group",
		zap.String("group", payload.GroupName),
		zap.String("action", action),
		zap.Int("successful", updatedCount),
		zap.Int("failed", len(failedUpdates)))

	// Build response with detailed results
	response := map[string]interface{}{
		"success": len(failedUpdates) == 0,
		"message": fmt.Sprintf("%d servers %s in group '%s'", updatedCount, action, payload.GroupName),
		"updated": updatedCount,
	}

	if len(failedUpdates) > 0 {
		response["partial_failure"] = true
		response["failed_servers"] = failedUpdates
		response["message"] = fmt.Sprintf("%d of %d servers %s in group '%s' (check failed_servers for details)",
			updatedCount, len(serversInGroup), action, payload.GroupName)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
