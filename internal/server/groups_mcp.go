package server

import (
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// ServerGroupAssignment stores server-to-group assignments
type ServerGroupAssignment struct {
	ServerName string `json:"server_name"`
	GroupName  string `json:"group_name"`
}

// In-memory storage for server-group assignments
var (
	serverGroupAssignments = make(map[string]string) // serverName -> groupName
	assignmentsMutex       = sync.RWMutex{}
)

// handleGroupsTool handles the groups MCP tool
func (s *Server) handleGroupsTool(args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter is required")
	}

	switch operation {
	case "list_groups":
		return s.listGroups()
	case "assign_server":
		return s.assignServerToGroup(args)
	case "unassign_server":
		return s.unassignServerFromGroup(args)
	case "list_assignments":
		return s.listServerAssignments()
	case "get_group_servers":
		return s.getGroupServers(args)
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

// listGroups returns all available groups
func (s *Server) listGroups() (interface{}, error) {
	groupsMutex.RLock()
	defer groupsMutex.RUnlock()

	groupList := make([]map[string]interface{}, 0, len(groups))
	for _, group := range groups {
		groupList = append(groupList, map[string]interface{}{
			"name":  group.Name,
			"color": group.Color,
		})
	}

	return map[string]interface{}{
		"groups": groupList,
	}, nil
}

// assignServerToGroup assigns a server to a group
func (s *Server) assignServerToGroup(args map[string]interface{}) (interface{}, error) {
	serverName, ok := args["server_name"].(string)
	if !ok || strings.TrimSpace(serverName) == "" {
		return nil, fmt.Errorf("server_name parameter is required")
	}

	groupName, ok := args["group_name"].(string)
	if !ok || strings.TrimSpace(groupName) == "" {
		return nil, fmt.Errorf("group_name parameter is required")
	}

	// Check if group exists
	groupsMutex.RLock()
	_, groupExists := groups[groupName]
	groupsMutex.RUnlock()

	if !groupExists {
		return nil, fmt.Errorf("group '%s' does not exist", groupName)
	}

	// Assign server to group
	assignmentsMutex.Lock()
	serverGroupAssignments[serverName] = groupName
	assignmentsMutex.Unlock()

	// Save to configuration file
	if err := s.SaveConfiguration(); err != nil {
		s.logger.Error("Failed to save configuration after assigning server to group", zap.Error(err))
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Server '%s' assigned to group '%s'", serverName, groupName),
	}, nil
}

// unassignServerFromGroup removes a server from its group
func (s *Server) unassignServerFromGroup(args map[string]interface{}) (interface{}, error) {
	serverName, ok := args["server_name"].(string)
	if !ok || strings.TrimSpace(serverName) == "" {
		return nil, fmt.Errorf("server_name parameter is required")
	}

	assignmentsMutex.Lock()
	delete(serverGroupAssignments, serverName)
	assignmentsMutex.Unlock()

	// Save to configuration file
	if err := s.SaveConfiguration(); err != nil {
		s.logger.Error("Failed to save configuration after unassigning server from group", zap.Error(err))
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Server '%s' unassigned from group", serverName),
	}, nil
}

// listServerAssignments returns all server-to-group assignments
func (s *Server) listServerAssignments() (interface{}, error) {
	assignmentsMutex.RLock()
	defer assignmentsMutex.RUnlock()

	assignments := make([]map[string]interface{}, 0, len(serverGroupAssignments))
	for serverName, groupName := range serverGroupAssignments {
		assignments = append(assignments, map[string]interface{}{
			"server_name": serverName,
			"group_name":  groupName,
		})
	}

	return map[string]interface{}{
		"assignments": assignments,
	}, nil
}

// getGroupServers returns all servers assigned to a specific group
func (s *Server) getGroupServers(args map[string]interface{}) (interface{}, error) {
	groupName, ok := args["group_name"].(string)
	if !ok || strings.TrimSpace(groupName) == "" {
		return nil, fmt.Errorf("group_name parameter is required")
	}

	assignmentsMutex.RLock()
	defer assignmentsMutex.RUnlock()

	servers := make([]string, 0)
	for serverName, assignedGroup := range serverGroupAssignments {
		if assignedGroup == groupName {
			servers = append(servers, serverName)
		}
	}

	return map[string]interface{}{
		"group_name": groupName,
		"servers":    servers,
	}, nil
}
