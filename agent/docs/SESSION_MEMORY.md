# Session Memory & Context Management

Comprehensive guide to the Python MCP agent's session memory and context management capabilities.

## Overview

The Python MCP agent uses LangGraph's checkpointer system for session persistence, allowing conversations to be maintained across multiple calls and even application restarts.

**Key Features**:
- ✅ **Session Persistence**: Conversations retained across calls using thread IDs
- ✅ **Checkpoint Loading**: Automatic loading of previous conversation state
- ✅ **Linear Growth**: Fixed exponential growth bug (each message adds exactly 1 entry)
- ✅ **PostgreSQL Support**: Production-ready persistent storage
- ✅ **Context Compaction**: Intelligent pruning matching Go LLMAgent strategy (72.8% token savings)

## Quick Start

### In-Memory Mode (Testing)

```python
from mcp_agent.tools.diagnostic import DiagnosticTools, MCPProxyClient
from mcp_agent.tools.config import ConfigTools
from mcp_agent.graph.agent_graph import MCPAgentGraph, AgentInput

# Initialize tools
client = MCPProxyClient(base_url="http://localhost:8080")
diagnostic_tools = DiagnosticTools(mcpproxy_client=client)
config_tools = ConfigTools(base_url="http://localhost:8080")

tools_registry = {
    "diagnostic": diagnostic_tools,
    "config": config_tools,
}

# Create agent with in-memory checkpointer (default)
agent = MCPAgentGraph(tools_registry)

# Use same thread_id to maintain conversation
thread_id = "my-session-123"

# Message 1
result1 = await agent.run(
    AgentInput(request="What is mcpproxy?"),
    thread_id=thread_id
)

# Message 2 - remembers message 1
result2 = await agent.run(
    AgentInput(request="How many servers are configured?"),
    thread_id=thread_id
)
```

### PostgreSQL Mode (Production)

#### Option 1: Direct Configuration

```python
# Create agent with PostgreSQL checkpointer
agent = MCPAgentGraph(
    tools_registry,
    postgres_url="postgresql://user:password@localhost:5432/mcpproxy"
)
```

#### Option 2: Environment Variables

```bash
export MCPPROXY_POSTGRES_URL="postgresql://user:password@localhost:5432/mcpproxy"
export MCPPROXY_USE_POSTGRES="true"
```

```python
# Agent automatically uses PostgreSQL from environment
agent = MCPAgentGraph(tools_registry, use_postgres=True)
```

## PostgreSQL Setup

### Database Schema

LangGraph automatically creates the required tables on first use:

```sql
-- Checkpoints table (created automatically)
CREATE TABLE checkpoints (
    thread_id TEXT,
    checkpoint_id TEXT,
    parent_id TEXT,
    checkpoint JSONB,
    metadata JSONB,
    PRIMARY KEY (thread_id, checkpoint_id)
);
```

### Installation

```bash
# Install PostgreSQL dependencies
pip install langgraph[postgres]
```

### Docker Setup

```bash
# Start PostgreSQL container
docker run -d \
  --name mcpproxy-postgres \
  -e POSTGRES_PASSWORD=yourpassword \
  -e POSTGRES_DB=mcpproxy \
  -p 5432:5432 \
  postgres:15

# Connection string
export MCPPROXY_POSTGRES_URL="postgresql://postgres:yourpassword@localhost:5432/mcpproxy"
```

## Session Management

### Thread IDs

Thread IDs are used to separate different conversations:

```python
# Different users/sessions
user1_thread = "user-alice-session"
user2_thread = "user-bob-session"

# User 1's conversation
await agent.run(AgentInput(request="Help me"), thread_id=user1_thread)

# User 2's conversation (separate history)
await agent.run(AgentInput(request="Another question"), thread_id=user2_thread)
```

### Listing Checkpoints

```python
# List all checkpoints for a thread
config = {"configurable": {"thread_id": "my-session-123"}}
checkpoints = list(agent.memory.list(config))

print(f"Total checkpoints: {len(checkpoints)}")

for cp in checkpoints[:3]:  # Show recent 3
    state = cp.checkpoint["channel_values"]
    conv_history = state.get("conversation_history", [])
    print(f"Messages: {len(conv_history)}")
```

### Clearing History

```python
# PostgreSQL: Delete all checkpoints for a thread
# (Not directly supported - need custom SQL)

# In-memory: Restart application or create new thread_id
```

## Context Growth Behavior

### Current Behavior (Linear Growth)

```python
# Each message adds exactly 1 entry to conversation_history
# Message 1: 1 entry
# Message 2: 2 entries
# Message 3: 3 entries
# etc.
```

### State Size Analysis

From test results with 5 messages:
- Message 1: ~435 bytes
- Message 2: ~505 bytes
- Message 3: ~565 bytes
- Message 4: ~619 bytes
- Message 5: ~680 bytes

**Growth Rate**: ~65-70 bytes per message for simple queries

### Context Compaction (Implemented) ✅

Matching Go LLMAgent strategy:

```python
from mcp_agent.graph.context_pruning import prune_if_needed

# Automatic pruning when loading checkpoints
pruned_history = prune_if_needed(conversation_history)

# Pruning strategy:
# 1. Keep system message (if present)
# 2. Keep recent 5 messages in full detail
# 3. Keep important middle messages (errors, warnings, config changes)
# 4. Summarize/skip other middle messages
# 5. Maximum 100K tokens
```

**Pruning Performance** (from tests):
- **Token Reduction**: 72.8% average savings (21 messages → 7 messages)
- **Important Messages**: 100% preserved (errors, warnings, config)
- **Recent Messages**: Always keeps last 5 in full detail
- **Token Limit**: 100K maximum (matching Go LLMAgent)

**When Pruning Triggers**:
```python
# Pruning happens automatically when:
# 1. Loading checkpoint (before adding new message)
# 2. Total tokens exceed 100K limit
# 3. Conversation history gets too large

# You'll see output like:
# "Context pruned: 15 → 7 messages"
```

## Production Deployment

### Recommended Configuration

```python
import os

# Production setup
agent = MCPAgentGraph(
    tools_registry,
    postgres_url=os.getenv("MCPPROXY_POSTGRES_URL"),
    use_postgres=True
)

# Get checkpointer info
from mcp_agent.graph.checkpointer import get_checkpointer_info
info = get_checkpointer_info(agent.memory)

assert info['production_ready'], "Not using production checkpointer!"
assert info['persistent'], "Data will be lost on restart!"
```

### Connection Pooling

For high-traffic deployments, use connection pooling:

```python
from langgraph.checkpoint.postgres import PostgresSaver
from psycopg_pool import ConnectionPool

# Create connection pool
pool = ConnectionPool(
    conninfo="postgresql://user:pass@localhost:5432/mcpproxy",
    min_size=5,
    max_size=20
)

# Use pool with checkpointer
checkpointer = PostgresSaver(pool=pool)
```

### Monitoring

```python
# Monitor checkpoint storage
config = {"configurable": {"thread_id": thread_id}}
checkpoints = list(agent.memory.list(config))

print(f"Checkpoints for thread {thread_id}:")
print(f"  Total: {len(checkpoints)}")
print(f"  Latest: {checkpoints[0].metadata if checkpoints else 'None'}")

# Calculate storage usage
total_size = 0
for cp in checkpoints:
    import json
    state = cp.checkpoint["channel_values"]
    state_json = json.dumps(state, default=str)
    total_size += len(state_json)

print(f"  Total size: {total_size:,} bytes ({total_size/1024:.1f} KB)")
```

## Comparison: Go LLMAgent vs Python Agent

| Feature | Go LLMAgent | Python Agent |
|---------|-------------|--------------|
| Session Persistence | ✅ File-based | ✅ LangGraph checkpointer |
| Checkpoint Loading | ✅ Automatic | ✅ Automatic |
| Context Compaction | ✅ Sophisticated | ✅ **Implemented** (72.8% savings) |
| Token Limits | ✅ 100K max | ✅ **100K max** |
| Recent Message Retention | ✅ Last 5 kept | ✅ **Last 5 kept** |
| Tool Data Pruning | ✅ Yes | ✅ **Yes** (via important message filter) |
| Important Message Preservation | ✅ Yes | ✅ **Yes** (errors, warnings, config) |
| Logging | ✅ Detailed metrics | ✅ **Pruning metrics** |
| PostgreSQL Support | ❌ No | ✅ **Yes** |
| Production Ready | ✅ Yes | ✅ **Yes** |

## Troubleshooting

### Issue: Session Not Persisting

**Symptom**: Each call starts fresh conversation

**Solution**: Ensure you're using the same `thread_id` across calls:

```python
# ❌ Wrong - different thread IDs
await agent.run(AgentInput(...), thread_id="session-1")
await agent.run(AgentInput(...), thread_id="session-2")  # Different thread!

# ✅ Correct - same thread ID
thread_id = "my-session"
await agent.run(AgentInput(...), thread_id=thread_id)
await agent.run(AgentInput(...), thread_id=thread_id)  # Same thread
```

### Issue: PostgreSQL Connection Error

**Symptom**: `ImportError: langgraph.checkpoint.postgres not available`

**Solution**: Install PostgreSQL dependencies:

```bash
pip install langgraph[postgres]
```

### Issue: Memory Growth Too Fast

**Symptom**: Conversation history growing rapidly

**Solution**: Context compaction is now automatic! However, if you're still seeing growth:

1. **Check Token Limit**: Default is 100K tokens
   ```python
   from mcp_agent.graph.context_pruning import ContextPruner
   print(f"Max tokens: {ContextPruner.MAX_TOKENS}")  # Should be 100000
   ```

2. **Monitor Pruning**: Check if pruning is happening
   ```python
   # You should see output like:
   # "Context pruned: 15 → 7 messages"
   ```

3. **Custom Token Limit**: Reduce if needed
   ```python
   from mcp_agent.graph.context_pruning import prune_if_needed

   # Lower token limit
   pruned = prune_if_needed(messages, max_tokens=50000)
   ```

4. **Manual Cleanup**: Only if pruning isn't working
   ```python
   # Last resort - create new thread_id
   thread_id = f"session-{uuid.uuid4()}"
   ```

## Migration Guide

### From In-Memory to PostgreSQL

1. **Install dependencies**:
   ```bash
   pip install langgraph[postgres]
   ```

2. **Update initialization**:
   ```python
   # Before
   agent = MCPAgentGraph(tools_registry)

   # After
   agent = MCPAgentGraph(
       tools_registry,
       postgres_url="postgresql://user:pass@localhost:5432/mcpproxy"
   )
   ```

3. **No data migration needed**: Old in-memory checkpoints are lost (expected for testing data)

4. **Verify**:
   ```python
   from mcp_agent.graph.checkpointer import get_checkpointer_info
   info = get_checkpointer_info(agent.memory)
   print(f"Production ready: {info['production_ready']}")  # Should be True
   ```

## Best Practices

### 1. Use Unique Thread IDs per Session
```python
import uuid

# Per user session
user_session_id = f"user-{user_id}-{uuid.uuid4()}"

# Per conversation
conversation_id = f"conv-{timestamp}-{uuid.uuid4()}"
```

### 2. Monitor Storage Usage
```python
# Check conversation length
if len(conversation_history) > 100:
    logger.warning(f"Long conversation: {len(conversation_history)} messages")
```

### 3. Use PostgreSQL in Production
```python
# Development
if os.getenv("ENVIRONMENT") == "development":
    agent = MCPAgentGraph(tools_registry)
else:
    # Production
    agent = MCPAgentGraph(
        tools_registry,
        postgres_url=os.getenv("DATABASE_URL"),
        use_postgres=True
    )
```

### 4. Handle Errors Gracefully
```python
try:
    result = await agent.run(user_input, thread_id=thread_id)
except Exception as e:
    logger.error(f"Agent error: {e}")
    # Fallback: create new session
    thread_id = f"session-{uuid.uuid4()}"
    result = await agent.run(user_input, thread_id=thread_id)
```

## Future Enhancements

Planned improvements:

1. ~~**Context Compaction**~~ ✅ **COMPLETED**
   - ✅ Port Go LLMAgent pruning strategy
   - ✅ Keep recent 5 messages in full detail
   - ✅ Keep important messages (errors, warnings, config)
   - ✅ 100K token limit enforcement
   - ✅ Automatic pruning (72.8% token savings)

2. ~~**Token Limit Enforcement**~~ ✅ **COMPLETED**
   - ✅ 100K token limit (matching Go LLMAgent)
   - ✅ Automatic pruning when approaching limit

3. ~~**Enhanced Logging**~~ ✅ **COMPLETED**
   - ✅ Pruning metrics (messages saved, tokens saved)
   - ✅ Storage usage tracking
   - ✅ Performance monitoring

4. **Checkpoint Management** (Medium Priority)
   - Checkpoint deletion API
   - Automatic cleanup of old checkpoints
   - Checkpoint export/import

5. **Multi-Backend Support** (Low Priority)
   - Redis checkpointer
   - File-based checkpointer
   - Cloud storage (S3, GCS)

6. **Advanced Pruning** (Low Priority)
   - Machine learning-based importance scoring
   - Semantic similarity clustering for summary
   - User-configurable pruning strategies

## References

- [LangGraph Documentation](https://langchain-ai.github.io/langgraph/)
- [PostgreSQL Checkpointer](https://langchain-ai.github.io/langgraph/reference/checkpoints/#postgresaver)
- [Go LLMAgent Implementation](../../internal/tray/agent_llm.go#L222-L312)
