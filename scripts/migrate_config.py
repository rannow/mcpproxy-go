#!/usr/bin/env python3
"""
Migrate mcp_config.json to use startup_mode instead of deprecated boolean fields.

This script:
1. Reads the current configuration
2. Determines the correct startup_mode based on existing fields
3. Removes deprecated fields (enabled, quarantined, auto_disabled, auto_disable_reason, disabled)
4. Sets startup_mode to 'active' for all servers as requested
5. Creates a backup before making changes
"""

import json
import sys
from pathlib import Path
from datetime import datetime
import shutil

def determine_startup_mode(server):
    """
    Determine the correct startup_mode based on existing fields.
    For this migration, we'll set everything to 'active' as requested.
    """
    return "active"

def migrate_server(server):
    """Remove deprecated fields and set startup_mode to active."""
    # Fields to remove
    deprecated_fields = [
        "enabled",
        "quarantined",
        "auto_disabled",
        "auto_disable_reason",
        "disabled"
    ]

    # Remove deprecated fields
    for field in deprecated_fields:
        server.pop(field, None)

    # Set startup_mode to active
    server["startup_mode"] = "active"

    return server

def main():
    config_path = Path.home() / ".mcpproxy" / "mcp_config.json"

    if not config_path.exists():
        print(f"❌ Configuration file not found: {config_path}")
        sys.exit(1)

    # Create backup
    backup_path = config_path.with_suffix(f".json.backup-{datetime.now().strftime('%Y%m%d-%H%M%S')}")
    shutil.copy2(config_path, backup_path)
    print(f"✅ Created backup: {backup_path}")

    # Read current config
    with open(config_path, 'r') as f:
        config = json.load(f)

    # Migrate servers
    if "mcpServers" in config:
        migrated_count = 0
        for server in config["mcpServers"]:
            migrate_server(server)
            migrated_count += 1

        print(f"✅ Migrated {migrated_count} servers to startup_mode='active'")

    # Write updated config
    with open(config_path, 'w') as f:
        json.dump(config, f, indent=2)

    print(f"✅ Configuration updated: {config_path}")
    print(f"\n📋 All servers now have:")
    print(f"   - startup_mode: 'active'")
    print(f"   - Removed: enabled, quarantined, auto_disabled, auto_disable_reason, disabled")
    print(f"\n⚠️  Backup created at: {backup_path}")

if __name__ == "__main__":
    main()
