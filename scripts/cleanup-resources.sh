#!/bin/bash
# mcpproxy Resource Cleanup Script
# Bereinigt verwaiste Container, Prozesse und temporäre Dateien

set -euo pipefail

# Farben
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

# Optionen
DRY_RUN=false
FORCE=false
VERBOSE=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --force)
            FORCE=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --dry-run    Zeige was bereinigt würde, ohne es auszuführen"
            echo "  --force      Keine Rückfrage vor Cleanup"
            echo "  --verbose    Detaillierte Ausgabe"
            echo "  --help       Diese Hilfe anzeigen"
            exit 0
            ;;
        *)
            echo "Unbekannte Option: $1"
            echo "Verwende --help für Hilfe"
            exit 1
            ;;
    esac
done

log() {
    if [ "$VERBOSE" = true ] || [ "$DRY_RUN" = true ]; then
        echo -e "$@"
    fi
}

# Header
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  mcpproxy Resource Cleanup${NC}"
echo -e "${BLUE}  $(date '+%Y-%m-%d %H:%M:%S')${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}⚠️  DRY RUN MODE - Keine Änderungen werden vorgenommen${NC}"
    echo ""
fi

# Cleanup 1: Verwaiste Docker Container
cleanup_docker_containers() {
    echo -e "${BLUE}🐳 Docker Container Cleanup${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    if ! docker info >/dev/null 2>&1; then
        log "${YELLOW}⚠️  Docker Daemon nicht erreichbar${NC}"
        echo ""
        return 0
    fi

    # Finde MCP Container
    local mcp_containers=$(docker ps -a --format "{{.ID}} {{.Names}}" 2>/dev/null | grep -E "aws-mcp|k8s-mcp" || true)

    if [ -z "$mcp_containers" ]; then
        echo -e "${GREEN}✅ Keine MCP Container gefunden${NC}"
        echo ""
        return 0
    fi

    local container_count=$(echo "$mcp_containers" | wc -l | xargs)
    echo "Gefunden: $container_count MCP Container"
    echo ""

    if [ "$DRY_RUN" = true ]; then
        echo "$mcp_containers" | while read -r line; do
            local id=$(echo "$line" | awk '{print $1}')
            local name=$(echo "$line" | awk '{print $2}')
            echo "  Würde löschen: $name ($id)"
        done
    else
        if [ "$FORCE" = false ]; then
            read -p "Alle $container_count Container löschen? [y/N] " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                echo "Abgebrochen"
                echo ""
                return 0
            fi
        fi

        echo "$mcp_containers" | while read -r line; do
            local id=$(echo "$line" | awk '{print $1}')
            local name=$(echo "$line" | awk '{print $2}')
            log "Lösche: $name ($id)"
            docker rm -f "$id" >/dev/null 2>&1 || true
        done

        echo -e "${GREEN}✅ $container_count Container gelöscht${NC}"
    fi
    echo ""
}

# Cleanup 2: cidfiles
cleanup_cidfiles() {
    echo -e "${BLUE}📄 cidfiles Cleanup${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    local cidfiles=$(find /var/folders /tmp -name "mcpproxy-cid-*.txt" 2>/dev/null || true)

    if [ -z "$cidfiles" ]; then
        echo -e "${GREEN}✅ Keine cidfiles gefunden${NC}"
        echo ""
        return 0
    fi

    local cidfile_count=$(echo "$cidfiles" | wc -l | xargs)
    echo "Gefunden: $cidfile_count cidfiles"
    echo ""

    if [ "$DRY_RUN" = true ]; then
        echo "$cidfiles" | while read -r file; do
            echo "  Würde löschen: $file"
        done
    else
        if [ "$FORCE" = false ]; then
            read -p "Alle $cidfile_count cidfiles löschen? [y/N] " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                echo "Abgebrochen"
                echo ""
                return 0
            fi
        fi

        echo "$cidfiles" | while read -r file; do
            log "Lösche: $file"
            rm -f "$file" 2>/dev/null || true
        done

        echo -e "${GREEN}✅ $cidfile_count cidfiles gelöscht${NC}"
    fi
    echo ""
}

# Cleanup 3: Verwaiste mcpproxy Prozesse (nur wenn mehr als 1)
cleanup_orphaned_processes() {
    echo -e "${BLUE}🔄 Prozess Cleanup${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    local process_pids=$(pgrep -f "^\./mcpproxy" || true)

    if [ -z "$process_pids" ]; then
        echo -e "${GREEN}✅ Keine mcpproxy Prozesse laufen${NC}"
        echo ""
        return 0
    fi

    local process_count=$(echo "$process_pids" | wc -l | xargs)

    if [ "$process_count" -le 1 ]; then
        echo -e "${GREEN}✅ Nur 1 mcpproxy Prozess läuft (normal)${NC}"
        echo ""
        return 0
    fi

    echo -e "${YELLOW}⚠️  $process_count mcpproxy Prozesse gefunden${NC}"
    echo ""

    if [ "$DRY_RUN" = true ]; then
        echo "$process_pids" | while read -r pid; do
            local info=$(ps -p "$pid" -o pid,etime,comm 2>/dev/null | tail -1)
            echo "  Würde beenden: PID $info"
        done
    else
        if [ "$FORCE" = false ]; then
            read -p "Alle außer dem neuesten Prozess beenden? [y/N] " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                echo "Abgebrochen"
                echo ""
                return 0
            fi
        fi

        # Behalte nur den neuesten Prozess
        local newest_pid=$(echo "$process_pids" | tail -1)
        local old_pids=$(echo "$process_pids" | head -n -1)

        echo "$old_pids" | while read -r pid; do
            log "Beende alten Prozess: PID $pid"
            kill -9 "$pid" 2>/dev/null || true
        done

        local killed_count=$((process_count - 1))
        echo -e "${GREEN}✅ $killed_count alte Prozesse beendet${NC}"
        echo -e "${GREEN}   Behalten: PID $newest_pid${NC}"
    fi
    echo ""
}

# Cleanup 4: Alte Log-Dateien (>7 Tage)
cleanup_old_logs() {
    echo -e "${BLUE}📝 Log Cleanup${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    local old_logs=$(find /tmp -name "mcpproxy*.log" -mtime +7 2>/dev/null || true)

    if [ -z "$old_logs" ]; then
        echo -e "${GREEN}✅ Keine alten Logs gefunden${NC}"
        echo ""
        return 0
    fi

    local log_count=$(echo "$old_logs" | wc -l | xargs)
    echo "Gefunden: $log_count Logs älter als 7 Tage"
    echo ""

    if [ "$DRY_RUN" = true ]; then
        echo "$old_logs" | while read -r file; do
            local size=$(du -h "$file" 2>/dev/null | awk '{print $1}')
            echo "  Würde löschen: $file ($size)"
        done
    else
        if [ "$FORCE" = false ]; then
            read -p "Alle $log_count alten Logs löschen? [y/N] " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                echo "Abgebrochen"
                echo ""
                return 0
            fi
        fi

        echo "$old_logs" | while read -r file; do
            log "Lösche: $file"
            rm -f "$file" 2>/dev/null || true
        done

        echo -e "${GREEN}✅ $log_count alte Logs gelöscht${NC}"
    fi
    echo ""
}

# Cleanup 5: BBolt Database Compaction (optional)
cleanup_database() {
    echo -e "${BLUE}💾 Database Compaction${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    local db_path="$HOME/.mcpproxy/config.db"

    if [ ! -f "$db_path" ]; then
        echo -e "${YELLOW}⚠️  Database nicht gefunden${NC}"
        echo ""
        return 0
    fi

    local db_size=$(stat -f%z "$db_path" 2>/dev/null || stat -c%s "$db_path" 2>/dev/null)
    local db_size_mb=$((db_size / 1024 / 1024))

    echo "Database-Größe: ${db_size_mb} MB"

    if [ "$db_size" -lt 10485760 ]; then  # < 10 MB
        echo -e "${GREEN}✅ Database klein genug, keine Compaction nötig${NC}"
        echo ""
        return 0
    fi

    echo -e "${YELLOW}⚠️  Database >10MB - Compaction empfohlen${NC}"

    if [ "$DRY_RUN" = true ]; then
        echo "  Würde Compaction durchführen"
    else
        echo ""
        echo "Hinweis: Compaction wird beim nächsten mcpproxy-Start automatisch durchgeführt"
        echo "         Siehe: internal/storage/bbolt.go CompactDB()"
    fi
    echo ""
}

# Main Cleanup Flow
main() {
    cleanup_docker_containers
    cleanup_cidfiles
    cleanup_orphaned_processes
    cleanup_old_logs
    cleanup_database

    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    if [ "$DRY_RUN" = true ]; then
        echo -e "${YELLOW}DRY RUN abgeschlossen - Keine Änderungen vorgenommen${NC}"
    else
        echo -e "${GREEN}✅ Cleanup abgeschlossen${NC}"
    fi

    echo ""
}

# Cleanup ausführen
main
