#!/usr/bin/env python3
"""
Consolidate all mcp_config*.json files into a single Markdown table.
"""

import json
import os
from pathlib import Path
from collections import defaultdict

def extract_server_info(config_data):
    """Extract server information from config data."""
    servers = {}

    if not isinstance(config_data, dict):
        return servers

    # Handle different config structures
    mcp_servers = config_data.get('mcpServers', config_data.get('upstreamServers', config_data.get('upstream_servers', [])))

    if not isinstance(mcp_servers, list):
        return servers

    for server in mcp_servers:
        if not isinstance(server, dict):
            continue

        name = server.get('name', '')
        if not name:
            continue

        # Extract all relevant fields
        server_info = {
            'name': name,
            'args': ', '.join(server.get('args', [])) if isinstance(server.get('args', []), list) else '',
            'command': server.get('command', ''),
            'description': server.get('description', ''),
            'env': ', '.join([f"{k}={v}" for k, v in server.get('env', {}).items()]) if isinstance(server.get('env', {}), dict) else '',
            'ever_connected': 'Yes' if server.get('ever_connected', False) else 'No',
            'group_id': server.get('group_id', ''),
            'protocol': server.get('protocol', ''),
            'repository_url': server.get('repository_url', ''),
            'tool_count': str(server.get('tool_count', 0)),
            'getestet': '',
            'basic_mcp': ''
        }

        # Use name as key to avoid duplicates (will keep the most recent/complete entry)
        if name not in servers or len(str(server_info.get('description', ''))) > len(str(servers[name].get('description', ''))):
            servers[name] = server_info

    return servers

def main():
    config_dir = Path.home() / '.mcpproxy'
    all_servers = {}

    # Find all mcp_config*.json files
    config_files = sorted(config_dir.glob('mcp_config*.json'))

    print(f"Found {len(config_files)} config files")

    for config_file in config_files:
        try:
            with open(config_file, 'r', encoding='utf-8') as f:
                config_data = json.load(f)
                servers = extract_server_info(config_data)

                # Merge servers (keeping most complete information)
                for name, info in servers.items():
                    if name not in all_servers:
                        all_servers[name] = info
                    else:
                        # Update with non-empty values
                        for key, value in info.items():
                            if value and (not all_servers[name].get(key) or len(str(value)) > len(str(all_servers[name].get(key, '')))):
                                all_servers[name][key] = value

                print(f"Processed {config_file.name}: {len(servers)} servers")
        except Exception as e:
            print(f"Error processing {config_file.name}: {e}")

    # Generate Markdown table
    output_file = Path('/Users/hrannow/Library/CloudStorage/OneDrive-Persönlich/workspace/mcp-server/mcpproxy-go/docs/mcp_servers_consolidated.md')
    output_file.parent.mkdir(parents=True, exist_ok=True)

    with open(output_file, 'w', encoding='utf-8') as f:
        f.write("# MCP Servers Consolidated Configuration\n\n")
        f.write(f"Total unique servers: **{len(all_servers)}**\n\n")

        # Table header
        headers = ['Name', 'Command', 'Args', 'Description', 'Protocol', 'Repository URL', 'Tool Count', 'Ever Connected', 'Group ID', 'Env', 'Getestet', 'Basic MCP']
        f.write('| ' + ' | '.join(headers) + ' |\n')
        f.write('| ' + ' | '.join(['---'] * len(headers)) + ' |\n')

        # Sort servers by name
        for name in sorted(all_servers.keys()):
            server = all_servers[name]
            row = [
                server.get('name', ''),
                server.get('command', ''),
                server.get('args', ''),
                server.get('description', ''),
                server.get('protocol', ''),
                server.get('repository_url', ''),
                server.get('tool_count', '0'),
                server.get('ever_connected', 'No'),
                server.get('group_id', ''),
                server.get('env', ''),
                server.get('getestet', ''),
                server.get('basic_mcp', '')
            ]
            # Escape pipe characters in content
            row = [str(cell).replace('|', '\\|').replace('\n', ' ') for cell in row]
            f.write('| ' + ' | '.join(row) + ' |\n')

    print(f"\n✅ Consolidated table written to: {output_file}")
    print(f"Total unique servers: {len(all_servers)}")

    # List files to be archived
    files_to_archive = [f for f in config_files if f.name != 'mcp_config.json']
    print(f"\n📦 Files to be archived ({len(files_to_archive)}):")
    for f in files_to_archive:
        print(f"  - {f.name}")

if __name__ == '__main__':
    main()
