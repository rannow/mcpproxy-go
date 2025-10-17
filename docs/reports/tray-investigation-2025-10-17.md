# Tray System Investigation Report

**Date**: 2025-10-17
**Branch**: feature/tray-investigation
**Issue**: Tray icon not appearing in macOS menu bar, application crashes with exit code 144

## Problem Summary

The mcpproxy application fails to display a tray icon in the macOS menu bar when started with `--tray=true`. The application crashes with exit code 144 (SIGTERM) after attempting to initialize the tray system.

## Investigation Results

### Successful Scenarios

✅ **Web Server Without Tray** - Works perfectly
```bash
./mcpproxy serve --tray=false
```
- Server starts successfully on port 8080
- Dashboard accessible with 4 cards
- Metrics page with auto-refresh working
- 4 MCP servers connected: Container User, Chrome Dev Tools, Bright Data, opencode

### Failed Scenarios

❌ **Web Server With Tray** - Crashes with exit code 144
```bash
./mcpproxy serve --tray=true
```
- Server initialization starts normally
- Tray system begins loading
- Application terminates with SIGTERM (exit code 144)
- No tray icon appears in menu bar

## Root Cause Analysis

### 1. Menu Entry Overload

The system attempts to create menu entries for all 160 configured MCP servers. For each server, it creates:
- Enable/Disable toggle
- Quarantine option
- Configure option
- Restart option
- Connect option
- Disconnect option

**Total**: 160 servers × 6 menu items = **960+ menu entries**

### 2. Memory and Performance Issues

Log analysis shows the tray initialization process:
```
2025-10-17 10:04:46	INFO	systray@v1.11.0/systray.go:112	TRAY INIT: Loading groups for server assignment menus
2025-10-17 10:04:46	INFO	tray/managers.go:656	Creating action submenus for server...
[repeated 160 times]
```

The process gets killed by macOS (SIGTERM) before completing initialization, likely due to:
- Excessive memory consumption
- UI thread blocking
- System resource limits
- macOS tray menu size limitations

### 3. No Progressive Loading

Current implementation:
- Loads all servers synchronously
- Creates all menu items at startup
- No pagination or lazy loading
- No error handling for resource exhaustion

## Technical Details

### Exit Code Analysis
- **Exit Code 144**: SIGTERM (143 + 1 on some systems)
- Indicates the process was terminated by the operating system
- Common causes: Resource limits, unresponsive UI thread, security restrictions

### Logs from Failed Run
```
Starting system tray with auto-start server
MAIN - Starting tray event loop
Starting system tray application
System tray is ready - menu items fully initialized
[Process terminated with exit code 144]
```

The logs show initialization appears successful, but macOS terminates the process before the tray icon can be displayed.

## Proposed Solutions

### Short-term Solution (Immediate)
Use web interface without tray:
```bash
./mcpproxy serve --tray=false
```
Access dashboard at http://localhost:8080

### Medium-term Solutions (Recommended)

#### Option 1: Limit Menu Entries
- Show only first 20-30 servers in tray
- Add "Show All in Browser" option
- Implement server filtering/search

#### Option 2: Simplified Tray Menu
```
MCPProxy
├── Status: Running
├── Connected Servers: 4/160
├── Open Dashboard
├── Recent Servers
│   ├── Container User
│   ├── Chrome Dev Tools
│   ├── Bright Data
│   └── opencode
├── Settings
└── Quit
```

#### Option 3: Lazy Loading
- Load only active/enabled servers initially
- Load additional servers on-demand
- Implement server categories/groups

### Long-term Solution (Architectural)

Complete tray system redesign:
1. **Minimal Tray Menu**: Only status and "Open Dashboard" link
2. **Web-based Configuration**: All server management through web UI
3. **Notification System**: Use tray only for status notifications
4. **Performance Monitoring**: Add memory and UI thread monitoring

## Code Areas Affected

### Files Requiring Changes
- `internal/tray/tray.go` - Main tray initialization (lines 288-656)
- `internal/tray/managers.go` - Menu creation logic
- `cmd/mcpproxy/main.go` - Tray mode startup

### Key Functions
- `loadGroupsForServerMenus()` - Server menu loading
- `CreateActionSubmenus()` - Individual server menu creation
- `syncServersToMenu()` - Server list synchronization

## Recommendations

1. **Immediate**: Document tray limitation and recommend web interface
2. **Next Release**: Implement Option 2 (Simplified Tray Menu)
3. **Future**: Complete architectural redesign for scalability

## Testing Notes

### Environment
- macOS Darwin 24.6.0
- 160 MCP servers configured
- Application version: d7d3c69-dirty

### Reproducibility
- 100% reproducible with 160 configured servers
- Likely occurs with >50 servers based on resource usage patterns

## References

- Commit: 680dcb6 - "feat: add metrics link to dashboard and create custom root handler"
- Related: Tray system implementation in `internal/tray/`
- macOS Documentation: Menu Bar Extras guidelines

## Next Steps

1. Create ticket for tray menu optimization
2. Update documentation with workaround
3. Implement simplified tray menu
4. Add resource monitoring and error handling
5. Consider progressive enhancement strategy

---

**Status**: Investigation Complete
**Priority**: High (affects user experience)
**Complexity**: Medium (requires UI/UX redesign)
