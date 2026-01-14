
import requests
import json
import sys

def fetch_tools():
    try:
        # Get list of servers
        print("Fetching server list...", flush=True)
        response = requests.get("http://localhost:8080/api/servers", timeout=2)
        response.raise_for_status()
        data = response.json()
        
        servers = data.get("servers", [])
        connected_servers = [s["name"] for s in servers if s.get("connection_state") == "Ready"]
        
        print(f"Found {len(connected_servers)} connected servers.", flush=True)
        
        all_tools = {}
        
        for i, server in enumerate(connected_servers):
            if i % 5 == 0:
                print(f"Processing server {i+1}/{len(connected_servers)}: {server}...", flush=True)
                
            try:
                tools_response = requests.get(f"http://localhost:8080/api/servers/{server}/tools", timeout=3)
                tools_response.raise_for_status()
                tools_data = tools_response.json()
                tools = tools_data.get("tools", [])
                
                if tools is None:
                    tools = []
                    
                all_tools[server] = tools
            except Exception as e:
                print(f"Error fetching tools for {server}: {e}", flush=True)
                all_tools[server] = []
                
        # Write to file
        print("Saving to all_tools.json...", flush=True)
        with open("all_tools.json", "w") as f:
            json.dump(all_tools, f, indent=2)
            
        print(f"Successfully saved tools for {len(all_tools)} servers to all_tools.json", flush=True)
        
    except Exception as e:
        print(f"Fatal error: {e}", flush=True)
        sys.exit(1)

if __name__ == "__main__":
    fetch_tools()
