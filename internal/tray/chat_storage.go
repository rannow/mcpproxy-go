//go:build !nogui && !headless && !linux

package tray

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// FileChatStorage implements ChatStorage using local files
type FileChatStorage struct {
	logger    *zap.Logger
	dataDir   string
	mutex     sync.RWMutex
}

// NewFileChatStorage creates a new file-based chat storage
func NewFileChatStorage(logger *zap.Logger, dataDir string) (*FileChatStorage, error) {
	chatDir := filepath.Join(dataDir, "chat_sessions")

	// Create chat directory if it doesn't exist
	if err := os.MkdirAll(chatDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create chat directory: %w", err)
	}

	return &FileChatStorage{
		logger:  logger,
		dataDir: chatDir,
	}, nil
}

// SaveSession saves a chat session to file
func (fs *FileChatStorage) SaveSession(session *ChatSession) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	sessionPath := fs.getSessionPath(session.ID)

	// Create directory if needed
	sessionDir := filepath.Dir(sessionPath)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	// Marshal session to JSON
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Write to file
	if err := os.WriteFile(sessionPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	fs.logger.Debug("Saved chat session",
		zap.String("session_id", session.ID),
		zap.String("server", session.ServerName))

	return nil
}

// LoadSession loads a chat session from file
func (fs *FileChatStorage) LoadSession(sessionID string) (*ChatSession, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	sessionPath := fs.getSessionPath(sessionID)

	// Check if file exists
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Read file
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	// Unmarshal JSON
	var session ChatSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// LoadSessionsByServer loads all chat sessions for a specific server
func (fs *FileChatStorage) LoadSessionsByServer(serverName string) ([]*ChatSession, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	allSessions, err := fs.loadAllSessions()
	if err != nil {
		return nil, err
	}

	var serverSessions []*ChatSession
	for _, session := range allSessions {
		if session.ServerName == serverName {
			serverSessions = append(serverSessions, session)
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(serverSessions, func(i, j int) bool {
		return serverSessions[i].CreatedAt.After(serverSessions[j].CreatedAt)
	})

	return serverSessions, nil
}

// DeleteSession deletes a chat session file
func (fs *FileChatStorage) DeleteSession(sessionID string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	sessionPath := fs.getSessionPath(sessionID)

	if err := os.Remove(sessionPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("session not found: %s", sessionID)
		}
		return fmt.Errorf("failed to delete session file: %w", err)
	}

	fs.logger.Info("Deleted chat session", zap.String("session_id", sessionID))
	return nil
}

// ListSessions lists all chat sessions
func (fs *FileChatStorage) ListSessions() ([]*ChatSession, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	sessions, err := fs.loadAllSessions()
	if err != nil {
		return nil, err
	}

	// Sort by update time (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}

// loadAllSessions loads all session files from disk
func (fs *FileChatStorage) loadAllSessions() ([]*ChatSession, error) {
	var sessions []*ChatSession

	// Walk through the chat directory
	err := filepath.Walk(fs.dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-JSON files
		if info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		// Read and parse session file
		data, err := os.ReadFile(path)
		if err != nil {
			fs.logger.Warn("Failed to read session file", zap.String("path", path), zap.Error(err))
			return nil // Continue with other files
		}

		var session ChatSession
		if err := json.Unmarshal(data, &session); err != nil {
			fs.logger.Warn("Failed to parse session file", zap.String("path", path), zap.Error(err))
			return nil // Continue with other files
		}

		sessions = append(sessions, &session)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk chat directory: %w", err)
	}

	return sessions, nil
}

// getSessionPath returns the file path for a session
func (fs *FileChatStorage) getSessionPath(sessionID string) string {
	return filepath.Join(fs.dataDir, fmt.Sprintf("%s.json", sessionID))
}

// CleanupOldSessions removes old sessions (older than specified days)
func (fs *FileChatStorage) CleanupOldSessions(maxAgeDays int) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	sessions, err := fs.loadAllSessions()
	if err != nil {
		return err
	}

	cutoffTime := time.Now().AddDate(0, 0, -maxAgeDays)
	deletedCount := 0

	for _, session := range sessions {
		if session.UpdatedAt.Before(cutoffTime) && session.Status != "active" {
			if err := fs.DeleteSession(session.ID); err != nil {
				fs.logger.Warn("Failed to delete old session",
					zap.String("session_id", session.ID),
					zap.Error(err))
			} else {
				deletedCount++
			}
		}
	}

	fs.logger.Info("Cleaned up old chat sessions",
		zap.Int("deleted_count", deletedCount),
		zap.Int("max_age_days", maxAgeDays))

	return nil
}

// GetStorageStats returns storage statistics
func (fs *FileChatStorage) GetStorageStats() (map[string]interface{}, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	sessions, err := fs.loadAllSessions()
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_sessions": len(sessions),
		"active_sessions": 0,
		"archived_sessions": 0,
		"servers": make(map[string]int),
	}

	for _, session := range sessions {
		if session.Status == "active" {
			stats["active_sessions"] = stats["active_sessions"].(int) + 1
		} else if session.Status == "archived" {
			stats["archived_sessions"] = stats["archived_sessions"].(int) + 1
		}

		servers := stats["servers"].(map[string]int)
		servers[session.ServerName] = servers[session.ServerName] + 1
	}

	return stats, nil
}