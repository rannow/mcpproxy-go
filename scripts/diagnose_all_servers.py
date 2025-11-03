#!/usr/bin/env python3
"""
Comprehensive MCP Server Diagnostic Tool
Tests each failed server and generates detailed categorized report
"""

import json
import subprocess
import sys
from pathlib import Path
from datetime import datetime
from typing import Dict, List, Tuple

# Configuration
CONFIG_FILE = Path.home() / ".mcpproxy" / "mcp_config.json"
FAILED_SERVERS_FILE = "/tmp/failed_servers.txt"
LOG_DIR = Path.home() / "Library" / "Logs" / "mcpproxy"

class ServerDiagnostic:
    def __init__(self, name: str, config: Dict):
        self.name = name
        self.config = config
        self.command = config.get("command", "")
        self.args = config.get("args", [])
        self.protocol = config.get("protocol", "stdio")
        self.env = config.get("env", {})
        self.category = ""
        self.issue = ""
        self.fix = ""
        self.quick_fix_available = False

    def diagnose(self) -> None:
        """Run diagnostic tests on the server"""

        # Check if command exists
        try:
            result = subprocess.run(
                ["which", self.command],
                capture_output=True,
                timeout=2
            )
            command_exists = result.returncode == 0
        except:
            command_exists = False

        if not command_exists:
            self.category = "📦 Package Issue"
            self.issue = f"Command `{self.command}` not found in PATH"
            self.fix = self._get_install_instructions()
            self.quick_fix_available = True
            return

        # Check log file for specific errors
        log_file = LOG_DIR / f"server-{self.name}.log"
        if log_file.exists():
            self._analyze_log_file(log_file)
        else:
            # No log = never attempted, likely lazy loading or disabled
            self.category = "⏳ Not Attempted"
            self.issue = "Server was not attempted during startup (lazy loading)"
            self.fix = "Server will connect when first accessed via tool call"

    def _analyze_log_file(self, log_file: Path) -> None:
        """Analyze server log file for error patterns"""
        try:
            with open(log_file, 'r') as f:
                logs = f.read()

            if "context deadline exceeded" in logs:
                self.category = "⏱️ Timeout/Slow"
                self.issue = "Server takes >60s to initialize (NPX download or slow startup)"
                self.fix = """
**Options**:
1. Increase timeout to 90s in config
2. Pre-install package globally:
   ```bash
   """ + self._get_preinstall_command() + """
   ```
"""
                self.quick_fix_available = True

            elif "authentication" in logs.lower() or "oauth" in logs.lower():
                self.category = "🔐 Authentication Required"
                self.issue = "Server requires OAuth or API key configuration"
                self.fix = self._get_auth_instructions()

            elif "connection refused" in logs or "ECONNREFUSED" in logs:
                self.category = "🏗️ Infrastructure Missing"
                self.issue = "Server requires external service (database, API, etc.)"
                self.fix = self._get_infrastructure_instructions()

            elif "command not found" in logs or "cannot find" in logs:
                self.category = "⚙️ Configuration Error"
                self.issue = "Command or dependency not found"
                self.fix = "Check command path and install missing dependencies"
                self.quick_fix_available = True

            elif "deprecated" in logs.lower() or "no longer supported" in logs.lower():
                self.category = "❌ Broken/Deprecated"
                self.issue = "Package is deprecated or no longer maintained"
                self.fix = "Find alternative package or remove server"

            else:
                self.category = "🔧 Unknown Error"
                self.issue = "See log file for details"
                self.fix = f"Check `{log_file}` for error messages"

        except Exception as e:
            self.category = "❓ Cannot Analyze"
            self.issue = f"Unable to read log file: {e}"
            self.fix = "Manual investigation required"

    def _get_install_instructions(self) -> str:
        """Generate installation instructions based on command type"""
        if self.command in ["uvx", "uv"]:
            return """
**Install uv**:
```bash
pip install uv
# or
curl -LsSf https://astral.sh/uv/install.sh | sh
```
"""
        elif self.command == "npx":
            return """
**Install Node.js** (npx comes with it):
```bash
# macOS
brew install node

# or download from https://nodejs.org/
```
"""
        elif self.command == "pipx":
            return """
**Install pipx**:
```bash
pip install pipx
pipx ensurepath
```
"""
        else:
            return f"Install `{self.command}` command"

    def _get_preinstall_command(self) -> str:
        """Get command to pre-install the package"""
        if not self.args:
            return f"# No args specified"

        package = self.args[0] if self.args else ""

        if self.command in ["uvx", "uv"]:
            return f"uv tool install {package}"
        elif self.command == "npx":
            return f"npm install -g {package}"
        elif self.command == "pipx":
            return f"pipx install {package}"
        else:
            return f"# Manual installation required for {self.command}"

    def _get_auth_instructions(self) -> str:
        """Get authentication setup instructions"""
        server_lower = self.name.lower()

        if "gdrive" in server_lower or "google" in server_lower:
            return """
**Google OAuth Setup**:
1. Create OAuth credentials in Google Cloud Console
2. Download credentials.json
3. Set environment variable or config
"""
        elif "github" in server_lower:
            return """
**GitHub Authentication**:
```bash
# Set GitHub token
export GITHUB_TOKEN="your_token_here"
```
"""
        elif "notion" in server_lower:
            return """
**Notion API Setup**:
1. Create integration at https://www.notion.so/my-integrations
2. Get API key
3. Set NOTION_API_KEY environment variable
"""
        else:
            return "Refer to package documentation for authentication setup"

    def _get_infrastructure_instructions(self) -> str:
        """Get infrastructure setup instructions"""
        server_lower = self.name.lower()

        if "elasticsearch" in server_lower:
            return """
**Install Elasticsearch**:
```bash
# Docker
docker run -d -p 9200:9200 elasticsearch:8.x

# macOS
brew install elasticsearch
```
"""
        elif "redis" in server_lower:
            return """
**Install Redis**:
```bash
# Docker
docker run -d -p 6379:6379 redis

# macOS
brew install redis
redis-server
```
"""
        elif "k8s" in server_lower or "kubernetes" in server_lower:
            return """
**Install Kubernetes**:
```bash
# minikube for local
brew install minikube
minikube start
```
"""
        else:
            return "Install required infrastructure service"

    def to_markdown(self) -> str:
        """Convert diagnostic to markdown format"""
        md = f"### {self.name}\n\n"
        md += f"**Category**: {self.category}\n"
        md += f"**Protocol**: `{self.protocol}`\n"
        args_str = ' '.join(self.args) if isinstance(self.args, list) else str(self.args)
        md += f"**Command**: `{self.command} {args_str}`\n"
        md += f"\n**Issue**: {self.issue}\n"
        md += f"\n**Fix**:{self.fix}\n"

        if self.quick_fix_available:
            md += "\n**Quick Fix**: ✅ Can be fixed automatically\n"

        md += "\n---\n\n"
        return md


def load_failed_servers() -> List[str]:
    """Load list of failed servers"""
    with open(FAILED_SERVERS_FILE, 'r') as f:
        return [line.strip() for line in f if line.strip()]


def load_config() -> Dict:
    """Load mcpproxy configuration"""
    with open(CONFIG_FILE, 'r') as f:
        return json.load(f)


def generate_report(diagnostics: List[ServerDiagnostic]) -> str:
    """Generate comprehensive markdown report"""

    # Group by category
    by_category = {}
    for diag in diagnostics:
        if diag.category not in by_category:
            by_category[diag.category] = []
        by_category[diag.category].append(diag)

    # Generate report
    report = f"""# Failed MCP Servers - Detailed Diagnostic Report

**Generated**: {datetime.now().strftime("%Y-%m-%d %H:%M:%S")}
**Total Failed Servers**: {len(diagnostics)}
**Analysis Method**: Individual server testing with log analysis

---

## Executive Summary

This report provides detailed diagnostics for each MCP server that failed to connect.

### Category Breakdown

"""

    # Add category summary
    for category, servers in sorted(by_category.items()):
        count = len(servers)
        quick_fix_count = sum(1 for s in servers if s.quick_fix_available)
        report += f"- **{category}**: {count} servers"
        if quick_fix_count > 0:
            report += f" ({quick_fix_count} quick-fixable)"
        report += "\n"

    report += "\n---\n\n"

    # Add detailed diagnostics by category
    for category in sorted(by_category.keys()):
        servers = by_category[category]
        report += f"## {category} ({len(servers)} servers)\n\n"

        for server in sorted(servers, key=lambda s: s.name):
            report += server.to_markdown()

    return report


def main():
    print("🔍 MCP Server Diagnostic Tool")
    print("=" * 60)
    print()

    # Load data
    print("Loading configuration...")
    config = load_config()
    failed_servers = load_failed_servers()

    print(f"Found {len(failed_servers)} failed servers")
    print()

    # Run diagnostics
    diagnostics = []
    for i, server_name in enumerate(failed_servers, 1):
        print(f"[{i}/{len(failed_servers)}] Diagnosing: {server_name}")

        server_config = next(
            (s for s in config["mcpServers"] if s["name"] == server_name),
            None
        )

        if server_config:
            diag = ServerDiagnostic(server_name, server_config)
            diag.diagnose()
            diagnostics.append(diag)

    print()
    print("Generating report...")

    # Generate and save report
    report = generate_report(diagnostics)
    output_file = "FAILED_SERVERS_DETAILED_REPORT.md"

    with open(output_file, 'w') as f:
        f.write(report)

    print(f"✅ Report generated: {output_file}")
    print()

    # Print summary
    quick_fix = sum(1 for d in diagnostics if d.quick_fix_available)
    print(f"📊 Summary:")
    print(f"  Total diagnosed: {len(diagnostics)}")
    print(f"  Quick-fixable: {quick_fix}")
    print(f"  Need manual setup: {len(diagnostics) - quick_fix}")


if __name__ == "__main__":
    main()
