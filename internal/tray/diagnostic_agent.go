package tray

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"mcpproxy-go/internal/config"
	"go.uber.org/zap"
)

// DiagnosticAgent analyzes MCP server connection issues using AI
type DiagnosticAgent struct {
	logger    *zap.Logger
	llmClient LLMClient
}

// DiagnosticReport contains the analysis results
type DiagnosticReport struct {
	ServerName       string                 `json:"server_name"`
	Issues           []string               `json:"issues"`
	Recommendations  []string               `json:"recommendations"`
	LogAnalysis      LogAnalysis            `json:"log_analysis"`
	ConfigAnalysis   ConfigAnalysis         `json:"config_analysis"`
	RepoAnalysis     RepositoryAnalysis     `json:"repo_analysis"`
	AIAnalysis       string                 `json:"ai_analysis"`
	SuggestedConfig  string                 `json:"suggested_config,omitempty"`
	Timestamp        time.Time              `json:"timestamp"`
}

type LogAnalysis struct {
	ErrorCount     int      `json:"error_count"`
	CommonErrors   []string `json:"common_errors"`
	LastError      string   `json:"last_error"`
	ConnectionAttempts int   `json:"connection_attempts"`
}

type ConfigAnalysis struct {
	Valid          bool     `json:"valid"`
	MissingFields  []string `json:"missing_fields"`
	InvalidValues  []string `json:"invalid_values"`
	Suggestions    []string `json:"suggestions"`
}

type RepositoryAnalysis struct {
	HasReadme      bool     `json:"has_readme"`
	InstallSteps   []string `json:"install_steps"`
	Requirements   []string `json:"requirements"`
	CommonIssues   []string `json:"common_issues"`
}

// NewDiagnosticAgent creates a new diagnostic agent with AI capabilities
func NewDiagnosticAgent(logger *zap.Logger, llmConfig *config.LLMConfig) *DiagnosticAgent {
	return &DiagnosticAgent{
		logger:    logger,
		llmClient: NewLLMClientFromConfig(llmConfig),
	}
}

// LoadMemory reads the diagnostic memory file on startup
func (d *DiagnosticAgent) LoadMemory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	memoryPath := filepath.Join(homeDir, ".mcpproxy", "memory.md")
	content, err := os.ReadFile(memoryPath)
	if err != nil {
		if os.IsNotExist(err) {
			d.logger.Info("Memory file does not exist yet", zap.String("path", memoryPath))
			return "", nil
		}
		return "", fmt.Errorf("failed to read memory file: %w", err)
	}

	d.logger.Info("Loaded diagnostic memory", zap.String("path", memoryPath), zap.Int("size", len(content)))
	return string(content), nil
}

// AddMemory appends a new finding to the diagnostic memory file
func (d *DiagnosticAgent) AddMemory(finding string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	memoryPath := filepath.Join(homeDir, ".mcpproxy", "memory.md")

	// Read existing content
	content, err := os.ReadFile(memoryPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read memory file: %w", err)
	}

	// Append new finding with timestamp
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	newEntry := fmt.Sprintf("\n### %s\n%s\n", timestamp, finding)

	updatedContent := string(content) + newEntry

	// Write back to file
	if err := os.WriteFile(memoryPath, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("failed to write memory file: %w", err)
	}

	d.logger.Info("Added new memory entry", zap.String("timestamp", timestamp))
	return nil
}

// DiagnoseServer analyzes connection issues for a specific server
func (d *DiagnosticAgent) DiagnoseServer(serverName string) (*DiagnosticReport, error) {
	d.logger.Info("Starting diagnostic analysis", zap.String("server", serverName))

	// Load memory at the start of diagnosis
	memory, err := d.LoadMemory()
	if err != nil {
		d.logger.Warn("Failed to load diagnostic memory", zap.Error(err))
	} else if memory != "" {
		d.logger.Info("Using diagnostic memory for analysis")
	}

	report := &DiagnosticReport{
		ServerName: serverName,
		Timestamp:  time.Now(),
		Issues:     []string{},
		Recommendations: []string{},
	}

	// Load server configuration
	_, server, err := d.loadServerConfig(serverName)
	if err != nil {
		report.Issues = append(report.Issues, fmt.Sprintf("Failed to load server config: %v", err))
		return report, nil
	}

	// Analyze configuration
	report.ConfigAnalysis = d.analyzeConfiguration(server)
	
	// Analyze logs
	report.LogAnalysis = d.analyzeLogs(serverName)
	
	// Analyze repository if available
	if server.RepositoryURL != "" {
		report.RepoAnalysis = d.analyzeRepository(server.RepositoryURL)
	}

	// AI-powered analysis
	d.logger.Info("Running AI analysis", zap.String("server", serverName))
	aiAnalysis, err := d.performAIAnalysis(report, server)
	if err != nil {
		d.logger.Warn("AI analysis failed", zap.Error(err))
		report.AIAnalysis = fmt.Sprintf("AI analysis unavailable: %v", err)
	} else {
		report.AIAnalysis = aiAnalysis
	}

	// Generate AI-suggested configuration if issues found
	if len(report.Issues) > 0 {
		d.logger.Info("Generating AI configuration suggestions", zap.String("server", serverName))
		suggestedConfig, err := d.generateConfigSuggestions(server, report)
		if err != nil {
			d.logger.Warn("Config generation failed", zap.Error(err))
		} else {
			report.SuggestedConfig = suggestedConfig
		}
	}

	// Generate recommendations based on analysis
	d.generateRecommendations(report)

	return report, nil
}

// loadServerConfig loads the server configuration
func (d *DiagnosticAgent) loadServerConfig(serverName string) (*config.Config, *config.ServerConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, err
	}

	configPath := filepath.Join(homeDir, ".mcpproxy", "mcp_config.json")
	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		return nil, nil, err
	}

	for _, server := range cfg.Servers {
		if server.Name == serverName {
			return cfg, server, nil
		}
	}

	return nil, nil, fmt.Errorf("server %s not found in configuration", serverName)
}

// analyzeConfiguration checks the server configuration for issues
func (d *DiagnosticAgent) analyzeConfiguration(server *config.ServerConfig) ConfigAnalysis {
	analysis := ConfigAnalysis{
		Valid:         true,
		MissingFields: []string{},
		InvalidValues: []string{},
		Suggestions:   []string{},
	}

	// Check required fields
	if server.Name == "" {
		analysis.MissingFields = append(analysis.MissingFields, "name")
		analysis.Valid = false
	}

	if server.Protocol == "stdio" {
		if server.Command == "" {
			analysis.MissingFields = append(analysis.MissingFields, "command")
			analysis.Valid = false
		}
	} else {
		if server.URL == "" {
			analysis.MissingFields = append(analysis.MissingFields, "url")
			analysis.Valid = false
		}
	}

	// Check for common configuration issues
	if server.Protocol == "stdio" && strings.Contains(server.Command, "npx") {
		analysis.Suggestions = append(analysis.Suggestions, "Consider using 'npx -y' to auto-install packages")
	}

	if server.Protocol == "stdio" && strings.Contains(server.Command, "uvx") {
		analysis.Suggestions = append(analysis.Suggestions, "Ensure uvx is installed: 'pip install uv'")
	}

	return analysis
}

// analyzeLogs examines recent logs for the server
func (d *DiagnosticAgent) analyzeLogs(serverName string) LogAnalysis {
	analysis := LogAnalysis{
		ErrorCount:         0,
		CommonErrors:       []string{},
		ConnectionAttempts: 0,
	}

	// Read recent log files
	homeDir, _ := os.UserHomeDir()
	logPath := filepath.Join(homeDir, "Library", "Logs", "mcpproxy", "main.log")
	
	logContent, err := os.ReadFile(logPath)
	if err != nil {
		d.logger.Warn("Could not read log file", zap.Error(err))
		return analysis
	}

	lines := strings.Split(string(logContent), "\n")
	
	// Analyze last 1000 lines for this server
	serverLines := []string{}
	for i := len(lines) - 1; i >= 0 && len(serverLines) < 1000; i-- {
		if strings.Contains(lines[i], serverName) {
			serverLines = append([]string{lines[i]}, serverLines...)
		}
	}

	// Count errors and connection attempts
	for _, line := range serverLines {
		if strings.Contains(line, "ERROR") {
			analysis.ErrorCount++
			if analysis.LastError == "" {
				analysis.LastError = d.extractErrorMessage(line)
			}
		}
		if strings.Contains(line, "Connecting to upstream") {
			analysis.ConnectionAttempts++
		}
	}

	// Identify common error patterns
	analysis.CommonErrors = d.identifyCommonErrors(serverLines)

	return analysis
}

// analyzeRepository fetches and analyzes repository documentation
func (d *DiagnosticAgent) analyzeRepository(repoURL string) RepositoryAnalysis {
	analysis := RepositoryAnalysis{
		HasReadme:    false,
		InstallSteps: []string{},
		Requirements: []string{},
		CommonIssues: []string{},
	}

	// Convert GitHub URL to raw README URL
	readmeURL := d.convertToReadmeURL(repoURL)
	if readmeURL == "" {
		return analysis
	}

	// Fetch README content
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", readmeURL, nil)
	if err != nil {
		return analysis
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return analysis
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return analysis
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return analysis
	}

	analysis.HasReadme = true
	readmeText := string(content)

	// Extract installation steps
	analysis.InstallSteps = d.extractInstallSteps(readmeText)
	
	// Extract requirements
	analysis.Requirements = d.extractRequirements(readmeText)
	
	// Extract common issues
	analysis.CommonIssues = d.extractCommonIssues(readmeText)

	return analysis
}

// Helper functions
func (d *DiagnosticAgent) extractErrorMessage(logLine string) string {
	// Extract error message from log line
	parts := strings.Split(logLine, "|")
	if len(parts) > 3 {
		return strings.TrimSpace(parts[len(parts)-1])
	}
	return logLine
}

func (d *DiagnosticAgent) identifyCommonErrors(lines []string) []string {
	errorPatterns := map[string]string{
		"command not found":     "Command not found - check if the command is installed",
		"permission denied":     "Permission denied - check file permissions",
		"connection refused":    "Connection refused - check if service is running",
		"timeout":              "Connection timeout - check network connectivity",
		"module not found":     "Module not found - check if dependencies are installed",
	}

	found := []string{}
	for _, line := range lines {
		for pattern, description := range errorPatterns {
			if strings.Contains(strings.ToLower(line), pattern) {
				found = append(found, description)
				break
			}
		}
	}

	return d.removeDuplicates(found)
}

func (d *DiagnosticAgent) convertToReadmeURL(repoURL string) string {
	// Convert GitHub repo URL to raw README URL
	if strings.Contains(repoURL, "github.com") {
		repoURL = strings.Replace(repoURL, "github.com", "raw.githubusercontent.com", 1)
		repoURL = strings.TrimSuffix(repoURL, "/")
		return repoURL + "/main/README.md"
	}
	return ""
}

func (d *DiagnosticAgent) extractInstallSteps(readme string) []string {
	steps := []string{}
	
	// Look for installation sections
	installRegex := regexp.MustCompile(`(?i)##?\s*(install|setup|getting started).*?\n(.*?)(?=##|\z)`)
	matches := installRegex.FindAllStringSubmatch(readme, -1)
	
	for _, match := range matches {
		if len(match) > 2 {
			// Extract code blocks and commands
			codeRegex := regexp.MustCompile("```[^`]*```|`[^`]+`")
			codes := codeRegex.FindAllString(match[2], -1)
			for _, code := range codes {
				cleaned := strings.Trim(code, "`")
				if strings.Contains(cleaned, "npm") || strings.Contains(cleaned, "pip") || strings.Contains(cleaned, "install") {
					steps = append(steps, cleaned)
				}
			}
		}
	}
	
	return steps
}

func (d *DiagnosticAgent) extractRequirements(readme string) []string {
	requirements := []string{}
	
	// Look for requirements sections
	reqRegex := regexp.MustCompile(`(?i)##?\s*(requirements|prerequisites|dependencies).*?\n(.*?)(?=##|\z)`)
	matches := reqRegex.FindAllStringSubmatch(readme, -1)
	
	for _, match := range matches {
		if len(match) > 2 {
			lines := strings.Split(match[2], "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
					requirements = append(requirements, strings.TrimSpace(line[1:]))
				}
			}
		}
	}
	
	return requirements
}

func (d *DiagnosticAgent) extractCommonIssues(readme string) []string {
	issues := []string{}
	
	// Look for troubleshooting or issues sections
	issueRegex := regexp.MustCompile(`(?i)##?\s*(troubleshooting|issues|problems|faq).*?\n(.*?)(?=##|\z)`)
	matches := issueRegex.FindAllStringSubmatch(readme, -1)
	
	for _, match := range matches {
		if len(match) > 2 {
			lines := strings.Split(match[2], "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") || strings.HasPrefix(line, "Q:") {
					issues = append(issues, strings.TrimSpace(line))
				}
			}
		}
	}
	
	return issues
}

func (d *DiagnosticAgent) generateRecommendations(report *DiagnosticReport) {
	// Generate recommendations based on analysis
	if !report.ConfigAnalysis.Valid {
		report.Recommendations = append(report.Recommendations, "Fix configuration issues: "+strings.Join(report.ConfigAnalysis.MissingFields, ", "))
	}

	if report.LogAnalysis.ErrorCount > 0 {
		report.Recommendations = append(report.Recommendations, "Check recent errors in logs")
	}

	if len(report.LogAnalysis.CommonErrors) > 0 {
		report.Recommendations = append(report.Recommendations, "Address common errors: "+strings.Join(report.LogAnalysis.CommonErrors, "; "))
	}

	if report.RepoAnalysis.HasReadme && len(report.RepoAnalysis.InstallSteps) > 0 {
		report.Recommendations = append(report.Recommendations, "Follow installation steps from repository documentation")
	}

	if len(report.RepoAnalysis.Requirements) > 0 {
		report.Recommendations = append(report.Recommendations, "Ensure all requirements are met: "+strings.Join(report.RepoAnalysis.Requirements, ", "))
	}
}

func (d *DiagnosticAgent) removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	result := []string{}
	
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

// ShowDiagnosticReport displays the diagnostic report in a user-friendly format
func (d *DiagnosticAgent) ShowDiagnosticReport(report *DiagnosticReport) {
	fmt.Printf("\n🔍 MCP Server Diagnostic Report for: %s\n", report.ServerName)
	fmt.Printf("Generated: %s\n\n", report.Timestamp.Format("2006-01-02 15:04:05"))

	// Configuration Analysis
	fmt.Printf("📋 Configuration Analysis:\n")
	if report.ConfigAnalysis.Valid {
		fmt.Printf("  ✅ Configuration is valid\n")
	} else {
		fmt.Printf("  ❌ Configuration has issues:\n")
		for _, field := range report.ConfigAnalysis.MissingFields {
			fmt.Printf("    - Missing field: %s\n", field)
		}
		for _, value := range report.ConfigAnalysis.InvalidValues {
			fmt.Printf("    - Invalid value: %s\n", value)
		}
	}
	
	if len(report.ConfigAnalysis.Suggestions) > 0 {
		fmt.Printf("  💡 Suggestions:\n")
		for _, suggestion := range report.ConfigAnalysis.Suggestions {
			fmt.Printf("    - %s\n", suggestion)
		}
	}

	// Log Analysis
	fmt.Printf("\n📊 Log Analysis:\n")
	fmt.Printf("  - Error count: %d\n", report.LogAnalysis.ErrorCount)
	fmt.Printf("  - Connection attempts: %d\n", report.LogAnalysis.ConnectionAttempts)
	
	if report.LogAnalysis.LastError != "" {
		fmt.Printf("  - Last error: %s\n", report.LogAnalysis.LastError)
	}
	
	if len(report.LogAnalysis.CommonErrors) > 0 {
		fmt.Printf("  - Common errors:\n")
		for _, error := range report.LogAnalysis.CommonErrors {
			fmt.Printf("    - %s\n", error)
		}
	}

	// Repository Analysis
	fmt.Printf("\n📚 Repository Analysis:\n")
	if report.RepoAnalysis.HasReadme {
		fmt.Printf("  ✅ README found\n")
		
		if len(report.RepoAnalysis.InstallSteps) > 0 {
			fmt.Printf("  📦 Installation steps:\n")
			for _, step := range report.RepoAnalysis.InstallSteps {
				fmt.Printf("    - %s\n", step)
			}
		}
		
		if len(report.RepoAnalysis.Requirements) > 0 {
			fmt.Printf("  📋 Requirements:\n")
			for _, req := range report.RepoAnalysis.Requirements {
				fmt.Printf("    - %s\n", req)
			}
		}
		
		if len(report.RepoAnalysis.CommonIssues) > 0 {
			fmt.Printf("  ⚠️  Known issues:\n")
			for _, issue := range report.RepoAnalysis.CommonIssues {
				fmt.Printf("    - %s\n", issue)
			}
		}
	} else {
		fmt.Printf("  ❌ No README found or accessible\n")
	}

	// Recommendations
	if len(report.Recommendations) > 0 {
		fmt.Printf("\n💡 Recommendations:\n")
		for i, rec := range report.Recommendations {
			fmt.Printf("  %d. %s\n", i+1, rec)
		}
	}

	fmt.Printf("\n" + strings.Repeat("-", 60) + "\n")
}
// performAIAnalysis uses LLM to analyze server issues intelligently
func (d *DiagnosticAgent) performAIAnalysis(report *DiagnosticReport, server *config.ServerConfig) (string, error) {
	// Gather documentation if available
	documentation := ""
	if server.RepositoryURL != "" {
		doc, err := d.fetchDocumentation(server.RepositoryURL)
		if err == nil {
			documentation = doc
		}
	}

	// Create comprehensive analysis prompt
	prompt := fmt.Sprintf(`
Analyze this MCP server configuration and diagnostic data:

SERVER: %s
COMMAND: %s
ARGS: %v
URL: %s
ENABLED: %t

ISSUES FOUND:
%s

LOG ANALYSIS:
- Error Count: %d
- Common Errors: %v
- Last Error: %s

CONFIG ANALYSIS:
- Valid: %t
- Missing Fields: %v
- Invalid Values: %v

DOCUMENTATION:
%s

Please provide:
1. Root cause analysis of the issues
2. Step-by-step troubleshooting recommendations
3. Configuration fixes needed
4. Best practices for this server type

Focus on actionable solutions that can be implemented immediately.
`, 
		server.Name,
		server.Command,
		server.Args,
		server.URL,
		server.StartupMode == "active",
		fmt.Sprintf("%v", report.Issues),
		report.LogAnalysis.ErrorCount,
		report.LogAnalysis.CommonErrors,
		report.LogAnalysis.LastError,
		report.ConfigAnalysis.Valid,
		report.ConfigAnalysis.MissingFields,
		report.ConfigAnalysis.InvalidValues,
		documentation,
	)

	return d.llmClient.Analyze(prompt)
}

// generateConfigSuggestions uses AI to generate corrected configuration
func (d *DiagnosticAgent) generateConfigSuggestions(server *config.ServerConfig, report *DiagnosticReport) (string, error) {
	// Get current config as JSON
	currentConfigJSON, err := json.MarshalIndent(server, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal current config: %w", err)
	}

	// Fetch documentation
	documentation := ""
	if server.RepositoryURL != "" {
		doc, err := d.fetchDocumentation(server.RepositoryURL)
		if err == nil {
			documentation = doc
		}
	}

	return d.llmClient.GenerateConfig(server.Name, documentation, string(currentConfigJSON))
}

// fetchDocumentation retrieves documentation from repository
func (d *DiagnosticAgent) fetchDocumentation(repoURL string) (string, error) {
	// Convert GitHub repo URL to raw README URL
	readmeURL := d.convertToReadmeURL(repoURL)
	if readmeURL == "" {
		return "", fmt.Errorf("unsupported repository URL format")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(readmeURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch documentation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("documentation not found (status: %d)", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read documentation: %w", err)
	}

	return string(body), nil
}
