# Creating a New Claude Code Sub Agent

## Overview

This guide walks you through creating a new specialized sub agent for Claude Code. Sub agents are spawned using Claude Code's Task tool and coordinate via claude-flow hooks and memory.

## Quick Start

**3-Step Process:**
1. Define agent specification (capabilities, tools, coordination)
2. Create agent template files
3. Register agent in CLAUDE.md

---

## 1. Agent Specification Template

```yaml
---
agent_name: "my-custom-agent"
agent_type: "specialist"  # specialist, coordinator, orchestrator
category: "Development"   # Development, Analysis, Quality, Infrastructure
purpose: "Specific task this agent excels at"
---

Capabilities:
  - Primary capability 1
  - Primary capability 2
  - Primary capability 3

Tools (Claude Code):
  - Read, Write, Edit, MultiEdit  # File operations
  - Grep, Glob                    # Search operations
  - Bash                          # Terminal commands
  - TodoWrite                     # Task management
  - Task                          # Spawn sub-agents

MCP Integration:
  - Primary: Sequential           # Main MCP server to use
  - Secondary: Context7           # Fallback/complementary
  - Optional: Magic, Playwright   # Additional if needed

Coordination:
  - Pre-task: Load context and validate requirements
  - During: Update memory with progress
  - Post-task: Save results and notify other agents

Auto-Activation Triggers:
  - Keywords: ["keyword1", "keyword2", "keyword3"]
  - File patterns: ["*.ext", "directory/*"]
  - Complexity threshold: 0.6-0.8
  - Domain indicators: ["domain-specific terms"]
```

---

## 2. Agent Template Files

### 2.1 Agent Documentation (`docs/agents/{agent-name}.md`)

```markdown
# {Agent Name} - Specialized Agent

## Purpose
Brief description of what this agent does and when to use it.

## Capabilities
- **Primary Capability**: Detailed description
- **Secondary Capability**: Detailed description
- **Integration Capability**: How it works with other agents

## When to Use
- Use case 1: Specific scenario
- Use case 2: Specific scenario
- Use case 3: Specific scenario

## Tool Orchestration
Primary tools and how they're used:
- **Read/Grep/Glob**: Discovery and analysis
- **Edit/Write**: Implementation
- **Bash**: Execution and validation
- **TodoWrite**: Progress tracking

## MCP Coordination
- **Sequential**: Complex analysis and planning
- **Context7**: Documentation and patterns
- **Magic**: UI component generation (if applicable)

## Spawning Instructions

### Via Task Tool (Recommended)
\`\`\`javascript
Task("Agent description",
     "Detailed task instructions with context...",
     "agent-type-name")
\`\`\`

### Manual Invocation
\`\`\`bash
# Agent-specific command pattern
npx claude-flow@alpha hooks pre-task --description "task"
# ... agent work ...
npx claude-flow@alpha hooks post-task --task-id "task"
\`\`\`

## Coordination Protocol

### Before Work
\`\`\`bash
npx claude-flow@alpha hooks pre-task --description "{task}"
npx claude-flow@alpha hooks session-restore --session-id "swarm-{id}"
\`\`\`

### During Work
\`\`\`bash
npx claude-flow@alpha hooks post-edit --file "{file}" --memory-key "swarm/{agent}/{step}"
npx claude-flow@alpha hooks notify --message "{progress update}"
\`\`\`

### After Work
\`\`\`bash
npx claude-flow@alpha hooks post-task --task-id "{task}"
npx claude-flow@alpha hooks session-end --export-metrics true
\`\`\`

## Example Workflows

### Workflow 1: {Primary Use Case}
\`\`\`javascript
[Single Message - Parallel Execution]:
  Task("Agent 1", "Task 1 instructions...", "agent-type")
  Task("{Your Agent}", "{Task instructions for your agent}...", "{agent-name}")
  Task("Agent 3", "Task 3 instructions...", "agent-type")

  TodoWrite { todos: [...all todos batched...] }

  // All file operations together
  Read "file1.js"
  Read "file2.js"
  Write "output.js"
\`\`\`

### Workflow 2: {Secondary Use Case}
Detailed example of another common usage pattern.

## Quality Standards
- **Validation**: How agent validates its work
- **Evidence**: What evidence agent provides
- **Metrics**: Success criteria and measurements

## Integration with Other Agents
- **Architect**: Receives design guidance
- **Coder**: Provides implementation details
- **Tester**: Coordinates testing validation
- **Reviewer**: Submits work for review

## Auto-Activation Matrix

| Trigger | Confidence | Action |
|---------|-----------|---------|
| Keyword match | 85% | Direct activation |
| File pattern | 75% | Conditional activation |
| Domain context | 90% | Auto-spawn with others |

## Performance Benchmarks
- **Token efficiency**: Target range
- **Execution time**: Expected duration
- **Success rate**: Historical performance
```

### 2.2 Agent Memory Structure (`memory/agents/{agent-name}/`)

```
memory/agents/{agent-name}/
├── state.json           # Agent state and configuration
├── knowledge.md         # Agent-specific knowledge base
├── tasks.json          # Completed and active tasks
├── calibration.json    # Performance tuning
└── examples/           # Example executions
    ├── example-1.md
    └── example-2.md
```

**state.json:**
```json
{
  "agent_id": "agent-name-001",
  "agent_type": "my-custom-agent",
  "status": "active",
  "capabilities": [
    "capability1",
    "capability2",
    "capability3"
  ],
  "created_at": "2025-11-17T10:00:00Z",
  "last_active": "2025-11-17T10:30:00Z",
  "metrics": {
    "tasks_completed": 0,
    "success_rate": 0.0,
    "avg_execution_time": 0
  }
}
```

**knowledge.md:**
```markdown
# {Agent Name} Knowledge Base

## Learned Patterns
- Pattern 1: Description and when to use
- Pattern 2: Description and when to use

## Common Issues & Solutions
1. **Issue**: Description
   **Solution**: Steps to resolve

2. **Issue**: Description
   **Solution**: Steps to resolve

## Best Practices
- Best practice 1
- Best practice 2
- Best practice 3

## Integration Notes
- How this agent works with Architect
- How this agent works with Coder
- How this agent works with Tester
```

---

## 3. Register Agent in CLAUDE.md

Add your agent to the main configuration:

```markdown
## 🚀 Available Agents (55 Total)

### Your Category
`your-agent-name`, `other-agents`

...

### Agent Descriptions

**`your-agent-name`** - Brief description
- **Purpose**: What it does
- **Auto-Activates**: When it auto-spawns
- **Coordinates With**: Related agents
```

---

## 4. Example: Creating a "Database Specialist" Agent

### 4.1 Specification

```yaml
---
agent_name: "database-specialist"
agent_type: "specialist"
category: "Development"
purpose: "Database schema design, migration, and optimization"
---

Capabilities:
  - Design database schemas with proper normalization
  - Create and validate database migrations
  - Optimize queries and indexes for performance
  - Generate SQL/ORM code for various databases

Tools:
  - Read, Write, Edit  # Schema files, migrations
  - Grep, Glob         # Search existing schemas
  - Bash               # Run migrations, tests
  - TodoWrite          # Track migration steps

MCP Integration:
  - Primary: Sequential      # Complex schema analysis
  - Secondary: Context7      # Database best practices

Coordination:
  - Pre-task: Read existing schema and database config
  - During: Store schema decisions in memory
  - Post-task: Generate migration files and documentation

Auto-Activation:
  - Keywords: ["database", "schema", "migration", "SQL"]
  - File patterns: ["*.sql", "migrations/*", "schema/*"]
  - Complexity: 0.6
  - Domain: ["database", "PostgreSQL", "MySQL", "MongoDB"]
```

### 4.2 Documentation (`docs/agents/database-specialist.md`)

```markdown
# Database Specialist Agent

## Purpose
Specialized in database schema design, migrations, query optimization, and ensuring data integrity across SQL and NoSQL databases.

## Capabilities
- **Schema Design**: Create normalized, scalable database schemas
- **Migration Management**: Generate and validate database migrations
- **Query Optimization**: Analyze and improve query performance
- **Data Modeling**: Design efficient data models for different databases

## When to Use
- Designing new database schemas
- Creating database migrations
- Optimizing slow queries
- Database refactoring or normalization
- Multi-database integration

## Spawning Instructions

\`\`\`javascript
Task("Database Specialist",
     "Design PostgreSQL schema for user management system with authentication, profiles, and permissions. Include migrations and indexes.",
     "database-specialist")
\`\`\`

## Example Workflow

\`\`\`javascript
[Single Message]:
  Task("Architect", "Design overall system architecture", "system-architect")
  Task("Database Specialist", "Design database schema for user/auth system", "database-specialist")
  Task("Backend Developer", "Implement API endpoints", "backend-dev")

  TodoWrite { todos: [
    {content: "Design system architecture", status: "in_progress"},
    {content: "Create database schema", status: "in_progress"},
    {content: "Implement API layer", status: "pending"},
    {content: "Write integration tests", status: "pending"}
  ]}
\`\`\`

## Quality Standards
- **Normalization**: Minimum 3NF for relational databases
- **Performance**: All queries under 100ms with proper indexes
- **Validation**: Schema validates against database constraints
- **Documentation**: Complete ERD diagrams and data dictionary
```

---

## 5. Testing Your New Agent

### 5.1 Test Spawn

```javascript
// Test basic spawning
Task("Test Agent",
     "Test task to validate agent capabilities and coordination. Should use hooks and update memory.",
     "your-agent-name")
```

### 5.2 Test Coordination

```bash
# Monitor hook execution
tail -f ~/.claude-flow/logs/*.log

# Check memory updates
cat ~/.claude-flow/memory/swarm/*.json

# Verify agent state
cat memory/agents/your-agent-name/state.json
```

### 5.3 Test Integration

```javascript
// Test with other agents
[Single Message]:
  Task("Architect", "Design system", "system-architect")
  Task("Your Agent", "Execute specialized task", "your-agent-name")
  Task("Reviewer", "Review output", "reviewer")
```

---

## 6. Optimization & Calibration

### 6.1 Performance Tuning (`calibration.json`)

```json
{
  "token_efficiency": {
    "target_range": "5000-15000",
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
  }
}
```

### 6.2 Learning from Execution

After each execution, update `knowledge.md`:

```markdown
## Execution Log: 2025-11-17

### What Worked
- Using Sequential for complex analysis improved accuracy by 40%
- Batching file operations reduced execution time by 2x
- Pre-loading context from memory reduced redundant reads

### What Didn't Work
- Auto-activation was too aggressive (adjusted threshold from 0.6 → 0.75)
- Needed more coordination with Backend agent for API contracts
- Memory keys were too generic (now using hierarchical structure)

### Improvements Applied
- Updated auto-activation threshold in calibration.json
- Added explicit Backend coordination step
- Changed memory key pattern to: swarm/{agent}/{domain}/{step}
```

---

## 7. Best Practices

### ✅ DO
- **Batch Operations**: Always batch file operations and tool calls in single messages
- **Use Hooks**: Integrate pre-task, post-edit, and post-task hooks
- **Update Memory**: Store decisions and progress in memory for coordination
- **Provide Evidence**: Document decisions with code references and validation
- **Coordinate**: Use memory and hooks to communicate with other agents
- **Track Progress**: Use TodoWrite for multi-step workflows

### ❌ DON'T
- **Multiple Messages**: Never split related operations across messages
- **Skip Hooks**: Don't skip coordination hooks (breaks multi-agent workflows)
- **Ignore Context**: Always check memory for prior decisions
- **Work in Isolation**: Coordinate with relevant agents via memory/hooks
- **Forget Validation**: Always validate output before marking complete

---

## 8. Common Patterns

### Pattern 1: Analysis → Implementation → Validation
```javascript
[Wave 1 - Analysis]:
  Task("Analyzer", "Analyze requirements", "analyzer")

[Wave 2 - Implementation]:
  Task("Your Agent", "Implement solution", "your-agent-name")
  Task("Coder", "Implement supporting code", "coder")

[Wave 3 - Validation]:
  Task("Tester", "Validate implementation", "tester")
  Task("Reviewer", "Review quality", "reviewer")
```

### Pattern 2: Parallel Specialists
```javascript
[Single Message - Parallel]:
  Task("Database Specialist", "Design schema", "database-specialist")
  Task("API Specialist", "Design endpoints", "api-docs")
  Task("Security Specialist", "Design auth", "code-analyzer")

  // Coordinator reviews results
  Task("Architect", "Integrate designs", "system-architect")
```

### Pattern 3: Iterative Refinement
```javascript
// Loop with --loop flag or explicit iterations
for iteration in [1, 2, 3]:
  Task("Your Agent", f"Refine solution (iteration {iteration})", "your-agent-name")
  Task("Reviewer", "Review and provide feedback", "reviewer")
```

---

## 9. Advanced Features

### 9.1 Multi-Agent Memory Coordination

```javascript
// Agent 1 stores decision
npx claude-flow@alpha hooks post-edit \
  --file "schema.sql" \
  --memory-key "swarm/database/schema/users"

// Agent 2 reads decision
npx claude-flow@alpha hooks session-restore \
  --session-id "swarm-database"
```

### 9.2 Conditional Auto-Activation

```yaml
Auto-Activation Logic:
  IF keywords_match >= 2 AND (file_pattern_match OR domain_match):
    confidence = 0.85
    spawn_agent()
  ELIF complexity > 0.8 AND domain_match:
    confidence = 0.75
    spawn_agent()
  ELSE:
    confidence = 0.5
    suggest_agent()
```

### 9.3 Dynamic Tool Selection

```python
# Agent intelligence layer
def select_tools(task_complexity, domain):
    if complexity > 0.8:
        return ["Sequential", "Context7", "All MCP"]
    elif domain == "frontend":
        return ["Magic", "Context7"]
    elif domain == "backend":
        return ["Sequential", "Context7"]
    else:
        return ["Sequential"]
```

---

## 10. Troubleshooting

### Agent Not Auto-Activating
- **Check Keywords**: Are keywords in auto-activation spec?
- **Check Confidence**: Is threshold too high? (lower to 0.6-0.7)
- **Check Context**: Is domain context clear enough?
- **Manual Spawn**: Use explicit Task() call to test

### Poor Performance
- **Token Usage**: Enable compression with --uc flag
- **Execution Time**: Break into smaller sub-tasks
- **Tool Selection**: Use lighter MCP servers (Context7 vs Sequential)
- **Batching**: Ensure all operations batched in single messages

### Coordination Issues
- **Hooks Missing**: Verify pre/post hooks are called
- **Memory Keys**: Check key naming consistency
- **Session ID**: Ensure all agents use same session ID
- **Event Bus**: Verify event publishing is working

---

## 11. Complete Example: Building "API Documentation Agent"

See full example in: `docs/agents/examples/api-docs-agent-example.md`

---

## Next Steps

1. **Define Your Agent**: Fill out the specification template
2. **Create Documentation**: Write agent docs in `docs/agents/`
3. **Setup Memory**: Create directory in `memory/agents/`
4. **Register**: Add to CLAUDE.md agent list
5. **Test**: Spawn and validate functionality
6. **Calibrate**: Tune performance and auto-activation
7. **Document Learning**: Update knowledge base

---

## Resources

- **Main Config**: `CLAUDE.md` - All agents and configuration
- **Agent API**: `docs/AGENT_API.md` - REST API for agents
- **Design Doc**: `docs/MCP_AGENT_DESIGN.md` - System architecture
- **Memory**: `memory/agents/README.md` - Memory structure
- **Hooks**: `docs/hooks/` - Hook system documentation
