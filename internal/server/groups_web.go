package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"os"
	
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
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            padding: 30px;
        }
        h1 {
            color: #333;
            margin-bottom: 30px;
            text-align: center;
        }
        .group-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .group-card {
            border: 2px solid #ddd;
            border-radius: 8px;
            padding: 20px;
            background: #fafafa;
            position: relative;
        }
        .group-header {
            display: flex;
            align-items: center;
            margin-bottom: 15px;
        }
        .group-color {
            width: 20px;
            height: 20px;
            border-radius: 50%;
            margin-right: 10px;
            border: 2px solid #ccc;
        }
        .group-name {
            font-size: 18px;
            font-weight: bold;
            flex: 1;
        }
        .group-actions {
            display: flex;
            gap: 10px;
            margin-top: 15px;
        }
        .btn {
            padding: 8px 16px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
            transition: background-color 0.2s;
        }
        .btn-primary {
            background: #007bff;
            color: white;
        }
        .btn-primary:hover {
            background: #0056b3;
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
        .create-group {
            border: 2px dashed #007bff;
            background: #f8f9ff;
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 150px;
            cursor: pointer;
            transition: all 0.2s;
        }
        .create-group:hover {
            background: #e6f3ff;
            border-color: #0056b3;
        }
        .create-group-text {
            text-align: center;
            color: #007bff;
            font-size: 16px;
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
        .modal-actions {
            display: flex;
            gap: 10px;
            justify-content: flex-end;
            margin-top: 20px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🏷️ Server Group Management</h1>
        
        <div class="group-grid" id="groupGrid">
            <!-- Groups will be loaded here -->
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

        let groups = [];
        let editingGroup = null;
        let selectedColor = colors[0];

        // Initialize page
        document.addEventListener('DOMContentLoaded', function() {
            loadGroups();
            initColorPicker();
        });

        // Load groups from server
        function loadGroups() {
            fetch('/api/groups')
                .then(response => response.json())
                .then(data => {
                    groups = data.groups || [];
                    renderGroups();
                })
                .catch(error => {
                    console.error('Failed to load groups:', error);
                    renderGroups(); // Render empty state
                });
        }

        // Render groups grid
        function renderGroups() {
            const grid = document.getElementById('groupGrid');
            grid.innerHTML = '';

            // Add create group card
            const createCard = document.createElement('div');
            createCard.className = 'group-card create-group';
            createCard.onclick = () => openCreateModal();
            createCard.innerHTML = '<div class="create-group-text">➕<br>Create New Group</div>';
            grid.appendChild(createCard);

            // Add existing groups
            groups.forEach(group => {
                const card = document.createElement('div');
                card.className = 'group-card';
                card.innerHTML = ` + "`" + `
                    <div class="group-header">
                        <div class="group-color" style="background-color: ${group.color}"></div>
                        <div class="group-name">${group.name}</div>
                    </div>
                    <div class="group-actions">
                        <button class="btn btn-primary" onclick="editGroup('${group.name}')">Edit</button>
                        <button class="btn btn-danger" onclick="deleteGroup('${group.name}')">Delete</button>
                    </div>
                ` + "`" + `;
                grid.appendChild(card);
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
            selectedColor = colors[0];
            document.querySelector('.color-option').classList.add('selected');
            document.getElementById('groupModal').style.display = 'block';
        }

        // Edit group
        function editGroup(groupName) {
            const group = groups.find(g => g.name === groupName);
            if (!group) return;

            editingGroup = group;
            document.getElementById('modalTitle').textContent = 'Edit Group';
            document.getElementById('groupName').value = group.name;
            
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
                    loadGroups();
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

	// Create new group
	s.setGroup(name, &Group{
		Name:  name,
		Color: color,
	})

	s.logger.Info("Creating group", zap.String("name", name), zap.String("color", color))

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

	// Update group with proper mutex handling to avoid deadlock
	var updateResult struct {
		success bool
		errorMsg string
	}

	func() {
		groupsMutex.Lock()
		defer groupsMutex.Unlock()

		// Check if old group exists
		_, exists := groups[oldName]
		if !exists {
			updateResult.success = false
			updateResult.errorMsg = fmt.Sprintf("Group '%s' not found", oldName)
			return
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

		// Update/create group
		groups[newName] = &Group{
			Name:  newName,
			Color: color,
		}
		updateResult.success = true
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

	s.logger.Info("Updating group", zap.String("old_name", oldName), zap.String("new_name", newName), zap.String("color", color))

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
