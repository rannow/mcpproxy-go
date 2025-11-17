# MCP-Server-Diagnostic Agent - Test & Verwendung

## Schnellstart

### 1. Agent direkt spawnen (Empfohlen)

```javascript
Task("MCP-Server-Diagnostic",
     "Analysiere alle fehlgeschlagenen MCP-Server. Erstelle einen detaillierten Diagnosebericht mit:\n" +
     "1. Root-Cause-Analyse für jeden fehlerhaften Server\n" +
     "2. Liste fehlender Environment-Variablen\n" +
     "3. Fragenkatalog für unklare Situationen\n" +
     "4. Schritt-für-Schritt-Anleitung zur Fehlerbehebung\n" +
     "Speichere den Report als docs/SERVER_DIAGNOSTIC_REPORT.md",
     "mcp-server-diagnostic")
```

### 2. Einzelnen Server analysieren

```javascript
Task("MCP-Server-Diagnostic",
     "Analysiere den Server 'github-mcp' im Detail:\n" +
     "- Lese Config-Eintrag aus ~/.mcpproxy/config.json\n" +
     "- Prüfe Logs in ~/Library/Logs/mcpproxy/github-mcp.log\n" +
     "- Teste mit mcp-cli\n" +
     "- Identifiziere Root-Cause\n" +
     "- Erstelle detaillierten Report in docs/github-mcp-diagnostic.md",
     "mcp-server-diagnostic")
```

### 3. Quick Health Check

```javascript
Task("MCP-Server-Diagnostic",
     "Quick Health Check:\n" +
     "- Zähle erfolgreiche vs. fehlgeschlagene Server\n" +
     "- Top 3 Fehlertypen\n" +
     "- Kritische Probleme highlighten\n" +
     "Speichere als docs/SERVER_HEALTH_CHECK.md",
     "mcp-server-diagnostic")
```

---

## Beispiel: Vollständige Diagnose

### Input (was der Agent macht)

```javascript
[Single Message - Alle Operationen zusammen]:
  Task("MCP-Server-Diagnostic",
       "Vollständige System-Diagnose durchführen",
       "mcp-server-diagnostic")

  TodoWrite { todos: [
    {content: "Config und Logs lesen", status: "in_progress", activeForm: "Lese Config und Logs"},
    {content: "Fehlermuster analysieren", status: "pending", activeForm: "Analysiere Fehlermuster"},
    {content: "Root-Causes identifizieren", status: "pending", activeForm: "Identifiziere Root-Causes"},
    {content: "Fragenkatalog erstellen", status: "pending", activeForm: "Erstelle Fragenkatalog"},
    {content: "Fix-Anleitung generieren", status: "pending", activeForm: "Generiere Fix-Anleitung"},
    {content: "Report schreiben", status: "pending", activeForm: "Schreibe Report"}
  ]}

  // Alle File-Operationen batchen
  Read "~/.mcpproxy/config.json"
  Read "~/.mcpproxy/failed_servers.log"
  Glob "~/Library/Logs/mcpproxy/*.log"
```

### Output (was du bekommst)

**Datei**: `docs/SERVER_DIAGNOSTIC_REPORT.md`

```markdown
# MCP-Server Diagnose-Bericht

**Erstellt**: 2025-11-17 12:30:00
**Agent**: mcp-server-diagnostic-001
**Analysierte Server**: 15

---

## 📊 Executive Summary

### Statistik
- **Gesamt-Server**: 15
- **Fehlerhafte Server**: 3
- **Erfolgsrate**: 80%
- **Kritische Probleme**: 1

### Top 3 Fehlertypen
1. Authentifizierung: 2 Server (GitHub, Slack)
2. Konfiguration: 1 Server (Database)
3. Dependencies: 0 Server

---

## 🔴 Kritische Server

### Server: github-mcp
**Status**: ❌ Failed
**Fehlertyp**: Authentifizierung
**Seit**: 2025-11-15 14:22:00

#### Problem
GitHub Personal Access Token ist abgelaufen. Server kann keine API-Anfragen mehr stellen.

#### Root Cause
Token wurde am 2025-10-15 erstellt mit 30-Tage-Gültigkeit und ist nun abgelaufen.

#### Fehlende Daten
- ❓ **Environment-Variable**: `GITHUB_TOKEN` - Neuer Personal Access Token erforderlich

#### Schritt-für-Schritt-Fehlerbehebung

1. **GitHub Token generieren**
   ```bash
   # Gehe zu: https://github.com/settings/tokens
   # Klicke "Generate new token (classic)"
   # Wähle Scopes: repo, read:org, read:user
   # Kopiere Token: ghp_xxxxxxxxxxxxxxxxxxxx
   ```

2. **Token in Environment setzen**
   ```bash
   # In ~/.mcpproxy/.env hinzufügen:
   echo "GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx" >> ~/.mcpproxy/.env

   # ODER in Config eintragen:
   jq '.mcpServers[] | select(.name=="github-mcp") | .env.GITHUB_TOKEN = "ghp_xxx"' \
     ~/.mcpproxy/config.json > ~/.mcpproxy/config.json.tmp
   mv ~/.mcpproxy/config.json.tmp ~/.mcpproxy/config.json
   ```

3. **Server neu starten**
   ```bash
   # mcpproxy neu starten
   pkill -f mcpproxy
   ./bin/mcpproxy serve
   ```

4. **Validierung**
   ```bash
   # Mit mcp-cli testen
   npx @wong2/mcp-cli test ~/.mcpproxy/servers/github-mcp.json

   # Status prüfen
   curl http://localhost:8080/api/v1/agent/servers/github-mcp | jq .status
   ```

---

## ❓ Fragenkatalog

### Server: github-mcp

1. **Environment-Variablen**
   - ❓ Hast du einen GitHub Personal Access Token erstellt?
   - ❓ Welche Scopes hat der Token? (Erforderlich: repo, read:org, read:user)
   - ❓ Ist der Token im richtigen Format? (Muss mit ghp_ beginnen)

2. **Konfiguration**
   - ❓ Ist die GITHUB_TOKEN Variable in .env oder config.json gesetzt?
   - ❓ Läuft mcpproxy mit den richtigen Permissions?

3. **Testing**
   - ❓ Funktioniert der Token direkt gegen GitHub API?
     ```bash
     curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user
     ```

---

(... weitere Server ...)
```

---

## Was der Agent analysiert

### 1. Konfiguration (`~/.mcpproxy/config.json`)
```json
{
  "mcpServers": [
    {
      "name": "github-mcp",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "${GITHUB_TOKEN}"  // ← Agent prüft ob gesetzt
      }
    }
  ]
}
```

**Agent prüft**:
- ✅ JSON-Syntax valide?
- ✅ Pflichtfelder vorhanden?
- ✅ Environment-Variablen referenziert?
- ✅ Pfade existieren?
- ✅ Ports verfügbar?

### 2. Failed Servers Log (`~/.mcpproxy/failed_servers.log`)
```
2025-11-17 10:15:23 [ERROR] github-mcp: Connection failed - 401 Unauthorized
2025-11-17 10:15:24 [ERROR] github-mcp: Token validation failed
2025-11-17 10:15:25 [ERROR] Auto-disabled after 3 consecutive failures
```

**Agent erkennt**:
- 🔍 Fehlertyp: Authentifizierung (401)
- 🔍 Pattern: 3 aufeinanderfolgende Fehler
- 🔍 Aktion: Auto-disabled
- 🔍 Zeitstempel: Wann begann das Problem?

### 3. Server Logs (`~/Library/Logs/mcpproxy/github-mcp.log`)
```
[2025-11-17 10:15:23] INFO  Starting server github-mcp
[2025-11-17 10:15:23] ERROR Token validation failed: Bad credentials
[2025-11-17 10:15:23] ERROR GitHub API returned 401 Unauthorized
[2025-11-17 10:15:24] INFO  Retry attempt 1/3
[2025-11-17 10:15:24] ERROR Token validation failed: Bad credentials
```

**Agent extrahiert**:
- 📊 Fehler-Timeline
- 📊 Retry-Versuche
- 📊 Spezifische Error-Messages
- 📊 Kontext um Fehler herum

### 4. Direct Testing mit mcp-cli
```bash
# Agent führt aus:
npx @wong2/mcp-cli test ~/.mcpproxy/servers/github-mcp.json

# Ergebnis:
❌ Connection failed: 401 Unauthorized
❌ Tool list failed: Authentication required
```

**Agent validiert**:
- ✅ Kann Server überhaupt starten?
- ✅ Sind Tools verfügbar?
- ✅ Funktionieren Basis-Operationen?

---

## Verwendungsszenarien

### Szenario 1: Nach mcpproxy-Neustart
```javascript
// Viele Server sind disabled, warum?
Task("MCP-Server-Diagnostic",
     "Analysiere warum Server nach Neustart disabled sind.\n" +
     "Prüfe failed_servers.log und erstelle Prioritätsliste.",
     "mcp-server-diagnostic")
```

### Szenario 2: Neuer Server funktioniert nicht
```javascript
// Gerade hinzugefügter Server startet nicht
Task("MCP-Server-Diagnostic",
     "Server 'new-weather-api' wurde gerade hinzugefügt aber startet nicht.\n" +
     "Analysiere Config, Dependencies und Environment.\n" +
     "Erstelle Setup-Anleitung in docs/new-weather-api-setup.md",
     "mcp-server-diagnostic")
```

### Szenario 3: Intermittierende Fehler
```javascript
// Server funktioniert manchmal, manchmal nicht
Task("MCP-Server-Diagnostic",
     "Server 'slack-mcp' hat intermittierende Fehler.\n" +
     "Analysiere Logs über letzte 7 Tage.\n" +
     "Identifiziere Pattern (Zeitpunkt, Häufigkeit, Trigger).",
     "mcp-server-diagnostic")
```

### Szenario 4: Komplettes System-Audit
```javascript
// Monatlicher Health-Check
Task("MCP-Server-Diagnostic",
     "Monatlicher System-Health-Check:\n" +
     "- Alle Server analysieren\n" +
     "- Trends identifizieren\n" +
     "- Proaktive Wartungsempfehlungen\n" +
     "- Report für Management: docs/MONTHLY_HEALTH_REPORT.md",
     "mcp-server-diagnostic")
```

---

## Integration mit anderen Agents

### Workflow: Full-Stack Diagnose + Fix

```javascript
[Single Message - Parallel Execution]:
  // 1. Diagnose
  Task("MCP-Server-Diagnostic",
       "Analysiere alle Server, erstelle Diagnose-Report",
       "mcp-server-diagnostic")

  // 2. Config-Validierung
  Task("Config Validator",
       "Validiere mcp_config.json auf Syntax- und Schema-Fehler",
       "config-validator")

  // 3. Sicherheitsprüfung
  Task("Security Analyzer",
       "Scanne Config auf hardcoded Secrets und unsichere Settings",
       "code-analyzer")

  // 4. DevOps Review
  Task("DevOps Engineer",
       "Review Deployment-Konfiguration und Dependencies",
       "cicd-engineer")

  TodoWrite { todos: [
    {content: "Server-Diagnose", status: "in_progress", activeForm: "Diagnostiziere Server"},
    {content: "Config-Validierung", status: "in_progress", activeForm: "Validiere Config"},
    {content: "Security-Scan", status: "in_progress", activeForm: "Scanne Security"},
    {content: "DevOps-Review", status: "in_progress", activeForm: "Reviewe Deployment"},
    {content: "Fixes anwenden", status: "pending", activeForm: "Wende Fixes an"},
    {content: "Re-Test", status: "pending", activeForm: "Teste erneut"}
  ]}
```

---

## Erwartete Ergebnisse

Nach Agent-Ausführung hast du:

### 📄 Diagnostic Report
**Datei**: `docs/SERVER_DIAGNOSTIC_REPORT.md`
- Executive Summary mit Statistiken
- Detaillierte Analyse pro Server
- Root-Cause für jeden Fehler
- Fragenkatalog für fehlende Informationen
- Schritt-für-Schritt-Fehlerbehebung
- Validierungs-Commands

### 📊 Insights
- Top 3 Fehlertypen
- Kritische vs. unkritische Probleme
- Zeitliche Muster (wann treten Fehler auf?)
- Dependency-Probleme
- Environment-Lücken

### 🔧 Actionable Fixes
- Konkrete Commands zum Ausführen
- Environment-Variablen zum Setzen
- Config-Änderungen mit Beispielen
- Test-Commands zur Validierung

### ❓ Fragenkatalog
- Was fehlt für vollständige Diagnose?
- Welche Informationen brauchst du?
- Welche Entscheidungen müssen getroffen werden?

---

## Nächste Schritte nach Diagnose

1. **Review Report**
   ```bash
   cat docs/SERVER_DIAGNOSTIC_REPORT.md
   ```

2. **Kritische Probleme zuerst**
   - Folge Schritt-für-Schritt-Anleitung
   - Validiere jeden Fix

3. **Environment-Variablen setzen**
   ```bash
   # Basierend auf Fragenkatalog
   echo "MISSING_VAR=value" >> ~/.mcpproxy/.env
   ```

4. **Re-Test**
   ```javascript
   Task("MCP-Server-Diagnostic",
        "Re-teste alle Server nach Fixes. Vergleiche mit vorherigem Report.",
        "mcp-server-diagnostic")
   ```

5. **Monitoring**
   - Überwache Logs weiterhin
   - Setze Alerts für kritische Fehler

---

## Troubleshooting

### Agent findet keine Logs
**Problem**: `~/Library/Logs/mcpproxy/` ist leer
**Lösung**: mcpproxy wurde vielleicht noch nie gestartet oder Logging ist disabled

### Report ist unvollständig
**Problem**: Einige Analysen fehlen
**Lösung**:
```javascript
// Mehr Context geben
Task("MCP-Server-Diagnostic",
     "Analysiere github-mcp mit maximalem Detail-Level.\n" +
     "Include: Full logs, complete config, all environment vars, mcp-cli test results",
     "mcp-server-diagnostic")
```

### mcp-cli Tests schlagen fehl
**Problem**: `npx @wong2/mcp-cli` nicht verfügbar
**Lösung**:
```bash
npm install -g @wong2/mcp-cli
```

---

## Performance

### Typische Ausführungszeiten
- **Einzelner Server**: ~15 Sekunden
- **10 Server**: ~2-3 Minuten
- **50+ Server**: ~5-8 Minuten

### Token-Verbrauch
- **Quick Check**: ~5K tokens
- **Vollständige Diagnose**: ~15-20K tokens
- **Mit mcp-cli Tests**: ~25K tokens

---

## Weitere Beispiele

### Beispiel 1: GitHub Server Debug
```javascript
Task("MCP-Server-Diagnostic",
     "GitHub MCP Server schlägt fehl mit 401.\n" +
     "Prüfe: Token validity, Scopes, Expiration, Config.\n" +
     "Teste mit mcp-cli gegen GitHub API.\n" +
     "docs/github-mcp-debug.md",
     "mcp-server-diagnostic")
```

### Beispiel 2: Database Connection Debug
```javascript
Task("MCP-Server-Diagnostic",
     "Postgres MCP kann nicht connecten.\n" +
     "Prüfe: Connection string, DB running, Permissions, Network.\n" +
     "Teste mit psql direkt.\n" +
     "docs/postgres-mcp-debug.md",
     "mcp-server-diagnostic")
```

### Beispiel 3: Alle stdio-Server checken
```javascript
Task("MCP-Server-Diagnostic",
     "Analysiere alle stdio-Server (command-based).\n" +
     "Prüfe: NPM dependencies, Working directory, Node version.\n" +
     "Identifiziere gemeinsame Probleme.\n" +
     "docs/stdio-servers-diagnostic.md",
     "mcp-server-diagnostic")
```

---

**Ready to use!** 🚀

Spawne den Agent jetzt und lass ihn deine MCP-Server analysieren!
