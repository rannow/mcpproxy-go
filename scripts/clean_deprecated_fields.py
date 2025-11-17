#!/usr/bin/env python3
"""
Remove deprecated fields from mcpServers in mcp_config.json.

Removes: enabled, auto_disabled, auto_disable_reason, quarantined
Keeps: startup_mode and all other fields
"""

import json
import sys
from pathlib import Path
from datetime import datetime
import shutil

def clean_server(server):
    """Remove deprecated fields from a server configuration."""
    deprecated_fields = [
        "enabled",
        "auto_disabled",
        "auto_disable_reason",
        "quarantined",
        "disabled"
    ]

    removed = []
    for field in deprecated_fields:
        if field in server:
            del server[field]
            removed.append(field)

    return removed

def main():
    config_path = Path.home() / ".mcpproxy" / "mcp_config.json"

    if not config_path.exists():
        print(f"❌ Configuration file not found: {config_path}")
        sys.exit(1)

    # Create backup
    backup_path = config_path.with_suffix(f".json.backup-clean-{datetime.now().strftime('%Y%m%d-%H%M%S')}")
    shutil.copy2(config_path, backup_path)
    print(f"✅ Created backup: {backup_path}")

    # Read config
    with open(config_path, 'r') as f:
        config = json.load(f)

    if "mcpServers" not in config:
        print("❌ No mcpServers found in config")
        sys.exit(1)

    # Clean all servers
    total_removed = {}
    for server in config["mcpServers"]:
        removed = clean_server(server)
        for field in removed:
            total_removed[field] = total_removed.get(field, 0) + 1

    # Write cleaned config
    with open(config_path, 'w') as f:
        json.dump(config, f, indent=2)

    print(f"\n✅ Cleaned {len(config['mcpServers'])} servers")
    print(f"\n📋 Removed fields:")
    for field, count in sorted(total_removed.items()):
        print(f"   - {field}: {count} occurrences")

    print(f"\n💾 Updated config: {config_path}")
    print(f"🔒 Backup: {backup_path}")

if __name__ == "__main__":
    main()
