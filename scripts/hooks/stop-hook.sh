#!/bin/bash
# Stop Hook with Sound Notification for Claude Code
# This hook runs when a task or session stops/ends
# Plays a sound notification to alert completion

set -euo pipefail

# Configuration
HOOK_NAME="stop-hook"
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%S.%3NZ")
LOG_FILE="${HOME}/.claude-flow/logs/${HOOK_NAME}.log"
NOTIFICATION_URL="${NOTIFICATION_URL:-}"  # Optional webhook URL

# Sound configuration
SOUND_ENABLED="${SOUND_ENABLED:-true}"
SOUND_TYPE="${SOUND_TYPE:-system}"  # system, custom, or tts
CUSTOM_SOUND_PATH="${CUSTOM_SOUND_PATH:-}"
TTS_MESSAGE="${TTS_MESSAGE:-Task completed}"
SYSTEM_SOUND="${SYSTEM_SOUND:-Glass}"  # macOS system sound name

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    local level=$1
    shift
    local message="$*"
    echo -e "${TIMESTAMP} [${level}] ${message}" | tee -a "$LOG_FILE"
}

# Play sound notification
play_sound() {
    if [ "$SOUND_ENABLED" != "true" ]; then
        log "INFO" "Sound disabled, skipping notification"
        return 0
    fi

    log "INFO" "${BLUE}🔊 Playing sound notification...${NC}"

    case "$SOUND_TYPE" in
        system)
            # macOS system sound
            if command -v afplay &> /dev/null; then
                local sound_file="/System/Library/Sounds/${SYSTEM_SOUND}.aiff"
                if [ -f "$sound_file" ]; then
                    afplay "$sound_file" &
                    log "INFO" "${GREEN}✓ Played system sound: ${SYSTEM_SOUND}${NC}"
                else
                    # Fallback to default sound
                    afplay /System/Library/Sounds/Glass.aiff &
                    log "INFO" "${GREEN}✓ Played default system sound${NC}"
                fi
            else
                log "WARN" "${YELLOW}⚠ afplay not found (macOS required)${NC}"
                return 1
            fi
            ;;
        custom)
            # Custom sound file
            if [ -n "$CUSTOM_SOUND_PATH" ] && [ -f "$CUSTOM_SOUND_PATH" ]; then
                if command -v afplay &> /dev/null; then
                    afplay "$CUSTOM_SOUND_PATH" &
                    log "INFO" "${GREEN}✓ Played custom sound: ${CUSTOM_SOUND_PATH}${NC}"
                else
                    log "WARN" "${YELLOW}⚠ afplay not found${NC}"
                    return 1
                fi
            else
                log "WARN" "${YELLOW}⚠ Custom sound file not found: ${CUSTOM_SOUND_PATH}${NC}"
                return 1
            fi
            ;;
        tts)
            # Text-to-speech
            if command -v say &> /dev/null; then
                say "$TTS_MESSAGE" &
                log "INFO" "${GREEN}✓ Spoke message: ${TTS_MESSAGE}${NC}"
            else
                log "WARN" "${YELLOW}⚠ say command not found (macOS required)${NC}"
                return 1
            fi
            ;;
        *)
            log "WARN" "${YELLOW}⚠ Unknown sound type: ${SOUND_TYPE}${NC}"
            return 1
            ;;
    esac

    return 0
}

# Send notification if webhook URL is configured
send_notification() {
    local status=$1
    local task_id=${2:-"unknown"}
    local duration=${3:-"unknown"}

    if [ -n "$NOTIFICATION_URL" ]; then
        log "INFO" "Sending notification to webhook..."

        local payload=$(cat <<EOF
{
  "hook": "${HOOK_NAME}",
  "status": "${status}",
  "task_id": "${task_id}",
  "duration": "${duration}",
  "timestamp": "${TIMESTAMP}",
  "host": "$(hostname)"
}
EOF
)

        if curl -s -X POST "$NOTIFICATION_URL" \
            -H "Content-Type: application/json" \
            -d "$payload" > /dev/null 2>&1; then
            log "INFO" "${GREEN}✓ Notification sent${NC}"
        else
            log "WARN" "${YELLOW}⚠ Notification failed${NC}"
        fi
    fi
}

# Save stop metrics
save_metrics() {
    local task_id=${1:-"unknown"}
    local status=${2:-"completed"}
    local metrics_file="${HOME}/.claude-flow/metrics/stop-metrics.json"

    mkdir -p "$(dirname "$metrics_file")"

    # Append to metrics file
    cat >> "$metrics_file" <<EOF
{
  "timestamp": "${TIMESTAMP}",
  "task_id": "${task_id}",
  "status": "${status}",
  "host": "$(hostname)",
  "sound_type": "${SOUND_TYPE}",
  "sound_result": "${SOUND_RESULT:-unknown}"
}
EOF

    log "INFO" "Metrics saved to ${metrics_file}"
}

# Main execution
main() {
    local task_id=${TASK_ID:-"unknown"}
    local task_description=${TASK_DESCRIPTION:-"Task completed"}
    local start_time=${TASK_START_TIME:-$(date +%s)}
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    log "INFO" "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    log "INFO" "${BLUE}🛑 Stop Hook Triggered${NC}"
    log "INFO" "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    log "INFO" "Task ID: ${task_id}"
    log "INFO" "Description: ${task_description}"
    log "INFO" "Duration: ${duration}s"
    log "INFO" ""

    # Play sound notification
    if play_sound; then
        SOUND_RESULT="played"
        log "INFO" "${GREEN}✓ Sound notification completed${NC}"
    else
        SOUND_RESULT="failed"
        log "WARN" "${YELLOW}⚠ Sound notification failed${NC}"
    fi

    # Save metrics
    save_metrics "$task_id" "completed"

    # Send notification if configured
    send_notification "stopped" "$task_id" "${duration}s"

    # Export session state
    log "INFO" "Exporting session state..."
    if command -v npx &> /dev/null; then
        npx claude-flow@alpha hooks session-end --export-metrics true 2>&1 | tee -a "$LOG_FILE" || true
    fi

    log "INFO" ""
    log "INFO" "${GREEN}✓ Stop hook completed successfully${NC}"
    log "INFO" "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Create log directory if it doesn't exist
mkdir -p "$(dirname "$LOG_FILE")"

# Run main function
main "$@"
