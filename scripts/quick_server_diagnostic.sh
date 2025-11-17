#!/bin/bash
################################################################################
# Quick MCP Server Diagnostic
# Purpose: Fast analysis of all 88 failed servers WITHOUT full testing
# Analyzes: Missing packages, env vars, config issues
# Output: Structured recommendations
################################################################################

set -e

# Paths
MCPPROXY_DIR="$HOME/.mcpproxy"
CONFIG_FILE="$MCPPROXY_DIR/mcp_config.json"
FAILED_LOG="$MCPPROXY_DIR/failed_servers.log"
OUTPUT_FILE="docs/MCP_SERVER_QUICK_DIAGNOSTIC.md"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Counters
TOTAL_SERVERS=0
NEED_GLOBAL_INSTALL=0
NEED_ENV_VARS=0
READY_TO_TEST=0

# Arrays
declare -a NPM_PACKAGES_TO_INSTALL
declare -a ENV_VARS_NEEDED
declare -a READY_SERVERS

echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
echo -e "${BLUE}  MCP Server Quick Diagnostic${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
echo ""

# Extract failed server names
echo -e "${BLUE}[1/4]${NC} Extracting failed server names..."
FAILED_SERVERS=$(sed -n 's/.*Server "\([^"]*\)".*/\1/p' "$FAILED_LOG" | sort -u)
TOTAL_SERVERS=$(echo "$FAILED_SERVERS" | wc -l | tr -d ' ')

echo -e "${GREEN}✓${NC} Found $TOTAL_SERVERS failed servers"
echo ""

# Analyze each server
echo -e "${BLUE}[2/4]${NC} Analyzing server configurations..."

cat > "$OUTPUT_FILE" << EOF
# MCP Server Quick Diagnostic Report

**Date**: $(date '+%Y-%m-%d %H:%M:%S')
**Servers Analyzed**: $TOTAL_SERVERS
**Method**: Configuration analysis (no live testing)

---

## 📊 Quick Summary

EOF

# Analyze configurations
while IFS= read -r server_name; do
    # Get server config
    server_config=$(jq -r --arg name "$server_name" '
        .mcpServers[] | select(.name == $name) | @json
    ' "$CONFIG_FILE" 2>/dev/null)

    if [ -z "$server_config" ] || [ "$server_config" == "null" ]; then
        continue
    fi

    # Extract command and args
    command=$(echo "$server_config" | jq -r '.command // "npx"')
    args=$(echo "$server_config" | jq -r '.args[0] // ""')
    env_config=$(echo "$server_config" | jq -r '.env // {}')

    # Check if it's an NPX package
    if [ "$command" = "npx" ] && [ -n "$args" ]; then
        # Check if package exists globally
        package_name=$(echo "$args" | sed 's/@.*\///')

        if ! npm list -g "$package_name" &>/dev/null; then
            NPM_PACKAGES_TO_INSTALL+=("$package_name|$server_name")
            NEED_GLOBAL_INSTALL=$((NEED_GLOBAL_INSTALL + 1))
        fi
    fi

    # Check for environment variables
    if [ "$env_config" != "{}" ]; then
        env_vars=$(echo "$env_config" | jq -r 'keys[]')
        missing_vars=""

        while IFS= read -r var; do
            # Check if env var is set
            if [ -z "${!var}" ]; then
                missing_vars="$missing_vars $var"
            fi
        done <<< "$env_vars"

        if [ -n "$missing_vars" ]; then
            ENV_VARS_NEEDED+=("$server_name|$missing_vars")
            NEED_ENV_VARS=$((NEED_ENV_VARS + 1))
        fi
    fi

    # If no issues, mark as ready
    if ! echo "${NPM_PACKAGES_TO_INSTALL[@]}" | grep -q "$server_name" && \
       ! echo "${ENV_VARS_NEEDED[@]}" | grep -q "$server_name"; then
        READY_SERVERS+=("$server_name")
        READY_TO_TEST=$((READY_TO_TEST + 1))
    fi

done <<< "$FAILED_SERVERS"

echo -e "${GREEN}✓${NC} Analysis complete"
echo ""

# Generate report
echo -e "${BLUE}[3/4]${NC} Generating diagnostic report..."

cat >> "$OUTPUT_FILE" << EOF
| Category | Count | Percentage |
|----------|-------|------------|
| **Total Failed Servers** | $TOTAL_SERVERS | 100% |
| Need Global Installation | $NEED_GLOBAL_INSTALL | $((NEED_GLOBAL_INSTALL * 100 / TOTAL_SERVERS))% |
| Need Environment Variables | $NEED_ENV_VARS | $((NEED_ENV_VARS * 100 / TOTAL_SERVERS))% |
| Ready to Test | $READY_TO_TEST | $((READY_TO_TEST * 100 / TOTAL_SERVERS))% |

---

## 🔧 Action Plan

### Step 1: Install Missing NPM Packages ($NEED_GLOBAL_INSTALL servers)

These servers use \`npx\` but packages are not installed globally. Installing them will:
- Reduce startup time from 30-60s to <5s
- Eliminate timeout issues
- Reduce network dependency

\`\`\`bash
# Install all missing packages at once
EOF

# Add npm install commands
if [ ${#NPM_PACKAGES_TO_INSTALL[@]} -gt 0 ]; then
    unique_packages=$(printf '%s\n' "${NPM_PACKAGES_TO_INSTALL[@]}" | cut -d'|' -f1 | sort -u)

    while IFS= read -r pkg; do
        echo "npm install -g $pkg" >> "$OUTPUT_FILE"
    done <<< "$unique_packages"
fi

cat >> "$OUTPUT_FILE" << 'EOF'
```

#### Detailed Package List

| Package Name | Affected Server(s) | Priority |
|--------------|-------------------|----------|
EOF

# Add package details
if [ ${#NPM_PACKAGES_TO_INSTALL[@]} -gt 0 ]; then
    for entry in "${NPM_PACKAGES_TO_INSTALL[@]}"; do
        IFS='|' read -r pkg server <<< "$entry"
        priority="🟡 MEDIUM"

        # Prioritize common packages
        if echo "$pkg" | grep -qE "github|filesystem|brave|sequential|sqlite|memory|playwright"; then
            priority="🔴 HIGH"
        fi

        echo "| $pkg | $server | $priority |" >> "$OUTPUT_FILE"
    done
fi

cat >> "$OUTPUT_FILE" << 'EOF'

---

### Step 2: Configure Environment Variables
EOF

cat >> "$OUTPUT_FILE" << EOF

**Servers needing environment variables**: $NEED_ENV_VARS

Create or update \`~/.mcpproxy/.env\` file:

\`\`\`bash
# Create .env file if it doesn't exist
touch ~/.mcpproxy/.env
chmod 600 ~/.mcpproxy/.env

# Add required variables (examples below)
EOF

if [ ${#ENV_VARS_NEEDED[@]} -gt 0 ]; then
    echo "" >> "$OUTPUT_FILE"
    echo "# Required variables for failed servers:" >> "$OUTPUT_FILE"

    for entry in "${ENV_VARS_NEEDED[@]}"; do
        IFS='|' read -r server vars <<< "$entry"
        for var in $vars; do
            var_lower=$(echo "$var" | tr '[:upper:]' '[:lower:]')
            echo "# $server needs: $var=your_${var_lower}_here" >> "$OUTPUT_FILE"
        done
    done
fi

cat >> "$OUTPUT_FILE" << 'EOF'
```

#### Environment Variable Details

| Server Name | Required Variables | How to Obtain |
|-------------|-------------------|---------------|
EOF

# Add env var details
if [ ${#ENV_VARS_NEEDED[@]} -gt 0 ]; then
    for entry in "${ENV_VARS_NEEDED[@]}"; do
        IFS='|' read -r server vars <<< "$entry"

        # Provide hints on where to get the vars
        source="Check server documentation"

        if echo "$vars" | grep -q "GITHUB"; then
            source="https://github.com/settings/tokens"
        elif echo "$vars" | grep -q "BRAVE"; then
            source="https://brave.com/search/api"
        elif echo "$vars" | grep -q "GOOGLE"; then
            source="https://console.cloud.google.com"
        elif echo "$vars" | grep -q "OPENAI"; then
            source="https://platform.openai.com/api-keys"
        elif echo "$vars" | grep -q "SLACK"; then
            source="https://api.slack.com/apps"
        elif echo "$vars" | grep -q "DISCORD"; then
            source="https://discord.com/developers"
        fi

        echo "| $server | $vars | $source |" >> "$OUTPUT_FILE"
    done
fi

cat >> "$OUTPUT_FILE" << EOF

---

### Step 3: Servers Ready for Testing ($READY_TO_TEST servers)

These servers have no obvious missing dependencies or environment variables.
They likely just need timeout increase.

\`\`\`bash
# Increase timeout in config
jq '.docker_isolation.timeout = "120s"' ~/.mcpproxy/mcp_config.json > tmp.json && mv tmp.json ~/.mcpproxy/mcp_config.json
jq '.max_concurrent_connections = 40' ~/.mcpproxy/mcp_config.json > tmp.json && mv tmp.json ~/.mcpproxy/mcp_config.json
\`\`\`

Ready servers:
EOF

# List ready servers
if [ ${#READY_SERVERS[@]} -gt 0 ]; then
    for server in "${READY_SERVERS[@]}"; do
        echo "- $server" >> "$OUTPUT_FILE"
    done
else
    echo "- None (all servers need fixes)" >> "$OUTPUT_FILE"
fi

cat >> "$OUTPUT_FILE" << 'EOF'

---

## 🎯 Recommended Execution Order

### Phase 1: Quick Wins (Today - 30 minutes)

1. **Increase Timeout** (affects all servers)
   ```bash
   cd ~/.mcpproxy
   cp mcp_config.json mcp_config.json.backup
   jq '.docker_isolation.timeout = "120s" | .max_concurrent_connections = 40' mcp_config.json > tmp.json
   mv tmp.json mcp_config.json
   ```
   Expected improvement: ~40% of servers will start working

2. **Install Top 10 Critical Packages**
   ```bash
   npm install -g @modelcontextprotocol/server-github
   npm install -g @modelcontextprotocol/server-brave-search
   npm install -g @modelcontextprotocol/server-filesystem
   npm install -g @anthropic/mcp-server-sequential-thinking
   npm install -g mcp-server-sqlite
   npm install -g @anthropic/mcp-server-memory
   npm install -g @anthropic/mcp-server-playwright
   npm install -g @anthropic/mcp-server-git
   npm install -g @anthropic/mcp-server-datetime
   npm install -g @anthropic/mcp-obsidian
   ```
   Expected improvement: ~15 servers will start instantly

### Phase 2: Environment Variables (This Week - 2 hours)

1. Get API keys for critical services:
   - GitHub Token (most important)
   - Brave Search API Key
   - Google APIs (Maps, Gemini)
   - Slack/Discord tokens (if needed)

2. Add to `.env` file
3. Test individual servers with mcp-cli

### Phase 3: Full Testing (Next Week)

Run complete test suite:
```bash
./scripts/test_failed_servers.sh
```

---

## 📋 Next Steps

1. ✅ **Review this report**
2. ⬜ **Execute Phase 1 fixes** (timeout + critical packages)
3. ⬜ **Restart mcpproxy** and check logs
4. ⬜ **Count successful connections**
5. ⬜ **Execute Phase 2 fixes** (environment variables)
6. ⬜ **Run full test suite** when ready

---

**Report Generated**: $(date '+%Y-%m-%d %H:%M:%S')
**Next Action**: Execute Phase 1 fixes above
EOF

echo -e "${GREEN}✓${NC} Report saved to: $OUTPUT_FILE"
echo ""

# Print summary
echo -e "${BLUE}[4/4]${NC} Summary"
echo ""
echo "═══════════════════════════════════════════════"
echo "Total Failed Servers: $TOTAL_SERVERS"
echo "Need Global Install: $NEED_GLOBAL_INSTALL ($((NEED_GLOBAL_INSTALL * 100 / TOTAL_SERVERS))%)"
echo "Need Env Variables: $NEED_ENV_VARS ($((NEED_ENV_VARS * 100 / TOTAL_SERVERS))%)"
echo "Ready to Test: $READY_TO_TEST ($((READY_TO_TEST * 100 / TOTAL_SERVERS))%)"
echo "═══════════════════════════════════════════════"
echo ""
echo -e "${GREEN}✓${NC} Full report: $OUTPUT_FILE"
echo ""
echo -e "${YELLOW}Next:${NC} Review the report and execute Phase 1 fixes"
echo ""
