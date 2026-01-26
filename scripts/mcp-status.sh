#!/bin/bash
# MCP Server Status Übersicht
# Zeigt den aktuellen Status aller MCP Server

API_URL="${MCP_API_URL:-http://localhost:8080}"

curl -s "${API_URL}/api/servers" 2>/dev/null | python3 -c "
import json, sys

try:
    data = json.load(sys.stdin)
except json.JSONDecodeError:
    print('❌ Fehler: Konnte keine Verbindung zum MCP Proxy herstellen')
    print(f'   Stelle sicher, dass der Server unter \$MCP_API_URL läuft')
    sys.exit(1)

servers = data.get('servers', [])
connected = [s for s in servers if s.get('connected')]
disconnected = [s for s in servers if not s.get('connected')]
enabled = [s for s in servers if s.get('enabled')]

# Unterscheide zwischen manuell disabled und auto-disabled
# HINWEIS: auto_disabled kann true sein auch wenn enabled=true ist!
auto_disabled = [s for s in servers if s.get('auto_disabled', False)]
manually_disabled = [s for s in servers if not s.get('enabled') and not s.get('auto_disabled', False)]
disabled = auto_disabled + manually_disabled

print('═' * 60)
print('           MCP SERVER STATUS ÜBERSICHT')
print('═' * 60)
print(f'📊 Gesamt: {len(servers)} Server')
print(f'✅ Connected: {len(connected)}')
print(f'❌ Disconnected: {len(disconnected)}')
print(f'🟢 Enabled: {len(enabled)}')
print(f'🔴 Disabled: {len(disabled)} (manuell: {len(manually_disabled)}, auto: {len(auto_disabled)})')
print()
print('─' * 60)
print('CONNECTED SERVERS:')
print('─' * 60)
for s in sorted(connected, key=lambda x: x['name']):
    tools = s.get('tool_count', 0)
    print(f'  ✅ {s[\"name\"]:25} ({tools:3} tools) - {s.get(\"connection_state\", \"\")}')
print()
# Disconnected = nicht connected, enabled UND nicht auto_disabled
disconnected_enabled = [s for s in servers if not s.get('connected') and s.get('enabled') and not s.get('auto_disabled', False)]
print('─' * 60)
print(f'DISCONNECTED (aber enabled, {len(disconnected_enabled)}):')
print('─' * 60)
for s in sorted(disconnected_enabled, key=lambda x: x['name']):
    failures = s.get('consecutive_failures', 0)
    state = s.get('connection_state', '')
    extra = f' [failures: {failures}]' if failures > 0 else ''
    print(f'  ⚠️  {s[\"name\"]:25} - {state}{extra}')

if auto_disabled:
    print()
    print('─' * 60)
    print(f'🤖 AUTO-DISABLED SERVERS ({len(auto_disabled)}):')
    print('   (Auto-disabled nach 7 fehlgeschlagenen Verbindungsversuchen)')
    print('─' * 60)
    for s in sorted(auto_disabled, key=lambda x: x['name']):
        failures = s.get('consecutive_failures', 0)
        reason = s.get('auto_disable_reason', '')
        last_err = s.get('last_error', '')[:50] if s.get('last_error') else ''
        print(f'  🤖 {s[\"name\"]:25} [failures: {failures}]')
        if reason:
            print(f'      └─ Grund: {reason}')
        if last_err:
            print(f'      └─ Error: {last_err}...')

if manually_disabled:
    print()
    print('─' * 60)
    print(f'🔴 MANUELL DISABLED SERVERS ({len(manually_disabled)}):')
    print('─' * 60)
    for s in sorted(manually_disabled, key=lambda x: x['name']):
        print(f'  🔴 {s[\"name\"]:25}')
"
