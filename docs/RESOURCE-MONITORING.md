# mcpproxy Resource Monitoring Guide

**Letzte Aktualisierung**: 2025-10-17
**Zweck**: Umfassende Ressourcen-Überwachung für mcpproxy und abhängige Services

## System-Übersicht

### Aktueller Status
```
CPU-Last: 57.62% user, 18.54% sys, 23.83% idle
Speicher: 125G used (6659M wired, 3555M compressor), 2039M unused
Prozesse: 1151 total, 13 running, 1138 sleeping
Threads: 11268 system-weit
```

## 1. mcpproxy Hauptprozess

### CPU & Memory Monitoring

**Befehl zur Überwachung**:
```bash
# Hauptprozess finden
MCPPROXY_PID=$(pgrep -f "^\./mcpproxy" | head -1)

# Ressourcen überwachen
ps -p $MCPPROXY_PID -o pid,ppid,%cpu,%mem,vsz,rss,etime,comm,args
```

**Beispiel-Output**:
```
PID   PPID  %CPU %MEM    VSZ   RSS     ELAPSED COMMAND          ARGS
12345 1     2.5  0.8  850432 131072  01:23:45 mcpproxy         ./mcpproxy
```

**Metriken**:
- **PID**: Process ID
- **%CPU**: CPU-Auslastung (Durchschnitt)
- **%MEM**: Speicher-Auslastung (Prozent)
- **VSZ**: Virtual Memory Size (KB)
- **RSS**: Resident Set Size - tatsächlich verwendeter RAM (KB)
- **ELAPSED**: Laufzeit seit Start

### Thread-Überwachung

**Befehl**:
```bash
# Thread-Count für mcpproxy
ps -M -p $MCPPROXY_PID | tail -n +2 | wc -l

# Detaillierte Thread-Info
ps -M -p $MCPPROXY_PID -o pid,tid,user,%cpu,%mem,state,comm
```

**Erwartete Werte**:
- **Normal**: 10-20 Threads (Basis-Goroutinen)
- **Pro Server**: +15 Threads (Menu-Handler-Goroutinen)
- **Warnung**: >100 Threads (möglicher Goroutine-Leak)
- **Kritisch**: >500 Threads (definitiver Leak)

### File Descriptors

**Befehl**:
```bash
# Anzahl offener File Descriptors
lsof -p $MCPPROXY_PID 2>/dev/null | wc -l

# Detaillierte Auflistung
lsof -p $MCPPROXY_PID | head -50
```

**Kategorien**:
- **Sockets**: MCP Server Connections
- **Pipes**: Stdio Communications
- **Regular Files**: Logs, Config, Database
- **TTY**: Terminal Connections

**Grenzwerte**:
- **macOS Default**: 256 per process
- **Normal**: 20-50 FDs
- **Warnung**: >100 FDs
- **Kritisch**: >200 FDs

## 2. Docker Backend Prozesse

### Docker Daemon Monitoring

**Befehle**:
```bash
# Docker Backend-Prozesse finden
ps aux | grep -i "docker" | grep -v grep

# Haupt-Docker-Prozesse:
# - com.docker.backend (Services)
# - com.docker.virtualization (VM)
# - com.docker.build (Build-System)
# - com.docker.vmnetd (Network)
```

**Beispiel-Output**:
```
USER    PID   %CPU %MEM    VSZ     RSS  COMMAND
user    12541  0.2  0.1  413779  171744 com.docker.backend services
user    66724  0.0  0.0  411858   31888 com.docker.virtualization
user    15435  0.0  0.0  411858   43200 com.docker.build
```

**Warnung**: Mehrere `com.docker.backend`-Instanzen können auf Probleme hinweisen!

### Docker Container Status

**Befehle**:
```bash
# Alle Container mit Status
docker ps -a --format "table {{.ID}}\t{{.Names}}\t{{.Status}}\t{{.Size}}"

# MCP-spezifische Container
docker ps -a | grep -E "aws-mcp-server|k8s-mcp-server"

# Container-Anzahl
docker ps -q | wc -l  # Running
docker ps -aq | wc -l # All (including stopped)
```

### Docker Container Ressourcen

**Live-Monitoring**:
```bash
# Echtzeit-Statistiken
docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}\t{{.BlockIO}}"
```

**Beispiel-Output**:
```
NAME              CPU %   MEM USAGE / LIMIT   MEM %   NET I/O       BLOCK I/O
aws-mcp-server    0.5%    45.2MiB / 7.68GiB  0.57%   1.2kB / 0B   0B / 0B
k8s-mcp-server    1.2%    67.8MiB / 7.68GiB  0.86%   2.5kB / 0B   0B / 0B
```

### Docker Volume Mounts

**Befehle**:
```bash
# Container-Mounts anzeigen
docker inspect $(docker ps -q) --format '{{.Name}}: {{json .Mounts}}' | jq

# Vereinfachte Ansicht
docker ps -q | xargs -I {} docker inspect {} --format \
  '{{.Name}}: {{range .Mounts}}{{.Source}}->{{.Destination}} {{end}}'
```

**Typische Mounts für MCP Server**:
```
/aws-mcp-server:
  - ~/.aws:/home/appuser/.aws:ro (AWS Credentials)

/k8s-mcp-server:
  - ~/.kube:/home/appuser/.kube:ro (Kubernetes Config)
  - ENV: K8S_CONTEXT=my-cluster
  - ENV: K8S_NAMESPACE=my-namespace
```

## 3. Subprozesse & MCP Server

### Prozess-Hierarchie

**Befehl**:
```bash
# Prozessbaum anzeigen
pstree -p $(pgrep -f mcpproxy | head -1) 2>/dev/null || \
ps -ef | grep mcpproxy | grep -v grep | awk '{print $2}' | \
xargs -I {} ps -o pid,ppid,comm,args --ppid {}
```

**Erwartete Struktur**:
```
mcpproxy (PID: 12345)
├── npx (PID: 12346) - MCP Server Wrapper
│   └── node (PID: 12347) - aws-mcp-server
├── docker (PID: 12348) - Container Manager
│   └── aws-mcp-server (Container)
└── docker (PID: 12349) - Container Manager
    └── k8s-mcp-server (Container)
```

### MCP Server Verbindungen

**Befehle**:
```bash
# Netzwerk-Verbindungen von mcpproxy
lsof -i -a -p $MCPPROXY_PID

# Alle Socket-Verbindungen
netstat -an | grep -E "ESTABLISHED|LISTEN" | grep $(pgrep mcpproxy | head -1)
```

**Typische Verbindungen**:
- **HTTP MCP Servers**: TCP Sockets zu localhost:XXXX
- **Stdio MCP Servers**: Pipes (FIFO)
- **Docker MCP Servers**: Unix Domain Sockets zu Docker Daemon

### Zombie-Prozesse Erkennung

**Befehle**:
```bash
# Zombie-Prozesse finden
ps aux | awk '$8=="Z" {print $2, $11}'

# Zombie-Prozesse von mcpproxy
ps --ppid $MCPPROXY_PID -o pid,stat,comm | grep "Z"
```

**Warnung**: Zombie-Prozesse (<defunct>) zeigen, dass Subprozesse nicht korrekt bereinigt wurden!

## 4. Temporäre Dateien & Caches

### cidfiles (Docker Container IDs)

**Befehle**:
```bash
# Alle cidfiles finden
find /tmp /var/folders -name "mcpproxy-cid-*.txt" 2>/dev/null

# Anzahl und Größe
find /var/folders -name "mcpproxy-cid-*.txt" -exec ls -lh {} \; | wc -l
```

**Erwartung**:
- **Normal**: 0-2 cidfiles (aktive Container)
- **Problem**: >5 cidfiles (verwaiste Container)

### Log-Dateien

**Befehle**:
```bash
# Log-Dateien finden
ls -lh /tmp/mcpproxy*.log ~/.mcpproxy/*.log 2>/dev/null

# Größe aller Logs
du -sh ~/.mcpproxy/logs/ 2>/dev/null
```

### Database Files

**Befehle**:
```bash
# Config Database
ls -lh ~/.mcpproxy/config.db

# Database-Größe überwachen
du -h ~/.mcpproxy/config.db
```

**Warnung**: Siehe `docs/storage-and-state.md` für BBolt Compaction bei Größe >10MB

## 5. Goroutine Monitoring (Erweitert)

### pprof Integration (zukünftig)

**Implementierung erforderlich**:
```go
// In main.go hinzufügen:
import _ "net/http/pprof"

go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

**Monitoring-Befehle** (nach Implementation):
```bash
# Goroutine-Count
curl http://localhost:6060/debug/pprof/goroutine?debug=1

# Heap Profile
curl http://localhost:6060/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# CPU Profile
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof
```

## 6. Monitoring-Script

### Automatisches Monitoring-Tool

Siehe: `scripts/monitor-resources.sh`

**Features**:
- CPU/Memory/Thread-Überwachung
- Docker Container Status
- Goroutine Leak Detection
- Temporäre Dateien Cleanup
- Alert-Thresholds

**Verwendung**:
```bash
# Einmaliger Snapshot
./scripts/monitor-resources.sh

# Kontinuierliches Monitoring (jede 10 Sekunden)
./scripts/monitor-resources.sh --continuous --interval 10

# Alert-Modus (nur bei Problemen)
./scripts/monitor-resources.sh --alert-only --threshold-cpu 80 --threshold-mem 1000
```

## 7. Alert-Schwellenwerte

### CPU
- **Normal**: 0-5% im Idle, 10-30% bei Aktivität
- **Warnung**: >50% für >5 Minuten
- **Kritisch**: >80% für >1 Minute

### Memory (RSS)
- **Normal**: 50-150 MB
- **Warnung**: >300 MB
- **Kritisch**: >500 MB

### Threads
- **Normal**: 10-50 Threads
- **Warnung**: >100 Threads
- **Kritisch**: >200 Threads (Goroutine Leak!)

### File Descriptors
- **Normal**: 20-50 FDs
- **Warnung**: >100 FDs
- **Kritisch**: >200 FDs

### Docker Container
- **Normal**: 0-5 MCP Container (abhängig von Konfiguration)
- **Warnung**: >10 Container
- **Kritisch**: >20 Container (Container Leak!)

## 8. Troubleshooting

### Hohe CPU-Last

**Diagnose**:
```bash
# CPU-intensive Prozesse identifizieren
top -o cpu | head -20

# mcpproxy CPU-Profil
sample $MCPPROXY_PID 10 -file mcpproxy-cpu.txt
```

**Mögliche Ursachen**:
- Zu viele MCP Server Connections
- Goroutine-Leaks verursachen Scheduling-Overhead
- Docker Container-Probleme

### Hoher Memory-Verbrauch

**Diagnose**:
```bash
# Memory-intensive Prozesse
top -o mem | head -20

# Heap-Analyse (pprof erforderlich)
curl http://localhost:6060/debug/pprof/heap > heap.prof
go tool pprof -top heap.prof
```

**Mögliche Ursachen**:
- Goroutine-Leaks
- MCP Response Caching ohne Limits
- BBolt Database Wachstum

### Viele Zombie-Prozesse

**Diagnose**:
```bash
ps aux | grep "<defunct>" | grep mcpproxy
```

**Ursache**: Subprozesse (MCP Server, Docker) werden nicht korrekt beendet

**Fix**: Siehe `docs/RESOURCE-ANALYSIS.md` → Docker Cleanup Fixes

## 9. Performance Baselines

### Idle State (keine aktiven MCP Connections)
```
CPU: 0.1-0.5%
Memory: 50-80 MB RSS
Threads: 10-15
File Descriptors: 15-25
Docker Containers: 0
```

### Normal Operation (5 MCP Servers aktiv)
```
CPU: 1-5%
Memory: 100-150 MB RSS
Threads: 20-80 (15 per Server)
File Descriptors: 30-60
Docker Containers: 0-5 (abhängig von Server-Typ)
```

### Peak Load (10 MCP Servers, hohe Aktivität)
```
CPU: 10-25%
Memory: 200-300 MB RSS
Threads: 50-150
File Descriptors: 60-100
Docker Containers: 0-10
```

## 10. Monitoring Best Practices

### Regelmäßige Checks
1. **Täglich**: Docker Container Cleanup (`docker ps -a`)
2. **Wöchentlich**: BBolt Database Compaction
3. **Monatlich**: Log-Rotation und Cleanup
4. **Bei Neustart**: Prozess- und Container-Cleanup

### Automatisierung
```bash
# Cron Job (täglich um 3 Uhr)
0 3 * * * /path/to/mcpproxy-go/scripts/cleanup-resources.sh

# Systemd Timer (alternative)
systemctl enable --now mcpproxy-cleanup.timer
```

### Metrics Sammlung (zukünftig)
- Prometheus Integration für Langzeit-Monitoring
- Grafana Dashboards für Visualisierung
- Alert-Manager für automatische Benachrichtigungen

---

**Nächste Schritte**:
1. `scripts/monitor-resources.sh` Script erstellen
2. pprof Integration in main.go
3. Prometheus Exporter implementieren
4. Automated cleanup scripts einrichten

**Siehe auch**:
- `docs/RESOURCE-ANALYSIS.md` - Bekannte Resource Leaks
- `docs/storage-and-state.md` - BBolt Database Management
- `docs/MERMAID-DIAGRAMS.md` - Architektur-Diagramme
