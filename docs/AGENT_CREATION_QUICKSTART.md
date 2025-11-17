# Agent Creation Quickstart Guide

**5-minute guide** to create your first Claude Code sub agent.

---

## Prerequisites

- Claude Code with claude-flow installed
- Basic understanding of how agents coordinate
- mcpproxy running (for testing)

---

## Step 1: Choose Your Agent Type (30 seconds)

Pick what your agent will specialize in:

| Type | Examples | Use When |
|------|----------|----------|
| **Specialist** | database-specialist, api-docs | Focused expertise in one domain |
| **Coordinator** | project-manager, release-manager | Orchestrating multiple agents |
| **Analyst** | code-analyzer, performance-analyzer | Deep analysis and insights |
| **Builder** | frontend-builder, backend-builder | Creating and implementing |

**Your Choice**: `_________________`

---

## Step 2: Define Core Capabilities (1 minute)

List 3-5 things your agent does really well:

1. `_______________________________________`
2. `_______________________________________`
3. `_______________________________________`
4. `_______________________________________` (optional)
5. `_______________________________________` (optional)

---

## Step 3: Set Auto-Activation Triggers (1 minute)

### Keywords (what words trigger your agent?)
Example: `["database", "schema", "migration"]`

**Your keywords**: `_______________________________________`

### File Patterns (what files should activate your agent?)
Example: `["*.sql", "migrations/*", "schema/*"]`

**Your patterns**: `_______________________________________`

---

## Step 4: Copy Template & Fill In (2 minutes)

```bash
# Copy the template
cp docs/agents/agent-template.md docs/agents/YOUR-AGENT-NAME.md

# Edit with your information
# Replace all {placeholders} with your values from Steps 1-3
```

---

## Step 5: Create Memory Directory (30 seconds)

```bash
mkdir -p memory/agents/YOUR-AGENT-NAME
cd memory/agents/YOUR-AGENT-NAME

# Create initial state
cat > state.json <<EOF
{
  "agent_id": "YOUR-AGENT-NAME-001",
  "agent_type": "YOUR-AGENT-TYPE",
  "status": "active",
  "capabilities": [
    "CAPABILITY-1",
    "CAPABILITY-2",
    "CAPABILITY-3"
  ],
  "created_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "metrics": {
    "tasks_completed": 0,
    "success_rate": 0.0
  }
}
EOF

# Create knowledge base
cat > knowledge.md <<EOF
# YOUR AGENT NAME Knowledge Base

## Learned Patterns
(Will be filled as agent learns)

## Common Issues
(Will be filled during testing)

## Best Practices
(Document as you discover them)
EOF
```

---

## Step 6: Register in CLAUDE.md (30 seconds)

Add to the agent list in `CLAUDE.md`:

```markdown
## 🚀 Available Agents (55 Total)

### Your Category
`your-agent-name`, `other-agents-in-category`
```

---

## Step 7: Test Your Agent! (1 minute)

```javascript
// Spawn your agent
Task("YOUR AGENT NAME",
     "Test task: Verify agent can execute basic capabilities",
     "your-agent-type-identifier")
```

**Expected**: Agent executes and updates memory in `memory/agents/YOUR-AGENT-NAME/`

---

## Verification Checklist

After running your test, verify:

- ✅ Agent spawned successfully
- ✅ Memory directory updated with execution data
- ✅ State.json shows `tasks_completed: 1`
- ✅ No errors in claude-flow logs
- ✅ Agent produced expected output

Check logs:
```bash
# Check claude-flow logs
tail -f ~/.claude-flow/logs/*.log

# Check agent state
cat memory/agents/YOUR-AGENT-NAME/state.json

# Check memory updates
ls -la memory/agents/YOUR-AGENT-NAME/
```

---

## Next Steps

Now that your agent is working, enhance it:

1. **Add More Capabilities**: Expand what your agent can do
2. **Tune Auto-Activation**: Adjust keywords and patterns based on testing
3. **Create Examples**: Document real usage in agent docs
4. **Optimize Performance**: Tune calibration.json for better efficiency
5. **Coordinate**: Test with other agents in multi-agent workflows

---

## Example: Complete "Log Analyzer" Agent in 5 Minutes

Here's a real example walkthrough:

### Step 1: Agent Type
**Choice**: Analyst

### Step 2: Capabilities
1. Parse and analyze log files
2. Detect error patterns and anomalies
3. Generate insights and recommendations
4. Summarize log events by severity
5. Correlate errors across multiple logs

### Step 3: Triggers
**Keywords**: `["logs", "analyze", "errors", "debug", "trace"]`
**Patterns**: `["*.log", "logs/*", "*.txt"]`

### Step 4: Documentation
```bash
cp docs/agents/agent-template.md docs/agents/log-analyzer.md
# Fill in placeholders with values above
```

### Step 5: Memory Setup
```bash
mkdir -p memory/agents/log-analyzer
cd memory/agents/log-analyzer

cat > state.json <<EOF
{
  "agent_id": "log-analyzer-001",
  "agent_type": "log-analyzer",
  "status": "active",
  "capabilities": [
    "parse-logs",
    "detect-patterns",
    "generate-insights",
    "severity-summary",
    "cross-log-correlation"
  ],
  "created_at": "2025-11-17T10:00:00Z",
  "metrics": {
    "tasks_completed": 0,
    "success_rate": 0.0
  }
}
EOF
```

### Step 6: Register
Added to CLAUDE.md under "Analysis Agents"

### Step 7: Test
```javascript
Task("Log Analyzer",
     "Analyze mcpproxy.log for errors in the last hour. Identify top 5 error patterns and suggest fixes.",
     "log-analyzer")
```

### Result
✅ Agent successfully analyzed logs
✅ Found 3 error patterns
✅ Generated fix suggestions
✅ Updated memory/agents/log-analyzer/state.json
✅ Execution time: 45 seconds

---

## Common Pitfalls

### ❌ Agent Not Auto-Activating
**Fix**: Lower confidence threshold in calibration.json from 0.75 to 0.65

### ❌ Poor Performance
**Fix**: Enable compression, batch file operations, use lighter MCP servers

### ❌ Coordination Issues
**Fix**: Verify hooks are being called (check ~/.claude-flow/logs/)

### ❌ Missing Memory Updates
**Fix**: Ensure post-edit and post-task hooks are executed

---

## Resources

- **Full Guide**: `docs/CREATE_NEW_AGENT.md`
- **Template**: `docs/agents/agent-template.md`
- **Example**: `docs/agents/config-validator-agent.md`
- **Main Config**: `CLAUDE.md`

---

## Support

**Questions?** Check:
1. `docs/CREATE_NEW_AGENT.md` for detailed guidance
2. `docs/agents/config-validator-agent.md` for a working example
3. Claude-flow documentation: https://github.com/ruvnet/claude-flow

---

**Time to create your first agent**: ~5 minutes
**Total setup**: ~10 minutes including testing

Go build something amazing! 🚀
