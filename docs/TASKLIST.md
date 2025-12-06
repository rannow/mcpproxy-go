# MCPProxy Task List

## Server Management Tasks

### 1. Stop All Servers Menu
- [ ] Check "Stop All Servers" menu functionality
- [ ] Verify it kills all processes correctly
- [ ] Test graceful shutdown first, then force kill after timeout

### 2. Tool Connectivity Testing
- [ ] Check all connected servers can connect tools
- [ ] Verify tools work correctly
- [ ] Create automated test suite for tool validation

### 3. Diagnostic Tool
- [ ] Test Diagnostic Tool for fixing server startup problems
- [ ] Verify it can run over all servers that don't start correctly
- [ ] Generate detailed error reports

### 4. Lazy Loading Startup
- [ ] Check the function to start servers with lazy loading
- [ ] Verify deferred connection behavior

### 5. Config Reload
- [ ] Verify config reload detects changes
- [ ] Check that changed servers get restarted
- [ ] Test hot-reload without full restart

### 6. Disabled Server Handling
- [ ] Check disabled server state management
- [ ] Verify disabled servers don't consume resources

---

## Tray Changes

### 1. Menu Sorting
- [ ] Sort server menus by name alphabetically

---

## Startup Process Analysis (Claude Flow)

### Requirements:
- [ ] Analyze server startup in batches
- [ ] Auto-disable after 5 connection failures (currently 7)
- [ ] Create detailed error reports with:
  - Missing packages
  - Timeouts
  - Specific errors
- [ ] Check DB storage of failure info
- [ ] Verify no resource leaks from failed servers
- [ ] Document startup process

### Shutdown Process:
- [ ] Graceful shutdown for all servers
- [ ] Force kill after timeout if not responding
- [ ] Apply to: single server restart, stop all, quit application

---

## New Feature: Encrypted Keystore

### Web Page for Secrets
- [ ] Create web page to save secrets in encrypted keystore
- [ ] Implement encryption/decryption
- [ ] Secure key management

### Config Loader Integration
- [ ] Extend config loader to support keystore secrets
- [ ] Replace environment variable references with keystore lookups
- [ ] Update README.md with keystore documentation

---

## Code Refactoring Analysis (Claude Flow)

### State Management
- [ ] Analyze all state types for consistency
- [ ] Review Auto-Disabled state handling
- [ ] Check Start/Stop process state transitions
- [ ] Ensure killed servers are cleaned up
- [ ] Verify graceful vs force shutdown logic

### Code Quality
- [ ] Find old and inconsistent code changes
- [ ] Identify technical debt
- [ ] Create refactoring recommendations

---

## MCP Server Problem Analyzer Agent

### Agent Capabilities:
1. **Detailed Analysis**
   - Read `mcp_config.json` for server configurations
   - Analyze `failed_servers.log` in `~/.mcpproxy/`
   - Parse logs in `~/Library/Logs/mcpproxy/`

2. **Problem Detection**
   - Missing environment variables
   - Package/dependency issues
   - Timeout problems
   - Connection errors
   - Configuration errors

3. **Output**
   - Generate detailed MD report
   - List of questions for missing data
   - Troubleshooting hints
   - Links to server repositories

4. **Testing**
   - Use https://github.com/wong2/mcp-cli for direct testing

---

## Priority Order

1. **Critical**: Stop All Servers, Graceful Shutdown
2. **High**: Startup Process Analysis, Auto-Disable Logic
3. **Medium**: Diagnostic Tool, Config Reload
4. **Normal**: Tray Sorting, Keystore Feature
5. **Research**: Refactoring Analysis, Problem Analyzer Agent
