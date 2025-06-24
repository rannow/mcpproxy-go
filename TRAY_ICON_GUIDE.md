# System Tray Icon Guide

This guide explains how to set up and use the enhanced system tray functionality for mcpproxy-go.

## Enhanced Tray Features

The system tray now provides comprehensive status information and control capabilities:

### Status Information
- **Dynamic Tooltip**: Shows current proxy status, connection URL, connected servers count, and total available tools
- **Real-time Updates**: Status updates every 5 seconds automatically
- **Server Status**: Displays which upstream servers are connected/disconnected

### Control Features
- **Start/Stop Server**: Toggle proxy server on/off from the tray menu
- **Upstream Server Monitoring**: View detailed status of all configured upstream servers
- **Server Statistics**: See tool counts per server and connection status

## Tray Menu Structure

```
Smart MCP Proxy
├── Status: Running (localhost:8080)
├── ─────────────────────────────
├── Start/Stop Server
├── ─────────────────────────────
├── Upstream Servers (2/3)
│   └── [Hover for server details]
├── ─────────────────────────────
├── Check for Updates…
├── Open Config
├── ─────────────────────────────
└── Quit
```

## Tooltip Information

The tray icon tooltip displays comprehensive status in multiple lines:
- **Server Status**: "mcpproxy (Ready) - http://localhost:3001" (phase and listen address)
- **Server Connections**: "Servers: 2/3 connected" (connected/total configured)
- **Tool Count**: "Tools: 15 available" (total tools across all connected servers)

When stopped: "mcpproxy is stopped"

## Server Control

### Start/Stop Server
- Click "Start Server" to start the proxy when stopped
- Click "Stop Server" to stop the proxy when running
- Status updates immediately after control actions

### Upstream Server Details
Hover over "Upstream Servers" to see detailed information:
```
• GitHub Tools: Connected (8 tools)
• Weather API: Disconnected (0 tools)
• File Manager: Connected (5 tools)
```

## Quick Fix for Missing Tray Icon

If you're not seeing the tray icon on macOS, it's because the icon wasn't embedded in the binary. This has been fixed in the latest version by:

1. **Added icon files** to `internal/tray/`:
   - `icon-32.png` - Color version (32x32 pixels)
   - `icon-mono-44.png` - Monochrome version (44x44 pixels, works better on macOS)

2. **Updated tray code** to embed and use the icon:
   ```go
   //go:embed icon-mono-44.png
   var iconData []byte
   
   func (a *App) onReady() {
       systray.SetIcon(iconData)
       // On macOS, also try setting as template icon for better integration
       if runtime.GOOS == "darwin" {
           systray.SetTemplateIcon(iconData, iconData)
       }
   }
   ```

## Testing the Enhanced Features

1. **Rebuild the application**:
   ```bash
   go build -ldflags "-X main.version=v0.3.0-enhanced" ./cmd/mcpproxy
   ```

2. **Run with tray enabled**:
   ```bash
   ./mcpproxy --tray=true --listen=:8080
   ```

3. **Test the features**:
   - Look for the tray icon in your system tray
   - Hover over the icon to see the enhanced tooltip
   - Right-click to access the menu
   - Try starting/stopping the server
   - Check upstream server status

## Configuration for Enhanced Features

### Server Configuration
Ensure your configuration includes upstream servers and listening address:

```json
{
  "listen": ":8080",
  "enable_tray": true,
  "mcpServers": [
    {
      "name": "GitHub Tools",
      "url": "http://localhost:3001",
      "type": "http",
      "enabled": true
    },
    {
      "name": "Weather API",
      "command": "weather-mcp-server",
      "type": "stdio",
      "enabled": true
    }
  ]
}
```

### Environment Variables
You can also control the tray behavior:
```bash
export MCPPROXY_TRAY=true
export MCPPROXY_LISTEN=:8080
./mcpproxy
```

## Icon Design Guidelines

### OS Specific Requirements

- **macOS**: 44x44 pixels, transparent PNG. For dynamic adaptation to light/dark system themes, the icon should be black with only an alpha channel.
- **Windows**: 32x32 pixels.
- **Linux**: 24x24 pixels. **Note: System tray functionality is currently disabled on Linux** due to compatibility issues with the systray library. Use `--tray=false` or run without tray on Linux.
- **Format**: PNG with transparency is generally recommended.
- **Color**: Monochrome icons often work better and adapt to system theme on macOS.
- **Style**: Simple, high contrast designs work best.
- **Template**: Consider using template images that adapt to dark/light modes.

### Platform Limitations

#### Linux Support
The system tray functionality is currently **disabled on Linux builds** due to incomplete support in the underlying `fyne.io/systray` library. On Linux, the application will:

- Automatically use the stub implementation (no tray)  
- Run normally in server-only mode
- Log: "Tray functionality disabled (nogui/headless build)"

**Workarounds for Linux users:**
```bash
# Run without tray (recommended)
./mcpproxy --tray=false

# Or use environment variable
export MCPPROXY_TRAY=false
./mcpproxy
```

**Alternative Linux tray solutions:**
- Use a different systray library (requires code changes)
- Run the server and use a separate GUI client
- Use command-line interface for management

### Current Icons

The project includes two icon versions:

1. **Color Icon** (`icon-32.png`):
   - Blue theme with connected nodes
   - Represents the proxy concept visually
   - Good for branded applications

2. **Monochrome Icon** (`icon-mono-44.png`):
   - Black and white design
   - Better system integration on macOS
   - Adapts to system theme changes

## Customizing the Icon

### Option 1: Replace Existing Icon

1. **Create your icon** (44x44 PNG recommended for macOS):
   ```bash
   # Using your preferred graphics editor, create a 44x44 PNG
   # Save it as icon-mono-44.png
   ```

2. **Replace the embedded icon**:
   ```bash
   cp your-custom-icon.png internal/tray/icon-mono-44.png
   ```

3. **Rebuild**:
   ```bash
   go build ./cmd/mcpproxy
   ```

### Option 2: Use Different Icon

1. **Add new icon** to `internal/tray/`:
   ```bash
   cp your-icon.png internal/tray/my-custom-icon.png
   ```

2. **Update the embed directive** in `internal/tray/tray.go`:
   ```go
   //go:embed my-custom-icon.png
   var iconData []byte
   ```

3. **Rebuild** the application

### Option 3: Use Color Icon

To use the color version instead:

1. **Update the embed directive** in `internal/tray/tray.go`:
   ```go
   //go:embed icon-32.png
   var iconData []byte
   ```

2. **Rebuild** the application

## Creating Custom Icons

### Using Inkscape (Free)

1. **Install Inkscape**:
   ```bash
   brew install inkscape
   ```

2. **Create SVG** with 64x64 viewBox:
   ```xml
   <?xml version="1.0" encoding="UTF-8"?>
   <svg width="64" height="64" viewBox="0 0 64 64" xmlns="http://www.w3.org/2000/svg">
     <!-- Your icon design here -->
   </svg>
   ```

3. **Export to PNG**:
   ```bash
   inkscape --export-filename=icon-mono-44.png --export-width=44 --export-height=44 your-icon.svg
   ```

### Using ImageMagick

1. **Install ImageMagick**:
   ```bash
   brew install imagemagick
   ```

2. **Convert existing image**:
   ```bash
   convert your-image.png -resize 44x44 icon-mono-44.png
   ```

3. **Create simple shape**:
   ```bash
   convert -size 44x44 xc:transparent -fill black -draw "circle 22,22 22,11" icon-mono-44.png
   ```

## Advanced Configuration

### Template Icons (macOS)

For better macOS integration, the tray automatically uses template icons on macOS:

```go
func (a *App) onReady() {
    systray.SetIcon(iconData)
    if runtime.GOOS == "darwin" {
        systray.SetTemplateIcon(iconData, iconData) // macOS template icon
    }
}
```

### Status Update Frequency

The tray updates status every 5 seconds by default. You can modify this in the code:

```go
// Start background status updater (every 5 seconds)
go func() {
    ticker := time.NewTicker(5 * time.Second) // Change this duration
    // ...
}()
```

## Troubleshooting

### Icon Not Showing

1. **Check if the icon file exists**:
   ```bash
   ls -la internal/tray/*.png
   ```

2. **Verify embed is working**:
   ```bash
   go build -v ./cmd/mcpproxy 2>&1 | grep embed
   ```

3. **Check system tray visibility** (macOS):
   - System Preferences → Dock & Menu Bar → Other Icons
   - Make sure "Show in Menu Bar" is enabled for relevant items

### Status Not Updating

1. **Check server configuration**:
   ```bash
   ./mcpproxy --log-level=debug --tray=true
   ```

2. **Verify upstream connections**:
   - Look for connection errors in logs
   - Check if upstream servers are running
   - Verify network connectivity

### Menu Not Responding

1. **Restart the application**:
   ```bash
   pkill mcpproxy
   ./mcpproxy --tray=true
   ```

2. **Check for conflicting tray applications**
3. **Verify system permissions** for UI access

## Technical Implementation Details

### ServerInterface
The tray communicates with the server through a defined interface:

```go
type ServerInterface interface {
    IsRunning() bool
    GetListenAddress() string
    GetUpstreamStats() map[string]interface{}
    StartServer(ctx context.Context) error
    StopServer() error
}
```

### Status Updates
- **Background Process**: Updates run every 5 seconds
- **Thread Safety**: Uses proper locking for concurrent access
- **Error Handling**: Gracefully handles server connection issues

### Performance Considerations
- **Efficient Polling**: Only queries necessary information
- **Cached Results**: Avoids repeated expensive operations
- **Memory Management**: Proper cleanup of resources

## Future Enhancements

Planned improvements include:
- **Dynamic Submenus**: Individual server control from tray
- **Notification System**: Pop-up alerts for server status changes
- **Statistics Graphs**: Mini-graphs in tooltip or submenu
- **Quick Actions**: Direct tool execution from tray menu 