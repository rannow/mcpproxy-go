#!/bin/bash
# mcpproxy Resource Monitoring Script
# Überwacht CPU, Memory, Threads, Prozesse, Docker Container und Subprozesse

set -euo pipefail

# Farben für Output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Alert-Schwellenwerte (konfigurierbar)
CPU_WARN=50
CPU_CRIT=80
MEM_WARN=300000  # KB (300 MB)
MEM_CRIT=500000  # KB (500 MB)
THREAD_WARN=100
THREAD_CRIT=200
FD_WARN=100
FD_CRIT=200
CONTAINER_WARN=10
CONTAINER_CRIT=20

# Optionen
CONTINUOUS=false
INTERVAL=10
ALERT_ONLY=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --continuous)
            CONTINUOUS=true
            shift
            ;;
        --interval)
            INTERVAL="$2"
            shift 2
            ;;
        --alert-only)
            ALERT_ONLY=true
            shift
            ;;
        --threshold-cpu)
            CPU_WARN="$2"
            shift 2
            ;;
        --threshold-mem)
            MEM_WARN="$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --continuous         Kontinuierliches Monitoring"
            echo "  --interval N         Update-Intervall in Sekunden (default: 10)"
            echo "  --alert-only         Nur Warnungen anzeigen"
            echo "  --threshold-cpu N    CPU-Warn-Schwelle in % (default: 50)"
            echo "  --threshold-mem N    Memory-Warn-Schwelle in KB (default: 300000)"
            echo "  --help               Diese Hilfe anzeigen"
            exit 0
            ;;
        *)
            echo "Unbekannte Option: $1"
            echo "Verwende --help für Hilfe"
            exit 1
            ;;
    esac
done

# Funktion: Header ausgeben
print_header() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  mcpproxy Resource Monitor${NC}"
    echo -e "${BLUE}  $(date '+%Y-%m-%d %H:%M:%S')${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

# Funktion: Status-Symbol basierend auf Schwellenwerten
get_status() {
    local value=$1
    local warn=$2
    local crit=$3

    if (( $(echo "$value >= $crit" | bc -l) )); then
        echo -e "${RED}🔴 CRITICAL${NC}"
    elif (( $(echo "$value >= $warn" | bc -l) )); then
        echo -e "${YELLOW}⚠️  WARNING${NC}"
    else
        echo -e "${GREEN}✅ OK${NC}"
    fi
}

# Funktion: mcpproxy Hauptprozess überwachen
monitor_main_process() {
    echo -e "${BLUE}📊 mcpproxy Hauptprozess${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    local mcpproxy_pid=$(pgrep -f "^\./mcpproxy" | head -1)

    if [ -z "$mcpproxy_pid" ]; then
        echo -e "${RED}❌ mcpproxy läuft nicht!${NC}"
        return 1
    fi

    # Prozess-Informationen sammeln
    local ps_output=$(ps -p "$mcpproxy_pid" -o pid,ppid,%cpu,%mem,vsz,rss,etime,comm 2>/dev/null | tail -1)

    if [ -z "$ps_output" ]; then
        echo -e "${RED}❌ Konnte Prozess-Informationen nicht abrufen${NC}"
        return 1
    fi

    local cpu=$(echo "$ps_output" | awk '{print $3}')
    local mem=$(echo "$ps_output" | awk '{print $4}')
    local vsz=$(echo "$ps_output" | awk '{print $5}')
    local rss=$(echo "$ps_output" | awk '{print $6}')
    local etime=$(echo "$ps_output" | awk '{print $7}')

    echo "PID:      $mcpproxy_pid"
    echo "Laufzeit: $etime"
    echo ""

    # CPU
    local cpu_int=${cpu%.*}
    local cpu_status=$(get_status "$cpu_int" "$CPU_WARN" "$CPU_CRIT")
    echo -e "CPU:      ${cpu}% $cpu_status"

    # Memory
    local rss_mb=$((rss / 1024))
    local mem_status=$(get_status "$rss" "$MEM_WARN" "$MEM_CRIT")
    echo -e "Memory:   ${rss_mb} MB (RSS) $mem_status"
    echo "          $(printf "%.1f" $(echo "scale=1; $vsz/1024" | bc)) MB (VSZ)"
    echo ""
}

# Funktion: Threads überwachen
monitor_threads() {
    echo -e "${BLUE}🧵 Thread-Überwachung${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    local mcpproxy_pid=$(pgrep -f "^\./mcpproxy" | head -1)

    if [ -z "$mcpproxy_pid" ]; then
        echo -e "${RED}❌ mcpproxy läuft nicht!${NC}"
        return 1
    fi

    local thread_count=$(ps -M -p "$mcpproxy_pid" 2>/dev/null | tail -n +2 | wc -l | xargs)

    if [ -z "$thread_count" ] || [ "$thread_count" = "0" ]; then
        echo -e "${YELLOW}⚠️  Thread-Count konnte nicht ermittelt werden${NC}"
        return 0
    fi

    local thread_status=$(get_status "$thread_count" "$THREAD_WARN" "$THREAD_CRIT")
    echo -e "Threads:  $thread_count $thread_status"

    # Warnung bei möglichem Goroutine-Leak
    if [ "$thread_count" -gt "$THREAD_WARN" ]; then
        echo -e "${YELLOW}⚠️  Möglicher Goroutine-Leak erkannt!${NC}"
        echo "   Siehe: docs/RESOURCE-ANALYSIS.md"
    fi
    echo ""
}

# Funktion: File Descriptors überwachen
monitor_file_descriptors() {
    echo -e "${BLUE}📁 File Descriptors${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    local mcpproxy_pid=$(pgrep -f "^\./mcpproxy" | head -1)

    if [ -z "$mcpproxy_pid" ]; then
        echo -e "${RED}❌ mcpproxy läuft nicht!${NC}"
        return 1
    fi

    local fd_count=$(lsof -p "$mcpproxy_pid" 2>/dev/null | tail -n +2 | wc -l | xargs)

    if [ -z "$fd_count" ]; then
        fd_count=0
    fi

    local fd_status=$(get_status "$fd_count" "$FD_WARN" "$FD_CRIT")
    echo -e "Open FDs: $fd_count $fd_status"

    # Breakdown nach Typ
    if [ "$fd_count" -gt 0 ]; then
        echo ""
        echo "Typ-Verteilung:"
        lsof -p "$mcpproxy_pid" 2>/dev/null | tail -n +2 | awk '{print $5}' | sort | uniq -c | \
            awk '{printf "  %-10s %3d\n", $2, $1}'
    fi
    echo ""
}

# Funktion: Docker Container überwachen
monitor_docker_containers() {
    echo -e "${BLUE}🐳 Docker Container${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    # Prüfen ob Docker läuft
    if ! docker info >/dev/null 2>&1; then
        echo -e "${YELLOW}⚠️  Docker Daemon nicht erreichbar${NC}"
        return 0
    fi

    # Alle Container zählen
    local total_containers=$(docker ps -aq 2>/dev/null | wc -l | xargs)
    local running_containers=$(docker ps -q 2>/dev/null | wc -l | xargs)

    echo "Gesamt:   $total_containers Container"
    echo "Running:  $running_containers Container"
    echo ""

    # MCP-spezifische Container
    local mcp_containers=$(docker ps -a --format "{{.Names}}" 2>/dev/null | grep -E "aws-mcp|k8s-mcp" | wc -l | xargs)

    if [ "$mcp_containers" -gt 0 ]; then
        local container_status=$(get_status "$mcp_containers" "$CONTAINER_WARN" "$CONTAINER_CRIT")
        echo -e "MCP:      $mcp_containers Container $container_status"
        echo ""

        # Container-Details
        docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Size}}" 2>/dev/null | \
            grep -E "NAME|aws-mcp|k8s-mcp" | \
            awk '{printf "  %-20s %-20s %s\n", $1, $2" "$3" "$4, $5" "$6}'

        # Warnung bei zu vielen Containern
        if [ "$mcp_containers" -gt "$CONTAINER_WARN" ]; then
            echo ""
            echo -e "${YELLOW}⚠️  Zu viele MCP Container! Möglicher Container-Leak${NC}"
            echo "   Cleanup: docker ps -a | grep mcp | awk '{print \$1}' | xargs docker rm -f"
        fi
    else
        echo -e "${GREEN}✅ Keine MCP Container${NC}"
    fi
    echo ""
}

# Funktion: Docker Stats (Live)
monitor_docker_stats() {
    echo -e "${BLUE}📈 Docker Container Stats (Live)${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    if ! docker info >/dev/null 2>&1; then
        echo -e "${YELLOW}⚠️  Docker Daemon nicht erreichbar${NC}"
        return 0
    fi

    local running=$(docker ps -q 2>/dev/null | wc -l | xargs)

    if [ "$running" -eq 0 ]; then
        echo -e "${GREEN}✅ Keine laufenden Container${NC}"
        echo ""
        return 0
    fi

    # Stats mit Timeout
    timeout 5s docker stats --no-stream --format \
        "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}" 2>/dev/null || \
        echo -e "${YELLOW}⚠️  Docker Stats Timeout${NC}"
    echo ""
}

# Funktion: Subprozesse & Prozessbaum
monitor_subprocess_tree() {
    echo -e "${BLUE}🌳 Prozess-Hierarchie${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    local mcpproxy_pid=$(pgrep -f "^\./mcpproxy" | head -1)

    if [ -z "$mcpproxy_pid" ]; then
        echo -e "${RED}❌ mcpproxy läuft nicht!${NC}"
        return 1
    fi

    echo "Parent PID: $mcpproxy_pid"
    echo ""

    # Direkte Children finden
    local children=$(pgrep -P "$mcpproxy_pid" 2>/dev/null)

    if [ -z "$children" ]; then
        echo -e "${GREEN}✅ Keine Subprozesse${NC}"
    else
        echo "Subprozesse:"
        echo "$children" | while read -r child_pid; do
            ps -p "$child_pid" -o pid,%cpu,%mem,comm,args 2>/dev/null | tail -1 | \
                awk '{printf "  PID %-6s CPU %5s%% MEM %5s%% %s\n", $1, $2, $3, $4" "$5" "$6}'
        done
    fi

    # Zombie-Prozesse prüfen
    local zombies=$(ps --ppid "$mcpproxy_pid" -o pid,stat,comm 2>/dev/null | grep "Z" | wc -l | xargs)
    if [ "$zombies" -gt 0 ]; then
        echo ""
        echo -e "${RED}⚠️  $zombies Zombie-Prozess(e) gefunden!${NC}"
        ps --ppid "$mcpproxy_pid" -o pid,stat,comm 2>/dev/null | grep "Z"
    fi
    echo ""
}

# Funktion: Temporäre Dateien
monitor_temp_files() {
    echo -e "${BLUE}📄 Temporäre Dateien${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    # cidfiles
    local cidfiles=$(find /var/folders /tmp -name "mcpproxy-cid-*.txt" 2>/dev/null | wc -l | xargs)
    echo "cidfiles: $cidfiles"

    if [ "$cidfiles" -gt 5 ]; then
        echo -e "${YELLOW}⚠️  Zu viele cidfiles! Mögliche verwaiste Container${NC}"
        echo "   Cleanup: rm -f /var/folders/.../mcpproxy-cid-*.txt"
    fi

    # Log-Dateien
    local log_size=$(du -sh /tmp/mcpproxy*.log 2>/dev/null | awk '{print $1}' | head -1)
    if [ -n "$log_size" ]; then
        echo "Logs:     $log_size (/tmp/)"
    fi

    # Config Database
    if [ -f "$HOME/.mcpproxy/config.db" ]; then
        local db_size=$(du -h "$HOME/.mcpproxy/config.db" | awk '{print $1}')
        echo "Database: $db_size"

        # Warnung bei großer DB
        local db_bytes=$(stat -f%z "$HOME/.mcpproxy/config.db" 2>/dev/null || stat -c%s "$HOME/.mcpproxy/config.db" 2>/dev/null)
        if [ "$db_bytes" -gt 10485760 ]; then  # 10 MB
            echo -e "${YELLOW}⚠️  Database >10MB - Compaction empfohlen${NC}"
            echo "   Siehe: docs/storage-and-state.md"
        fi
    fi
    echo ""
}

# Funktion: System-Übersicht
monitor_system_overview() {
    echo -e "${BLUE}💻 System-Übersicht${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    # macOS spezifisch
    if [[ "$OSTYPE" == "darwin"* ]]; then
        top -l 1 -n 0 | head -10 | tail -8
    else
        # Linux
        free -h
        echo ""
        uptime
    fi
    echo ""
}

# Hauptfunktion: Monitoring durchführen
run_monitoring() {
    if [ "$ALERT_ONLY" = false ]; then
        clear
        print_header
        monitor_system_overview
    fi

    monitor_main_process || true
    monitor_threads || true
    monitor_file_descriptors || true
    monitor_docker_containers || true
    monitor_docker_stats || true
    monitor_subprocess_tree || true
    monitor_temp_files || true

    if [ "$ALERT_ONLY" = false ]; then
        echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo ""
    fi
}

# Main Loop
if [ "$CONTINUOUS" = true ]; then
    echo "Kontinuierliches Monitoring gestartet (Interval: ${INTERVAL}s)"
    echo "Beenden mit Ctrl+C"
    echo ""

    while true; do
        run_monitoring

        if [ "$ALERT_ONLY" = false ]; then
            echo "Nächstes Update in ${INTERVAL}s..."
        fi

        sleep "$INTERVAL"
    done
else
    run_monitoring
fi
