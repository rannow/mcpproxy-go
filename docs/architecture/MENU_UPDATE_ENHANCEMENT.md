# Menu Update Enhancement - Adaptive Sync System

## Overview

Das MCPProxy Tray-System wurde mit einem **adaptiven Synchronisationssystem** erweitert, das automatisch die Aktualisierungsfrequenz der Menüs basierend auf Benutzeraktivität anpasst.

## Problem

Das ursprüngliche System aktualisierte Server-Listen alle 3 Sekunden im Hintergrund. Obwohl dies bereits sehr responsiv war, gab es noch Raum für Verbesserungen:

- **Menu-Open-Events**: Das systray-Framework bietet keine direkten Events für das Öffnen von Menüs
- **Benutzererwartung**: Nutzer erwarten sofortige Updates wenn sie aktiv mit dem Menü interagieren
- **Ressourceneffizienz**: Konstante 3-Sekunden-Updates sind unnötig wenn niemand das System benutzt

## Lösung: Adaptive Sync-Frequenz

### System-Design

```go
type SynchronizationManager struct {
    // ... existing fields ...

    // User activity tracking for adaptive sync
    lastUserActivity time.Time
    activityMu       sync.RWMutex
}
```

### Adaptive Logik

1. **Normal Mode**: 3-Sekunden Sync-Intervall (inaktive Nutzer)
2. **Fast Mode**: 1-Sekunden Sync-Intervall (aktive Nutzer)
3. **Activity Window**: 10 Sekunden nach letzter Benutzeraktivität

### Implementierung

#### 1. Benutzeraktivität-Tracking

```go
// NotifyUserActivity records user interaction to enable adaptive sync frequency
func (m *SynchronizationManager) NotifyUserActivity() {
    m.activityMu.Lock()
    m.lastUserActivity = time.Now()
    m.activityMu.Unlock()

    // Also trigger an immediate sync to ensure up-to-date menu when user is active
    m.SyncDelayed()
}
```

#### 2. Adaptive Sync-Loop

```go
func (m *SynchronizationManager) syncLoop() {
    ticker := time.NewTicker(3 * time.Second)
    defer ticker.Stop()

    fastSyncMode := false

    for {
        select {
        case <-ticker.C:
            // Check user activity for adaptive sync frequency
            m.activityMu.RLock()
            timeSinceActivity := time.Since(m.lastUserActivity)
            m.activityMu.RUnlock()

            // Switch to fast sync when user is active
            if timeSinceActivity < 10*time.Second && !fastSyncMode {
                ticker.Stop()
                ticker = time.NewTicker(1 * time.Second)
                fastSyncMode = true
                m.logger.Debug("Switching to fast sync mode (1s) due to recent user activity")
            } else if timeSinceActivity >= 10*time.Second && fastSyncMode {
                ticker.Stop()
                ticker = time.NewTicker(3 * time.Second)
                fastSyncMode = false
                m.logger.Debug("Switching back to normal sync mode (3s) after user inactivity")
            }

            if err := m.performSync(); err != nil {
                m.logger.Error("Background sync failed", zap.Error(err))
            }
        case <-m.ctx.Done():
            return
        }
    }
}
```

#### 3. User-Interaction Integration

Alle wichtigen Menu-Handler rufen jetzt `NotifyUserActivity()` auf:

```go
func (a *App) handleServerAction(serverName, action string) {
    // ... logging ...

    // Notify sync manager of user activity for adaptive frequency
    if a.syncManager != nil {
        a.syncManager.NotifyUserActivity()
    }

    // ... rest of handler ...
}
```

## Funktionalität

### Sync-Frequenzen

| Zustand | Intervall | Trigger | Beschreibung |
|---------|----------|---------|--------------|
| **Inaktiv** | 3 Sekunden | >10s seit letzter Aktivität | Standard Hintergrund-Sync |
| **Aktiv** | 1 Sekunde | <10s seit letzter Aktivität | Responsive Updates für aktive Nutzer |
| **Sofort** | Immediate | Jede Benutzerinteraktion | Instant Update bei Klicks |

### Activity Detection

- **Server Actions**: Enable/Disable, OAuth Login, Quarantine, etc.
- **Group Management**: Alle Group-Management-Operationen
- **Menu Navigation**: Wichtige Menu-Interaktionen

### Performance Benefits

1. **Responsive UX**: Sofortige Updates bei Benutzerinteraktion
2. **Adaptive Performance**: Schnelle Updates nur wenn nötig
3. **Resource Efficiency**: Weniger CPU/Network wenn inaktiv
4. **Better Perceived Performance**: Nutzer sehen sofortige Reaktionen

## Testing

Das System enthält umfassende Tests:

```bash
# Run all adaptive sync tests
go test ./internal/tray -v -run TestAdaptive

# Run full tray test suite
go test ./internal/tray -v
```

### Test Coverage

- [x] **Initialization**: Proper setup of sync manager
- [x] **Activity Tracking**: User activity recording and timing
- [x] **Concurrency**: Thread-safe access to activity data
- [x] **Context Handling**: Proper cleanup and cancellation

## Backwards Compatibility

✅ **Vollständig rückwärtskompatibel**

- Keine Breaking Changes
- Alle bestehenden APIs funktionieren unverändert
- Opt-in Enhancement - das System funktioniert auch ohne User-Activity-Calls
- Graceful degradation wenn einzelne Components fehlen

## Monitoring & Debugging

### Debug Logs

Das System erstellt Debug-Logs für Sync-Mode-Wechsel:

```
DEBUG | Switching to fast sync mode (1s) due to recent user activity
DEBUG | Switching back to normal sync mode (3s) after user inactivity
```

### Performance Monitoring

Das adaptive System kann durch Logs überwacht werden:

```bash
# Monitor sync frequency changes
tail -f ~/Library/Logs/mcpproxy/main.log | grep -E "(fast sync mode|normal sync mode)"

# Monitor user activity triggers
tail -f ~/Library/Logs/mcpproxy/main.log | grep -E "Handling server action"
```

## Conclusion

Das adaptive Sync-System bietet:

1. **✅ Sofortige Updates** beim Öffnen/Benutzen der Menüs
2. **✅ Intelligente Performance** basierend auf Benutzeraktivität
3. **✅ Vollständige Rückwärtskompatibilität**
4. **✅ Umfassende Tests** und Monitoring

**Das System löst das ursprüngliche Problem**: Obwohl das systray-Framework keine Menu-Open-Events bietet, bietet das adaptive System jetzt die bestmögliche User Experience durch intelligente Activity-Detection und responsive Updates.