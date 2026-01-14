
import json
import os
import subprocess
import sys

CONFIG_PATH = os.path.expanduser("~/.mcpproxy/config.json")

def fix_servers():
    print(f"Reading config via {CONFIG_PATH}...")
    try:
        with open(CONFIG_PATH, "r") as f:
            config = json.load(f)
    except Exception as e:
        print(f"Error reading config: {e}")
        return

    servers = config.get("mcpServers", config.get("servers", []))
    print(f"Found {len(servers)} servers in config.")
    
    modified_count = 0
    match_count = 0
    
    for i, server in enumerate(servers):
        name = server.get("name", "")
        cmd = server.get("command", "")
        
        # Specific debug for target
        if "iam-mcp-server" in name:
            print(f"DEBUG TARGET: name='{name}', command='{cmd}', auto_disabled={server.get('auto_disabled')}")
            
        if cmd == "uvx" and name.startswith("awslabs."):
            match_count += 1
            current_args = server.get("args", [])
            
            if not current_args:
                print(f"Skipping {name}: no args found")
                continue
                
            pkg_arg = current_args[0]
            
            print(f"[{name}] Installing {pkg_arg}...")
            try:
                subprocess.run(["uv", "tool", "install", pkg_arg], check=True)
                print(f"[{name}] Successfully installed.")
            except subprocess.CalledProcessError as e:
                print(f"[{name}] Failed to install: {e}")
            
            server["command"] = "uv"
            server["args"] = ["tool", "run", pkg_arg]
            
            if server.get("auto_disabled"):
                server["auto_disabled"] = False
                server["auto_disable_reason"] = ""
                
            modified_count += 1
            print(f"[{name}] Config updated.")

    print(f"Total matches found: {match_count}")

    if modified_count > 0:
        print(f"Saving config with {modified_count} updates...")
        try:
            with open(CONFIG_PATH, "w") as f:
                json.dump(config, f, indent=2)
            print("Config saved.")
        except Exception as e:
            print(f"Error saving config: {e}")
    else:
        print("No servers matched criteria.")

if __name__ == "__main__":
    fix_servers()
