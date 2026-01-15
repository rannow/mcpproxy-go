
import json
import textwrap

def generate_plan():
    try:
        with open("all_tools.json", "r") as f:
            all_tools = json.load(f)
            
        md_lines = []
        md_lines.append("# MCP Server Test Plan & Analysis")
        md_lines.append("")
        md_lines.append("This document outlines a plan to test the tools provided by connected MCP servers and analyzes their functionality.")
        md_lines.append("")
        
        # Sort servers by name
        sorted_servers = sorted(all_tools.keys())
        
        for server in sorted_servers:
            tools = all_tools[server]
            
            md_lines.append(f"## Server: `{server}`")
            
            if not tools:
                md_lines.append("*Status: No tools available or retrieval failed.*")
                md_lines.append("")
                continue
                
            md_lines.append(f"**Total Tools:** {len(tools)}")
            md_lines.append("")
            
            # Analysis Section (Heuristic)
            descriptions = [t.get("description", "") for t in tools]
            all_text = " ".join(descriptions).lower()
            
            analysis = "Generic utility server."
            if "search" in all_text:
                analysis = "Provides search capabilities."
            elif "database" in all_text or "sql" in all_text or "postgres" in all_text:
                 analysis = "Database interaction tools."
            elif "file" in all_text or "filesystem" in all_text:
                 analysis = "Filesystem interaction tools."
            elif "api" in all_text:
                 analysis = "API integration tools."
            elif "git" in all_text:
                 analysis = "Version control integration."
            
            md_lines.append(f"**Analysis:** {analysis}")
            md_lines.append("")
            
            md_lines.append("### Test Cases")
            
            for tool in tools:
                name = tool.get("name", "Unknown")
                desc = tool.get("description", "No description provided.")
                params = tool.get("inputSchema", {}).get("properties", {})
                required = tool.get("inputSchema", {}).get("required", [])
                
                md_lines.append(f"#### Tool: `{name}`")
                md_lines.append(f"> {desc}")
                md_lines.append("")
                
                if params:
                    md_lines.append("**Parameters:**")
                    for param_name, param_info in params.items():
                        req_mark = "*" if param_name in required else ""
                        p_desc = param_info.get("description", "")
                        p_type = param_info.get("type", "any")
                        md_lines.append(f"- `{param_name}` ({p_type}){req_mark}: {p_desc}")
                else:
                    md_lines.append("**Parameters:** None")
                    
                md_lines.append("")
                md_lines.append(f"- [ ] Verify `{name}` execution")
                md_lines.append("")
                
            md_lines.append("---")
            md_lines.append("")
            
        with open("test_plan.md", "w") as f:
            f.write("\n".join(md_lines))
            
        print("Successfully generated test_plan.md")
        
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    generate_plan()
