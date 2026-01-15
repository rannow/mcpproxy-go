# Stop Hook with Sound Notification - Usage Guide

## Overview

The stop hook is triggered when a Claude Code task or session ends. It includes:
- **🔊 Sound notification** to alert task completion
- **📊 Metrics collection** for task completion tracking
- **🔔 Webhook notifications** (optional)
- **💾 Session state export** via Claude Flow

## Features

### 🔊 Sound Notifications
- **System sounds**: Use macOS built-in sounds (Glass, Ping, Pop, etc.)
- **Custom sounds**: Play your own audio files
- **Text-to-Speech**: Spoken messages for task completion
- Configurable sound type and behavior

### 📊 Metrics Collection
- Tracks task completion time
- Records ping results
- Saves to JSON metrics file
- Includes timestamp and host information

### 🔔 Webhook Notifications
- Optional HTTP POST to webhook URL
- JSON payload with task details
- Configurable via environment variable

### 💾 Session State Export
- Integrates with Claude Flow hooks
- Exports metrics and session data
- Preserves context for future sessions

## Configuration

### Environment Variables

```bash
# Sound configuration
export SOUND_ENABLED="true"                           # Enable/disable sound (default: true)
export SOUND_TYPE="system"                            # system, custom, or tts (default: system)
export SYSTEM_SOUND="Glass"                           # macOS system sound name
export CUSTOM_SOUND_PATH="/path/to/your/sound.mp3"   # Path to custom sound file
export TTS_MESSAGE="Task completed"                   # Text-to-speech message

# Notification configuration
export NOTIFICATION_URL="https://your-webhook-url.com/notify"  # Optional webhook

# Task information (automatically set by Claude Code)
export TASK_ID="task-123"
export TASK_DESCRIPTION="Feature implementation"
export TASK_START_TIME="1234567890"
```

### Available System Sounds (macOS)

- `Basso`, `Blow`, `Bottle`, `Frog`, `Funk`, `Glass` (default)
- `Hero`, `Morse`, `Ping`, `Pop`, `Purr`, `Sosumi`, `Submarine`, `Tink`

To see all available sounds:
```bash
ls /System/Library/Sounds/
```

## Usage

### Manual Execution

```bash
# Basic usage (plays default Glass sound)
./scripts/hooks/stop-hook.sh

# With different system sound
SYSTEM_SOUND="Ping" ./scripts/hooks/stop-hook.sh

# With text-to-speech
SOUND_TYPE="tts" TTS_MESSAGE="Build completed successfully" ./scripts/hooks/stop-hook.sh

# With custom sound file
SOUND_TYPE="custom" CUSTOM_SOUND_PATH="/path/to/my-sound.mp3" ./scripts/hooks/stop-hook.sh

# Disable sound
SOUND_ENABLED="false" ./scripts/hooks/stop-hook.sh

# With task information
TASK_ID="feature-auth" TASK_DESCRIPTION="Implement authentication" ./scripts/hooks/stop-hook.sh
```

### Integration with Claude Flow

Add to your Claude Code workflow:

```bash
# After task completion
npx claude-flow@alpha hooks post-task --task-id "task-123"
./scripts/hooks/stop-hook.sh

# On session end
npx claude-flow@alpha hooks session-end --export-metrics true
./scripts/hooks/stop-hook.sh
```

### Automatic Integration

Add to your `.claude-flow/config.json`:

```json
{
  "hooks": {
    "post-task": "./scripts/hooks/stop-hook.sh",
    "session-end": "./scripts/hooks/stop-hook.sh"
  }
}
```

## Output Example

```
2025-11-17T10:30:00.000Z [INFO] ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
2025-11-17T10:30:00.000Z [INFO] 🛑 Stop Hook Triggered
2025-11-17T10:30:00.000Z [INFO] ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
2025-11-17T10:30:00.000Z [INFO] Task ID: feature-auth
2025-11-17T10:30:00.000Z [INFO] Description: Implement authentication
2025-11-17T10:30:00.000Z [INFO] Duration: 120s
2025-11-17T10:30:00.000Z [INFO]
2025-11-17T10:30:00.000Z [INFO] 🔊 Playing sound notification...
2025-11-17T10:30:01.000Z [INFO] ✓ Played system sound: Glass
2025-11-17T10:30:01.000Z [INFO] ✓ Sound notification completed
2025-11-17T10:30:01.000Z [INFO] Metrics saved to ~/.claude-flow/metrics/stop-metrics.json
2025-11-17T10:30:01.000Z [INFO] Exporting session state...
2025-11-17T10:30:02.000Z [INFO]
2025-11-17T10:30:02.000Z [INFO] ✓ Stop hook completed successfully
2025-11-17T10:30:02.000Z [INFO] ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Metrics File Format

Location: `~/.claude-flow/metrics/stop-metrics.json`

```json
{
  "timestamp": "2025-11-17T10:30:00.000Z",
  "task_id": "feature-auth",
  "status": "completed",
  "host": "macbook-pro.local",
  "sound_type": "system",
  "sound_result": "played"
}
```

## Webhook Payload

If `NOTIFICATION_URL` is configured, the following JSON is sent:

```json
{
  "hook": "stop-hook",
  "status": "stopped",
  "task_id": "feature-auth",
  "duration": "120s",
  "timestamp": "2025-11-17T10:30:00.000Z",
  "host": "macbook-pro.local"
}
```

## Logs

Logs are saved to: `~/.claude-flow/logs/stop-hook.log`

## Troubleshooting

### No Sound Playing

If sound doesn't play:
1. **Check macOS audio settings** - ensure volume is not muted
2. **Verify sound file exists** - for custom sounds, check file path
3. **Test afplay manually**: `afplay /System/Library/Sounds/Glass.aiff`
4. **Check logs**: `~/.claude-flow/logs/stop-hook.log`

### Sound Not Found

If system sound isn't found:
```bash
# List all available sounds
ls /System/Library/Sounds/

# Test a sound directly
afplay /System/Library/Sounds/Ping.aiff
```

### Webhook Issues

If notifications aren't working:
1. Verify `NOTIFICATION_URL` is set correctly
2. Check webhook endpoint is accessible
3. Review logs in `~/.claude-flow/logs/stop-hook.log`

### Permission Errors

```bash
# Ensure hook is executable
chmod +x ./scripts/hooks/stop-hook.sh
```

## Advanced Usage

### Different Sounds for Different Tasks

```bash
# Success sound
SYSTEM_SOUND="Glass" ./scripts/hooks/stop-hook.sh

# Error sound
SYSTEM_SOUND="Basso" ./scripts/hooks/stop-hook.sh

# Build complete
SOUND_TYPE="tts" TTS_MESSAGE="Build completed successfully" ./scripts/hooks/stop-hook.sh
```

### Conditional Execution

```bash
# Only run if task succeeded
if [ "$TASK_STATUS" = "success" ]; then
    ./scripts/hooks/stop-hook.sh
fi
```

### Integration with CI/CD

```yaml
# GitHub Actions example
- name: Run stop hook
  run: |
    export TASK_ID="${{ github.run_id }}"
    export TASK_DESCRIPTION="${{ github.event.head_commit.message }}"
    export NOTIFICATION_URL="${{ secrets.WEBHOOK_URL }}"
    ./scripts/hooks/stop-hook.sh
```

## See Also

- [Claude Flow Hooks Documentation](https://github.com/ruvnet/claude-flow)
- [Pre-task Hook](./pre-task-hook.md)
- [Post-task Hook](./post-task-hook.md)
