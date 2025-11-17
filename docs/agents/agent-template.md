# {Agent Name} Agent - Template

> **Copy this template** to create your own agent documentation.
> **Replace all {placeholders}** with your agent-specific information.

---

## Purpose
{Brief 1-2 sentence description of what this agent does and when to use it}

---

## Capabilities

### Primary Capabilities
- **{Capability 1}**: {Detailed description}
- **{Capability 2}**: {Detailed description}
- **{Capability 3}**: {Detailed description}

### Secondary Capabilities
- **{Capability 4}**: {Detailed description}
- **{Capability 5}**: {Detailed description}

---

## When to Use

Use this agent for:
- ✅ {Use case 1}
- ✅ {Use case 2}
- ✅ {Use case 3}

Do NOT use this agent for:
- ❌ {Non-use case 1}
- ❌ {Non-use case 2}

---

## Tool Orchestration

### Claude Code Tools
- **Read/Grep/Glob**: {How agent uses search tools}
- **Write/Edit/MultiEdit**: {How agent modifies files}
- **Bash**: {What commands agent executes}
- **TodoWrite**: {How agent tracks progress}
- **Task**: {When agent spawns sub-agents}

### MCP Server Integration
- **Primary MCP**: {Main MCP server} - {Why and how used}
- **Secondary MCP**: {Fallback server} - {Why and how used}
- **Optional MCP**: {Additional servers} - {When used}

---

## Auto-Activation

### Triggers
- **Keywords**: `{keyword1}`, `{keyword2}`, `{keyword3}`
- **File Patterns**: `{*.ext}`, `{directory/*}`
- **Domain Indicators**: `{domain-term-1}`, `{domain-term-2}`
- **Complexity Threshold**: `{0.6-0.8}`

### Confidence Matrix

| Trigger Type | Match | Confidence | Action |
|-------------|-------|-----------|---------|
| Keyword exact match | 3+ | 90% | Auto-spawn immediately |
| File pattern match | 5+ files | 85% | Auto-spawn with context |
| Domain context | Clear | 80% | Suggest to user |
| Complexity only | >0.8 | 70% | Suggest with alternatives |

---

## Spawning Instructions

### Recommended: Task Tool
```javascript
Task("{Agent Name}",
     "{Detailed task description with context and requirements...}",
     "{agent-type-identifier}")
```

### Alternative: Manual Invocation
```bash
# Step 1: Pre-task hook
npx claude-flow@alpha hooks pre-task \
  --description "{task description}"

# Step 2: Agent work (varies by agent)
# ... agent-specific operations ...

# Step 3: Post-task hook
npx claude-flow@alpha hooks post-task \
  --task-id "{task-id}"
```

---

## Coordination Protocol

### Before Starting Work
```bash
# 1. Load previous context
npx claude-flow@alpha hooks session-restore \
  --session-id "swarm-{session-id}"

# 2. Announce task start
npx claude-flow@alpha hooks pre-task \
  --description "{agent-name}: {task}"

# 3. Read memory for dependencies
npx claude-flow@alpha memory read \
  --key "swarm/{related-agent}/{context}"
```

### During Work
```bash
# Update memory after each significant step
npx claude-flow@alpha hooks post-edit \
  --file "{modified-file}" \
  --memory-key "swarm/{agent-name}/{step-name}"

# Notify other agents of progress
npx claude-flow@alpha hooks notify \
  --message "{agent-name}: Completed {step}"
```

### After Completing Work
```bash
# 1. Save final results to memory
npx claude-flow@alpha memory store \
  --key "swarm/{agent-name}/result" \
  --value "{result-summary}"

# 2. Mark task complete
npx claude-flow@alpha hooks post-task \
  --task-id "{task-id}"

# 3. Export session state
npx claude-flow@alpha hooks session-end \
  --export-metrics true
```

---

## Example Workflows

### Workflow 1: {Primary Use Case Name}

**Scenario**: {Describe the scenario}

**Execution**:
```javascript
[Single Message - All Operations Together]:
  // Spawn all agents in parallel
  Task("{Related Agent 1}", "{Task 1 instructions...}", "{agent-type-1}")
  Task("{Your Agent}", "{Your agent's task...}", "{your-agent-type}")
  Task("{Related Agent 2}", "{Task 2 instructions...}", "{agent-type-2}")

  // Batch all todos together
  TodoWrite { todos: [
    {content: "{Task 1}", status: "in_progress", activeForm: "{Doing Task 1}"},
    {content: "{Task 2}", status: "in_progress", activeForm: "{Doing Task 2}"},
    {content: "{Task 3}", status: "pending", activeForm: "{Doing Task 3}"},
    {content: "{Task 4}", status: "pending", activeForm: "{Doing Task 4}"}
  ]}

  // Batch all file operations
  Read "{file1.ext}"
  Read "{file2.ext}"
  Write "{output.ext}"
  Edit "{existing.ext}"
```

**Expected Output**:
- {Expected output 1}
- {Expected output 2}
- {Expected output 3}

---

### Workflow 2: {Secondary Use Case Name}

**Scenario**: {Describe the scenario}

**Execution**:
```javascript
[Wave 1 - Analysis]:
  Task("Analyzer", "{Analysis task}", "analyzer")

[Wave 2 - Implementation]:
  Task("{Your Agent}", "{Implementation task}", "{your-agent-type}")
  Task("Coder", "{Supporting code}", "coder")

[Wave 3 - Validation]:
  Task("Tester", "{Test task}", "tester")
  Task("Reviewer", "{Review task}", "reviewer")
```

**Expected Output**:
- {Expected output 1}
- {Expected output 2}

---

## Quality Standards

### Validation Criteria
- ✅ {Validation criterion 1}
- ✅ {Validation criterion 2}
- ✅ {Validation criterion 3}

### Evidence Requirements
- 📊 {Evidence type 1}: {Description}
- 📊 {Evidence type 2}: {Description}
- 📊 {Evidence type 3}: {Description}

### Success Metrics
| Metric | Target | Measurement |
|--------|--------|-------------|
| {Metric 1} | {Target value} | {How measured} |
| {Metric 2} | {Target value} | {How measured} |
| {Metric 3} | {Target value} | {How measured} |

---

## Integration with Other Agents

### Receives Input From
- **{Agent 1}**: {What input and why}
- **{Agent 2}**: {What input and why}

### Provides Output To
- **{Agent 1}**: {What output and why}
- **{Agent 2}**: {What output and why}

### Coordinates With
- **{Agent 1}**: {How they coordinate}
- **{Agent 2}**: {How they coordinate}

---

## Performance Benchmarks

### Token Efficiency
- **Target Range**: {5,000-15,000} tokens
- **Average**: {10,000} tokens
- **Optimization**: {Compression enabled/disabled}

### Execution Time
- **Target**: {120} seconds
- **Timeout**: {300} seconds
- **Average**: {90} seconds

### Success Rate
- **Target**: {85%}
- **Current**: {Update after testing}
- **Improvement Actions**: {List of improvements}

---

## Common Issues & Solutions

### Issue 1: {Issue Name}
**Symptoms**: {Description of symptoms}
**Root Cause**: {Why it happens}
**Solution**: {Step-by-step fix}

### Issue 2: {Issue Name}
**Symptoms**: {Description of symptoms}
**Root Cause**: {Why it happens}
**Solution**: {Step-by-step fix}

### Issue 3: {Issue Name}
**Symptoms**: {Description of symptoms}
**Root Cause**: {Why it happens}
**Solution**: {Step-by-step fix}

---

## Knowledge Base

### Learned Patterns
1. **Pattern**: {Pattern name}
   - **Context**: {When to use}
   - **Implementation**: {How to implement}
   - **Benefits**: {Why it works}

2. **Pattern**: {Pattern name}
   - **Context**: {When to use}
   - **Implementation**: {How to implement}
   - **Benefits**: {Why it works}

### Best Practices
- 📌 {Best practice 1}
- 📌 {Best practice 2}
- 📌 {Best practice 3}
- 📌 {Best practice 4}

### Anti-Patterns
- ⚠️ {Anti-pattern 1}: {Why to avoid}
- ⚠️ {Anti-pattern 2}: {Why to avoid}
- ⚠️ {Anti-pattern 3}: {Why to avoid}

---

## Configuration

### Agent State (`memory/agents/{agent-name}/state.json`)
```json
{
  "agent_id": "{agent-name}-001",
  "agent_type": "{agent-type-identifier}",
  "status": "active",
  "capabilities": [
    "{capability-1}",
    "{capability-2}",
    "{capability-3}"
  ],
  "created_at": "2025-11-17T10:00:00Z",
  "last_active": "2025-11-17T10:30:00Z",
  "metrics": {
    "tasks_completed": 0,
    "success_rate": 0.0,
    "avg_execution_time": 0,
    "token_efficiency": 0.0
  },
  "preferences": {
    "compression": true,
    "mcp_primary": "{primary-mcp}",
    "auto_activate": true
  }
}
```

### Calibration (`memory/agents/{agent-name}/calibration.json`)
```json
{
  "token_efficiency": {
    "target_range": "{5000-15000}",
    "compression_enabled": true,
    "batch_operations": true
  },
  "execution_time": {
    "target_seconds": 120,
    "timeout_seconds": 300
  },
  "quality_thresholds": {
    "validation_score": 0.95,
    "test_coverage": 0.90,
    "success_rate": 0.85
  },
  "auto_activation": {
    "confidence_threshold": 0.75,
    "keyword_weight": 0.3,
    "context_weight": 0.4,
    "history_weight": 0.2,
    "performance_weight": 0.1
  },
  "mcp_servers": {
    "primary": "{primary-mcp-server}",
    "secondary": "{secondary-mcp-server}",
    "fallback": ["context7", "sequential"]
  }
}
```

---

## Testing

### Unit Tests
```bash
# Test basic agent spawning
Task("{Agent Name} Test",
     "Test basic capabilities: {specific test scenario}",
     "{agent-type}")

# Verify output
ls -la {expected-output-files}
cat {expected-output-file}
```

### Integration Tests
```bash
# Test coordination with other agents
[Single Message]:
  Task("Architect", "Design {system}", "system-architect")
  Task("{Your Agent}", "Implement {feature}", "{your-agent-type}")
  Task("Tester", "Validate {feature}", "tester")

# Verify coordination
cat ~/.claude-flow/memory/swarm/*.json
tail -f ~/.claude-flow/logs/*.log
```

### Performance Tests
```bash
# Measure token usage
# Expected: {5000-15000} tokens

# Measure execution time
# Expected: <{120} seconds

# Measure success rate
# Expected: >{85%}
```

---

## Changelog

### Version 1.0.0 (2025-11-17)
- Initial agent creation
- Basic capabilities implemented
- Auto-activation configured

### Version 1.1.0 (YYYY-MM-DD)
- {Enhancement 1}
- {Enhancement 2}
- {Bug fix 1}

---

## Resources

- **CLAUDE.md**: Main agent configuration
- **CREATE_NEW_AGENT.md**: Agent creation guide
- **AGENT_API.md**: REST API documentation
- **MCP_AGENT_DESIGN.md**: System architecture

---

## License

{Your license information if applicable}

---

## Maintainers

- **Primary**: {Your name/team}
- **Contributors**: {List of contributors}

---

**Last Updated**: {YYYY-MM-DD}
**Status**: {Active / Beta / Deprecated}
