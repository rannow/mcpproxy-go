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
disabled = [s for s in servers if not s.get('enabled')]

print('═' * 60)
print('           MCP SERVER STATUS ÜBERSICHT')
print('═' * 60)
print(f'📊 Gesamt: {len(servers)} Server')
print(f'✅ Connected: {len(connected)}')
print(f'❌ Disconnected: {len(disconnected)}')
print(f'🟢 Enabled: {len(enabled)}')
print(f'🔴 Disabled: {len(disabled)}')
print()
print('─' * 60)
print('CONNECTED SERVERS:')
print('─' * 60)
for s in connected:
    tools = s.get('tool_count', 0)
    print(f'  ✅ {s[\"name\"]:25} ({tools:3} tools) - {s.get(\"connection_state\", \"\")}')
print()
print('─' * 60)
print(f'DISCONNECTED (enabled={len([s for s in disconnected if s.get(\"enabled\")])}):')
print('─' * 60)
for s in disconnected:
    if s.get('enabled'):
        print(f'  ⚠️  {s[\"name\"]:25} - {s.get(\"connection_state\", \"\")}')

if disabled:
    print()
    print('─' * 60)
    print(f'DISABLED SERVERS ({len(disabled)}):')
    print('─' * 60)
    for s in disabled:
        print(f'  🔴 {s[\"name\"]:25}')
"
