//go:build !nogui && !headless && !linux

package tray

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"
)

// DocAnalyzerAgent analyzes documentation and installation guides
type DocAnalyzerAgent struct {
	logger *zap.Logger
}

// DocumentationAnalysis contains the results of documentation analysis
type DocumentationAnalysis struct {
	HasReadme        bool                   `json:"has_readme"`
	InstallSteps     []InstallationStep     `json:"install_steps"`
	Requirements     []Requirement          `json:"requirements"`
	ConfigExamples   []ConfigExample        `json:"config_examples"`
	TroubleShooting  []TroubleshootingTip   `json:"troubleshooting"`
	APIDocumentation []APIEndpoint          `json:"api_docs"`
	Summary          string                 `json:"summary"`
}

// InstallationStep represents a single installation step
type InstallationStep struct {
	StepNumber  int    `json:"step_number"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
	Platform    string `json:"platform,omitempty"`
}

// Requirement represents a system requirement
type Requirement struct {
	Type        string `json:"type"`        // "software", "hardware", "environment"
	Name        string `json:"name"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// ConfigExample represents a configuration example
type ConfigExample struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Format      string `json:"format"` // "json", "yaml", "env"
}

// TroubleshootingTip represents a troubleshooting tip
type TroubleshootingTip struct {
	Problem  string `json:"problem"`
	Solution string `json:"solution"`
	Commands []string `json:"commands,omitempty"`
}

// APIEndpoint represents an API endpoint documentation
type APIEndpoint struct {
	Path        string `json:"path"`
	Method      string `json:"method"`
	Description string `json:"description"`
	Parameters  []string `json:"parameters,omitempty"`
}

// NewDocAnalyzerAgent creates a new documentation analyzer agent
func NewDocAnalyzerAgent(logger *zap.Logger) *DocAnalyzerAgent {
	return &DocAnalyzerAgent{
		logger: logger,
	}
}

// ProcessMessage processes a message requesting documentation analysis
func (da *DocAnalyzerAgent) ProcessMessage(ctx context.Context, message ChatMessage, session *ChatSession) (*ChatMessage, error) {
	da.logger.Info("Documentation analyzer processing message",
		zap.String("session_id", session.ID),
		zap.String("server", session.ServerName))

	content := strings.ToLower(message.Content)

	var response string
	var metadata map[string]interface{}

	switch {
	case da.containsKeywords(content, []string{"installation", "install", "setup", "how to install"}):
		response, metadata = da.analyzeInstallationGuide(session.ServerName, message.Content)

	case da.containsKeywords(content, []string{"requirements", "dependencies", "prerequisites"}):
		response, metadata = da.analyzeRequirements(session.ServerName)

	case da.containsKeywords(content, []string{"configuration", "config", "setup config", "configure"}):
		response, metadata = da.analyzeConfigurationGuide(session.ServerName)

	case da.containsKeywords(content, []string{"troubleshooting", "problems", "issues", "faq"}):
		response, metadata = da.analyzeTroubleshootingGuide(session.ServerName)

	case da.containsKeywords(content, []string{"api", "endpoints", "tools", "methods"}):
		response, metadata = da.analyzeAPIDocumentation(session.ServerName)

	case da.containsKeywords(content, []string{"documentation", "docs", "readme", "guide"}):
		response, metadata = da.analyzeFullDocumentation(session.ServerName)

	default:
		response, metadata = da.provideDocAnalysisGuidance(session.ServerName)
	}

	return &ChatMessage{
		ID:        generateMessageID(),
		Role:      "assistant",
		Content:   response,
		AgentType: string(AgentTypeDocAnalyzer),
		Timestamp: time.Now(),
		Metadata:  metadata,
	}, nil
}

// GetCapabilities returns the capabilities of the documentation analyzer agent
func (da *DocAnalyzerAgent) GetCapabilities() []string {
	return []string{
		"Installation guide analysis",
		"Requirements extraction",
		"Configuration examples",
		"Troubleshooting tips",
		"API documentation analysis",
		"README parsing",
		"Setup validation",
	}
}

// GetAgentType returns the agent type
func (da *DocAnalyzerAgent) GetAgentType() AgentType {
	return AgentTypeDocAnalyzer
}

// CanHandle determines if this agent can handle a message
func (da *DocAnalyzerAgent) CanHandle(message ChatMessage) bool {
	content := strings.ToLower(message.Content)
	keywords := []string{
		"documentation", "docs", "readme", "install", "installation",
		"setup", "guide", "requirements", "dependencies", "config",
		"configuration", "troubleshooting", "api", "endpoints",
	}

	return da.containsKeywords(content, keywords)
}

// containsKeywords checks if content contains any of the specified keywords
func (da *DocAnalyzerAgent) containsKeywords(content string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

// analyzeInstallationGuide analyzes installation documentation
func (da *DocAnalyzerAgent) analyzeInstallationGuide(serverName, userQuery string) (string, map[string]interface{}) {
	da.logger.Info("Analyzing installation guide", zap.String("server", serverName))

	// Get server configuration to find repository URL
	repoURL := da.getRepositoryURL(serverName)
	if repoURL == "" {
		return da.provideGenericInstallationHelp(serverName), map[string]interface{}{
			"has_repo": false,
		}
	}

	// Fetch and analyze documentation
	analysis, err := da.fetchAndAnalyzeDocumentation(repoURL)
	if err != nil {
		da.logger.Warn("Failed to fetch documentation", zap.Error(err))
		return da.provideGenericInstallationHelp(serverName), map[string]interface{}{
			"error": err.Error(),
		}
	}

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("📖 **Installation Guide for %s**\n\n", serverName))

	if len(analysis.InstallSteps) == 0 {
		responseBuilder.WriteString("❌ **No installation steps found in documentation.**\n\n")
		responseBuilder.WriteString("Let me provide generic installation guidance:\n\n")
		responseBuilder.WriteString(da.provideGenericInstallationHelp(serverName))
	} else {
		responseBuilder.WriteString("✅ **Found installation instructions:**\n\n")

		for i, step := range analysis.InstallSteps {
			if i >= 10 { // Limit to 10 steps
				break
			}
			responseBuilder.WriteString(fmt.Sprintf("**Step %d: %s**\n", step.StepNumber, step.Title))
			if step.Description != "" {
				responseBuilder.WriteString(fmt.Sprintf("%s\n", step.Description))
			}
			if step.Command != "" {
				responseBuilder.WriteString(fmt.Sprintf("```bash\n%s\n```\n", step.Command))
			}
			responseBuilder.WriteString("\n")
		}

		if len(analysis.Requirements) > 0 {
			responseBuilder.WriteString("📋 **Requirements:**\n")
			for _, req := range analysis.Requirements {
				if req.Required {
					responseBuilder.WriteString(fmt.Sprintf("• **%s**: %s", req.Name, req.Description))
					if req.Version != "" {
						responseBuilder.WriteString(fmt.Sprintf(" (version %s)", req.Version))
					}
					responseBuilder.WriteString("\n")
				}
			}
			responseBuilder.WriteString("\n")
		}
	}

	responseBuilder.WriteString("💡 **Need help with a specific step?** Just ask me about it!")

	return responseBuilder.String(), map[string]interface{}{
		"analysis":     analysis,
		"install_steps": analysis.InstallSteps,
		"requirements": analysis.Requirements,
	}
}

// analyzeRequirements analyzes system requirements
func (da *DocAnalyzerAgent) analyzeRequirements(serverName string) (string, map[string]interface{}) {
	da.logger.Info("Analyzing requirements", zap.String("server", serverName))

	repoURL := da.getRepositoryURL(serverName)
	if repoURL == "" {
		return da.provideGenericRequirements(serverName), map[string]interface{}{
			"has_repo": false,
		}
	}

	analysis, err := da.fetchAndAnalyzeDocumentation(repoURL)
	if err != nil {
		return da.provideGenericRequirements(serverName), map[string]interface{}{
			"error": err.Error(),
		}
	}

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("📋 **Requirements for %s**\n\n", serverName))

	if len(analysis.Requirements) == 0 {
		responseBuilder.WriteString("❌ **No specific requirements found in documentation.**\n\n")
		responseBuilder.WriteString(da.provideGenericRequirements(serverName))
	} else {
		// Categorize requirements
		required := []Requirement{}
		optional := []Requirement{}

		for _, req := range analysis.Requirements {
			if req.Required {
				required = append(required, req)
			} else {
				optional = append(optional, req)
			}
		}

		if len(required) > 0 {
			responseBuilder.WriteString("✅ **Required Dependencies:**\n")
			for _, req := range required {
				responseBuilder.WriteString(fmt.Sprintf("• **%s**", req.Name))
				if req.Version != "" {
					responseBuilder.WriteString(fmt.Sprintf(" (v%s)", req.Version))
				}
				if req.Description != "" {
					responseBuilder.WriteString(fmt.Sprintf(" - %s", req.Description))
				}
				responseBuilder.WriteString("\n")
			}
			responseBuilder.WriteString("\n")
		}

		if len(optional) > 0 {
			responseBuilder.WriteString("🔧 **Optional Dependencies:**\n")
			for _, req := range optional {
				responseBuilder.WriteString(fmt.Sprintf("• **%s**", req.Name))
				if req.Version != "" {
					responseBuilder.WriteString(fmt.Sprintf(" (v%s)", req.Version))
				}
				if req.Description != "" {
					responseBuilder.WriteString(fmt.Sprintf(" - %s", req.Description))
				}
				responseBuilder.WriteString("\n")
			}
			responseBuilder.WriteString("\n")
		}
	}

	responseBuilder.WriteString("💡 **Want help installing any of these?** Just ask!")

	return responseBuilder.String(), map[string]interface{}{
		"requirements": analysis.Requirements,
	}
}

// analyzeConfigurationGuide analyzes configuration documentation
func (da *DocAnalyzerAgent) analyzeConfigurationGuide(serverName string) (string, map[string]interface{}) {
	da.logger.Info("Analyzing configuration guide", zap.String("server", serverName))

	repoURL := da.getRepositoryURL(serverName)
	if repoURL == "" {
		return da.provideGenericConfigurationHelp(serverName), map[string]interface{}{
			"has_repo": false,
		}
	}

	analysis, err := da.fetchAndAnalyzeDocumentation(repoURL)
	if err != nil {
		return da.provideGenericConfigurationHelp(serverName), map[string]interface{}{
			"error": err.Error(),
		}
	}

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("⚙️ **Configuration Guide for %s**\n\n", serverName))

	if len(analysis.ConfigExamples) == 0 {
		responseBuilder.WriteString("❌ **No configuration examples found in documentation.**\n\n")
		responseBuilder.WriteString(da.provideGenericConfigurationHelp(serverName))
	} else {
		responseBuilder.WriteString("✅ **Found configuration examples:**\n\n")

		for i, config := range analysis.ConfigExamples {
			if i >= 5 { // Limit to 5 examples
				break
			}
			responseBuilder.WriteString(fmt.Sprintf("**%s**\n", config.Title))
			if config.Description != "" {
				responseBuilder.WriteString(fmt.Sprintf("%s\n\n", config.Description))
			}
			responseBuilder.WriteString(fmt.Sprintf("```%s\n%s\n```\n\n", config.Format, config.Content))
		}
	}

	responseBuilder.WriteString("💡 **Need help with your specific configuration?** Share your config and I'll help!")

	return responseBuilder.String(), map[string]interface{}{
		"config_examples": analysis.ConfigExamples,
	}
}

// analyzeTroubleshootingGuide analyzes troubleshooting documentation
func (da *DocAnalyzerAgent) analyzeTroubleshootingGuide(serverName string) (string, map[string]interface{}) {
	da.logger.Info("Analyzing troubleshooting guide", zap.String("server", serverName))

	repoURL := da.getRepositoryURL(serverName)
	if repoURL == "" {
		return da.provideGenericTroubleshooting(serverName), map[string]interface{}{
			"has_repo": false,
		}
	}

	analysis, err := da.fetchAndAnalyzeDocumentation(repoURL)
	if err != nil {
		return da.provideGenericTroubleshooting(serverName), map[string]interface{}{
			"error": err.Error(),
		}
	}

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("🔧 **Troubleshooting Guide for %s**\n\n", serverName))

	if len(analysis.TroubleShooting) == 0 {
		responseBuilder.WriteString("❌ **No troubleshooting guide found in documentation.**\n\n")
		responseBuilder.WriteString(da.provideGenericTroubleshooting(serverName))
	} else {
		responseBuilder.WriteString("✅ **Common Issues and Solutions:**\n\n")

		for i, tip := range analysis.TroubleShooting {
			if i >= 10 { // Limit to 10 tips
				break
			}
			responseBuilder.WriteString(fmt.Sprintf("**Problem**: %s\n", tip.Problem))
			responseBuilder.WriteString(fmt.Sprintf("**Solution**: %s\n", tip.Solution))

			if len(tip.Commands) > 0 {
				responseBuilder.WriteString("**Commands**:\n")
				for _, cmd := range tip.Commands {
					responseBuilder.WriteString(fmt.Sprintf("```bash\n%s\n```\n", cmd))
				}
			}
			responseBuilder.WriteString("\n")
		}
	}

	responseBuilder.WriteString("💡 **Have a specific issue?** Describe it and I'll help you troubleshoot!")

	return responseBuilder.String(), map[string]interface{}{
		"troubleshooting": analysis.TroubleShooting,
	}
}

// analyzeAPIDocumentation analyzes API documentation
func (da *DocAnalyzerAgent) analyzeAPIDocumentation(serverName string) (string, map[string]interface{}) {
	da.logger.Info("Analyzing API documentation", zap.String("server", serverName))

	repoURL := da.getRepositoryURL(serverName)
	if repoURL == "" {
		return "❌ **No repository URL found for this server.**\n\nI can't analyze API documentation without access to the project repository.", map[string]interface{}{
			"has_repo": false,
		}
	}

	analysis, err := da.fetchAndAnalyzeDocumentation(repoURL)
	if err != nil {
		return fmt.Sprintf("❌ **Failed to fetch documentation**: %v", err), map[string]interface{}{
			"error": err.Error(),
		}
	}

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("📚 **API Documentation for %s**\n\n", serverName))

	if len(analysis.APIDocumentation) == 0 {
		responseBuilder.WriteString("❌ **No API documentation found.**\n\n")
		responseBuilder.WriteString("The documentation doesn't contain specific API endpoint information. ")
		responseBuilder.WriteString("You can try testing the server to see what tools are available.\n\n")
		responseBuilder.WriteString("💡 **Try asking**: \"Test my server tools\" or \"List available tools\"")
	} else {
		responseBuilder.WriteString("✅ **Available API Endpoints:**\n\n")

		for i, endpoint := range analysis.APIDocumentation {
			if i >= 15 { // Limit to 15 endpoints
				break
			}
			responseBuilder.WriteString(fmt.Sprintf("**%s %s**\n", endpoint.Method, endpoint.Path))
			if endpoint.Description != "" {
				responseBuilder.WriteString(fmt.Sprintf("%s\n", endpoint.Description))
			}
			if len(endpoint.Parameters) > 0 {
				responseBuilder.WriteString("Parameters: " + strings.Join(endpoint.Parameters, ", ") + "\n")
			}
			responseBuilder.WriteString("\n")
		}
	}

	return responseBuilder.String(), map[string]interface{}{
		"api_docs": analysis.APIDocumentation,
	}
}

// analyzeFullDocumentation performs comprehensive documentation analysis
func (da *DocAnalyzerAgent) analyzeFullDocumentation(serverName string) (string, map[string]interface{}) {
	da.logger.Info("Performing full documentation analysis", zap.String("server", serverName))

	repoURL := da.getRepositoryURL(serverName)
	if repoURL == "" {
		return da.provideGenericDocumentationHelp(serverName), map[string]interface{}{
			"has_repo": false,
		}
	}

	analysis, err := da.fetchAndAnalyzeDocumentation(repoURL)
	if err != nil {
		return fmt.Sprintf("❌ **Failed to analyze documentation**: %v", err), map[string]interface{}{
			"error": err.Error(),
		}
	}

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("📊 **Complete Documentation Analysis for %s**\n\n", serverName))

	// Summary
	responseBuilder.WriteString("**📋 Summary:**\n")
	responseBuilder.WriteString(fmt.Sprintf("• Documentation available: %t\n", analysis.HasReadme))
	responseBuilder.WriteString(fmt.Sprintf("• Installation steps: %d\n", len(analysis.InstallSteps)))
	responseBuilder.WriteString(fmt.Sprintf("• Requirements: %d\n", len(analysis.Requirements)))
	responseBuilder.WriteString(fmt.Sprintf("• Configuration examples: %d\n", len(analysis.ConfigExamples)))
	responseBuilder.WriteString(fmt.Sprintf("• Troubleshooting tips: %d\n", len(analysis.TroubleShooting)))
	responseBuilder.WriteString(fmt.Sprintf("• API endpoints: %d\n\n", len(analysis.APIDocumentation)))

	// Analysis summary
	if analysis.Summary != "" {
		responseBuilder.WriteString(fmt.Sprintf("**🔍 Analysis**: %s\n\n", analysis.Summary))
	}

	responseBuilder.WriteString("**📖 Available Sections:**\n")
	if len(analysis.InstallSteps) > 0 {
		responseBuilder.WriteString("• Installation guide ✅\n")
	}
	if len(analysis.Requirements) > 0 {
		responseBuilder.WriteString("• Requirements list ✅\n")
	}
	if len(analysis.ConfigExamples) > 0 {
		responseBuilder.WriteString("• Configuration examples ✅\n")
	}
	if len(analysis.TroubleShooting) > 0 {
		responseBuilder.WriteString("• Troubleshooting guide ✅\n")
	}
	if len(analysis.APIDocumentation) > 0 {
		responseBuilder.WriteString("• API documentation ✅\n")
	}

	responseBuilder.WriteString("\n💡 **Ask me about any specific section!**")

	return responseBuilder.String(), map[string]interface{}{
		"full_analysis": analysis,
	}
}

// provideDocAnalysisGuidance provides guidance on documentation analysis options
func (da *DocAnalyzerAgent) provideDocAnalysisGuidance(serverName string) (string, map[string]interface{}) {
	response := fmt.Sprintf(`📚 **Documentation Analysis for %s**

I can help you understand the documentation for this MCP server:

**🔍 Available Analysis Types:**

**1. Installation Guide** - "How do I install this?"
• Step-by-step installation instructions
• Platform-specific guidance

**2. Requirements** - "What are the requirements?"
• System dependencies
• Software prerequisites

**3. Configuration** - "How do I configure this?"
• Configuration examples
• Settings explanations

**4. Troubleshooting** - "Help with problems"
• Common issues and solutions
• FAQ and known problems

**5. API Documentation** - "What tools are available?"
• Available endpoints/tools
• Usage examples

**6. Complete Analysis** - "Analyze all documentation"
• Comprehensive overview
• All available sections

**💡 Example Requests:**
• "Show me the installation steps"
• "What are the requirements?"
• "How do I configure this server?"
• "What troubleshooting options are there?"
• "Analyze all the documentation"

What would you like me to help you with?`, serverName)

	return response, map[string]interface{}{
		"guidance_type": "documentation_analysis",
		"available_analyses": []string{
			"installation",
			"requirements",
			"configuration",
			"troubleshooting",
			"api_docs",
			"full_analysis",
		},
	}
}

// getRepositoryURL gets the repository URL for a server (placeholder)
func (da *DocAnalyzerAgent) getRepositoryURL(serverName string) string {
	// This would integrate with the server configuration to get the actual repository URL
	// For now, return empty string to simulate missing repository info
	return ""
}

// fetchAndAnalyzeDocumentation fetches and analyzes documentation from a repository
func (da *DocAnalyzerAgent) fetchAndAnalyzeDocumentation(repoURL string) (*DocumentationAnalysis, error) {
	// Convert to raw README URL if it's a GitHub repo
	readmeURL := da.convertToReadmeURL(repoURL)

	// Fetch README content
	content, err := da.fetchDocumentationContent(readmeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch documentation: %w", err)
	}

	// Analyze the content
	analysis := &DocumentationAnalysis{
		HasReadme: true,
	}

	analysis.InstallSteps = da.extractInstallationSteps(content)
	analysis.Requirements = da.extractRequirements(content)
	analysis.ConfigExamples = da.extractConfigurationExamples(content)
	analysis.TroubleShooting = da.extractTroubleshootingTips(content)
	analysis.APIDocumentation = da.extractAPIDocumentation(content)
	analysis.Summary = da.generateDocumentationSummary(analysis)

	return analysis, nil
}

// convertToReadmeURL converts a repository URL to a README URL
func (da *DocAnalyzerAgent) convertToReadmeURL(repoURL string) string {
	if strings.Contains(repoURL, "github.com") {
		repoURL = strings.Replace(repoURL, "github.com", "raw.githubusercontent.com", 1)
		repoURL = strings.TrimSuffix(repoURL, "/")
		return repoURL + "/main/README.md"
	}
	return repoURL
}

// fetchDocumentationContent fetches documentation content from a URL
func (da *DocAnalyzerAgent) fetchDocumentationContent(url string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// extractInstallationSteps extracts installation steps from documentation
func (da *DocAnalyzerAgent) extractInstallationSteps(content string) []InstallationStep {
	var steps []InstallationStep

	// Look for installation sections
	installRegex := regexp.MustCompile(`(?i)##?\s*(install|setup|getting started).*?\n(.*?)(?=##|\z)`)
	matches := installRegex.FindAllStringSubmatch(content, -1)

	stepNumber := 1
	for _, match := range matches {
		if len(match) > 2 {
			sectionContent := match[2]

			// Extract numbered or bullet steps
			stepRegex := regexp.MustCompile(`(?m)^[\s]*(?:\d+\.|\*|\-)\s*(.+)`)
			stepMatches := stepRegex.FindAllStringSubmatch(sectionContent, -1)

			for _, stepMatch := range stepMatches {
				if len(stepMatch) > 1 {
					stepText := strings.TrimSpace(stepMatch[1])

					// Extract command if it looks like one
					var command string
					if strings.Contains(stepText, "`") {
						cmdRegex := regexp.MustCompile("`([^`]+)`")
						cmdMatch := cmdRegex.FindStringSubmatch(stepText)
						if len(cmdMatch) > 1 {
							command = cmdMatch[1]
						}
					}

					steps = append(steps, InstallationStep{
						StepNumber:  stepNumber,
						Title:       fmt.Sprintf("Step %d", stepNumber),
						Description: stepText,
						Command:     command,
					})
					stepNumber++
				}
			}
		}
	}

	return steps
}

// extractRequirements extracts requirements from documentation
func (da *DocAnalyzerAgent) extractRequirements(content string) []Requirement {
	var requirements []Requirement

	// Look for requirements sections
	reqRegex := regexp.MustCompile(`(?i)##?\s*(requirements|prerequisites|dependencies).*?\n(.*?)(?=##|\z)`)
	matches := reqRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 2 {
			sectionContent := match[2]

			// Extract individual requirements
			reqItemRegex := regexp.MustCompile(`(?m)^[\s]*(?:\*|\-)\s*(.+)`)
			reqMatches := reqItemRegex.FindAllStringSubmatch(sectionContent, -1)

			for _, reqMatch := range reqMatches {
				if len(reqMatch) > 1 {
					reqText := strings.TrimSpace(reqMatch[1])

					// Try to extract name and version
					var name, version string
					if strings.Contains(reqText, ">=") || strings.Contains(reqText, "version") {
						parts := strings.Fields(reqText)
						if len(parts) > 0 {
							name = parts[0]
							for _, part := range parts[1:] {
								if regexp.MustCompile(`\d+\.\d+`).MatchString(part) {
									version = part
									break
								}
							}
						}
					} else {
						name = reqText
					}

					requirements = append(requirements, Requirement{
						Type:        "software",
						Name:        name,
						Version:     version,
						Description: reqText,
						Required:    true,
					})
				}
			}
		}
	}

	return requirements
}

// extractConfigurationExamples extracts configuration examples from documentation
func (da *DocAnalyzerAgent) extractConfigurationExamples(content string) []ConfigExample {
	var examples []ConfigExample

	// Look for configuration sections
	configRegex := regexp.MustCompile(`(?i)##?\s*(configuration|config|setup).*?\n(.*?)(?=##|\z)`)
	matches := configRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 2 {
			sectionContent := match[2]

			// Extract code blocks
			codeRegex := regexp.MustCompile("```(\\w*)\\n([^`]+)```")
			codeMatches := codeRegex.FindAllStringSubmatch(sectionContent, -1)

			for i, codeMatch := range codeMatches {
				if len(codeMatch) > 2 {
					format := "text"
					if codeMatch[1] != "" {
						format = codeMatch[1]
					}

					examples = append(examples, ConfigExample{
						Title:       fmt.Sprintf("Configuration Example %d", i+1),
						Description: "Configuration example from documentation",
						Content:     strings.TrimSpace(codeMatch[2]),
						Format:      format,
					})
				}
			}
		}
	}

	return examples
}

// extractTroubleshootingTips extracts troubleshooting information from documentation
func (da *DocAnalyzerAgent) extractTroubleshootingTips(content string) []TroubleshootingTip {
	var tips []TroubleshootingTip

	// Look for troubleshooting sections
	troubleRegex := regexp.MustCompile(`(?i)##?\s*(troubleshooting|issues|problems|faq).*?\n(.*?)(?=##|\z)`)
	matches := troubleRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 2 {
			sectionContent := match[2]

			// Extract Q&A or problem-solution pairs
			qaRegex := regexp.MustCompile(`(?m)^[\s]*(?:\*|\-)\s*(.+)`)
			qaMatches := qaRegex.FindAllStringSubmatch(sectionContent, -1)

			for _, qaMatch := range qaMatches {
				if len(qaMatch) > 1 {
					text := strings.TrimSpace(qaMatch[1])

					tips = append(tips, TroubleshootingTip{
						Problem:  "Common Issue",
						Solution: text,
					})
				}
			}
		}
	}

	return tips
}

// extractAPIDocumentation extracts API documentation from content
func (da *DocAnalyzerAgent) extractAPIDocumentation(content string) []APIEndpoint {
	var endpoints []APIEndpoint

	// Look for API sections
	apiRegex := regexp.MustCompile(`(?i)##?\s*(api|endpoints|tools|methods).*?\n(.*?)(?=##|\z)`)
	matches := apiRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 2 {
			sectionContent := match[2]

			// Extract API endpoints
			endpointRegex := regexp.MustCompile(`(?m)^[\s]*(?:\*|\-)\s*(.+)`)
			endpointMatches := endpointRegex.FindAllStringSubmatch(sectionContent, -1)

			for _, endpointMatch := range endpointMatches {
				if len(endpointMatch) > 1 {
					text := strings.TrimSpace(endpointMatch[1])

					endpoints = append(endpoints, APIEndpoint{
						Path:        text,
						Method:      "GET",
						Description: text,
					})
				}
			}
		}
	}

	return endpoints
}

// generateDocumentationSummary generates a summary of the documentation analysis
func (da *DocAnalyzerAgent) generateDocumentationSummary(analysis *DocumentationAnalysis) string {
	var parts []string

	if len(analysis.InstallSteps) > 0 {
		parts = append(parts, fmt.Sprintf("%d installation steps", len(analysis.InstallSteps)))
	}

	if len(analysis.Requirements) > 0 {
		parts = append(parts, fmt.Sprintf("%d requirements", len(analysis.Requirements)))
	}

	if len(analysis.ConfigExamples) > 0 {
		parts = append(parts, fmt.Sprintf("%d configuration examples", len(analysis.ConfigExamples)))
	}

	if len(analysis.TroubleShooting) > 0 {
		parts = append(parts, fmt.Sprintf("%d troubleshooting tips", len(analysis.TroubleShooting)))
	}

	if len(parts) == 0 {
		return "Documentation found but limited structured information extracted"
	}

	return "Documentation contains: " + strings.Join(parts, ", ")
}

// Generic helper methods for when documentation is not available

func (da *DocAnalyzerAgent) provideGenericInstallationHelp(serverName string) string {
	return fmt.Sprintf(`**Generic Installation Steps for %s:**

1. **Check Requirements**
   - Verify Node.js (for npm/npx servers) or Python (for uvx servers)
   - Check system compatibility

2. **Install Dependencies**
   ` + "```" + `bash
   # For npm packages
   npm install -g package-name

   # For Python packages
   pip install package-name
   ` + "```" + `

3. **Configure Server**
   - Set up environment variables
   - Create configuration files
   - Test connectivity

4. **Verify Installation**
   - Test server startup
   - Check tool availability

💡 **Need specific help?** Ask the installer agent to help with dependencies!`, serverName)
}

func (da *DocAnalyzerAgent) provideGenericRequirements(serverName string) string {
	return `**Common MCP Server Requirements:**

**System Requirements:**
• Operating System: macOS, Linux, or Windows
• Memory: At least 512MB RAM available
• Network: Internet connectivity for external APIs

**Software Dependencies:**
• Node.js (v14+) for npm/npx servers
• Python (3.8+) for Python-based servers
• Git for repository-based installations

**Environment:**
• API keys (if required by the service)
• Proper network permissions
• File system write access

💡 **Want to check your system?** Ask me to verify these requirements!`
}

func (da *DocAnalyzerAgent) provideGenericConfigurationHelp(serverName string) string {
	return fmt.Sprintf(`**Configuration Help for %s:**

**Basic Configuration Structure:**
` + "```" + `json
{
  "name": "%s",
  "command": "command-here",
  "args": ["arg1", "arg2"],
  "protocol": "stdio",
  "enabled": true,
  "env": {
    "API_KEY": "your-api-key-here"
  }
}
` + "```" + `

**Common Configuration Fields:**
• **name**: Unique identifier for the server
• **command**: Command to run (npx, uvx, python, etc.)
• **args**: Arguments for the command
• **protocol**: Communication protocol (usually "stdio")
• **env**: Environment variables (API keys, etc.)

💡 **Need help with your specific config?** Share it and I'll help optimize it!`, serverName, serverName)
}

func (da *DocAnalyzerAgent) provideGenericTroubleshooting(serverName string) string {
	return `**Common Troubleshooting Steps:**

**Connection Issues:**
• Check network connectivity
• Verify server is enabled
• Test credentials/API keys

**Installation Problems:**
• Verify dependencies are installed
• Check command path availability
• Review error logs

**Configuration Issues:**
• Validate JSON syntax
• Check required fields
• Verify environment variables

**Performance Issues:**
• Monitor resource usage
• Check timeout settings
• Review log files for errors

💡 **Have a specific error?** Share the error message and I'll help diagnose it!`
}

func (da *DocAnalyzerAgent) provideGenericDocumentationHelp(serverName string) string {
	return fmt.Sprintf(`**Documentation Help for %s:**

I can help you with various aspects of this MCP server:

**Available Assistance:**
• Installation guidance
• Configuration setup
• Troubleshooting common issues
• Testing and validation
• Requirements checking

**What I Can Do:**
• Analyze any documentation you provide
• Guide you through setup steps
• Help troubleshoot problems
• Test server functionality

**Next Steps:**
• Ask about specific issues you're having
• Request help with installation or configuration
• Test the server to see what tools are available

💡 **Just tell me what you need help with!**`, serverName)
}