# mcpproxy Utility Scripts

Hilfs-Scripts für Monitoring, Cleanup und Wartung von mcpproxy.

## 📊 monitor-resources.sh

Umfassendes Ressourcen-Monitoring für mcpproxy und abhängige Services.

### Features

- **Hauptprozess**: CPU, Memory, Threads, File Descriptors
- **Docker**: Container Status, Stats, Mounts
- **Subprozesse**: Prozess-Hierarchie, Zombie-Detection
- **Temporäre Dateien**: cidfiles, Logs, Database-Größe
- **System-Übersicht**: CPU/Memory-Last, Uptime
- **Alert-System**: Farbcodierte Warnungen bei Grenzwertüberschreitung

### Verwendung

#### Einmaliger Snapshot
```bash
./scripts/monitor-resources.sh
```

#### Kontinuierliches Monitoring
```bash
# Alle 10 Sekunden (Standard)
./scripts/monitor-resources.sh --continuous

# Custom-Intervall (alle 5 Sekunden)
./scripts/monitor-resources.sh --continuous --interval 5
```

#### Alert-Modus
```bash
# Nur Warnungen anzeigen
./scripts/monitor-resources.sh --alert-only

# Custom Alert-Schwellenwerte
./scripts/monitor-resources.sh \
  --alert-only \
  --threshold-cpu 80 \
  --threshold-mem 500000
```

### Alert-Schwellenwerte (Default)

| Ressource | Warning | Critical |
|-----------|---------|----------|
| CPU | 50% | 80% |
| Memory | 300 MB | 500 MB |
| Threads | 100 | 200 |
| File Descriptors | 100 | 200 |
| Docker Container | 10 | 20 |

### Beispiel-Output

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  mcpproxy Resource Monitor
  2025-10-17 21:45:30
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

💻 System-Übersicht
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Processes: 1151 total, 13 running, 1138 sleeping
CPU usage: 57.6% user, 18.5% sys, 23.8% idle
PhysMem: 125G used, 2039M unused

📊 mcpproxy Hauptprozess
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
PID:      12345
Laufzeit: 01:23:45

CPU:      2.5% ✅ OK
Memory:   128 MB (RSS) ✅ OK
          830 MB (VSZ)

🧵 Thread-Überwachung
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Threads:  45 ✅ OK

🐳 Docker Container
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Gesamt:   0 Container
Running:  0 Container
✅ Keine MCP Container
```

## 🧹 cleanup-resources.sh

Automatische Bereinigung von verwaisten Ressourcen.

### Features

- **Docker Container**: Entfernt verwaiste MCP Container
- **cidfiles**: Löscht temporäre Container-ID-Dateien
- **Prozesse**: Beendet alte mcpproxy-Instanzen (behält neueste)
- **Logs**: Entfernt alte Log-Dateien (>7 Tage)
- **Database**: Warnung bei großer Database (Compaction-Empfehlung)

### Verwendung

#### Dry-Run (Simulation)
```bash
./scripts/cleanup-resources.sh --dry-run
```

#### Interaktives Cleanup
```bash
./scripts/cleanup-resources.sh
# Fragt vor jeder Aktion nach Bestätigung
```

#### Automatisches Cleanup
```bash
./scripts/cleanup-resources.sh --force
# Keine Rückfragen, führt alles direkt aus
```

#### Verbose Mode
```bash
./scripts/cleanup-resources.sh --verbose
# Detaillierte Ausgabe aller Aktionen
```

### Beispiel-Output

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  mcpproxy Resource Cleanup
  2025-10-17 21:50:00
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🐳 Docker Container Cleanup
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Gefunden: 5 MCP Container
Alle 5 Container löschen? [y/N] y
✅ 5 Container gelöscht

📄 cidfiles Cleanup
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Gefunden: 8 cidfiles
Alle 8 cidfiles löschen? [y/N] y
✅ 8 cidfiles gelöscht

✅ Cleanup abgeschlossen
```

## 🔄 Automatisierung

### Cron Job (täglich um 3 Uhr)

```bash
# Crontab bearbeiten
crontab -e

# Zeile hinzufügen:
0 3 * * * /path/to/mcpproxy-go/scripts/cleanup-resources.sh --force >> /tmp/mcpproxy-cleanup.log 2>&1
```

### Systemd Timer (Linux)

```ini
# /etc/systemd/system/mcpproxy-cleanup.timer
[Unit]
Description=Daily mcpproxy resource cleanup
Requires=mcpproxy-cleanup.service

[Timer]
OnCalendar=daily
Persistent=true

[Install]
WantedBy=timers.target
```

```ini
# /etc/systemd/system/mcpproxy-cleanup.service
[Unit]
Description=mcpproxy resource cleanup

[Service]
Type=oneshot
ExecStart=/path/to/mcpproxy-go/scripts/cleanup-resources.sh --force
```

Aktivieren:
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now mcpproxy-cleanup.timer
```

## 📈 Monitoring Integration

### Prometheus (zukünftig)

Nach pprof-Implementation können Metriken exportiert werden:

```bash
# HTTP Endpoint für Metriken
curl http://localhost:6060/debug/pprof/goroutine?debug=1

# In Prometheus scrape_configs:
- job_name: 'mcpproxy'
  static_configs:
    - targets: ['localhost:6060']
```

### Grafana Dashboard (zukünftig)

Metrics für Dashboard:
- CPU Usage (%)
- Memory RSS (MB)
- Thread Count
- File Descriptor Count
- Docker Container Count
- Goroutine Count (via pprof)

## 🔍 Troubleshooting

### Script läuft nicht

```bash
# Prüfe Permissions
ls -l scripts/*.sh

# Sollte sein: -rwxr-xr-x (executable)
chmod +x scripts/*.sh
```

### Docker-Befehle schlagen fehl

```bash
# Prüfe Docker-Daemon
docker info

# macOS: Docker Desktop starten
open -a Docker
```

### Keine Prozesse gefunden

```bash
# Prüfe ob mcpproxy läuft
pgrep -f mcpproxy

# Starte mcpproxy
cd /path/to/mcpproxy-go
./mcpproxy
```

## 📝 Logs

### Script-Logs

```bash
# Monitor-Log (bei kontinuierlichem Betrieb)
./scripts/monitor-resources.sh --continuous > /tmp/mcpproxy-monitor.log 2>&1

# Cleanup-Log (bei Cron)
# Siehe /tmp/mcpproxy-cleanup.log
```

## 🆘 Support

Bei Problemen siehe:
- `docs/RESOURCE-MONITORING.md` - Detaillierte Dokumentation
- `docs/RESOURCE-ANALYSIS.md` - Bekannte Resource Leaks
- `docs/storage-and-state.md` - Database Management

## 🔜 Zukünftige Features

- [ ] pprof Integration für Goroutine-Monitoring
- [ ] Prometheus Metrics Exporter
- [ ] Grafana Dashboard Templates
- [ ] Automated Alert-Emails
- [ ] Slack/Discord Notifications
- [ ] Resource Usage History (Time-Series DB)
- [ ] Web UI für Monitoring
