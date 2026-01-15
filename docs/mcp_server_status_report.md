# MCP Server Status Report

**Generated:** 2026-01-07 (Automated Check)
**mcpproxy Version:** Running on port 8080

## Summary

| Metric | Value |
|--------|-------|
| Total Servers | 43 |
| Connected | 43 |
| Connecting | 0 |
| Errors | 0 |
| Timeout | false |

**Result: ✅ Alle Server sind verbunden und funktionieren korrekt.**

## Server Status Table

| # | Server Name | Status | Connected | Startup Mode | Tools | Connection Time |
|---|-------------|--------|-----------|--------------|-------|-----------------|
| 1 | archon | Ready | true | active | 16 | 14ms |
| 2 | calculator | Ready | true | active | 1 | 1.5s |
| 3 | claude-code-mcp | Ready | true | active | 1 | 7.0s |
| 4 | confluence | Ready | true | active | 5 | 14.8s |
| 5 | context7 | Ready | true | active | 2 | 13.1s |
| 6 | desktop-commander | Ready | true | active | 26 | 13.8s |
| 7 | docker | Ready | true | active | 1 | 13.6s |
| 8 | exa | Ready | true | active | 2 | 20.9s |
| 9 | excel | Ready | true | active | 6 | 6.8s |
| 10 | excel-advanced | Ready | true | active | 25 | 8.7s |
| 11 | gdrive | Ready | true | active | 4 | 25.1s |
| 12 | github | Ready | true | active | 17 | 5.8s |
| 13 | gitlab | Ready | true | active | 9 | 6.6s |
| 14 | gmail | Ready | true | active | 15 | 7.3s |
| 15 | google-calendar | Ready | true | active | 6 | 12.1s |
| 16 | grep-mcp | Ready | true | active | 2 | 8.9s |
| 17 | imap-mobilis | Ready | true | active | 12 | 20.1s |
| 18 | imap-smsprofi | Ready | true | active | 12 | 18.3s |
| 19 | markdownify | Ready | true | active | 10 | 22.3s |
| 20 | mcp-obsidian | Ready | true | active | 2 | 20.7s |
| 21 | mcp-youtube | Ready | true | active | 3 | 14.0s |
| 22 | medium | Ready | true | active | 5 | 3.4s |
| 23 | mem0 | Ready | true | active | 2 | 10.5s |
| 24 | ms365 | Ready | true | active | 66 | 6.2s |
| 25 | ms365-private | Ready | true | active | 66 | 14.6s |
| 26 | multi-ai-mcp | Ready | true | active | 17 | 3.1s |
| 27 | mysql | Ready | true | active | 92 | 17.4s |
| 28 | mysql-prod | Ready | true | active | 85 | 7.5s |
| 29 | n8n | Ready | true | active | 7 | 14.7s |
| 30 | neo4j | Ready | true | active | 4 | 9.3s |
| 31 | notebooklm | Ready | true | active | 31 | 4.9s |
| 32 | pal-mcp | Ready | true | active | 18 | 6.4s |
| 33 | playwright | Ready | true | active | 22 | 10.3s |
| 34 | postgres | Ready | true | active | 1 | 7.5s |
| 35 | postman | Ready | true | active | 40 | 21.9s |
| 36 | qdrant | Ready | true | active | 4 | 12.1s |
| 37 | reddit | Ready | true | active | 6 | 11.1s |
| 38 | supabase | Ready | true | active | 29 | 8.6s |
| 39 | targetprocess | Ready | true | active | 19 | 30.4s |
| 40 | taskmaster | Ready | true | active | 7 | 25.9s |
| 41 | test-filesystem-server | Ready | true | active | 11 | 5.2s |
| 42 | zen-mcp-server | Ready | true | active | 16 | 13.0s |
| 43 | zep-graphiti | Ready | true | active | 9 | 9ms |

## Statistik nach Tool-Anzahl

### Top 10 Server nach Tool-Anzahl
| Server | Tools |
|--------|-------|
| mysql | 92 |
| mysql-prod | 85 |
| ms365 | 66 |
| ms365-private | 66 |
| postman | 40 |
| notebooklm | 31 |
| supabase | 29 |
| desktop-commander | 26 |
| excel-advanced | 25 |
| playwright | 22 |

### Gesamte Tool-Anzahl: 689 Tools

## Connection Time Analyse

### Schnellste Server (< 1s)
| Server | Connection Time |
|--------|-----------------|
| zep-graphiti | 9ms |
| archon | 14ms |

### Langsamste Server (> 20s)
| Server | Connection Time |
|--------|-----------------|
| targetprocess | 30.4s |
| taskmaster | 25.9s |
| gdrive | 25.1s |
| markdownify | 22.3s |
| postman | 21.9s |
| exa | 20.9s |
| mcp-obsidian | 20.7s |
| imap-mobilis | 20.1s |

## Probleme

**Keine Probleme gefunden.**

Alle 43 Server sind:
- ✅ Verbunden (connected: true)
- ✅ Status: Ready
- ✅ Startup Mode: active
- ✅ Keine Fehler gemeldet

## Empfehlungen

1. **Langsame Server beobachten**: Einige Server haben lange Connection-Zeiten (>20s). Dies könnte bei Neustarts zu Verzögerungen führen.

2. **Tool-Count prüfen**: Server wie `mysql` (92 Tools) und `ms365` (66 Tools) haben sehr viele Tools. Bei Bedarf könnten diese in separate Server aufgeteilt werden.
