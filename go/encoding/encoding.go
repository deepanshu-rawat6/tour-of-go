package encoding

import (
	"fmt"
	"os"
)

// Run executes all encoding examples
func Run() {
	fmt.Println("=== encoding ===")
	fmt.Println()

	jsonBasicExample()
	fmt.Println()

	jsonCustomMarshalingExample()
	fmt.Println()

	textMarshalerExample()
	fmt.Println()

	jsonStreamingExample()
	fmt.Println()
}

// RunExample runs a specific encoding example by name
func RunExample(name string) {
	fmt.Printf("=== encoding: %s ===\n\n", name)

	switch name {
	case "json-basic":
		jsonBasicExample()
	case "custom-marshaling":
		jsonCustomMarshalingExample()
	case "text-marshaler":
		textMarshalerExample()
	case "json-streaming":
		jsonStreamingExample()
	default:
		fmt.Printf("Unknown example: %s\n", name)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  json-basic         - Marshal/Unmarshal, struct tags, omitempty")
		fmt.Println("  custom-marshaling  - MarshalJSON/UnmarshalJSON, time formats, enums")
		fmt.Println("  text-marshaler     - encoding.TextMarshaler/TextUnmarshaler interface")
		fmt.Println("  json-streaming     - json.Encoder/Decoder for large payloads")
		os.Exit(1)
	}
}
