# Fehlende Informationen für MCP Server

Diese Server benötigen zusätzliche Konfiguration, um korrekt zu funktionieren:

## 🔐 Server mit fehlenden API Keys / Umgebungsvariablen

### 1. Bright Data Server
**Status**: Verbindet sich, trennt dann sofort
**Benötigt**:
- API Key/Token für Bright Data Service
- Account-Zugangsdaten

**Fragen**:
- Haben Sie einen Bright Data Account? JA
- Falls ja, wo finde ich den API Key? der API Key ist unter 
- Welche Umgebungsvariable soll gesetzt werden? (z.B. `BRIGHT_DATA_API_KEY`)

---

### 2. Discord MCP Server (`mcp-discord`)
**Status**: Verbindet sich, trennt dann sofort
**Benötigt**:
- Discord Bot Token
- Guild/Server ID (optional)

**Fragen**:
- Haben Sie einen Discord Bot erstellt?
- Falls ja, wie lautet der Bot Token?
- Welche Discord Server sollen überwacht werden?

**Setup-Anleitung**: https://discord.com/developers/applications

---

### 3. AWS Server (diverse)
**Status**: Timeout beim Starten
**Betroffene Server**:
- `awslabs.core-mcp-server`
- `awslabs.eks-mcp-server`
- `awslabs.iam-mcp-server`
- `awslabs.terraform-mcp-server`
- `awslabs.git-repo-research-mcp-server`
- `awslabs.aws-diagram-mcp-server`
- `awslabs.nova-canvas-mcp-server`

**Benötigt**:
- AWS Access Key ID
- AWS Secret Access Key
- AWS Region (z.B. `eu-central-1`, `us-east-1`)
- Optional: AWS Session Token (für temporäre Credentials)

**Fragen**:
- Haben Sie AWS CLI bereits konfiguriert?
- Falls ja, welches AWS Profil soll verwendet werden?
- Falls nein, benötige ich:
  - Access Key ID
  - Secret Access Key
  - Bevorzugte Region

**Prüfen Sie**: `cat ~/.aws/credentials` und `cat ~/.aws/config`

---

### 4. Firecrawl Server (`mcp-server-firecrawl`)
**Status**: Timeout beim Starten
**Benötigt**:
- Firecrawl API Key

**Fragen**:
- Haben Sie einen Firecrawl Account?
- Falls ja, wie lautet der API Key?
- API Key Location: https://www.firecrawl.dev/app/api-keys

---

### 5. Docker MCP Server (`docker-mcp`)
**Status**: Verbindet sich, trennt dann sofort
**Benötigt**:
- Docker muss laufen
- Docker Socket Zugriff

**Fragen**:
- Läuft Docker Desktop?
- Prüfen Sie: `docker ps` funktioniert?
- Falls nicht: Docker starten oder installieren

---

### 6. WCGW Server
**Status**: Timeout beim Starten
**Benötigt**:
- Weitere Konfiguration (Server-spezifische Docs fehlen)

**Fragen**:
- Verwendungszweck dieses Servers?
- Benötigt dieser Server spezielle Konfiguration?

---

## 📋 Nächste Schritte

### Option A: Automatische Konfiguration
Ich kann ein interaktives Setup-Skript erstellen, das Sie durch die Konfiguration führt:

```bash
# Skript würde folgendes tun:
1. Nach API Keys fragen
2. .env Datei generieren
3. Umgebungsvariablen in mcp_config.json eintragen
4. Server-Konfigurationsdateien erstellen
```

Soll ich dieses Skript erstellen?

### Option B: Manuelle Konfiguration
Sie können die Informationen hier eintragen, und ich aktualisiere die Config entsprechend.

### Option C: Server deaktivieren
Nicht benötigte Server können auf `startup_mode: "disabled"` gesetzt werden:

```bash
# Agent-Befehl zum Deaktivieren spezifischer Server
python3 .claude/agents/mcp/manage-servers.py --disable bright-data discord mcp-discord
```

---

## 🔧 Empfohlene Maßnahmen

### Sofort (High Priority):
1. **AWS Credentials**: Falls AWS genutzt werden soll
2. **Docker**: Falls Docker-Integration benötigt wird

### Optional (Low Priority):
1. **Discord Bot**: Nur falls Discord-Integration gewünscht
2. **Bright Data**: Nur falls Web Scraping benötigt wird
3. **Firecrawl**: Nur falls Web Crawling benötigt wird

---

## ✅ Bereits Konfiguriert

Diese Server laufen bereits erfolgreich (~59 von 71):
- k8s-mcp-server (mit Docker)
- aws-mcp-server (mit Docker)
- Und viele weitere...

---

## Fragen an Sie:

1. **AWS**: Nutzen Sie AWS? Falls ja, ist AWS CLI konfiguriert?
2. **Docker**: Soll Docker-Integration aktiviert bleiben?
3. **Discord**: Benötigen Sie Discord-Integration?
4. **Bright Data / Firecrawl**: Nutzen Sie Web Scraping?
5. **Nicht benötigte Server**: Welche Server sind nicht relevant und können deaktiviert werden?

Bitte beantworten Sie diese Fragen, damit ich die Konfiguration vervollständigen kann.
