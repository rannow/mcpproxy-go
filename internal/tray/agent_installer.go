//go:build !nogui && !headless && !linux

package tray

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
)

// InstallerAgent handles service installation and dependency management
type InstallerAgent struct {
	logger *zap.Logger
}

// InstallationTask represents an installation task
type InstallationTask struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Commands    []string `json:"commands"`
	Platform    string   `json:"platform"`
	Required    bool     `json:"required"`
	Status      string   `json:"status"` // "pending", "running", "completed", "failed"
}

// SystemInfo contains system information
type SystemInfo struct {
	OS           string            `json:"os"`
	Architecture string            `json:"architecture"`
	HasNode      bool              `json:"has_node"`
	NodeVersion  string            `json:"node_version,omitempty"`
	HasPython    bool              `json:"has_python"`
	PythonVersion string           `json:"python_version,omitempty"`
	HasGit       bool              `json:"has_git"`
	GitVersion   string            `json:"git_version,omitempty"`
	HasNpm       bool              `json:"has_npm"`
	NpmVersion   string            `json:"npm_version,omitempty"`
	HasPip       bool              `json:"has_pip"`
	PipVersion   string            `json:"pip_version,omitempty"`
	Environment  map[string]string `json:"environment"`
}

// NewInstallerAgent creates a new installer agent
func NewInstallerAgent(logger *zap.Logger) *InstallerAgent {
	return &InstallerAgent{
		logger: logger,
	}
}

// ProcessMessage processes a message requesting installation assistance
func (ia *InstallerAgent) ProcessMessage(ctx context.Context, message ChatMessage, session *ChatSession) (*ChatMessage, error) {
	ia.logger.Info("Installer agent processing message",
		zap.String("session_id", session.ID),
		zap.String("server", session.ServerName))

	content := strings.ToLower(message.Content)

	var response string
	var metadata map[string]interface{}

	switch {
	case ia.containsKeywords(content, []string{"check system", "system requirements", "what do i need"}):
		response, metadata = ia.checkSystemRequirements(session.ServerName)

	case ia.containsKeywords(content, []string{"install node", "install nodejs", "node.js"}):
		response, metadata = ia.installNodeJS()

	case ia.containsKeywords(content, []string{"install python", "install python3"}):
		response, metadata = ia.installPython()

	case ia.containsKeywords(content, []string{"install git"}):
		response, metadata = ia.installGit()

	case ia.containsKeywords(content, []string{"install dependencies", "install deps", "install requirements"}):
		response, metadata = ia.installDependencies(session.ServerName, message.Content)

	case ia.containsKeywords(content, []string{"verify installation", "check installation", "test installation"}):
		response, metadata = ia.verifyInstallation(session.ServerName)

	case ia.containsKeywords(content, []string{"fix permissions", "permission", "access denied"}):
		response, metadata = ia.fixPermissions()

	case ia.containsKeywords(content, []string{"environment setup", "setup environment", "configure environment"}):
		response, metadata = ia.setupEnvironment(session.ServerName)

	default:
		response, metadata = ia.provideInstallationGuidance(session.ServerName)
	}

	return &ChatMessage{
		ID:        generateMessageID(),
		Role:      "assistant",
		Content:   response,
		AgentType: string(AgentTypeInstaller),
		Timestamp: time.Now(),
		Metadata:  metadata,
	}, nil
}

// GetCapabilities returns the capabilities of the installer agent
func (ia *InstallerAgent) GetCapabilities() []string {
	return []string{
		"System requirements checking",
		"Node.js installation guidance",
		"Python installation guidance",
		"Git installation guidance",
		"Dependency management",
		"Environment setup",
		"Permission fixing",
		"Installation verification",
	}
}

// GetAgentType returns the agent type
func (ia *InstallerAgent) GetAgentType() AgentType {
	return AgentTypeInstaller
}

// CanHandle determines if this agent can handle a message
func (ia *InstallerAgent) CanHandle(message ChatMessage) bool {
	content := strings.ToLower(message.Content)
	keywords := []string{
		"install", "installation", "setup", "dependencies", "requirements",
		"node", "python", "git", "npm", "pip", "system", "environment",
		"permission", "verify", "check", "fix",
	}

	return ia.containsKeywords(content, keywords)
}

// containsKeywords checks if content contains any of the specified keywords
func (ia *InstallerAgent) containsKeywords(content string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

// checkSystemRequirements checks system requirements
func (ia *InstallerAgent) checkSystemRequirements(serverName string) (string, map[string]interface{}) {
	ia.logger.Info("Checking system requirements", zap.String("server", serverName))

	systemInfo := ia.gatherSystemInfo()

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("🔍 **System Requirements Check for %s**\n\n", serverName))

	// System Information
	responseBuilder.WriteString("**📋 System Information:**\n")
	responseBuilder.WriteString(fmt.Sprintf("• **Operating System**: %s\n", systemInfo.OS))
	responseBuilder.WriteString(fmt.Sprintf("• **Architecture**: %s\n\n", systemInfo.Architecture))

	// Check core requirements
	responseBuilder.WriteString("**✅ Core Requirements:**\n")

	// Node.js check
	if systemInfo.HasNode {
		responseBuilder.WriteString(fmt.Sprintf("✅ **Node.js**: %s installed\n", systemInfo.NodeVersion))
	} else {
		responseBuilder.WriteString("❌ **Node.js**: Not installed (required for npm/npx servers)\n")
	}

	// Python check
	if systemInfo.HasPython {
		responseBuilder.WriteString(fmt.Sprintf("✅ **Python**: %s installed\n", systemInfo.PythonVersion))
	} else {
		responseBuilder.WriteString("❌ **Python**: Not installed (required for Python-based servers)\n")
	}

	// Git check
	if systemInfo.HasGit {
		responseBuilder.WriteString(fmt.Sprintf("✅ **Git**: %s installed\n", systemInfo.GitVersion))
	} else {
		responseBuilder.WriteString("❌ **Git**: Not installed (recommended for repository access)\n")
	}

	// Package managers
	responseBuilder.WriteString("\n**📦 Package Managers:**\n")
	if systemInfo.HasNpm {
		responseBuilder.WriteString(fmt.Sprintf("✅ **npm**: %s available\n", systemInfo.NpmVersion))
	} else {
		responseBuilder.WriteString("❌ **npm**: Not available\n")
	}

	if systemInfo.HasPip {
		responseBuilder.WriteString(fmt.Sprintf("✅ **pip**: %s available\n", systemInfo.PipVersion))
	} else {
		responseBuilder.WriteString("❌ **pip**: Not available\n")
	}

	// Generate recommendations
	responseBuilder.WriteString("\n**💡 Recommendations:**\n")
	if !systemInfo.HasNode {
		responseBuilder.WriteString("• Install Node.js from https://nodejs.org/ or use 'install nodejs'\n")
	}
	if !systemInfo.HasPython {
		responseBuilder.WriteString("• Install Python from https://python.org/ or use 'install python'\n")
	}
	if !systemInfo.HasGit {
		responseBuilder.WriteString("• Install Git from https://git-scm.com/ or use 'install git'\n")
	}

	responseBuilder.WriteString("\n🔧 **Need help installing any of these?** Just ask!")

	return responseBuilder.String(), map[string]interface{}{
		"system_info": systemInfo,
	}
}

// installNodeJS provides Node.js installation guidance
func (ia *InstallerAgent) installNodeJS() (string, map[string]interface{}) {
	ia.logger.Info("Providing Node.js installation guidance")

	var responseBuilder strings.Builder
	responseBuilder.WriteString("📦 **Node.js Installation Guide**\n\n")

	osType := runtime.GOOS

	switch osType {
	case "darwin": // macOS
		responseBuilder.WriteString("**🍎 macOS Installation:**\n\n")
		responseBuilder.WriteString("**Option 1: Official Installer (Recommended)**\n")
		responseBuilder.WriteString("1. Visit https://nodejs.org/\n")
		responseBuilder.WriteString("2. Download the LTS version\n")
		responseBuilder.WriteString("3. Run the installer\n\n")

		responseBuilder.WriteString("**Option 2: Homebrew**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("# Install Homebrew (if not installed)\n")
		responseBuilder.WriteString("/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"\n\n")
		responseBuilder.WriteString("# Install Node.js\n")
		responseBuilder.WriteString("brew install node\n")
		responseBuilder.WriteString("```\n\n")

		responseBuilder.WriteString("**Option 3: NVM (Node Version Manager)**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("# Install nvm\n")
		responseBuilder.WriteString("curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash\n\n")
		responseBuilder.WriteString("# Restart terminal, then install Node.js\n")
		responseBuilder.WriteString("nvm install --lts\n")
		responseBuilder.WriteString("nvm use --lts\n")
		responseBuilder.WriteString("```\n")

	case "linux":
		responseBuilder.WriteString("**🐧 Linux Installation:**\n\n")
		responseBuilder.WriteString("**Option 1: Package Manager**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("# Ubuntu/Debian\n")
		responseBuilder.WriteString("curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -\n")
		responseBuilder.WriteString("sudo apt-get install -y nodejs\n\n")
		responseBuilder.WriteString("# CentOS/RHEL/Fedora\n")
		responseBuilder.WriteString("curl -fsSL https://rpm.nodesource.com/setup_lts.x | sudo bash -\n")
		responseBuilder.WriteString("sudo yum install -y nodejs\n")
		responseBuilder.WriteString("```\n\n")

		responseBuilder.WriteString("**Option 2: NVM**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash\n")
		responseBuilder.WriteString("source ~/.bashrc\n")
		responseBuilder.WriteString("nvm install --lts\n")
		responseBuilder.WriteString("```\n")

	case "windows":
		responseBuilder.WriteString("**🪟 Windows Installation:**\n\n")
		responseBuilder.WriteString("**Option 1: Official Installer (Recommended)**\n")
		responseBuilder.WriteString("1. Visit https://nodejs.org/\n")
		responseBuilder.WriteString("2. Download the Windows LTS installer\n")
		responseBuilder.WriteString("3. Run the .msi file and follow the wizard\n\n")

		responseBuilder.WriteString("**Option 2: Chocolatey**\n")
		responseBuilder.WriteString("```powershell\n")
		responseBuilder.WriteString("# Install Chocolatey (if not installed)\n")
		responseBuilder.WriteString("Set-ExecutionPolicy Bypass -Scope Process -Force\n")
		responseBuilder.WriteString("[System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072\n")
		responseBuilder.WriteString("iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))\n\n")
		responseBuilder.WriteString("# Install Node.js\n")
		responseBuilder.WriteString("choco install nodejs\n")
		responseBuilder.WriteString("```\n")

	default:
		responseBuilder.WriteString("**Installation for your platform:**\n")
		responseBuilder.WriteString("Visit https://nodejs.org/ and download the appropriate installer for your system.\n")
	}

	responseBuilder.WriteString("\n**✅ Verification:**\n")
	responseBuilder.WriteString("After installation, verify with:\n")
	responseBuilder.WriteString("```bash\n")
	responseBuilder.WriteString("node --version\n")
	responseBuilder.WriteString("npm --version\n")
	responseBuilder.WriteString("```\n\n")

	responseBuilder.WriteString("💡 **Need help with the installation process?** Just ask!")

	return responseBuilder.String(), map[string]interface{}{
		"platform": osType,
		"installation_type": "nodejs",
	}
}

// installPython provides Python installation guidance
func (ia *InstallerAgent) installPython() (string, map[string]interface{}) {
	ia.logger.Info("Providing Python installation guidance")

	var responseBuilder strings.Builder
	responseBuilder.WriteString("🐍 **Python Installation Guide**\n\n")

	osType := runtime.GOOS

	switch osType {
	case "darwin": // macOS
		responseBuilder.WriteString("**🍎 macOS Installation:**\n\n")
		responseBuilder.WriteString("**Option 1: Official Installer (Recommended)**\n")
		responseBuilder.WriteString("1. Visit https://python.org/downloads/\n")
		responseBuilder.WriteString("2. Download Python 3.x for macOS\n")
		responseBuilder.WriteString("3. Run the installer\n\n")

		responseBuilder.WriteString("**Option 2: Homebrew**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("brew install python3\n")
		responseBuilder.WriteString("```\n\n")

		responseBuilder.WriteString("**Option 3: pyenv (Python Version Manager)**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("# Install pyenv\n")
		responseBuilder.WriteString("brew install pyenv\n\n")
		responseBuilder.WriteString("# Install Python\n")
		responseBuilder.WriteString("pyenv install 3.11.0\n")
		responseBuilder.WriteString("pyenv global 3.11.0\n")
		responseBuilder.WriteString("```\n")

	case "linux":
		responseBuilder.WriteString("**🐧 Linux Installation:**\n\n")
		responseBuilder.WriteString("**Ubuntu/Debian:**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("sudo apt update\n")
		responseBuilder.WriteString("sudo apt install python3 python3-pip python3-venv\n")
		responseBuilder.WriteString("```\n\n")

		responseBuilder.WriteString("**CentOS/RHEL/Fedora:**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("sudo dnf install python3 python3-pip\n")
		responseBuilder.WriteString("# OR for older systems:\n")
		responseBuilder.WriteString("sudo yum install python3 python3-pip\n")
		responseBuilder.WriteString("```\n")

	case "windows":
		responseBuilder.WriteString("**🪟 Windows Installation:**\n\n")
		responseBuilder.WriteString("**Option 1: Official Installer (Recommended)**\n")
		responseBuilder.WriteString("1. Visit https://python.org/downloads/\n")
		responseBuilder.WriteString("2. Download Python 3.x for Windows\n")
		responseBuilder.WriteString("3. Run installer and **check \"Add Python to PATH\"**\n\n")

		responseBuilder.WriteString("**Option 2: Microsoft Store**\n")
		responseBuilder.WriteString("1. Open Microsoft Store\n")
		responseBuilder.WriteString("2. Search for \"Python 3.x\"\n")
		responseBuilder.WriteString("3. Install the latest version\n\n")

		responseBuilder.WriteString("**Option 3: Chocolatey**\n")
		responseBuilder.WriteString("```powershell\n")
		responseBuilder.WriteString("choco install python3\n")
		responseBuilder.WriteString("```\n")

	default:
		responseBuilder.WriteString("**Installation for your platform:**\n")
		responseBuilder.WriteString("Visit https://python.org/downloads/ and download the appropriate installer.\n")
	}

	responseBuilder.WriteString("\n**✅ Verification:**\n")
	responseBuilder.WriteString("After installation, verify with:\n")
	responseBuilder.WriteString("```bash\n")
	responseBuilder.WriteString("python3 --version\n")
	responseBuilder.WriteString("pip3 --version\n")
	responseBuilder.WriteString("```\n\n")

	responseBuilder.WriteString("**🔧 Additional Tools:**\n")
	responseBuilder.WriteString("For Python MCP servers, you might also need:\n")
	responseBuilder.WriteString("```bash\n")
	responseBuilder.WriteString("# Install uvx (Python package runner)\n")
	responseBuilder.WriteString("pip3 install uvx\n")
	responseBuilder.WriteString("```\n\n")

	responseBuilder.WriteString("💡 **Need help with the installation process?** Just ask!")

	return responseBuilder.String(), map[string]interface{}{
		"platform": osType,
		"installation_type": "python",
	}
}

// installGit provides Git installation guidance
func (ia *InstallerAgent) installGit() (string, map[string]interface{}) {
	ia.logger.Info("Providing Git installation guidance")

	var responseBuilder strings.Builder
	responseBuilder.WriteString("📚 **Git Installation Guide**\n\n")

	osType := runtime.GOOS

	switch osType {
	case "darwin": // macOS
		responseBuilder.WriteString("**🍎 macOS Installation:**\n\n")
		responseBuilder.WriteString("**Option 1: Xcode Command Line Tools**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("xcode-select --install\n")
		responseBuilder.WriteString("```\n\n")

		responseBuilder.WriteString("**Option 2: Homebrew**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("brew install git\n")
		responseBuilder.WriteString("```\n\n")

		responseBuilder.WriteString("**Option 3: Official Installer**\n")
		responseBuilder.WriteString("1. Visit https://git-scm.com/download/mac\n")
		responseBuilder.WriteString("2. Download the installer\n")
		responseBuilder.WriteString("3. Run the installation\n")

	case "linux":
		responseBuilder.WriteString("**🐧 Linux Installation:**\n\n")
		responseBuilder.WriteString("**Ubuntu/Debian:**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("sudo apt update\n")
		responseBuilder.WriteString("sudo apt install git\n")
		responseBuilder.WriteString("```\n\n")

		responseBuilder.WriteString("**CentOS/RHEL/Fedora:**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("sudo dnf install git\n")
		responseBuilder.WriteString("# OR for older systems:\n")
		responseBuilder.WriteString("sudo yum install git\n")
		responseBuilder.WriteString("```\n")

	case "windows":
		responseBuilder.WriteString("**🪟 Windows Installation:**\n\n")
		responseBuilder.WriteString("**Option 1: Git for Windows (Recommended)**\n")
		responseBuilder.WriteString("1. Visit https://git-scm.com/download/win\n")
		responseBuilder.WriteString("2. Download Git for Windows\n")
		responseBuilder.WriteString("3. Run installer with default settings\n\n")

		responseBuilder.WriteString("**Option 2: GitHub Desktop**\n")
		responseBuilder.WriteString("1. Visit https://desktop.github.com/\n")
		responseBuilder.WriteString("2. Download and install GitHub Desktop\n")
		responseBuilder.WriteString("3. Git will be included\n")

	default:
		responseBuilder.WriteString("**Installation for your platform:**\n")
		responseBuilder.WriteString("Visit https://git-scm.com/downloads and download the appropriate installer.\n")
	}

	responseBuilder.WriteString("\n**✅ Verification:**\n")
	responseBuilder.WriteString("After installation, verify with:\n")
	responseBuilder.WriteString("```bash\n")
	responseBuilder.WriteString("git --version\n")
	responseBuilder.WriteString("```\n\n")

	responseBuilder.WriteString("**⚙️ Initial Configuration:**\n")
	responseBuilder.WriteString("Set up your Git identity:\n")
	responseBuilder.WriteString("```bash\n")
	responseBuilder.WriteString("git config --global user.name \"Your Name\"\n")
	responseBuilder.WriteString("git config --global user.email \"your.email@example.com\"\n")
	responseBuilder.WriteString("```\n\n")

	responseBuilder.WriteString("💡 **Need help with Git configuration?** Just ask!")

	return responseBuilder.String(), map[string]interface{}{
		"platform": osType,
		"installation_type": "git",
	}
}

// installDependencies provides dependency installation guidance
func (ia *InstallerAgent) installDependencies(serverName, userMessage string) (string, map[string]interface{}) {
	ia.logger.Info("Providing dependency installation guidance", zap.String("server", serverName))

	// Check what type of server this might be based on common patterns
	var serverType string
	if strings.Contains(strings.ToLower(userMessage), "npm") || strings.Contains(strings.ToLower(userMessage), "node") {
		serverType = "npm"
	} else if strings.Contains(strings.ToLower(userMessage), "python") || strings.Contains(strings.ToLower(userMessage), "pip") {
		serverType = "python"
	} else {
		serverType = "unknown"
	}

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("📦 **Dependency Installation for %s**\n\n", serverName))

	if serverType == "npm" {
		responseBuilder.WriteString("**📦 Node.js/npm Dependencies:**\n\n")
		responseBuilder.WriteString("**For npm packages:**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("# Install globally\n")
		responseBuilder.WriteString("npm install -g package-name\n\n")
		responseBuilder.WriteString("# Install locally\n")
		responseBuilder.WriteString("npm install package-name\n\n")
		responseBuilder.WriteString("# Using npx (no installation needed)\n")
		responseBuilder.WriteString("npx package-name\n")
		responseBuilder.WriteString("```\n\n")

		responseBuilder.WriteString("**Common MCP npm packages:**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("# Filesystem server\n")
		responseBuilder.WriteString("npm install -g @modelcontextprotocol/server-filesystem\n\n")
		responseBuilder.WriteString("# Git server\n")
		responseBuilder.WriteString("npm install -g @modelcontextprotocol/server-git\n\n")
		responseBuilder.WriteString("# Everything server (for testing)\n")
		responseBuilder.WriteString("npm install -g @modelcontextprotocol/server-everything\n")
		responseBuilder.WriteString("```\n")

	} else if serverType == "python" {
		responseBuilder.WriteString("**🐍 Python Dependencies:**\n\n")
		responseBuilder.WriteString("**For Python packages:**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("# Install with pip\n")
		responseBuilder.WriteString("pip3 install package-name\n\n")
		responseBuilder.WriteString("# Install with uvx (recommended for tools)\n")
		responseBuilder.WriteString("uvx install package-name\n\n")
		responseBuilder.WriteString("# Or run directly with uvx\n")
		responseBuilder.WriteString("uvx run package-name\n")
		responseBuilder.WriteString("```\n\n")

		responseBuilder.WriteString("**Setup uvx if not installed:**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("pip3 install uvx\n")
		responseBuilder.WriteString("```\n")

	} else {
		responseBuilder.WriteString("**🔧 General Dependencies:**\n\n")
		responseBuilder.WriteString("**For npm-based servers:**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("npm install -g package-name\n")
		responseBuilder.WriteString("# or use npx for direct execution\n")
		responseBuilder.WriteString("npx package-name\n")
		responseBuilder.WriteString("```\n\n")

		responseBuilder.WriteString("**For Python-based servers:**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("pip3 install package-name\n")
		responseBuilder.WriteString("# or use uvx\n")
		responseBuilder.WriteString("uvx install package-name\n")
		responseBuilder.WriteString("```\n")
	}

	responseBuilder.WriteString("\n**🔍 Finding Dependencies:**\n")
	responseBuilder.WriteString("• Check the server's documentation\n")
	responseBuilder.WriteString("• Look for package.json (npm) or requirements.txt (Python)\n")
	responseBuilder.WriteString("• Check the repository README\n")
	responseBuilder.WriteString("• Ask the documentation analyzer for requirements\n\n")

	responseBuilder.WriteString("**🚨 Common Issues:**\n")
	responseBuilder.WriteString("• **Permission errors**: Use sudo (Linux/Mac) or run as administrator (Windows)\n")
	responseBuilder.WriteString("• **Path issues**: Ensure npm/pip are in your PATH\n")
	responseBuilder.WriteString("• **Version conflicts**: Use virtual environments or nvm/pyenv\n\n")

	responseBuilder.WriteString("💡 **Need help with a specific dependency?** Tell me the package name!")

	return responseBuilder.String(), map[string]interface{}{
		"server_type": serverType,
		"dependencies": true,
	}
}

// verifyInstallation verifies installation status
func (ia *InstallerAgent) verifyInstallation(serverName string) (string, map[string]interface{}) {
	ia.logger.Info("Verifying installation", zap.String("server", serverName))

	systemInfo := ia.gatherSystemInfo()

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("✅ **Installation Verification for %s**\n\n", serverName))

	// System check
	responseBuilder.WriteString("**🔍 System Check:**\n")
	score := 0
	total := 5

	if systemInfo.HasNode {
		responseBuilder.WriteString(fmt.Sprintf("✅ Node.js: %s\n", systemInfo.NodeVersion))
		score++
	} else {
		responseBuilder.WriteString("❌ Node.js: Not found\n")
	}

	if systemInfo.HasNpm {
		responseBuilder.WriteString(fmt.Sprintf("✅ npm: %s\n", systemInfo.NpmVersion))
		score++
	} else {
		responseBuilder.WriteString("❌ npm: Not found\n")
	}

	if systemInfo.HasPython {
		responseBuilder.WriteString(fmt.Sprintf("✅ Python: %s\n", systemInfo.PythonVersion))
		score++
	} else {
		responseBuilder.WriteString("❌ Python: Not found\n")
	}

	if systemInfo.HasPip {
		responseBuilder.WriteString(fmt.Sprintf("✅ pip: %s\n", systemInfo.PipVersion))
		score++
	} else {
		responseBuilder.WriteString("❌ pip: Not found\n")
	}

	if systemInfo.HasGit {
		responseBuilder.WriteString(fmt.Sprintf("✅ Git: %s\n", systemInfo.GitVersion))
		score++
	} else {
		responseBuilder.WriteString("❌ Git: Not found\n")
	}

	// Overall score
	percentage := (score * 100) / total
	responseBuilder.WriteString(fmt.Sprintf("\n**📊 Overall Score: %d/%d (%d%%)**\n\n", score, total, percentage))

	if percentage >= 80 {
		responseBuilder.WriteString("🎉 **Excellent!** Your system is well-prepared for MCP servers.\n")
	} else if percentage >= 60 {
		responseBuilder.WriteString("👍 **Good!** Most requirements are met. Consider installing missing components.\n")
	} else {
		responseBuilder.WriteString("⚠️ **Needs Work!** Several key components are missing. Install them for the best experience.\n")
	}

	responseBuilder.WriteString("\n**🧪 Test Commands:**\n")
	responseBuilder.WriteString("Try these commands to test functionality:\n")
	responseBuilder.WriteString("```bash\n")
	if systemInfo.HasNode {
		responseBuilder.WriteString("# Test npm installation\n")
		responseBuilder.WriteString("npx @modelcontextprotocol/server-everything\n\n")
	}
	if systemInfo.HasPython {
		responseBuilder.WriteString("# Test Python installation\n")
		responseBuilder.WriteString("python3 --version\n\n")
	}
	responseBuilder.WriteString("```\n")

	responseBuilder.WriteString("💡 **Found issues?** Ask me to help install missing components!")

	return responseBuilder.String(), map[string]interface{}{
		"system_info": systemInfo,
		"score":       score,
		"total":       total,
		"percentage":  percentage,
	}
}

// fixPermissions provides permission fixing guidance
func (ia *InstallerAgent) fixPermissions() (string, map[string]interface{}) {
	ia.logger.Info("Providing permission fixing guidance")

	var responseBuilder strings.Builder
	responseBuilder.WriteString("🔐 **Permission Issues Help**\n\n")

	osType := runtime.GOOS

	switch osType {
	case "darwin", "linux": // macOS and Linux
		responseBuilder.WriteString("**🔧 Common Permission Fixes:**\n\n")

		responseBuilder.WriteString("**For npm global installations:**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("# Option 1: Use npm prefix (recommended)\n")
		responseBuilder.WriteString("mkdir ~/.npm-global\n")
		responseBuilder.WriteString("npm config set prefix '~/.npm-global'\n")
		responseBuilder.WriteString("echo 'export PATH=~/.npm-global/bin:$PATH' >> ~/.bashrc\n")
		responseBuilder.WriteString("source ~/.bashrc\n\n")
		responseBuilder.WriteString("# Option 2: Fix npm permissions\n")
		responseBuilder.WriteString("sudo chown -R $(whoami) $(npm config get prefix)/{lib/node_modules,bin,share}\n")
		responseBuilder.WriteString("```\n\n")

		responseBuilder.WriteString("**For Python pip installations:**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("# Use user installation (recommended)\n")
		responseBuilder.WriteString("pip3 install --user package-name\n\n")
		responseBuilder.WriteString("# Or create virtual environment\n")
		responseBuilder.WriteString("python3 -m venv myenv\n")
		responseBuilder.WriteString("source myenv/bin/activate\n")
		responseBuilder.WriteString("pip install package-name\n")
		responseBuilder.WriteString("```\n\n")

		responseBuilder.WriteString("**File permissions:**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("# Make file executable\n")
		responseBuilder.WriteString("chmod +x filename\n\n")
		responseBuilder.WriteString("# Fix directory permissions\n")
		responseBuilder.WriteString("chmod 755 directory-name\n")
		responseBuilder.WriteString("```\n")

	case "windows":
		responseBuilder.WriteString("**🪟 Windows Permission Issues:**\n\n")

		responseBuilder.WriteString("**Run as Administrator:**\n")
		responseBuilder.WriteString("1. Right-click Command Prompt or PowerShell\n")
		responseBuilder.WriteString("2. Select \"Run as administrator\"\n")
		responseBuilder.WriteString("3. Try your installation command again\n\n")

		responseBuilder.WriteString("**Execution Policy (PowerShell):**\n")
		responseBuilder.WriteString("```powershell\n")
		responseBuilder.WriteString("# Allow script execution\n")
		responseBuilder.WriteString("Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser\n")
		responseBuilder.WriteString("```\n\n")

		responseBuilder.WriteString("**User-level installations:**\n")
		responseBuilder.WriteString("```bash\n")
		responseBuilder.WriteString("# npm without admin\n")
		responseBuilder.WriteString("npm install -g package-name --prefix=%APPDATA%/npm\n\n")
		responseBuilder.WriteString("# pip user installation\n")
		responseBuilder.WriteString("pip install --user package-name\n")
		responseBuilder.WriteString("```\n")

	default:
		responseBuilder.WriteString("**General Permission Guidelines:**\n")
		responseBuilder.WriteString("• Use user-level installations when possible\n")
		responseBuilder.WriteString("• Avoid running package managers as root/administrator\n")
		responseBuilder.WriteString("• Use virtual environments for Python\n")
		responseBuilder.WriteString("• Configure npm to use a user directory\n")
	}

	responseBuilder.WriteString("\n**🚨 Important Notes:**\n")
	responseBuilder.WriteString("• Avoid using `sudo` with npm (use npm prefix instead)\n")
	responseBuilder.WriteString("• Use virtual environments for Python projects\n")
	responseBuilder.WriteString("• Never run package managers as root unnecessarily\n")
	responseBuilder.WriteString("• Check if your PATH is correctly configured\n\n")

	responseBuilder.WriteString("💡 **Still having permission issues?** Share the exact error and I'll help!")

	return responseBuilder.String(), map[string]interface{}{
		"platform": osType,
		"guidance_type": "permissions",
	}
}

// setupEnvironment provides environment setup guidance
func (ia *InstallerAgent) setupEnvironment(serverName string) (string, map[string]interface{}) {
	ia.logger.Info("Providing environment setup guidance", zap.String("server", serverName))

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("🌍 **Environment Setup for %s**\n\n", serverName))

	responseBuilder.WriteString("**📋 Environment Checklist:**\n\n")

	responseBuilder.WriteString("**1. System PATH Configuration**\n")
	responseBuilder.WriteString("Ensure these are in your PATH:\n")
	responseBuilder.WriteString("• Node.js and npm binaries\n")
	responseBuilder.WriteString("• Python and pip binaries\n")
	responseBuilder.WriteString("• Git binary\n")
	responseBuilder.WriteString("• Global npm packages directory\n\n")

	responseBuilder.WriteString("**2. Environment Variables**\n")
	responseBuilder.WriteString("Common variables for MCP servers:\n")
	responseBuilder.WriteString("```bash\n")
	responseBuilder.WriteString("# API keys (replace with actual values)\n")
	responseBuilder.WriteString("export API_KEY=\"your-api-key-here\"\n")
	responseBuilder.WriteString("export OPENAI_API_KEY=\"your-openai-key\"\n")
	responseBuilder.WriteString("export GITHUB_TOKEN=\"your-github-token\"\n\n")
	responseBuilder.WriteString("# Configuration\n")
	responseBuilder.WriteString("export NODE_ENV=\"development\"\n")
	responseBuilder.WriteString("export DEBUG=\"true\"\n")
	responseBuilder.WriteString("```\n\n")

	responseBuilder.WriteString("**3. Working Directories**\n")
	responseBuilder.WriteString("Set up proper working directories:\n")
	responseBuilder.WriteString("```bash\n")
	responseBuilder.WriteString("# Create project directories\n")
	responseBuilder.WriteString("mkdir -p ~/mcp-servers\n")
	responseBuilder.WriteString("mkdir -p ~/.mcpproxy\n\n")
	responseBuilder.WriteString("# Set permissions\n")
	responseBuilder.WriteString("chmod 755 ~/mcp-servers\n")
	responseBuilder.WriteString("```\n\n")

	responseBuilder.WriteString("**4. Shell Configuration**\n")
	responseBuilder.WriteString("Add to your shell profile (~/.bashrc, ~/.zshrc, etc.):\n")
	responseBuilder.WriteString("```bash\n")
	responseBuilder.WriteString("# Node.js\n")
	responseBuilder.WriteString("export PATH=\"$HOME/.npm-global/bin:$PATH\"\n\n")
	responseBuilder.WriteString("# Python\n")
	responseBuilder.WriteString("export PATH=\"$HOME/.local/bin:$PATH\"\n\n")
	responseBuilder.WriteString("# Custom MCP directory\n")
	responseBuilder.WriteString("export MCP_SERVER_DIR=\"$HOME/mcp-servers\"\n")
	responseBuilder.WriteString("```\n\n")

	responseBuilder.WriteString("**5. Testing the Environment**\n")
	responseBuilder.WriteString("```bash\n")
	responseBuilder.WriteString("# Test commands\n")
	responseBuilder.WriteString("which node npm python3 pip3 git\n")
	responseBuilder.WriteString("echo $PATH\n")
	responseBuilder.WriteString("env | grep -E '(API_KEY|NODE_ENV)'\n")
	responseBuilder.WriteString("```\n\n")

	responseBuilder.WriteString("**🔄 After Setup:**\n")
	responseBuilder.WriteString("1. Restart your terminal or run `source ~/.bashrc`\n")
	responseBuilder.WriteString("2. Verify installations with the verification command\n")
	responseBuilder.WriteString("3. Test MCP server functionality\n\n")

	responseBuilder.WriteString("💡 **Need help with specific environment setup?** Just ask!")

	return responseBuilder.String(), map[string]interface{}{
		"guidance_type": "environment_setup",
		"checklist": []string{
			"PATH configuration",
			"Environment variables",
			"Working directories",
			"Shell configuration",
			"Testing",
		},
	}
}

// provideInstallationGuidance provides general installation guidance
func (ia *InstallerAgent) provideInstallationGuidance(serverName string) (string, map[string]interface{}) {
	response := fmt.Sprintf(`🔧 **Installation Assistant for %s**

I can help you install and set up everything needed for your MCP server:

**🔍 System Analysis:**
• "Check system requirements"
• "Verify my installation"
• "What do I need to install?"

**📦 Core Dependencies:**
• "Install Node.js" - JavaScript runtime and npm
• "Install Python" - Python runtime and pip
• "Install Git" - Version control system

**🛠️ Package Management:**
• "Install dependencies" - Server-specific packages
• "Help with npm packages"
• "Help with Python packages"

**🔐 Environment Setup:**
• "Setup environment" - PATH and variables
• "Fix permissions" - Resolve access issues
• "Configure shell" - Terminal setup

**✅ Verification:**
• "Test my installation"
• "Check if everything works"
• "Verify dependencies"

**💡 Example Requests:**
• "Check what I need to install"
• "Help me install Node.js"
• "Fix permission errors"
• "Set up my environment"
• "Install dependencies for this server"

**🚨 Common Issues I Can Help With:**
• Permission denied errors
• Command not found errors
• PATH configuration problems
• Package installation failures
• Environment variable setup

What installation help do you need?`, serverName)

	return response, map[string]interface{}{
		"guidance_type": "installation",
		"available_actions": []string{
			"check_requirements",
			"install_nodejs",
			"install_python",
			"install_git",
			"install_dependencies",
			"fix_permissions",
			"setup_environment",
			"verify_installation",
		},
	}
}

// gatherSystemInfo gathers information about the current system
func (ia *InstallerAgent) gatherSystemInfo() *SystemInfo {
	info := &SystemInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		Environment:  make(map[string]string),
	}

	// Check Node.js
	if output, err := exec.Command("node", "--version").Output(); err == nil {
		info.HasNode = true
		info.NodeVersion = strings.TrimSpace(string(output))
	}

	// Check npm
	if output, err := exec.Command("npm", "--version").Output(); err == nil {
		info.HasNpm = true
		info.NpmVersion = strings.TrimSpace(string(output))
	}

	// Check Python
	if output, err := exec.Command("python3", "--version").Output(); err == nil {
		info.HasPython = true
		info.PythonVersion = strings.TrimSpace(string(output))
	}

	// Check pip
	if output, err := exec.Command("pip3", "--version").Output(); err == nil {
		info.HasPip = true
		info.PipVersion = strings.TrimSpace(string(output))
	}

	// Check Git
	if output, err := exec.Command("git", "--version").Output(); err == nil {
		info.HasGit = true
		info.GitVersion = strings.TrimSpace(string(output))
	}

	return info
}