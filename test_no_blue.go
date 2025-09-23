package main
import "fmt"
func getColorEmojiForHex(hex string) string {
	switch hex {
	case "#ff9900": return "🟠"
	case "#28a745": return "🟢" 
	case "#dc3545": return "🔴"
	case "#6f42c1": return "🟣"
	default: return "⚫"
	}
}
func main() {
	fmt.Println("Unknown color:", getColorEmojiForHex("#unknown"))
	fmt.Println("Blue color:", getColorEmojiForHex("#007bff"))
}
