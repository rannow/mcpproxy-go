
import requests
import json
import time

def enable_all_auto_disabled():
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
    
    count = 0
    for server in servers:
        name = server.get("name", "")
        # Target any server that is auto_disabled, or explicitly awslabs if we want to be safe, 
        # but user said "all auto_disabled server"
        if server.get("auto_disabled") is True:
            print(f"Enabling {name}...")
            
            payload = {
                "name": name,
                "enabled": True,
                "auto_disabled": False,
                "startup_mode": "active", # Explicitly set to active
                "protocol": server.get("protocol", "stdio"),
                "command": server.get("command"),
                "args": server.get("args"),
                "working_dir": server.get("working_dir", ""),
                "url": server.get("url", ""),
                "env": server.get("env") or {}
            }
            
            try:
                put_url = f"http://localhost:8080/api/servers/{name}/config"
                r = requests.put(put_url, json=payload)
                r.raise_for_status()
                print(f"[{name}] Successfully enabled: {r.status_code}")
                count += 1
            except Exception as e:
                print(f"[{name}] Failed to enable: {e}")
                if 'r' in locals():
                    print(f"Response: {r.text}")
                    
    print(f"Process complete. Enabled {count} servers.")

if __name__ == "__main__":
    enable_all_auto_disabled()
