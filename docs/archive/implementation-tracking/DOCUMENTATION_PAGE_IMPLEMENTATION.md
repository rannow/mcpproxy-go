# MCPProxy Documentation Page Implementation

## Overview

A comprehensive API documentation page has been implemented for MCPProxy that displays all MCP servers with their tools, parameters, and detailed information.

## Implementation Details

### Files Modified/Created

1. **`internal/server/docs.go`** (NEW)
   - Main documentation handler implementation
   - Parses tool metadata from connected servers
   - Renders HTML documentation with server sections and tool cards
   - Includes parameter tables with type information and descriptions

2. **`internal/server/server.go`** (MODIFIED)
   - Added route handler: `mux.HandleFunc("/docs", s.handleDocs)` at line 1655
   - Registered new `/docs` endpoint in the HTTP server

3. **`internal/server/dashboard.go`** (MODIFIED)
   - Added "API Documentation" card to dashboard grid
   - Provides link to `/docs` page from main dashboard

## Features Implemented

### Documentation Page Features

1. **Navigation**
   - Header with title and "Back to Dashboard" link
   - Clean, professional layout matching existing dashboard design

2. **Server Sections**
   - Collapsible/expandable server sections (click to toggle)
   - Server status badges (Connected/Disconnected)
   - Tool count display for each server
   - Server details (URL, Command, Protocol)

3. **Tool Documentation**
   - Tool cards displayed in responsive grid layout
   - Tool name and full server:tool format
   - Tool descriptions
   - Parameter tables showing:
     - Parameter name
     - Type information
     - Required/Optional badges
     - Parameter descriptions

4. **Design**
   - Consistent color scheme with dashboard (purple gradient header)
   - Responsive grid layout for tool cards
   - Hover effects and transitions
   - Professional typography and spacing
   - Status badges with appropriate colors

5. **Behavior**
   - JavaScript toggle for server sections
   - Handles disconnected servers gracefully
   - Shows message when no tools available
   - 30-second timeout for tool retrieval

### Dashboard Integration

- New "API Documentation" card added at the top of the dashboard grid
- Uses 📚 emoji icon
- Description: "Browse comprehensive documentation for all MCP server tools and parameters"
- Button text: "View Documentation"

## Technical Architecture

### Data Flow

```
User Request → handleDocs()
    ↓
Get All Servers (upstreamManager.ListServers())
    ↓
For Each Server:
    - Get Client (upstreamManager.GetClient())
    - Check Connection Status
    - List Tools (client.ListTools()) if connected
    - Parse Tool Metadata (parseToolsForDocs())
    ↓
Render HTML
    - Server Sections (renderServerSection())
    - Tool Cards (renderToolCard())
    - Parameter Tables
    ↓
Send Response
```

### Key Functions

1. **`handleDocs()`**
   - Main handler for `/docs` endpoint
   - Retrieves all servers and their tools
   - Builds HTML response

2. **`parseToolsForDocs()`**
   - Parses `ToolMetadata` into documentation-friendly format
   - Extracts parameters from ParamsJSON
   - Identifies required vs optional parameters
   - Sorts parameters (required first, then alphabetically)

3. **`renderServerSection()`**
   - Renders a complete server section
   - Handles connected/disconnected states
   - Includes server details and tools grid

4. **`renderToolCard()`**
   - Renders individual tool cards
   - Creates parameter tables
   - Handles tools with no parameters

5. **`getDocsHeader()` / `getDocsFooter()`**
   - HTML template generation
   - Includes CSS styles and JavaScript

## CSS Styling

- Gradient background matching dashboard
- White container with rounded corners and shadow
- Purple gradient navigation header
- Color-coded status badges (green for connected, red for disconnected)
- Hover effects on server headers and tool cards
- Responsive grid layout for tools
- Professional parameter tables
- Required/Optional badges with appropriate colors

## JavaScript Functionality

```javascript
function toggleServer(serverName) {
    // Toggles collapsed state for server sections
    // Rotates toggle icon
}
```

## Usage

### Accessing Documentation

1. Start mcpproxy server: `./mcpproxy serve`
2. Navigate to: `http://localhost:8080/docs`
3. Or click "View Documentation" from dashboard

### Documentation Structure

```
/docs
  └─ Server 1 (expandable)
      ├─ Server Details (URL, Command, Protocol)
      └─ Tools Grid
          ├─ Tool 1 Card
          │   ├─ Tool Name
          │   ├─ Description
          │   └─ Parameter Table
          └─ Tool 2 Card
              └─ ...
```

## Error Handling

- Disconnected servers show appropriate status
- Servers with no tools show "No tools available" message
- Failed tool retrieval logs warning but continues
- 30-second timeout prevents hanging requests

## Future Enhancements

Potential improvements:
1. Search/filter functionality for tools
2. Export documentation to JSON/Markdown
3. Direct tool testing interface
4. Parameter examples/samples
5. Response schema documentation
6. Tool usage statistics

## Testing

Build verification:
```bash
cd /path/to/mcpproxy-go
go build -o mcpproxy ./cmd/mcpproxy
```

Manual testing:
1. Start server with multiple MCP servers configured
2. Navigate to `/docs`
3. Verify all servers appear
4. Click server headers to toggle sections
5. Verify tool information displays correctly
6. Check parameter tables for accuracy

## Notes

- Documentation is generated dynamically from connected servers
- Only connected servers show their tools
- Tool information is retrieved in real-time
- Parameter schemas are parsed from JSON stored in database
- Design is consistent with existing MCPProxy web interfaces

## Summary

The documentation page provides a comprehensive, user-friendly interface for developers to explore available MCP server tools, understand their parameters, and integrate them into their applications. The implementation follows MCPProxy's existing design patterns and integrates seamlessly with the current dashboard structure.
