# MCP Servers Configuration - Zusammenfassung

## Übersicht

**Gesamtanzahl einzigartiger Server:** 159

**Verarbeitete Config-Dateien:** 22 (21 ins Archiv verschoben)

**Verbleibende aktive Config:** `~/.mcpproxy/mcp_config.json`

**Archivierte Configs:** `~/.mcpproxy/Archive/`

## Dateien-Status

### Aktiv
- ✅ `mcp_config.json` (71 Server)

### Archiviert (21 Dateien)
- `mcp_config Kopie 2.backup.20251116-121206.json`
- `mcp_config Kopie 2.json` (159 Server)
- `mcp_config Kopie 3.json` (159 Server)
- `mcp_config Kopie 4.json` (149 Server)
- `mcp_config Kopie 5.json` (148 Server)
- `mcp_config Kopie 6.json` (148 Server)
- `mcp_config Kopie.json` (159 Server)
- `mcp_config.backup-20251115-190047.json` (159 Server)
- `mcp_config.backup.20251118-083942.json` (149 Server)
- `mcp_config.backup.20251118-083943.json` (149 Server)
- `mcp_config.backup.20251118-083944.json` (149 Server)
- `mcp_config.backup.20251119-224747.json` (155 Server)
- `mcp_config.backup.20251119-232732.json` (148 Server)
- `mcp_config.backup.20251120-084108.json` (71 Server)
- `mcp_config.backup.20251121-134202.json` (71 Server)
- `mcp_config_connected_only.json` (148 Server)
- `mcp_config_tested.json` (155 Server)
- `mcp_config_ultra_clean.json` (79 Server)
- 3 Dateien mit Parse-Fehlern (korrupt/unvollständig)

## Konsolidierte Tabelle

Die vollständige konsolidierte Tabelle befindet sich in:
**[mcp_servers_consolidated.md](./mcp_servers_consolidated.md)**

### Spalten-Struktur

| Spalte | Beschreibung |
|--------|--------------|
| Name | Server-Name |
| Command | Ausführungsbefehl (npx, uvx, docker, etc.) |
| Args | Kommandozeilen-Argumente |
| Description | Server-Beschreibung |
| Protocol | Kommunikationsprotokoll (stdio, http, sse) |
| Repository URL | GitHub/Source Repository |
| Tool Count | Anzahl verfügbarer Tools |
| Ever Connected | Ob der Server jemals verbunden war (Yes/No) |
| Group ID | Zugewiesene Gruppe |
| Env | Umgebungsvariablen |
| Getestet | *Leer - für manuelle Tests* |
| Basic MCP | *Leer - für MCP-Standard-Klassifizierung* |

## Top Server nach Tool Count

1. **MCP_DOCKER** - 231 Tools
2. **awslabs.lambda-tool-mcp-server** - 50 Tools
3. **awslabs.iam-mcp-server** - 29 Tools
4. **awslabs.aws-serverless-mcp-server** - 25 Tools
5. **awslabs.stepfunctions-tool-mcp-server** - 16 Tools
6. **awslabs.eks-mcp-server** - 16 Tools
7. **archon** - 16 Tools

## Server-Kategorien

### AWS Services (28 Server)
- Lambda, EKS, ECS, IAM, CloudFormation, CDK, etc.
- Verschiedene AWS-spezifische MCP-Server

### Entwickler-Tools (20+ Server)
- Browser-Automation, Code-Sandboxes, Testing
- Versionskontrolle, Code-Analyse

### Datenbanken (8 Server)
- PostgreSQL, MySQL, SQLite, BigQuery, etc.

### Integration & APIs (15+ Server)
- GitHub, GitLab, Google Drive, Slack, etc.

### AI & ML (10+ Server)
- Various AI model integrations
- Embedding-Dienste

## Verbindungsstatus

**Ever Connected: Yes** - ~70 Server
**Ever Connected: No** - ~89 Server

## Nächste Schritte

1. ✅ Konsolidierte Tabelle erstellt
2. ✅ Alle Config-Dateien archiviert (außer mcp_config.json)
3. ⏳ Manuelle Spalten ausfüllen:
   - **Getestet**: Dokumentation welche Server getestet wurden
   - **Basic MCP**: Markierung ob Server MCP-Standard entspricht

## Generiert am

**Datum:** 2025-11-22
**Skript:** [consolidate_mcp_configs.py](../scripts/consolidate_mcp_configs.py)
