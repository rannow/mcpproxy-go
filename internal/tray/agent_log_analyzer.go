//go:build !nogui && !headless && !linux

package tray

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

// LogAnalyzerAgent analyzes logs to identify issues and patterns
type LogAnalyzerAgent struct {
	logger *zap.Logger
}

// LogAnalysisResult contains the results of log analysis
type LogAnalysisResult struct {
	TotalLines      int                 `json:"total_lines"`
	ErrorCount      int                 `json:"error_count"`
	WarningCount    int                 `json:"warning_count"`
	RecentErrors    []LogEntry          `json:"recent_errors"`
	ErrorPatterns   map[string]int      `json:"error_patterns"`
	Recommendations []string            `json:"recommendations"`
	TimeRange       LogTimeRange        `json:"time_range"`
	Summary         string              `json:"summary"`
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source"`
}

// LogTimeRange represents the time range of analyzed logs
type LogTimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// NewLogAnalyzerAgent creates a new log analyzer agent
func NewLogAnalyzerAgent(logger *zap.Logger) *LogAnalyzerAgent {
	return &LogAnalyzerAgent{
		logger: logger,
	}
}

// ProcessMessage processes a message requesting log analysis
func (la *LogAnalyzerAgent) ProcessMessage(ctx context.Context, message ChatMessage, session *ChatSession) (*ChatMessage, error) {
	la.logger.Info("Log analyzer processing message",
		zap.String("session_id", session.ID),
		zap.String("server", session.ServerName))

	content := strings.ToLower(message.Content)

	var response string
	var metadata map[string]interface{}

	switch {
	case la.containsKeywords(content, []string{"recent errors", "latest errors", "show errors"}):
		response, metadata = la.analyzeRecentErrors(session.ServerName)

	case la.containsKeywords(content, []string{"error patterns", "common errors", "recurring"}):
		response, metadata = la.analyzeErrorPatterns(session.ServerName)

	case la.containsKeywords(content, []string{"connection", "timeout", "network"}):
		response, metadata = la.analyzeConnectionIssues(session.ServerName)

	case la.containsKeywords(content, []string{"performance", "slow", "response time"}):
		response, metadata = la.analyzePerformanceIssues(session.ServerName)

	case la.containsKeywords(content, []string{"authentication", "auth", "login", "oauth"}):
		response, metadata = la.analyzeAuthenticationIssues(session.ServerName)

	case la.containsKeywords(content, []string{"full analysis", "complete analysis", "analyze all"}):
		response, metadata = la.performFullAnalysis(session.ServerName)

	default:
		response, metadata = la.provideLogAnalysisGuidance(session.ServerName)
	}

	return &ChatMessage{
		ID:        generateMessageID(),
		Role:      "assistant",
		Content:   response,
		AgentType: string(AgentTypeLogAnalyzer),
		Timestamp: time.Now(),
		Metadata:  metadata,
	}, nil
}

// GetCapabilities returns the capabilities of the log analyzer agent
func (la *LogAnalyzerAgent) GetCapabilities() []string {
	return []string{
		"Error pattern analysis",
		"Recent error identification",
		"Connection issue diagnosis",
		"Performance problem detection",
		"Authentication failure analysis",
		"Log trend analysis",
		"Issue categorization",
	}
}

// GetAgentType returns the agent type
func (la *LogAnalyzerAgent) GetAgentType() AgentType {
	return AgentTypeLogAnalyzer
}

// CanHandle determines if this agent can handle a message
func (la *LogAnalyzerAgent) CanHandle(message ChatMessage) bool {
	content := strings.ToLower(message.Content)
	keywords := []string{
		"log", "logs", "error", "errors", "analyze", "analysis",
		"debug", "troubleshoot", "issue", "problem", "failure",
		"connection", "timeout", "performance", "slow", "auth",
	}

	return la.containsKeywords(content, keywords)
}

// containsKeywords checks if content contains any of the specified keywords
func (la *LogAnalyzerAgent) containsKeywords(content string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

// analyzeRecentErrors analyzes recent error messages
func (la *LogAnalyzerAgent) analyzeRecentErrors(serverName string) (string, map[string]interface{}) {
	la.logger.Info("Analyzing recent errors", zap.String("server", serverName))

	analysis, err := la.readAndAnalyzeLogs(serverName, 50) // Last 50 lines
	if err != nil {
		return fmt.Sprintf("❌ Failed to analyze logs: %v", err), map[string]interface{}{
			"error": err.Error(),
		}
	}

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("📋 **Recent Errors for %s**\n\n", serverName))

	if analysis.ErrorCount == 0 {
		responseBuilder.WriteString("✅ **No recent errors found!**\n\n")
		responseBuilder.WriteString("Your server appears to be running smoothly. ")
		responseBuilder.WriteString("The logs show no error messages in the recent activity.\n")
	} else {
		responseBuilder.WriteString(fmt.Sprintf("🔍 **Found %d error(s) in recent logs**\n\n", analysis.ErrorCount))

		if len(analysis.RecentErrors) > 0 {
			responseBuilder.WriteString("**Most Recent Errors:**\n")
			for i, entry := range analysis.RecentErrors {
				if i >= 5 { // Show max 5 recent errors
					break
				}
				responseBuilder.WriteString(fmt.Sprintf("• `%s` - %s\n",
					entry.Timestamp.Format("15:04:05"), entry.Message))
			}
			responseBuilder.WriteString("\n")
		}

		if len(analysis.Recommendations) > 0 {
			responseBuilder.WriteString("💡 **Recommendations:**\n")
			for _, rec := range analysis.Recommendations {
				responseBuilder.WriteString(fmt.Sprintf("• %s\n", rec))
			}
		}
	}

	return responseBuilder.String(), map[string]interface{}{
		"error_count":    analysis.ErrorCount,
		"recent_errors":  analysis.RecentErrors,
		"recommendations": analysis.Recommendations,
	}
}

// analyzeErrorPatterns analyzes error patterns and frequency
func (la *LogAnalyzerAgent) analyzeErrorPatterns(serverName string) (string, map[string]interface{}) {
	la.logger.Info("Analyzing error patterns", zap.String("server", serverName))

	analysis, err := la.readAndAnalyzeLogs(serverName, 1000) // Analyze more lines for patterns
	if err != nil {
		return fmt.Sprintf("❌ Failed to analyze logs: %v", err), map[string]interface{}{
			"error": err.Error(),
		}
	}

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("📊 **Error Patterns for %s**\n\n", serverName))

	if len(analysis.ErrorPatterns) == 0 {
		responseBuilder.WriteString("✅ **No error patterns detected!**\n\n")
		responseBuilder.WriteString("Your server logs don't show any recurring error patterns.\n")
	} else {
		responseBuilder.WriteString("🔍 **Common Error Patterns:**\n\n")

		// Sort patterns by frequency
		type patternCount struct {
			pattern string
			count   int
		}
		var patterns []patternCount
		for pattern, count := range analysis.ErrorPatterns {
			patterns = append(patterns, patternCount{pattern: pattern, count: count})
		}
		sort.Slice(patterns, func(i, j int) bool {
			return patterns[i].count > patterns[j].count
		})

		for i, pc := range patterns {
			if i >= 5 { // Show top 5 patterns
				break
			}
			responseBuilder.WriteString(fmt.Sprintf("**%d. %s** (occurred %d times)\n", i+1, pc.pattern, pc.count))
		}

		responseBuilder.WriteString("\n💡 **Pattern Analysis:**\n")
		for _, rec := range analysis.Recommendations {
			responseBuilder.WriteString(fmt.Sprintf("• %s\n", rec))
		}
	}

	return responseBuilder.String(), map[string]interface{}{
		"error_patterns":  analysis.ErrorPatterns,
		"total_errors":    analysis.ErrorCount,
		"recommendations": analysis.Recommendations,
	}
}

// analyzeConnectionIssues analyzes connection-related problems
func (la *LogAnalyzerAgent) analyzeConnectionIssues(serverName string) (string, map[string]interface{}) {
	la.logger.Info("Analyzing connection issues", zap.String("server", serverName))

	analysis, err := la.readAndAnalyzeLogs(serverName, 200)
	if err != nil {
		return fmt.Sprintf("❌ Failed to analyze logs: %v", err), map[string]interface{}{
			"error": err.Error(),
		}
	}

	connectionIssues := la.filterConnectionIssues(analysis.RecentErrors)

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("🌐 **Connection Analysis for %s**\n\n", serverName))

	if len(connectionIssues) == 0 {
		responseBuilder.WriteString("✅ **No connection issues detected!**\n\n")
		responseBuilder.WriteString("The server appears to have stable connectivity.\n")
	} else {
		responseBuilder.WriteString(fmt.Sprintf("🔍 **Found %d connection-related issues:**\n\n", len(connectionIssues)))

		for i, issue := range connectionIssues {
			if i >= 5 {
				break
			}
			responseBuilder.WriteString(fmt.Sprintf("• `%s` - %s\n",
				issue.Timestamp.Format("15:04:05"), issue.Message))
		}

		responseBuilder.WriteString("\n💡 **Connection Troubleshooting:**\n")
		responseBuilder.WriteString("• Check network connectivity\n")
		responseBuilder.WriteString("• Verify server endpoint URLs\n")
		responseBuilder.WriteString("• Check firewall settings\n")
		responseBuilder.WriteString("• Validate authentication credentials\n")
	}

	return responseBuilder.String(), map[string]interface{}{
		"connection_issues": connectionIssues,
		"issue_count":      len(connectionIssues),
	}
}

// analyzePerformanceIssues analyzes performance-related problems
func (la *LogAnalyzerAgent) analyzePerformanceIssues(serverName string) (string, map[string]interface{}) {
	la.logger.Info("Analyzing performance issues", zap.String("server", serverName))

	analysis, err := la.readAndAnalyzeLogs(serverName, 200)
	if err != nil {
		return fmt.Sprintf("❌ Failed to analyze logs: %v", err), map[string]interface{}{
			"error": err.Error(),
		}
	}

	performanceIssues := la.filterPerformanceIssues(analysis.RecentErrors)

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("⚡ **Performance Analysis for %s**\n\n", serverName))

	if len(performanceIssues) == 0 {
		responseBuilder.WriteString("✅ **No performance issues detected!**\n\n")
		responseBuilder.WriteString("The server appears to be performing well.\n")
	} else {
		responseBuilder.WriteString(fmt.Sprintf("🔍 **Found %d performance-related issues:**\n\n", len(performanceIssues)))

		for i, issue := range performanceIssues {
			if i >= 5 {
				break
			}
			responseBuilder.WriteString(fmt.Sprintf("• `%s` - %s\n",
				issue.Timestamp.Format("15:04:05"), issue.Message))
		}

		responseBuilder.WriteString("\n💡 **Performance Optimization:**\n")
		responseBuilder.WriteString("• Check server resource usage\n")
		responseBuilder.WriteString("• Monitor response times\n")
		responseBuilder.WriteString("• Consider caching strategies\n")
		responseBuilder.WriteString("• Review timeout settings\n")
	}

	return responseBuilder.String(), map[string]interface{}{
		"performance_issues": performanceIssues,
		"issue_count":       len(performanceIssues),
	}
}

// analyzeAuthenticationIssues analyzes authentication-related problems
func (la *LogAnalyzerAgent) analyzeAuthenticationIssues(serverName string) (string, map[string]interface{}) {
	la.logger.Info("Analyzing authentication issues", zap.String("server", serverName))

	analysis, err := la.readAndAnalyzeLogs(serverName, 200)
	if err != nil {
		return fmt.Sprintf("❌ Failed to analyze logs: %v", err), map[string]interface{}{
			"error": err.Error(),
		}
	}

	authIssues := la.filterAuthenticationIssues(analysis.RecentErrors)

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("🔐 **Authentication Analysis for %s**\n\n", serverName))

	if len(authIssues) == 0 {
		responseBuilder.WriteString("✅ **No authentication issues detected!**\n\n")
		responseBuilder.WriteString("Authentication appears to be working correctly.\n")
	} else {
		responseBuilder.WriteString(fmt.Sprintf("🔍 **Found %d authentication-related issues:**\n\n", len(authIssues)))

		for i, issue := range authIssues {
			if i >= 5 {
				break
			}
			responseBuilder.WriteString(fmt.Sprintf("• `%s` - %s\n",
				issue.Timestamp.Format("15:04:05"), issue.Message))
		}

		responseBuilder.WriteString("\n💡 **Authentication Troubleshooting:**\n")
		responseBuilder.WriteString("• Verify API keys and credentials\n")
		responseBuilder.WriteString("• Check OAuth token validity\n")
		responseBuilder.WriteString("• Review authentication configuration\n")
		responseBuilder.WriteString("• Test authentication flow manually\n")
	}

	return responseBuilder.String(), map[string]interface{}{
		"auth_issues":  authIssues,
		"issue_count": len(authIssues),
	}
}

// performFullAnalysis performs a comprehensive log analysis
func (la *LogAnalyzerAgent) performFullAnalysis(serverName string) (string, map[string]interface{}) {
	la.logger.Info("Performing full log analysis", zap.String("server", serverName))

	analysis, err := la.readAndAnalyzeLogs(serverName, 500)
	if err != nil {
		return fmt.Sprintf("❌ Failed to analyze logs: %v", err), map[string]interface{}{
			"error": err.Error(),
		}
	}

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("📊 **Complete Log Analysis for %s**\n\n", serverName))

	// Summary
	responseBuilder.WriteString("**📋 Summary:**\n")
	responseBuilder.WriteString(fmt.Sprintf("• Total log lines analyzed: %d\n", analysis.TotalLines))
	responseBuilder.WriteString(fmt.Sprintf("• Errors found: %d\n", analysis.ErrorCount))
	responseBuilder.WriteString(fmt.Sprintf("• Warnings found: %d\n", analysis.WarningCount))
	responseBuilder.WriteString(fmt.Sprintf("• Time range: %s to %s\n\n",
		analysis.TimeRange.Start.Format("Jan 02 15:04"),
		analysis.TimeRange.End.Format("Jan 02 15:04")))

	// Analysis summary
	responseBuilder.WriteString(analysis.Summary)

	if len(analysis.Recommendations) > 0 {
		responseBuilder.WriteString("\n\n💡 **Recommendations:**\n")
		for _, rec := range analysis.Recommendations {
			responseBuilder.WriteString(fmt.Sprintf("• %s\n", rec))
		}
	}

	return responseBuilder.String(), map[string]interface{}{
		"full_analysis": analysis,
	}
}

// provideLogAnalysisGuidance provides guidance on log analysis options
func (la *LogAnalyzerAgent) provideLogAnalysisGuidance(serverName string) (string, map[string]interface{}) {
	response := fmt.Sprintf(`📋 **Log Analysis Options for %s**

I can help you analyze server logs in several ways:

**🔍 Available Analysis Types:**

**1. Recent Errors** - "Show me recent errors"
• Find the latest error messages
• Identify immediate issues

**2. Error Patterns** - "Analyze error patterns"
• Find recurring issues
• Identify common problems

**3. Connection Issues** - "Check connection problems"
• Analyze network connectivity
• Find timeout and connection failures

**4. Performance Issues** - "Check performance problems"
• Find slow response times
• Identify performance bottlenecks

**5. Authentication Issues** - "Check auth problems"
• Find login failures
• Analyze credential issues

**6. Full Analysis** - "Perform complete analysis"
• Comprehensive log review
• Overall health assessment

**💡 Example Requests:**
• "Show me the latest errors"
• "What connection problems do you see?"
• "Analyze performance issues"
• "Check for authentication failures"
• "Give me a complete log analysis"

What type of analysis would you like me to perform?`, serverName)

	return response, map[string]interface{}{
		"guidance_type": "log_analysis",
		"available_analyses": []string{
			"recent_errors",
			"error_patterns",
			"connection_issues",
			"performance_issues",
			"authentication_issues",
			"full_analysis",
		},
	}
}

// readAndAnalyzeLogs reads and analyzes log files for a server
func (la *LogAnalyzerAgent) readAndAnalyzeLogs(serverName string, maxLines int) (*LogAnalysisResult, error) {
	// Determine log file paths
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Try different log locations
	logPaths := []string{
		filepath.Join(homeDir, "Library", "Logs", "mcpproxy", fmt.Sprintf("server-%s.log", serverName)),
		filepath.Join(homeDir, ".mcpproxy", "logs", fmt.Sprintf("server-%s.log", serverName)),
		filepath.Join(homeDir, "Library", "Logs", "mcpproxy", "main.log"),
		filepath.Join(homeDir, ".mcpproxy", "logs", "main.log"),
	}

	var logEntries []LogEntry
	var totalLines int

	for _, logPath := range logPaths {
		entries, lines, err := la.readLogFile(logPath, maxLines)
		if err != nil {
			la.logger.Debug("Could not read log file", zap.String("path", logPath), zap.Error(err))
			continue
		}
		logEntries = append(logEntries, entries...)
		totalLines += lines
		if len(logEntries) >= maxLines {
			break
		}
	}

	if len(logEntries) == 0 {
		return nil, fmt.Errorf("no log files found or readable")
	}

	// Analyze the log entries
	return la.analyzeLogEntries(logEntries, totalLines), nil
}

// readLogFile reads a single log file and extracts log entries
func (la *LogAnalyzerAgent) readLogFile(logPath string, maxLines int) ([]LogEntry, int, error) {
	file, err := os.Open(logPath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(file)
	lineCount := 0

	// Regex patterns for different log formats
	timestampPattern := regexp.MustCompile(`(\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2})`)
	levelPattern := regexp.MustCompile(`(?i)(ERROR|WARN|INFO|DEBUG|TRACE)`)

	for scanner.Scan() && lineCount < maxLines {
		line := scanner.Text()
		lineCount++

		// Extract timestamp
		timestampMatch := timestampPattern.FindStringSubmatch(line)
		var timestamp time.Time
		if len(timestampMatch) > 1 {
			timestamp, _ = time.Parse("2006-01-02T15:04:05", timestampMatch[1])
			if timestamp.IsZero() {
				timestamp, _ = time.Parse("2006-01-02 15:04:05", timestampMatch[1])
			}
		}
		if timestamp.IsZero() {
			timestamp = time.Now() // Fallback
		}

		// Extract log level
		levelMatch := levelPattern.FindStringSubmatch(line)
		level := "INFO"
		if len(levelMatch) > 1 {
			level = strings.ToUpper(levelMatch[1])
		}

		entries = append(entries, LogEntry{
			Timestamp: timestamp,
			Level:     level,
			Message:   line,
			Source:    filepath.Base(logPath),
		})
	}

	return entries, lineCount, scanner.Err()
}

// analyzeLogEntries analyzes log entries and generates insights
func (la *LogAnalyzerAgent) analyzeLogEntries(entries []LogEntry, totalLines int) *LogAnalysisResult {
	result := &LogAnalysisResult{
		TotalLines:    totalLines,
		ErrorPatterns: make(map[string]int),
		TimeRange: LogTimeRange{
			Start: time.Now(),
			End:   time.Now().Add(-24 * time.Hour),
		},
	}

	var errors []LogEntry
	var warnings []LogEntry

	// Analyze each entry
	for _, entry := range entries {
		// Update time range
		if entry.Timestamp.Before(result.TimeRange.Start) {
			result.TimeRange.Start = entry.Timestamp
		}
		if entry.Timestamp.After(result.TimeRange.End) {
			result.TimeRange.End = entry.Timestamp
		}

		// Count by level
		switch entry.Level {
		case "ERROR":
			result.ErrorCount++
			errors = append(errors, entry)

			// Extract error patterns
			pattern := la.extractErrorPattern(entry.Message)
			if pattern != "" {
				result.ErrorPatterns[pattern]++
			}
		case "WARN":
			result.WarningCount++
			warnings = append(warnings, entry)
		}
	}

	// Sort errors by timestamp (newest first)
	sort.Slice(errors, func(i, j int) bool {
		return errors[i].Timestamp.After(errors[j].Timestamp)
	})

	// Keep recent errors
	if len(errors) > 10 {
		result.RecentErrors = errors[:10]
	} else {
		result.RecentErrors = errors
	}

	// Generate recommendations
	result.Recommendations = la.generateRecommendations(result)

	// Generate summary
	result.Summary = la.generateSummary(result)

	return result
}

// extractErrorPattern extracts a pattern from an error message
func (la *LogAnalyzerAgent) extractErrorPattern(message string) string {
	// Common error patterns
	patterns := map[string]*regexp.Regexp{
		"Connection timeout":     regexp.MustCompile(`(?i)timeout|timed out`),
		"Connection refused":     regexp.MustCompile(`(?i)connection refused|connect: connection refused`),
		"Authentication failed":  regexp.MustCompile(`(?i)authentication failed|auth.*fail|401|unauthorized`),
		"Not found":             regexp.MustCompile(`(?i)not found|404|no such`),
		"Permission denied":      regexp.MustCompile(`(?i)permission denied|403|forbidden`),
		"Network error":          regexp.MustCompile(`(?i)network.*error|dns.*error|no route to host`),
		"Server error":           regexp.MustCompile(`(?i)internal server error|500|server error`),
		"Configuration error":    regexp.MustCompile(`(?i)config.*error|configuration.*invalid`),
	}

	for pattern, regex := range patterns {
		if regex.MatchString(message) {
			return pattern
		}
	}

	return ""
}

// filterConnectionIssues filters for connection-related issues
func (la *LogAnalyzerAgent) filterConnectionIssues(entries []LogEntry) []LogEntry {
	var issues []LogEntry
	connectionKeywords := []string{
		"connection", "timeout", "network", "dns", "refused", "unreachable",
	}

	for _, entry := range entries {
		message := strings.ToLower(entry.Message)
		for _, keyword := range connectionKeywords {
			if strings.Contains(message, keyword) {
				issues = append(issues, entry)
				break
			}
		}
	}

	return issues
}

// filterPerformanceIssues filters for performance-related issues
func (la *LogAnalyzerAgent) filterPerformanceIssues(entries []LogEntry) []LogEntry {
	var issues []LogEntry
	performanceKeywords := []string{
		"slow", "performance", "timeout", "delay", "latency", "response time",
	}

	for _, entry := range entries {
		message := strings.ToLower(entry.Message)
		for _, keyword := range performanceKeywords {
			if strings.Contains(message, keyword) {
				issues = append(issues, entry)
				break
			}
		}
	}

	return issues
}

// filterAuthenticationIssues filters for authentication-related issues
func (la *LogAnalyzerAgent) filterAuthenticationIssues(entries []LogEntry) []LogEntry {
	var issues []LogEntry
	authKeywords := []string{
		"auth", "login", "oauth", "token", "credential", "unauthorized", "401",
	}

	for _, entry := range entries {
		message := strings.ToLower(entry.Message)
		for _, keyword := range authKeywords {
			if strings.Contains(message, keyword) {
				issues = append(issues, entry)
				break
			}
		}
	}

	return issues
}

// generateRecommendations generates recommendations based on analysis
func (la *LogAnalyzerAgent) generateRecommendations(result *LogAnalysisResult) []string {
	var recommendations []string

	if result.ErrorCount == 0 {
		recommendations = append(recommendations, "No issues detected - server is running smoothly")
		return recommendations
	}

	// Check for common patterns
	for pattern, count := range result.ErrorPatterns {
		if count > 3 {
			switch pattern {
			case "Connection timeout":
				recommendations = append(recommendations, "Consider increasing timeout values or checking network connectivity")
			case "Authentication failed":
				recommendations = append(recommendations, "Verify API credentials and authentication configuration")
			case "Connection refused":
				recommendations = append(recommendations, "Check if the target service is running and accessible")
			case "Not found":
				recommendations = append(recommendations, "Verify server endpoints and URL configurations")
			}
		}
	}

	if result.ErrorCount > 10 {
		recommendations = append(recommendations, "High error rate detected - consider investigating root causes")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Review error messages and check server configuration")
	}

	return recommendations
}

// generateSummary generates a summary of the analysis
func (la *LogAnalyzerAgent) generateSummary(result *LogAnalysisResult) string {
	if result.ErrorCount == 0 && result.WarningCount == 0 {
		return "✅ **Healthy**: No errors or warnings detected in the analyzed logs."
	}

	var summaryParts []string

	if result.ErrorCount > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("🔴 **%d errors** detected", result.ErrorCount))
	}

	if result.WarningCount > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("🟡 **%d warnings** found", result.WarningCount))
	}

	// Add pattern information
	if len(result.ErrorPatterns) > 0 {
		var topPattern string
		var maxCount int
		for pattern, count := range result.ErrorPatterns {
			if count > maxCount {
				maxCount = count
				topPattern = pattern
			}
		}
		summaryParts = append(summaryParts, fmt.Sprintf("Most common issue: **%s** (%d occurrences)", topPattern, maxCount))
	}

	return strings.Join(summaryParts, " • ")
}