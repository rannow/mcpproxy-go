# Agent Memory - Persistent Knowledge Base

## Overview

The Agent Memory system provides persistent, cross-session memory for the MCP Agent. This allows the agent to accumulate knowledge, patterns, and learnings that persist across different sessions and operations.

**Key Features:**
- ✅ Automatic loading on agent startup
- ✅ Structured markdown format with predefined sections
- ✅ Thread-safe concurrent access
- ✅ Automatic persistence with auto-save
- ✅ Section-based organization
- ✅ Timestamped entries
- ✅ Export capabilities

## Architecture

### Core Components

#### `AgentMemory` (`internal/tray/agent_memory.go`)
Manages the persistent memory file and provides structured access:
```go
type AgentMemory struct {
    FilePath        string
    LastModified    time.Time
    Content         string
    Sections        map[string]string
    mu              sync.RWMutex
    logger          *zap.Logger
    autoSave        bool
}
```

#### Integration with `AgentClient`
Memory is automatically initialized when using `NewAgentClientWithMemory`:
```go
client := tray.NewAgentClientWithMemory(
    "http://localhost:5000",
    "/path/to/data/dir",  // Memory file stored here
    logger,
)
```

### Memory File Structure

The memory is stored as a markdown file with predefined sections:

```markdown
# Agent Memory - Persistent Knowledge Base

**Last Updated**: 2025-11-01T10:30:00Z

## Server Configurations
### Known Issues
<!-- Server configuration problems and solutions -->

### Successful Patterns
<!-- Working server setups -->

## Diagnostic Patterns
### Common Problems
<!-- Frequently encountered issues -->

### Solution Strategies
<!-- Effective diagnostic approaches -->

## Tool Discoveries
### High-Value Tools
<!-- Useful tools and use cases -->

### Tool Combinations
<!-- Effective tool patterns -->

## Performance Insights
### Bottlenecks
<!-- Performance issues and fixes -->

### Best Practices
<!-- Performance optimization patterns -->

## Security Findings
### Vulnerabilities
<!-- Security issues and remediations -->

### Security Patterns
<!-- Effective security configurations -->

## Agent Learnings
### Successful Workflows
<!-- Workflow patterns that work well -->

### Failed Approaches
<!-- Approaches to avoid -->

### Optimization Opportunities
<!-- Potential improvements -->

## Custom Notes
<!-- Free-form notes and observations -->
```

## Storage Location

- **Default Path**: `~/.mcpproxy/agent-memory.md`
- **Custom Path**: Specified via `dataDir` parameter
- **Auto-creation**: Created automatically if it doesn't exist

## Usage Examples

### Example 1: Initialize Agent with Memory

```go
import (
    "mcpproxy-go/internal/tray"
    "go.uber.org/zap"
)

// Create agent client with memory
logger, _ := zap.NewDevelopment()
agentClient := tray.NewAgentClientWithMemory(
    "http://localhost:5000",
    "/Users/username/.mcpproxy",
    logger,
)

// Memory is now loaded and ready to use
fmt.Println("Memory initialized:", agentClient.HasMemory())
```

### Example 2: Read Memory Content

```go
// Read full memory
fullMemory, err := agentClient.ReadMemory()
if err != nil {
    log.Fatal(err)
}
fmt.Println(fullMemory)

// Read specific section
diagnostics, err := agentClient.ReadMemorySection("Diagnostic Patterns")
if err != nil {
    log.Fatal(err)
}
fmt.Println("Diagnostic Patterns:", diagnostics)

// List all sections
sections, err := agentClient.ListMemorySections()
for _, section := range sections {
    fmt.Println("- ", section)
}
```

### Example 3: Record Diagnostic Findings

```go
// Record a diagnostic finding automatically
err := agentClient.RecordDiagnosticFinding(
    "github-server",
    "Server failed to start due to missing credentials",
    "Added GITHUB_TOKEN environment variable to server config",
)
if err != nil {
    log.Fatal(err)
}

// Adds entry like:
// **[2025-11-01 10:30:00 - github-server]**
// - Issue: Server failed to start due to missing credentials
// - Solution: Added GITHUB_TOKEN environment variable to server config
```

### Example 4: Record Tool Discoveries

```go
// Record a useful tool discovery
err := agentClient.RecordToolDiscovery(
    "github-server",
    "create_issue",
    "Automatically create GitHub issues from diagnostic findings",
)
if err != nil {
    log.Fatal(err)
}

// Adds entry like:
// **[2025-11-01 10:30:00 - github-server:create_issue]**
// - Use Case: Automatically create GitHub issues from diagnostic findings
```

### Example 5: Record Successful Workflows

```go
// Record a successful workflow pattern
workflowDescription := `
This workflow successfully diagnosed and fixed a connection timeout issue:
1. Used ListTools to discover available diagnostic tools
2. Called get_logs to retrieve error messages
3. Identified missing environment variable
4. Updated server config with correct credentials
5. Verified fix by testing connection
`

err := agentClient.RecordSuccessfulWorkflow(
    "Connection Timeout Fix",
    workflowDescription,
)
```

### Example 6: Manual Memory Updates

```go
// Append content to a section
err := agentClient.UpdateMemorySection(
    "Performance Insights",
    "- Caching tool responses reduced API calls by 80%",
    true, // append mode
)

// Replace entire section content
err := agentClient.UpdateMemorySection(
    "Custom Notes",
    "New custom content here...",
    false, // replace mode
)

// Add a timestamped note
err := agentClient.AppendMemoryNote(
    "Discovered that tool X works better with parameter Y set to Z",
)
```

### Example 7: Export Memory Sections

```go
// Export a specific section to a separate file
err := agentClient.ExportMemorySection(
    "Diagnostic Patterns",
    "./reports/diagnostic-patterns-export.md",
)
if err != nil {
    log.Fatal(err)
}

// Creates a standalone markdown file with the section content
```

### Example 8: Memory Lifecycle Management

```go
// Get memory path
memoryPath, err := agentClient.GetMemoryPath()
fmt.Println("Memory stored at:", memoryPath)

// Manually save memory (auto-save is enabled by default)
err = agentClient.SaveMemory()

// Reload memory from disk (useful if externally modified)
err = agentClient.ReloadMemory()

// Check if memory is initialized
if agentClient.HasMemory() {
    fmt.Println("Memory is ready")
}
```

## API Reference

### AgentClient Memory Methods

#### Initialization
```go
// Create client with memory support
NewAgentClientWithMemory(baseURL string, dataDir string, logger *zap.Logger) *AgentClient

// Manually initialize memory (if using regular NewAgentClient)
InitializeMemory(dataDir string) error

// Check if memory is initialized
HasMemory() bool
```

#### Reading Memory
```go
// Get full memory content
ReadMemory() (string, error)

// Get specific section content
ReadMemorySection(sectionName string) (string, error)

// List all available sections
ListMemorySections() ([]string, error)

// Get memory file path
GetMemoryPath() (string, error)
```

#### Updating Memory
```go
// Update or append to a section
UpdateMemorySection(sectionName, content string, append bool) error

// Add timestamped note to Custom Notes
AppendMemoryNote(note string) error

// Record diagnostic finding
RecordDiagnosticFinding(serverName, issue, solution string) error

// Record tool discovery
RecordToolDiscovery(serverName, toolName, useCase string) error

// Record successful workflow
RecordSuccessfulWorkflow(workflowName, description string) error
```

#### Memory Management
```go
// Manually save memory to disk
SaveMemory() error

// Reload memory from disk
ReloadMemory() error

// Export section to file
ExportMemorySection(sectionName, outputPath string) error
```

### AgentMemory Direct Methods

For advanced use cases with direct `AgentMemory` access:

```go
// Get memory instance
memory, err := agentClient.GetMemory()

// Core operations
GetFullMemory() string
GetSection(sectionName string) (string, bool)
ListSections() []string
AppendToSection(sectionName, content string) error
UpdateSection(sectionName, newContent string) error
AddCustomNote(note string) error
ExportSection(sectionName, outputPath string) error

// Lifecycle
Save() error
Reload() error
SetAutoSave(enabled bool)
GetMemoryPath() string
```

## Memory Sections

The memory is organized into predefined sections:

1. **Server Configurations**
   - Known Issues: Server setup problems and solutions
   - Successful Patterns: Working configurations

2. **Diagnostic Patterns**
   - Common Problems: Frequently encountered issues
   - Solution Strategies: Effective diagnostic approaches

3. **Tool Discoveries**
   - High-Value Tools: Particularly useful tools
   - Tool Combinations: Effective tool patterns

4. **Performance Insights**
   - Bottlenecks: Performance issues
   - Best Practices: Optimization patterns

5. **Security Findings**
   - Vulnerabilities: Security issues found
   - Security Patterns: Effective security configurations

6. **Agent Learnings**
   - Successful Workflows: Patterns that work well
   - Failed Approaches: Approaches to avoid
   - Optimization Opportunities: Potential improvements

7. **Custom Notes**
   - Free-form notes and observations

## Integration Patterns

### Pattern 1: Automatic Learning from Diagnostics

```go
// Run diagnostic
diagnosticReport, err := agentClient.DiagnoseServer(serverName)
if err != nil {
    return err
}

// If issues found, record them
if len(diagnosticReport.Issues) > 0 {
    for i, issue := range diagnosticReport.Issues {
        solution := ""
        if i < len(diagnosticReport.Recommendations) {
            solution = diagnosticReport.Recommendations[i]
        }

        agentClient.RecordDiagnosticFinding(serverName, issue, solution)
    }
}
```

### Pattern 2: Learning from Tool Discovery

```go
// Discover tools from standalone server
catalog, err := agentClient.DiscoverToolCatalog(standaloneServer)
if err != nil {
    return err
}

// Record valuable tools
for _, tool := range catalog.Tools {
    if isHighValueTool(tool) {
        agentClient.RecordToolDiscovery(
            catalog.ServerName,
            tool.Name,
            tool.Description,
        )
    }
}
```

### Pattern 3: Workflow Documentation

```go
// Execute complex workflow
startTime := time.Now()
steps := []string{}

// Step 1
steps = append(steps, "1. Listed available servers")
servers, _ := agentClient.ListServers()

// Step 2
steps = append(steps, "2. Ran diagnostics on failing server")
report, _ := agentClient.DiagnoseServer("failing-server")

// Step 3
steps = append(steps, "3. Applied recommended fix")
// ... apply fix ...

// Record successful workflow
if workflowSuccessful {
    duration := time.Since(startTime)
    workflowDescription := fmt.Sprintf(`
Completed in %v:
%s
`, duration, strings.Join(steps, "\n"))

    agentClient.RecordSuccessfulWorkflow(
        "Server Recovery Workflow",
        workflowDescription,
    )
}
```

## Best Practices

1. **Automatic Recording**: Use the convenience methods (`RecordDiagnosticFinding`, `RecordToolDiscovery`) for consistent formatting

2. **Structured Sections**: Keep entries organized in appropriate sections for easy retrieval

3. **Timestamped Entries**: All recording methods add automatic timestamps for historical tracking

4. **Regular Exports**: Periodically export sections for backup and analysis

5. **Memory Initialization**: Use `NewAgentClientWithMemory` to ensure memory is loaded at startup

6. **Error Handling**: Always check errors when updating memory to avoid data loss

7. **Section Names**: Use exact section names from the template for consistency

8. **Custom Notes**: Use the Custom Notes section for ad-hoc observations

## Troubleshooting

### Memory File Not Found
Memory is automatically created on first access. If you see errors, check:
- Data directory exists and is writable
- User has permissions to create files
- Path is absolute, not relative

### Memory Not Loading
```go
// Check if memory is initialized
if !agentClient.HasMemory() {
    // Manually initialize if needed
    err := agentClient.InitializeMemory("/path/to/data/dir")
    if err != nil {
        log.Fatalf("Failed to initialize memory: %v", err)
    }
}
```

### Concurrent Access Issues
The memory system uses `sync.RWMutex` for thread safety. All operations are automatically synchronized.

### Section Not Found
Ensure you're using the exact section name from the template:
```go
sections, _ := agentClient.ListMemorySections()
for _, s := range sections {
    fmt.Println("Available section:", s)
}
```

## Performance Considerations

- **Auto-Save**: Memory is automatically saved after each update (can be disabled)
- **Section Parsing**: Happens only on load/reload, not on every read
- **Thread-Safety**: Uses read-write locks for optimal concurrent access
- **File Size**: Keep memory file under 1MB for best performance
- **Export Large Sections**: Use `ExportSection` for archiving old entries

## Summary

The Agent Memory system provides:
- ✅ Persistent cross-session knowledge base
- ✅ Structured organization with predefined sections
- ✅ Thread-safe concurrent access
- ✅ Automatic timestamping of entries
- ✅ Convenient recording methods
- ✅ Export capabilities
- ✅ Seamless integration with AgentClient
- ✅ Automatic initialization and persistence

This enables the agent to learn from experience and maintain context across multiple sessions, improving diagnostic accuracy and operational efficiency over time.
