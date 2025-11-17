# MCPProxy Documentation Page - Visual Example

## Page Layout

```
┌─────────────────────────────────────────────────────────────────┐
│  📚 API Documentation          [← Back to Dashboard]            │
│  (Purple gradient header)                                       │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                    MCP Server Tools Reference                    │
│  This page provides comprehensive documentation for all          │
│  available MCP server tools...                                   │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│  github-server            [Connected]  5 tools              ▼   │
├─────────────────────────────────────────────────────────────────┤
│  URL: https://api.github.com                                    │
│  Protocol: http                                                  │
│                                                                   │
│  ┌──────────────────┐  ┌──────────────────┐  ┌───────────────┐│
│  │ create_issue     │  │ list_repos       │  │ create_pr     ││
│  │ github:create... │  │ github:list...   │  │ github:cre... ││
│  │                  │  │                  │  │               ││
│  │ Create a new     │  │ List all repos   │  │ Create a pull ││
│  │ GitHub issue     │  │ for a user       │  │ request       ││
│  │                  │  │                  │  │               ││
│  │ Parameters:      │  │ No parameters    │  │ Parameters:   ││
│  │ ┌──────────────┐│  │ required         │  │ ┌───────────┐││
│  │ │Name │Type│Req││  │                  │  │ │Name│Type│R││
│  │ │title│str │Yes││  │                  │  │ │base│str │Y││
│  │ │body │str │No ││  │                  │  │ │head│str │Y││
│  │ │repo │str │Yes││  │                  │  │ │repo│str │Y││
│  │ └──────────────┘│  │                  │  │ └───────────┘││
│  └──────────────────┘  └──────────────────┘  └───────────────┘│
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│  weather-api          [Disconnected]  0 tools               ▼   │
├─────────────────────────────────────────────────────────────────┤
│  Server is not connected. Connect the server to view tools.     │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│  MCPProxy API Documentation | Generated on 2025-11-10 14:47    │
└─────────────────────────────────────────────────────────────────┘
```

## Color Scheme

- **Header Background**: Linear gradient (purple #667eea → #764ba2)
- **Header Text**: White
- **Connected Badge**: Green background (#d4edda), dark green text
- **Disconnected Badge**: Red background (#f8d7da), dark red text
- **Required Badge**: Yellow background (#ffc107), brown text
- **Optional Badge**: Light blue background (#e7f3ff), dark blue text
- **Tool Name Code**: Light blue background (#e7f3ff), blue text
- **Server Header Hover**: Light gray (#e9ecef)
- **Tool Card Hover**: Subtle shadow and 2px lift

## Interactive Elements

1. **Server Header Click**
   - Toggles server content visibility
   - Rotates toggle icon (▼ → ▲)
   - Smooth transition animation

2. **Back to Dashboard Link**
   - Hover: Lighter background
   - Navigates to `/` (dashboard)

3. **Tool Cards**
   - Hover: Shadow effect and slight upward movement
   - Clean border and rounded corners

## Responsive Behavior

- **Desktop**: Tools displayed in grid (400px minimum width per card)
- **Tablet**: 2 columns
- **Mobile**: Single column stack

## Content Sections

### 1. Navigation Header
- Page title with emoji icon
- Return link to dashboard

### 2. Introduction Box
- Light gray background with blue left border
- Brief description of documentation purpose

### 3. Server Sections (Collapsible)
Each server shows:
- Server name (large, bold)
- Status badge (colored)
- Tool count
- Toggle icon

When expanded:
- Server details panel (URL, Command, Protocol)
- Tools grid with cards

### 4. Tool Cards
Each tool card contains:
- Tool name (heading)
- Full server:tool name (code style)
- Description paragraph
- Parameters section with table

### 5. Parameter Tables
Table structure:
```
┌──────────┬──────────┬──────────┬─────────────────┐
│ Name     │ Type     │ Required │ Description     │
├──────────┼──────────┼──────────┼─────────────────┤
│ title    │ string   │ Required │ Issue title     │
│ body     │ string   │ Optional │ Issue body text │
│ labels   │ array    │ Optional │ Issue labels    │
└──────────┴──────────┴──────────┴─────────────────┘
```

### 6. Footer
- Centered text
- Generation timestamp
- MCPProxy branding

## Example Server States

### Connected Server with Tools
- Shows green "Connected" badge
- Displays tool count
- Renders tool cards with full information

### Connected Server without Tools
- Shows green "Connected" badge
- Shows "0 tools"
- Message: "No tools available from this server."

### Disconnected Server
- Shows red "Disconnected" badge
- Shows current state (e.g., "Error", "Connecting")
- Message: "Server is not connected. Connect the server to view available tools."

## Tool Parameter Types

Common parameter types displayed:
- `string` - Text values
- `number` - Numeric values
- `boolean` - True/false values
- `array` - List of items
- `object` - Complex structured data
- `any` - Any type (when not specified)

## Usage Flow

1. User clicks "View Documentation" on dashboard
2. Page loads with all servers listed
3. User clicks server header to expand
4. Server section shows tools in grid
5. User reviews tool names, descriptions, and parameters
6. User clicks "Back to Dashboard" to return

## Accessibility Features

- Semantic HTML structure
- Proper heading hierarchy (h1 → h2 → h3 → h4)
- Table headers for parameter tables
- Clear visual hierarchy
- Keyboard navigation support (click handlers)
- Adequate color contrast

## Performance Considerations

- 30-second timeout for tool retrieval
- Graceful handling of slow servers
- No blocking on disconnected servers
- Minimal JavaScript (only toggle functionality)
- CSS-based animations (smooth, hardware-accelerated)

## Example Tool Documentation

### Tool: create_issue
- **Full Name**: `github:create_issue`
- **Description**: Creates a new issue in a GitHub repository
- **Parameters**:
  | Name | Type | Required | Description |
  |------|------|----------|-------------|
  | title | string | Required | The title of the issue |
  | body | string | Optional | The body text of the issue |
  | repo | string | Required | Repository name (owner/repo) |
  | labels | array | Optional | Labels to apply to the issue |

### Tool: get_weather
- **Full Name**: `weather-api:get_weather`
- **Description**: Get current weather information for a city
- **Parameters**:
  | Name | Type | Required | Description |
  |------|------|----------|-------------|
  | city | string | Required | City name |
  | units | string | Optional | Temperature units (celsius/fahrenheit) |

## Browser Compatibility

- Modern browsers (Chrome, Firefox, Safari, Edge)
- CSS Grid support required
- ES6 JavaScript (const, arrow functions, template literals)
- HTML5 semantic elements

## Summary

The documentation page provides a professional, comprehensive view of all available MCP tools with clear parameter documentation, making it easy for developers to understand and integrate MCP servers into their applications.
