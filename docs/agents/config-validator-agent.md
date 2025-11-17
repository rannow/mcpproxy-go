# Config Validator Agent

> **Example Agent**: This demonstrates a complete, working agent implementation.

---

## Purpose
Specialized agent for validating configuration files, detecting errors, ensuring consistency, and suggesting fixes across YAML, JSON, TOML, and environment files.

---

## Capabilities

### Primary Capabilities
- **Configuration Validation**: Parse and validate config files for syntax and semantic errors
- **Schema Compliance**: Verify configs against schemas and best practices
- **Cross-File Consistency**: Ensure consistency across related configuration files
- **Security Scanning**: Detect hardcoded secrets, insecure settings, and vulnerabilities

### Secondary Capabilities
- **Auto-Fix Suggestions**: Generate fixes for common configuration errors
- **Documentation Generation**: Create config documentation from schemas
- **Migration Assistance**: Help migrate configs between versions

---

## When to Use

Use this agent for:
- ✅ Validating configuration files before deployment
- ✅ Detecting configuration errors in existing projects
- ✅ Ensuring consistency across multiple config files
- ✅ Security auditing of configuration settings
- ✅ Migrating configurations to new formats/versions

Do NOT use this agent for:
- ❌ Writing application code (use `coder` agent)
- ❌ Database schema validation (use `database-specialist` agent)
- ❌ Performance optimization (use `performance-optimizer` agent)

---

## Tool Orchestration

### Claude Code Tools
- **Read/Grep/Glob**: Search for config files and read content
- **Write/Edit**: Apply fixes to configuration files
- **Bash**: Run validation tools (yamllint, jsonlint, etc.)
- **TodoWrite**: Track validation steps across multiple files

### MCP Server Integration
- **Primary MCP**: Sequential - Complex multi-file validation logic
- **Secondary MCP**: Context7 - Configuration best practices and schemas
- **Optional MCP**: None required

---

## Auto-Activation

### Triggers
- **Keywords**: `config`, `validate`, `configuration`, `settings`, `environment`
- **File Patterns**: `*.yml`, `*.yaml`, `*.json`, `*.toml`, `*.env`, `config/*`
- **Domain Indicators**: `validation`, `lint`, `schema`, `compliance`
- **Complexity Threshold**: `0.5`

### Confidence Matrix

| Trigger Type | Match | Confidence | Action |
|-------------|-------|-----------|---------|
| File pattern + keyword | 3+ files | 95% | Auto-spawn immediately |
| Keyword "validate config" | Exact | 90% | Auto-spawn immediately |
| File pattern only | 5+ config files | 80% | Suggest to user |
| Domain context | Clear | 75% | Suggest with alternatives |

---

## Spawning Instructions

### Recommended: Task Tool
```javascript
Task("Config Validator",
     "Validate all configuration files in the project. Check for syntax errors, security issues, and cross-file consistency. Provide detailed error reports and fix suggestions.",
     "config-validator")
```

### Alternative: Direct Invocation
```bash
# Step 1: Pre-task hook
npx claude-flow@alpha hooks pre-task \
  --description "Config validation for project"

# Step 2: Validation work
# Agent validates configs and generates report

# Step 3: Post-task hook
npx claude-flow@alpha hooks post-task \
  --task-id "config-validation-001"
```

---

## Coordination Protocol

### Before Starting Work
```bash
# 1. Load context
npx claude-flow@alpha hooks session-restore \
  --session-id "swarm-config-validation"

# 2. Announce start
npx claude-flow@alpha hooks pre-task \
  --description "config-validator: Starting validation"

# 3. Check for previous validation results
npx claude-flow@alpha memory read \
  --key "swarm/config-validator/last-run"
```

### During Work
```bash
# Update after validating each file
npx claude-flow@alpha hooks post-edit \
  --file "config/app.yml" \
  --memory-key "swarm/config-validator/results/app-yml"

# Notify progress
npx claude-flow@alpha hooks notify \
  --message "config-validator: Validated 5/10 files"
```

### After Completing Work
```bash
# 1. Save validation report
npx claude-flow@alpha memory store \
  --key "swarm/config-validator/report" \
  --value "$(cat validation-report.json)"

# 2. Mark complete
npx claude-flow@alpha hooks post-task \
  --task-id "config-validation-001"

# 3. Export metrics
npx claude-flow@alpha hooks session-end \
  --export-metrics true
```

---

## Example Workflows

### Workflow 1: Pre-Deployment Validation

**Scenario**: Validate all configs before deploying to production

**Execution**:
```javascript
[Single Message - All Operations Together]:
  // Spawn validation agents
  Task("Config Validator", "Validate all config files for syntax, security, and consistency", "config-validator")
  Task("Security Analyzer", "Scan configs for security vulnerabilities", "code-analyzer")
  Task("DevOps Engineer", "Review deployment configs", "cicd-engineer")

  // Batch todos
  TodoWrite { todos: [
    {content: "Validate YAML/JSON syntax", status: "in_progress", activeForm: "Validating syntax"},
    {content: "Check for hardcoded secrets", status: "in_progress", activeForm: "Scanning for secrets"},
    {content: "Verify cross-file consistency", status: "pending", activeForm: "Checking consistency"},
    {content: "Generate validation report", status: "pending", activeForm: "Generating report"},
    {content: "Apply auto-fixes if approved", status: "pending", activeForm: "Applying fixes"}
  ]}

  // Batch file operations
  Read "config/app.yml"
  Read "config/database.yml"
  Read ".env.example"
  Read "docker-compose.yml"
```

**Expected Output**:
- Validation report with errors/warnings
- List of detected security issues
- Auto-fix suggestions for common errors
- Consistency check results across configs

---

### Workflow 2: Configuration Migration

**Scenario**: Migrate configs from v1 format to v2 format

**Execution**:
```javascript
[Wave 1 - Analysis]:
  Task("Config Validator", "Analyze v1 configs and identify migration requirements", "config-validator")

[Wave 2 - Migration]:
  Task("Config Validator", "Migrate configs to v2 format", "config-validator")
  Task("Coder", "Update code to use v2 configs", "coder")

[Wave 3 - Validation]:
  Task("Config Validator", "Validate migrated configs", "config-validator")
  Task("Tester", "Test with new configs", "tester")
```

**Expected Output**:
- Migrated configuration files
- Migration report with changes
- Updated application code
- Validation results for new configs

---

## Quality Standards

### Validation Criteria
- ✅ All config files have valid syntax (JSON, YAML, TOML)
- ✅ No hardcoded secrets or sensitive data
- ✅ Schema compliance for all validated files
- ✅ Cross-file references are valid
- ✅ Required fields are present

### Evidence Requirements
- 📊 **Validation Report**: JSON report with all errors/warnings
- 📊 **Security Scan Results**: List of detected vulnerabilities
- 📊 **Fix Suggestions**: Auto-generated fixes for issues
- 📊 **Test Results**: Proof that configs load correctly

### Success Metrics
| Metric | Target | Measurement |
|--------|--------|-------------|
| Syntax Error Detection | 100% | All syntax errors found |
| Secret Detection Rate | >95% | Hardcoded secrets found |
| False Positive Rate | <5% | Invalid warnings |
| Auto-Fix Success | >80% | Fixes work without errors |

---

## Integration with Other Agents

### Receives Input From
- **DevOps Engineer**: Deployment requirements and constraints
- **Security Analyzer**: Security policies and compliance rules
- **Architect**: Configuration schemas and standards

### Provides Output To
- **DevOps Engineer**: Validated configs ready for deployment
- **Coder**: Config structure for application code
- **Tester**: Test configs and environment setup

### Coordinates With
- **Security Analyzer**: Security scanning of config values
- **Code Analyzer**: Validating code references to configs
- **Tester**: Ensuring configs work in test environments

---

## Performance Benchmarks

### Token Efficiency
- **Target Range**: 5,000-12,000 tokens
- **Average**: 8,000 tokens
- **Optimization**: Compression enabled for large configs

### Execution Time
- **Target**: 60 seconds for 10 config files
- **Timeout**: 180 seconds
- **Average**: 45 seconds

### Success Rate
- **Target**: 90%
- **Current**: 92% (based on testing)
- **Improvement Actions**: Enhanced schema detection, better error messages

---

## Common Issues & Solutions

### Issue 1: YAML Indentation Errors
**Symptoms**: Parser fails with indentation errors
**Root Cause**: Mixed tabs and spaces, or incorrect nesting
**Solution**:
```bash
# Use yamllint to detect issues
yamllint config/*.yml

# Auto-fix with prettier
npx prettier --write config/*.yml
```

### Issue 2: Environment Variable References
**Symptoms**: Unresolved ${VAR} references
**Root Cause**: Missing environment variables or typos
**Solution**:
```bash
# Check .env file for missing vars
grep -r '\${' config/ | sed 's/.*\${\([^}]*\)}.*/\1/' | sort -u > required-vars.txt
comm -23 required-vars.txt <(cut -d= -f1 .env | sort) # Shows missing vars
```

### Issue 3: Cross-File Inconsistencies
**Symptoms**: Port numbers, service names differ across files
**Root Cause**: Manual updates not synchronized
**Solution**:
```javascript
// Extract all service references
Grep "service_name:" "config/"
Grep "port:" "config/"

// Report inconsistencies
Edit "config/inconsistencies.md"
```

---

## Knowledge Base

### Learned Patterns

1. **Pattern**: YAML Validation Pipeline
   - **Context**: Validating multiple YAML files
   - **Implementation**: yamllint → schema validation → security scan → consistency check
   - **Benefits**: Catches 95% of errors before deployment

2. **Pattern**: Secret Detection
   - **Context**: Finding hardcoded secrets in configs
   - **Implementation**: Regex patterns + entropy analysis + known secret formats
   - **Benefits**: Prevents credential leaks

3. **Pattern**: Configuration Inheritance
   - **Context**: Base configs with environment overrides
   - **Implementation**: Merge base.yml + env-specific.yml, validate combined result
   - **Benefits**: Reduces duplication, ensures consistency

### Best Practices
- 📌 Always validate configs in CI/CD pipeline before deployment
- 📌 Use environment variables for sensitive data
- 📌 Maintain schemas for all critical configuration files
- 📌 Document required vs optional configuration fields
- 📌 Version control all configuration changes
- 📌 Test configs in staging before production

### Anti-Patterns
- ⚠️ Hardcoding secrets in config files: Use environment variables or secret managers
- ⚠️ Skipping validation in development: Catch errors early, not in production
- ⚠️ Manual config synchronization: Automate with templates and inheritance

---

## Configuration

### Agent State (`memory/agents/config-validator/state.json`)
```json
{
  "agent_id": "config-validator-001",
  "agent_type": "config-validator",
  "status": "active",
  "capabilities": [
    "yaml-validation",
    "json-validation",
    "secret-detection",
    "schema-compliance",
    "cross-file-consistency"
  ],
  "created_at": "2025-11-17T10:00:00Z",
  "last_active": "2025-11-17T11:30:00Z",
  "metrics": {
    "tasks_completed": 15,
    "success_rate": 0.92,
    "avg_execution_time": 45,
    "token_efficiency": 8000,
    "files_validated": 150,
    "errors_found": 42,
    "auto_fixes_applied": 35
  },
  "preferences": {
    "compression": true,
    "mcp_primary": "sequential",
    "auto_activate": true,
    "strict_mode": true
  }
}
```

### Calibration (`memory/agents/config-validator/calibration.json`)
```json
{
  "token_efficiency": {
    "target_range": "5000-12000",
    "compression_enabled": true,
    "batch_operations": true
  },
  "execution_time": {
    "target_seconds": 60,
    "timeout_seconds": 180
  },
  "quality_thresholds": {
    "validation_score": 0.98,
    "secret_detection_rate": 0.95,
    "false_positive_rate": 0.05,
    "auto_fix_success": 0.80
  },
  "auto_activation": {
    "confidence_threshold": 0.75,
    "keyword_weight": 0.35,
    "file_pattern_weight": 0.40,
    "context_weight": 0.15,
    "history_weight": 0.10
  },
  "mcp_servers": {
    "primary": "sequential",
    "secondary": "context7",
    "fallback": ["sequential"]
  },
  "validation_tools": {
    "yaml": ["yamllint", "prettier"],
    "json": ["jsonlint", "jq"],
    "toml": ["taplo"],
    "env": ["dotenv-linter"]
  }
}
```

---

## Testing

### Unit Tests
```bash
# Test basic validation
Task("Config Validator Test",
     "Validate test-config.yml for syntax and schema compliance",
     "config-validator")

# Verify output
cat docs/validation-report.json
grep -q "errors: 0" docs/validation-report.json && echo "✅ PASS" || echo "❌ FAIL"
```

### Integration Tests
```bash
# Test full workflow
[Single Message]:
  Task("Config Validator", "Validate all configs", "config-validator")
  Task("Security Analyzer", "Scan for secrets", "code-analyzer")
  Task("DevOps Engineer", "Review for deployment", "cicd-engineer")

# Verify coordination
cat ~/.claude-flow/memory/swarm/config-validator/*.json
tail -f ~/.claude-flow/logs/config-validator.log
```

### Performance Tests
```bash
# Validate 20 config files
time Task("Config Validator", "Validate configs/", "config-validator")

# Expected: <60 seconds
# Token usage: 5,000-12,000 tokens
```

---

## Validation Report Format

### JSON Report Structure
```json
{
  "timestamp": "2025-11-17T11:30:00Z",
  "agent": "config-validator-001",
  "summary": {
    "total_files": 10,
    "files_with_errors": 2,
    "total_errors": 5,
    "total_warnings": 8,
    "critical_issues": 1
  },
  "files": [
    {
      "path": "config/app.yml",
      "status": "valid",
      "errors": [],
      "warnings": [
        {
          "line": 15,
          "column": 3,
          "severity": "warning",
          "message": "Unused configuration key 'legacy_mode'",
          "suggestion": "Remove unused key or document its purpose"
        }
      ]
    },
    {
      "path": ".env",
      "status": "error",
      "errors": [
        {
          "line": 8,
          "severity": "critical",
          "message": "Hardcoded API key detected",
          "suggestion": "Move to secure secret manager",
          "auto_fix": false
        }
      ],
      "warnings": []
    }
  ],
  "security_issues": [
    {
      "file": ".env",
      "type": "hardcoded-secret",
      "severity": "critical",
      "description": "API_KEY contains what appears to be a real API key",
      "remediation": "Use environment variable or secret manager"
    }
  ],
  "auto_fixes": [
    {
      "file": "config/app.yml",
      "applied": true,
      "description": "Fixed YAML indentation on lines 12-15"
    }
  ]
}
```

---

## Changelog

### Version 1.0.0 (2025-11-17)
- Initial agent creation
- YAML, JSON, TOML, ENV validation
- Secret detection capability
- Auto-fix for common errors
- Schema compliance checking

### Version 1.1.0 (Future)
- Add XML validation support
- Enhanced secret detection with ML
- Configuration diff comparison
- Real-time validation via file watchers

---

## Resources

- **CLAUDE.md**: Main agent configuration
- **CREATE_NEW_AGENT.md**: Agent creation guide
- **AGENT_API.md**: REST API documentation
- **yamllint**: https://yamllint.readthedocs.io/
- **jsonlint**: https://github.com/zaach/jsonlint

---

## License

MIT License - Part of mcpproxy-go project

---

## Maintainers

- **Primary**: Config Validation Team
- **Contributors**: DevOps, Security Teams

---

**Last Updated**: 2025-11-17
**Status**: Active
