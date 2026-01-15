# MCP Server Configuration Documentation

## Overview

This document describes the consolidated MCP server configuration for mcpproxy-go, including all activated servers, their capabilities, and integration patterns.

## Configuration Files

### Primary Configuration: `.mcp.json`
Main MCP server configuration with all activated servers and metadata.

### Settings Configuration: `.claude/settings.json`
Claude Code integration settings including hooks, permissions, and environment variables.

### Local Server Configuration: `mcp.json`
Legacy configuration file for mcpproxy local server (now consolidated into `.mcp.json`).

## Activated MCP Servers

### 1. claude-flow@alpha (Priority 1)
**Primary orchestration server** with comprehensive AI agent coordination.

**Command**: `npx claude-flow@alpha mcp start`

**Capabilities**:
- Swarm orchestration with multiple topologies (hierarchical, mesh, ring, star)
- 54 specialized agents across development, testing, architecture, and coordination
- SPARC methodology support (Specification, Pseudocode, Architecture, Refinement, Completion)
- Hooks automation for pre/post operations
- Memory management with cross-session persistence
- Neural pattern training and learning
- Performance tracking and bottleneck analysis
- GitHub integration for PR, issues, releases, and workflows

**Key Features**:
- 84.8% SWE-Bench solve rate
- 32.3% token reduction
- 2.8-4.4x speed improvement
- 27+ neural models
- Automatic topology selection
- Self-healing workflows

**Integration**:
- Fully integrated with `.claude/settings.json` hooks
- Pre-operation validation and resource preparation
- Post-operation formatting and memory updates
- Session lifecycle management

### 2. flow-nexus (Priority 2)
**Cloud-based AI swarm deployment** with 70+ specialized MCP tools.

**Command**: `npx flow-nexus@latest mcp start`

**Capabilities**:
- Cloud sandbox execution (E2B integration)
- Neural network training and deployment
- Pre-built template marketplace
- GitHub automation and repository management
- Real-time execution streaming
- Storage management
- User authentication and payments
- Seraphina AI chat assistant

**Key Tool Categories**:
- **Swarm & Agents**: swarm_init, swarm_scale, agent_spawn, task_orchestrate
- **Sandboxes**: sandbox_create, sandbox_execute, sandbox_upload
- **Templates**: template_list, template_deploy
- **Neural AI**: neural_train, neural_patterns, seraphina_chat
- **GitHub**: github_repo_analyze, github_pr_manage
- **Real-time**: execution_stream_subscribe, realtime_subscribe
- **Storage**: storage_upload, storage_list

**Authentication**:
```bash
# Register new user
npx flow-nexus@latest register

# Login existing user
npx flow-nexus@latest login
```

### 3. ruv-swarm (Priority 3)
**Enhanced swarm coordination** with advanced consensus mechanisms.

**Command**: `npx ruv-swarm mcp start`

**Capabilities**:
- Advanced coordination protocols
- Byzantine fault tolerance
- Raft consensus management
- Gossip protocol coordination
- Collective intelligence systems
- Distributed swarm memory
- CRDT synchronization
- Quorum management

**Key Features**:
- Multi-consensus protocol support
- Fault-tolerant distributed systems
- Enhanced coordination patterns
- Collective intelligence aggregation

### 4. mcpproxy (Priority 4)
**Local HTTP streaming server** for upstream MCP server management.

**URL**: `http://localhost:8080/mcp`

**Capabilities**:
- Upstream MCP server aggregation
- HTTP streaming support
- Server health monitoring
- Local coordination hub
- Configuration management

**Integration**:
- Manages upstream MCP servers via HTTP API
- Provides streamable-http interface
- Aggregates multiple server connections
- Local development coordination

## Server Priority System

Servers are prioritized for operation orchestration:

1. **Priority 1** (claude-flow@alpha): Primary orchestration and coordination
2. **Priority 2** (flow-nexus): Cloud execution and advanced features
3. **Priority 3** (ruv-swarm): Enhanced coordination and consensus
4. **Priority 4** (mcpproxy): Local server management

## Hooks Integration

All MCP servers integrate with Claude Code hooks system defined in `.claude/settings.json`:

### Pre-Operation Hooks
- **PreToolUse/Bash**: Command validation and resource preparation
- **PreToolUse/Write|Edit|MultiEdit**: Auto-agent assignment and context loading
- **PreCompact**: Agent context guidance before compaction

### Post-Operation Hooks
- **PostToolUse/Bash**: Metrics tracking and result storage
- **PostToolUse/Write|Edit|MultiEdit**: Code formatting and memory updates

### Session Hooks
- **Stop**: Session summary, state persistence, and metrics export

## Environment Variables

Key environment variables from `.claude/settings.json`:

```bash
CLAUDE_FLOW_AUTO_COMMIT=false
CLAUDE_FLOW_AUTO_PUSH=false
CLAUDE_FLOW_HOOKS_ENABLED=true
CLAUDE_FLOW_TELEMETRY_ENABLED=true
CLAUDE_FLOW_REMOTE_EXECUTION=true
CLAUDE_FLOW_CHECKPOINTS_ENABLED=true
```

## Usage Patterns

### Basic Swarm Initialization
```bash
# Use claude-flow@alpha for primary coordination
npx claude-flow@alpha swarm init --topology mesh --max-agents 8

# Use flow-nexus for cloud-based execution
npx flow-nexus@latest swarm create --template quickstart

# Use ruv-swarm for advanced consensus
npx ruv-swarm mcp start
```

### Agent Spawning
```bash
# Claude Code Task tool (primary method)
Task("Coder agent", "Implement feature X", "coder")
Task("Tester agent", "Create tests", "tester")

# MCP coordination (setup only)
mcp__claude-flow__agent_spawn { type: "coder" }
```

### Memory Management
```bash
# Store coordination data
npx claude-flow@alpha hooks post-edit --file "src/app.js" --memory-key "swarm/coder/implementation"

# Restore session context
npx claude-flow@alpha hooks session-restore --session-id "swarm-123"
```

## Performance Optimization

### Token Efficiency
- claude-flow@alpha: 32.3% token reduction through neural patterns
- Batch operations in single messages (GOLDEN RULE)
- Reuse memory across sessions

### Speed Improvements
- Parallel agent execution: 2.8-4.4x faster
- Concurrent operations: Use Task tool for parallel spawning
- Cached patterns: Reduced redundant analysis

### Resource Management
- Priority-based server selection
- Automatic topology optimization
- Self-healing workflow recovery

## Troubleshooting

### Server Connection Issues
```bash
# Test server availability
npx claude-flow@alpha mcp test
npx flow-nexus@latest mcp test
npx ruv-swarm mcp test

# Check mcpproxy local server
curl http://localhost:8080/mcp
```

### Authentication Issues (flow-nexus)
```bash
# Re-authenticate
npx flow-nexus@latest login

# Check auth status
npx flow-nexus@latest auth status
```

### Hook Execution Issues
```bash
# Verify hooks are enabled
grep CLAUDE_FLOW_HOOKS_ENABLED .claude/settings.json

# Test hook execution
npx claude-flow@alpha hooks pre-command --command "test" --validate-safety true
```

## Migration Notes

### From Previous Configuration
1. `.mcp.json` now consolidates both stdio and http servers
2. `mcp.json` retained for backward compatibility
3. ruv-swarm added (previously referenced in settings.json but not configured)
4. Comprehensive metadata and capabilities documented

### Breaking Changes
- None - backward compatible with existing workflows
- All existing hooks and permissions maintained
- Server URLs and commands unchanged

## Best Practices

1. **Use Claude Code Task Tool**: Primary method for agent spawning
2. **Batch Operations**: Follow GOLDEN RULE (1 MESSAGE = ALL OPERATIONS)
3. **Priority Ordering**: Use servers according to priority hierarchy
4. **Memory Coordination**: Store decisions in memory for cross-agent communication
5. **Hook Integration**: Let hooks handle automation (formatting, metrics, memory)
6. **Error Handling**: Implement graceful degradation and fallback strategies

## Resources

- claude-flow@alpha: https://github.com/ruvnet/claude-flow
- flow-nexus: https://flow-nexus.ruv.io
- mcpproxy-go: https://github.com/ruvnet/mcpproxy-go (local repository)

## Version History

### v2.0.0 (2025-11-23)
- Consolidated .mcp.json and mcp.json configurations
- Added ruv-swarm server activation
- Enhanced server metadata with capabilities and priorities
- Comprehensive documentation
- Optimized for performance and clarity

### v1.0.0 (Previous)
- Initial claude-flow@alpha and flow-nexus configuration
- Basic mcpproxy local server setup
- Hooks integration
