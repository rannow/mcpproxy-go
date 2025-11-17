# Claude Code Sub-Agents Documentation

**Complete guide** to creating, using, and managing specialized Claude Code sub-agents.

---

## Quick Links

| Resource | Purpose | Time Required |
|----------|---------|---------------|
| **[Quickstart Guide](../AGENT_CREATION_QUICKSTART.md)** | Create your first agent in 5 minutes | 5-10 min |
| **[Complete Creation Guide](../CREATE_NEW_AGENT.md)** | Comprehensive agent development guide | 30-60 min |
| **[Agent Template](agent-template.md)** | Copy-paste template for documentation | 2 min |
| **[Example: Config Validator](config-validator-agent.md)** | Working agent implementation | 10 min read |

---

## What Are Sub-Agents?

Sub-agents are specialized AI assistants spawned by Claude Code to handle specific tasks:

- **Specialized Expertise**: Each agent excels at specific domains
- **Parallel Execution**: Multiple agents work concurrently
- **Coordination**: Agents share context via memory and hooks
- **Auto-Activation**: Smart routing based on task requirements

---

## Getting Started

### I want to... → You should...

| Goal | Resource | Time |
|------|----------|------|
| **Create a new agent quickly** | [Quickstart Guide](../AGENT_CREATION_QUICKSTART.md) | 5 min |
| **Understand agent architecture** | [Complete Guide](../CREATE_NEW_AGENT.md) | 30 min |
| **See a working example** | [Config Validator Agent](config-validator-agent.md) | 10 min |
| **Copy a template** | [Agent Template](agent-template.md) | 2 min |
| **Use existing agents** | [CLAUDE.md](../../CLAUDE.md) | 5 min |

---

## Available Agent Types

### Development Agents
- **coder**: General-purpose code implementation
- **backend-dev**: Server-side development
- **mobile-dev**: React Native mobile development
- **frontend-builder**: UI component creation
- **database-specialist**: Database schema and queries *(example to create)*

### Analysis Agents
- **code-analyzer**: Code quality analysis
- **perf-analyzer**: Performance optimization
- **system-architect**: System design
- **log-analyzer**: Log analysis *(example to create)*

### Quality Agents
- **tester**: Test creation and execution
- **reviewer**: Code review
- **production-validator**: Production readiness
- **config-validator**: Configuration validation *(example included)*

### Infrastructure Agents
- **cicd-engineer**: CI/CD pipelines
- **repo-architect**: Repository management
- **release-manager**: Release coordination

### Coordination Agents
- **hierarchical-coordinator**: Tree-based coordination
- **mesh-coordinator**: Peer-to-peer coordination
- **task-orchestrator**: Task management

[See full list in CLAUDE.md](../../CLAUDE.md)

---

## Creating Your First Agent

### 5-Minute Path (Quickstart)

```bash
# 1. Choose your agent type and capabilities
# 2. Copy the template
cp docs/agents/agent-template.md docs/agents/my-agent.md

# 3. Create memory directory
mkdir -p memory/agents/my-agent

# 4. Test your agent
# Spawn via Task tool in Claude Code
```

**[Full Quickstart Guide →](../AGENT_CREATION_QUICKSTART.md)**

### 30-Minute Path (Comprehensive)

Learn about:
- Agent specification and design
- Tool orchestration strategies
- MCP server integration
- Coordination protocols
- Performance optimization
- Auto-activation tuning

**[Complete Creation Guide →](../CREATE_NEW_AGENT.md)**

---

## Agent Development Workflow

```
1. Define Specification
   ↓
2. Create Documentation (use template)
   ↓
3. Setup Memory Structure
   ↓
4. Register in CLAUDE.md
   ↓
5. Test Basic Functionality
   ↓
6. Calibrate Performance
   ↓
7. Document Learning
   ↓
8. Deploy & Monitor
```

---

## Agent Coordination

### How Agents Work Together

```javascript
// Example: Multi-agent workflow
[Single Message - All agents spawned concurrently]:
  Task("Architect", "Design system architecture", "system-architect")
  Task("Database Specialist", "Design database schema", "database-specialist")
  Task("Backend Dev", "Implement API layer", "backend-dev")
  Task("Tester", "Create test suite", "tester")

  TodoWrite { todos: [
    {content: "Architecture design", status: "in_progress"},
    {content: "Database schema", status: "in_progress"},
    {content: "API implementation", status: "pending"},
    {content: "Test suite", status: "pending"}
  ]}
```

### Coordination via Hooks

```bash
# Pre-task: Load context
npx claude-flow@alpha hooks session-restore --session-id "swarm-001"

# During: Update memory
npx claude-flow@alpha hooks post-edit --file "schema.sql" --memory-key "swarm/db/schema"

# Post-task: Save results
npx claude-flow@alpha hooks post-task --task-id "db-design"
```

---

## Best Practices

### ✅ DO
- **Batch Operations**: All related operations in single messages
- **Use Hooks**: Pre-task, post-edit, post-task for coordination
- **Update Memory**: Store decisions and progress
- **Provide Evidence**: Code references and validation
- **Auto-Activate**: Set smart triggers for common tasks

### ❌ DON'T
- **Multiple Messages**: Split operations across messages
- **Skip Coordination**: Work without hooks/memory
- **Ignore Context**: Fail to check prior decisions
- **Work Alone**: Agents should coordinate with related specialists
- **Over-Activate**: Set activation threshold too low

---

## Agent Performance Targets

| Metric | Target | Good | Needs Work |
|--------|--------|------|------------|
| Token Efficiency | 5K-15K | <20K | >25K |
| Execution Time | <120s | <180s | >300s |
| Success Rate | >85% | >75% | <70% |
| Auto-Activation Accuracy | >80% | >70% | <60% |

---

## File Structure

```
docs/
├── CREATE_NEW_AGENT.md              # Complete creation guide
├── AGENT_CREATION_QUICKSTART.md     # 5-minute quickstart
└── agents/
    ├── README.md                    # This file
    ├── agent-template.md            # Copy-paste template
    ├── config-validator-agent.md    # Example agent
    └── {your-agent}.md              # Your agents here

memory/
└── agents/
    ├── {agent-name}/
    │   ├── state.json               # Agent state
    │   ├── knowledge.md             # Learned patterns
    │   ├── tasks.json               # Task history
    │   └── calibration.json         # Performance tuning
    └── shared/
        ├── common_knowledge.md      # Cross-agent knowledge
        └── global_config.json       # Global settings

CLAUDE.md                            # Main agent registry
```

---

## Common Agent Patterns

### Pattern 1: Specialist Collaboration
```javascript
// Multiple specialists working in parallel
Task("Database Specialist", "Design schema", "database-specialist")
Task("API Specialist", "Design endpoints", "api-docs")
Task("Security Specialist", "Design auth", "code-analyzer")
```

### Pattern 2: Wave-Based Execution
```javascript
// Sequential waves for dependencies
[Wave 1]: Task("Analyzer", "Analyze requirements", "analyzer")
[Wave 2]: Task("Coder", "Implement solution", "coder")
[Wave 3]: Task("Tester", "Validate implementation", "tester")
```

### Pattern 3: Coordinator Pattern
```javascript
// Coordinator manages specialist agents
Task("Project Coordinator", "Coordinate full-stack development", "task-orchestrator")
  // Coordinator spawns sub-agents as needed
```

---

## Troubleshooting

### Agent Not Working?

| Problem | Solution |
|---------|----------|
| Not auto-activating | Lower confidence threshold in calibration.json |
| Poor performance | Enable compression, optimize tool selection |
| Coordination failing | Verify hooks are executed, check memory keys |
| Wrong output | Review agent capabilities, may need specialist |

### Debug Commands

```bash
# Check agent state
cat memory/agents/{agent-name}/state.json

# Monitor hook execution
tail -f ~/.claude-flow/logs/*.log

# View memory updates
cat ~/.claude-flow/memory/swarm/*.json

# Check spawned processes
ps aux | grep claude-flow
```

---

## Examples Gallery

### Example 1: Config Validator
**What it does**: Validates YAML, JSON, TOML, ENV files
**When to use**: Pre-deployment validation, security audits
**[View Full Documentation →](config-validator-agent.md)**

### Example 2: Database Specialist *(to be created)*
**What it does**: Schema design, migrations, query optimization
**When to use**: Database design, performance tuning

### Example 3: Log Analyzer *(to be created)*
**What it does**: Parse logs, detect patterns, generate insights
**When to use**: Debugging, performance analysis

### Add Your Example!
Create your agent and add it to this gallery.

---

## Contributing

### Want to Create a New Agent?

1. Read the [Quickstart Guide](../AGENT_CREATION_QUICKSTART.md)
2. Use the [Agent Template](agent-template.md)
3. Follow the [Complete Guide](../CREATE_NEW_AGENT.md) for advanced features
4. Test thoroughly before deployment
5. Document your learnings

### Improving Existing Agents

1. Update agent documentation with new patterns
2. Add to `knowledge.md` in agent's memory directory
3. Tune `calibration.json` based on performance data
4. Share successful workflows in examples

---

## Resources

### Documentation
- **[CLAUDE.md](../../CLAUDE.md)** - Main agent configuration
- **[AGENT_API.md](../AGENT_API.md)** - REST API for agents
- **[MCP_AGENT_DESIGN.md](../MCP_AGENT_DESIGN.md)** - System architecture

### External
- **Claude Flow**: https://github.com/ruvnet/claude-flow
- **Claude Code**: https://claude.com/claude-code
- **MCP Specification**: https://modelcontextprotocol.io/

---

## FAQ

**Q: How many agents can run at once?**
A: Default is 8 concurrent agents, configurable in swarm settings.

**Q: Can agents spawn sub-agents?**
A: Yes! Use the Task tool within an agent to spawn specialists.

**Q: How do I share data between agents?**
A: Use claude-flow hooks and memory system with consistent keys.

**Q: Can I use agents in CI/CD?**
A: Yes! Agents can be triggered via CLI and API calls.

**Q: How do I version control agents?**
A: Agent documentation lives in `docs/agents/`, commit to git.

---

## What's Next?

### Ready to Build?
1. **[Start with Quickstart →](../AGENT_CREATION_QUICKSTART.md)** (5 min)
2. **[Copy Template →](agent-template.md)** (2 min)
3. **[Study Example →](config-validator-agent.md)** (10 min)
4. **Create your first agent!** 🚀

### Want to Go Deeper?
1. **[Read Complete Guide →](../CREATE_NEW_AGENT.md)** (30 min)
2. **[Study Architecture →](../MCP_AGENT_DESIGN.md)** (45 min)
3. **[Understand API →](../AGENT_API.md)** (20 min)

---

**Last Updated**: 2025-11-17
**Total Agents**: 54+ (and growing!)

---

*Happy agent building! Need help? Check the documentation or ask Claude Code for assistance.*
