#!/usr/bin/env python3
"""
MCP Server Comprehensive Test Suite
Tests all 88 failed servers from FAILED_SERVERS_TABLE.md
Uses both direct npx calls and mcp-cli for validation
"""

import subprocess
import time
import json
import os
import sys
from pathlib import Path
from datetime import datetime
from typing import Dict, Tuple, List

# Configuration
TIMEOUT_SECONDS = 90
RESULTS_DIR = Path.home() / ".mcpproxy" / "test-results"
TIMESTAMP = datetime.now().strftime("%Y%m%d_%H%M%S")
RESULTS_FILE = RESULTS_DIR / f"server_test_results_{TIMESTAMP}.log"
SUMMARY_FILE = RESULTS_DIR / f"server_test_summary_{TIMESTAMP}.md"
CSV_FILE = RESULTS_DIR / f"server_test_results_{TIMESTAMP}.csv"

# Colors
class Colors:
    RED = '\033[0;31m'
    GREEN = '\033[0;32m'
    YELLOW = '\033[1;33m'
    BLUE = '\033[0;34m'
    NC = '\033[0m'  # No Color

# Server list from FAILED_SERVERS_TABLE.md
SERVERS = {
    "search-mcp-server": "search-mcp-server",
    "mcp-pandoc": "mcp-pandoc",
    "infinity-swiss": "infinity-swiss",
    "toolfront-database": "toolfront-database",
    "test-weather-server": "test-weather-server",
    "mcp-server-git": "@modelcontextprotocol/server-git",
    "bigquery-lucashild": "bigquery-lucashild",
    "awslabs-cloudwatch": "@awslabs/cloudwatch-logs-mcp-server",
    "auto-mcp": "auto-mcp",
    "dbhub-universal": "dbhub-universal",
    "mcp-computer-use": "mcp-computer-use",
    "mcp-openai": "@modelcontextprotocol/server-openai",
    "mcp-datetime": "@modelcontextprotocol/server-datetime",
    "n8n-mcp-server": "n8n-mcp-server",
    "documents-vector-search": "documents-vector-search",
    "mcp-perplexity": "mcp-perplexity",
    "travel-planner": "travel-planner-mcp-server",
    "mcp-youtube-transcript": "mcp-youtube-transcript",
    "youtube-transcript-2": "youtube-transcript-2",
    "mcp-server-todoist": "@modelcontextprotocol/server-todoist",
    "todoist-lucashild": "todoist-lucashild",
    "mcp-telegram": "mcp-telegram",
    "mcp-obsidian": "@modelcontextprotocol/server-obsidian",
    "todoist": "todoist",
    "supabase-mcp-server": "@supabase/mcp-server",
    "mcp-server-mongodb": "@mongodb/mcp-server",
    "tavily-mcp-server": "tavily-mcp-server",
    "mcp-memory": "@modelcontextprotocol/server-memory",
    "mcp-gsuite": "mcp-gsuite",
    "mcp-linear": "mcp-linear",
    "mcp-server-airtable": "mcp-server-airtable",
    "qstash-lucashild": "qstash-lucashild",
    "google-maps-mcp": "google-maps-mcp",
    "google-sheets-brightdata": "google-sheets-brightdata",
    "google-places-api": "google-places-api",
    "fastmcp-elevenlabs": "fastmcp-elevenlabs",
    "mcp-e2b": "mcp-e2b",
    "mcp-server-docker": "mcp-server-docker",
    "mcp-server-vscode": "mcp-server-vscode",
    "youtube-transcript": "youtube-transcript",
    "gmail-mcp-server": "gmail-mcp-server",
    "mcp-shell-lucashild": "mcp-shell-lucashild",
    "mcp-http-lucashild": "mcp-http-lucashild",
    "mcp-http-server": "mcp-http-server",
    "mcp-snowflake-database": "mcp-snowflake-database",
    "everart-mcp": "everart-mcp",
    "mcp-search": "search",
    "mcp-discord": "mcp-discord",
    "mcp-server-playwright": "@modelcontextprotocol/server-playwright",
    "tldraw-mcp-server": "tldraw-mcp-server",
    "mcp-instagram": "mcp-instagram",
    "mcp-firecrawl": "mcp-firecrawl",
    "strapi-mcp-server": "strapi-mcp-server",
    "coinbase-mcp-server": "coinbase-mcp-server",
    "convex-mcp-server": "convex-mcp-server",
    "lancedb-lucashild": "lancedb-lucashild",
    "google-gemini": "google-gemini",
    "mcp-knowledge-graph": "mcp-knowledge-graph",
    "mcp-markdown": "mcp-markdown",
    "minato": "minato",
    "json-database": "json-database",
    "obsidian-vault": "obsidian-vault",
    "upstash-vector": "upstash-vector",
    "browserbase-mcp": "browserbase-mcp",
    "figma-mcp": "figma-mcp",
    "mcp-langfuse-obsidian": "mcp-langfuse-obsidian-integration",
    "mcp-pocketbase": "mcp-pocketbase",
    "mcp-miroAI": "mcp-miroAI",
    "cloudflare-r2-brightdata": "cloudflare-r2-brightdata",
    "mcp-cloudflare-langfuse": "mcp-cloudflare-langfuse",
    "mlflow-mcp": "mlflow-mcp",
    "mcp-aws-eb-manager": "mcp-aws-eb-manager",
    "mcp-openbb": "mcp-openbb",
    "mcp-reasoner": "mcp-reasoner",
    "slack-mcp": "slack-mcp",
    "gitlab-mcp-server": "gitlab-mcp-server",
    "code-reference-mcp": "code-reference-mcp",
    "docker-mcp-server": "docker-mcp-server",
    "mcp-pandoc-pdf-docx": "mcp-pandoc-pdf-docx",
    "mcp-reddit": "mcp-reddit",
    "shopify-brightdata": "shopify-brightdata",
    "x-api-brightdata": "x-api-brightdata",
    "mcp-twitter": "mcp-twitter",
    "mcp-github": "@modelcontextprotocol/server-github",
    "brave-search": "@modelcontextprotocol/server-brave-search",
    "filesystem": "@modelcontextprotocol/server-filesystem",
    "sequential-thinking": "@modelcontextprotocol/server-sequential-thinking",
    "sqlite": "@modelcontextprotocol/server-sqlite",
}

# Priority mapping
PRIORITIES = {
    "mcp-github": "CRITICAL",
    "brave-search": "CRITICAL",
    "filesystem": "CRITICAL",
    "sequential-thinking": "CRITICAL",
    "sqlite": "CRITICAL",
    "mcp-server-git": "HIGH",
    "mcp-openai": "HIGH",
    "mcp-datetime": "HIGH",
    "mcp-obsidian": "HIGH",
    "mcp-memory": "HIGH",
    "mcp-server-playwright": "HIGH",
    "supabase-mcp-server": "HIGH",
    "mcp-server-mongodb": "HIGH",
    "google-maps-mcp": "HIGH",
    "slack-mcp": "HIGH",
}

# Stats
stats = {
    "total": 0,
    "success_direct": 0,
    "success_mcpcli": 0,
    "failed_direct": 0,
    "failed_mcpcli": 0,
    "timeout": 0
}

def log(level: str, message: str):
    """Log a message with color coding"""
    colors = {
        "INFO": Colors.BLUE,
        "SUCCESS": Colors.GREEN,
        "WARNING": Colors.YELLOW,
        "ERROR": Colors.RED
    }
    color = colors.get(level, Colors.NC)
    formatted_msg = f"{color}[{level}]{Colors.NC} {message}"
    print(formatted_msg)

    with open(RESULTS_FILE, 'a') as f:
        f.write(f"[{level}] {message}\n")

def print_header(title: str):
    """Print a section header"""
    header = f"\n{'=' * 60}\n{title}\n{'=' * 60}"
    print(header)
    with open(RESULTS_FILE, 'a') as f:
        f.write(header + "\n")

def test_server_direct(name: str, package: str) -> Tuple[int, float]:
    """Test server with direct npx call"""
    log("INFO", f"Testing {name} directly with npx...")

    start_time = time.time()
    try:
        result = subprocess.run(
            ["npx", "-y", package, "--help"],
            capture_output=True,
            timeout=TIMEOUT_SECONDS
        )
        duration = time.time() - start_time
        exit_code = result.returncode

        with open(CSV_FILE, 'a') as f:
            f.write(f"{name},{package},direct,{exit_code},{duration:.2f}\n")

        if exit_code == 0:
            log("SUCCESS", f"Direct test passed in {duration:.2f}s")
            return 0, duration
        else:
            log("ERROR", f"Direct test failed (exit code: {exit_code}, time: {duration:.2f}s)")
            return 1, duration

    except subprocess.TimeoutExpired:
        duration = time.time() - start_time
        log("ERROR", f"Direct test TIMEOUT ({TIMEOUT_SECONDS}s)")
        stats["timeout"] += 1
        with open(CSV_FILE, 'a') as f:
            f.write(f"{name},{package},direct,124,{duration:.2f}\n")
        return 2, duration
    except Exception as e:
        duration = time.time() - start_time
        log("ERROR", f"Direct test exception: {e}")
        return 1, duration

def test_server_mcpcli(name: str, package: str) -> Tuple[int, float]:
    """Test server with mcp-cli"""
    log("INFO", f"Testing {name} with mcp-cli...")

    # Create temporary config
    config = {
        "mcpServers": {
            name: {
                "command": "npx",
                "args": ["-y", package]
            }
        }
    }

    config_file = RESULTS_DIR / f"temp_config_{name}.json"
    with open(config_file, 'w') as f:
        json.dump(config, f)

    start_time = time.time()
    try:
        result = subprocess.run(
            ["npx", "-y", "@wong2/mcp-cli", "test", str(config_file), name],
            capture_output=True,
            timeout=TIMEOUT_SECONDS
        )
        duration = time.time() - start_time
        exit_code = result.returncode

        # Cleanup
        config_file.unlink(missing_ok=True)

        with open(CSV_FILE, 'a') as f:
            f.write(f"{name},{package},mcp-cli,{exit_code},{duration:.2f}\n")

        if exit_code == 0:
            log("SUCCESS", f"mcp-cli test passed in {duration:.2f}s")
            return 0, duration
        else:
            log("ERROR", f"mcp-cli test failed (exit code: {exit_code}, time: {duration:.2f}s)")
            return 1, duration

    except subprocess.TimeoutExpired:
        duration = time.time() - start_time
        log("ERROR", f"mcp-cli test TIMEOUT ({TIMEOUT_SECONDS}s)")
        config_file.unlink(missing_ok=True)
        with open(CSV_FILE, 'a') as f:
            f.write(f"{name},{package},mcp-cli,124,{duration:.2f}\n")
        return 2, duration
    except Exception as e:
        duration = time.time() - start_time
        log("ERROR", f"mcp-cli test exception: {e}")
        config_file.unlink(missing_ok=True)
        return 1, duration

def generate_summary():
    """Generate summary report"""
    total = stats["total"]

    summary = f"""# MCP Server Test Results Summary
**Generated**: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}
**Test Duration**: {TIMEOUT_SECONDS}s timeout per server
**Total Servers Tested**: {total}

---

## 📊 Overall Results

| Test Method | Success | Failed | Success Rate |
|-------------|---------|--------|--------------|
| Direct npx  | {stats['success_direct']} | {stats['failed_direct']} | {stats['success_direct']*100//total if total > 0 else 0}% |
| mcp-cli     | {stats['success_mcpcli']} | {stats['failed_mcpcli']} | {stats['success_mcpcli']*100//total if total > 0 else 0}% |

**Timeouts**: {stats['timeout']}

---

## 📈 Comparison with Original Status

**Before (from FAILED_SERVERS_TABLE.md)**:
- Failed: 88/159 (55.3%)
- Success: 71/159 (44.7%)

**After Testing**:
- Direct Success: {stats['success_direct']}/88 ({stats['success_direct']*100//88}%)
- mcp-cli Success: {stats['success_mcpcli']}/88 ({stats['success_mcpcli']*100//88}%)

---

## 📁 Output Files

- **Detailed Log**: `{RESULTS_FILE}`
- **CSV Data**: `{CSV_FILE}`
- **This Summary**: `{SUMMARY_FILE}`

---

## 🔍 Analysis

"""

    if stats['success_direct'] > 44:
        summary += "✅ **Improvement**: More servers are working now than before!\n\n"
    else:
        summary += "⚠️ **Status**: Similar or worse than before. Timeout fix may be needed.\n\n"

    summary += "### Top Recommendations\n\n"

    if stats['timeout'] > 50:
        summary += f"1. **URGENT**: Increase timeout from {TIMEOUT_SECONDS}s to 120s or more\n"

    summary += "2. Consider global installation for frequently failing servers\n"
    summary += "3. Check environment variables for servers requiring API keys\n"
    summary += "\n---\n\n"
    summary += f"**Next Steps**: Review `{CSV_FILE}` for per-server timing data\n"

    with open(SUMMARY_FILE, 'w') as f:
        f.write(summary)

    print(summary)

def main():
    """Main test execution"""
    # Create results directory
    RESULTS_DIR.mkdir(parents=True, exist_ok=True)

    # Initialize CSV
    with open(CSV_FILE, 'w') as f:
        f.write("Server,Package,TestType,ExitCode,Duration\n")

    print_header("MCP Server Comprehensive Test Suite")
    log("INFO", f"Started at: {datetime.now()}")
    log("INFO", f"Timeout: {TIMEOUT_SECONDS}s per test")
    log("INFO", f"Total servers to test: {len(SERVERS)}")
    log("INFO", "")

    # Test each server
    for i, (name, package) in enumerate(SERVERS.items(), 1):
        stats["total"] += 1
        priority = PRIORITIES.get(name, "MEDIUM")

        print_header(f"[{i}/{len(SERVERS)}] Testing: {name} (Priority: {priority})")
        log("INFO", f"Package: {package}")

        # Test 1: Direct npx call
        direct_result, _ = test_server_direct(name, package)
        if direct_result == 0:
            stats["success_direct"] += 1
        else:
            stats["failed_direct"] += 1

        # Test 2: mcp-cli validation
        mcpcli_result, _ = test_server_mcpcli(name, package)
        if mcpcli_result == 0:
            stats["success_mcpcli"] += 1
        else:
            stats["failed_mcpcli"] += 1

        # Summary for this server
        if direct_result == 0 and mcpcli_result == 0:
            log("SUCCESS", "✅ BOTH TESTS PASSED")
        elif direct_result == 0 or mcpcli_result == 0:
            log("WARNING", "⚠️  PARTIAL SUCCESS (one test passed)")
        else:
            log("ERROR", "❌ BOTH TESTS FAILED")

        print("")  # Spacing

        # Small delay between servers
        time.sleep(2)

    # Generate summary
    print_header("TEST SUMMARY")
    generate_summary()

    log("INFO", "")
    log("INFO", "Testing completed!")
    log("INFO", f"Results saved to: {RESULTS_DIR}")
    log("INFO", "")
    log("SUCCESS", f"✅ Direct Success: {stats['success_direct']}/{stats['total']}")
    log("SUCCESS", f"✅ mcp-cli Success: {stats['success_mcpcli']}/{stats['total']}")
    log("ERROR", f"❌ Direct Failed: {stats['failed_direct']}/{stats['total']}")
    log("ERROR", f"❌ mcp-cli Failed: {stats['failed_mcpcli']}/{stats['total']}")
    log("WARNING", f"⏱️  Timeouts: {stats['timeout']}")

if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\nTest interrupted by user")
        generate_summary()
        sys.exit(1)
    except Exception as e:
        print(f"\n\nFatal error: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)
