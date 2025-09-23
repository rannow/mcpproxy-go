//go:build !nogui && !headless && !linux

package tray

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"mcpproxy-go/internal/config"
)

// ChatMessage represents a single chat message
type ChatMessage struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`      // "user", "assistant", "agent"
	Content   string    `json:"content"`
	AgentType string    `json:"agent_type,omitempty"` // Which agent responded
	Timestamp time.Time `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ChatSession represents a chat session with diagnostic agents
type ChatSession struct {
	ID           string        `json:"id"`
	ServerName   string        `json:"server_name"`
	Messages     []ChatMessage `json:"messages"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	Status       string        `json:"status"` // "active", "completed", "archived"
}

// AgentType represents different types of diagnostic agents
type AgentType string

const (
	AgentTypeCoordinator  AgentType = "coordinator"
	AgentTypeLogAnalyzer  AgentType = "log_analyzer"
	AgentTypeDocAnalyzer  AgentType = "doc_analyzer"
	AgentTypeConfigUpdate AgentType = "config_update"
	AgentTypeInstaller    AgentType = "installer"
	AgentTypeTester       AgentType = "tester"
)

// ChatSystem manages the multi-agent chat interface
type ChatSystem struct {
	logger     *zap.Logger
	storage    ChatStorage
	agents     map[AgentType]DiagnosticAgentInterface
	mutex      sync.RWMutex

	// Server interface for interactions
	serverManager interface {
		GetServerTools(serverName string) ([]map[string]interface{}, error)
		EnableServer(serverName string, enabled bool) error
		GetAllServers() ([]map[string]interface{}, error)
		ReloadConfiguration() error
	}
}

// DiagnosticAgentInterface defines the interface for all diagnostic agents
type DiagnosticAgentInterface interface {
	ProcessMessage(ctx context.Context, message ChatMessage, session *ChatSession) (*ChatMessage, error)
	GetCapabilities() []string
	GetAgentType() AgentType
	CanHandle(message ChatMessage) bool
}

// ChatStorage interface for persisting chat sessions
type ChatStorage interface {
	SaveSession(session *ChatSession) error
	LoadSession(sessionID string) (*ChatSession, error)
	LoadSessionsByServer(serverName string) ([]*ChatSession, error)
	DeleteSession(sessionID string) error
	ListSessions() ([]*ChatSession, error)
}

// NewChatSystem creates a new chat system
func NewChatSystem(logger *zap.Logger, storage ChatStorage, serverManager interface {
	GetServerTools(serverName string) ([]map[string]interface{}, error)
	EnableServer(serverName string, enabled bool) error
	GetAllServers() ([]map[string]interface{}, error)
	ReloadConfiguration() error
}) *ChatSystem {
	cs := &ChatSystem{
		logger:        logger,
		storage:       storage,
		agents:        make(map[AgentType]DiagnosticAgentInterface),
		serverManager: serverManager,
	}

	// Initialize agents
	cs.initializeAgents()

	return cs
}

// initializeAgents creates and registers all diagnostic agents
func (cs *ChatSystem) initializeAgents() {
	cs.agents[AgentTypeCoordinator] = NewCoordinatorAgent(cs.logger, cs.serverManager)
	cs.agents[AgentTypeLogAnalyzer] = NewLogAnalyzerAgent(cs.logger)
	cs.agents[AgentTypeDocAnalyzer] = NewDocAnalyzerAgent(cs.logger)
	cs.agents[AgentTypeConfigUpdate] = NewConfigUpdateAgent(cs.logger, cs.serverManager)
	cs.agents[AgentTypeInstaller] = NewInstallerAgent(cs.logger)
	cs.agents[AgentTypeTester] = NewTesterAgent(cs.logger, cs.serverManager)
}

// CreateSession creates a new chat session for a server
func (cs *ChatSystem) CreateSession(serverName string) (*ChatSession, error) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	session := &ChatSession{
		ID:         generateSessionID(),
		ServerName: serverName,
		Messages:   []ChatMessage{},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Status:     "active",
	}

	// Add welcome message from coordinator
	welcomeMsg := ChatMessage{
		ID:        generateMessageID(),
		Role:      "assistant",
		Content:   fmt.Sprintf("Hello! I'm your diagnostic coordinator for the MCP server '%s'. I can help you with configuration, installation, testing, and troubleshooting. How can I assist you today?", serverName),
		AgentType: string(AgentTypeCoordinator),
		Timestamp: time.Now(),
	}

	session.Messages = append(session.Messages, welcomeMsg)

	if err := cs.storage.SaveSession(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	cs.logger.Info("Created new chat session",
		zap.String("session_id", session.ID),
		zap.String("server", serverName))

	return session, nil
}

// ProcessMessage processes a user message and generates agent responses
func (cs *ChatSystem) ProcessMessage(sessionID string, userMessage string) (*ChatSession, error) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	// Load session
	session, err := cs.storage.LoadSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	// Add user message
	userMsg := ChatMessage{
		ID:        generateMessageID(),
		Role:      "user",
		Content:   userMessage,
		Timestamp: time.Now(),
	}

	session.Messages = append(session.Messages, userMsg)
	session.UpdatedAt = time.Now()

	// Determine which agent should handle this message
	responsibleAgent := cs.selectAgent(userMsg, session)

	// Process message with selected agent
	ctx := context.Background()
	agentResponse, err := responsibleAgent.ProcessMessage(ctx, userMsg, session)
	if err != nil {
		cs.logger.Error("Agent failed to process message",
			zap.Error(err),
			zap.String("agent", string(responsibleAgent.GetAgentType())))

		// Create error response
		agentResponse = &ChatMessage{
			ID:        generateMessageID(),
			Role:      "assistant",
			Content:   fmt.Sprintf("I encountered an error while processing your request: %v", err),
			AgentType: string(responsibleAgent.GetAgentType()),
			Timestamp: time.Now(),
		}
	}

	session.Messages = append(session.Messages, *agentResponse)
	session.UpdatedAt = time.Now()

	// Save updated session
	if err := cs.storage.SaveSession(session); err != nil {
		cs.logger.Error("Failed to save session after processing", zap.Error(err))
	}

	return session, nil
}

// selectAgent determines which agent should handle a message
func (cs *ChatSystem) selectAgent(message ChatMessage, session *ChatSession) DiagnosticAgentInterface {
	// Check if any agent specifically can handle this message
	for _, agent := range cs.agents {
		if agent.CanHandle(message) {
			return agent
		}
	}

	// Default to coordinator
	return cs.agents[AgentTypeCoordinator]
}

// GetSession retrieves a chat session
func (cs *ChatSystem) GetSession(sessionID string) (*ChatSession, error) {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	return cs.storage.LoadSession(sessionID)
}

// GetSessionsByServer retrieves all chat sessions for a server
func (cs *ChatSystem) GetSessionsByServer(serverName string) ([]*ChatSession, error) {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	return cs.storage.LoadSessionsByServer(serverName)
}

// ListAllSessions retrieves all chat sessions
func (cs *ChatSystem) ListAllSessions() ([]*ChatSession, error) {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	return cs.storage.ListSessions()
}

// ArchiveSession archives a chat session
func (cs *ChatSystem) ArchiveSession(sessionID string) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	session, err := cs.storage.LoadSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	session.Status = "archived"
	session.UpdatedAt = time.Now()

	return cs.storage.SaveSession(session)
}

// DeleteSession deletes a chat session
func (cs *ChatSystem) DeleteSession(sessionID string) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	return cs.storage.DeleteSession(sessionID)
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}