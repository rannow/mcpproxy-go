# MCP Server Fehler-Report

**Erstellt:** 2026-01-08
**Zweck:** Dokumentation fehlerhafter MCP Server für spätere Korrektur

---

## Zusammenfassung

| Kategorie | Anzahl | Prozent |
|-----------|--------|---------|
| ❌ Fehlgeschlagen | 3 | 7.0% |
| ⚠️ Teilweise funktional | 7 | 16.3% |
| **Gesamt zu korrigieren** | **10** | **23.3%** |

---

## ❌ Fehlgeschlagene Server (3)

### 1. exa
| Feld | Wert |
|------|------|
| **Status** | ❌ Fehlgeschlagen |
| **Protokoll** | stdio |
| **Tools** | 2 |
| **Fehler** | HTTP 401 Unauthorized |
| **Fehlermeldung** | `Invalid API Key. Your API key is invalid` |
| **Ursache** | EXA_API_KEY Umgebungsvariable fehlt oder ist ungültig |
| **Korrektur** | Gültigen API-Key von exa.ai besorgen und als Umgebungsvariable setzen |

### 2. postman
| Feld | Wert |
|------|------|
| **Status** | ❌ Fehlgeschlagen |
| **Protokoll** | stdio |
| **Tools** | 40 |
| **Fehler** | HTTP 401 Unauthorized |
| **Fehlermeldung** | `AuthenticationError: Invalid Credentials` |
| **Ursache** | Postman API-Key fehlt oder ist ungültig |
| **Korrektur** | Gültigen Postman API-Key konfigurieren |

### 3. reddit
| Feld | Wert |
|------|------|
| **Status** | ❌ Fehlgeschlagen |
| **Protokoll** | stdio |
| **Tools** | 6 |
| **Fehler** | HTTP 401 Unauthorized |
| **Fehlermeldung** | `Error searching subreddits: HTTP error 401` |
| **Ursache** | Reddit OAuth Credentials (Client ID, Client Secret) fehlen |
| **Korrektur** | Reddit App erstellen unter reddit.com/prefs/apps und Credentials konfigurieren |

---

## ⚠️ Teilweise funktionale Server (7)

### 1. gdrive
| Feld | Wert |
|------|------|
| **Status** | ⚠️ Teilweise |
| **Protokoll** | stdio |
| **Tools** | 4 |
| **Fehler** | Token abgelaufen |
| **Fehlermeldung** | `Token has been expired or revoked` |
| **Ursache** | Google OAuth Token ist abgelaufen |
| **Korrektur** | OAuth Flow erneut durchführen, Token erneuern |

### 2. mcp-obsidian
| Feld | Wert |
|------|------|
| **Status** | ⚠️ Teilweise |
| **Protokoll** | stdio |
| **Tools** | 2 |
| **Fehler** | Vault nicht gefunden |
| **Fehlermeldung** | `Could not find vault path` |
| **Ursache** | Obsidian Vault-Pfad in Konfiguration fehlt oder ist ungültig |
| **Korrektur** | Korrekten Vault-Pfad in Server-Konfiguration setzen |

### 3. ms365
| Feld | Wert |
|------|------|
| **Status** | ⚠️ Teilweise |
| **Protokoll** | stdio |
| **Tools** | 66 |
| **Fehler** | Token-Akquisition fehlgeschlagen |
| **Fehlermeldung** | `Failed to acquire token silently` |
| **Ursache** | Microsoft 365 OAuth Token fehlt oder ist ungültig |
| **Korrektur** | Microsoft 365 OAuth Flow erneut durchführen |

### 4. ms365-private
| Feld | Wert |
|------|------|
| **Status** | ⚠️ Teilweise |
| **Protokoll** | stdio |
| **Tools** | 66 |
| **Fehler** | Token-Akquisition fehlgeschlagen |
| **Fehlermeldung** | `Failed to acquire token silently` |
| **Ursache** | Microsoft 365 OAuth Token fehlt oder ist ungültig |
| **Korrektur** | Microsoft 365 OAuth Flow erneut durchführen |

### 5. neo4j
| Feld | Wert |
|------|------|
| **Status** | ⚠️ Teilweise |
| **Protokoll** | stdio |
| **Tools** | 4 |
| **Fehler** | Umgebungsvariable fehlt |
| **Fehlermeldung** | `Environment variable NEO4J_USERNAME is not set` |
| **Ursache** | Neo4j Credentials nicht konfiguriert |
| **Korrektur** | NEO4J_USERNAME, NEO4J_PASSWORD, NEO4J_URL Umgebungsvariablen setzen |

### 6. notebooklm
| Feld | Wert |
|------|------|
| **Status** | ⚠️ Teilweise |
| **Protokoll** | stdio |
| **Tools** | 31 |
| **Fehler** | Keine Notebooks |
| **Fehlermeldung** | `No notebooks found` |
| **Ursache** | Keine NotebookLM Notebooks vorhanden oder Zugriff fehlt |
| **Korrektur** | NotebookLM Notebooks erstellen oder API-Zugriff konfigurieren |

### 7. supabase
| Feld | Wert |
|------|------|
| **Status** | ⚠️ Teilweise |
| **Protokoll** | stdio |
| **Tools** | 29 |
| **Fehler** | Access Token fehlt |
| **Fehlermeldung** | `SUPABASE_ACCESS_TOKEN environment variable is not set` |
| **Ursache** | Supabase Credentials nicht konfiguriert |
| **Korrektur** | SUPABASE_ACCESS_TOKEN Umgebungsvariable setzen |

---

## Korrektur-Checkliste

- [ ] **exa**: API-Key von exa.ai besorgen
- [ ] **postman**: Postman API-Key konfigurieren
- [ ] **reddit**: Reddit App Credentials erstellen
- [ ] **gdrive**: Google OAuth Token erneuern
- [ ] **mcp-obsidian**: Vault-Pfad korrigieren
- [ ] **ms365**: Microsoft 365 OAuth Flow durchführen
- [ ] **ms365-private**: Microsoft 365 OAuth Flow durchführen
- [ ] **neo4j**: Neo4j Credentials setzen
- [ ] **notebooklm**: NotebookLM Zugriff prüfen
- [ ] **supabase**: Supabase Access Token setzen

---

## Hinweise

1. **API-Keys**: Die meisten Fehler sind auf fehlende oder ungültige API-Keys zurückzuführen
2. **OAuth Tokens**: Google und Microsoft OAuth Tokens müssen regelmäßig erneuert werden
3. **Umgebungsvariablen**: Prüfen Sie die Server-Konfiguration in MCPProxy auf fehlende Variablen
4. **Test nach Korrektur**: Nach Korrektur Server mit den Test-Anweisungen aus `docs/mcp_server_test_instructions.md` erneut testen
