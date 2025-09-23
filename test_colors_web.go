package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/test-colors", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		
		// Read config
		configPath := "/Users/hrannow/.mcpproxy/mcp_config.json"
		data, err := os.ReadFile(configPath)
		if err != nil {
			fmt.Fprintf(w, "<h1>Error reading config: %v</h1>", err)
			return
		}

		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err != nil {
			fmt.Fprintf(w, "<h1>Error parsing config: %v</h1>", err)
			return
		}

		fmt.Fprintf(w, `
		<html>
		<head><title>MCPProxy Color Test</title></head>
		<body>
		<h1>MCPProxy Color Test</h1>
		<h2>Colors from Config File:</h2>
		<table border="1" style="font-size: 20px;">
		<tr><th>Group Name</th><th>Color Code</th><th>Color Emoji</th><th>Expected</th><th>Status</th></tr>
		`)

		expected := map[string]string{
			"To Update":    "🩷",
			"Neu":          "🟡", 
			"To Test":      "🟠",
			"AWS Services": "🟣",
			"OK":           "🟢",
		}

		if groups, ok := config["groups"].([]interface{}); ok {
			for _, groupInterface := range groups {
				if group, ok := groupInterface.(map[string]interface{}); ok {
					name, _ := group["name"].(string)
					color, _ := group["color"].(string)
					colorEmoji, _ := group["color_emoji"].(string)
					
					expectedEmoji := expected[name]
					status := "❌ WRONG"
					if colorEmoji == expectedEmoji {
						status = "✅ CORRECT"
					}
					
					fmt.Fprintf(w, `
					<tr>
						<td>%s</td>
						<td>%s</td>
						<td style="font-size: 30px;">%s</td>
						<td style="font-size: 30px;">%s</td>
						<td>%s</td>
					</tr>
					`, name, color, colorEmoji, expectedEmoji, status)
				}
			}
		}

		fmt.Fprintf(w, `
		</table>
		<br>
		<h2>Test MCPProxy API:</h2>
		<button onclick="testAPI()">Test MCPProxy Groups API</button>
		<div id="api-result"></div>
		
		<script>
		function testAPI() {
			fetch('http://localhost:8080/api/groups')
				.then(response => response.json())
				.then(data => {
					document.getElementById('api-result').innerHTML = 
						'<h3>API Response:</h3><pre>' + JSON.stringify(data, null, 2) + '</pre>';
				})
				.catch(error => {
					document.getElementById('api-result').innerHTML = 
						'<h3>API Error:</h3>' + error;
				});
		}
		</script>
		</body>
		</html>
		`)
	})

	fmt.Println("Color test server running at http://localhost:9999/test-colors")
	http.ListenAndServe(":9999", nil)
}
