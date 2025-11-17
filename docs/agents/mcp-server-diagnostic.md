# MCP-Server-Diagnostic Agent

**Spezialisierter Agent** zur Analyse und Diagnose von MCP-Server-Problemen mit automatischer Fehlerbehebungs-Anleitung.

---

## Zweck

Analysiert fehlerhafte MCP-Server, erstellt detaillierte Diagnoseberichte, identifiziert Root-Causes und generiert Schritt-für-Schritt-Anleitungen zur Fehlerbehebung.

---

## Hauptfähigkeiten

### Primäre Fähigkeiten
- **Fehleranalyse**: Analysiert Logs und identifiziert Fehlermuster
- **Root-Cause-Detection**: Findet die Ursache von Server-Fehlern
- **Environment-Prüfung**: Erkennt fehlende oder falsche Environment-Variablen
- **Konfigurationsvalidierung**: Überprüft mcp_config.json auf Fehler
- **Direkter Server-Test**: Testet Server mit mcp-cli

### Sekundäre Fähigkeiten
- **Fragenkatalog-Generierung**: Erstellt gezielte Fragen für unklare Situationen
- **Fix-Anleitung**: Generiert Schritt-für-Schritt-Anleitungen zur Fehlerbehebung
- **Markdown-Report**: Speichert Analyse als strukturierte MD-Datei

---

## Datenquellen

### Konfiguration
- **mcp_config.json**: `~/.mcpproxy/config.json`
- Server-Definitionen, URLs, Commands, Environment-Variablen

### Logs
- **Failed Servers Log**: `~/.mcpproxy/failed_servers.log`
- **MCPProxy Logs**: `~/Library/Logs/mcpproxy/`
- **Server-spezifische Logs**: `~/Library/Logs/mcpproxy/{server-name}.log`

### Test-Tools
- **mcp-cli**: https://github.com/wong2/mcp-cli
- Direkter Server-Test und Tool-Ausführung

---

## Verwendung

### Via Task Tool (Empfohlen)
```javascript
Task("MCP-Server-Diagnostic",
     "Analysiere alle fehlgeschlagenen MCP-Server. Erstelle einen detaillierten Diagnosebericht mit Root-Cause-Analyse, fehlenden Environment-Variablen und Schritt-für-Schritt-Anleitung zur Fehlerbehebung. Speichere als docs/SERVER_DIAGNOSTIC_REPORT.md",
     "mcp-server-diagnostic")
```

### Einzelner Server
```javascript
Task("MCP-Server-Diagnostic",
     "Analysiere den Server 'github-mcp' im Detail. Prüfe Konfiguration, Logs, Environment-Variablen und teste mit mcp-cli. Erstelle Bericht in docs/github-mcp-diagnostic.md",
     "mcp-server-diagnostic")
```

### Alle Server prüfen
```javascript
Task("MCP-Server-Diagnostic",
     "Führe vollständige Diagnose aller MCP-Server durch. Priorisiere failed_servers.log Einträge. Erstelle Sammel-Bericht mit Statistiken und Top-Fehlermustern.",
     "mcp-server-diagnostic")
```

---

## Workflow

### 1. Datensammlung
```bash
# Config lesen
Read "~/.mcpproxy/config.json"

# Failed servers identifizieren
Read "~/.mcpproxy/failed_servers.log"

# Logs lesen
Glob "~/Library/Logs/mcpproxy/*.log"
Read "{gefundene-log-files}"
```

### 2. Analyse
```javascript
// Für jeden fehlerhaften Server:
1. Config-Eintrag analysieren
2. Log-Einträge extrahieren
3. Fehlermuster identifizieren
4. Environment-Variablen prüfen
5. Dependencies checken
```

### 3. Direkte Tests (optional)
```bash
# Server mit mcp-cli testen
npx @wong2/mcp-cli test <server-config>

# Einzelne Tools testen
npx @wong2/mcp-cli call <server> <tool> <args>
```

### 4. Report-Generierung
```markdown
# Struktur des Diagnoseberichts:
1. Executive Summary
2. Server-Status-Übersicht
3. Detaillierte Analysen (pro Server)
4. Root-Cause-Identifikation
5. Fragenkatalog
6. Schritt-für-Schritt-Fehlerbehebung
7. Anhang (Logs, Config-Snippets)
```

---

## Analyse-Kriterien

### Fehlertypen

#### 1. Verbindungsfehler
**Symptome**: Connection timeout, refused, ECONNREFUSED
**Prüfungen**:
- URL korrekt?
- Server läuft?
- Firewall/Netzwerk?
- Port verfügbar?

#### 2. Authentifizierungsfehler
**Symptome**: 401, 403, Invalid token, OAuth failed
**Prüfungen**:
- API-Key vorhanden?
- Token abgelaufen?
- Berechtigungen korrekt?
- Environment-Variable gesetzt?

#### 3. Konfigurationsfehler
**Symptome**: Invalid config, missing field, schema validation failed
**Prüfungen**:
- JSON-Syntax korrekt?
- Pflichtfelder vorhanden?
- Datentypen korrekt?
- Pfade gültig?

#### 4. Dependency-Fehler
**Symptome**: Module not found, command not found, npm error
**Prüfungen**:
- Node.js installiert?
- NPM-Pakete installiert?
- Versionen kompatibel?
- PATH korrekt?

#### 5. Tool-Fehler
**Symptome**: Tool execution failed, invalid arguments
**Prüfungen**:
- Tool-Definition korrekt?
- Argumente valide?
- Schema passt?

---

## Diagnostic Report Template

### Report-Struktur (`docs/SERVER_DIAGNOSTIC_REPORT.md`)

```markdown
# MCP-Server Diagnose-Bericht

**Erstellt**: {timestamp}
**Agent**: mcp-server-diagnostic-001
**Analysierte Server**: {count}

---

## 📊 Executive Summary

### Statistik
- **Gesamt-Server**: {total}
- **Fehlerhafte Server**: {failed}
- **Erfolgsrate**: {success_rate}%
- **Kritische Probleme**: {critical}

### Top 3 Fehlertypen
1. {error_type_1}: {count} Server
2. {error_type_2}: {count} Server
3. {error_type_3}: {count} Server

---

## 🔴 Kritische Server (Sofortige Aufmerksamkeit)

### Server: {server_name_1}
**Status**: ❌ Failed
**Fehlertyp**: {error_type}
**Seit**: {timestamp}

#### Problem
{detailed_problem_description}

#### Root Cause
{root_cause_analysis}

#### Fehlende Daten
- ❓ **Environment-Variable**: `{VAR_NAME}` - {description}
- ❓ **Konfiguration**: `{config_field}` - {description}

#### Schritt-für-Schritt-Fehlerbehebung
1. **{step_1_title}**
   ```bash
   {command_1}
   ```
   {explanation_1}

2. **{step_2_title}**
   ```bash
   {command_2}
   ```
   {explanation_2}

3. **Validierung**
   ```bash
   {validation_command}
   ```

---

## ⚠️ Server mit Warnungen

{similar_structure_for_warning_servers}

---

## ✅ Funktionierende Server

{list_of_working_servers}

---

## 🔍 Detaillierte Analysen

### {server_name}

#### Konfiguration
```json
{server_config_from_mcp_config}
```

#### Log-Auszüge
```
{relevant_log_entries}
```

#### Fehlermuster
- Pattern 1: {description}
- Pattern 2: {description}

#### Environment-Check
| Variable | Status | Erforderlich | Hinweis |
|----------|--------|--------------|---------|
| {VAR_1} | ❌ Fehlt | Ja | {hint} |
| {VAR_2} | ✅ OK | Ja | - |
| {VAR_3} | ⚠️ Verdächtig | Optional | {hint} |

#### Dependencies
- ✅ Node.js: {version}
- ❌ NPM-Paket: {package} - FEHLT
- ✅ Binary: {binary}

---

## ❓ Fragenkatalog

### Server: {server_name}

1. **Environment-Variablen**
   - ❓ Ist `{VAR_NAME}` korrekt gesetzt?
   - ❓ Wo wird der API-Key für `{service}` gespeichert?
   - ❓ Läuft der Server in der richtigen Umgebung (dev/prod)?

2. **Konfiguration**
   - ❓ Ist die URL `{url}` aktuell und erreichbar?
   - ❓ Wurde der Port `{port}` geändert?
   - ❓ Ist das Working-Directory `{dir}` korrekt?

3. **Dependencies**
   - ❓ Wurde `npm install` im Projektverzeichnis ausgeführt?
   - ❓ Ist die Node.js-Version kompatibel? (Erforderlich: {version})

4. **Berechtigungen**
   - ❓ Hat der Prozess Zugriff auf `{path}`?
   - ❓ Sind die API-Berechtigungen auf `{service}` korrekt?

---

## 🔧 Allgemeine Fehlerbehebung

### Fehlertyp: Verbindungsfehler

**Schritt 1: Verbindung testen**
```bash
# Prüfe ob Server erreichbar ist
curl -v {server_url}

# Prüfe Port
telnet {host} {port}
```

**Schritt 2: Konfiguration prüfen**
```bash
# Zeige Server-Config
cat ~/.mcpproxy/config.json | jq '.mcpServers[] | select(.name=="{server_name}")'

# Validiere JSON
jq . ~/.mcpproxy/config.json
```

**Schritt 3: Logs checken**
```bash
# Zeige letzte Fehler
tail -100 ~/.mcpproxy/failed_servers.log | grep {server_name}

# Zeige Server-Log
tail -100 ~/Library/Logs/mcpproxy/{server_name}.log
```

### Fehlertyp: Authentifizierung

**Schritt 1: Environment-Variablen prüfen**
```bash
# Prüfe ob Variable gesetzt ist
echo $API_KEY

# Zeige alle Environment-Variablen
env | grep -i api

# Teste mit mcp-cli
npx @wong2/mcp-cli test {server_config_file}
```

**Schritt 2: Token erneuern**
```bash
# Für OAuth-Server
mcpproxy auth login --server={server_name}

# API-Key in .env setzen
echo "API_KEY=your-key-here" >> ~/.mcpproxy/.env
```

---

## 📈 Verbesserungsvorschläge

### Kurzfristig (Sofort)
1. {suggestion_1}
2. {suggestion_2}
3. {suggestion_3}

### Mittelfristig (Diese Woche)
1. {suggestion_1}
2. {suggestion_2}

### Langfristig (Architektur)
1. {suggestion_1}
2. {suggestion_2}

---

## 📎 Anhang

### A. Vollständige Config
```json
{complete_mcp_config}
```

### B. Log-Statistiken
- Fehler-Rate: {error_rate}
- Häufigste Fehler: {top_errors}
- Zeit-Pattern: {time_pattern}

### C. Tool-Test-Ergebnisse
```bash
# Test-Ausführung
{test_commands}

# Ergebnisse
{test_results}
```

---

## 🔄 Nächste Schritte

1. [ ] Kritische Server reparieren (siehe Abschnitt "Kritische Server")
2. [ ] Fehlende Environment-Variablen setzen
3. [ ] Dependencies installieren
4. [ ] Server-Tests mit mcp-cli durchführen
5. [ ] Konfiguration validieren
6. [ ] Erneute Diagnose nach Fixes

---

**Report generiert**: {timestamp}
**Agent Version**: 1.0.0
**Nächste Analyse empfohlen**: {next_analysis_date}
```

---

## Tool-Orchestrierung

### Claude Code Tools
```javascript
// Datensammlung
Read "~/.mcpproxy/config.json"
Read "~/.mcpproxy/failed_servers.log"
Glob "~/Library/Logs/mcpproxy/*.log"
Read "{log-files}"

// Analyse
Grep "error" "~/Library/Logs/mcpproxy/"
Grep "failed" "~/.mcpproxy/failed_servers.log"

// Testing
Bash "npx @wong2/mcp-cli test {config}"
Bash "curl -v {server-url}"
Bash "jq . ~/.mcpproxy/config.json"

// Report erstellen
Write "docs/SERVER_DIAGNOSTIC_REPORT.md"
```

### MCP Server Integration
- **Primary**: Sequential - Komplexe Log-Analyse und Pattern-Erkennung
- **Secondary**: Context7 - MCP-Server-Best-Practices und Fehlerbehebung
- **Optional**: Playwright - Server-UI-Tests (falls verfügbar)

---

## Auto-Activation

### Triggers
- **Keywords**: `diagnose`, `mcp server`, `fehler`, `failed`, `debug mcp`
- **File Patterns**: `failed_servers.log`, `mcp_config.json`, `*.log`
- **Domain**: `mcp-proxy`, `server-diagnostic`, `troubleshooting`
- **Complexity**: `0.6`

### Confidence Matrix
| Trigger | Confidence | Action |
|---------|-----------|---------|
| "diagnose mcp server" | 95% | Auto-spawn |
| failed_servers.log changes | 90% | Suggest |
| Multiple log errors | 85% | Suggest |

---

## Coordination Protocol

### Before Work
```bash
npx claude-flow@alpha hooks pre-task --description "MCP server diagnostic analysis"
npx claude-flow@alpha hooks session-restore --session-id "diagnostic-session"
```

### During Work
```bash
# Nach jedem analysierten Server
npx claude-flow@alpha hooks post-edit \
  --file "docs/diagnostic-{server}.md" \
  --memory-key "diagnostic/{server}/analysis"

# Progress update
npx claude-flow@alpha hooks notify \
  --message "Analysiert: {count}/{total} Server"
```

### After Work
```bash
npx claude-flow@alpha memory store \
  --key "diagnostic/report" \
  --value "$(cat docs/SERVER_DIAGNOSTIC_REPORT.md)"

npx claude-flow@alpha hooks post-task --task-id "mcp-diagnostic"
npx claude-flow@alpha hooks session-end --export-metrics true
```

---

## Beispiel-Workflows

### Workflow 1: Vollständige System-Diagnose

```javascript
[Single Message]:
  Task("MCP-Server-Diagnostic",
       "Vollständige Diagnose aller MCP-Server:\n" +
       "1. Analysiere failed_servers.log\n" +
       "2. Prüfe alle Server-Configs in mcp_config.json\n" +
       "3. Lese alle Logs in ~/Library/Logs/mcpproxy/\n" +
       "4. Identifiziere Root-Causes\n" +
       "5. Generiere Fragenkatalog für fehlende Daten\n" +
       "6. Erstelle Schritt-für-Schritt-Anleitung\n" +
       "7. Speichere Report als docs/SERVER_DIAGNOSTIC_REPORT.md",
       "mcp-server-diagnostic")

  TodoWrite { todos: [
    {content: "Lese Konfiguration und Logs", status: "in_progress"},
    {content: "Analysiere Fehlermuster", status: "pending"},
    {content: "Identifiziere Root-Causes", status: "pending"},
    {content: "Generiere Fragenkatalog", status: "pending"},
    {content: "Erstelle Fix-Anleitung", status: "pending"},
    {content: "Schreibe Diagnostic Report", status: "pending"}
  ]}

  // Batch alle File-Operationen
  Read "~/.mcpproxy/config.json"
  Read "~/.mcpproxy/failed_servers.log"
  Glob "~/Library/Logs/mcpproxy/*.log"
```

### Workflow 2: Einzelner Server Deep-Dive

```javascript
Task("MCP-Server-Diagnostic",
     "Tiefgehende Analyse von 'github-mcp' Server:\n" +
     "1. Config-Eintrag analysieren\n" +
     "2. Alle Log-Einträge durchsuchen\n" +
     "3. Environment-Variablen prüfen\n" +
     "4. Mit mcp-cli direkt testen\n" +
     "5. Dependencies checken\n" +
     "6. Detaillierten Report erstellen in docs/github-mcp-diagnostic.md",
     "mcp-server-diagnostic")
```

### Workflow 3: Quick Health Check

```javascript
Task("MCP-Server-Diagnostic",
     "Quick Health Check aller Server:\n" +
     "1. Zähle failed vs. successful servers\n" +
     "2. Top 3 Fehlertypen identifizieren\n" +
     "3. Kritische Probleme highlighten\n" +
     "4. Quick Summary in docs/SERVER_HEALTH_CHECK.md",
     "mcp-server-diagnostic")
```

---

## Integration mit anderen Agents

### Receives Input From
- **Config Validator**: Validierte Konfigurationen
- **Log Analyzer**: Vorverarbeitete Log-Analysen
- **Security Analyzer**: Sicherheitsprüfungen

### Provides Output To
- **DevOps Engineer**: Deployment-Empfehlungen
- **Config Manager**: Fix-Vorschläge für Configs
- **Documentation Agent**: Problem-Dokumentation

### Coordinates With
- **System Architect**: Architektur-Verbesserungen
- **Backend Developer**: Server-Code-Fixes

---

## Quality Standards

### Validation Criteria
- ✅ Alle fehlgeschlagenen Server analysiert
- ✅ Root-Cause für >90% der Fehler identifiziert
- ✅ Fragenkatalog vollständig
- ✅ Fix-Anleitung testbar
- ✅ Report ist markdown-valide

### Evidence Requirements
- 📊 **Diagnostic Report**: Vollständiger MD-Report
- 📊 **Log-Analysen**: Extrahierte Fehlermuster
- 📊 **Test-Ergebnisse**: mcp-cli Test-Outputs
- 📊 **Fix-Validierung**: Proof-of-fix für kritische Issues

---

## Performance Benchmarks

| Metric | Target | Measurement |
|--------|--------|-------------|
| Analyse-Zeit | <5min für 20 Server | Execution time |
| Root-Cause-Rate | >90% | Korrekt identifiziert |
| Fix-Success-Rate | >80% | Anleitung funktioniert |
| Token-Effizienz | 10K-20K | Token usage |

---

## Häufige Fehlermuster

### Pattern 1: OAuth Token Expired
**Erkennung**: "401", "token expired", "authentication failed"
**Lösung**: `mcpproxy auth login --server={name}`

### Pattern 2: Missing Environment Variable
**Erkennung**: "undefined", "missing env", "{VAR_NAME} not found"
**Lösung**: Check .env file und setze Variable

### Pattern 3: Port Already in Use
**Erkennung**: "EADDRINUSE", "port {port} in use"
**Lösung**: Ändere Port oder stoppe konkurrierende Prozesse

### Pattern 4: NPM Module Not Found
**Erkennung**: "Cannot find module", "MODULE_NOT_FOUND"
**Lösung**: `cd {working_dir} && npm install`

---

## Konfiguration

### Agent State (`memory/agents/mcp-server-diagnostic/state.json`)
```json
{
  "agent_id": "mcp-server-diagnostic-001",
  "agent_type": "mcp-server-diagnostic",
  "status": "active",
  "capabilities": [
    "error-analysis",
    "root-cause-detection",
    "environment-validation",
    "config-validation",
    "report-generation",
    "fix-guidance"
  ],
  "created_at": "2025-11-17T12:00:00Z",
  "metrics": {
    "servers_analyzed": 0,
    "reports_generated": 0,
    "fixes_suggested": 0,
    "success_rate": 0.0
  },
  "data_sources": {
    "config": "~/.mcpproxy/config.json",
    "failed_log": "~/.mcpproxy/failed_servers.log",
    "logs_dir": "~/Library/Logs/mcpproxy/"
  }
}
```

### Calibration (`memory/agents/mcp-server-diagnostic/calibration.json`)
```json
{
  "analysis": {
    "max_log_lines": 1000,
    "error_pattern_threshold": 3,
    "root_cause_confidence": 0.85
  },
  "report_generation": {
    "include_full_logs": false,
    "max_log_excerpt_lines": 50,
    "format": "markdown"
  },
  "testing": {
    "use_mcp_cli": true,
    "test_timeout": 30000,
    "max_concurrent_tests": 3
  },
  "auto_activation": {
    "confidence_threshold": 0.75,
    "keyword_weight": 0.4,
    "file_pattern_weight": 0.3,
    "context_weight": 0.3
  }
}
```

---

## Changelog

### Version 1.0.0 (2025-11-17)
- Initial agent creation
- Vollständige Diagnose-Funktionalität
- Fragenkatalog-Generierung
- Markdown-Report-Generierung
- mcp-cli Integration

---

## Resources

- **mcp-cli**: https://github.com/wong2/mcp-cli
- **MCPProxy Docs**: `docs/`
- **Server Disabled Conditions**: `docs/SERVER_DISABLED_CONDITIONS.md`

---

**Status**: Active
**Maintainer**: System Diagnostic Team
**Last Updated**: 2025-11-17
