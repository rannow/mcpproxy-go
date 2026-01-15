
import requests
import json
import time

def update_via_api():
    print("Fetching server list...")
    try:
        resp = requests.get("http://localhost:8080/api/servers")
        resp.raise_for_status()
        data = resp.json()
    except Exception as e:
        print(f"Failed to fetch servers: {e}")
        return

    servers = data.get("servers", [])
    print(f"Found {len(servers)} servers.")

    for server in servers:
        name = server.get("name", "")
        if name.startswith("awslabs.") and server.get("command") == "uvx":
            print(f"Updating {name}...")
            
            # Construct new config
            # We need to preserve current settings but change command/args
            current_args = server.get("args", [])
            if not current_args:
                print(f"Skipping {name}: no args")
                continue
                
            pkg_arg = current_args[0]
            
            # Prepare payload matching struct in server.go
            payload = {
                "name": name,
                "enabled": True, # Re-enable!
                "auto_disabled": False, # Try to force clear
                "protocol": server.get("protocol", "stdio"),
                "command": "uv",
                "args": ["tool", "run", pkg_arg],
                "working_dir": server.get("working_dir", ""),
                "url": server.get("url", ""),
                "env": server.get("env") or {} # Convert None to empty dict if needed
            }
            
            try:
                put_url = f"http://localhost:8080/api/servers/{name}/config"
                print(f"Putting to {put_url} with command 'uv'...")
                r = requests.put(put_url, json=payload)
                r.raise_for_status()
                print(f"[{name}] Successfully updated via API: {r.status_code}")
            except Exception as e:
                print(f"[{name}] Failed to update: {e}")
                if 'r' in locals():
                    print(f"Response: {r.text}")

if __name__ == "__main__":
    update_via_api()
