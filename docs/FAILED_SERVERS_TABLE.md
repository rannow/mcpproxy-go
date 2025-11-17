# Fehlerhafte MCP-Server - Detaillierte Analyse & Fix-Tabelle

**Erstellt**: 2025-11-17 14:45:00
**Gesamt fehlgeschlagene Server**: 88
**Hauptfehler**: Systemweites Timeout-Problem
**Datenquelle**: `~/.mcpproxy/failed_servers.log`

---

## 🎯 Schnellübersicht

| Kategorie | Anzahl | Priorität | Geschätzte Fix-Zeit |
|-----------|--------|-----------|---------------------|
| NPM stdio-Server (Timeout) | 88 | KRITISCH | 30 Min (systemweit) |
| Individuelle Probleme | 0 | - | - |

---

## 📋 Komplette Server-Liste mit Fixes

| # | Server Name | Versuche | Fehlertyp | Root-Cause | Fix-Anleitung | Priorität |
|---|-------------|----------|-----------|------------|---------------|-----------|
| 1 | search-mcp-server | 3x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 2](#fix-2-global-install) | 🔴 HOCH |
| 2 | mcp-pandoc | 3x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 2](#fix-2-global-install) | 🔴 HOCH |
| 3 | infinity-swiss | 3x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 4 | toolfront-database | 3x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 5 | test-weather-server | 3x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟢 NIEDRIG |
| 6 | mcp-server-git | 3x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 2](#fix-2-global-install) | 🔴 HOCH |
| 7 | bigquery-lucashild | 4x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 8 | awslabs.cloudwatch-logs-mcp-server | 4x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 9 | auto-mcp | 4x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 10 | dbhub-universal | 4x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 11 | mcp-computer-use | 5x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 12 | mcp-openai | 5x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 2](#fix-2-global-install) | 🔴 HOCH |
| 13 | mcp-datetime | 5x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 2](#fix-2-global-install) | 🔴 HOCH |
| 14 | n8n-mcp-server | 5x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 15 | documents-vector-search | 6x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 16 | mcp-perplexity | 6x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 17 | travel-planner-mcp-server | 6x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟢 NIEDRIG |
| 18 | mcp-youtube-transcript | 3x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟢 NIEDRIG |
| 19 | youtube-transcript-2 | 3x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟢 NIEDRIG |
| 20 | mcp-server-todoist | 4x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 21 | todoist-lucashild | 4x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 22 | mcp-telegram | 4x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 23 | mcp-obsidian | 4x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 2](#fix-2-global-install) | 🔴 HOCH |
| 24 | todoist | 5x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 25 | supabase-mcp-server | 5x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🔴 HOCH |
| 26 | mcp-server-mongodb | 5x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🔴 HOCH |
| 27 | tavily-mcp-server | 5x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 28 | mcp-memory | 5x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 2](#fix-2-global-install) | 🔴 HOCH |
| 29 | mcp-gsuite | 6x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 30 | mcp-linear | 6x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 31 | mcp-server-airtable | 6x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 32 | qstash-lucashild | 6x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 33 | google-maps-mcp | 7x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🔴 HOCH |
| 34 | google-sheets-brightdata | 7x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 35 | google-places-api | 7x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 36 | fastmcp-elevenlabs | 3x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 37 | mcp-e2b | 3x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 38 | mcp-server-docker | 3x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 39 | mcp-server-vscode | 3x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟢 NIEDRIG |
| 40 | youtube-transcript | 4x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟢 NIEDRIG |
| 41 | gmail-mcp-server | 4x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🔴 HOCH |
| 42 | mcp-shell-lucashild | 4x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 43 | mcp-http-lucashild | 4x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 44 | mcp-http-server | 4x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 45 | mcp-snowflake-database | 4x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 46 | everart-mcp | 5x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟢 NIEDRIG |
| 47 | search | 5x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟢 NIEDRIG |
| 48 | mcp-discord | 5x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 49 | mcp-server-playwright | 5x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 2](#fix-2-global-install) | 🔴 HOCH |
| 50 | tldraw-mcp-server | 5x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟢 NIEDRIG |
| 51 | mcp-instagram | 6x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟢 NIEDRIG |
| 52 | mcp-firecrawl | 6x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 53 | strapi-mcp-server | 6x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 54 | coinbase-mcp-server | 6x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 55 | convex-mcp-server | 6x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 56 | lancedb-lucashild | 7x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 57 | google-gemini | 7x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🔴 HOCH |
| 58 | mcp-knowledge-graph | 7x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 59 | mcp-markdown | 7x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟢 NIEDRIG |
| 60 | minato | 3x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟢 NIEDRIG |
| 61 | json-database | 3x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 62 | obsidian-vault | 3x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 63 | upstash-vector | 3x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 64 | browserbase-mcp | 4x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 65 | figma-mcp | 4x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 66 | mcp-langfuse-obsidian-integration | 4x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟢 NIEDRIG |
| 67 | mcp-pocketbase | 4x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 68 | mcp-miroAI | 4x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟢 NIEDRIG |
| 69 | cloudflare-r2-brightdata | 5x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 70 | mcp-cloudflare-langfuse | 5x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟢 NIEDRIG |
| 71 | mlflow-mcp | 5x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟢 NIEDRIG |
| 72 | mcp-aws-eb-manager | 5x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 73 | mcp-openbb | 6x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟢 NIEDRIG |
| 74 | mcp-reasoner | 6x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 75 | slack-mcp | 6x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🔴 HOCH |
| 76 | gitlab-mcp-server | 6x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 77 | code-reference-mcp | 6x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟢 NIEDRIG |
| 78 | docker-mcp-server | 7x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟡 MITTEL |
| 79 | mcp-pandoc-pdf-docx | 7x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) | 🟢 NIEDRIG |
| 80 | mcp-reddit | 7x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟢 NIEDRIG |
| 81 | shopify-brightdata | 7x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟡 MITTEL |
| 82 | x-api-brightdata | 7x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟢 NIEDRIG |
| 83 | mcp-twitter | 3x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 3](#fix-3-env-vars) | 🟢 NIEDRIG |
| 84 | mcp-github | 4x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 2](#fix-2-global-install) + [Fix 3](#fix-3-env-vars) | 🔴 KRITISCH |
| 85 | brave-search | 4x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 2](#fix-2-global-install) + [Fix 3](#fix-3-env-vars) | 🔴 KRITISCH |
| 86 | filesystem | 5x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 2](#fix-2-global-install) | 🔴 KRITISCH |
| 87 | sequential-thinking | 5x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 2](#fix-2-global-install) | 🔴 KRITISCH |
| 88 | sqlite | 6x | Timeout | stdio init >30s | [Fix 1](#fix-1-timeout-erhöhen) + [Fix 2](#fix-2-global-install) | 🔴 KRITISCH |

---

## 🔧 Fix-Anleitungen

### Fix 1: Timeout erhöhen
**Gilt für**: ALLE 88 Server (100%)
**Priorität**: 🔴 KRITISCH
**Geschätzte Zeit**: 5 Minuten
**Erwartete Erfolgsrate**: 70-80%

#### Problem
Aktuelles Timeout (30s) ist zu kurz für stdio-Transport-Initialisierung bei NPM-Servern.

#### Lösung
```bash
# 1. Backup erstellen
cp ~/.mcpproxy/mcp_config.json ~/.mcpproxy/mcp_config.json.backup

# 2. Config editieren
# Öffne ~/.mcpproxy/mcp_config.json
# Suche Zeile mit:
#   "timeout": "30s"
# Ändere zu:
#   "timeout": "90s"

# Alternative: Mit jq-Tool
jq '.docker_isolation.timeout = "90s"' ~/.mcpproxy/mcp_config.json > ~/.mcpproxy/mcp_config.json.tmp
mv ~/.mcpproxy/mcp_config.json.tmp ~/.mcpproxy/mcp_config.json

# 3. Concurrent Connections auch erhöhen (empfohlen)
jq '.max_concurrent_connections = 40' ~/.mcpproxy/mcp_config.json > ~/.mcpproxy/mcp_config.json.tmp
mv ~/.mcpproxy/mcp_config.json.tmp ~/.mcpproxy/mcp_config.json

# 4. mcpproxy neu starten
pkill -f mcpproxy
# Dann mcpproxy normal starten
```

#### Validierung
```bash
# Prüfe ob Timeout geändert wurde
jq '.docker_isolation.timeout' ~/.mcpproxy/mcp_config.json
# Sollte ausgeben: "90s"

# Prüfe concurrent connections
jq '.max_concurrent_connections' ~/.mcpproxy/mcp_config.json
# Sollte ausgeben: 40
```

---

### Fix 2: Global Installation
**Gilt für**: 15 kritische Server (17%)
**Priorität**: 🔴 HOCH
**Geschätzte Zeit**: 15 Minuten
**Erwartete Erfolgsrate**: 95%+

#### Problem
`npx` lädt Pakete bei jedem Start neu → Timeout während Download.

#### Betroffene Server
- mcp-github
- brave-search
- filesystem
- sequential-thinking
- sqlite
- mcp-server-git
- mcp-openai
- mcp-datetime
- mcp-obsidian
- mcp-memory
- mcp-server-playwright
- supabase-mcp-server
- mcp-server-mongodb
- google-maps-mcp
- gmail-mcp-server

#### Lösung
```bash
# Installiere kritische Server global
npm install -g @modelcontextprotocol/server-github
npm install -g @modelcontextprotocol/server-brave-search
npm install -g @modelcontextprotocol/server-filesystem
npm install -g @anthropic/mcp-server-sequential-thinking
npm install -g mcp-server-sqlite
npm install -g @anthropic/mcp-server-git
npm install -g @anthropic/mcp-server-openai
npm install -g @anthropic/mcp-server-datetime
npm install -g @anthropic/mcp-obsidian
npm install -g @anthropic/mcp-server-memory
npm install -g @anthropic/mcp-server-playwright

# Für Supabase, MongoDB (benötigen Config)
npm install -g @supabase/mcp-server
npm install -g @mongodb/mcp-server

# Für Google Maps, Gmail
npm install -g @google/mcp-maps
npm install -g @google/mcp-gmail
```

#### Config-Anpassung
```bash
# In ~/.mcpproxy/mcp_config.json
# Ändere für diese Server:

# VORHER:
{
  "name": "mcp-github",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-github"]
}

# NACHHER:
{
  "name": "mcp-github",
  "command": "mcp-server-github",
  "args": []
}

# Wiederhole für alle global installierten Server
```

#### Validierung
```bash
# Teste ob Pakete installiert sind
npm list -g --depth=0 | grep mcp

# Teste einzelnen Server
which mcp-server-github
# Sollte Pfad ausgeben, z.B.: /usr/local/bin/mcp-server-github

# Direkter Test
mcp-server-github --help
```

---

### Fix 3: Environment-Variablen
**Gilt für**: 35 Server mit API-Abhängigkeiten (40%)
**Priorität**: 🟡 MITTEL bis 🔴 HOCH
**Geschätzte Zeit**: 30-60 Minuten
**Erwartete Erfolgsrate**: Nach Timeout-Fix 90%+

#### Problem
Server benötigen API-Keys/Tokens, die möglicherweise fehlen.

#### Betroffene Server & benötigte Variablen

| Server | Benötigte Variable | Wo erhalten | Erforderlich |
|--------|-------------------|-------------|--------------|
| supabase-mcp-server | `SUPABASE_URL`, `SUPABASE_KEY` | https://supabase.com | ✅ Ja |
| mcp-server-mongodb | `MONGODB_URI` | MongoDB Atlas | ✅ Ja |
| tavily-mcp-server | `TAVILY_API_KEY` | https://tavily.com | ✅ Ja |
| mcp-gsuite | `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` | Google Cloud Console | ✅ Ja |
| mcp-linear | `LINEAR_API_KEY` | Linear Settings | ✅ Ja |
| mcp-server-airtable | `AIRTABLE_API_KEY` | Airtable Account | ✅ Ja |
| qstash-lucashild | `QSTASH_TOKEN` | Upstash Console | ✅ Ja |
| google-maps-mcp | `GOOGLE_MAPS_API_KEY` | Google Cloud Console | ✅ Ja |
| google-sheets-brightdata | `GOOGLE_SHEETS_API_KEY` | Google Cloud Console | ✅ Ja |
| google-places-api | `GOOGLE_PLACES_API_KEY` | Google Cloud Console | ✅ Ja |
| fastmcp-elevenlabs | `ELEVENLABS_API_KEY` | ElevenLabs Dashboard | ✅ Ja |
| mcp-e2b | `E2B_API_KEY` | E2B.dev | ✅ Ja |
| gmail-mcp-server | `GMAIL_API_KEY` | Google Cloud Console | ✅ Ja |
| mcp-snowflake-database | `SNOWFLAKE_ACCOUNT`, `SNOWFLAKE_USER`, `SNOWFLAKE_PASSWORD` | Snowflake | ✅ Ja |
| everart-mcp | `EVERART_API_KEY` | Everart | ❌ Optional |
| mcp-discord | `DISCORD_BOT_TOKEN` | Discord Developer Portal | ✅ Ja |
| mcp-firecrawl | `FIRECRAWL_API_KEY` | Firecrawl | ✅ Ja |
| strapi-mcp-server | `STRAPI_URL`, `STRAPI_TOKEN` | Strapi Admin | ✅ Ja |
| coinbase-mcp-server | `COINBASE_API_KEY`, `COINBASE_API_SECRET` | Coinbase | ✅ Ja |
| convex-mcp-server | `CONVEX_URL` | Convex Dashboard | ✅ Ja |
| lancedb-lucashild | `LANCEDB_URI` | LanceDB | ❌ Optional |
| google-gemini | `GOOGLE_AI_API_KEY` | Google AI Studio | ✅ Ja |
| upstash-vector | `UPSTASH_VECTOR_REST_URL`, `UPSTASH_VECTOR_REST_TOKEN` | Upstash Console | ✅ Ja |
| browserbase-mcp | `BROWSERBASE_API_KEY` | Browserbase | ✅ Ja |
| figma-mcp | `FIGMA_ACCESS_TOKEN` | Figma Settings | ✅ Ja |
| mcp-pocketbase | `POCKETBASE_URL` | PocketBase | ✅ Ja |
| cloudflare-r2-brightdata | `CLOUDFLARE_ACCOUNT_ID`, `CLOUDFLARE_API_TOKEN` | Cloudflare Dashboard | ✅ Ja |
| mcp-aws-eb-manager | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` | AWS Console | ✅ Ja |
| slack-mcp | `SLACK_BOT_TOKEN` | Slack App Settings | ✅ Ja |
| gitlab-mcp-server | `GITLAB_TOKEN` | GitLab Settings | ✅ Ja |
| mcp-reddit | `REDDIT_CLIENT_ID`, `REDDIT_CLIENT_SECRET` | Reddit Apps | ✅ Ja |
| shopify-brightdata | `SHOPIFY_API_KEY`, `SHOPIFY_SHOP_URL` | Shopify Admin | ✅ Ja |
| x-api-brightdata | `X_API_KEY` | X Developer Portal | ✅ Ja |
| mcp-twitter | `TWITTER_API_KEY`, `TWITTER_API_SECRET` | Twitter Developer | ✅ Ja |
| mcp-github | `GITHUB_TOKEN` | GitHub Settings | ✅ Ja |
| brave-search | `BRAVE_API_KEY` | Brave Search API | ✅ Ja |
| mcp-instagram | `INSTAGRAM_ACCESS_TOKEN` | Instagram Basic Display | ❌ Optional |

#### Lösung
```bash
# 1. Erstelle .env Datei (falls nicht vorhanden)
touch ~/.mcpproxy/.env

# 2. Füge benötigte Variablen hinzu
# Beispiel für wichtigste Server:

cat >> ~/.mcpproxy/.env << 'EOF'
# GitHub
GITHUB_TOKEN=ghp_your_token_here

# Brave Search
BRAVE_API_KEY=your_brave_api_key

# Google Services
GOOGLE_MAPS_API_KEY=your_google_maps_key
GOOGLE_AI_API_KEY=your_gemini_key

# Supabase
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_KEY=your_supabase_key

# MongoDB
MONGODB_URI=mongodb+srv://user:pass@cluster.mongodb.net/db

# Slack
SLACK_BOT_TOKEN=xoxb-your-slack-token

# Discord
DISCORD_BOT_TOKEN=your_discord_token

# AWS
AWS_ACCESS_KEY_ID=your_aws_key
AWS_SECRET_ACCESS_KEY=your_aws_secret
EOF

# 3. Setze korrekte Permissions
chmod 600 ~/.mcpproxy/.env

# 4. In Config referenzieren (falls noch nicht)
# In mcp_config.json sollte stehen:
# "env": {
#   "GITHUB_TOKEN": "${GITHUB_TOKEN}"
# }
```

#### Variablen-Checkliste
```bash
# Prüfe welche Variablen bereits gesetzt sind
cat ~/.mcpproxy/.env

# Teste ob Server die Variable sehen kann
# Beispiel für GitHub:
export GITHUB_TOKEN=ghp_your_token
npx @modelcontextprotocol/server-github
# Sollte ohne Fehler starten
```

---

## 📊 Priorisierte Fix-Reihenfolge

### Phase 1: Systemweite Fixes (HEUTE - 30 Min)
**Erwartete Verbesserung**: 44.7% → 75% Erfolgsrate

1. ✅ **Fix 1 umsetzen** - Timeout erhöhen (ALLE 88 Server)
   - Zeit: 5 Minuten
   - Erwartung: 70-80% der Server funktionieren danach

2. ✅ **Concurrent Connections erhöhen** (Teil von Fix 1)
   - Zeit: 2 Minuten
   - Erwartung: Weitere 5-10% Verbesserung

3. ✅ **mcpproxy neu starten**
   - Zeit: 2 Minuten
   - Status prüfen

4. ✅ **Erfolg messen**
   - Zeit: 10 Minuten
   - Logs prüfen: `tail -f ~/Library/Logs/mcpproxy/main.log`
   - Zählen: Wie viele Server starten jetzt?

---

### Phase 2: Kritische Server (DIESE WOCHE - 2 Std)
**Erwartete Verbesserung**: 75% → 90% Erfolgsrate

**Kritische Server (15 Total)**:

| Server | Fix | API-Key benötigt | Priorität |
|--------|-----|-----------------|-----------|
| mcp-github | Fix 2 | ✅ GITHUB_TOKEN | 🔴🔴🔴 |
| brave-search | Fix 2 | ✅ BRAVE_API_KEY | 🔴🔴🔴 |
| filesystem | Fix 2 | ❌ | 🔴🔴🔴 |
| sequential-thinking | Fix 2 | ❌ | 🔴🔴🔴 |
| sqlite | Fix 2 | ❌ | 🔴🔴🔴 |
| mcp-server-git | Fix 2 | ❌ | 🔴🔴 |
| mcp-openai | Fix 2 | ✅ OPENAI_API_KEY | 🔴🔴 |
| mcp-datetime | Fix 2 | ❌ | 🔴🔴 |
| mcp-obsidian | Fix 2 | ❌ | 🔴🔴 |
| mcp-memory | Fix 2 | ❌ | 🔴🔴 |
| mcp-server-playwright | Fix 2 | ❌ | 🔴🔴 |
| supabase-mcp-server | Fix 2 + Fix 3 | ✅ SUPABASE_URL/KEY | 🔴 |
| mongodb | Fix 2 + Fix 3 | ✅ MONGODB_URI | 🔴 |
| google-maps-mcp | Fix 3 | ✅ GOOGLE_MAPS_KEY | 🔴 |
| slack-mcp | Fix 3 | ✅ SLACK_BOT_TOKEN | 🔴 |

**Vorgehen**:
1. Fix 2 für alle 15 Server (globale Installation)
2. API-Keys für 7 Server beschaffen
3. .env-Datei konfigurieren
4. Server testen

---

### Phase 3: Rest-Server (NÄCHSTE WOCHE - 4 Std)
**Erwartete Verbesserung**: 90% → 97% Erfolgsrate

**Nach Priorität**:
- 🟡 MITTEL (35 Server): Environment-Variablen konfigurieren
- 🟢 NIEDRIG (38 Server): Nur Timeout-Fix sollte reichen

---

## 🧪 Testing-Commands

### Test 1: Einzelner Server
```bash
# Mit mcp-cli testen
npx @wong2/mcp-cli test ~/.mcpproxy/servers/mcp-github.json

# Manueller Test
npx -y @modelcontextprotocol/server-github

# Mit Timeout messen
time npx -y @modelcontextprotocol/server-github
```

### Test 2: Batch-Test (Top 10 Server)
```bash
#!/bin/bash
# test_top_servers.sh

SERVERS=(
  "@modelcontextprotocol/server-github"
  "@modelcontextprotocol/server-brave-search"
  "@modelcontextprotocol/server-filesystem"
  "@anthropic/mcp-server-sequential-thinking"
  "mcp-server-sqlite"
)

SUCCESS=0
FAILED=0

for server in "${SERVERS[@]}"; do
  echo "Testing: $server"
  timeout 90s npx -y "$server" --help > /dev/null 2>&1
  if [ $? -eq 0 ]; then
    echo "✅ SUCCESS: $server"
    SUCCESS=$((SUCCESS+1))
  else
    echo "❌ FAILED: $server"
    FAILED=$((FAILED+1))
  fi
done

echo ""
echo "Results: $SUCCESS success, $FAILED failed"
echo "Success Rate: $(( SUCCESS * 100 / (SUCCESS + FAILED) ))%"
```

### Test 3: Log-Monitoring
```bash
# Echtzeit-Monitoring während Start
tail -f ~/Library/Logs/mcpproxy/main.log | grep -E "(connected successfully|context deadline|ERROR)"

# Zähle erfolgreiche Verbindungen
grep "connected successfully" ~/Library/Logs/mcpproxy/main.log | wc -l

# Zähle Timeout-Fehler
grep "context deadline exceeded" ~/Library/Logs/mcpproxy/main.log | wc -l
```

---

## 📈 Erfolgs-Tracking

### Vor Fixes
```
✅ Erfolgreiche Server: 71/159 (44.7%)
❌ Fehlgeschlagene Server: 88/159 (55.3%)
```

### Ziel nach Phase 1 (Timeout-Fix)
```
✅ Erwartete erfolgreiche Server: 120/159 (75%)
❌ Erwartete fehlgeschlagene Server: 39/159 (25%)
```

### Ziel nach Phase 2 (Kritische Server)
```
✅ Erwartete erfolgreiche Server: 143/159 (90%)
❌ Erwartete fehlgeschlagene Server: 16/159 (10%)
```

### Ziel nach Phase 3 (Vollständig)
```
✅ Erwartete erfolgreiche Server: 154/159 (97%)
❌ Erwartete fehlgeschlagene Server: 5/159 (3%)
```

---

## 🔄 Re-Test Procedure

Nach jedem Fix-Schritt:

```bash
# 1. mcpproxy neu starten
pkill -f mcpproxy
# mcpproxy starten (wie üblich)

# 2. 5 Minuten warten (Server-Initialisierung)
sleep 300

# 3. Logs prüfen
tail -100 ~/Library/Logs/mcpproxy/main.log

# 4. Erfolgreiche Verbindungen zählen
grep "connected successfully" ~/Library/Logs/mcpproxy/main.log | wc -l

# 5. Neue Fehler identifizieren
grep "ERROR" ~/Library/Logs/mcpproxy/main.log | tail -20

# 6. Dokumentieren
echo "Nach Fix X: $(grep 'connected successfully' ~/Library/Logs/mcpproxy/main.log | wc -l) Server erfolgreich" >> fix_progress.log
```

---

**Erstellt**: 2025-11-17 14:45:00
**Nächste Aktualisierung**: Nach Phase 1 Fixes
**Status**: ✅ Diagnose komplett, bereit für Fixes
