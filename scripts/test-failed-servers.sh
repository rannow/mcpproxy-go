#!/bin/bash

###############################################################################
# MCP Server Comprehensive Test Suite
# Tests all 88 failed servers from FAILED_SERVERS_TABLE.md
# Uses both direct npx calls and mcp-cli for validation
###############################################################################

# Note: Not using set -u due to bash array limitations with some server names

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TIMEOUT_SECONDS=90
RESULTS_DIR="$HOME/.mcpproxy/test-results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULTS_FILE="$RESULTS_DIR/server_test_results_$TIMESTAMP.log"
SUMMARY_FILE="$RESULTS_DIR/server_test_summary_$TIMESTAMP.md"
CSV_FILE="$RESULTS_DIR/server_test_results_$TIMESTAMP.csv"

# Create results directory
mkdir -p "$RESULTS_DIR"

# Initialize counters
TOTAL_SERVERS=0
SUCCESS_DIRECT=0
SUCCESS_MCPCLI=0
FAILED_DIRECT=0
FAILED_MCPCLI=0
TIMEOUT_COUNT=0

# Server list from FAILED_SERVERS_TABLE.md
declare -A SERVERS=(
    # Format: ["name"]="npx_package"
    ["search-mcp-server"]="search-mcp-server"
    ["mcp-pandoc"]="mcp-pandoc"
    ["infinity-swiss"]="infinity-swiss"
    ["toolfront-database"]="toolfront-database"
    ["test-weather-server"]="test-weather-server"
    ["mcp-server-git"]="@modelcontextprotocol/server-git"
    ["bigquery-lucashild"]="bigquery-lucashild"
    ["awslabs.cloudwatch-logs-mcp-server"]="@awslabs/cloudwatch-logs-mcp-server"
    ["auto-mcp"]="auto-mcp"
    ["dbhub-universal"]="dbhub-universal"
    ["mcp-computer-use"]="mcp-computer-use"
    ["mcp-openai"]="@modelcontextprotocol/server-openai"
    ["mcp-datetime"]="@modelcontextprotocol/server-datetime"
    ["n8n-mcp-server"]="n8n-mcp-server"
    ["documents-vector-search"]="documents-vector-search"
    ["mcp-perplexity"]="mcp-perplexity"
    ["travel-planner-mcp-server"]="travel-planner-mcp-server"
    ["mcp-youtube-transcript"]="mcp-youtube-transcript"
    ["youtube-transcript-2"]="youtube-transcript-2"
    ["mcp-server-todoist"]="@modelcontextprotocol/server-todoist"
    ["todoist-lucashild"]="todoist-lucashild"
    ["mcp-telegram"]="mcp-telegram"
    ["mcp-obsidian"]="@modelcontextprotocol/server-obsidian"
    ["todoist"]="todoist"
    ["supabase-mcp-server"]="@supabase/mcp-server"
    ["mcp-server-mongodb"]="@mongodb/mcp-server"
    ["tavily-mcp-server"]="tavily-mcp-server"
    ["mcp-memory"]="@modelcontextprotocol/server-memory"
    ["mcp-gsuite"]="mcp-gsuite"
    ["mcp-linear"]="mcp-linear"
    ["mcp-server-airtable"]="mcp-server-airtable"
    ["qstash-lucashild"]="qstash-lucashild"
    ["google-maps-mcp"]="google-maps-mcp"
    ["google-sheets-brightdata"]="google-sheets-brightdata"
    ["google-places-api"]="google-places-api"
    ["fastmcp-elevenlabs"]="fastmcp-elevenlabs"
    ["mcp-e2b"]="mcp-e2b"
    ["mcp-server-docker"]="mcp-server-docker"
    ["mcp-server-vscode"]="mcp-server-vscode"
    ["youtube-transcript"]="youtube-transcript"
    ["gmail-mcp-server"]="gmail-mcp-server"
    ["mcp-shell-lucashild"]="mcp-shell-lucashild"
    ["mcp-http-lucashild"]="mcp-http-lucashild"
    ["mcp-http-server"]="mcp-http-server"
    ["mcp-snowflake-database"]="mcp-snowflake-database"
    ["everart-mcp"]="everart-mcp"
    ["mcp-search"]="search"
    ["mcp-discord"]="mcp-discord"
    ["mcp-server-playwright"]="@modelcontextprotocol/server-playwright"
    ["tldraw-mcp-server"]="tldraw-mcp-server"
    ["mcp-instagram"]="mcp-instagram"
    ["mcp-firecrawl"]="mcp-firecrawl"
    ["strapi-mcp-server"]="strapi-mcp-server"
    ["coinbase-mcp-server"]="coinbase-mcp-server"
    ["convex-mcp-server"]="convex-mcp-server"
    ["lancedb-lucashild"]="lancedb-lucashild"
    ["google-gemini"]="google-gemini"
    ["mcp-knowledge-graph"]="mcp-knowledge-graph"
    ["mcp-markdown"]="mcp-markdown"
    ["minato"]="minato"
    ["json-database"]="json-database"
    ["obsidian-vault"]="obsidian-vault"
    ["upstash-vector"]="upstash-vector"
    ["browserbase-mcp"]="browserbase-mcp"
    ["figma-mcp"]="figma-mcp"
    ["mcp-langfuse-obsidian-integration"]="mcp-langfuse-obsidian-integration"
    ["mcp-pocketbase"]="mcp-pocketbase"
    ["mcp-miroAI"]="mcp-miroAI"
    ["cloudflare-r2-brightdata"]="cloudflare-r2-brightdata"
    ["mcp-cloudflare-langfuse"]="mcp-cloudflare-langfuse"
    ["mlflow-mcp"]="mlflow-mcp"
    ["mcp-aws-eb-manager"]="mcp-aws-eb-manager"
    ["mcp-openbb"]="mcp-openbb"
    ["mcp-reasoner"]="mcp-reasoner"
    ["slack-mcp"]="slack-mcp"
    ["gitlab-mcp-server"]="gitlab-mcp-server"
    ["code-reference-mcp"]="code-reference-mcp"
    ["docker-mcp-server"]="docker-mcp-server"
    ["mcp-pandoc-pdf-docx"]="mcp-pandoc-pdf-docx"
    ["mcp-reddit"]="mcp-reddit"
    ["shopify-brightdata"]="shopify-brightdata"
    ["x-api-brightdata"]="x-api-brightdata"
    ["mcp-twitter"]="mcp-twitter"
    ["mcp-github"]="@modelcontextprotocol/server-github"
    ["brave-search"]="@modelcontextprotocol/server-brave-search"
    ["filesystem"]="@modelcontextprotocol/server-filesystem"
    ["sequential-thinking"]="@modelcontextprotocol/server-sequential-thinking"
    ["sqlite"]="@modelcontextprotocol/server-sqlite"
)

# Priority mapping
declare -A PRIORITIES=(
    ["mcp-github"]="CRITICAL"
    ["brave-search"]="CRITICAL"
    ["filesystem"]="CRITICAL"
    ["sequential-thinking"]="CRITICAL"
    ["sqlite"]="CRITICAL"
    ["mcp-server-git"]="HIGH"
    ["mcp-openai"]="HIGH"
    ["mcp-datetime"]="HIGH"
    ["mcp-obsidian"]="HIGH"
    ["mcp-memory"]="HIGH"
    ["mcp-server-playwright"]="HIGH"
    ["supabase-mcp-server"]="HIGH"
    ["mcp-server-mongodb"]="HIGH"
    ["google-maps-mcp"]="HIGH"
    ["slack-mcp"]="HIGH"
)

###############################################################################
# Functions
###############################################################################

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$RESULTS_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$RESULTS_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$RESULTS_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$RESULTS_FILE"
}

print_header() {
    echo "" | tee -a "$RESULTS_FILE"
    echo "========================================" | tee -a "$RESULTS_FILE"
    echo "$1" | tee -a "$RESULTS_FILE"
    echo "========================================" | tee -a "$RESULTS_FILE"
}

test_server_direct() {
    local name=$1
    local package=$2
    local start_time
    local end_time
    local duration
    local exit_code

    log_info "Testing $name directly with npx..."

    # Measure startup time
    start_time=$(date +%s.%N)

    # Try to run with --help flag (most servers support this)
    timeout $TIMEOUT_SECONDS npx -y "$package" --help >/dev/null 2>&1
    exit_code=$?

    end_time=$(date +%s.%N)
    duration=$(echo "$end_time - $start_time" | bc)

    # Store result
    echo "$name,$package,direct,$exit_code,$duration" >> "$CSV_FILE"

    if [ $exit_code -eq 0 ]; then
        log_success "Direct test passed in ${duration}s"
        return 0
    elif [ $exit_code -eq 124 ]; then
        log_error "Direct test TIMEOUT (${TIMEOUT_SECONDS}s)"
        ((TIMEOUT_COUNT++))
        return 2
    else
        log_error "Direct test failed (exit code: $exit_code, time: ${duration}s)"
        return 1
    fi
}

test_server_mcpcli() {
    local name=$1
    local package=$2
    local start_time
    local end_time
    local duration
    local exit_code

    log_info "Testing $name with mcp-cli..."

    # Create temporary config for mcp-cli
    local temp_config=$(mktemp)
    cat > "$temp_config" <<EOF
{
  "mcpServers": {
    "$name": {
      "command": "npx",
      "args": ["-y", "$package"]
    }
  }
}
EOF

    # Measure startup time
    start_time=$(date +%s.%N)

    # Run mcp-cli test
    timeout $TIMEOUT_SECONDS npx -y @wong2/mcp-cli test "$temp_config" "$name" >/dev/null 2>&1
    exit_code=$?

    end_time=$(date +%s.%N)
    duration=$(echo "$end_time - $start_time" | bc)

    # Cleanup
    rm -f "$temp_config"

    # Store result
    echo "$name,$package,mcp-cli,$exit_code,$duration" >> "$CSV_FILE"

    if [ $exit_code -eq 0 ]; then
        log_success "mcp-cli test passed in ${duration}s"
        return 0
    elif [ $exit_code -eq 124 ]; then
        log_error "mcp-cli test TIMEOUT (${TIMEOUT_SECONDS}s)"
        return 2
    else
        log_error "mcp-cli test failed (exit code: $exit_code, time: ${duration}s)"
        return 1
    fi
}

###############################################################################
# Main Testing Loop
###############################################################################

print_header "MCP Server Comprehensive Test Suite"
log_info "Started at: $(date)"
log_info "Timeout: ${TIMEOUT_SECONDS}s per test"
log_info "Total servers to test: ${#SERVERS[@]}"
log_info ""

# CSV Header
echo "Server,Package,TestType,ExitCode,Duration" > "$CSV_FILE"

# Test each server
for name in "${!SERVERS[@]}"; do
    ((TOTAL_SERVERS++))
    package="${SERVERS[$name]}"
    priority="${PRIORITIES[$name]:-MEDIUM}"

    print_header "[$TOTAL_SERVERS/${#SERVERS[@]}] Testing: $name (Priority: $priority)"
    log_info "Package: $package"

    # Test 1: Direct npx call
    test_server_direct "$name" "$package"
    direct_result=$?

    if [ $direct_result -eq 0 ]; then
        ((SUCCESS_DIRECT++))
    else
        ((FAILED_DIRECT++))
    fi

    # Test 2: mcp-cli validation
    test_server_mcpcli "$name" "$package"
    mcpcli_result=$?

    if [ $mcpcli_result -eq 0 ]; then
        ((SUCCESS_MCPCLI++))
    else
        ((FAILED_MCPCLI++))
    fi

    # Summary for this server
    if [ $direct_result -eq 0 ] && [ $mcpcli_result -eq 0 ]; then
        log_success "✅ BOTH TESTS PASSED"
    elif [ $direct_result -eq 0 ] || [ $mcpcli_result -eq 0 ]; then
        log_warning "⚠️  PARTIAL SUCCESS (one test passed)"
    else
        log_error "❌ BOTH TESTS FAILED"
    fi

    echo "" | tee -a "$RESULTS_FILE"

    # Small delay between servers to avoid overwhelming the system
    sleep 2
done

###############################################################################
# Generate Summary Report
###############################################################################

print_header "TEST SUMMARY"

cat > "$SUMMARY_FILE" <<EOF
# MCP Server Test Results Summary
**Generated**: $(date)
**Test Duration**: ${TIMEOUT_SECONDS}s timeout per server
**Total Servers Tested**: $TOTAL_SERVERS

---

## 📊 Overall Results

| Test Method | Success | Failed | Success Rate |
|-------------|---------|--------|--------------|
| Direct npx  | $SUCCESS_DIRECT | $FAILED_DIRECT | $(echo "scale=1; $SUCCESS_DIRECT * 100 / $TOTAL_SERVERS" | bc)% |
| mcp-cli     | $SUCCESS_MCPCLI | $FAILED_MCPCLI | $(echo "scale=1; $SUCCESS_MCPCLI * 100 / $TOTAL_SERVERS" | bc)% |

**Timeouts**: $TIMEOUT_COUNT

---

## 📈 Comparison with Original Status

**Before (from FAILED_SERVERS_TABLE.md)**:
- Failed: 88/159 (55.3%)
- Success: 71/159 (44.7%)

**After Testing**:
- Direct Success: $SUCCESS_DIRECT/88 ($(echo "scale=1; $SUCCESS_DIRECT * 100 / 88" | bc)%)
- mcp-cli Success: $SUCCESS_MCPCLI/88 ($(echo "scale=1; $SUCCESS_MCPCLI * 100 / 88" | bc)%)

---

## 📁 Output Files

- **Detailed Log**: \`$RESULTS_FILE\`
- **CSV Data**: \`$CSV_FILE\`
- **This Summary**: \`$SUMMARY_FILE\`

---

## 🔍 Analysis

EOF

# Add detailed analysis to summary
if [ $SUCCESS_DIRECT -gt 44 ]; then
    echo "✅ **Improvement**: More servers are working now than before!" >> "$SUMMARY_FILE"
else
    echo "⚠️ **Status**: Similar or worse than before. Timeout fix may be needed." >> "$SUMMARY_FILE"
fi

echo "" >> "$SUMMARY_FILE"
echo "### Top Recommendations" >> "$SUMMARY_FILE"
echo "" >> "$SUMMARY_FILE"

if [ $TIMEOUT_COUNT -gt 50 ]; then
    echo "1. **URGENT**: Increase timeout from ${TIMEOUT_SECONDS}s to 120s or more" >> "$SUMMARY_FILE"
fi

echo "2. Consider global installation for frequently failing servers" >> "$SUMMARY_FILE"
echo "3. Check environment variables for servers requiring API keys" >> "$SUMMARY_FILE"

echo "" >> "$SUMMARY_FILE"
echo "---" >> "$SUMMARY_FILE"
echo "" >> "$SUMMARY_FILE"
echo "**Next Steps**: Review \`$CSV_FILE\` for per-server timing data" >> "$SUMMARY_FILE"

# Display summary
cat "$SUMMARY_FILE"

log_info ""
log_info "Testing completed!"
log_info "Results saved to: $RESULTS_DIR"
log_info ""
log_success "✅ Direct Success: $SUCCESS_DIRECT/$TOTAL_SERVERS"
log_success "✅ mcp-cli Success: $SUCCESS_MCPCLI/$TOTAL_SERVERS"
log_error "❌ Direct Failed: $FAILED_DIRECT/$TOTAL_SERVERS"
log_error "❌ mcp-cli Failed: $FAILED_MCPCLI/$TOTAL_SERVERS"
log_warning "⏱️  Timeouts: $TIMEOUT_COUNT"
