# Resource Leak Analysis - mcpproxy-go

**Datum**: 2025-10-17
**Status**: Kritisch
**Priorität**: Hoch

## Zusammenfassung

Bei der Analyse wurden **3 Hauptkategorien von Resource Leaks** identifiziert:

1. **Docker Container Leaks** (21 verwaiste Container)
2. **Prozess-Leaks** (24 mcpproxy-Instanzen)
3. **Goroutine Leaks** (15+ Goroutinen pro Server)

## 1. Docker Container Leaks

### Problem
Docker-Container werden beim Shutdown nicht korrekt bereinigt:

```
✓ Gefunden: 21 verwaiste Docker-Container
  - aws-mcp-server: 11 Container
  - k8s-mcp-server: 10 Container

✓ Laufzeit: Einige seit >12 Stunden (seit 10:06am)
✓ cidfiles: 10 verwaiste Dateien in /var/folders/.../T/
```

### Ursache (connection.go:364-405)

```go
cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cleanupCancel()
```

**Fehler**:
- 10-Sekunden-Timeout ist zu kurz für Docker API-Calls
- "context deadline exceeded" Fehler werden ignoriert
- Container bleiben orphaned, wenn API-Call feailed

### Log-Evidenz
```
Failed to list Docker containers for cleanup: context deadline exceeded
```

## 2. Prozess-Leaks

### Problem
Mehrere mcpproxy-Instanzen laufen gleichzeitig:

```
✓ Gefunden: 24 parallel laufende mcpproxy-Prozesse
✓ Erwartung: Nur 1 Prozess sollte laufen
```

### Ursache
- Tests/Restarts starten neue Instanzen ohne alte zu beenden
- Kein PID-File oder Single-Instance-Lock
- Docker Backend läuft doppelt (PIDs: 12541, 64110)

## 3. Goroutine Leaks

### Problem - Critical!
**15 Goroutinen pro Server** werden niemals gestoppt:

### Betroffene Dateien

#### managers.go (10 Goroutinen)
```go
// Zeile 589: unquarantine handler
go func(name string, item *systray.MenuItem) {
    for range item.ClickedCh {
        go m.onServerAction(name, "unquarantine")
    }
}(serverName, menuItem)

// Zeile 1134: OAuth login handler
// Zeile 1153: quarantine handler
// Zeile 1169: configure handler
// Zeile 1184: restart handler
// Zeile 1199: open_log handler
// Zeile 1219: open_repo handler
// Zeile 1235: enable/disable handler
// Zeile 1294: remove from group handler
// Zeile 1316: assign to group handler
```

#### tray.go (5 Goroutinen)
```go
// Zeile 2068: rename group handler
// Zeile 2094: change group color handler
// Zeile 2109: done button handler
// Zeile 2117: delete group handler
// Zeile 3074: remove server from group handler
```

### Root Cause
```go
// cleanupServerActionItems versteckt nur die Items:
func (m *MenuManager) cleanupServerActionItems(serverName string) {
    if actionItem, ok := m.serverActionItems[serverName]; ok {
        actionItem.Hide()  // ❌ Goroutine läuft weiter!
        delete(m.serverActionItems, serverName)
    }
    // ... mehr Hide() calls, aber KEINE Goroutine-Cleanup
}
```

**Problem**: `Hide()` stoppt die Goroutine NICHT!
- Die `for range ClickedCh` Loop blockiert weiter
- Goroutine wartet auf Events, die nie kommen
- Memory leak wächst mit jedem Server-Refresh

## Lösungsvorschläge

### 1. Docker Cleanup Fix (Priorität: Kritisch)

```go
// Option A: Timeout erhöhen
cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cleanupCancel()

// Option B: Retry-Logik mit Exponential Backoff
func (c *Connection) cleanupDockerWithRetry(maxRetries int) error {
    for i := 0; i < maxRetries; i++ {
        timeout := time.Duration(10*(i+1)) * time.Second
        ctx, cancel := context.WithTimeout(context.Background(), timeout)

        if err := c.killDockerContainerByNameWithContext(ctx, c.containerName); err == nil {
            cancel()
            return nil
        }
        cancel()

        if i < maxRetries-1 {
            time.Sleep(time.Duration(i+1) * time.Second)
        }
    }
    return fmt.Errorf("failed to cleanup Docker after %d retries", maxRetries)
}
```

### 2. Goroutine Cleanup Fix (Priorität: Kritisch)

**Strategie**: Context-basiertes Goroutine-Management

```go
type MenuManager struct {
    // ... existing fields ...
    serverContexts map[string]context.CancelFunc  // NEU!
}

// Bei Goroutine-Erstellung:
func (m *MenuManager) createServerActionSubmenus(serverMenuItem *systray.MenuItem, server map[string]interface{}, serverName string) {
    // Context für diesen Server erstellen
    ctx, cancel := context.WithCancel(context.Background())
    m.serverContexts[serverName] = cancel

    // Goroutine mit Context
    go func(name string, item *systray.MenuItem) {
        for {
            select {
            case <-ctx.Done():
                return  // ✅ Goroutine stoppt sauber!
            case <-item.ClickedCh:
                if m.onServerAction != nil {
                    go m.onServerAction(name, "unquarantine")
                }
            }
        }
    }(serverName, quarantineItem)
}

// Bei Cleanup:
func (m *MenuManager) cleanupServerActionItems(serverName string) {
    // Context canceln → stoppt ALLE Goroutinen für diesen Server
    if cancel, ok := m.serverContexts[serverName]; ok {
        cancel()  // ✅ Goroutinen werden beendet!
        delete(m.serverContexts, serverName)
    }

    // Dann Menu Items cleanup
    if actionItem, ok := m.serverActionItems[serverName]; ok {
        actionItem.Hide()
        delete(m.serverActionItems, serverName)
    }
    // ... rest of cleanup
}
```

### 3. Single-Instance Guard (Priorität: Mittel)

```go
// Beim mcpproxy-Start:
func ensureSingleInstance() error {
    pidFile := filepath.Join(os.TempDir(), "mcpproxy.pid")

    // Prüfen ob bereits läuft
    if data, err := os.ReadFile(pidFile); err == nil {
        oldPID, _ := strconv.Atoi(string(data))
        if processExists(oldPID) {
            return fmt.Errorf("mcpproxy already running with PID %d", oldPID)
        }
    }

    // PID-File schreiben
    return os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
}
```

## Auswirkungen

### Performance
- **Memory**: ~200MB+ für 21 verwaiste Container
- **Goroutinen**: 15 Goroutinen × Anzahl Server × Refresh-Zyklen
- **CPU**: Idle, aber verhindert GC cleanup

### Stabilität
- Docker API wird langsamer durch viele verwaiste Container
- System kann instabil werden bei zu vielen Goroutinen
- Möglicher Out-of-Memory bei langem Betrieb

## Testing Checklist

Nach Fix-Implementation:

- [ ] Docker Container werden korrekt bereinigt
- [ ] cidfiles werden gelöscht
- [ ] Goroutinen stoppen bei cleanupServerActionItems
- [ ] Nur eine mcpproxy-Instanz läuft
- [ ] Memory-Usage bleibt stabil über Zeit
- [ ] Keine "context deadline exceeded" Errors in Logs

## Metrics

### Vorher
- Docker Container: 21 orphaned
- mcpproxy Prozesse: 24 parallel
- Goroutinen: 15 × Server-Count × Refresh-Count
- cidfiles: 10 orphaned

### Nachher (nach Cleanup)
- Docker Container: 0 orphaned ✅
- mcpproxy Prozesse: 1 running ✅
- Goroutinen: Monitoring erforderlich
- cidfiles: 0 orphaned ✅

## Nächste Schritte

1. **Sofort**: Docker cleanup timeouts erhöhen (connection.go)
2. **Kritisch**: Context-basiertes Goroutine-Management (managers.go, tray.go)
3. **Wichtig**: Single-instance guard implementieren
4. **Monitoring**: Goroutine-Count tracken (pprof integration)

---

**Status**: Dokumentation abgeschlossen
**Docker**: Manuell bereinigt ✅
**Prozesse**: Manuell beendet ✅
**Code-Fixes**: Ausstehend ⚠️
