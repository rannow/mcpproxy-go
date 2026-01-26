# MCP Proxy Startup Connection Scheduler

## Übersicht

Der Connection Scheduler verwendet einen **Wave-basierten Ansatz** mit exponentiellen Backoff-Timeouts, um MCP-Server beim Start zu verbinden. Jede Wave verarbeitet alle Server mit einem bestimmten Timeout, bevor zur nächsten Wave übergegangen wird.

## Wave-Timeout Konfiguration

| Wave | Timeout | Beschreibung |
|------|---------|--------------|
| 1 | 20s | Initiale Verbindung aller Server |
| 2 | 40s | Retry fehlgeschlagener Server |
| 3 | 80s | Retry fehlgeschlagener Server |
| 4 | 160s | Retry fehlgeschlagener Server |
| 5 | 320s | Finale Retry-Versuche |

## Architektur-Diagramm

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     CONNECTION SCHEDULER START                           │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    ELIGIBLE CLIENTS SAMMELN                              │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │ Filter:                                                          │    │
│  │   • ShouldConnectOnStartup() == true                            │    │
│  │   • IsConnected() == false                                       │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         WAVE PROCESSING                                  │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │ Wave 1-5 Loop (solange pendingJobs > 0)                         │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┴───────────────┐
                    ▼                               ▼
    ┌───────────────────────────┐   ┌───────────────────────────┐
    │      WORKER POOL          │   │     JOB QUEUE             │
    │  ┌─────────────────────┐  │   │  ┌─────────────────────┐  │
    │  │ Worker 1 ──────────►│  │   │  │ Job 1 (server-a)    │  │
    │  │ Worker 2 ──────────►│  │◄──│  │ Job 2 (server-b)    │  │
    │  │ Worker 3 ──────────►│  │   │  │ Job 3 (server-c)    │  │
    │  │ ...                  │  │   │  │ ...                 │  │
    │  │ Worker N (max 20)   │  │   │  │ Job M               │  │
    │  └─────────────────────┘  │   │  └─────────────────────┘  │
    └───────────────────────────┘   └───────────────────────────┘
                    │
                    ▼
    ┌───────────────────────────────────────────────────────────┐
    │                  CONNECTION ATTEMPT                        │
    │  ┌─────────────────────────────────────────────────────┐  │
    │  │ ctx, cancel := context.WithTimeout(timeout)         │  │
    │  │ err := job.client.Connect(ctx)                      │  │
    │  │ elapsed := time.Since(startTime)                    │  │
    │  └─────────────────────────────────────────────────────┘  │
    └───────────────────────────────────────────────────────────┘
                    │
        ┌───────────┴───────────┐
        ▼                       ▼
┌───────────────┐       ┌───────────────┐
│   SUCCESS     │       │    FAILED     │
│  ┌─────────┐  │       │  ┌─────────┐  │
│  │ atomic  │  │       │  │ → Retry │  │
│  │ +1      │  │       │  │   Queue │  │
│  │ success │  │       │  └─────────┘  │
│  └─────────┘  │       └───────────────┘
└───────────────┘               │
                                ▼
                    ┌───────────────────────┐
                    │  NÄCHSTE WAVE         │
                    │  (höherer Timeout)    │
                    └───────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         SCHEDULER RESULT                                 │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │ • Duration: Gesamtdauer                                         │    │
│  │ • TotalJobs: Anzahl eligibler Server                            │    │
│  │ • Successful: Erfolgreiche Verbindungen                         │    │
│  │ • Failed: Fehlgeschlagene nach allen Waves                      │    │
│  │ • Retried: Anzahl Retry-Versuche                                │    │
│  │ • Timing Metrics: Min/Max/Avg Connection Times                  │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

## Worker Pool Mechanismus

Der Scheduler verwendet einen **Fan-Out/Fan-In** Pattern:

```
                    ┌─────────────┐
                    │  Job Queue  │
                    │  (buffered  │
                    │   channel)  │
                    └──────┬──────┘
                           │
         ┌─────────────────┼─────────────────┐
         ▼                 ▼                 ▼
    ┌─────────┐       ┌─────────┐       ┌─────────┐
    │Worker 1 │       │Worker 2 │       │Worker N │
    │  ┌───┐  │       │  ┌───┐  │       │  ┌───┐  │
    │  │Job│  │       │  │Job│  │       │  │Job│  │
    │  └─┬─┘  │       │  └─┬─┘  │       │  └─┬─┘  │
    │    │    │       │    │    │       │    │    │
    │ Connect │       │ Connect │       │ Connect │
    │    │    │       │    │    │       │    │    │
    │ Result  │       │ Result  │       │ Result  │
    └────┬────┘       └────┬────┘       └────┬────┘
         │                 │                 │
         └─────────────────┼─────────────────┘
                           ▼
                    ┌─────────────┐
                    │Result Queue │
                    │  (buffered  │
                    │   channel)  │
                    └─────────────┘
```

### Konfiguration

- **Default Worker Count**: 10 (kann über `globalConfig.MaxConcurrentConnections` überschrieben werden)
- **Job Channel**: Gepuffert mit Kapazität = Anzahl Jobs
- **Result Channel**: Gepuffert mit Kapazität = Anzahl Jobs

## Timing Metriken

Der Scheduler erfasst detaillierte Timing-Metriken für alle Verbindungsversuche:

### Alle Versuche (inkl. Fehlgeschlagene)
- **MinConnectTime**: Kürzeste Verbindungszeit
- **MaxConnectTime**: Längste Verbindungszeit
- **AvgConnectTime**: Durchschnittliche Verbindungszeit

### Nur erfolgreiche Verbindungen
- **SuccessMinTime**: Kürzeste erfolgreiche Verbindung
- **SuccessMaxTime**: Längste erfolgreiche Verbindung
- **SuccessAvgTime**: Durchschnitt erfolgreicher Verbindungen

### Theoretische Grenzen

| Metrik | Minimum | Maximum |
|--------|---------|---------|
| Einzelne Verbindung | ~0ms (sofort) | Wave-Timeout (10-160s) |
| Wave 1 | 0ms | 20s |
| Wave 2 | 0ms | 40s |
| Wave 3 | 0ms | 80s |
| Wave 4 | 0ms | 160s |
| Wave 5 | 0ms | 320s |
| Gesamtdauer (worst case) | 0ms | 620s (alle Waves) |

## Retry-Queue Mechanismus

```
Wave 1 (10s timeout)
    │
    ├── Server A ─── SUCCESS ──► Done
    ├── Server B ─── TIMEOUT ──► Retry Queue
    ├── Server C ─── ERROR ────► Retry Queue
    └── Server D ─── SUCCESS ──► Done
                         │
                         ▼
Wave 2 (20s timeout)
    │
    ├── Server B ─── SUCCESS ──► Done
    └── Server C ─── ERROR ────► Retry Queue
                         │
                         ▼
Wave 3 (40s timeout)
    │
    └── Server C ─── SUCCESS ──► Done
```

### Regeln:
1. **Fehlgeschlagene Jobs** (Timeout ODER Error) werden zur nächsten Wave weitergereicht
2. **Erfolgreiche Jobs** werden aus der Queue entfernt
3. **Nach Wave 5** werden verbleibende Jobs als endgültig fehlgeschlagen markiert
4. **Auto-Disable Threshold**: Nach 7 konsekutiven Fehlversuchen wird ein Server auto-disabled

## Code-Struktur

### Hauptkomponenten

```go
// scheduler.go

type ConnectionScheduler struct {
    manager     *Manager
    workerCount int
    logger      *zap.Logger
    ctx         context.Context
    cancel      context.CancelFunc

    // Atomic Metriken
    totalAttempts int64
    successful    int64
    failed        int64
}

type connectionJob struct {
    id     string
    client *managed.Client
}

type connectionResult struct {
    job     *connectionJob
    success bool
    err     error
    elapsed time.Duration
}

type waveResults struct {
    failedJobs   []*connectionJob
    allTimes     []time.Duration
    successTimes []time.Duration
}

type SchedulerResult struct {
    Duration       time.Duration
    TotalJobs      int
    Successful     int
    Failed         int
    Retried        int
    MinConnectTime time.Duration
    MaxConnectTime time.Duration
    AvgConnectTime time.Duration
    SuccessMinTime time.Duration
    SuccessMaxTime time.Duration
    SuccessAvgTime time.Duration
}
```

### Hauptfunktionen

| Funktion | Beschreibung |
|----------|--------------|
| `NewConnectionScheduler()` | Erstellt neuen Scheduler mit Worker-Count |
| `Start(clients)` | Startet Wave-Processing für alle Clients |
| `processWave()` | Verarbeitet eine Wave mit spezifischem Timeout |
| `waveWorker()` | Worker-Goroutine für Connection-Versuche |
| `calculateTimingMetrics()` | Berechnet Min/Max/Avg aus Duration-Slice |
| `Stop()` | Stoppt alle laufenden Operationen |
| `GetMetrics()` | Gibt aktuelle Scheduler-Metriken zurück |

## Log-Ausgaben

Der Scheduler produziert strukturierte Logs mit dem Prefix `STARTUP:`:

```
STARTUP: Connection scheduler starting          (worker_count, total_clients, max_waves)
STARTUP: Eligible clients collected             (eligible_count)
STARTUP: Starting wave                          (wave, max_waves, timeout, servers_to_process)
STARTUP: Worker starting connection             (worker_id, wave, server, timeout)
STARTUP: Connection successful                  (worker_id, wave, server, elapsed)
STARTUP: Connection failed                      (worker_id, wave, server, elapsed, timeout, error)
STARTUP: Wave completed                         (wave, wave_duration, successful, failed, remaining_waves)
STARTUP: Servers queued for next wave           (count, next_timeout)
STARTUP: Max retries exceeded                   (server, attempts)
STARTUP: Connection scheduler completed         (total_duration, total_servers, successful, failed, total_retried)
STARTUP: Connection timing metrics (all)        (min, max, avg)
STARTUP: Connection timing metrics (success)    (min, max, avg)
```

## Beispiel Log-Ausgabe

```
═══════════════════════════════════════════════════════════════
STARTUP: Connection scheduler starting  worker_count=20 total_clients=22 max_waves=5
STARTUP: Eligible clients collected     eligible_count=21
───────────────────────────────────────────────────────────────
STARTUP: Starting wave                  wave=1 max_waves=5 timeout=10s servers_to_process=21
STARTUP: Connection successful          worker_id=3 wave=1 server=mcp-filesystem elapsed=2.3s
STARTUP: Connection failed              worker_id=7 wave=1 server=slow-server elapsed=10.007s error=context deadline exceeded
STARTUP: Wave completed                 wave=1 wave_duration=10.012s successful=1 failed=20 remaining_waves=4
───────────────────────────────────────────────────────────────
STARTUP: Starting wave                  wave=2 max_waves=5 timeout=20s servers_to_process=20
...
═══════════════════════════════════════════════════════════════
STARTUP: Connection scheduler completed total_duration=60.8s total_servers=21 successful=3 failed=18 total_retried=38
STARTUP: Connection timing metrics (all) min=2.3s max=40.5s avg=18.7s
STARTUP: Connection timing metrics (success) min=2.3s max=30.8s avg=12.1s
═══════════════════════════════════════════════════════════════
```
