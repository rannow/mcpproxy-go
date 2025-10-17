# MCP Server Connection Issues

## Problems Found:

### 1. Missing/Incorrect Package Names
- ❌ `mcp-server-todoist` → Should be `mcp-todoist` 
- ❌ `mcp-server-twitter` → Package doesn't exist
- ❌ `mcp-server-notion` → Package doesn't exist

### 2. Connection Timeouts
- Servers failing with "context deadline exceeded"
- Usually caused by missing packages or incorrect commands

### 3. Server Count Mismatch
- Config has 154 servers but only 107-108 are successfully connecting
- 47 servers are failing to connect

## Fixes Needed:

### Fix 1: Update todoist server configuration
```json
{
  "name": "mcp-todoist",
  "command": "npx",
  "args": ["-y", "mcp-todoist"]
}
```

### Fix 2: Remove non-existent servers
- Remove `mcp-server-twitter` (package doesn't exist)
- Remove `mcp-server-notion` (package doesn't exist)
- Or find correct package names

### Fix 3: Check other failing servers
Need to verify all npx-based servers have correct package names.

## Recommended Actions:
1. Fix known incorrect package names
2. Remove servers with non-existent packages
3. Add validation to prevent adding servers with invalid packages
4. Implement better error reporting for missing dependencies
