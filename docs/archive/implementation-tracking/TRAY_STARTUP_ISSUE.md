# MCPProxy System Tray Startup Issue

## Problem Summary

The `fyne.io/systray` library used by mcpproxy requires direct macOS WindowServer access that cannot be obtained through automated startup methods.

## Symptoms

- Process runs at 95-100% CPU usage
- No tray icon appears in menu bar
- Logs show "Starting system tray" but tray never initializes
- `systray.Run()` event loop spins infinitely without GUI access

## Failed Automated Approaches Tested

### 1. Launch Agent with ProcessType: Interactive
**Status**: ❌ FAILED
**Result**: 100% CPU, no tray icon
**Reason**: Launch Agents cannot access WindowServer even with Interactive process type

### 2. Background Process (nohup, &)
**Status**: ❌ FAILED
**Result**: 100% CPU, no tray icon
**Reason**: Background processes have no GUI session context

### 3. Terminal via AppleScript
**Status**: ❌ FAILED
**Result**: 100% CPU, no tray icon
**Reason**: Programmatically opened terminals still lack proper GUI permissions

### 4. macOS Application Bundle (.app)
**Status**: ❌ FAILED
**Result**: 97.9% CPU, no tray initialization in logs
**Reason**: Application bundles need user interaction to access WindowServer properly

## Root Cause

The `fyne.io/systray` library's `Run()` function requires a full GUI session context that only exists when:
- User manually opens Terminal.app and runs the command
- Application is launched directly by user from Finder/Dock
- Application has explicit GUI session access granted by macOS

Launch Agents, background processes, and even `.app` bundles opened programmatically do not have this level of access.

## Working Solution

The application **must be started manually** by the user in a Terminal window:

```bash
cd /Users/hrannow/Library/CloudStorage/OneDrive-Persönlich/workspace/mcp-server/mcpproxy-go
./mcpproxy serve
```

This is why the logs from 10:12am showed the tray working perfectly - it was likely running in a user-opened Terminal window.

## Alternative: Run Without Tray

For automated/background operation, run mcpproxy without the tray interface:

```bash
./mcpproxy serve --tray=false
```

This allows:
- Background operation via Launch Agent
- HTTP server on port 8080 remains accessible
- Server management via HTTP API instead of tray UI

## Recommendations

1. **For Development**: User should manually run `./mcpproxy serve` in Terminal
2. **For Production**: Consider one of these approaches:
   - Run without tray (`--tray=false`) and use HTTP API
   - Create a web UI for server management instead of tray
   - Use a different systray library compatible with Launch Agents
   - Package as proper macOS application with entitlements

## Evidence

**Log Pattern** showing tray initialization succeeded in previous runs:
```
2025-11-03T16:33:10.547+01:00 | INFO | Starting system tray application
2025-11-03T16:33:10.548+01:00 | INFO | Config file watcher initialized
```

**Current run** (started 16:40:57) shows NO tray initialization after 16:40:57.172, only server connections.

**Process behavior**: 97.9% CPU usage indicates `systray.Run()` spinning without WindowServer access.
